package detection

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigLoader_LoadGitleaksConfig(t *testing.T) {
	tests := []struct {
		name         string
		setup        func() (string, func())
		customPath   string
		wantErr      bool
		wantEmbedded bool
	}{
		{
			name: "custom path exists",
			setup: func() (string, func()) {
				tmpFile, err := os.CreateTemp("", "test-gitleaks-*.toml")
				require.NoError(t, err)
				_, err = tmpFile.WriteString("[extend]\nuseDefault = true\n")
				require.NoError(t, err)
				tmpFile.Close()
				return tmpFile.Name(), func() { os.Remove(tmpFile.Name()) }
			},
			customPath:   "", // Will be set by setup
			wantErr:      false,
			wantEmbedded: false,
		},
		{
			name: "custom path not found",
			setup: func() (string, func()) {
				return "/non/existent/path.toml", func() {}
			},
			customPath: "/non/existent/path.toml",
			wantErr:    true,
		},
		{
			name: "falls back to embedded config",
			setup: func() (string, func()) {
				// Change to a directory where no config exists
				origWd, _ := os.Getwd()
				tmpDir, err := os.MkdirTemp("", "test-no-config-*")
				require.NoError(t, err)
				err = os.Chdir(tmpDir)
				require.NoError(t, err)
				return "", func() {
					err := os.Chdir(origWd)
					if err != nil {
						t.Logf("Failed to restore working directory: %v", err)
					}
					os.RemoveAll(tmpDir)
				}
			},
			customPath:   "",
			wantErr:      false,
			wantEmbedded: true,
		},
		{
			name: "finds config in current directory",
			setup: func() (string, func()) {
				// Create config in current directory
				tmpDir, err := os.MkdirTemp("", "test-with-config-*")
				require.NoError(t, err)

				// Create config directory and file
				configDir := filepath.Join(tmpDir, "config")
				err = os.Mkdir(configDir, 0755)
				require.NoError(t, err)
				configFile := filepath.Join(configDir, "gitleaks.toml")
				err = os.WriteFile(configFile, []byte("[extend]\nuseDefault = true\n"), 0644)
				require.NoError(t, err)

				// Change to temp directory
				origWd, _ := os.Getwd()
				err = os.Chdir(tmpDir)
				require.NoError(t, err)

				return "", func() {
					err := os.Chdir(origWd)
					if err != nil {
						t.Logf("Failed to restore working directory: %v", err)
					}
					os.RemoveAll(tmpDir)
				}
			},
			customPath:   "",
			wantErr:      false,
			wantEmbedded: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			customPath, cleanup := tt.setup()
			defer cleanup()

			if tt.customPath == "" && customPath != "" {
				tt.customPath = customPath
			}

			loader := NewConfigLoader()
			gotPath, err := loader.LoadGitleaksConfig(tt.customPath)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.NotEmpty(t, gotPath)

			// Check if file exists
			_, err = os.Stat(gotPath)
			assert.NoError(t, err)

			// If embedded, the path should be a temp file
			if tt.wantEmbedded {
				assert.Contains(t, gotPath, "gitleaks-embedded")
				// Clean up temp file
				defer os.Remove(gotPath)
			}
		})
	}
}

func TestConfigLoader_SearchPaths(t *testing.T) {
	loader := NewConfigLoader()

	// Should have multiple search paths
	assert.Greater(t, len(loader.searchPaths), 3)

	// Should include standard locations
	assert.Contains(t, loader.searchPaths, "config/gitleaks.toml")
	assert.Contains(t, loader.searchPaths, "gitleaks.toml")

	// Should include system paths
	hasSystemPath := false
	for _, path := range loader.searchPaths {
		cleanPath := filepath.Clean(path)
		if strings.HasPrefix(cleanPath, "/etc/") || strings.HasPrefix(cleanPath, "/usr/local/etc/") {
			hasSystemPath = true
			break
		}
	}
	assert.True(t, hasSystemPath, "Should include system config paths")
}

func TestGetEmbeddedConfig(t *testing.T) {
	config := GetEmbeddedConfig()

	// Should not be empty
	assert.NotEmpty(t, config)

	// Should be valid TOML
	assert.Contains(t, config, "[")
	assert.Contains(t, config, "]")

	// Should contain Australian PI rules
	assert.Contains(t, config, "australian-tfn")
	assert.Contains(t, config, "australian-abn")
	assert.Contains(t, config, "australian-medicare")
}
