package processing

import (
	"context"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStreamProcessor_ProcessFile(t *testing.T) {
	tests := []struct {
		name          string
		fileSize      int64
		maxFileSize   int64
		memoryLimit   int64
		expectStream  bool
		expectError   bool
		errorContains string
	}{
		{
			name:         "small file loads entirely",
			fileSize:     1024,       // 1KB
			maxFileSize:  10485760,   // 10MB
			memoryLimit:  1073741824, // 1GB
			expectStream: false,
			expectError:  false,
		},
		{
			name:         "large file streams when option enabled",
			fileSize:     104857600,  // 100MB
			maxFileSize:  10485760,   // 10MB
			memoryLimit:  1073741824, // 1GB
			expectStream: true,
			expectError:  false,
		},
		{
			name:         "insufficient memory for entire file",
			fileSize:     104857600, // 100MB
			maxFileSize:  10485760,  // 10MB
			memoryLimit:  1048576,   // 1MB
			expectStream: true,
			expectError:  false, // Should succeed with streaming
		},
		{
			name:          "insufficient memory even for streaming",
			fileSize:      1024,     // 1KB
			maxFileSize:   10485760, // 10MB
			memoryLimit:   100,      // 100 bytes - too small even for 1KB
			expectStream:  false,
			expectError:   true,
			errorContains: "insufficient memory",
		},
		{
			name:         "file at exact threshold",
			fileSize:     10485760,   // 10MB
			maxFileSize:  10485760,   // 10MB
			memoryLimit:  1073741824, // 1GB
			expectStream: false,
			expectError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test file
			tmpFile := createTestFile(t, tt.fileSize)
			defer os.Remove(tmpFile.Name())

			// Setup processor
			memTracker := NewMemoryTracker(tt.memoryLimit)
			processor := NewStreamProcessor(
				WithMaxFileSize(tt.maxFileSize),
				WithMemoryTracker(memTracker),
				WithChunkSize(64), // Small chunk size for testing
			)

			// Process file
			content, err := processor.ProcessFile(
				context.Background(),
				tmpFile.Name(),
				ProcessOptions{
					StreamLargeFiles: true,
					MaxMemoryPerFile: 0, // Don't limit per-file memory in options
				},
			)

			// Verify results
			if tt.expectError {
				assert.Error(t, err)
				if tt.errorContains != "" && err != nil {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
				// Verify all memory was released even on error
				assert.Equal(t, int64(0), memTracker.CurrentUsage())
				return
			}

			require.NoError(t, err)
			require.NotNil(t, content)
			assert.Equal(t, tmpFile.Name(), content.Path)
			assert.Equal(t, tt.expectStream, content.WasStreamed)

			// Verify content integrity
			fullContent := content.GetContent()
			assert.Equal(t, tt.fileSize, int64(len(fullContent)))

			// Verify all memory was released
			assert.Equal(t, int64(0), memTracker.CurrentUsage())
		})
	}
}

func TestStreamProcessor_ContextCancellation(t *testing.T) {
	// Create large file that takes time to process
	tmpFile := createTestFile(t, 100*1024*1024) // 100MB
	defer os.Remove(tmpFile.Name())

	processor := NewStreamProcessor(
		WithMaxFileSize(1024), // Force streaming
		WithChunkSize(1024),   // Small chunks to have more iterations
	)

	// Create cancellable context
	ctx, cancel := context.WithCancel(context.Background())

	// Start processing in goroutine
	errCh := make(chan error, 1)
	go func() {
		_, err := processor.ProcessFile(ctx, tmpFile.Name(), ProcessOptions{
			StreamLargeFiles: true,
		})
		errCh <- err
	}()

	// Cancel after short delay
	time.Sleep(10 * time.Millisecond)
	cancel()

	// Should receive context error
	err := <-errCh
	assert.Error(t, err)
	assert.Equal(t, context.Canceled, err)
}

func TestStreamProcessor_ConcurrentProcessing(t *testing.T) {
	// Create multiple test files
	numFiles := 10
	files := make([]*os.File, numFiles)
	for i := 0; i < numFiles; i++ {
		files[i] = createTestFile(t, 1024*1024) // 1MB each
		defer os.Remove(files[i].Name())
	}

	// Setup processor with limited memory
	memTracker := NewMemoryTracker(5 * 1024 * 1024) // 5MB total
	processor := NewStreamProcessor(
		WithMemoryTracker(memTracker),
	)

	// Process files concurrently
	var wg sync.WaitGroup
	errors := make(chan error, numFiles)

	for _, file := range files {
		wg.Add(1)
		go func(f *os.File) {
			defer wg.Done()

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			_, err := processor.ProcessFile(ctx, f.Name(), ProcessOptions{
				StreamLargeFiles: true,
			})
			if err != nil {
				errors <- err
			}
		}(file)
	}

	wg.Wait()
	close(errors)

	// Check for errors
	var errorCount int
	for err := range errors {
		if err != nil {
			// Memory errors are expected in this test
			if !strings.Contains(err.Error(), "insufficient memory") && err != context.DeadlineExceeded {
				t.Errorf("Unexpected error: %v", err)
			}
			errorCount++
		}
	}

	// Some files might fail due to memory limits, but not all
	assert.LessOrEqual(t, errorCount, numFiles/2)

	// All memory should be released
	assert.Equal(t, int64(0), memTracker.CurrentUsage())
}

func TestStreamProcessor_LineMapping(t *testing.T) {
	// Create file with known content
	content := []string{
		"Line 1",
		"Line 2 is longer",
		"Line 3",
		"", // Empty line
		"Line 5 is the last",
	}

	tmpFile, err := os.CreateTemp("", "test-lines-*.txt")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.WriteString(strings.Join(content, "\n"))
	require.NoError(t, err)
	tmpFile.Close()

	// Process file
	processor := NewStreamProcessor()
	fileContent, err := processor.ProcessFile(
		context.Background(),
		tmpFile.Name(),
		ProcessOptions{},
	)
	require.NoError(t, err)

	// Verify line mapping
	assert.NotNil(t, fileContent.LineMap)

	// The actual number of entries depends on the implementation
	// We have 5 lines but the last line doesn't end with newline
	expectedLines := 5
	assert.Equal(t, expectedLines, len(fileContent.LineMap))

	// Verify specific line offsets
	assert.Equal(t, int64(0), fileContent.LineMap[0])  // Line 1 starts at 0
	assert.Equal(t, int64(7), fileContent.LineMap[1])  // After "Line 1\n"
	assert.Equal(t, int64(24), fileContent.LineMap[2]) // After "Line 2 is longer\n"
	assert.Equal(t, int64(31), fileContent.LineMap[3]) // After "Line 3\n"
	assert.Equal(t, int64(32), fileContent.LineMap[4]) // After empty line
}

func TestStreamProcessor_GetLines(t *testing.T) {
	// Create file with test content
	content := "Line 1\nLine 2\nLine 3"
	tmpFile := createTestFileWithContent(t, content)
	defer os.Remove(tmpFile.Name())

	// Process file
	processor := NewStreamProcessor()
	fileContent, err := processor.ProcessFile(
		context.Background(),
		tmpFile.Name(),
		ProcessOptions{},
	)
	require.NoError(t, err)

	// Get lines
	lines := fileContent.GetLines()
	assert.Len(t, lines, 3)
	assert.Equal(t, "Line 1", string(lines[0]))
	assert.Equal(t, "Line 2", string(lines[1]))
	assert.Equal(t, "Line 3", string(lines[2]))
}

func TestStreamProcessor_NonRegularFile(t *testing.T) {
	// Try to process a directory
	tmpDir := t.TempDir()

	processor := NewStreamProcessor()
	_, err := processor.ProcessFile(
		context.Background(),
		tmpDir,
		ProcessOptions{},
	)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not a regular file")
}

func TestStreamProcessor_MissingFile(t *testing.T) {
	processor := NewStreamProcessor()
	_, err := processor.ProcessFile(
		context.Background(),
		"/non/existent/file.txt",
		ProcessOptions{},
	)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "stat file")
}

func TestMemoryTracker_ConcurrentAllocation(t *testing.T) {
	tracker := NewMemoryTracker(100 * 1024 * 1024) // 100MB limit

	var wg sync.WaitGroup
	successCount := atomic.Int32{}
	allocationSize := int64(2 * 1024 * 1024) // 2MB per allocation

	// Spawn 100 goroutines trying to allocate memory
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			if err := tracker.TryAcquire(allocationSize); err == nil {
				successCount.Add(1)
				// Simulate work
				time.Sleep(10 * time.Millisecond)
				tracker.Release(allocationSize)
			}
		}()
	}

	wg.Wait()

	// Should have succeeded for ~50 allocations (100MB / 2MB)
	successful := successCount.Load()
	assert.Greater(t, successful, int32(40))
	assert.Less(t, successful, int32(60))

	// All memory should be released
	assert.Equal(t, int64(0), tracker.CurrentUsage())
}

func TestMemoryTracker_BlockingAcquire(t *testing.T) {
	tracker := NewMemoryTracker(10 * 1024 * 1024) // 10MB limit

	// Allocate most of the memory
	require.NoError(t, tracker.TryAcquire(8*1024*1024))

	// Try to acquire more memory with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	start := time.Now()
	err := tracker.Acquire(ctx, 5*1024*1024) // 5MB more
	duration := time.Since(start)

	// Should timeout
	assert.Error(t, err)
	assert.Equal(t, context.DeadlineExceeded, err)
	assert.Greater(t, duration, 90*time.Millisecond)

	// Release memory and try again
	tracker.Release(8 * 1024 * 1024)
	err = tracker.Acquire(context.Background(), 5*1024*1024)
	assert.NoError(t, err)
}

func TestFileContent_StreamedVsEntire(t *testing.T) {
	// Test streamed content
	streamedContent := &FileContent{
		Path:        "test.txt",
		WasStreamed: true,
		Chunks: [][]byte{
			[]byte("Chunk 1\n"),
			[]byte("Chunk 2\n"),
			[]byte("Chunk 3"),
		},
	}

	combined := streamedContent.GetContent()
	assert.Equal(t, "Chunk 1\nChunk 2\nChunk 3", string(combined))

	// Test non-streamed content
	entireContent := &FileContent{
		Path:        "test.txt",
		WasStreamed: false,
		Data:        []byte("Entire content"),
	}

	assert.Equal(t, "Entire content", string(entireContent.GetContent()))
}

func BenchmarkStreamProcessor_SmallFile(b *testing.B) {
	// Create 1KB test file
	tmpFile := createTestFile(b, 1024)
	defer os.Remove(tmpFile.Name())

	processor := NewStreamProcessor()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := processor.ProcessFile(
			context.Background(),
			tmpFile.Name(),
			ProcessOptions{},
		)
		if err != nil {
			b.Fatal(err)
		}
	}

	b.ReportMetric(float64(1024*b.N)/float64(b.Elapsed().Seconds()), "bytes/sec")
}

func BenchmarkStreamProcessor_LargeFile(b *testing.B) {
	// Create 10MB test file
	tmpFile := createTestFile(b, 10*1024*1024)
	defer os.Remove(tmpFile.Name())

	processor := NewStreamProcessor()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := processor.ProcessFile(
			context.Background(),
			tmpFile.Name(),
			ProcessOptions{
				StreamLargeFiles: true,
			},
		)
		if err != nil {
			b.Fatal(err)
		}
	}

	b.ReportMetric(float64(10*1024*1024*b.N)/float64(b.Elapsed().Seconds()), "bytes/sec")
}

// Helper functions

func createTestFile(t testing.TB, size int64) *os.File {
	t.Helper()

	tmpFile, err := os.CreateTemp("", "test-stream-*.txt")
	require.NoError(t, err)

	// Write test data with lines
	lineSize := 100 // Characters per line
	line := make([]byte, lineSize)
	for i := range line {
		if i < lineSize-1 {
			line[i] = byte('A' + (i % 26))
		} else {
			line[i] = '\n'
		}
	}

	written := int64(0)
	for written < size {
		toWrite := size - written
		if toWrite > int64(len(line)) {
			toWrite = int64(len(line))
		}
		n, err := tmpFile.Write(line[:toWrite])
		require.NoError(t, err)
		written += int64(n)
	}

	tmpFile.Close()
	return tmpFile
}

func createTestFileWithContent(t testing.TB, content string) *os.File {
	t.Helper()

	tmpFile, err := os.CreateTemp("", "test-content-*.txt")
	require.NoError(t, err)

	_, err = tmpFile.WriteString(content)
	require.NoError(t, err)

	tmpFile.Close()
	return tmpFile
}
