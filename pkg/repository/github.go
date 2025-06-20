package repository

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

// GitHubManager handles GitHub repository operations
type GitHubManager interface {
	// Authentication
	CheckAuthentication(ctx context.Context) error

	// Repository operations
	CloneRepository(ctx context.Context, repoURL string) (*RepositoryInfo, error)
	GetRepositoryInfo(localPath string) (*RepositoryInfo, error)
	CleanupRepository(localPath string) error

	// URL parsing
	ParseRepositoryURL(url string) (owner, repo string, err error)
}

// RepositoryInfo contains information about a cloned repository
type RepositoryInfo struct {
	URL       string    `json:"url"`
	Owner     string    `json:"owner"`
	Name      string    `json:"name"`
	LocalPath string    `json:"local_path"`
	Size      int64     `json:"size"`       // Total size in bytes
	FileCount int       `json:"file_count"` // Number of files (excluding .git)
	ClonedAt  time.Time `json:"cloned_at"`
	IsShallow bool      `json:"is_shallow"` // Whether it's a shallow clone
}

// GitHubConfig configures GitHub repository operations
type GitHubConfig struct {
	// Authentication
	UseGitHubCLI  bool   // Use GitHub CLI (gh) for authenticated operations
	PersonalToken string // GitHub personal access token (alternative to CLI)

	// Clone options
	ShallowClone bool          // Use shallow clone (--depth 1)
	CloneTimeout time.Duration // Timeout for clone operations
	TempDir      string        // Base directory for temporary clones

	// Limits
	MaxRepositorySize int64 // Maximum repository size in bytes (0 = no limit)
	MaxFileCount      int   // Maximum number of files (0 = no limit)

	// Cleanup
	AutoCleanup bool // Automatically cleanup on errors
}

// DefaultGitHubConfig returns sensible defaults
func DefaultGitHubConfig() GitHubConfig {
	return GitHubConfig{
		UseGitHubCLI:      true,
		ShallowClone:      true,
		CloneTimeout:      5 * time.Minute,
		TempDir:           os.TempDir(),
		MaxRepositorySize: 1024 * 1024 * 1024, // 1GB
		MaxFileCount:      50000,              // 50k files
		AutoCleanup:       true,
	}
}

// gitHubManager implements GitHubManager
type gitHubManager struct {
	config     GitHubConfig
	gitCommand func(ctx context.Context, args ...string) error
}

// NewGitHubManager creates a new GitHub repository manager
func NewGitHubManager(config GitHubConfig) GitHubManager {
	manager := &gitHubManager{
		config: config,
	}

	// Set up git command function
	manager.gitCommand = manager.executeGitCommand

	return manager
}

// CheckAuthentication verifies GitHub authentication
func (g *gitHubManager) CheckAuthentication(ctx context.Context) error {
	if g.config.UseGitHubCLI {
		// Check GitHub CLI authentication
		cmd := exec.CommandContext(ctx, "gh", "auth", "status")
		output, err := cmd.CombinedOutput()
		if err != nil {
			if strings.Contains(string(output), "not logged into") {
				return fmt.Errorf("not authenticated with GitHub CLI: run 'gh auth login'")
			}
			if strings.Contains(err.Error(), "not found") {
				return fmt.Errorf("gh CLI not found: install from https://cli.github.com/")
			}
			return fmt.Errorf("GitHub CLI authentication check failed: %w", err)
		}
		return nil
	}

	if g.config.PersonalToken != "" {
		// TODO: Validate personal token by making an API call
		return nil
	}

	return fmt.Errorf("no authentication method configured")
}

// ParseRepositoryURL parses various GitHub URL formats
func (g *gitHubManager) ParseRepositoryURL(url string) (owner, repo string, err error) {
	if url == "" {
		return "", "", fmt.Errorf("empty repository URL")
	}

	// Remove .git suffix if present
	url = strings.TrimSuffix(url, ".git")

	// GitHub CLI format: owner/repo
	if matched, _ := regexp.MatchString(`^[a-zA-Z0-9]([a-zA-Z0-9._-]*[a-zA-Z0-9])?/[a-zA-Z0-9]([a-zA-Z0-9._-]*[a-zA-Z0-9])?$`, url); matched {
		parts := strings.Split(url, "/")
		if len(parts) == 2 {
			return parts[0], parts[1], nil
		}
	}

	// HTTPS URL: https://github.com/owner/repo
	httpsRegex := regexp.MustCompile(`^https://github\.com/([^/]+)/([^/]+)(?:/.*)?$`)
	if matches := httpsRegex.FindStringSubmatch(url); len(matches) == 3 {
		return matches[1], matches[2], nil
	}

	// SSH URL: git@github.com:owner/repo.git
	sshRegex := regexp.MustCompile(`^git@github\.com:([^/]+)/([^/]+)(?:\.git)?$`)
	if matches := sshRegex.FindStringSubmatch(url); len(matches) == 3 {
		return matches[1], matches[2], nil
	}

	return "", "", fmt.Errorf("invalid GitHub repository URL: %s", url)
}

// CloneRepository clones a GitHub repository to a temporary directory
func (g *gitHubManager) CloneRepository(ctx context.Context, repoURL string) (*RepositoryInfo, error) {
	// Parse repository URL
	owner, repo, err := g.ParseRepositoryURL(repoURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse repository URL: %w", err)
	}

	// Create temporary directory
	tempDir := filepath.Join(g.config.TempDir, fmt.Sprintf("pi-scanner-%d", time.Now().UnixNano()))
	cloneDir := filepath.Join(tempDir, repo)

	// Ensure parent directory exists
	err = os.MkdirAll(tempDir, 0755)
	if err != nil {
		return nil, fmt.Errorf("failed to create temporary directory: %w", err)
	}

	// Prepare clone arguments
	var cloneArgs []string
	cloneArgs = append(cloneArgs, "clone")

	if g.config.ShallowClone {
		cloneArgs = append(cloneArgs, "--depth", "1")
	}

	// Use GitHub CLI format or full URL
	if g.config.UseGitHubCLI {
		cloneArgs = append(cloneArgs, fmt.Sprintf("https://github.com/%s/%s.git", owner, repo))
	} else {
		cloneArgs = append(cloneArgs, repoURL)
	}

	cloneArgs = append(cloneArgs, cloneDir)

	// Clone with timeout
	cloneCtx := ctx
	if g.config.CloneTimeout > 0 {
		var cancel context.CancelFunc
		cloneCtx, cancel = context.WithTimeout(ctx, g.config.CloneTimeout)
		defer cancel()
	}

	// Execute clone
	err = g.gitCommand(cloneCtx, cloneArgs...)
	if err != nil {
		// Cleanup on failure if auto-cleanup is enabled
		if g.config.AutoCleanup {
			os.RemoveAll(tempDir)
		}
		return nil, fmt.Errorf("failed to clone repository: %w", err)
	}

	// Get repository information
	repoInfo, err := g.GetRepositoryInfo(cloneDir)
	if err != nil {
		// Cleanup on failure if auto-cleanup is enabled
		if g.config.AutoCleanup {
			os.RemoveAll(tempDir)
		}
		return nil, fmt.Errorf("failed to get repository information: %w", err)
	}

	// Update repository info with clone details
	repoInfo.URL = repoURL
	repoInfo.Owner = owner
	repoInfo.ClonedAt = time.Now()
	repoInfo.IsShallow = g.config.ShallowClone

	// Check size limits
	if g.config.MaxRepositorySize > 0 && repoInfo.Size > g.config.MaxRepositorySize {
		// Don't fail, but log warning
		// In a real implementation, you'd use a logger here
	}

	if g.config.MaxFileCount > 0 && repoInfo.FileCount > g.config.MaxFileCount {
		// Don't fail, but log warning
		// In a real implementation, you'd use a logger here
	}

	return repoInfo, nil
}

// GetRepositoryInfo gathers information about a local repository
func (g *gitHubManager) GetRepositoryInfo(localPath string) (*RepositoryInfo, error) {
	// Check if path exists
	if _, err := os.Stat(localPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("repository path does not exist: %s", localPath)
	}

	info := &RepositoryInfo{
		LocalPath: localPath,
		Name:      filepath.Base(localPath),
		ClonedAt:  time.Now(),
	}

	// Walk directory to calculate size and count files
	var totalSize int64
	var fileCount int

	err := filepath.WalkDir(localPath, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			// Skip permission errors and continue
			return nil
		}

		// Skip .git directory
		if entry.IsDir() && entry.Name() == ".git" {
			return fs.SkipDir
		}

		// Count files only (not directories)
		if !entry.IsDir() {
			fileInfo, err := entry.Info()
			if err == nil {
				totalSize += fileInfo.Size()
				fileCount++
			}
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to analyze repository: %w", err)
	}

	info.Size = totalSize
	info.FileCount = fileCount

	return info, nil
}

// CleanupRepository removes a cloned repository directory
func (g *gitHubManager) CleanupRepository(localPath string) error {
	if localPath == "" {
		return nil
	}

	// Safety check: ensure we're cleaning up a temporary directory or test directory
	isTempDir := strings.Contains(localPath, "pi-scanner-") ||
		strings.Contains(localPath, "Test") ||
		strings.Contains(localPath, os.TempDir())

	if !isTempDir {
		return fmt.Errorf("refusing to cleanup directory that doesn't appear to be a temporary dir: %s", localPath)
	}

	// Check if path exists
	if _, err := os.Stat(localPath); os.IsNotExist(err) {
		// Already cleaned up
		return nil
	}

	// Remove the entire directory tree
	err := os.RemoveAll(localPath)
	if err != nil {
		return fmt.Errorf("failed to cleanup repository: %w", err)
	}

	// Also try to remove parent temp directory if it's empty
	parentDir := filepath.Dir(localPath)
	if strings.Contains(parentDir, "pi-scanner-") {
		// Only try to remove if it exists and is empty
		if entries, err := os.ReadDir(parentDir); err == nil && len(entries) == 0 {
			os.Remove(parentDir) // Ignore error if removal fails
		}
	}

	return nil
}

// executeGitCommand executes a git command with the given arguments
func (g *gitHubManager) executeGitCommand(ctx context.Context, args ...string) error {
	cmd := exec.CommandContext(ctx, "git", args...)

	// Set working directory if needed
	// cmd.Dir = workingDir

	// Capture output for debugging
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git command failed: %w (output: %s)", err, string(output))
	}

	return nil
}

// RepositoryManager provides high-level repository management
type RepositoryManager struct {
	github      GitHubManager
	activeRepos map[string]*RepositoryInfo // Track active repositories
	mu          sync.RWMutex               // Protect activeRepos map
}

// NewRepositoryManager creates a new repository manager
func NewRepositoryManager(githubConfig GitHubConfig) *RepositoryManager {
	return &RepositoryManager{
		github:      NewGitHubManager(githubConfig),
		activeRepos: make(map[string]*RepositoryInfo),
	}
}

// CloneAndTrack clones a repository and tracks it for later cleanup
func (rm *RepositoryManager) CloneAndTrack(ctx context.Context, repoURL string) (*RepositoryInfo, error) {
	repoInfo, err := rm.github.CloneRepository(ctx, repoURL)
	if err != nil {
		return nil, err
	}

	// Track the repository with mutex protection
	rm.mu.Lock()
	rm.activeRepos[repoInfo.LocalPath] = repoInfo
	rm.mu.Unlock()

	return repoInfo, nil
}

// CleanupAll cleans up all tracked repositories
func (rm *RepositoryManager) CleanupAll() error {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	var errs []string

	for path, _ := range rm.activeRepos {
		if err := rm.github.CleanupRepository(path); err != nil {
			errs = append(errs, fmt.Sprintf("failed to cleanup %s: %v", path, err))
		}
		delete(rm.activeRepos, path)
	}

	if len(errs) > 0 {
		return fmt.Errorf("cleanup errors: %s", strings.Join(errs, "; "))
	}

	return nil
}

// GetActiveRepositories returns information about all active repositories
func (rm *RepositoryManager) GetActiveRepositories() map[string]*RepositoryInfo {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	// Return a copy to prevent external modification
	result := make(map[string]*RepositoryInfo)
	for k, v := range rm.activeRepos {
		result[k] = v
	}
	return result
}

// CheckAuthentication checks GitHub authentication
func (rm *RepositoryManager) CheckAuthentication(ctx context.Context) error {
	return rm.github.CheckAuthentication(ctx)
}
