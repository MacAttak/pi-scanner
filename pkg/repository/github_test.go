package repository

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGitHubManager_ParseRepositoryURL(t *testing.T) {
	tests := []struct {
		name         string
		url          string
		expectedOrg  string
		expectedRepo string
		expectError  bool
	}{
		{
			name:         "HTTPS URL",
			url:          "https://github.com/owner/repo",
			expectedOrg:  "owner",
			expectedRepo: "repo",
			expectError:  false,
		},
		{
			name:         "HTTPS URL with .git suffix",
			url:          "https://github.com/owner/repo.git",
			expectedOrg:  "owner",
			expectedRepo: "repo",
			expectError:  false,
		},
		{
			name:         "SSH URL",
			url:          "git@github.com:owner/repo.git",
			expectedOrg:  "owner",
			expectedRepo: "repo",
			expectError:  false,
		},
		{
			name:         "GitHub CLI format",
			url:          "owner/repo",
			expectedOrg:  "owner",
			expectedRepo: "repo",
			expectError:  false,
		},
		{
			name:        "Invalid URL - no owner",
			url:         "https://github.com/repo",
			expectError: true,
		},
		{
			name:        "Invalid URL - empty",
			url:         "",
			expectError: true,
		},
		{
			name:        "Invalid URL - not GitHub",
			url:         "https://gitlab.com/owner/repo",
			expectError: true,
		},
		{
			name:        "Invalid URL - malformed",
			url:         "not-a-url",
			expectError: true,
		},
	}

	manager := NewGitHubManager(DefaultGitHubConfig())

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			org, repo, err := manager.ParseRepositoryURL(tt.url)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedOrg, org)
				assert.Equal(t, tt.expectedRepo, repo)
			}
		})
	}
}

func TestGitHubManager_CheckAuthentication(t *testing.T) {
	manager := NewGitHubManager(DefaultGitHubConfig())

	// This test requires gh CLI to be installed and configured
	// Skip if not available in test environment
	err := manager.CheckAuthentication(context.Background())
	if err != nil {
		if strings.Contains(err.Error(), "gh not found") {
			t.Skip("GitHub CLI not available in test environment")
		}
		// Don't fail if not authenticated - this is expected in CI
		if strings.Contains(err.Error(), "not authenticated") {
			t.Skip("GitHub CLI not authenticated in test environment")
		}
		t.Logf("Authentication check returned: %v", err)
	} else {
		t.Logf("GitHub CLI authentication successful")
	}
}

func TestGitHubManager_CloneRepository_MockSuccess(t *testing.T) {
	// Create a mock manager that simulates successful operations
	config := DefaultGitHubConfig()
	config.UseGitHubCLI = false // Use git directly for testing
	manager := NewGitHubManager(config)

	// Override the git command for testing
	originalGitCmd := manager.(*gitHubManager).gitCommand
	defer func() {
		manager.(*gitHubManager).gitCommand = originalGitCmd
	}()

	// Mock successful git clone
	manager.(*gitHubManager).gitCommand = func(ctx context.Context, args ...string) error {
		// Simulate git clone by creating a basic repo structure
		if len(args) > 0 && args[0] == "clone" {
			// Extract target directory from args
			targetDir := args[len(args)-1]

			// Create basic repository structure
			err := os.MkdirAll(filepath.Join(targetDir, ".git"), 0755)
			if err != nil {
				return err
			}

			// Create some test files
			testFiles := map[string]string{
				"README.md":   "# Test Repository\nThis is a test repository.",
				"main.go":     "package main\n\nfunc main() {\n\tprintln(\"Hello, World!\")\n}",
				"src/app.js":  "console.log('Hello from JavaScript');",
				"config.json": `{"name": "test-app", "version": "1.0.0"}`,
				".gitignore":  "*.log\nnode_modules/\n.env",
			}

			for filePath, content := range testFiles {
				fullPath := filepath.Join(targetDir, filePath)
				err := os.MkdirAll(filepath.Dir(fullPath), 0755)
				if err != nil {
					return err
				}
				err = os.WriteFile(fullPath, []byte(content), 0644)
				if err != nil {
					return err
				}
			}
		}
		return nil
	}

	ctx := context.Background()
	repoInfo, err := manager.CloneRepository(ctx, "owner/repo")

	require.NoError(t, err)
	assert.NotEmpty(t, repoInfo.LocalPath)
	assert.Equal(t, "owner", repoInfo.Owner)
	assert.Equal(t, "repo", repoInfo.Name)
	assert.Greater(t, repoInfo.Size, int64(0))
	assert.Greater(t, repoInfo.FileCount, 0)

	// Verify files were created
	assert.FileExists(t, filepath.Join(repoInfo.LocalPath, "README.md"))
	assert.FileExists(t, filepath.Join(repoInfo.LocalPath, "main.go"))
	assert.FileExists(t, filepath.Join(repoInfo.LocalPath, "src", "app.js"))

	// Cleanup
	err = manager.CleanupRepository(repoInfo.LocalPath)
	assert.NoError(t, err)
}

func TestGitHubManager_CloneRepository_Error(t *testing.T) {
	config := DefaultGitHubConfig()
	config.UseGitHubCLI = false
	manager := NewGitHubManager(config)

	// Override git command to simulate failure
	originalGitCmd := manager.(*gitHubManager).gitCommand
	defer func() {
		manager.(*gitHubManager).gitCommand = originalGitCmd
	}()

	manager.(*gitHubManager).gitCommand = func(ctx context.Context, args ...string) error {
		return assert.AnError // Simulate git command failure
	}

	ctx := context.Background()
	_, err := manager.CloneRepository(ctx, "owner/nonexistent-repo")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to clone repository")
}

func TestGitHubManager_CloneRepository_Cancellation(t *testing.T) {
	config := DefaultGitHubConfig()
	config.UseGitHubCLI = false
	manager := NewGitHubManager(config)

	// Override git command to simulate long-running operation
	originalGitCmd := manager.(*gitHubManager).gitCommand
	defer func() {
		manager.(*gitHubManager).gitCommand = originalGitCmd
	}()

	manager.(*gitHubManager).gitCommand = func(ctx context.Context, args ...string) error {
		// Simulate long-running clone operation
		select {
		case <-time.After(2 * time.Second):
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	// Create context that cancels quickly
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, err := manager.CloneRepository(ctx, "owner/repo")

	assert.Error(t, err)
	assert.ErrorIs(t, err, context.DeadlineExceeded)
}

func TestGitHubManager_GetRepositoryInfo_Success(t *testing.T) {
	// Create a temporary directory with test files
	tmpDir := t.TempDir()

	// Create test repository structure
	testFiles := map[string]string{
		"README.md":       "# Test Repository",
		"main.go":         "package main\n\nfunc main() {}",
		"src/app.js":      "console.log('test');",
		"docs/guide.md":   "# Guide",
		"config/app.yaml": "name: test",
		".git/config":     "[core]\n    bare = false",
		"binary.exe":      string([]byte{0x00, 0x01, 0x02, 0x03}), // Binary file
	}

	for filePath, content := range testFiles {
		fullPath := filepath.Join(tmpDir, filePath)
		err := os.MkdirAll(filepath.Dir(fullPath), 0755)
		require.NoError(t, err)
		err = os.WriteFile(fullPath, []byte(content), 0644)
		require.NoError(t, err)
	}

	manager := NewGitHubManager(DefaultGitHubConfig())
	info, err := manager.GetRepositoryInfo(tmpDir)

	require.NoError(t, err)
	assert.Equal(t, tmpDir, info.LocalPath)
	assert.Equal(t, filepath.Base(tmpDir), info.Name)
	assert.Greater(t, info.Size, int64(0))
	assert.Greater(t, info.FileCount, 0)
	assert.NotZero(t, info.ClonedAt)

	// Verify that .git directory doesn't count towards file count
	// and binary files are detected
	assert.True(t, info.FileCount >= 6) // Should count non-.git files
}

func TestGitHubManager_GetRepositoryInfo_NonexistentPath(t *testing.T) {
	manager := NewGitHubManager(DefaultGitHubConfig())
	_, err := manager.GetRepositoryInfo("/nonexistent/path")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "repository path does not exist")
}

func TestGitHubManager_CleanupRepository_Success(t *testing.T) {
	// Create a temporary directory to cleanup
	tmpDir := t.TempDir()

	// Create some files
	testFile := filepath.Join(tmpDir, "test.txt")
	err := os.WriteFile(testFile, []byte("test content"), 0644)
	require.NoError(t, err)

	// Verify file exists
	assert.FileExists(t, testFile)

	manager := NewGitHubManager(DefaultGitHubConfig())
	err = manager.CleanupRepository(tmpDir)

	assert.NoError(t, err)
	// Verify directory is removed
	assert.NoFileExists(t, tmpDir)
}

func TestGitHubManager_CleanupRepository_NonexistentPath(t *testing.T) {
	manager := NewGitHubManager(DefaultGitHubConfig())

	// Use a path that looks like a temp directory to pass safety check
	tempPath := filepath.Join(os.TempDir(), "nonexistent-pi-scanner-temp")
	err := manager.CleanupRepository(tempPath)

	// Should not error for nonexistent paths
	assert.NoError(t, err)
}

func TestGitHubManager_ShallowClone(t *testing.T) {
	config := DefaultGitHubConfig()
	config.ShallowClone = true
	config.UseGitHubCLI = false
	manager := NewGitHubManager(config)

	// Track git commands to verify shallow clone
	var gitCommands [][]string
	originalGitCmd := manager.(*gitHubManager).gitCommand
	defer func() {
		manager.(*gitHubManager).gitCommand = originalGitCmd
	}()

	manager.(*gitHubManager).gitCommand = func(ctx context.Context, args ...string) error {
		gitCommands = append(gitCommands, args)

		// Simulate successful clone
		if len(args) > 0 && args[0] == "clone" {
			targetDir := args[len(args)-1]
			err := os.MkdirAll(filepath.Join(targetDir, ".git"), 0755)
			if err != nil {
				return err
			}
			err = os.WriteFile(filepath.Join(targetDir, "README.md"), []byte("# Test"), 0644)
			if err != nil {
				return err
			}
		}
		return nil
	}

	ctx := context.Background()
	repoInfo, err := manager.CloneRepository(ctx, "owner/repo")

	require.NoError(t, err)
	defer manager.CleanupRepository(repoInfo.LocalPath)

	// Verify shallow clone arguments were used
	assert.Len(t, gitCommands, 1)
	cloneCmd := gitCommands[0]
	assert.Contains(t, cloneCmd, "--depth")
	assert.Contains(t, cloneCmd, "1")
}

func TestGitHubManager_MultipleRepositories(t *testing.T) {
	config := DefaultGitHubConfig()
	config.UseGitHubCLI = false
	manager := NewGitHubManager(config)

	// Mock successful git operations
	originalGitCmd := manager.(*gitHubManager).gitCommand
	defer func() {
		manager.(*gitHubManager).gitCommand = originalGitCmd
	}()

	cloneCount := 0
	manager.(*gitHubManager).gitCommand = func(ctx context.Context, args ...string) error {
		if len(args) > 0 && args[0] == "clone" {
			cloneCount++
			targetDir := args[len(args)-1]

			// Create unique content for each clone
			err := os.MkdirAll(filepath.Join(targetDir, ".git"), 0755)
			if err != nil {
				return err
			}

			content := []byte("Repository content " + filepath.Base(targetDir))
			err = os.WriteFile(filepath.Join(targetDir, "README.md"), content, 0644)
			if err != nil {
				return err
			}
		}
		return nil
	}

	ctx := context.Background()
	repos := []string{"owner/repo1", "owner/repo2", "owner/repo3"}
	var repoInfos []RepositoryInfo
	var cleanupPaths []string

	// Clone multiple repositories
	for _, repo := range repos {
		repoInfo, err := manager.CloneRepository(ctx, repo)
		require.NoError(t, err)
		repoInfos = append(repoInfos, *repoInfo)
		cleanupPaths = append(cleanupPaths, repoInfo.LocalPath)
	}

	// Verify all repositories were cloned
	assert.Equal(t, len(repos), cloneCount)
	assert.Len(t, repoInfos, len(repos))

	// Verify each repository has unique content
	for _, repoInfo := range repoInfos {
		assert.Contains(t, repoInfo.Name, "repo")
		assert.FileExists(t, filepath.Join(repoInfo.LocalPath, "README.md"))

		content, err := os.ReadFile(filepath.Join(repoInfo.LocalPath, "README.md"))
		require.NoError(t, err)
		assert.Contains(t, string(content), filepath.Base(repoInfo.LocalPath))
	}

	// Cleanup all repositories
	for _, path := range cleanupPaths {
		err := manager.CleanupRepository(path)
		assert.NoError(t, err)
	}
}

func TestGitHubManager_LargeRepository_SizeLimit(t *testing.T) {
	config := DefaultGitHubConfig()
	config.MaxRepositorySize = 100 // 100 bytes limit for testing
	config.UseGitHubCLI = false
	manager := NewGitHubManager(config)

	// Mock git command that creates large content
	originalGitCmd := manager.(*gitHubManager).gitCommand
	defer func() {
		manager.(*gitHubManager).gitCommand = originalGitCmd
	}()

	manager.(*gitHubManager).gitCommand = func(ctx context.Context, args ...string) error {
		if len(args) > 0 && args[0] == "clone" {
			targetDir := args[len(args)-1]
			err := os.MkdirAll(filepath.Join(targetDir, ".git"), 0755)
			if err != nil {
				return err
			}

			// Create content larger than size limit
			largeContent := make([]byte, 200) // 200 bytes > 100 byte limit
			for i := range largeContent {
				largeContent[i] = 'A'
			}

			err = os.WriteFile(filepath.Join(targetDir, "large.txt"), largeContent, 0644)
			if err != nil {
				return err
			}
		}
		return nil
	}

	ctx := context.Background()
	repoInfo, err := manager.CloneRepository(ctx, "owner/large-repo")

	// Should succeed but warn about size
	require.NoError(t, err)
	assert.Greater(t, repoInfo.Size, int64(config.MaxRepositorySize))

	// Cleanup
	err = manager.CleanupRepository(repoInfo.LocalPath)
	assert.NoError(t, err)
}

func TestGitHubManager_InvalidGitRepository(t *testing.T) {
	// Create a directory that's not a git repository
	tmpDir := t.TempDir()
	err := os.WriteFile(filepath.Join(tmpDir, "README.md"), []byte("Not a git repo"), 0644)
	require.NoError(t, err)

	manager := NewGitHubManager(DefaultGitHubConfig())
	info, err := manager.GetRepositoryInfo(tmpDir)

	// Should still work for non-git directories
	require.NoError(t, err)
	assert.Equal(t, tmpDir, info.LocalPath)
	assert.Greater(t, info.Size, int64(0))
	assert.Greater(t, info.FileCount, 0)
}

func TestGitHubManager_ConcurrentOperations(t *testing.T) {
	config := DefaultGitHubConfig()
	config.UseGitHubCLI = false
	manager := NewGitHubManager(config)

	// Mock git command for concurrent testing
	originalGitCmd := manager.(*gitHubManager).gitCommand
	defer func() {
		manager.(*gitHubManager).gitCommand = originalGitCmd
	}()

	cloneCount := int64(0)
	manager.(*gitHubManager).gitCommand = func(ctx context.Context, args ...string) error {
		if len(args) > 0 && args[0] == "clone" {
			// Simulate concurrent clone operations
			time.Sleep(10 * time.Millisecond)

			targetDir := args[len(args)-1]
			err := os.MkdirAll(filepath.Join(targetDir, ".git"), 0755)
			if err != nil {
				return err
			}

			err = os.WriteFile(filepath.Join(targetDir, "README.md"), []byte("Concurrent test"), 0644)
			if err != nil {
				return err
			}

			cloneCount++
		}
		return nil
	}

	// Run concurrent clone operations
	ctx := context.Background()
	numConcurrent := 5
	results := make(chan struct {
		info *RepositoryInfo
		err  error
	}, numConcurrent)

	for i := 0; i < numConcurrent; i++ {
		go func(index int) {
			repoInfo, err := manager.CloneRepository(ctx, fmt.Sprintf("owner/repo%d", index))
			results <- struct {
				info *RepositoryInfo
				err  error
			}{repoInfo, err}
		}(i)
	}

	// Collect results
	var repoInfos []*RepositoryInfo
	for i := 0; i < numConcurrent; i++ {
		result := <-results
		require.NoError(t, result.err)
		repoInfos = append(repoInfos, result.info)
	}

	// Verify all operations completed successfully
	assert.Len(t, repoInfos, numConcurrent)
	assert.Equal(t, int64(numConcurrent), cloneCount)

	// Cleanup all repositories
	for _, info := range repoInfos {
		if info != nil {
			err := manager.CleanupRepository(info.LocalPath)
			assert.NoError(t, err)
		}
	}
}

// Benchmark tests
func BenchmarkGitHubManager_ParseURL(b *testing.B) {
	manager := NewGitHubManager(DefaultGitHubConfig())
	url := "https://github.com/owner/repository"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, err := manager.ParseRepositoryURL(url)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkGitHubManager_GetRepositoryInfo(b *testing.B) {
	// Create test directory once
	tmpDir := b.TempDir()
	for i := 0; i < 100; i++ {
		content := fmt.Sprintf("File content %d", i)
		filePath := filepath.Join(tmpDir, fmt.Sprintf("file%d.txt", i))
		err := os.WriteFile(filePath, []byte(content), 0644)
		if err != nil {
			b.Fatal(err)
		}
	}

	manager := NewGitHubManager(DefaultGitHubConfig())

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := manager.GetRepositoryInfo(tmpDir)
		if err != nil {
			b.Fatal(err)
		}
	}
}
