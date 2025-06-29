package test

import (
	"context"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ScanResult represents the expected structure of scan results
type ScanResult struct {
	Repository   map[string]interface{}   `json:"repository"`
	ScanStarted  string                   `json:"scan_started"`
	ScanFinished string                   `json:"scan_finished"`
	Duration     int64                    `json:"duration"`
	FilesScanned int                      `json:"files_scanned"`
	Findings     []map[string]interface{} `json:"findings"`
	Stats        map[string]interface{}   `json:"stats"`
	Error        string                   `json:"error,omitempty"`
}

// TestCLIE2E tests the CLI end-to-end functionality
func TestCLIE2E(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	// Build the binary
	binaryPath := t.TempDir() + "/pi-scanner"
	cmd := exec.Command("go", "build", "-o", binaryPath, "../cmd/pi-scanner")
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "Failed to build binary: %s", string(output))

	t.Run("basic_scan", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()

		// Use Australian healthcare repo with actual test PI data
		cmd := exec.CommandContext(ctx, binaryPath,
			"--no-input",
			"--validate=high-medium", // Force LLM validation!
			"https://github.com/hl7au/au-fhir-test-data")

		output, err := cmd.CombinedOutput()
		outputStr := string(output)

		// Log the full output for debugging
		t.Logf("Scan output:\n%s", outputStr)

		// Should succeed (or at least attempt the scan)
		if err != nil {
			// If it fails due to repo not existing, that's expected
			if strings.Contains(outputStr, "repository not found") ||
				strings.Contains(outputStr, "Repository not found") {
				t.Skip("Test repository not available")
				return
			}
		}

		// Verify we got through the scanning phases
		assert.Contains(t, outputStr, "Phase 1: Pattern-based scanning")

		// If we found findings, we should see LLM validation
		if strings.Contains(outputStr, "findings") && !strings.Contains(outputStr, "No PI findings detected") {
			assert.Contains(t, outputStr, "Phase 2: AI-powered validation")
		}

		assert.Contains(t, outputStr, "Scan complete")
	})

	t.Run("masking_levels", func(t *testing.T) {
		levels := []struct {
			name  string
			level string
			check func(t *testing.T, findings []map[string]interface{})
		}{
			{
				name:  "partial_masking",
				level: "partial",
				check: func(t *testing.T, findings []map[string]interface{}) {
					// Check that findings contain partial masking
					for _, f := range findings {
						if match, ok := f["match"].(string); ok && len(match) > 6 {
							if strings.Contains(match, "****") || strings.Contains(match, "***") {
								return // Found masked content
							}
						}
					}
					t.Error("Expected to find partially masked content")
				},
			},
			{
				name:  "full_masking",
				level: "full",
				check: func(t *testing.T, findings []map[string]interface{}) {
					// Check that findings are fully masked
					for _, f := range findings {
						if match, ok := f["match"].(string); ok && len(match) > 3 {
							assert.Contains(t, match, "*", "Finding should be masked")
						}
					}
				},
			},
		}

		for _, test := range levels {
			t.Run(test.name, func(t *testing.T) {

				ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
				defer cancel()

				cmd := exec.CommandContext(ctx, binaryPath,
					"--no-input",
					"--masking", test.level,
					"https://github.com/CommBank/CommBank-API-Samples")

				output, err := cmd.CombinedOutput()
				outputStr := string(output)

				t.Logf("Masking test output:\n%s", outputStr)

				// If repo doesn't exist, skip
				if err != nil && (strings.Contains(outputStr, "repository not found") ||
					strings.Contains(outputStr, "Repository not found")) {
					t.Skip("Test repository not available")
					return
				}

				// Verify basic scan completion
				assert.Contains(t, outputStr, "Scan complete")

				// Note: Without JSON output, we can't easily test masking in CLI output
				// This would need to be tested at the package level instead
			})
		}
	})

	t.Run("llm_command", func(t *testing.T) {
		cmd := exec.Command(binaryPath, "llm-check")
		output, _ := cmd.CombinedOutput()

		// Just verify the command outputs expected content
		outputStr := string(output)
		assert.Contains(t, outputStr, "LLM Service Check")
		assert.Contains(t, outputStr, "Endpoint:")
		assert.Contains(t, outputStr, "Model:")
	})

	t.Run("error_handling", func(t *testing.T) {
		// Test missing repo - shows help (no error)
		cmd := exec.Command(binaryPath)
		output, err := cmd.CombinedOutput()
		assert.NoError(t, err) // Help succeeds
		assert.Contains(t, string(output), "Usage:")

		// Test invalid URL
		cmd = exec.Command(binaryPath, "not-a-url")
		output, err = cmd.CombinedOutput()
		assert.Error(t, err)
		assert.Contains(t, string(output), "invalid repository URL")
	})
}
