package e2e

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ScanResult represents the expected structure of scan results
type ScanResult struct {
	Repository   RepositoryInfo `json:"repository"`
	ScanStarted  string         `json:"scan_started"`
	ScanFinished string         `json:"scan_finished"`
	Duration     int64          `json:"duration"`
	FilesScanned int            `json:"files_scanned"`
	Findings     []Finding      `json:"findings"`
	Stats        ScanStats      `json:"stats"`
	Error        string         `json:"error,omitempty"`
}

type RepositoryInfo struct {
	URL       string   `json:"url"`
	Owner     string   `json:"owner"`
	Name      string   `json:"name"`
	LocalPath string   `json:"local_path"`
	FileCount int      `json:"file_count"`
	Size      int64    `json:"size"`
	Languages []string `json:"languages"`
}

type Finding struct {
	Type         string `json:"type"`
	Match        string `json:"match"`
	File         string `json:"file"`
	Line         int    `json:"line"`
	Column       int    `json:"column"`
	RiskLevel    string `json:"risk_level"`
	Detector     string `json:"detector"`
	Context      string `json:"context,omitempty"`
	LLMValidated bool   `json:"llm_validated,omitempty"`
	LLMRisk      string `json:"llm_risk,omitempty"`
}

type ScanStats struct {
	TotalFiles     int            `json:"total_files"`
	ScannedFiles   int            `json:"scanned_files"`
	SkippedFiles   int            `json:"skipped_files"`
	TotalSize      int64          `json:"total_size"`
	FindingsByType map[string]int `json:"findings_by_type"`
	FindingsByRisk map[string]int `json:"findings_by_risk"`
}

// TestCLIBasicScan tests basic CLI scanning functionality
func TestCLIBasicScan(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping e2e test in short mode")
	}

	// Build the binary
	binaryPath := buildBinary(t)

	tests := []struct {
		name         string
		repo         string
		extraArgs    []string
		expectError  bool
		checkResults func(t *testing.T, result *ScanResult)
	}{
		{
			name: "scan_test_repo",
			repo: "https://github.com/MacAttak/pi-scanner-test-data",
			checkResults: func(t *testing.T, result *ScanResult) {
				assert.NotEmpty(t, result.Findings, "Expected findings in test repo")
				assert.Greater(t, result.FilesScanned, 0, "Should have scanned files")
				assert.NotEmpty(t, result.Stats.FindingsByType, "Should have findings by type")
			},
		},
		{
			name:      "scan_with_verbose",
			repo:      "https://github.com/MacAttak/pi-scanner-test-data",
			extraArgs: []string{"--verbose"},
			checkResults: func(t *testing.T, result *ScanResult) {
				assert.NotEmpty(t, result.Findings)
			},
		},
		{
			name:      "scan_with_full_masking",
			repo:      "https://github.com/MacAttak/pi-scanner-test-data",
			extraArgs: []string{"--masking", "full"},
			checkResults: func(t *testing.T, result *ScanResult) {
				// Check that all findings are fully masked
				for _, finding := range result.Findings {
					if len(finding.Match) > 3 {
						assert.Contains(t, finding.Match, "*", "Finding should be masked")
					}
				}
			},
		},
		{
			name:      "scan_with_no_masking",
			repo:      "https://github.com/MacAttak/pi-scanner-test-data",
			extraArgs: []string{"--masking", "none"},
			checkResults: func(t *testing.T, result *ScanResult) {
				// Check that findings are not masked
				for _, finding := range result.Findings {
					assert.NotContains(t, finding.Match, "*", "Finding should not be masked")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp directory for output
			tempDir := t.TempDir()
			outputFile := filepath.Join(tempDir, "results.json")

			// Prepare command arguments
			args := []string{"scan", "--repo", tt.repo, "--output", outputFile}
			args = append(args, tt.extraArgs...)

			// Run the command
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
			defer cancel()

			cmd := exec.CommandContext(ctx, binaryPath, args...)
			output, err := cmd.CombinedOutput()

			if tt.expectError {
				assert.Error(t, err, "Expected command to fail")
				return
			}

			require.NoError(t, err, "Command failed: %s", string(output))

			// Read and parse results
			data, err := os.ReadFile(outputFile)
			require.NoError(t, err)

			var result ScanResult
			err = json.Unmarshal(data, &result)
			require.NoError(t, err)

			// Basic validation
			assert.NotEmpty(t, result.Repository.URL)
			assert.NotZero(t, result.Duration)
			assert.NotZero(t, result.Stats.TotalFiles)

			// Test-specific checks
			if tt.checkResults != nil {
				tt.checkResults(t, &result)
			}
		})
	}
}

// TestCLILLMIntegration tests LLM integration
func TestCLILLMIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping e2e test in short mode")
	}

	binaryPath := buildBinary(t)

	// First check if LLM is available
	cmd := exec.Command(binaryPath, "llm")
	output, err := cmd.CombinedOutput()

	llmAvailable := err == nil && strings.Contains(string(output), "✅ LLM Service Available")

	t.Run("llm_status_command", func(t *testing.T) {
		// Test the llm command itself
		assert.Contains(t, string(output), "LLM Service Check")
		assert.Contains(t, string(output), "Endpoint:")
		assert.Contains(t, string(output), "Model:")
	})

	if !llmAvailable {
		t.Log("LLM service not available, skipping LLM-enabled tests")
		return
	}

	t.Run("scan_with_llm", func(t *testing.T) {
		tempDir := t.TempDir()
		outputFile := filepath.Join(tempDir, "results.json")

		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
		defer cancel()

		cmd := exec.CommandContext(ctx, binaryPath, "scan",
			"--repo", "https://github.com/MacAttak/pi-scanner-test-data",
			"--output", outputFile,
			"--enable-llm",
			"--verbose")

		output, err := cmd.CombinedOutput()
		require.NoError(t, err, "Command failed: %s", string(output))

		// Check output mentions LLM
		assert.Contains(t, string(output), "LLM validation enabled")

		// Check results
		data, err := os.ReadFile(outputFile)
		require.NoError(t, err)

		var result ScanResult
		err = json.Unmarshal(data, &result)
		require.NoError(t, err)

		// Look for LLM-validated findings
		llmValidatedCount := 0
		for _, finding := range result.Findings {
			if finding.LLMValidated {
				llmValidatedCount++
				assert.NotEmpty(t, finding.LLMRisk, "LLM-validated finding should have LLM risk")
			}
		}

		assert.Greater(t, llmValidatedCount, 0, "Should have some LLM-validated findings")
	})
}

// TestCLIAustralianRepositories tests scanning real Australian repositories
func TestCLIAustralianRepositories(t *testing.T) {
	if testing.Short() || os.Getenv("RUN_AUSTRALIAN_REPOS") != "true" {
		t.Skip("Skipping Australian repository tests (set RUN_AUSTRALIAN_REPOS=true to run)")
	}

	binaryPath := buildBinary(t)

	// Select a few representative repos
	repos := []struct {
		name        string
		url         string
		expectTypes []string // Expected PI types
		maxDuration time.Duration
	}{
		{
			name:        "health_fhir_test_data",
			url:         "https://github.com/hl7au/au-fhir-test-data",
			expectTypes: []string{"MEDICARE", "NAME"}, // Likely to have test Medicare numbers
			maxDuration: 3 * time.Minute,
		},
		{
			name:        "govau_design_system",
			url:         "https://github.com/govau/design-system-components",
			expectTypes: []string{}, // Design system, unlikely to have PI
			maxDuration: 2 * time.Minute,
		},
		{
			name:        "commbank_api_samples",
			url:         "https://github.com/CommBank/CommBank-API-Samples",
			expectTypes: []string{"BSB"}, // May have test BSB numbers
			maxDuration: 2 * time.Minute,
		},
	}

	for _, repo := range repos {
		t.Run(repo.name, func(t *testing.T) {
			tempDir := t.TempDir()
			outputFile := filepath.Join(tempDir, "results.json")

			ctx, cancel := context.WithTimeout(context.Background(), repo.maxDuration)
			defer cancel()

			cmd := exec.CommandContext(ctx, binaryPath, "scan",
				"--repo", repo.url,
				"--output", outputFile,
				"--masking", "full", // Always use full masking for real repos
				"--verbose")

			start := time.Now()
			output, err := cmd.CombinedOutput()
			duration := time.Since(start)

			require.NoError(t, err, "Command failed: %s", string(output))
			assert.Less(t, duration, repo.maxDuration, "Scan took too long")

			// Parse results
			data, err := os.ReadFile(outputFile)
			require.NoError(t, err)

			var result ScanResult
			err = json.Unmarshal(data, &result)
			require.NoError(t, err)

			// Log summary
			t.Logf("Repository: %s", repo.url)
			t.Logf("Files scanned: %d", result.FilesScanned)
			t.Logf("Total findings: %d", len(result.Findings))
			t.Logf("Findings by type: %v", result.Stats.FindingsByType)
			t.Logf("Scan duration: %v", duration)

			// Verify expected types if any
			if len(repo.expectTypes) > 0 {
				for _, expectedType := range repo.expectTypes {
					count, found := result.Stats.FindingsByType[expectedType]
					assert.True(t, found && count > 0, "Expected to find %s type", expectedType)
				}
			}

			// Ensure all findings are properly masked
			for _, finding := range result.Findings {
				if len(finding.Match) > 3 {
					assert.Contains(t, finding.Match, "*", "Finding should be masked in real repo scan")
				}
			}
		})
	}
}

// TestCLIPerformance tests performance with larger repos
func TestCLIPerformance(t *testing.T) {
	if testing.Short() || os.Getenv("RUN_PERFORMANCE_TESTS") != "true" {
		t.Skip("Skipping performance tests (set RUN_PERFORMANCE_TESTS=true to run)")
	}

	binaryPath := buildBinary(t)

	// Test with a larger repository
	tempDir := t.TempDir()
	outputFile := filepath.Join(tempDir, "results.json")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	// Use a larger, well-known repo
	cmd := exec.CommandContext(ctx, binaryPath, "scan",
		"--repo", "https://github.com/golang/go", // Large repo for performance testing
		"--output", outputFile,
		"--verbose")

	start := time.Now()
	output, err := cmd.CombinedOutput()
	duration := time.Since(start)

	require.NoError(t, err, "Command failed: %s", string(output))

	// Parse results
	data, err := os.ReadFile(outputFile)
	require.NoError(t, err)

	var result ScanResult
	err = json.Unmarshal(data, &result)
	require.NoError(t, err)

	// Performance metrics
	filesPerSecond := float64(result.FilesScanned) / duration.Seconds()
	mbPerSecond := float64(result.Stats.TotalSize) / (1024 * 1024) / duration.Seconds()

	t.Logf("Performance Metrics:")
	t.Logf("- Total duration: %v", duration)
	t.Logf("- Files scanned: %d", result.FilesScanned)
	t.Logf("- Total size: %.2f MB", float64(result.Stats.TotalSize)/(1024*1024))
	t.Logf("- Files/second: %.2f", filesPerSecond)
	t.Logf("- MB/second: %.2f", mbPerSecond)
	t.Logf("- Total findings: %d", len(result.Findings))

	// Basic performance assertions
	assert.Greater(t, filesPerSecond, 10.0, "Should process at least 10 files/second")
	assert.Greater(t, mbPerSecond, 1.0, "Should process at least 1 MB/second")
}

// Helper function to build the binary
func buildBinary(t *testing.T) string {
	t.Helper()

	// Build in a temporary directory
	tempDir := t.TempDir()
	binaryPath := filepath.Join(tempDir, "pi-scanner")

	cmd := exec.Command("go", "build", "-o", binaryPath, "../../cmd/pi-scanner")
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "Failed to build binary: %s", string(output))

	return binaryPath
}

// TestCLIErrorHandling tests various error conditions
func TestCLIErrorHandling(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping e2e test in short mode")
	}

	binaryPath := buildBinary(t)

	tests := []struct {
		name        string
		args        []string
		expectError bool
		errorCheck  func(t *testing.T, output string)
	}{
		{
			name:        "missing_repo",
			args:        []string{"scan"},
			expectError: true,
			errorCheck: func(t *testing.T, output string) {
				assert.Contains(t, output, "either --repo or --repo-list must be specified")
			},
		},
		{
			name:        "invalid_repo_url",
			args:        []string{"scan", "--repo", "not-a-url"},
			expectError: true,
			errorCheck: func(t *testing.T, output string) {
				assert.Contains(t, output, "Invalid repository URL")
			},
		},
		{
			name:        "invalid_masking_level",
			args:        []string{"scan", "--repo", "https://github.com/test/test", "--masking", "invalid"},
			expectError: false, // Should warn but continue with default
		},
		{
			name:        "llm_unavailable_port",
			args:        []string{"llm", "--endpoint", "http://localhost:9999/v1"},
			expectError: true,
			errorCheck: func(t *testing.T, output string) {
				assert.Contains(t, output, "LLM Service Unavailable")
				assert.Contains(t, output, "connection refused")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := exec.Command(binaryPath, tt.args...)
			output, err := cmd.CombinedOutput()

			if tt.expectError {
				assert.Error(t, err, "Expected command to fail")
			}

			if tt.errorCheck != nil {
				tt.errorCheck(t, string(output))
			}
		})
	}
}
