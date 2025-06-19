package inference

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/MacAttak/pi-scanner/pkg/ml/models"
)

// ModelRunnerConfig holds configuration for the model runner
type ModelRunnerConfig struct {
	ModelPath     string        `json:"model_path"`     // Path to ONNX model
	TokenizerPath string        `json:"tokenizer_path"` // Path to tokenizer
	MaxWorkers    int           `json:"max_workers"`    // Maximum concurrent workers
	QueueSize     int           `json:"queue_size"`     // Size of request queue
	BatchSize     int           `json:"batch_size"`     // Batch size for processing
	Timeout       time.Duration `json:"timeout"`        // Request timeout
}

// ValidationRequest represents a request to validate PI
type ValidationRequest struct {
	Candidate models.PICandidate
	Response  chan ValidationResponse
}

// ValidationResponse represents the response from validation
type ValidationResponse struct {
	Result *models.ValidationResult
	Error  error
}

// ModelRunnerStats holds runtime statistics
type ModelRunnerStats struct {
	TotalRequests      int64         `json:"total_requests"`
	SuccessfulRequests int64         `json:"successful_requests"`
	FailedRequests     int64         `json:"failed_requests"`
	AverageLatency     time.Duration `json:"average_latency"`
	QueueDepth         int           `json:"queue_depth"`
	ActiveWorkers      int           `json:"active_workers"`
}

// HealthStatus represents the health status of the runner
type HealthStatus struct {
	Healthy      bool              `json:"healthy"`
	Status       string            `json:"status"`
	Message      string            `json:"message"`
	LastCheck    time.Time         `json:"last_check"`
	ModelLoaded  bool              `json:"model_loaded"`
	WorkerCount  int               `json:"worker_count"`
	QueueDepth   int               `json:"queue_depth"`
	Stats        ModelRunnerStats  `json:"stats"`
}

// ModelInterface defines the interface for ML models
type ModelInterface interface {
	Initialize() error
	Close() error
	ValidatePI(ctx context.Context, candidate models.PICandidate) (*models.ValidationResult, error)
	BatchValidatePI(ctx context.Context, candidates []models.PICandidate) ([]*models.ValidationResult, error)
}

// ModelRunner manages concurrent model inference
type ModelRunner struct {
	config        ModelRunnerConfig
	model         ModelInterface
	requestQueue  chan ValidationRequest
	workerWg      sync.WaitGroup
	running       atomic.Bool
	stats         ModelRunnerStats
	statsLock     sync.RWMutex
	latencySum    int64 // nanoseconds
	latencyCount  int64
	stopChan      chan struct{}
}

// NewModelRunner creates a new model runner instance
func NewModelRunner(config ModelRunnerConfig) (*ModelRunner, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	// Auto-configure workers if not specified
	if config.MaxWorkers <= 0 {
		config.MaxWorkers = runtime.NumCPU()
	}

	// Set defaults
	if config.QueueSize <= 0 {
		config.QueueSize = config.MaxWorkers * 10
	}

	if config.BatchSize <= 0 {
		config.BatchSize = 8
	}

	if config.Timeout <= 0 {
		config.Timeout = 30 * time.Second
	}

	return &ModelRunner{
		config:       config,
		requestQueue: make(chan ValidationRequest, config.QueueSize),
		stopChan:     make(chan struct{}),
	}, nil
}

// Start initializes the model and starts worker goroutines
func (r *ModelRunner) Start() error {
	if r.running.Load() {
		return fmt.Errorf("model runner already running")
	}

	// Create and initialize the model
	modelConfig := models.GetDefaultDeBERTaConfig(r.config.ModelPath, r.config.TokenizerPath)
	modelConfig.BatchSize = r.config.BatchSize

	model, err := models.NewDeBERTaModel(modelConfig)
	if err != nil {
		return fmt.Errorf("failed to create model: %w", err)
	}

	if err := model.Initialize(); err != nil {
		return fmt.Errorf("failed to initialize model: %w", err)
	}

	r.model = model
	r.running.Store(true)

	// Start worker goroutines
	for i := 0; i < r.config.MaxWorkers; i++ {
		r.workerWg.Add(1)
		go r.worker(i)
	}

	return nil
}

// Stop gracefully shuts down the model runner
func (r *ModelRunner) Stop() error {
	if !r.running.Load() {
		return nil
	}

	// Signal stop
	r.running.Store(false)
	close(r.stopChan)

	// Close request queue
	close(r.requestQueue)

	// Wait for workers to finish
	r.workerWg.Wait()

	// Close the model
	if r.model != nil {
		if err := r.model.Close(); err != nil {
			return fmt.Errorf("failed to close model: %w", err)
		}
	}

	return nil
}

// IsRunning returns whether the runner is currently running
func (r *ModelRunner) IsRunning() bool {
	return r.running.Load()
}

// ValidatePI validates a single PI candidate
func (r *ModelRunner) ValidatePI(ctx context.Context, request ValidationRequest) (*models.ValidationResult, error) {
	if !r.running.Load() {
		return nil, fmt.Errorf("model runner not running")
	}

	// Create response channel if not provided
	if request.Response == nil {
		request.Response = make(chan ValidationResponse, 1)
	}

	// Submit request
	select {
	case r.requestQueue <- request:
		// Request queued
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-r.stopChan:
		return nil, fmt.Errorf("model runner stopping")
	}

	// Wait for response
	select {
	case response := <-request.Response:
		if response.Error != nil {
			return nil, response.Error
		}
		return response.Result, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// BatchValidatePI validates multiple PI candidates
func (r *ModelRunner) BatchValidatePI(ctx context.Context, requests []ValidationRequest) ([]*models.ValidationResult, error) {
	if !r.running.Load() {
		return nil, fmt.Errorf("model runner not running")
	}

	results := make([]*models.ValidationResult, len(requests))
	errors := make([]error, len(requests))
	var wg sync.WaitGroup

	for i := range requests {
		if requests[i].Response == nil {
			requests[i].Response = make(chan ValidationResponse, 1)
		}
		wg.Add(1)

		go func(idx int) {
			defer wg.Done()
			result, err := r.ValidatePI(ctx, requests[idx])
			results[idx] = result
			errors[idx] = err
		}(i)
	}

	wg.Wait()

	// Check for errors
	for i, err := range errors {
		if err != nil {
			return nil, fmt.Errorf("failed to validate candidate %d: %w", i, err)
		}
	}

	return results, nil
}

// worker processes validation requests
func (r *ModelRunner) worker(id int) {
	defer r.workerWg.Done()

	for request := range r.requestQueue {
		startTime := time.Now()
		atomic.AddInt64(&r.stats.TotalRequests, 1)

		// Process the request
		result, err := r.model.ValidatePI(context.Background(), request.Candidate)

		// Update stats
		latency := time.Since(startTime)
		atomic.AddInt64(&r.latencySum, int64(latency))
		atomic.AddInt64(&r.latencyCount, 1)

		if err != nil {
			atomic.AddInt64(&r.stats.FailedRequests, 1)
		} else {
			atomic.AddInt64(&r.stats.SuccessfulRequests, 1)
		}

		// Send response
		select {
		case request.Response <- ValidationResponse{
			Result: result,
			Error:  err,
		}:
		default:
			// Response channel might be closed if request timed out
		}
	}
}

// GetStats returns current statistics
func (r *ModelRunner) GetStats() ModelRunnerStats {
	r.statsLock.RLock()
	defer r.statsLock.RUnlock()

	stats := r.stats
	stats.QueueDepth = len(r.requestQueue)
	stats.ActiveWorkers = r.config.MaxWorkers

	// Calculate average latency
	count := atomic.LoadInt64(&r.latencyCount)
	if count > 0 {
		sum := atomic.LoadInt64(&r.latencySum)
		stats.AverageLatency = time.Duration(sum / count)
	}

	return stats
}

// HealthCheck performs a health check
func (r *ModelRunner) HealthCheck() HealthStatus {
	status := HealthStatus{
		LastCheck:   time.Now(),
		ModelLoaded: r.model != nil,
		WorkerCount: r.config.MaxWorkers,
		QueueDepth:  len(r.requestQueue),
		Stats:       r.GetStats(),
	}

	if !r.running.Load() {
		status.Healthy = false
		status.Status = "stopped"
		status.Message = "Model runner not running"
		return status
	}

	status.Healthy = true
	status.Status = "running"
	status.Message = "Model runner operational"

	// Check queue depth
	queueUsage := float64(status.QueueDepth) / float64(r.config.QueueSize)
	if queueUsage > 0.8 {
		status.Message = fmt.Sprintf("High queue usage: %.1f%%", queueUsage*100)
	}

	return status
}

// Validate validates the configuration
func (c *ModelRunnerConfig) Validate() error {
	if c.ModelPath == "" {
		return fmt.Errorf("model path cannot be empty")
	}

	if c.TokenizerPath == "" {
		return fmt.Errorf("tokenizer path cannot be empty")
	}

	if c.MaxWorkers == 0 || c.MaxWorkers < -1 {
		return fmt.Errorf("max workers must be positive or -1 for auto")
	}

	if c.QueueSize < 0 {
		return fmt.Errorf("queue size must be non-negative")
	}

	return nil
}

// CreateDefaultRunner creates a model runner with default configuration
func CreateDefaultRunner(modelPath, tokenizerPath string) (*ModelRunner, error) {
	config := ModelRunnerConfig{
		ModelPath:     modelPath,
		TokenizerPath: tokenizerPath,
		MaxWorkers:    runtime.NumCPU(),
		QueueSize:     100,
		BatchSize:     8,
		Timeout:       30 * time.Second,
	}

	return NewModelRunner(config)
}