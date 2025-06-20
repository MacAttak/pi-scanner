package processing

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/MacAttak/pi-scanner/pkg/detection"
	"github.com/MacAttak/pi-scanner/pkg/discovery"
)

// MockDetector for testing
type MockDetector struct {
	name     string
	findings []detection.Finding
	delay    time.Duration
	failures int32 // Counter for simulating failures
}

func NewMockDetector(name string, findings []detection.Finding) *MockDetector {
	return &MockDetector{
		name:     name,
		findings: findings,
	}
}

func (m *MockDetector) Name() string {
	return m.name
}

func (m *MockDetector) Detect(ctx context.Context, content []byte, filename string) ([]detection.Finding, error) {
	// Simulate processing delay
	if m.delay > 0 {
		select {
		case <-time.After(m.delay):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	// Simulate occasional failures
	if atomic.LoadInt32(&m.failures) > 0 {
		atomic.AddInt32(&m.failures, -1)
		return nil, errors.New("simulated detector failure")
	}

	// Return mock findings
	return m.findings, nil
}

func (m *MockDetector) SetDelay(delay time.Duration) {
	m.delay = delay
}

func (m *MockDetector) SetFailures(count int32) {
	atomic.StoreInt32(&m.failures, count)
}

func TestFileProcessor_BasicFunctionality(t *testing.T) {
	// Create mock detector
	mockFindings := []detection.Finding{
		{
			Type:         detection.PITypeEmail,
			Match:        "test@example.com",
			File:         "test.txt",
			Line:         1,
			DetectorName: "mock-detector",
		},
	}
	detector := NewMockDetector("mock-detector", mockFindings)

	// Create processor
	config := DefaultProcessorConfig()
	config.NumWorkers = 2
	processor := NewFileProcessor(config, []detection.Detector{detector})

	ctx := context.Background()
	err := processor.Start(ctx)
	require.NoError(t, err)
	defer processor.Stop()

	// Submit test job
	job := FileJob{
		FilePath: "/test/file.txt",
		Content:  []byte("Email: test@example.com"),
		FileInfo: discovery.FileResult{
			Path:     "/test/file.txt",
			Size:     23,
			IsBinary: false,
		},
	}

	err = processor.Submit(job)
	require.NoError(t, err)

	// Get result
	select {
	case result := <-processor.Results():
		assert.NoError(t, result.Error)
		assert.Equal(t, "/test/file.txt", result.FilePath)
		assert.Len(t, result.Findings, 1)
		assert.Equal(t, "/test/file.txt", result.Findings[0].File)
		assert.Equal(t, detection.PITypeEmail, result.Findings[0].Type)
		assert.Greater(t, result.Stats.BytesProcessed, int64(0))
		assert.Greater(t, result.Stats.LinesProcessed, 0)

	case <-time.After(5 * time.Second):
		t.Fatal("Test timed out waiting for result")
	}
}

func TestFileProcessor_MultipleDetectors(t *testing.T) {
	// Create multiple mock detectors
	emailFindings := []detection.Finding{
		{Type: detection.PITypeEmail, Match: "test@example.com", DetectorName: "email-detector"},
	}
	phoneFindings := []detection.Finding{
		{Type: detection.PITypePhone, Match: "0412345678", DetectorName: "phone-detector"},
	}

	detectors := []detection.Detector{
		NewMockDetector("email-detector", emailFindings),
		NewMockDetector("phone-detector", phoneFindings),
	}

	// Create processor
	config := DefaultProcessorConfig()
	config.NumWorkers = 1
	processor := NewFileProcessor(config, detectors)

	ctx := context.Background()
	err := processor.Start(ctx)
	require.NoError(t, err)
	defer processor.Stop()

	// Submit test job
	job := FileJob{
		FilePath: "/test/contacts.txt",
		Content:  []byte("Email: test@example.com\nPhone: 0412345678"),
		FileInfo: discovery.FileResult{Path: "/test/contacts.txt"},
	}

	err = processor.Submit(job)
	require.NoError(t, err)

	// Get result
	select {
	case result := <-processor.Results():
		assert.NoError(t, result.Error)
		assert.Len(t, result.Findings, 2)

		// Check both findings are present
		findingTypes := make(map[detection.PIType]bool)
		for _, finding := range result.Findings {
			findingTypes[finding.Type] = true
			assert.Equal(t, "/test/contacts.txt", finding.File)
		}
		assert.True(t, findingTypes[detection.PITypeEmail])
		assert.True(t, findingTypes[detection.PITypePhone])

	case <-time.After(5 * time.Second):
		t.Fatal("Test timed out waiting for result")
	}
}

func TestFileProcessor_ErrorHandling(t *testing.T) {
	// Create detector that will fail
	detector := NewMockDetector("failing-detector", nil)
	detector.SetFailures(1)

	config := DefaultProcessorConfig()
	config.NumWorkers = 1
	processor := NewFileProcessor(config, []detection.Detector{detector})

	ctx := context.Background()
	err := processor.Start(ctx)
	require.NoError(t, err)
	defer processor.Stop()

	// Submit test job
	job := FileJob{
		FilePath: "/test/file.txt",
		Content:  []byte("test content"),
		FileInfo: discovery.FileResult{Path: "/test/file.txt"},
	}

	err = processor.Submit(job)
	require.NoError(t, err)

	// Get result
	select {
	case result := <-processor.Results():
		assert.Error(t, result.Error)
		assert.Contains(t, result.Error.Error(), "simulated detector failure")
		assert.Equal(t, "/test/file.txt", result.FilePath)

	case <-time.After(5 * time.Second):
		t.Fatal("Test timed out waiting for result")
	}
}

func TestFileProcessor_Concurrency(t *testing.T) {
	numWorkers := 3
	numJobs := 10

	// Create detector with slight delay
	detector := NewMockDetector("concurrent-detector", []detection.Finding{
		{Type: detection.PITypeEmail, Match: "test@example.com"},
	})
	detector.SetDelay(50 * time.Millisecond)

	config := DefaultProcessorConfig()
	config.NumWorkers = numWorkers
	config.QueueSize = numJobs
	processor := NewFileProcessor(config, []detection.Detector{detector})

	ctx := context.Background()
	err := processor.Start(ctx)
	require.NoError(t, err)
	defer processor.Stop()

	// Submit multiple jobs
	start := time.Now()
	for i := 0; i < numJobs; i++ {
		job := FileJob{
			FilePath: fmt.Sprintf("/test/file%d.txt", i),
			Content:  []byte(fmt.Sprintf("Email: test%d@example.com", i)),
			FileInfo: discovery.FileResult{Path: fmt.Sprintf("/test/file%d.txt", i)},
		}

		err = processor.Submit(job)
		require.NoError(t, err)
	}

	// Collect results
	results := make([]ProcessingResult, 0, numJobs)
	for i := 0; i < numJobs; i++ {
		select {
		case result := <-processor.Results():
			assert.NoError(t, result.Error)
			results = append(results, result)

		case <-time.After(10 * time.Second):
			t.Fatal("Test timed out waiting for results")
		}
	}

	duration := time.Since(start)

	// Verify all results received
	assert.Len(t, results, numJobs)

	// With proper concurrency, should complete faster than sequential execution
	maxSequentialTime := time.Duration(numJobs) * 50 * time.Millisecond
	assert.Less(t, duration, maxSequentialTime,
		"Concurrent execution should be faster than sequential")

	// Verify each result has correct file path
	filePaths := make(map[string]bool)
	for _, result := range results {
		filePaths[result.FilePath] = true
		assert.Len(t, result.Findings, 1)
	}
	assert.Len(t, filePaths, numJobs, "All unique file paths should be processed")
}

func TestFileProcessor_Cancellation(t *testing.T) {
	// Create detector with long delay
	detector := NewMockDetector("slow-detector", []detection.Finding{
		{Type: detection.PITypeEmail, Match: "test@example.com"},
	})
	detector.SetDelay(200 * time.Millisecond)

	config := DefaultProcessorConfig()
	config.NumWorkers = 2
	processor := NewFileProcessor(config, []detection.Detector{detector})

	ctx, cancel := context.WithCancel(context.Background())
	err := processor.Start(ctx)
	require.NoError(t, err)
	defer processor.Stop()

	// Submit jobs
	for i := 0; i < 5; i++ {
		job := FileJob{
			FilePath: fmt.Sprintf("/test/file%d.txt", i),
			Content:  []byte("test content"),
			FileInfo: discovery.FileResult{Path: fmt.Sprintf("/test/file%d.txt", i)},
		}

		err = processor.Submit(job)
		require.NoError(t, err)
	}

	// Cancel after short delay
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	// Collect results (some may be cancelled)
	var results []ProcessingResult
	timeout := time.After(2 * time.Second)

	for {
		select {
		case result := <-processor.Results():
			results = append(results, result)
			if result.Error != nil {
				assert.ErrorIs(t, result.Error, context.Canceled)
			}

		case <-timeout:
			// Test completed
			goto done
		}
	}

done:
	// Some results should indicate cancellation
	cancelledResults := 0
	for _, result := range results {
		if result.Error != nil && errors.Is(result.Error, context.Canceled) {
			cancelledResults++
		}
	}

	assert.Greater(t, cancelledResults, 0, "Some results should be cancelled")
}

func TestFileProcessor_QueueCapacity(t *testing.T) {
	queueSize := 2
	detector := NewMockDetector("queue-test-detector", []detection.Finding{})
	detector.SetDelay(100 * time.Millisecond) // Slow processing

	config := DefaultProcessorConfig()
	config.NumWorkers = 1
	config.QueueSize = queueSize
	processor := NewFileProcessor(config, []detection.Detector{detector})

	ctx := context.Background()
	err := processor.Start(ctx)
	require.NoError(t, err)
	defer processor.Stop()

	// Fill queue capacity
	for i := 0; i < queueSize; i++ {
		job := FileJob{
			FilePath: fmt.Sprintf("/test/file%d.txt", i),
			Content:  []byte("test"),
			FileInfo: discovery.FileResult{Path: fmt.Sprintf("/test/file%d.txt", i)},
		}

		err = processor.Submit(job)
		require.NoError(t, err)
	}

	// This should fail as queue is full
	overflowJob := FileJob{
		FilePath: "/test/overflow.txt",
		Content:  []byte("overflow"),
		FileInfo: discovery.FileResult{Path: "/test/overflow.txt"},
	}

	err = processor.Submit(overflowJob)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "job queue is full")
}

func TestBatchProcessor_ProcessFiles(t *testing.T) {
	// Create detector
	detector := NewMockDetector("batch-detector", []detection.Finding{
		{Type: detection.PITypeEmail, Match: "test@example.com"},
	})

	config := DefaultProcessorConfig()
	config.NumWorkers = 2
	fileProcessor := NewFileProcessor(config, []detection.Detector{detector})

	batchProcessor := NewBatchProcessor(fileProcessor, 3)

	// Create test jobs
	jobs := []FileJob{
		{FilePath: "/test/file1.txt", Content: []byte("content1")},
		{FilePath: "/test/file2.txt", Content: []byte("content2")},
		{FilePath: "/test/file3.txt", Content: []byte("content3")},
		{FilePath: "/test/file4.txt", Content: []byte("content4")},
	}

	ctx := context.Background()
	results, err := batchProcessor.ProcessFiles(ctx, jobs)

	require.NoError(t, err)
	assert.Len(t, results, len(jobs))

	// Verify all files were processed
	processedFiles := make(map[string]bool)
	for _, result := range results {
		processedFiles[result.FilePath] = true
		assert.NoError(t, result.Error)
		assert.Len(t, result.Findings, 1)
	}

	for _, job := range jobs {
		assert.True(t, processedFiles[job.FilePath],
			"File %s should be processed", job.FilePath)
	}
}

func TestFileProcessor_Stats(t *testing.T) {
	detector := NewMockDetector("stats-detector", []detection.Finding{})

	config := DefaultProcessorConfig()
	config.NumWorkers = 3
	config.QueueSize = 10
	processor := NewFileProcessor(config, []detection.Detector{detector})

	// Check initial stats
	stats := processor.GetStats()
	assert.Equal(t, 3, stats.NumWorkers)
	assert.Equal(t, 0, stats.QueuedJobs)
	assert.Equal(t, 0, stats.PendingResults)
	assert.False(t, stats.IsStarted)

	// Start processor
	ctx := context.Background()
	err := processor.Start(ctx)
	require.NoError(t, err)
	defer processor.Stop()

	// Check stats after start
	stats = processor.GetStats()
	assert.True(t, stats.IsStarted)
}

func TestFileProcessor_LinesCounting(t *testing.T) {
	detector := NewMockDetector("lines-detector", []detection.Finding{})

	config := DefaultProcessorConfig()
	config.NumWorkers = 1
	processor := NewFileProcessor(config, []detection.Detector{detector})

	ctx := context.Background()
	err := processor.Start(ctx)
	require.NoError(t, err)
	defer processor.Stop()

	// Test content with multiple lines
	content := "Line 1\nLine 2\nLine 3\nLine 4"
	job := FileJob{
		FilePath: "/test/multiline.txt",
		Content:  []byte(content),
		FileInfo: discovery.FileResult{Path: "/test/multiline.txt"},
	}

	err = processor.Submit(job)
	require.NoError(t, err)

	// Get result
	select {
	case result := <-processor.Results():
		assert.NoError(t, result.Error)
		assert.Equal(t, 4, result.Stats.LinesProcessed)
		assert.Equal(t, int64(len(content)), result.Stats.BytesProcessed)

	case <-time.After(5 * time.Second):
		t.Fatal("Test timed out waiting for result")
	}
}

func TestFileProcessor_EmptyContent(t *testing.T) {
	detector := NewMockDetector("empty-detector", []detection.Finding{})

	config := DefaultProcessorConfig()
	config.NumWorkers = 1
	processor := NewFileProcessor(config, []detection.Detector{detector})

	ctx := context.Background()
	err := processor.Start(ctx)
	require.NoError(t, err)
	defer processor.Stop()

	// Test empty content
	job := FileJob{
		FilePath: "/test/empty.txt",
		Content:  []byte(""),
		FileInfo: discovery.FileResult{Path: "/test/empty.txt"},
	}

	err = processor.Submit(job)
	require.NoError(t, err)

	// Get result
	select {
	case result := <-processor.Results():
		assert.NoError(t, result.Error)
		assert.Equal(t, 1, result.Stats.LinesProcessed) // Even empty files have 1 line
		assert.Equal(t, int64(0), result.Stats.BytesProcessed)
		assert.Len(t, result.Findings, 0)

	case <-time.After(5 * time.Second):
		t.Fatal("Test timed out waiting for result")
	}
}

// Benchmark tests
func BenchmarkFileProcessor_SingleFile(b *testing.B) {
	detector := NewMockDetector("bench-detector", []detection.Finding{
		{Type: detection.PITypeEmail, Match: "test@example.com"},
	})

	config := DefaultProcessorConfig()
	config.NumWorkers = 1
	processor := NewFileProcessor(config, []detection.Detector{detector})

	ctx := context.Background()
	err := processor.Start(ctx)
	require.NoError(b, err)
	defer processor.Stop()

	content := []byte(strings.Repeat("This is test content with test@example.com email.\n", 100))

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		job := FileJob{
			FilePath: fmt.Sprintf("/test/file%d.txt", i),
			Content:  content,
			FileInfo: discovery.FileResult{Path: fmt.Sprintf("/test/file%d.txt", i)},
		}

		err = processor.Submit(job)
		require.NoError(b, err)

		// Wait for result
		<-processor.Results()
	}
}

func BenchmarkFileProcessor_Concurrent(b *testing.B) {
	detector := NewMockDetector("bench-concurrent-detector", []detection.Finding{
		{Type: detection.PITypeEmail, Match: "test@example.com"},
	})

	config := DefaultProcessorConfig()
	config.NumWorkers = 4
	processor := NewFileProcessor(config, []detection.Detector{detector})

	ctx := context.Background()
	err := processor.Start(ctx)
	require.NoError(b, err)
	defer processor.Stop()

	content := []byte(strings.Repeat("Test content with test@example.com.\n", 50))

	b.ResetTimer()

	// Submit all jobs first
	for i := 0; i < b.N; i++ {
		job := FileJob{
			FilePath: fmt.Sprintf("/test/file%d.txt", i),
			Content:  content,
			FileInfo: discovery.FileResult{Path: fmt.Sprintf("/test/file%d.txt", i)},
		}

		err = processor.Submit(job)
		require.NoError(b, err)
	}

	// Collect all results
	for i := 0; i < b.N; i++ {
		<-processor.Results()
	}
}
