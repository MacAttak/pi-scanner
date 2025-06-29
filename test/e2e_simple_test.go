//go:build !ci
// +build !ci

package test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPIScannerE2E_Simple(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E tests in short mode")
	}

	// Build the scanner binary
	binaryPath := buildSimpleBinary(t)
	defer os.Remove(binaryPath)

	t.Run("CLI Commands Work", func(t *testing.T) {
		testCLICommands(t, binaryPath)
	})

	t.Run("Authentication Works in CI", func(t *testing.T) {
		testCIAuthentication(t, binaryPath)
	})

	t.Run("Error Handling Works", func(t *testing.T) {
		testSimpleErrorHandling(t, binaryPath)
	})
}

func buildSimpleBinary(t *testing.T) string {
	t.Helper()

	// Find project root by looking for go.mod
	dir, err := os.Getwd()
	require.NoError(t, err)

	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			break
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("Could not find go.mod file")
		}
		dir = parent
	}

	// Create temporary binary
	tempDir := t.TempDir()
	binaryPath := filepath.Join(tempDir, "pi-scanner")

	// Build binary
	cmd := exec.Command("go", "build", "-o", binaryPath, "./cmd/pi-scanner")
	cmd.Dir = dir

	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "Failed to build pi-scanner: %s", string(output))

	return binaryPath
}

func testCLICommands(t *testing.T, binaryPath string) {
	tests := []struct {
		name     string
		args     []string
		patterns []string
	}{
		{
			name:     "Version Command",
			args:     []string{"version"},
			patterns: []string{"PI Scanner", "Version"},
		},
		{
			name:     "Help Command",
			args:     []string{"--help"},
			patterns: []string{"Usage:", "Available Commands:"},
		},
		{
			name:     "Scan Help",
			args:     []string{"--help"},
			patterns: []string{"Usage:", "repository-url", "--no-input"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			cmd := exec.CommandContext(ctx, binaryPath, test.args...)
			output, err := cmd.CombinedOutput()

			assert.NoError(t, err, "Command should execute successfully")
			assert.NotEmpty(t, output, "Should produce output")

			outputStr := string(output)
			for _, pattern := range test.patterns {
				assert.Contains(t, outputStr, pattern, "Output should contain: %s", pattern)
			}
		})
	}
}

func testCIAuthentication(t *testing.T, binaryPath string) {
	// Test that CI environment allows operation without explicit auth
	originalCI := os.Getenv("CI")
	os.Setenv("CI", "true")
	defer func() {
		if originalCI == "" {
			os.Unsetenv("CI")
		} else {
			os.Setenv("CI", originalCI)
		}
	}()

	// Try a quick scan that should not fail due to authentication
	args := []string{"--no-input", "https://github.com/octocat/Hello-World"}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, binaryPath, args...)
	cmd.Env = append(os.Environ(), "CI=true")
	output, err := cmd.CombinedOutput()

	outputStr := string(output)
	t.Logf("Scan output: %s", outputStr)

	// Should not fail due to authentication in CI environment
	if err != nil {
		assert.NotContains(t, outputStr, "no authentication method configured",
			"Should not require authentication in CI environment")
		assert.NotContains(t, outputStr, "run 'gh auth login'",
			"Should not request GitHub CLI login in CI environment")
	}

	// If scan succeeded, verify output was produced
	if err == nil {
		assert.Contains(t, outputStr, "Pattern Scan Results", "Should show scan results")
		assert.Contains(t, outputStr, "Scan complete", "Should complete successfully")
		t.Logf("Scan completed successfully")
	}
}

func testSimpleErrorHandling(t *testing.T, binaryPath string) {
	errorTests := []struct {
		name          string
		repositoryURL string
		expectError   bool
		errorPattern  string
	}{
		{
			name:          "Invalid Repository URL",
			repositoryURL: "invalid-url",
			expectError:   true,
			errorPattern:  "invalid repository URL",
		},
		{
			name:          "Malformed GitHub URL",
			repositoryURL: "https://github.com/",
			expectError:   true,
			errorPattern:  "invalid repository URL",
		},
	}

	for _, test := range errorTests {
		t.Run(test.name, func(t *testing.T) {
			args := []string{"--no-input", test.repositoryURL}

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			cmd := exec.CommandContext(ctx, binaryPath, args...)
			output, err := cmd.CombinedOutput()

			if test.expectError {
				assert.Error(t, err, "Should fail for %s", test.name)
				if test.errorPattern != "" {
					assert.Contains(t, string(output), test.errorPattern,
						"Error should contain pattern: %s", test.errorPattern)
				}
			} else {
				assert.NoError(t, err, "Should succeed for %s", test.name)
			}
		})
	}
}

// BenchmarkE2E_SimpleRepository provides a performance baseline
func BenchmarkE2E_SimpleRepository(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping benchmark in short mode")
	}

	// Only run if we have authentication
	if os.Getenv("GITHUB_TOKEN") == "" && os.Getenv("CI") == "" {
		b.Skip("Skipping benchmark: no authentication available")
	}

	binaryPath := buildSimpleBinary(&testing.T{})
	defer os.Remove(binaryPath)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		args := []string{"--no-input", "https://github.com/octocat/Hello-World"}

		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		cmd := exec.CommandContext(ctx, binaryPath, args...)

		if token := os.Getenv("GITHUB_TOKEN"); token != "" {
			cmd.Env = append(os.Environ(), "GITHUB_TOKEN="+token)
		}

		_, err := cmd.CombinedOutput()
		cancel()

		if err != nil {
			b.Logf("Benchmark iteration %d failed: %v", i, err)
		}
	}
}
