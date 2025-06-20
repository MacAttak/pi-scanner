package main

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMainCommand(t *testing.T) {
	tests := []struct {
		name           string
		args           []string
		expectedOutput []string
		expectedError  bool
		expectedCode   int
	}{
		{
			name: "no arguments shows help",
			args: []string{},
			expectedOutput: []string{
				"PI Scanner is a CLI tool",
				"Usage:",
				"Available Commands:",
				"scan",
				"report",
				"version",
			},
			expectedError: false,
			expectedCode:  0,
		},
		{
			name: "help flag shows help",
			args: []string{"--help"},
			expectedOutput: []string{
				"PI Scanner is a CLI tool",
				"scan",
				"report",
				"version",
			},
			expectedError: false,
			expectedCode:  0,
		},
		{
			name: "version command shows version",
			args: []string{"version"},
			expectedOutput: []string{
				"PI Scanner",
				"Version:",
				"Build:",
				"Go Version:",
			},
			expectedError: false,
			expectedCode:  0,
		},
		{
			name: "invalid command shows error",
			args: []string{"invalid"},
			expectedOutput: []string{
				"Error: unknown command",
			},
			expectedError: true,
			expectedCode:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture output
			var stdout, stderr bytes.Buffer

			// Set up command
			cmd := newRootCmd()
			cmd.SetOut(&stdout)
			cmd.SetErr(&stderr)
			cmd.SetArgs(tt.args)

			// Execute command
			err := cmd.Execute()

			// Check error expectation
			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			// Check output contains expected strings
			output := stdout.String() + stderr.String()
			for _, expected := range tt.expectedOutput {
				assert.Contains(t, output, expected,
					"Output should contain '%s'", expected)
			}
		})
	}
}

func TestScanCommand(t *testing.T) {
	tests := []struct {
		name           string
		args           []string
		expectedOutput []string
		expectedError  bool
	}{
		{
			name: "scan without repo shows error",
			args: []string{"scan"},
			expectedOutput: []string{
				"Error: either --repo or --repo-list must be specified",
			},
			expectedError: true,
		},
		{
			name: "scan with invalid repo URL",
			args: []string{"scan", "--repo", "invalid-url"},
			expectedOutput: []string{
				"Error: Invalid repository URL",
			},
			expectedError: true,
		},
		{
			name: "scan with valid repo URL",
			args: []string{"scan", "--repo", "https://github.com/test/repo"},
			expectedOutput: []string{
				"Scanning repository:",
				"https://github.com/test/repo",
			},
			expectedError: false,
		},
		{
			name: "scan with repo list file",
			args: []string{"scan", "--repo-list", "repos.txt"},
			expectedOutput: []string{
				"Reading repository list from:",
				"repos.txt",
			},
			expectedError: false,
		},
		{
			name: "scan with custom config",
			args: []string{"scan", "--repo", "https://github.com/test/repo", "--config", "custom.yaml"},
			expectedOutput: []string{
				"Using configuration:",
				"custom.yaml",
			},
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer

			cmd := newRootCmd()
			cmd.SetOut(&stdout)
			cmd.SetErr(&stderr)
			cmd.SetArgs(tt.args)

			err := cmd.Execute()

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				// For now, we'll skip the actual scanning
				// This will be implemented when we add the scan logic
				t.Skip("Scan implementation pending")
			}

			output := stdout.String() + stderr.String()
			for _, expected := range tt.expectedOutput {
				if tt.expectedError || !strings.Contains(expected, "Scanning") {
					assert.Contains(t, output, expected)
				}
			}
		})
	}
}

func TestReportCommand(t *testing.T) {
	// Create a temporary test results file
	testResults := `{"findings": [], "summary": {"total": 0}}`
	tmpFile, err := os.CreateTemp("", "scan-results-*.json")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.WriteString(testResults)
	require.NoError(t, err)
	tmpFile.Close()

	tests := []struct {
		name           string
		args           []string
		expectedOutput []string
		expectedError  bool
	}{
		{
			name: "report without input shows error",
			args: []string{"report"},
			expectedOutput: []string{
				"Error: required flag(s) \"input\" not set",
			},
			expectedError: true,
		},
		{
			name: "report with valid input",
			args: []string{"report", "--input", tmpFile.Name()},
			expectedOutput: []string{
				"Generating report from:",
			},
			expectedError: false,
		},
		{
			name: "report with format flag",
			args: []string{"report", "--input", tmpFile.Name(), "--format", "html"},
			expectedOutput: []string{
				"Generating HTML report",
			},
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer

			cmd := newRootCmd()
			cmd.SetOut(&stdout)
			cmd.SetErr(&stderr)
			cmd.SetArgs(tt.args)

			err := cmd.Execute()

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				// Skip actual report generation for now
				t.Skip("Report implementation pending")
			}

			output := stdout.String() + stderr.String()
			for _, expected := range tt.expectedOutput {
				if tt.expectedError {
					assert.Contains(t, output, expected)
				}
			}
		})
	}
}
