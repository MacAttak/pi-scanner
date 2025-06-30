package report

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Manager handles report directory creation and management
type Manager struct {
	baseDir string
}

// NewManager creates a new report manager
func NewManager(baseDir string) *Manager {
	if baseDir == "" {
		baseDir = "reports"
	}
	return &Manager{
		baseDir: baseDir,
	}
}

// CreateReportDirectory creates a structured directory for scan reports
func (m *Manager) CreateReportDirectory(repoURL string) (string, error) {
	// Extract repo name from URL
	repoName := extractRepoName(repoURL)

	// Create timestamp
	timestamp := time.Now().Format("20060102_150405")

	// Create directory name
	dirName := fmt.Sprintf("%s_%s", timestamp, repoName)
	fullPath := filepath.Join(m.baseDir, dirName)

	// Create directory
	if err := os.MkdirAll(fullPath, 0755); err != nil {
		return "", fmt.Errorf("failed to create report directory: %w", err)
	}

	return fullPath, nil
}

// GetPhase1Path returns the path for phase 1 pattern scan results
func (m *Manager) GetPhase1Path(reportDir string) string {
	return filepath.Join(reportDir, "phase1_pattern_scan.json")
}

// GetPhase2Path returns the path for phase 2 LLM validated results
func (m *Manager) GetPhase2Path(reportDir string) string {
	return filepath.Join(reportDir, "phase2_llm_validated.json")
}

// GetSummaryPath returns the path for the summary file
func (m *Manager) GetSummaryPath(reportDir string) string {
	return filepath.Join(reportDir, "summary.txt")
}

// ListReports lists all report directories
func (m *Manager) ListReports() ([]string, error) {
	entries, err := os.ReadDir(m.baseDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, err
	}

	var reports []string
	for _, entry := range entries {
		if entry.IsDir() {
			reports = append(reports, entry.Name())
		}
	}

	// Sort by newest first
	for i := 0; i < len(reports)-1; i++ {
		for j := i + 1; j < len(reports); j++ {
			if reports[i] < reports[j] {
				reports[i], reports[j] = reports[j], reports[i]
			}
		}
	}

	return reports, nil
}

// extractRepoName extracts a clean repo name from URL
func extractRepoName(repoURL string) string {
	// Remove protocol
	url := strings.TrimPrefix(repoURL, "https://")
	url = strings.TrimPrefix(url, "http://")
	url = strings.TrimPrefix(url, "git@")

	// Remove .git suffix
	url = strings.TrimSuffix(url, ".git")

	// Handle GitHub URLs
	if strings.Contains(url, "github.com") {
		parts := strings.Split(url, "/")
		if len(parts) >= 3 {
			// Return owner_repo format
			return fmt.Sprintf("%s_%s", parts[len(parts)-2], parts[len(parts)-1])
		}
	}

	// Fallback: use last part
	parts := strings.Split(url, "/")
	if len(parts) > 0 {
		name := parts[len(parts)-1]
		// Replace special characters
		name = strings.ReplaceAll(name, ":", "_")
		name = strings.ReplaceAll(name, " ", "_")
		return name
	}

	return "unknown_repo"
}
