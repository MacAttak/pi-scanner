package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMainCommand(t *testing.T) {
	tests := []struct {
		name           string
		args           []string
		expectedOutput []string
		expectedError  bool
	}{
		{
			name: "no arguments shows help",
			args: []string{},
			expectedOutput: []string{
				"PI Scanner - Australian Privacy Compliance",
				"Usage:",
				"pi-scanner [repository-url]",
				"Available Commands:",
				"llm-check",
				"version",
			},
			expectedError: false,
		},
		{
			name: "help flag shows help",
			args: []string{"--help"},
			expectedOutput: []string{
				"PI Scanner - Australian Privacy Compliance",
				"Pattern-based scanning",
				"AI-powered validation",
				"Examples:",
			},
			expectedError: false,
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
		},
		{
			name: "invalid command shows error",
			args: []string{"invalid"},
			expectedOutput: []string{
				"invalid repository URL",
				"URL must include protocol",
			},
			expectedError: true,
		},
		{
			name: "invalid repository URL",
			args: []string{"invalid-url"},
			expectedOutput: []string{
				"invalid repository URL",
			},
			expectedError: true,
		},
		{
			name: "valid repository URL format",
			args: []string{"https://github.com/test/repo", "--no-input", "--validate=none"},
			expectedOutput: []string{
				"GitHub authentication failed",
			},
			expectedError: true, // Will fail on auth in test environment
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
				// Some commands may error due to missing resources (like GitHub token)
				// but we're testing command structure, not full execution
				_ = err
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

func TestGlobalFlags(t *testing.T) {
	tests := []struct {
		name           string
		args           []string
		expectedOutput []string
	}{
		{
			name: "no-input flag",
			args: []string{"--help"},
			expectedOutput: []string{
				"--no-input",
				"Disable all interactive prompts",
			},
		},
		{
			name: "validate flag",
			args: []string{"--help"},
			expectedOutput: []string{
				"--validate",
				"Validation mode: none, high, high-medium, all",
			},
		},
		{
			name: "masking flag",
			args: []string{"--help"},
			expectedOutput: []string{
				"--masking",
				"Masking level for PI data",
			},
		},
		{
			name: "verbose flag",
			args: []string{"--help"},
			expectedOutput: []string{
				"-v, --verbose",
				"Enable verbose output",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout bytes.Buffer

			cmd := newRootCmd()
			cmd.SetOut(&stdout)
			cmd.SetArgs(tt.args)

			_ = cmd.Execute()

			output := stdout.String()
			for _, expected := range tt.expectedOutput {
				assert.Contains(t, output, expected)
			}
		})
	}
}

func TestLLMCheckCommand(t *testing.T) {
	tests := []struct {
		name           string
		args           []string
		expectedOutput []string
	}{
		{
			name: "llm-check help",
			args: []string{"llm-check", "--help"},
			expectedOutput: []string{
				"Check if the LLM service is available",
				"--endpoint",
				"--model",
			},
		},
		{
			name: "llm-check execution",
			args: []string{"llm-check"},
			expectedOutput: []string{
				"LLM Service Check",
				"Checking service availability",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer

			cmd := newRootCmd()
			cmd.SetOut(&stdout)
			cmd.SetErr(&stderr)
			cmd.SetArgs(tt.args)

			// We expect this to fail (no LLM service running in tests)
			// but we're checking the command structure
			_ = cmd.Execute()

			output := stdout.String() + stderr.String()
			for _, expected := range tt.expectedOutput {
				assert.Contains(t, output, expected)
			}
		})
	}
}

func TestNonInteractiveMode(t *testing.T) {
	// Test that --no-input flag prevents interactive prompts
	tests := []struct {
		name         string
		args         []string
		shouldPrompt bool
	}{
		{
			name:         "with --no-input flag",
			args:         []string{"https://github.com/test/repo", "--no-input"},
			shouldPrompt: false,
		},
		{
			name:         "with --no-input and --validate",
			args:         []string{"https://github.com/test/repo", "--no-input", "--validate=high"},
			shouldPrompt: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer

			cmd := newRootCmd()
			cmd.SetOut(&stdout)
			cmd.SetErr(&stderr)
			cmd.SetArgs(tt.args)

			// Execute (will fail due to no GitHub token, but that's ok)
			_ = cmd.Execute()

			output := stdout.String()
			// Should not see welcome screen in non-interactive mode
			if strings.Contains(output, "Press Ctrl+C at any time to exit") {
				t.Error("Found interactive prompt in non-interactive mode")
			}
		})
	}
}
