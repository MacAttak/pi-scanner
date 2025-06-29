package e2e

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGuidedScanE2E tests the new guided scanning workflow end-to-end
func TestGuidedScanE2E(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping e2e test in short mode")
	}

	// Check for GitHub token
	if os.Getenv("GITHUB_TOKEN") == "" {
		t.Skip("GITHUB_TOKEN not set")
	}

	// Build the binary
	binPath := filepath.Join(t.TempDir(), "pi-scanner")
	buildCmd := exec.Command("go", "build", "-o", binPath, "../../cmd/pi-scanner")
	buildOutput, err := buildCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to build binary: %v\nOutput: %s", err, buildOutput)
	}

	tests := []struct {
		name           string
		args           []string
		expectSuccess  bool
		expectedOutput []string
		skipOutput     []string // Strings that should NOT appear
	}{
		{
			name: "pattern scan only",
			args: []string{
				"https://github.com/MacAttak/pi-scanner",
				"--no-input",
				"--validate=none",
			},
			expectSuccess: true,
			expectedOutput: []string{
				"Phase 1: Pattern-based scanning",
				"Repository cloned",
				"Pattern Scan Results",
				"Findings by Risk Level:",
				"PI Types Detected:",
				"Pattern scan report saved to:",
			},
			skipOutput: []string{
				"Phase 2: AI-powered validation",
				"LLM Service Check",
			},
		},
		{
			name: "invalid repository",
			args: []string{
				"https://github.com/invalid/repo-that-does-not-exist",
				"--no-input",
			},
			expectSuccess: false,
			expectedOutput: []string{
				"Phase 1: Pattern-based scanning",
				"pattern scan failed",
			},
		},
		{
			name: "help command",
			args: []string{
				"--help",
			},
			expectSuccess: true,
			expectedOutput: []string{
				"PI Scanner - Australian Privacy Compliance",
				"Detects personally identifiable information",
				"Pattern-based scanning",
				"AI-powered validation",
				"Examples:",
			},
		},
		{
			name: "version command",
			args: []string{
				"version",
			},
			expectSuccess: true,
			expectedOutput: []string{
				"PI Scanner",
				"Version:",
				"Build:",
				"Go Version:",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
			defer cancel()

			cmd := exec.CommandContext(ctx, binPath, tt.args...)
			cmd.Env = append(os.Environ(),
				"GITHUB_TOKEN="+os.Getenv("GITHUB_TOKEN"),
			)

			var stdout, stderr bytes.Buffer
			cmd.Stdout = &stdout
			cmd.Stderr = &stderr

			err := cmd.Run()

			output := stdout.String() + stderr.String()
			t.Logf("Command output:\n%s", output)

			if tt.expectSuccess {
				assert.NoError(t, err, "Command should succeed")
			} else {
				assert.Error(t, err, "Command should fail")
			}

			// Check expected output
			for _, expected := range tt.expectedOutput {
				assert.Contains(t, output, expected,
					"Output should contain '%s'", expected)
			}

			// Check strings that should NOT appear
			for _, skip := range tt.skipOutput {
				assert.NotContains(t, output, skip,
					"Output should NOT contain '%s'", skip)
			}
		})
	}
}

// TestNonInteractiveValidation tests the full workflow with LLM validation
func TestNonInteractiveValidation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping e2e test in short mode")
	}

	// Check for GitHub token
	if os.Getenv("GITHUB_TOKEN") == "" {
		t.Skip("GITHUB_TOKEN not set")
	}

	// Check if LLM is available by running llm-check
	binPath := filepath.Join(t.TempDir(), "pi-scanner")
	buildCmd := exec.Command("go", "build", "-o", binPath, "../../cmd/pi-scanner")
	_, err := buildCmd.CombinedOutput()
	require.NoError(t, err, "Failed to build binary")

	// Test LLM availability
	llmCmd := exec.Command(binPath, "llm-check")
	llmOutput, _ := llmCmd.CombinedOutput()

	if !strings.Contains(string(llmOutput), "LLM Service Available") {
		t.Skip("LLM service not available - skipping validation test")
	}

	// Run full scan with validation
	cmd := exec.Command(binPath,
		"https://github.com/MacAttak/pi-scanner",
		"--no-input",
		"--validate=high",
		"--masking=partial",
	)
	cmd.Env = append(os.Environ(),
		"GITHUB_TOKEN="+os.Getenv("GITHUB_TOKEN"),
	)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	err = cmd.Start()
	require.NoError(t, err)

	done := make(chan error)
	go func() {
		done <- cmd.Wait()
	}()

	select {
	case err := <-done:
		output := stdout.String() + stderr.String()
		t.Logf("Full output:\n%s", output)

		assert.NoError(t, err, "Command should succeed")

		// Check for both phases
		assert.Contains(t, output, "Phase 1: Pattern-based scanning")
		assert.Contains(t, output, "Phase 2: AI-powered validation")
		assert.Contains(t, output, "Validation Complete")
		assert.Contains(t, output, "Risk Assessment Changes:")

	case <-ctx.Done():
		cmd.Process.Kill()
		t.Fatal("Command timed out")
	}
}

// TestReportDirectory tests that reports are saved to the correct location
func TestReportDirectory(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping e2e test in short mode")
	}

	if os.Getenv("GITHUB_TOKEN") == "" {
		t.Skip("GITHUB_TOKEN not set")
	}

	// Create a temporary directory for the test
	tempDir := t.TempDir()

	// Build binary
	binPath := filepath.Join(tempDir, "pi-scanner")
	buildCmd := exec.Command("go", "build", "-o", binPath, "../../cmd/pi-scanner")
	_, err := buildCmd.CombinedOutput()
	require.NoError(t, err)

	// Change to temp directory to test report creation
	oldWd, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(oldWd)

	err = os.Chdir(tempDir)
	require.NoError(t, err)

	// Run scan
	cmd := exec.Command(binPath,
		"https://github.com/MacAttak/pi-scanner",
		"--no-input",
		"--validate=none",
	)
	cmd.Env = append(os.Environ(),
		"GITHUB_TOKEN="+os.Getenv("GITHUB_TOKEN"),
	)

	output, err := cmd.CombinedOutput()
	t.Logf("Output: %s", output)

	// Check that reports directory was created
	reportsDir := filepath.Join(tempDir, "reports")
	info, err := os.Stat(reportsDir)
	assert.NoError(t, err, "reports directory should exist")
	assert.True(t, info.IsDir(), "reports should be a directory")

	// Check for report subdirectory (should contain timestamp and repo name)
	entries, err := os.ReadDir(reportsDir)
	assert.NoError(t, err)
	assert.NotEmpty(t, entries, "Should have at least one report directory")

	// Verify report structure
	for _, entry := range entries {
		if entry.IsDir() && strings.Contains(entry.Name(), "pi-scanner") {
			reportPath := filepath.Join(reportsDir, entry.Name())

			// Check for phase1 report
			phase1File := filepath.Join(reportPath, "phase1_pattern_scan.json")
			_, err := os.Stat(phase1File)
			assert.NoError(t, err, "phase1_pattern_scan.json should exist")

			// Check for summary
			summaryFile := filepath.Join(reportPath, "summary.txt")
			_, err = os.Stat(summaryFile)
			assert.NoError(t, err, "summary.txt should exist")

			break
		}
	}
}
