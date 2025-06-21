package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// MockGitHubManager provides a mock implementation for testing
type MockGitHubManager struct {
	AuthError        error
	CloneError       error
	MockRepositories map[string]*RepositoryInfo
	CallLog          []string
}

// NewMockGitHubManager creates a new mock GitHub manager
func NewMockGitHubManager() *MockGitHubManager {
	return &MockGitHubManager{
		MockRepositories: make(map[string]*RepositoryInfo),
		CallLog:          []string{},
	}
}

// CheckAuthentication simulates authentication check
func (m *MockGitHubManager) CheckAuthentication(ctx context.Context) error {
	m.CallLog = append(m.CallLog, "CheckAuthentication")
	return m.AuthError
}

// CloneRepository simulates repository cloning
func (m *MockGitHubManager) CloneRepository(ctx context.Context, repoURL string) (*RepositoryInfo, error) {
	m.CallLog = append(m.CallLog, fmt.Sprintf("CloneRepository(%s)", repoURL))

	if m.CloneError != nil {
		return nil, m.CloneError
	}

	// Return mock repository info if configured
	if mockInfo, exists := m.MockRepositories[repoURL]; exists {
		return mockInfo, nil
	}

	// Generate default mock repository info
	owner, repo, err := m.ParseRepositoryURL(repoURL)
	if err != nil {
		return nil, err
	}

	// Create a temporary directory structure for testing
	tempDir := filepath.Join(os.TempDir(), fmt.Sprintf("mock-repo-%d", time.Now().UnixNano()))
	repoDir := filepath.Join(tempDir, repo)

	err = os.MkdirAll(repoDir, 0755)
	if err != nil {
		return nil, fmt.Errorf("failed to create mock repository directory: %w", err)
	}

	// Create some mock files
	mockFiles := []string{
		"README.md",
		"main.go",
		"config.json",
		"test/example_test.go",
		"docs/api.md",
	}

	for _, file := range mockFiles {
		filePath := filepath.Join(repoDir, file)
		fileDir := filepath.Dir(filePath)

		err = os.MkdirAll(fileDir, 0755)
		if err != nil {
			continue
		}

		content := m.generateMockFileContent(file, repoURL)
		err = os.WriteFile(filePath, []byte(content), 0644)
		if err != nil {
			continue
		}
	}

	return &RepositoryInfo{
		URL:       repoURL,
		Owner:     owner,
		Name:      repo,
		LocalPath: repoDir,
		Size:      1024 * 50, // 50KB
		FileCount: len(mockFiles),
		ClonedAt:  time.Now(),
		IsShallow: true,
	}, nil
}

// GetRepositoryInfo returns mock repository information
func (m *MockGitHubManager) GetRepositoryInfo(localPath string) (*RepositoryInfo, error) {
	m.CallLog = append(m.CallLog, fmt.Sprintf("GetRepositoryInfo(%s)", localPath))

	// Count files in the local path
	fileCount := 0
	totalSize := int64(0)

	err := filepath.WalkDir(localPath, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil // Continue walking despite errors
		}

		if !d.IsDir() && d.Name() != ".git" {
			fileCount++
			if info, err := d.Info(); err == nil {
				totalSize += info.Size()
			}
		}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to analyze repository: %w", err)
	}

	repoName := filepath.Base(localPath)

	return &RepositoryInfo{
		URL:       fmt.Sprintf("https://github.com/mock/%s", repoName),
		Owner:     "mock",
		Name:      repoName,
		LocalPath: localPath,
		Size:      totalSize,
		FileCount: fileCount,
		ClonedAt:  time.Now(),
		IsShallow: true,
	}, nil
}

// CleanupRepository removes the mock repository
func (m *MockGitHubManager) CleanupRepository(localPath string) error {
	m.CallLog = append(m.CallLog, fmt.Sprintf("CleanupRepository(%s)", localPath))
	return os.RemoveAll(localPath)
}

// ParseRepositoryURL parses repository URLs (same as real implementation)
func (m *MockGitHubManager) ParseRepositoryURL(url string) (owner, repo string, err error) {
	m.CallLog = append(m.CallLog, fmt.Sprintf("ParseRepositoryURL(%s)", url))

	// Use the same logic as the real implementation
	realManager := &gitHubManager{}
	return realManager.ParseRepositoryURL(url)
}

// SetMockRepository configures a mock repository response
func (m *MockGitHubManager) SetMockRepository(url string, info *RepositoryInfo) {
	m.MockRepositories[url] = info
}

// SetAuthError configures authentication to fail
func (m *MockGitHubManager) SetAuthError(err error) {
	m.AuthError = err
}

// SetCloneError configures cloning to fail
func (m *MockGitHubManager) SetCloneError(err error) {
	m.CloneError = err
}

// GetCallLog returns the log of method calls
func (m *MockGitHubManager) GetCallLog() []string {
	return m.CallLog
}

// generateMockFileContent creates realistic content for test files
func (m *MockGitHubManager) generateMockFileContent(filename, repoURL string) string {
	switch filepath.Ext(filename) {
	case ".md":
		return fmt.Sprintf(`# Mock Repository

This is a mock repository created for testing purposes.

Repository: %s
Generated: %s

## Features
- Mock file structure
- Realistic content
- Test data examples

## Examples
- User ID: 12345
- Email: user@example.com
- Phone: (555) 123-4567
- Test SSN: 123-45-6789 (fake)
- Mock TFN: 123456789 (example)
`, repoURL, time.Now().Format(time.RFC3339))

	case ".go":
		return fmt.Sprintf(`package main

import (
	"fmt"
	"time"
)

// MockData represents test data structure
type MockData struct {
	ID       int    `+"`json:\"id\"`"+`
	Name     string `+"`json:\"name\"`"+`
	Email    string `+"`json:\"email\"`"+`
	Created  time.Time `+"`json:\"created\"`"+`
}

func main() {
	fmt.Println("Mock application generated from %s")

	// Example test data
	testUser := MockData{
		ID:    12345,
		Name:  "Test User",
		Email: "test@example.com",
		Created: time.Now(),
	}

	fmt.Printf("User: %%+v\n", testUser)
}
`, repoURL)

	case ".json":
		mockConfig := map[string]interface{}{
			"app_name":       "mock-app",
			"version":        "1.0.0",
			"environment":    "test",
			"database_url":   "mock://localhost:5432/testdb",
			"api_key":        "mock-api-key-123456789",
			"debug":          true,
			"generated_from": repoURL,
			"test_data": map[string]interface{}{
				"sample_id":    "12345",
				"sample_email": "mock@example.com",
				"sample_phone": "555-0123",
			},
		}

		jsonData, _ := json.MarshalIndent(mockConfig, "", "  ")
		return string(jsonData)

	default:
		return fmt.Sprintf(`Mock file content for %s
Generated from: %s
Timestamp: %s

This file contains mock data for testing purposes.

Example data:
- ID: mock-123456
- Email: example@test.com
- Phone: 555-MOCK (555-6625)
- Test SSN: 000-00-0000 (invalid)
`, filename, repoURL, time.Now().Format(time.RFC3339))
	}
}
