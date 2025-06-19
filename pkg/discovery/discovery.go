package discovery

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/bmatcuk/doublestar/v4"
)

// FileResult represents a discovered file with metadata
type FileResult struct {
	Path     string
	Size     int64
	IsBinary bool
	IsHidden bool
}

// Config holds file discovery configuration
type Config struct {
	// Include patterns (glob-style)
	IncludePatterns []string
	
	// Exclude patterns (glob-style)
	ExcludePatterns []string
	
	// Exclude binary files
	ExcludeBinary bool
	
	// Maximum file size to include (bytes)
	MaxFileSize int64
	
	// Include hidden files (starting with .)
	IncludeHidden bool
	
	// Follow symbolic links
	FollowSymlinks bool
}

// FileDiscovery handles file discovery with filtering
type FileDiscovery struct {
	config Config
}

// NewFileDiscovery creates a new file discovery instance
func NewFileDiscovery(config Config) *FileDiscovery {
	return &FileDiscovery{
		config: config,
	}
}

// DefaultConfig returns the default file discovery configuration
func DefaultConfig() Config {
	return Config{
		IncludePatterns: []string{
			"**/*.go", "**/*.py", "**/*.js", "**/*.ts", "**/*.java",
			"**/*.c", "**/*.cpp", "**/*.h", "**/*.hpp", "**/*.cs",
			"**/*.php", "**/*.rb", "**/*.rs", "**/*.swift", "**/*.kt",
			"**/*.scala", "**/*.clj", "**/*.sh", "**/*.bash", "**/*.zsh",
			"**/*.ps1", "**/*.bat", "**/*.cmd", "**/*.sql", "**/*.yaml",
			"**/*.yml", "**/*.json", "**/*.xml", "**/*.toml", "**/*.ini",
			"**/*.cfg", "**/*.conf", "**/*.config", "**/*.env", "**/*.properties",
			"**/*.md", "**/*.txt", "**/*.log", "**/*.dockerfile", "**/Dockerfile",
			"**/Makefile", "**/*.mk", "**/*.gradle", "**/*.maven", "**/*.pom",
		},
		ExcludePatterns: []string{
			"**/test/**", "**/*_test.*", "**/*.test.*", "**/tests/**",
			"**/testdata/**", "**/fixtures/**", "**/mocks/**", "**/mock/**",
			"**/vendor/**", "**/node_modules/**", "**/bower_components/**",
			"**/.git/**", "**/.svn/**", "**/.hg/**", "**/.bzr/**",
			"**/build/**", "**/dist/**", "**/out/**", "**/target/**",
			"**/.idea/**", "**/.vscode/**", "**/coverage/**",
			"**/*.min.js", "**/*.min.css", "**/*.bundle.*",
			"**/.DS_Store", "**/Thumbs.db", "**/*.tmp", "**/*.temp",
		},
		ExcludeBinary:   true,
		MaxFileSize:     10 * 1024 * 1024, // 10MB
		IncludeHidden:   true, // Include hidden files like .env
		FollowSymlinks:  false,
	}
}

// DiscoverFiles discovers all files in the given directory matching the configuration
func (fd *FileDiscovery) DiscoverFiles(ctx context.Context, rootPath string) ([]FileResult, error) {
	var results []FileResult
	
	// Check if root path exists
	if _, err := os.Stat(rootPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("directory does not exist: %s", rootPath)
	}
	
	err := filepath.WalkDir(rootPath, func(path string, entry fs.DirEntry, err error) error {
		// Check for context cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		
		// Handle permission errors gracefully
		if err != nil {
			// Skip permission denied errors
			if os.IsPermission(err) {
				return fs.SkipDir
			}
			return err
		}
		
		// Skip directories
		if entry.IsDir() {
			return nil
		}
		
		// Get file info
		info, err := entry.Info()
		if err != nil {
			// Skip files we can't stat
			return nil
		}
		
		// Check if file should be included
		if fd.shouldIncludeFile(path, info, rootPath) {
			// Detect if file is binary
			isBinary, err := fd.isBinaryFile(path)
			if err != nil {
				// If we can't determine, assume text
				isBinary = false
			}
			
			// Skip binary files if configured
			if fd.config.ExcludeBinary && isBinary {
				return nil
			}
			
			result := FileResult{
				Path:     path,
				Size:     info.Size(),
				IsBinary: isBinary,
				IsHidden: fd.isHiddenFile(path),
			}
			
			results = append(results, result)
		}
		
		return nil
	})
	
	if err != nil {
		return nil, fmt.Errorf("error walking directory: %w", err)
	}
	
	// Ensure we always return a non-nil slice
	if results == nil {
		results = []FileResult{}
	}
	
	return results, nil
}

// shouldIncludeFile determines if a file should be included based on patterns and size
func (fd *FileDiscovery) shouldIncludeFile(path string, info fs.FileInfo, rootPath string) bool {
	// Check file size
	if fd.config.MaxFileSize > 0 && info.Size() > fd.config.MaxFileSize {
		return false
	}
	
	// Check hidden files
	if !fd.config.IncludeHidden && fd.isHiddenFile(path) {
		return false
	}
	
	// Get relative path for pattern matching
	relPath, err := filepath.Rel(rootPath, path)
	if err != nil {
		relPath = path
	}
	
	// Check exclude patterns first
	for _, pattern := range fd.config.ExcludePatterns {
		if fd.matchesPattern(relPath, pattern) {
			return false
		}
	}
	
	// Check include patterns
	if len(fd.config.IncludePatterns) == 0 {
		return true // Include all if no patterns specified
	}
	
	for _, pattern := range fd.config.IncludePatterns {
		if fd.matchesPattern(relPath, pattern) {
			return true
		}
	}
	
	return false
}

// matchesPattern checks if a file path matches a glob-style pattern
func (fd *FileDiscovery) matchesPattern(path, pattern string) bool {
	// Convert to forward slashes for consistent matching
	path = filepath.ToSlash(path)
	pattern = filepath.ToSlash(pattern)
	
	// Use doublestar for robust glob matching
	matched, err := doublestar.Match(pattern, path)
	if err != nil {
		return false
	}
	
	return matched
}

// isHiddenFile checks if a file is hidden (starts with .)
func (fd *FileDiscovery) isHiddenFile(path string) bool {
	base := filepath.Base(path)
	return strings.HasPrefix(base, ".")
}

// isBinaryFile determines if a file is binary by reading a sample of its content
func (fd *FileDiscovery) isBinaryFile(path string) (bool, error) {
	file, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer file.Close()
	
	// Read first 512 bytes to determine if binary
	buffer := make([]byte, 512)
	n, err := file.Read(buffer)
	if err != nil && n == 0 {
		return false, err
	}
	
	// Truncate buffer to actual read size
	buffer = buffer[:n]
	
	// Check for null bytes (strong indicator of binary)
	for _, b := range buffer {
		if b == 0 {
			return true, nil
		}
	}
	
	// Check if content is valid UTF-8
	if !utf8.Valid(buffer) {
		return true, nil
	}
	
	// Check for high ratio of non-printable characters
	nonPrintable := 0
	for _, b := range buffer {
		if b < 32 && b != 9 && b != 10 && b != 13 { // Not tab, LF, or CR
			nonPrintable++
		}
	}
	
	// If more than 30% non-printable, consider binary
	if len(buffer) > 0 && float64(nonPrintable)/float64(len(buffer)) > 0.3 {
		return true, nil
	}
	
	return false, nil
}

// GetStats returns statistics about discovered files
func (fd *FileDiscovery) GetStats(results []FileResult) DiscoveryStats {
	stats := DiscoveryStats{
		TotalFiles: len(results),
	}
	
	for _, result := range results {
		stats.TotalSize += result.Size
		
		if result.IsBinary {
			stats.BinaryFiles++
		} else {
			stats.TextFiles++
		}
		
		if result.IsHidden {
			stats.HiddenFiles++
		}
		
		// Count by extension
		ext := strings.ToLower(filepath.Ext(result.Path))
		if ext != "" {
			if stats.Extensions == nil {
				stats.Extensions = make(map[string]int)
			}
			stats.Extensions[ext]++
		}
	}
	
	return stats
}

// DiscoveryStats holds statistics about file discovery
type DiscoveryStats struct {
	TotalFiles  int
	TextFiles   int
	BinaryFiles int
	HiddenFiles int
	TotalSize   int64
	Extensions  map[string]int
}