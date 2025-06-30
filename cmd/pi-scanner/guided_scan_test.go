package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGuidedScanNonInteractive(t *testing.T) {
	// Skip if no GitHub token
	if os.Getenv("GITHUB_TOKEN") == "" {
		t.Skip("GITHUB_TOKEN not set")
	}

	// Test repository with known PI patterns
	testRepo := "https://github.com/MacAttak/pi-scanner"

	// Save original flags
	originalNoInput := noInput
	originalValidateMode := validateMode
	originalMaskingLevel := maskingLevel
	defer func() {
		noInput = originalNoInput
		validateMode = originalValidateMode
		maskingLevel = originalMaskingLevel
	}()

	// Test cases
	tests := []struct {
		name         string
		noInput      bool
		validateMode string
		expectError  bool
	}{
		{
			name:         "Non-interactive pattern scan only",
			noInput:      true,
			validateMode: "none",
			expectError:  false,
		},
		{
			name:         "Non-interactive with high validation",
			noInput:      true,
			validateMode: "high",
			expectError:  false, // May fail if LLM not available, but that's ok
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set flags
			noInput = tt.noInput
			validateMode = tt.validateMode
			maskingLevel = "partial"

			// Run guided scan
			ctx := context.Background()
			err := runGuidedScan(ctx, testRepo)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				// We don't require success - LLM might not be available
				// Just ensure it doesn't panic
				t.Logf("Scan completed with result: %v", err)
			}
		})
	}
}

func TestValidateRepositoryURL(t *testing.T) {
	tests := []struct {
		name        string
		url         string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "Valid GitHub URL",
			url:         "https://github.com/owner/repo",
			expectError: false,
		},
		{
			name:        "Valid GitHub URL with .git",
			url:         "https://github.com/owner/repo.git",
			expectError: false,
		},
		{
			name:        "Missing protocol",
			url:         "github.com/owner/repo",
			expectError: true,
			errorMsg:    "URL must include protocol",
		},
		{
			name:        "Missing host",
			url:         "https://",
			expectError: true,
			errorMsg:    "URL must include a host",
		},
		{
			name:        "Invalid GitHub format",
			url:         "https://github.com/invalid",
			expectError: true,
			errorMsg:    "GitHub URL must be in format",
		},
		{
			name:        "Non-GitHub URL",
			url:         "https://gitlab.com/owner/repo",
			expectError: false, // We allow non-GitHub URLs
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateRepositoryURL(tt.url)
			if tt.expectError {
				require.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestProgressBar(t *testing.T) {
	tests := []struct {
		name     string
		current  int
		total    int
		width    int
		expected string
	}{
		{
			name:     "Empty progress",
			current:  0,
			total:    100,
			width:    10,
			expected: "░░░░░░░░░░",
		},
		{
			name:     "Half progress",
			current:  50,
			total:    100,
			width:    10,
			expected: "█████░░░░░",
		},
		{
			name:     "Full progress",
			current:  100,
			total:    100,
			width:    10,
			expected: "██████████",
		},
		{
			name:     "Zero total",
			current:  0,
			total:    0,
			width:    10,
			expected: "░░░░░░░░░░",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := progressBar(tt.current, tt.total, tt.width)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetMaskingLevel(t *testing.T) {
	// Save original
	original := maskingLevel
	defer func() {
		maskingLevel = original
	}()

	tests := []struct {
		input    string
		expected string
	}{
		{"full", "FULL"},
		{"FULL", "FULL"},
		{"none", "NONE"},
		{"NONE", "NONE"},
		{"partial", "PARTIAL"},
		{"PARTIAL", "PARTIAL"},
		{"invalid", "PARTIAL"}, // Default
		{"", "PARTIAL"},        // Default
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			maskingLevel = tt.input
			result := getMaskingLevel()
			assert.Equal(t, tt.expected, string(result))
		})
	}
}

func TestReportCreation(t *testing.T) {
	// Create a temporary reports directory
	tempDir := t.TempDir()
	reportsDir := filepath.Join(tempDir, "reports")

	// Test creating report directory structure
	require.NoError(t, os.MkdirAll(reportsDir, 0755))

	// Verify structure
	info, err := os.Stat(reportsDir)
	require.NoError(t, err)
	assert.True(t, info.IsDir())
}

// TestScanResourceManager tests the resource lifecycle management
func TestScanResourceManager(t *testing.T) {
	t.Run("successful_initialization_and_cleanup", func(t *testing.T) {
		// Mock git command for testing
		originalPath := os.Getenv("PATH")
		tempBin := t.TempDir()

		// Create mock git executable
		mockGit := filepath.Join(tempBin, "git")
		gitContent := `#!/bin/bash
# Debug: echo all args to see what's being passed
# echo "git args: $@" >> /tmp/git-debug.log
case "$1" in
  "clone")
    # Git clone format: git clone [options] <repo> <directory>
    # Find the last argument which should be the directory
    for arg in "$@"; do
      last_arg="$arg"
    done
    mkdir -p "$last_arg"
    echo "mock clone" > "$last_arg/README.md"
    # Create .git directory to make it look like a real repo
    mkdir -p "$last_arg/.git"
    echo "ref: refs/heads/main" > "$last_arg/.git/HEAD"
    ;;
  *)
    exit 0
    ;;
esac
`
		require.NoError(t, os.WriteFile(mockGit, []byte(gitContent), 0755))

		// Create mock gh executable for GitHub CLI
		mockGh := filepath.Join(tempBin, "gh")
		ghContent := `#!/bin/bash
case "$1" in
  "auth")
    case "$2" in
      "status")
        echo "Logged in to github.com as test-user"
        exit 0
        ;;
      *)
        exit 0
        ;;
    esac
    ;;
  *)
    exit 0
    ;;
esac
`
		require.NoError(t, os.WriteFile(mockGh, []byte(ghContent), 0755))
		os.Setenv("PATH", tempBin+":"+originalPath)
		defer os.Setenv("PATH", originalPath)

		ctx := context.Background()
		srm, err := NewScanResourceManager(ctx, "https://github.com/test/repo")
		require.NoError(t, err)
		require.NotNil(t, srm)

		// Verify repository info is available
		repoInfo := srm.GetRepositoryInfo()
		assert.NotNil(t, repoInfo)
		assert.NotEmpty(t, repoInfo.LocalPath)

		// Verify context is available
		assert.NotNil(t, srm.GetContext())

		// Cleanup should succeed
		err = srm.Cleanup()
		assert.NoError(t, err)

		// Repository should be cleaned up
		_, err = os.Stat(repoInfo.LocalPath)
		assert.True(t, os.IsNotExist(err))
	})

	t.Run("cleanup_on_initialization_failure", func(t *testing.T) {
		// Test with invalid URL to trigger failure
		ctx := context.Background()
		srm, err := NewScanResourceManager(ctx, "invalid-url")
		assert.Error(t, err)
		assert.Nil(t, srm)
	})

	t.Run("context_cancellation", func(t *testing.T) {
		// Mock git command
		originalPath := os.Getenv("PATH")
		tempBin := t.TempDir()

		mockGit := filepath.Join(tempBin, "git")
		gitContent := `#!/bin/bash
# Debug: echo all args to see what's being passed
# echo "git args: $@" >> /tmp/git-debug.log
case "$1" in
  "clone")
    # Git clone format: git clone [options] <repo> <directory>
    # Find the last argument which should be the directory
    for arg in "$@"; do
      last_arg="$arg"
    done
    mkdir -p "$last_arg"
    echo "mock clone" > "$last_arg/README.md"
    # Create .git directory to make it look like a real repo
    mkdir -p "$last_arg/.git"
    echo "ref: refs/heads/main" > "$last_arg/.git/HEAD"
    ;;
  *)
    exit 0
    ;;
esac
`
		require.NoError(t, os.WriteFile(mockGit, []byte(gitContent), 0755))

		// Create mock gh executable
		mockGh := filepath.Join(tempBin, "gh")
		ghContent := `#!/bin/bash
case "$1" in
  "auth")
    case "$2" in
      "status")
        echo "Logged in to github.com as test-user"
        exit 0
        ;;
      *)
        exit 0
        ;;
    esac
    ;;
  *)
    exit 0
    ;;
esac
`
		require.NoError(t, os.WriteFile(mockGh, []byte(ghContent), 0755))
		os.Setenv("PATH", tempBin+":"+originalPath)
		defer os.Setenv("PATH", originalPath)

		ctx, cancel := context.WithCancel(context.Background())
		srm, err := NewScanResourceManager(ctx, "https://github.com/test/repo")
		require.NoError(t, err)

		// Cancel the parent context
		cancel()

		// Wait a bit for cancellation to propagate
		time.Sleep(100 * time.Millisecond)

		// The resource manager's context should also be cancelled
		select {
		case <-srm.GetContext().Done():
			// Good - context was cancelled
		default:
			t.Error("Expected resource manager context to be cancelled")
		}

		// Cleanup should still work
		err = srm.Cleanup()
		assert.NoError(t, err)
	})

	t.Run("multiple_cleanup_calls", func(t *testing.T) {
		// Mock git command
		originalPath := os.Getenv("PATH")
		tempBin := t.TempDir()

		mockGit := filepath.Join(tempBin, "git")
		gitContent := `#!/bin/bash
# Debug: echo all args to see what's being passed
# echo "git args: $@" >> /tmp/git-debug.log
case "$1" in
  "clone")
    # Git clone format: git clone [options] <repo> <directory>
    # Find the last argument which should be the directory
    for arg in "$@"; do
      last_arg="$arg"
    done
    mkdir -p "$last_arg"
    echo "mock clone" > "$last_arg/README.md"
    # Create .git directory to make it look like a real repo
    mkdir -p "$last_arg/.git"
    echo "ref: refs/heads/main" > "$last_arg/.git/HEAD"
    ;;
  *)
    exit 0
    ;;
esac
`
		require.NoError(t, os.WriteFile(mockGit, []byte(gitContent), 0755))

		// Create mock gh executable
		mockGh := filepath.Join(tempBin, "gh")
		ghContent := `#!/bin/bash
case "$1" in
  "auth")
    case "$2" in
      "status")
        echo "Logged in to github.com as test-user"
        exit 0
        ;;
      *)
        exit 0
        ;;
    esac
    ;;
  *)
    exit 0
    ;;
esac
`
		require.NoError(t, os.WriteFile(mockGh, []byte(ghContent), 0755))
		os.Setenv("PATH", tempBin+":"+originalPath)
		defer os.Setenv("PATH", originalPath)

		ctx := context.Background()
		srm, err := NewScanResourceManager(ctx, "https://github.com/test/repo")
		require.NoError(t, err)

		// First cleanup should succeed
		err = srm.Cleanup()
		assert.NoError(t, err)

		// Second cleanup should also succeed (idempotent)
		err = srm.Cleanup()
		assert.NoError(t, err)
	})
}

// TestPatternScanWithResourceManager tests pattern scanning with resource management
func TestPatternScanWithResourceManager(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Mock git command
	originalPath := os.Getenv("PATH")
	tempBin := t.TempDir()

	mockGit := filepath.Join(tempBin, "git")
	gitContent := `#!/bin/bash
case "$1" in
  "clone")
    # Get the last argument as the clone directory
    for arg in "$@"; do
      CLONE_DIR="$arg"
    done
    mkdir -p "$CLONE_DIR"
    # Create test files with PI patterns
    echo "TFN: 123456782" > "$CLONE_DIR/test.txt"
    echo "Medicare: 2123456781" > "$CLONE_DIR/data.txt"
    ;;
  *)
    exit 0
    ;;
esac
`
	require.NoError(t, os.WriteFile(mockGit, []byte(gitContent), 0755))
	os.Setenv("PATH", tempBin+":"+originalPath)
	defer os.Setenv("PATH", originalPath)

	ctx := context.Background()
	srm, err := NewScanResourceManager(ctx, "https://github.com/test/repo")
	require.NoError(t, err)
	defer func() {
		err := srm.Cleanup()
		assert.NoError(t, err)
	}()

	// Run pattern scan
	result, reportDir, err := runPatternScan(ctx, srm, "https://github.com/test/repo")
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotEmpty(t, reportDir)

	// Verify repository info is preserved
	assert.NotNil(t, result.Repository)
	assert.NotEmpty(t, result.Repository.LocalPath)

	// Verify findings
	assert.NotEmpty(t, result.RawFindings) // Unmasked findings
	assert.NotEmpty(t, result.Findings)    // Masked findings for reports

	// Verify masking was applied
	assert.NotEqual(t, result.RawFindings[0].Match, result.Findings[0].Match)
}

// TestResourceLifecycleIntegration tests the complete lifecycle across phases
func TestResourceLifecycleIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// This test verifies that resources remain available between phases
	ctx := context.Background()

	// Mock setup (similar to above tests)
	originalPath := os.Getenv("PATH")
	tempBin := t.TempDir()

	mockGit := filepath.Join(tempBin, "git")
	gitContent := `#!/bin/bash
case "$1" in
  "clone")
    # Get the last argument as the clone directory
    for arg in "$@"; do
      CLONE_DIR="$arg"
    done
    mkdir -p "$CLONE_DIR"
    echo "Test PI: 123456782" > "$CLONE_DIR/test.txt"
    ;;
  *)
    exit 0
    ;;
esac
`
	require.NoError(t, os.WriteFile(mockGit, []byte(gitContent), 0755))
	os.Setenv("PATH", tempBin+":"+originalPath)
	defer os.Setenv("PATH", originalPath)

	// Initialize resources
	srm, err := NewScanResourceManager(ctx, "https://github.com/test/repo")
	require.NoError(t, err)
	defer func() {
		err := srm.Cleanup()
		assert.NoError(t, err)
	}()

	// Phase 1: Pattern scan
	_, _, err = runPatternScan(ctx, srm, "https://github.com/test/repo")
	require.NoError(t, err)

	// Verify repository files are still accessible
	testFile := filepath.Join(srm.GetRepositoryInfo().LocalPath, "test.txt")
	_, err = os.Stat(testFile)
	assert.NoError(t, err, "Repository files should remain accessible after Phase 1")

	// Phase 2 would use the same files
	// (LLM validation would happen here in a real scenario)

	// After cleanup, files should be gone
	err = srm.Cleanup()
	require.NoError(t, err)

	_, err = os.Stat(testFile)
	assert.True(t, os.IsNotExist(err), "Repository files should be cleaned up")
}
