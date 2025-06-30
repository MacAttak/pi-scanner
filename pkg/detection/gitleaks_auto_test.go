package detection

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewGitleaksDetectorAuto(t *testing.T) {
	// Test auto-detection from various working directories
	tests := []struct {
		name    string
		setup   func() func()
		wantErr bool
	}{
		{
			name: "from project root",
			setup: func() func() {
				// Assume we're already in project root
				return func() {}
			},
			wantErr: false,
		},
		{
			name: "from subdirectory",
			setup: func() func() {
				origWd, _ := os.Getwd()
				tmpDir, err := os.MkdirTemp("", "test-subdir-*")
				require.NoError(t, err)
				err = os.Chdir(tmpDir)
				require.NoError(t, err)
				return func() {
					err := os.Chdir(origWd)
					if err != nil {
						t.Logf("Failed to restore working directory: %v", err)
					}
					os.RemoveAll(tmpDir)
				}
			},
			wantErr: false, // Should use embedded config
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cleanup := tt.setup()
			defer cleanup()

			detector, err := NewGitleaksDetectorAuto()

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.NotNil(t, detector)
			assert.Equal(t, "gitleaks-detector", detector.Name())

			// Test that it can detect something
			ctx := context.Background()
			findings, err := detector.Detect(ctx, []byte(`var tfn = "123456782"`), "test.go")
			assert.NoError(t, err)
			assert.NotEmpty(t, findings)
		})
	}
}

func TestNewGitleaksDetector_PathResolution(t *testing.T) {
	tests := []struct {
		name       string
		configPath string
		setup      func() (string, func())
		wantErr    bool
	}{
		{
			name:       "empty path uses auto-detection",
			configPath: "",
			setup: func() (string, func()) {
				return "", func() {}
			},
			wantErr: false,
		},
		{
			name:       "absolute path that exists",
			configPath: "", // Will be set by setup
			setup: func() (string, func()) {
				tmpFile, err := os.CreateTemp("", "test-config-*.toml")
				require.NoError(t, err)
				_, err = tmpFile.WriteString(`[extend]
useDefault = true

[[rules]]
id = "test-rule"
description = "Test rule"
regex = '''test-pattern-12345'''
`)
				require.NoError(t, err)
				tmpFile.Close()
				return tmpFile.Name(), func() { os.Remove(tmpFile.Name()) }
			},
			wantErr: false,
		},
		{
			name:       "non-existent path",
			configPath: "/this/does/not/exist.toml",
			setup: func() (string, func()) {
				return "", func() {}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configPath, cleanup := tt.setup()
			defer cleanup()

			if configPath != "" && tt.configPath == "" {
				tt.configPath = configPath
			}

			detector, err := NewGitleaksDetector(tt.configPath)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.NotNil(t, detector)
		})
	}
}
