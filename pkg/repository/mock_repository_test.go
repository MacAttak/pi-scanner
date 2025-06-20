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

// MockRepository creates a realistic mock repository structure for testing
type MockRepository struct {
	Name        string
	Owner       string
	Files       map[string]string
	LargeFiles  map[string]int64 // filename -> size in bytes
	BinaryFiles []string
	BaseDir     string
}

// CreateMockRepository creates a mock repository on disk
func CreateMockRepository(baseDir string, mock MockRepository) error {
	repoDir := filepath.Join(baseDir, mock.Name)

	// Create repository directory
	err := os.MkdirAll(repoDir, 0755)
	if err != nil {
		return err
	}

	// Create .git directory
	gitDir := filepath.Join(repoDir, ".git")
	err = os.MkdirAll(gitDir, 0755)
	if err != nil {
		return err
	}

	// Create basic git config
	gitConfig := filepath.Join(gitDir, "config")
	configContent := fmt.Sprintf(`[core]
	repositoryformatversion = 0
	filemode = true
	bare = false
[remote "origin"]
	url = https://github.com/%s/%s.git
	fetch = +refs/heads/*:refs/remotes/origin/*
[branch "main"]
	remote = origin
	merge = refs/heads/main
`, mock.Owner, mock.Name)

	err = os.WriteFile(gitConfig, []byte(configContent), 0644)
	if err != nil {
		return err
	}

	// Create regular files
	for relativePath, content := range mock.Files {
		fullPath := filepath.Join(repoDir, relativePath)

		// Create directory if needed
		err = os.MkdirAll(filepath.Dir(fullPath), 0755)
		if err != nil {
			return err
		}

		err = os.WriteFile(fullPath, []byte(content), 0644)
		if err != nil {
			return err
		}
	}

	// Create large files
	for filename, size := range mock.LargeFiles {
		fullPath := filepath.Join(repoDir, filename)

		// Create directory if needed
		err = os.MkdirAll(filepath.Dir(fullPath), 0755)
		if err != nil {
			return err
		}

		// Create large file filled with repeated content
		file, err := os.Create(fullPath)
		if err != nil {
			return err
		}

		// Write data in chunks to avoid memory issues
		chunkSize := int64(1024) // 1KB chunks
		pattern := []byte("Large file content data chunk. ")

		for written := int64(0); written < size; {
			toWrite := chunkSize
			if written+chunkSize > size {
				toWrite = size - written
			}

			// Repeat pattern to fill chunk
			chunk := make([]byte, toWrite)
			for i := int64(0); i < toWrite; i++ {
				chunk[i] = pattern[i%int64(len(pattern))]
			}

			_, err = file.Write(chunk)
			if err != nil {
				file.Close()
				return err
			}

			written += toWrite
		}

		file.Close()
	}

	// Create binary files
	for _, filename := range mock.BinaryFiles {
		fullPath := filepath.Join(repoDir, filename)

		// Create directory if needed
		err = os.MkdirAll(filepath.Dir(fullPath), 0755)
		if err != nil {
			return err
		}

		// Create binary content (image-like header)
		binaryContent := []byte{
			0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 0x4A, 0x46, // JPEG header
			0x49, 0x46, 0x00, 0x01, 0x01, 0x01, 0x00, 0x48,
			0x00, 0x48, 0x00, 0x00, 0xFF, 0xDB, 0x00, 0x43,
		}

		// Add some random binary data
		for i := 0; i < 100; i++ {
			binaryContent = append(binaryContent, byte(i%256))
		}

		err = os.WriteFile(fullPath, binaryContent, 0644)
		if err != nil {
			return err
		}
	}

	return nil
}

func TestMockRepository_SmallRepository(t *testing.T) {
	tmpDir := t.TempDir()

	mock := MockRepository{
		Name:  "small-repo",
		Owner: "testorg",
		Files: map[string]string{
			"README.md":       "# Small Repository\nThis is a test repository.",
			"main.go":         "package main\n\nfunc main() {\n\tprintln(\"Hello, World!\")\n}",
			"src/utils.go":    "package src\n\nfunc Helper() string {\n\treturn \"helper\"\n}",
			"config/app.yaml": "name: small-app\nversion: 1.0.0\nport: 8080",
			"docs/api.md":     "# API Documentation\n\n## Endpoints\n\n- GET /health",
			".gitignore":      "*.log\n*.tmp\nnode_modules/\n.env",
			"Dockerfile":      "FROM golang:1.19\nWORKDIR /app\nCOPY . .\nRUN go build -o main .",
		},
	}

	err := CreateMockRepository(tmpDir, mock)
	require.NoError(t, err)

	// Test repository operations
	manager := NewGitHubManager(DefaultGitHubConfig())
	repoPath := filepath.Join(tmpDir, mock.Name)

	info, err := manager.GetRepositoryInfo(repoPath)
	require.NoError(t, err)

	assert.Equal(t, repoPath, info.LocalPath)
	assert.Equal(t, mock.Name, info.Name)
	assert.Greater(t, info.Size, int64(0))
	assert.Equal(t, len(mock.Files), info.FileCount) // Should exclude .git files

	// Verify specific files exist
	assert.FileExists(t, filepath.Join(repoPath, "README.md"))
	assert.FileExists(t, filepath.Join(repoPath, "main.go"))
	assert.FileExists(t, filepath.Join(repoPath, "src", "utils.go"))
	assert.DirExists(t, filepath.Join(repoPath, ".git"))

	// Test cleanup
	err = manager.CleanupRepository(repoPath)
	assert.NoError(t, err)
	assert.NoDirExists(t, repoPath)
}

func TestMockRepository_LargeRepository(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping large repository test in short mode")
	}

	tmpDir := t.TempDir()

	mock := MockRepository{
		Name:  "large-repo",
		Owner: "testorg",
		Files: map[string]string{
			"README.md":    "# Large Repository\nThis repository contains large files.",
			"package.json": `{"name": "large-app", "version": "1.0.0"}`,
			"src/index.js": "console.log('Large application');",
		},
		LargeFiles: map[string]int64{
			"data/dataset.csv":     5 * 1024 * 1024,  // 5MB
			"assets/video.mp4":     10 * 1024 * 1024, // 10MB
			"logs/application.log": 2 * 1024 * 1024,  // 2MB
		},
		BinaryFiles: []string{
			"images/logo.png",
			"assets/binary.dat",
		},
	}

	err := CreateMockRepository(tmpDir, mock)
	require.NoError(t, err)

	manager := NewGitHubManager(DefaultGitHubConfig())
	repoPath := filepath.Join(tmpDir, mock.Name)

	info, err := manager.GetRepositoryInfo(repoPath)
	require.NoError(t, err)

	assert.Equal(t, mock.Name, info.Name)
	assert.Greater(t, info.Size, int64(15*1024*1024)) // At least 15MB from large files
	assert.Greater(t, info.FileCount, 5)

	// Verify large files exist and have correct size
	for filename, expectedSize := range mock.LargeFiles {
		fullPath := filepath.Join(repoPath, filename)
		assert.FileExists(t, fullPath)

		stat, err := os.Stat(fullPath)
		require.NoError(t, err)
		assert.Equal(t, expectedSize, stat.Size())
	}

	// Verify binary files exist
	for _, filename := range mock.BinaryFiles {
		fullPath := filepath.Join(repoPath, filename)
		assert.FileExists(t, fullPath)
	}

	// Test cleanup
	err = manager.CleanupRepository(repoPath)
	assert.NoError(t, err)
	assert.NoDirExists(t, repoPath)
}

func TestMockRepository_WithPIContent(t *testing.T) {
	tmpDir := t.TempDir()

	mock := MockRepository{
		Name:  "pi-content-repo",
		Owner: "testorg",
		Files: map[string]string{
			"README.md": "# PI Content Repository\nContains various PI for testing.",
			"src/customer.go": `package main

import "fmt"

type Customer struct {
	Name     string
	Email    string
	Phone    string
	TFN      string
	Medicare string
}

func main() {
	// Example customer data (test data only)
	customer := Customer{
		Name:     "John Smith",
		Email:    "john.smith@example.com",
		Phone:    "0412345678",
		TFN:      "123456782", // Valid TFN for testing
		Medicare: "2123456701", // Valid Medicare for testing
	}
	
	fmt.Printf("Customer: %+v\n", customer)
}`,
			"config/database.yaml": `database:
  host: localhost
  port: 5432
  username: admin
  password: secret123
  
# Customer data examples
customers:
  - name: "Jane Doe"
    email: "jane.doe@test.com"  
    abn: "33051775556"  # Valid ABN for testing
    bsb: "062-000"      # Valid BSB
    account: "12345678"
`,
			"docs/examples.md": `# API Examples

## Customer Creation

POST /customers
{
  "name": "Test Customer",
  "email": "test@example.com",
  "phone": "+61412345678",
  "address": "123 Test Street, Sydney NSW 2000"
}

## Sample Test Data

- TFN: 123456782 (valid checksum for testing)
- ABN: 33051775556 (valid checksum for testing)  
- Medicare: 2123456701 (valid checksum for testing)
- BSB: 062-000 (Commonwealth Bank)
`,
			"test/fixtures/customers.json": `[
  {
    "id": 1,
    "name": "Alice Johnson",
    "email": "alice.johnson@mockdata.com",
    "tfn": "987654328",
    "medicare": "4123456709"
  },
  {
    "id": 2, 
    "name": "Bob Wilson",
    "email": "bob.wilson@testdata.org",
    "phone": "0423456789",
    "address": "456 Mock Avenue, Melbourne VIC 3000"
  }
]`,
			".env.example": `# Example environment variables
DB_HOST=localhost
DB_USER=admin
DB_PASS=secret123
API_KEY=example_key_12345
JWT_SECRET=mock_jwt_secret_for_testing

# Sample customer data for development
DEV_CUSTOMER_EMAIL=dev@example.com
DEV_CUSTOMER_TFN=123456782
`,
		},
	}

	err := CreateMockRepository(tmpDir, mock)
	require.NoError(t, err)

	manager := NewGitHubManager(DefaultGitHubConfig())
	repoPath := filepath.Join(tmpDir, mock.Name)

	info, err := manager.GetRepositoryInfo(repoPath)
	require.NoError(t, err)

	assert.Equal(t, mock.Name, info.Name)
	assert.Greater(t, info.Size, int64(0))
	assert.Equal(t, len(mock.Files), info.FileCount)

	// Verify PI content files exist
	assert.FileExists(t, filepath.Join(repoPath, "src", "customer.go"))
	assert.FileExists(t, filepath.Join(repoPath, "config", "database.yaml"))
	assert.FileExists(t, filepath.Join(repoPath, "docs", "examples.md"))
	assert.FileExists(t, filepath.Join(repoPath, "test", "fixtures", "customers.json"))
	assert.FileExists(t, filepath.Join(repoPath, ".env.example"))

	// Read and verify sample content contains expected patterns
	customerGoContent, err := os.ReadFile(filepath.Join(repoPath, "src", "customer.go"))
	require.NoError(t, err)

	content := string(customerGoContent)
	assert.Contains(t, content, "123456782")              // TFN
	assert.Contains(t, content, "2123456701")             // Medicare
	assert.Contains(t, content, "john.smith@example.com") // Email
	assert.Contains(t, content, "0412345678")             // Phone

	// Test cleanup
	err = manager.CleanupRepository(repoPath)
	assert.NoError(t, err)
	assert.NoDirExists(t, repoPath)
}

func TestIntegrationFlow_FullRepositoryLifecycle(t *testing.T) {
	// This test simulates the full lifecycle of repository operations
	// that would be used in the PI scanner

	tmpDir := t.TempDir()
	config := DefaultGitHubConfig()
	config.UseGitHubCLI = false
	manager := NewRepositoryManager(config)

	// Create multiple mock repositories
	repositories := []MockRepository{
		{
			Name:  "frontend-app",
			Owner: "company",
			Files: map[string]string{
				"package.json":  `{"name": "frontend", "version": "1.0.0"}`,
				"src/index.js":  "console.log('Frontend app');",
				"src/config.js": "export const API_URL = 'https://api.example.com';",
				".env.example":  "API_KEY=example\nDB_URL=postgres://localhost",
			},
		},
		{
			Name:  "backend-api",
			Owner: "company",
			Files: map[string]string{
				"main.go":          "package main\n\nfunc main() { /* API server */ }",
				"config/prod.yaml": "database:\n  host: prod.db.com\n  password: secret",
				"models/user.go":   "type User struct {\n  Email string\n  TFN string\n}",
				"README.md":        "# Backend API\n\nProduction API server.",
			},
		},
		{
			Name:  "data-pipeline",
			Owner: "company",
			Files: map[string]string{
				"pipeline.py":        "# Data processing pipeline",
				"config/secrets.env": "DB_PASS=supersecret\nAPI_TOKEN=abc123",
				"data/sample.csv":    "name,email,phone\nJohn,john@test.com,0412345678",
			},
		},
	}

	// Create all mock repositories
	for _, repo := range repositories {
		err := CreateMockRepository(tmpDir, repo)
		require.NoError(t, err)
	}

	// Override git command to simulate cloning from mock repos
	githubManager := manager.github.(*gitHubManager)
	originalGitCmd := githubManager.gitCommand
	defer func() {
		githubManager.gitCommand = originalGitCmd
	}()

	githubManager.gitCommand = func(ctx context.Context, args ...string) error {
		if len(args) > 0 && args[0] == "clone" {
			// Extract repository name from URL
			url := args[len(args)-2]
			targetDir := args[len(args)-1]

			var repoName string
			for _, repo := range repositories {
				if contains(url, repo.Name) {
					repoName = repo.Name
					break
				}
			}

			if repoName == "" {
				return fmt.Errorf("unknown repository: %s", url)
			}

			// Copy mock repository to target directory
			sourceDir := filepath.Join(tmpDir, repoName)
			return copyDirectory(sourceDir, targetDir)
		}
		return nil
	}

	ctx := context.Background()
	var repoInfos []*RepositoryInfo

	// Clone all repositories
	for _, repo := range repositories {
		repoURL := fmt.Sprintf("https://github.com/%s/%s", repo.Owner, repo.Name)

		repoInfo, err := manager.CloneAndTrack(ctx, repoURL)
		require.NoError(t, err)
		repoInfos = append(repoInfos, repoInfo)

		// Verify repository was cloned correctly
		assert.Equal(t, repo.Name, repoInfo.Name)
		assert.Equal(t, repo.Owner, repoInfo.Owner)
		assert.Greater(t, repoInfo.Size, int64(0))
		assert.Greater(t, repoInfo.FileCount, 0)

		// Verify key files exist
		for filename := range repo.Files {
			fullPath := filepath.Join(repoInfo.LocalPath, filename)
			assert.FileExists(t, fullPath, "File %s should exist in %s", filename, repo.Name)
		}
	}

	// Verify all repositories are tracked
	activeRepos := manager.GetActiveRepositories()
	assert.Len(t, activeRepos, len(repositories))

	// Simulate processing delay
	time.Sleep(100 * time.Millisecond)

	// Test concurrent access to repository info
	done := make(chan bool, 3)
	for i := 0; i < 3; i++ {
		go func() {
			defer func() { done <- true }()

			active := manager.GetActiveRepositories()
			assert.Len(t, active, len(repositories))

			// Verify each repository
			for _, info := range active {
				assert.DirExists(t, info.LocalPath)
				assert.FileExists(t, filepath.Join(info.LocalPath, ".git", "config"))
			}
		}()
	}

	// Wait for concurrent operations
	for i := 0; i < 3; i++ {
		<-done
	}

	// Cleanup all repositories at once
	err := manager.CleanupAll()
	assert.NoError(t, err)

	// Verify all repositories are cleaned up
	for _, info := range repoInfos {
		assert.NoDirExists(t, info.LocalPath)
	}

	// Verify tracking is cleared
	activeRepos = manager.GetActiveRepositories()
	assert.Empty(t, activeRepos)
}

// Helper functions

func contains(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr ||
			s[len(s)-len(substr):] == substr ||
			strings.Contains(s, substr))
}

func copyDirectory(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Calculate relative path
		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		dstPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			return os.MkdirAll(dstPath, info.Mode())
		}

		// Copy file
		srcFile, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		// Create directory if needed
		err = os.MkdirAll(filepath.Dir(dstPath), 0755)
		if err != nil {
			return err
		}

		return os.WriteFile(dstPath, srcFile, info.Mode())
	})
}
