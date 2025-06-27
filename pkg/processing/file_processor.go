package processing

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	contextval "github.com/MacAttak/pi-scanner/pkg/context"
	"github.com/MacAttak/pi-scanner/pkg/detection"
	"github.com/MacAttak/pi-scanner/pkg/discovery"
)

// FileJob represents a file to be processed through the detection pipeline
type FileJob struct {
	FilePath string
	Content  []byte
	FileInfo discovery.FileResult
}

// ProcessingResult represents the result of processing a file
type ProcessingResult struct {
	FilePath string
	Findings []detection.Finding
	Error    error
	Stats    ProcessingStats
}

// ProcessingStats tracks processing statistics
type ProcessingStats struct {
	BytesProcessed int64
	LinesProcessed int
	ProcessingTime int64 // nanoseconds
}

// FileProcessor handles concurrent file processing through the detection pipeline
type FileProcessor struct {
	detectors        []detection.Detector
	contextValidator *contextval.ContextValidator
	streamProcessor  *StreamProcessor
	config           ProcessorConfig
	numWorkers       int
	jobQueue         chan FileJob
	resultQueue      chan ProcessingResult
	workers          []*FileWorker
	wg               sync.WaitGroup
	ctx              context.Context
	cancel           context.CancelFunc
	started          bool
	mu               sync.RWMutex
}

// FileWorker represents a single worker processing files
type FileWorker struct {
	id          int
	processor   *FileProcessor
	jobQueue    <-chan FileJob
	resultQueue chan<- ProcessingResult
	ctx         context.Context
}

// ProcessorConfig configures the file processor
type ProcessorConfig struct {
	NumWorkers       int
	QueueSize        int
	MaxFileSize      int64
	MaxMemory        int64
	StreamLargeFiles bool
	EnablePatterns   bool
	EnableGitleaks   bool
	EnableCaching    bool
}

// DefaultProcessorConfig returns sensible defaults
func DefaultProcessorConfig() ProcessorConfig {
	return ProcessorConfig{
		NumWorkers:       runtime.NumCPU(),
		QueueSize:        10000,                  // Support very large repositories
		MaxFileSize:      10 * 1024 * 1024,       // 10MB
		MaxMemory:        2 * 1024 * 1024 * 1024, // 2GB
		StreamLargeFiles: true,
		EnablePatterns:   true,
		EnableGitleaks:   true,
		EnableCaching:    false,
	}
}

// NewFileProcessor creates a new file processor with detectors
func NewFileProcessor(config ProcessorConfig, detectors []detection.Detector) *FileProcessor {
	if config.NumWorkers <= 0 {
		config.NumWorkers = runtime.NumCPU()
	}
	if config.QueueSize <= 0 {
		config.QueueSize = 100
	}
	if config.MaxMemory <= 0 {
		config.MaxMemory = 2 * 1024 * 1024 * 1024 // 2GB default
	}

	// Create memory tracker
	memTracker := NewMemoryTracker(config.MaxMemory)

	// Create stream processor
	streamProcessor := NewStreamProcessor(
		WithMaxFileSize(config.MaxFileSize),
		WithMemoryTracker(memTracker),
	)

	return &FileProcessor{
		detectors:        detectors,
		contextValidator: contextval.NewContextValidator(),
		streamProcessor:  streamProcessor,
		config:           config,
		numWorkers:       config.NumWorkers,
		jobQueue:         make(chan FileJob, config.QueueSize),
		resultQueue:      make(chan ProcessingResult, config.QueueSize),
		workers:          make([]*FileWorker, config.NumWorkers),
	}
}

// Start initializes and starts all workers
func (fp *FileProcessor) Start(ctx context.Context) error {
	fp.mu.Lock()
	defer fp.mu.Unlock()

	if fp.started {
		return fmt.Errorf("file processor already started")
	}

	fp.ctx, fp.cancel = context.WithCancel(ctx)
	fp.started = true

	// Start workers
	for i := 0; i < fp.numWorkers; i++ {
		worker := &FileWorker{
			id:          i,
			processor:   fp,
			jobQueue:    fp.jobQueue,
			resultQueue: fp.resultQueue,
			ctx:         fp.ctx,
		}
		fp.workers[i] = worker
		fp.wg.Add(1)
		go worker.start()
	}

	return nil
}

// Submit adds a file to the processing queue
func (fp *FileProcessor) Submit(job FileJob) error {
	fp.mu.RLock()
	defer fp.mu.RUnlock()

	if !fp.started {
		return fmt.Errorf("file processor not started")
	}

	select {
	case fp.jobQueue <- job:
		return nil
	case <-fp.ctx.Done():
		return fp.ctx.Err()
	default:
		return fmt.Errorf("job queue is full")
	}
}

// Results returns a channel for receiving processing results
func (fp *FileProcessor) Results() <-chan ProcessingResult {
	return fp.resultQueue
}

// Stop gracefully stops the file processor
func (fp *FileProcessor) Stop() {
	fp.mu.Lock()
	defer fp.mu.Unlock()

	if !fp.started {
		return
	}

	// Close job queue to signal workers to stop
	close(fp.jobQueue)

	// Cancel context
	if fp.cancel != nil {
		fp.cancel()
	}

	// Wait for all workers to finish
	fp.wg.Wait()

	// Close result queue
	close(fp.resultQueue)

	fp.started = false
}

// GetStats returns processor statistics
func (fp *FileProcessor) GetStats() ProcessorStats {
	fp.mu.RLock()
	defer fp.mu.RUnlock()

	return ProcessorStats{
		NumWorkers:     fp.numWorkers,
		QueuedJobs:     len(fp.jobQueue),
		PendingResults: len(fp.resultQueue),
		IsStarted:      fp.started,
	}
}

// ProcessorStats provides processor statistics
type ProcessorStats struct {
	NumWorkers     int
	QueuedJobs     int
	PendingResults int
	IsStarted      bool
}

// start begins the worker's processing loop
func (w *FileWorker) start() {
	defer w.processor.wg.Done()

	for {
		select {
		case job, ok := <-w.jobQueue:
			if !ok {
				// Job queue closed, worker should stop
				return
			}

			// Process the file
			result := w.processFile(job)

			// Always try to send the result, even if context is cancelled
			// This ensures the caller knows what happened to every submitted job
			select {
			case w.resultQueue <- result:
				// Result sent successfully
			case <-time.After(5 * time.Second):
				// Timeout sending result - this prevents deadlock if result channel is full
				// In production, this should be logged
				return
			}

			// Check if we should stop after processing
			select {
			case <-w.ctx.Done():
				return
			default:
				// Continue processing
			}

		case <-w.ctx.Done():
			// Context cancelled, stop worker
			return
		}
	}
}

// processFile runs all detectors on a file and collects findings
func (w *FileWorker) processFile(job FileJob) ProcessingResult {
	startTime := time.Now()

	result := ProcessingResult{
		FilePath: job.FilePath,
		Findings: []detection.Finding{},
		Stats:    ProcessingStats{},
	}

	// Check context cancellation before processing
	select {
	case <-w.ctx.Done():
		result.Error = w.ctx.Err()
		return result
	default:
	}

	// Determine content source
	var content []byte
	if job.Content != nil {
		// Use provided content (even if empty)
		content = job.Content
		result.Stats.BytesProcessed = int64(len(content))

		// Count lines for stats
		lineCount := 1
		if len(content) > 0 {
			for _, b := range content {
				if b == '\n' {
					lineCount++
				}
			}
		}
		result.Stats.LinesProcessed = lineCount
	} else if job.FilePath != "" {
		// Content not provided, read from file
		fileContent, err := w.processor.streamProcessor.ProcessFile(
			w.ctx,
			job.FilePath,
			ProcessOptions{
				StreamLargeFiles: w.processor.config.StreamLargeFiles,
				MaxMemoryPerFile: w.processor.config.MaxFileSize,
				EnableCaching:    w.processor.config.EnableCaching,
			},
		)
		if err != nil {
			result.Error = fmt.Errorf("failed to read file: %w", err)
			return result
		}
		content = fileContent.GetContent()
		result.Stats.BytesProcessed = int64(len(content))

		// Count lines from line map if available
		if fileContent.LineMap != nil {
			result.Stats.LinesProcessed = len(fileContent.LineMap)
		} else {
			result.Stats.LinesProcessed = 1
		}
	} else {
		// No content and no file path
		result.Error = fmt.Errorf("no content or file path provided")
		return result
	}

	// Run all detectors on the file content
	filename := filepath.Base(job.FilePath)
	for _, detector := range w.processor.detectors {
		findings, err := detector.Detect(w.ctx, content, filename)
		if err != nil {
			// Log error but continue with other detectors
			if result.Error == nil {
				result.Error = fmt.Errorf("detector %s failed: %w", detector.Name(), err)
			}
			continue
		}

		// Update file path in findings and apply context validation
		var validFindings []detection.Finding
		for _, finding := range findings {
			// Create a copy of the finding to avoid race conditions
			f := finding
			f.File = job.FilePath

			// Apply context validation to reduce false positives
			validationResult, err := w.processor.contextValidator.Validate(w.ctx, f, string(content))
			if err == nil {
				if !validationResult.IsValid {
					// Skip invalid findings
					continue
				}
				// Update confidence based on context validation
				f.Confidence = float32(validationResult.Confidence)
			}

			validFindings = append(validFindings, f)
		}

		result.Findings = append(result.Findings, validFindings...)
	}

	// Record processing time
	result.Stats.ProcessingTime = time.Since(startTime).Nanoseconds()

	return result
}

// BatchProcessor handles processing multiple files efficiently
type BatchProcessor struct {
	processor *FileProcessor
	batchSize int
}

// NewBatchProcessor creates a batch processor for multiple files
func NewBatchProcessor(processor *FileProcessor, batchSize int) *BatchProcessor {
	if batchSize <= 0 {
		batchSize = 50 // Default batch size
	}

	return &BatchProcessor{
		processor: processor,
		batchSize: batchSize,
	}
}

// ProcessFiles processes a slice of file jobs in batches
func (bp *BatchProcessor) ProcessFiles(ctx context.Context, jobs []FileJob) ([]ProcessingResult, error) {
	if len(jobs) == 0 {
		return []ProcessingResult{}, nil
	}

	// Start the processor
	err := bp.processor.Start(ctx)
	if err != nil {
		return nil, err
	}
	defer bp.processor.Stop()

	// Submit all jobs
	for _, job := range jobs {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			err := bp.processor.Submit(job)
			if err != nil {
				return nil, fmt.Errorf("failed to submit job %s: %w", job.FilePath, err)
			}
		}
	}

	// Collect results
	var results []ProcessingResult
	resultsReceived := 0
	totalJobs := len(jobs)

	for resultsReceived < totalJobs {
		select {
		case result := <-bp.processor.Results():
			results = append(results, result)
			resultsReceived++

		case <-ctx.Done():
			return results, ctx.Err()
		}
	}

	return results, nil
}

// Pipeline represents a configurable processing pipeline
type Pipeline struct {
	fileDiscovery *discovery.FileDiscovery
	fileProcessor *FileProcessor
	config        PipelineConfig
}

// PipelineConfig configures the entire processing pipeline
type PipelineConfig struct {
	Discovery discovery.Config
	Processor ProcessorConfig
	Detectors []detection.Detector
}

// NewPipeline creates a complete processing pipeline
func NewPipeline(config PipelineConfig) *Pipeline {
	return &Pipeline{
		fileDiscovery: discovery.NewFileDiscovery(config.Discovery),
		fileProcessor: NewFileProcessor(config.Processor, config.Detectors),
		config:        config,
	}
}

// ProcessRepository processes an entire repository
func (p *Pipeline) ProcessRepository(ctx context.Context, repoPath string) (*RepositoryResult, error) {
	// Discover files
	files, err := p.fileDiscovery.DiscoverFiles(ctx, repoPath)
	if err != nil {
		return nil, fmt.Errorf("file discovery failed: %w", err)
	}

	if len(files) == 0 {
		return &RepositoryResult{
			RepoPath:     repoPath,
			FilesScanned: 0,
			Results:      []ProcessingResult{},
		}, nil
	}

	// Convert file results to jobs
	jobs := make([]FileJob, 0, len(files))
	for _, file := range files {
		// Skip binary files
		if file.IsBinary {
			continue
		}

		// Read file content here (implementation would read from disk)
		content := []byte{} // TODO: Implement file reading

		jobs = append(jobs, FileJob{
			FilePath: file.Path,
			Content:  content,
			FileInfo: file,
		})
	}

	// Process files using batch processor
	batchProcessor := NewBatchProcessor(p.fileProcessor, 50)
	results, err := batchProcessor.ProcessFiles(ctx, jobs)
	if err != nil {
		return nil, fmt.Errorf("file processing failed: %w", err)
	}

	return &RepositoryResult{
		RepoPath:     repoPath,
		FilesScanned: len(results),
		Results:      results,
		Stats:        p.fileDiscovery.GetStats(files),
	}, nil
}

// RepositoryResult represents the complete results for a repository
type RepositoryResult struct {
	RepoPath     string
	FilesScanned int
	Results      []ProcessingResult
	Stats        discovery.DiscoveryStats
}
