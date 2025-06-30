package detection

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGitleaksPathResolution verifies the gitleaks detector can be created from various contexts
func TestGitleaksPathResolution(t *testing.T) {
	tests := []struct {
		name          string
		setupWorkDir  func() (string, func())
		expectSuccess bool
		description   string
	}{
		{
			name: "from project root",
			setupWorkDir: func() (string, func()) {
				// Stay in current directory (should be project root in tests)
				pwd, _ := os.Getwd()
				return pwd, func() {}
			},
			expectSuccess: true,
			description:   "Should find config when run from project root",
		},
		{
			name: "from arbitrary directory",
			setupWorkDir: func() (string, func()) {
				// Create temp directory and change to it
				origWd, _ := os.Getwd()
				tmpDir, err := os.MkdirTemp("", "test-gitleaks-path-*")
				require.NoError(t, err)
				err = os.Chdir(tmpDir)
				require.NoError(t, err)
				return tmpDir, func() {
					err := os.Chdir(origWd)
					if err != nil {
						t.Logf("Failed to restore working directory: %v", err)
					}
					os.RemoveAll(tmpDir)
				}
			},
			expectSuccess: true,
			description:   "Should use embedded config when file not found",
		},
		{
			name: "from nested subdirectory",
			setupWorkDir: func() (string, func()) {
				// Create nested temp directory
				origWd, _ := os.Getwd()
				tmpDir, err := os.MkdirTemp("", "test-nested-*")
				require.NoError(t, err)
				nestedDir := filepath.Join(tmpDir, "a", "b", "c")
				err = os.MkdirAll(nestedDir, 0755)
				require.NoError(t, err)
				err = os.Chdir(nestedDir)
				require.NoError(t, err)
				return nestedDir, func() {
					err := os.Chdir(origWd)
					if err != nil {
						t.Logf("Failed to restore working directory: %v", err)
					}
					os.RemoveAll(tmpDir)
				}
			},
			expectSuccess: true,
			description:   "Should work from deeply nested directories",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pwd, cleanup := tt.setupWorkDir()
			defer cleanup()

			t.Logf("Testing from directory: %s", pwd)

			// Try to create detector with auto-detection
			detector, err := NewGitleaksDetectorAuto()

			if tt.expectSuccess {
				require.NoError(t, err, tt.description)
				assert.NotNil(t, detector)

				// Verify it can actually detect something
				ctx := context.Background()
				findings, err := detector.Detect(ctx, []byte(`var tfn = "123456782"`), "test.go")
				assert.NoError(t, err)
				assert.NotEmpty(t, findings, "Should detect valid TFN")
			} else {
				assert.Error(t, err, tt.description)
			}
		})
	}
}

// TestCustomConfigPath verifies custom config path works correctly
func TestCustomConfigPath(t *testing.T) {
	// Create a custom config file
	tmpFile, err := os.CreateTemp("", "custom-gitleaks-*.toml")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	// Write a minimal valid config
	configContent := `[extend]
useDefault = true

[[rules]]
id = "custom-test-rule"
description = "Custom test rule"
regex = '''CUSTOM_PATTERN_12345'''
`
	_, err = tmpFile.WriteString(configContent)
	require.NoError(t, err)
	tmpFile.Close()

	// Test with custom path
	detector, err := NewGitleaksDetector(tmpFile.Name())
	require.NoError(t, err)
	assert.NotNil(t, detector)

	// Verify custom rule works
	ctx := context.Background()
	findings, err := detector.Detect(ctx, []byte(`var secret = "CUSTOM_PATTERN_12345"`), "test.go")
	assert.NoError(t, err)
	assert.NotEmpty(t, findings, "Should detect custom pattern")
}

// TestConfigLoadingPriority verifies config loading follows correct priority
func TestConfigLoadingPriority(t *testing.T) {
	loader := NewConfigLoader()

	// Test 1: Custom path takes precedence
	customPath := "/tmp/custom-config.toml"
	err := os.WriteFile(customPath, []byte("[extend]\nuseDefault = true\n"), 0644)
	require.NoError(t, err)
	defer os.Remove(customPath)

	path, err := loader.LoadGitleaksConfig(customPath)
	assert.NoError(t, err)
	assert.Equal(t, customPath, path)

	// Test 2: Non-existent custom path returns error
	_, err = loader.LoadGitleaksConfig("/non/existent/path.toml")
	assert.Error(t, err)

	// Test 3: Empty path searches standard locations
	path, err = loader.LoadGitleaksConfig("")
	assert.NoError(t, err)
	assert.NotEmpty(t, path)

	// If it's a temp file (embedded), clean it up
	if filepath.Base(path) != "gitleaks.toml" {
		defer os.Remove(path)
	}
}

// TestDockerEnvironment simulates running in a Docker container
func TestDockerEnvironment(t *testing.T) {
	// Skip if not in Docker
	if _, err := os.Stat("/.dockerenv"); os.IsNotExist(err) {
		t.Skip("Not running in Docker")
	}

	// In Docker, the config might be in /etc/pi-scanner/config/
	detector, err := NewGitleaksDetectorAuto()
	assert.NoError(t, err, "Should work in Docker environment")
	assert.NotNil(t, detector)
}
