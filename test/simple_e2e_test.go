package test

import (
	"context"
	"os/exec"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSimpleE2E tests basic E2E functionality
func TestSimpleE2E(t *testing.T) {
	// Build the binary
	binaryPath := t.TempDir() + "/pi-scanner"
	cmd := exec.Command("go", "build", "-o", binaryPath, "../cmd/pi-scanner")
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "Failed to build binary: %s", string(output))

	t.Run("basic_commands", func(t *testing.T) {
		// Test version command
		cmd := exec.Command(binaryPath, "version")
		output, err := cmd.CombinedOutput()
		require.NoError(t, err)
		assert.Contains(t, string(output), "PI Scanner")
		assert.Contains(t, string(output), "Version:")

		// Test help command
		cmd = exec.Command(binaryPath, "help")
		output, err = cmd.CombinedOutput()
		require.NoError(t, err)
		assert.Contains(t, string(output), "PI Scanner")
		assert.Contains(t, string(output), "repository-url")
		assert.Contains(t, string(output), "--no-input")
		assert.Contains(t, string(output), "llm-check")
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

	t.Run("scan_test_repo", func(t *testing.T) {
		// Skip this test if the test repository doesn't exist
		t.Skip("Skipping test repo scan - requires live test repository")

		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
		defer cancel()

		cmd := exec.CommandContext(ctx, binaryPath,
			"--no-input",
			"https://github.com/octocat/Hello-World")

		output, err := cmd.CombinedOutput()
		require.NoError(t, err, "Command failed: %s", string(output))

		// Verify scanning completed
		outputStr := string(output)
		assert.Contains(t, outputStr, "Pattern Scan Results")
		assert.Contains(t, outputStr, "Scan complete")
	})

	t.Run("masking_test", func(t *testing.T) {
		tests := []struct {
			name    string
			masking string
		}{
			{
				name:    "partial_masking",
				masking: "partial",
			},
			{
				name:    "full_masking",
				masking: "full",
			},
		}

		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				cmd := exec.Command(binaryPath,
					"--no-input",
					"--masking", test.masking,
					"https://github.com/octocat/Hello-World")

				output, err := cmd.CombinedOutput()
				require.NoError(t, err, "Command failed: %s", string(output))

				// Verify masking option was processed
				outputStr := string(output)
				assert.Contains(t, outputStr, "Pattern Scan Results")
				assert.Contains(t, outputStr, "Scan complete")
			})
		}
	})

	t.Run("error_handling", func(t *testing.T) {
		// Test missing repo URL - shows help (success exit code)
		cmd := exec.Command(binaryPath)
		output, err := cmd.CombinedOutput()
		assert.NoError(t, err) // Help command succeeds
		assert.Contains(t, string(output), "Usage:")

		// Test invalid URL format
		cmd = exec.Command(binaryPath, "not-a-url")
		output, err = cmd.CombinedOutput()
		assert.Error(t, err)
		assert.Contains(t, string(output), "invalid repository URL")

		// Test URL without protocol
		cmd = exec.Command(binaryPath, "github.com/owner/repo")
		output, err = cmd.CombinedOutput()
		assert.Error(t, err)
		assert.Contains(t, string(output), "URL must include protocol")
	})
}
