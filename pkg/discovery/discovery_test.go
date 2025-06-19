package discovery

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFileDiscovery_DiscoverFiles(t *testing.T) {
	// Create a temporary directory structure for testing
	tmpDir := t.TempDir()
	
	// Create test file structure
	files := map[string]string{
		"main.go":                    "package main",
		"src/app.js":                "console.log('test')",
		"src/config.py":             "API_KEY = 'secret'",
		"test/main_test.go":         "func TestMain(t *testing.T) {}",
		"vendor/lib.go":             "package vendor",
		"node_modules/package.json": `{"name": "test"}`,
		".git/config":               "[core]",
		"README.md":                 "# Project",
		"binary.exe":                string([]byte{0x00, 0x01, 0x02}), // Binary file
		"large.txt":                 "content",
		".env":                      "SECRET=value",
		"docs/guide.md":             "# Guide",
		"scripts/deploy.sh":         "#!/bin/bash",
		"config/settings.yaml":      "database: localhost",
	}
	
	for path, content := range files {
		fullPath := filepath.Join(tmpDir, path)
		require.NoError(t, os.MkdirAll(filepath.Dir(fullPath), 0755))
		require.NoError(t, os.WriteFile(fullPath, []byte(content), 0644))
	}
	
	tests := []struct {
		name         string
		config       Config
		expectedFiles []string
		excludedFiles []string
		description   string
	}{
		{
			name:   "default discovery",
			config: DefaultConfig(),
			expectedFiles: []string{
				"main.go", "src/app.js", "src/config.py", 
				"README.md", ".env", "docs/guide.md", 
				"scripts/deploy.sh", "config/settings.yaml",
			},
			excludedFiles: []string{
				"test/main_test.go", "vendor/lib.go", 
				"node_modules/package.json", ".git/config", "binary.exe",
			},
			description: "Should include source files and exclude test/vendor/binary",
		},
		{
			name: "include test files",
			config: Config{
				IncludePatterns: []string{"**/*"},
				ExcludePatterns: []string{"vendor/**", "node_modules/**", ".git/**"},
				ExcludeBinary:   true,
				MaxFileSize:     10 * 1024 * 1024,
				IncludeHidden:   true, // Include hidden files like .env
			},
			expectedFiles: []string{
				"main.go", "src/app.js", "src/config.py", "test/main_test.go",
				"README.md", ".env", "docs/guide.md", "scripts/deploy.sh", "config/settings.yaml",
			},
			excludedFiles: []string{
				"vendor/lib.go", "node_modules/package.json", ".git/config", "binary.exe",
			},
			description: "Should include test files when configured",
		},
		{
			name: "custom file extensions only",
			config: Config{
				IncludePatterns: []string{"**/*.go", "**/*.py"},
				ExcludePatterns: []string{"test/**", "vendor/**"},
				ExcludeBinary:   true,
				MaxFileSize:     10 * 1024 * 1024,
			},
			expectedFiles: []string{"main.go", "src/config.py"},
			excludedFiles: []string{
				"src/app.js", "test/main_test.go", "vendor/lib.go", 
				"README.md", ".env", "docs/guide.md",
			},
			description: "Should only include specified file extensions",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			discovery := NewFileDiscovery(tt.config)
			
			results, err := discovery.DiscoverFiles(context.Background(), tmpDir)
			require.NoError(t, err, tt.description)
			
			// Check expected files are included
			resultPaths := make(map[string]bool)
			for _, result := range results {
				relPath, _ := filepath.Rel(tmpDir, result.Path)
				resultPaths[filepath.ToSlash(relPath)] = true
			}
			
			for _, expectedFile := range tt.expectedFiles {
				assert.True(t, resultPaths[expectedFile], 
					"Expected file %s should be included: %s", expectedFile, tt.description)
			}
			
			// Check excluded files are not included
			for _, excludedFile := range tt.excludedFiles {
				assert.False(t, resultPaths[excludedFile], 
					"Excluded file %s should not be included: %s", excludedFile, tt.description)
			}
		})
	}
}

func TestFileDiscovery_EdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		setupFunc   func(string) error
		config      Config
		expectError bool
		description string
	}{
		{
			name: "empty directory",
			setupFunc: func(dir string) error {
				return nil // No files
			},
			config:      DefaultConfig(),
			expectError: false,
			description: "Should handle empty directory gracefully",
		},
		{
			name: "nonexistent directory",
			setupFunc: func(dir string) error {
				return os.RemoveAll(dir) // Remove the directory
			},
			config:      DefaultConfig(),
			expectError: true,
			description: "Should return error for nonexistent directory",
		},
		{
			name: "permission denied directory",
			setupFunc: func(dir string) error {
				// Create a directory with no read permissions
				restrictedDir := filepath.Join(dir, "restricted")
				if err := os.MkdirAll(restrictedDir, 0755); err != nil {
					return err
				}
				if err := os.WriteFile(filepath.Join(restrictedDir, "secret.txt"), []byte("content"), 0644); err != nil {
					return err
				}
				return os.Chmod(restrictedDir, 0000) // No permissions
			},
			config:      DefaultConfig(),
			expectError: false, // Should skip permission denied directories
			description: "Should skip directories without read permissions",
		},
		{
			name: "symlink handling",
			setupFunc: func(dir string) error {
				// Create a file and symlink to it
				originalFile := filepath.Join(dir, "original.txt")
				if err := os.WriteFile(originalFile, []byte("content"), 0644); err != nil {
					return err
				}
				return os.Symlink(originalFile, filepath.Join(dir, "link.txt"))
			},
			config:      DefaultConfig(),
			expectError: false,
			description: "Should handle symlinks appropriately",
		},
		{
			name: "very long filename",
			setupFunc: func(dir string) error {
				// Create file with very long name
				longName := filepath.Join(dir, "very_long_filename_that_exceeds_normal_limits_"+
					"and_continues_for_a_very_long_time_to_test_edge_cases_in_file_discovery.txt")
				return os.WriteFile(longName, []byte("content"), 0644)
			},
			config:      DefaultConfig(),
			expectError: false,
			description: "Should handle very long filenames",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			
			if tt.setupFunc != nil {
				err := tt.setupFunc(tmpDir)
				if err != nil && tt.name != "permission denied directory" {
					require.NoError(t, err, "Setup function should not fail")
				}
			}
			
			discovery := NewFileDiscovery(tt.config)
			results, err := discovery.DiscoverFiles(context.Background(), tmpDir)
			
			if tt.expectError {
				assert.Error(t, err, tt.description)
			} else {
				assert.NoError(t, err, tt.description)
				// Results should not be nil, but can be empty
				assert.NotNil(t, results, "Results should not be nil")
			}
			
			// Cleanup permission denied directory for next tests
			if tt.name == "permission denied directory" {
				restrictedDir := filepath.Join(tmpDir, "restricted")
				os.Chmod(restrictedDir, 0755) // Restore permissions for cleanup
			}
		})
	}
}

func TestFileDiscovery_Performance(t *testing.T) {
	// Create a large directory structure
	tmpDir := t.TempDir()
	
	// Create many nested directories and files
	for i := 0; i < 10; i++ {
		subDir := filepath.Join(tmpDir, "dir"+fmt.Sprintf("%d", i))
		require.NoError(t, os.MkdirAll(subDir, 0755))
		
		for j := 0; j < 50; j++ {
			fileName := filepath.Join(subDir, fmt.Sprintf("file%d.txt", j))
			require.NoError(t, os.WriteFile(fileName, []byte("content"), 0644))
		}
	}
	
	discovery := NewFileDiscovery(DefaultConfig())
	
	// Test with timeout to ensure it completes quickly
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	start := time.Now()
	results, err := discovery.DiscoverFiles(ctx, tmpDir)
	duration := time.Since(start)
	
	require.NoError(t, err)
	assert.NotEmpty(t, results)
	assert.Less(t, duration, 3*time.Second, "Discovery should complete quickly")
	assert.Equal(t, 500, len(results), "Should find all 500 files")
}

func TestFileDiscovery_Cancellation(t *testing.T) {
	tmpDir := t.TempDir()
	
	// Create some files
	for i := 0; i < 10; i++ {
		fileName := filepath.Join(tmpDir, fmt.Sprintf("file%d.txt", i))
		require.NoError(t, os.WriteFile(fileName, []byte("content"), 0644))
	}
	
	discovery := NewFileDiscovery(DefaultConfig())
	
	// Create a context that gets cancelled immediately
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately
	
	results, err := discovery.DiscoverFiles(ctx, tmpDir)
	
	// Should handle cancellation gracefully
	assert.Error(t, err, "Should return error when context is cancelled")
	assert.ErrorIs(t, err, context.Canceled, "Error should be context cancellation")
	assert.Nil(t, results, "Results should be nil when cancelled")
}

func TestFileDiscovery_BinaryDetection(t *testing.T) {
	tmpDir := t.TempDir()
	
	files := map[string][]byte{
		"text.txt":     []byte("This is a text file with readable content"),
		"binary.exe":   {0x00, 0x01, 0x02, 0x03, 0xFF, 0xFE}, // Binary content
		"utf8.txt":     []byte("UTF-8 content: 测试 🔑"), 
		"empty.txt":    {},
		"mixed.dat":    append([]byte("Text start"), []byte{0x00, 0x01, 0x02}...),
		"image.jpg":    {0xFF, 0xD8, 0xFF, 0xE0}, // JPEG header
	}
	
	for name, content := range files {
		path := filepath.Join(tmpDir, name)
		require.NoError(t, os.WriteFile(path, content, 0644))
	}
	
	config := Config{
		IncludePatterns: []string{"**/*"},
		ExcludePatterns: []string{},
		ExcludeBinary:   true,
		MaxFileSize:     10 * 1024 * 1024,
	}
	
	discovery := NewFileDiscovery(config)
	results, err := discovery.DiscoverFiles(context.Background(), tmpDir)
	require.NoError(t, err)
	
	// Check which files are included
	resultPaths := make(map[string]bool)
	for _, result := range results {
		name := filepath.Base(result.Path)
		resultPaths[name] = true
		
		// Verify binary detection is set correctly
		if name == "binary.exe" || name == "image.jpg" {
			assert.True(t, result.IsBinary, "Binary files should be detected as binary")
		} else {
			assert.False(t, result.IsBinary, "Text files should not be detected as binary")
		}
	}
	
	// With ExcludeBinary=true, binary files should be excluded
	assert.False(t, resultPaths["binary.exe"], "Binary files should be excluded")
	assert.False(t, resultPaths["image.jpg"], "Image files should be excluded as binary")
	assert.True(t, resultPaths["text.txt"], "Text files should be included")
	assert.True(t, resultPaths["utf8.txt"], "UTF-8 files should be included")
	assert.True(t, resultPaths["empty.txt"], "Empty files should be included")
}

func TestFileDiscovery_SizeLimit(t *testing.T) {
	tmpDir := t.TempDir()
	
	// Create files of different sizes
	smallContent := []byte("small file")
	largeContent := make([]byte, 5*1024*1024) // 5MB
	for i := range largeContent {
		largeContent[i] = byte('A' + (i % 26))
	}
	
	files := map[string][]byte{
		"small.txt": smallContent,
		"large.txt": largeContent,
	}
	
	for name, content := range files {
		path := filepath.Join(tmpDir, name)
		require.NoError(t, os.WriteFile(path, content, 0644))
	}
	
	config := Config{
		IncludePatterns: []string{"**/*"},
		ExcludePatterns: []string{},
		ExcludeBinary:   false,
		MaxFileSize:     1024 * 1024, // 1MB limit
	}
	
	discovery := NewFileDiscovery(config)
	results, err := discovery.DiscoverFiles(context.Background(), tmpDir)
	require.NoError(t, err)
	
	// Check size filtering
	resultPaths := make(map[string]bool)
	for _, result := range results {
		name := filepath.Base(result.Path)
		resultPaths[name] = true
		
		// Verify size information
		assert.LessOrEqual(t, result.Size, int64(config.MaxFileSize), 
			"File size should be within limit")
	}
	
	assert.True(t, resultPaths["small.txt"], "Small files should be included")
	assert.False(t, resultPaths["large.txt"], "Large files should be excluded")
}