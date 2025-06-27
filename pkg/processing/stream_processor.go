package processing

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"sync"
	"sync/atomic"
	"time"
)

// StreamProcessor handles file processing with memory management and streaming support
type StreamProcessor struct {
	maxFileSize   int64
	chunkSize     int
	bufferPool    *sync.Pool
	memoryTracker *MemoryTracker
}

// ProcessOptions configures how files are processed
type ProcessOptions struct {
	StreamLargeFiles bool
	MaxMemoryPerFile int64
	EnableCaching    bool
}

// FileContent represents processed file content
type FileContent struct {
	Path        string
	Data        []byte
	Chunks      [][]byte
	LineMap     map[int]int64
	WasStreamed bool
}

// StreamProcessorOption configures the StreamProcessor
type StreamProcessorOption func(*StreamProcessor)

// WithMaxFileSize sets the maximum file size before streaming
func WithMaxFileSize(size int64) StreamProcessorOption {
	return func(sp *StreamProcessor) {
		sp.maxFileSize = size
	}
}

// WithChunkSize sets the chunk size for streaming
func WithChunkSize(size int) StreamProcessorOption {
	return func(sp *StreamProcessor) {
		sp.chunkSize = size
	}
}

// WithMemoryTracker sets the memory tracker
func WithMemoryTracker(tracker *MemoryTracker) StreamProcessorOption {
	return func(sp *StreamProcessor) {
		sp.memoryTracker = tracker
	}
}

// NewStreamProcessor creates a new stream processor
func NewStreamProcessor(opts ...StreamProcessorOption) *StreamProcessor {
	sp := &StreamProcessor{
		maxFileSize: 10 * 1024 * 1024, // 10MB default
		chunkSize:   1024 * 1024,      // 1MB chunks
		bufferPool: &sync.Pool{
			New: func() interface{} {
				buf := make([]byte, 1024*1024)
				return &buf
			},
		},
	}

	// Apply options
	for _, opt := range opts {
		opt(sp)
	}

	// Create default memory tracker if not provided
	if sp.memoryTracker == nil {
		sp.memoryTracker = NewMemoryTracker(2 * 1024 * 1024 * 1024) // 2GB default
	}

	return sp
}

// ProcessFile processes a file with memory management
func (sp *StreamProcessor) ProcessFile(ctx context.Context, path string, opts ProcessOptions) (*FileContent, error) {
	// Check if context is already cancelled
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	// Get file info
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("stat file: %w", err)
	}

	// Validate file
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("not a regular file: %s", path)
	}

	// Determine memory requirement
	memoryRequired := info.Size()
	if opts.MaxMemoryPerFile > 0 && memoryRequired > opts.MaxMemoryPerFile {
		memoryRequired = opts.MaxMemoryPerFile
	}

	// Try to acquire memory
	if err := sp.memoryTracker.TryAcquire(memoryRequired); err != nil {
		// If we can't get full memory, try streaming with smaller allocation
		if info.Size() > sp.maxFileSize && opts.StreamLargeFiles {
			// Calculate minimum memory for streaming (at least 2 chunks)
			streamMemory := int64(sp.chunkSize * 2)
			if streamMemory < 256 {
				streamMemory = 256 // Minimum 256 bytes for streaming
			}
			if err := sp.memoryTracker.TryAcquire(streamMemory); err != nil {
				return nil, fmt.Errorf("insufficient memory: %w", err)
			}
			defer sp.memoryTracker.Release(streamMemory)
			return sp.streamProcess(ctx, path)
		}
		return nil, fmt.Errorf("insufficient memory: %w", err)
	}
	defer sp.memoryTracker.Release(memoryRequired)

	// Stream large files
	if info.Size() > sp.maxFileSize && opts.StreamLargeFiles {
		return sp.streamProcess(ctx, path)
	}

	// Read smaller files entirely
	return sp.readEntire(ctx, path)
}

// streamProcess handles streaming processing of large files
func (sp *StreamProcessor) streamProcess(ctx context.Context, path string) (*FileContent, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open file: %w", err)
	}
	defer file.Close()

	// Get buffer from pool
	bufPtr := sp.bufferPool.Get().(*[]byte)
	defer sp.bufferPool.Put(bufPtr)
	buf := *bufPtr

	reader := bufio.NewReaderSize(file, sp.chunkSize)
	content := &FileContent{
		Path:        path,
		Chunks:      make([][]byte, 0),
		LineMap:     make(map[int]int64),
		WasStreamed: true,
	}

	scanner := bufio.NewScanner(reader)
	// Set buffer with max token size to handle long lines
	scanner.Buffer(buf[:cap(buf)], sp.chunkSize*10)

	lineNum := 0
	currentChunk := make([]byte, 0, sp.chunkSize)
	var fileOffset int64

	for scanner.Scan() {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		line := scanner.Bytes()
		content.LineMap[lineNum] = fileOffset

		// Append line to current chunk
		currentChunk = append(currentChunk, line...)
		currentChunk = append(currentChunk, '\n')

		// If chunk is full, save it
		if len(currentChunk) >= sp.chunkSize {
			chunkCopy := make([]byte, len(currentChunk))
			copy(chunkCopy, currentChunk)
			content.Chunks = append(content.Chunks, chunkCopy)
			currentChunk = currentChunk[:0]
		}

		fileOffset += int64(len(line) + 1) // +1 for newline
		lineNum++
	}

	// Save remaining data
	if len(currentChunk) > 0 {
		chunkCopy := make([]byte, len(currentChunk))
		copy(chunkCopy, currentChunk)
		content.Chunks = append(content.Chunks, chunkCopy)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan file: %w", err)
	}

	return content, nil
}

// readEntire reads the entire file into memory
func (sp *StreamProcessor) readEntire(ctx context.Context, path string) (*FileContent, error) {
	// Open file
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open file: %w", err)
	}
	defer file.Close()

	// Read with context support
	data, err := sp.readWithContext(ctx, file)
	if err != nil {
		return nil, err
	}

	content := &FileContent{
		Path:        path,
		Data:        data,
		WasStreamed: false,
		LineMap:     sp.buildLineMap(data),
	}

	return content, nil
}

// readWithContext reads from a reader with context cancellation support
func (sp *StreamProcessor) readWithContext(ctx context.Context, r io.Reader) ([]byte, error) {
	// Create a channel to signal read completion
	type result struct {
		data []byte
		err  error
	}
	resultCh := make(chan result, 1)

	// Start reading in a goroutine
	go func() {
		data, err := io.ReadAll(r)
		resultCh <- result{data: data, err: err}
	}()

	// Wait for either read completion or context cancellation
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case res := <-resultCh:
		if res.err != nil {
			return nil, fmt.Errorf("read file: %w", res.err)
		}
		return res.data, nil
	}
}

// buildLineMap creates a map of line numbers to byte offsets
func (sp *StreamProcessor) buildLineMap(data []byte) map[int]int64 {
	lineMap := make(map[int]int64)
	lineNum := 0
	lineMap[0] = 0 // First line starts at offset 0

	for i, b := range data {
		if b == '\n' && i+1 < len(data) {
			lineNum++
			lineMap[lineNum] = int64(i + 1)
		}
	}

	// If file doesn't end with newline, we may have missed the last line
	if len(data) > 0 && data[len(data)-1] != '\n' {
		lineNum++ //nolint:ineffassign // Intentional: counting lines even if not adding to map
		// Last line map entry is already set by the loop
	}

	return lineMap
}

// AddLine adds a line to the file content
func (fc *FileContent) AddLine(lineNum int, line []byte) {
	if fc.WasStreamed {
		// For streamed content, we build chunks
		if len(fc.Chunks) == 0 {
			fc.Chunks = append(fc.Chunks, make([]byte, 0))
		}
		lastChunk := fc.Chunks[len(fc.Chunks)-1]
		lastChunk = append(lastChunk, line...)
		lastChunk = append(lastChunk, '\n')
		fc.Chunks[len(fc.Chunks)-1] = lastChunk
	} else {
		// For non-streamed content, append to data
		fc.Data = append(fc.Data, line...)
		fc.Data = append(fc.Data, '\n')
	}
}

// GetContent returns the full content, whether streamed or not
func (fc *FileContent) GetContent() []byte {
	if fc.WasStreamed {
		// Combine all chunks
		var totalSize int
		for _, chunk := range fc.Chunks {
			totalSize += len(chunk)
		}

		combined := make([]byte, 0, totalSize)
		for _, chunk := range fc.Chunks {
			combined = append(combined, chunk...)
		}
		return combined
	}
	return fc.Data
}

// GetLines returns an iterator over lines in the content
func (fc *FileContent) GetLines() [][]byte {
	content := fc.GetContent()
	return bytes.Split(content, []byte{'\n'})
}

// MemoryTracker tracks memory usage across the application
type MemoryTracker struct {
	maxMemory    int64
	currentUsage atomic.Int64
	waitQueue    chan struct{}
}

// NewMemoryTracker creates a new memory tracker
func NewMemoryTracker(maxMemory int64) *MemoryTracker {
	return &MemoryTracker{
		maxMemory: maxMemory,
		waitQueue: make(chan struct{}, 100),
	}
}

// TryAcquire attempts to acquire memory without blocking
func (m *MemoryTracker) TryAcquire(size int64) error {
	for {
		current := m.currentUsage.Load()
		if current+size > m.maxMemory {
			return fmt.Errorf("memory limit exceeded: requested %d, available %d", size, m.maxMemory-current)
		}

		if m.currentUsage.CompareAndSwap(current, current+size) {
			return nil
		}
		// Retry if CAS failed
	}
}

// Acquire acquires memory, blocking if necessary
func (m *MemoryTracker) Acquire(ctx context.Context, size int64) error {
	// Fast path - try without blocking
	if err := m.TryAcquire(size); err == nil {
		return nil
	}

	// Slow path - wait for memory
	select {
	case m.waitQueue <- struct{}{}:
		defer func() { <-m.waitQueue }()

		ticker := time.NewTicker(10 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-ticker.C:
				if err := m.TryAcquire(size); err == nil {
					return nil
				}
			}
		}

	case <-ctx.Done():
		return ctx.Err()
	}
}

// Release releases allocated memory
func (m *MemoryTracker) Release(size int64) {
	m.currentUsage.Add(-size)
}

// CurrentUsage returns the current memory usage
func (m *MemoryTracker) CurrentUsage() int64 {
	return m.currentUsage.Load()
}

// AvailableMemory returns the available memory
func (m *MemoryTracker) AvailableMemory() int64 {
	return m.maxMemory - m.currentUsage.Load()
}
