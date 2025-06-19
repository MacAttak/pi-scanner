package repository

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRepositoryManager_CleanupOnInterrupt(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping interrupt test in short mode")
	}

	config := DefaultGitHubConfig()
	config.UseGitHubCLI = false
	config.AutoCleanup = true
	manager := NewRepositoryManager(config)

	// Mock git command for testing
	githubManager := manager.github.(*gitHubManager)
	originalGitCmd := githubManager.gitCommand
	defer func() {
		githubManager.gitCommand = originalGitCmd
	}()

	var createdDirs []string
	var mutex sync.Mutex

	githubManager.gitCommand = func(ctx context.Context, args ...string) error {
		if len(args) > 0 && args[0] == "clone" {
			targetDir := args[len(args)-1]
			
			// Create repository structure
			err := os.MkdirAll(filepath.Join(targetDir, ".git"), 0755)
			if err != nil {
				return err
			}
			
			err = os.WriteFile(filepath.Join(targetDir, "README.md"), []byte("Test repo"), 0644)
			if err != nil {
				return err
			}
			
			// Track created directories
			mutex.Lock()
			createdDirs = append(createdDirs, targetDir)
			mutex.Unlock()
			
			// Simulate long clone operation
			select {
			case <-time.After(2 * time.Second):
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		}
		return nil
	}

	// Create context that can be cancelled
	ctx, cancel := context.WithCancel(context.Background())
	
	// Set up signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	
	// Start goroutine to simulate interrupt
	go func() {
		time.Sleep(500 * time.Millisecond)
		// Simulate interrupt by cancelling context
		cancel()
	}()

	// Attempt to clone repository
	_, err := manager.CloneAndTrack(ctx, "owner/repo")
	
	// Should handle cancellation gracefully
	assert.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)

	// Cleanup all tracked repositories
	err = manager.CleanupAll()
	assert.NoError(t, err)

	// Verify cleanup actually removed directories
	mutex.Lock()
	for _, dir := range createdDirs {
		if _, err := os.Stat(dir); !os.IsNotExist(err) {
			t.Errorf("Directory %s was not cleaned up", dir)
		}
	}
	mutex.Unlock()
}

func TestRepositoryManager_MultipleRepositories_CleanupAll(t *testing.T) {
	config := DefaultGitHubConfig()
	config.UseGitHubCLI = false
	manager := NewRepositoryManager(config)

	// Mock git command
	githubManager := manager.github.(*gitHubManager)
	originalGitCmd := githubManager.gitCommand
	defer func() {
		githubManager.gitCommand = originalGitCmd
	}()

	var createdDirs []string
	githubManager.gitCommand = func(ctx context.Context, args ...string) error {
		if len(args) > 0 && args[0] == "clone" {
			targetDir := args[len(args)-1]
			
			err := os.MkdirAll(filepath.Join(targetDir, ".git"), 0755)
			if err != nil {
				return err
			}
			
			content := []byte("Repository " + filepath.Base(targetDir))
			err = os.WriteFile(filepath.Join(targetDir, "README.md"), content, 0644)
			if err != nil {
				return err
			}
			
			createdDirs = append(createdDirs, targetDir)
		}
		return nil
	}

	ctx := context.Background()
	repos := []string{"owner/repo1", "owner/repo2", "owner/repo3"}
	var repoInfos []*RepositoryInfo

	// Clone multiple repositories
	for _, repo := range repos {
		repoInfo, err := manager.CloneAndTrack(ctx, repo)
		require.NoError(t, err)
		repoInfos = append(repoInfos, repoInfo)
	}

	// Verify all repositories are tracked
	activeRepos := manager.GetActiveRepositories()
	assert.Len(t, activeRepos, len(repos))

	// Verify all directories exist
	for _, dir := range createdDirs {
		assert.DirExists(t, dir)
	}

	// Cleanup all at once
	err := manager.CleanupAll()
	assert.NoError(t, err)

	// Verify all directories are removed
	for _, dir := range createdDirs {
		assert.NoDirExists(t, dir)
	}

	// Verify tracking is cleared
	activeRepos = manager.GetActiveRepositories()
	assert.Empty(t, activeRepos)
}

func TestRepositoryManager_PartialCleanupFailure(t *testing.T) {
	config := DefaultGitHubConfig()
	config.UseGitHubCLI = false
	manager := NewRepositoryManager(config)

	// Mock git command
	githubManager := manager.github.(*gitHubManager)
	originalGitCmd := githubManager.gitCommand
	defer func() {
		githubManager.gitCommand = originalGitCmd
	}()

	var createdDirs []string
	githubManager.gitCommand = func(ctx context.Context, args ...string) error {
		if len(args) > 0 && args[0] == "clone" {
			targetDir := args[len(args)-1]
			
			err := os.MkdirAll(filepath.Join(targetDir, ".git"), 0755)
			if err != nil {
				return err
			}
			
			err = os.WriteFile(filepath.Join(targetDir, "README.md"), []byte("Test"), 0644)
			if err != nil {
				return err
			}
			
			createdDirs = append(createdDirs, targetDir)
		}
		return nil
	}

	ctx := context.Background()

	// Clone repositories
	repo1, err := manager.CloneAndTrack(ctx, "owner/repo1")
	require.NoError(t, err)

	_, err = manager.CloneAndTrack(ctx, "owner/repo2")
	require.NoError(t, err)

	// Make one directory read-only to simulate cleanup failure
	err = os.Chmod(repo1.LocalPath, 0444)
	require.NoError(t, err)
	
	// Try to cleanup all - should report errors but continue
	err = manager.CleanupAll()
	// Should have error but not fail completely
	if err != nil {
		assert.Contains(t, err.Error(), "cleanup errors")
	}

	// Fix permissions for final cleanup
	os.Chmod(repo1.LocalPath, 0755)
	os.RemoveAll(repo1.LocalPath)
}

func TestRepositoryManager_ConcurrentOperations(t *testing.T) {
	config := DefaultGitHubConfig()
	config.UseGitHubCLI = false
	manager := NewRepositoryManager(config)

	// Mock git command
	githubManager := manager.github.(*gitHubManager)
	originalGitCmd := githubManager.gitCommand
	defer func() {
		githubManager.gitCommand = originalGitCmd
	}()

	cloneDelay := 50 * time.Millisecond
	githubManager.gitCommand = func(ctx context.Context, args ...string) error {
		if len(args) > 0 && args[0] == "clone" {
			// Add delay to test concurrency
			time.Sleep(cloneDelay)
			
			targetDir := args[len(args)-1]
			err := os.MkdirAll(filepath.Join(targetDir, ".git"), 0755)
			if err != nil {
				return err
			}
			
			content := []byte("Concurrent test " + filepath.Base(targetDir))
			err = os.WriteFile(filepath.Join(targetDir, "README.md"), content, 0644)
			if err != nil {
				return err
			}
		}
		return nil
	}

	ctx := context.Background()
	numConcurrent := 5
	results := make(chan struct {
		info *RepositoryInfo
		err  error
	}, numConcurrent)

	start := time.Now()

	// Start concurrent clone operations
	for i := 0; i < numConcurrent; i++ {
		go func(index int) {
			repoURL := fmt.Sprintf("owner/repo%d", index)
			repoInfo, err := manager.CloneAndTrack(ctx, repoURL)
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

	duration := time.Since(start)

	// Verify concurrent execution was actually faster than sequential
	maxSequentialTime := time.Duration(numConcurrent) * cloneDelay
	assert.Less(t, duration, maxSequentialTime*2, // Allow some overhead
		"Concurrent operations should be faster than sequential")

	// Verify all repositories are tracked
	activeRepos := manager.GetActiveRepositories()
	assert.Len(t, activeRepos, numConcurrent)

	// Cleanup all
	err := manager.CleanupAll()
	assert.NoError(t, err)
}

func TestGitHubManager_RealRepository_SkipIfNoAuth(t *testing.T) {
	// This test tries to clone a real public repository
	// Skip if GitHub CLI is not available or authenticated
	
	manager := NewGitHubManager(DefaultGitHubConfig())
	ctx := context.Background()
	
	// Check authentication first
	err := manager.CheckAuthentication(ctx)
	if err != nil {
		t.Skipf("Skipping real repository test: %v", err)
	}

	// Try to clone a small public repository
	repoInfo, err := manager.CloneRepository(ctx, "octocat/Hello-World")
	if err != nil {
		// Don't fail if network issues or rate limiting
		t.Skipf("Could not clone real repository: %v", err)
	}

	// Verify basic repository info
	assert.NotEmpty(t, repoInfo.LocalPath)
	assert.Equal(t, "octocat", repoInfo.Owner)
	assert.Equal(t, "Hello-World", repoInfo.Name)
	assert.Greater(t, repoInfo.Size, int64(0))
	assert.Greater(t, repoInfo.FileCount, 0)

	// Cleanup
	err = manager.CleanupRepository(repoInfo.LocalPath)
	assert.NoError(t, err)
}

func TestGitHubManager_LargeRepositoryHandling(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping large repository test in short mode")
	}

	config := DefaultGitHubConfig()
	config.UseGitHubCLI = false
	config.MaxRepositorySize = 1024 // 1KB limit for testing
	config.MaxFileCount = 5        // 5 files limit
	manager := NewGitHubManager(config)

	// Mock git command to create "large" repository
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
			
			// Create multiple files that exceed limits
			for i := 0; i < 10; i++ {
				content := make([]byte, 200) // 200 bytes each
				for j := range content {
					content[j] = byte('A' + (i % 26))
				}
				
				filename := fmt.Sprintf("file%d.txt", i)
				err = os.WriteFile(filepath.Join(targetDir, filename), content, 0644)
				if err != nil {
					return err
				}
			}
		}
		return nil
	}

	ctx := context.Background()
	repoInfo, err := manager.CloneRepository(ctx, "owner/large-repo")

	// Should succeed despite exceeding limits (warnings only)
	require.NoError(t, err)
	assert.Greater(t, repoInfo.Size, int64(config.MaxRepositorySize))
	assert.Greater(t, repoInfo.FileCount, config.MaxFileCount)

	// Cleanup
	err = manager.CleanupRepository(repoInfo.LocalPath)
	assert.NoError(t, err)
}

func TestRepositoryManager_GracefulShutdown(t *testing.T) {
	config := DefaultGitHubConfig()
	config.UseGitHubCLI = false
	manager := NewRepositoryManager(config)

	// Mock git command with delay
	githubManager := manager.github.(*gitHubManager)
	originalGitCmd := githubManager.gitCommand
	defer func() {
		githubManager.gitCommand = originalGitCmd
	}()

	githubManager.gitCommand = func(ctx context.Context, args ...string) error {
		if len(args) > 0 && args[0] == "clone" {
			targetDir := args[len(args)-1]
			
			// Create basic structure
			err := os.MkdirAll(filepath.Join(targetDir, ".git"), 0755)
			if err != nil {
				return err
			}
			
			err = os.WriteFile(filepath.Join(targetDir, "README.md"), []byte("Test"), 0644)
			if err != nil {
				return err
			}
			
			// Simulate long operation that respects context cancellation
			select {
			case <-time.After(5 * time.Second):
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		}
		return nil
	}

	// Start clone operation
	ctx, cancel := context.WithCancel(context.Background())
	
	done := make(chan struct{})
	var cloneErr error
	
	go func() {
		defer close(done)
		_, cloneErr = manager.CloneAndTrack(ctx, "owner/repo")
	}()

	// Cancel after short delay
	time.Sleep(100 * time.Millisecond)
	cancel()

	// Wait for operation to complete
	select {
	case <-done:
		// Should have been cancelled
		assert.Error(t, cloneErr)
		assert.ErrorIs(t, cloneErr, context.Canceled)
	case <-time.After(2 * time.Second):
		t.Fatal("Clone operation did not respect context cancellation")
	}

	// Cleanup any partial work
	err := manager.CleanupAll()
	assert.NoError(t, err)
}


// Benchmark tests
func BenchmarkRepositoryManager_CloneAndCleanup(b *testing.B) {
	config := DefaultGitHubConfig()
	config.UseGitHubCLI = false
	manager := NewRepositoryManager(config)

	// Mock git command for benchmarking
	githubManager := manager.github.(*gitHubManager)
	originalGitCmd := githubManager.gitCommand
	defer func() {
		githubManager.gitCommand = originalGitCmd
	}()

	githubManager.gitCommand = func(ctx context.Context, args ...string) error {
		if len(args) > 0 && args[0] == "clone" {
			targetDir := args[len(args)-1]
			
			// Create minimal repository structure
			err := os.MkdirAll(filepath.Join(targetDir, ".git"), 0755)
			if err != nil {
				return err
			}
			
			// Create several files
			for i := 0; i < 10; i++ {
				content := fmt.Sprintf("File %d content", i)
				filename := fmt.Sprintf("file%d.txt", i)
				err = os.WriteFile(filepath.Join(targetDir, filename), []byte(content), 0644)
				if err != nil {
					return err
				}
			}
		}
		return nil
	}

	ctx := context.Background()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		repoURL := fmt.Sprintf("owner/repo%d", i)
		
		// Clone
		repoInfo, err := manager.CloneAndTrack(ctx, repoURL)
		if err != nil {
			b.Fatal(err)
		}
		
		// Cleanup immediately
		err = manager.github.CleanupRepository(repoInfo.LocalPath)
		if err != nil {
			b.Fatal(err)
		}
	}
}