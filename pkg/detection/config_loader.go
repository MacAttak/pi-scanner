package detection

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
)

// Embed the default gitleaks configuration
//
//go:embed gitleaks_default.toml
var embeddedGitleaksConfig string

// ConfigLoader handles loading configuration from various sources
type ConfigLoader struct {
	searchPaths []string
}

// NewConfigLoader creates a new config loader with default search paths
func NewConfigLoader() *ConfigLoader {
	executable, _ := os.Executable()
	execDir := filepath.Dir(executable)

	return &ConfigLoader{
		searchPaths: []string{
			// Current working directory
			"config/gitleaks.toml",
			"gitleaks.toml",
			// Executable directory
			filepath.Join(execDir, "config", "gitleaks.toml"),
			filepath.Join(execDir, "gitleaks.toml"),
			// System config directories
			"/etc/pi-scanner/gitleaks.toml",
			"/etc/pi-scanner/config/gitleaks.toml",
			"/usr/local/etc/pi-scanner/gitleaks.toml",
		},
	}
}

// LoadGitleaksConfig attempts to load gitleaks configuration from various sources
func (cl *ConfigLoader) LoadGitleaksConfig(customPath string) (string, error) {
	// If custom path provided, try it first
	if customPath != "" {
		if _, err := os.Stat(customPath); err == nil {
			return customPath, nil
		}
		return "", fmt.Errorf("custom config path not found: %s", customPath)
	}

	// Try all search paths
	for _, path := range cl.searchPaths {
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}

	// Fall back to embedded config
	tmpFile, err := os.CreateTemp("", "gitleaks-embedded-*.toml")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file for embedded config: %w", err)
	}

	if _, err := tmpFile.WriteString(embeddedGitleaksConfig); err != nil {
		tmpFile.Close()
		os.Remove(tmpFile.Name())
		return "", fmt.Errorf("failed to write embedded config: %w", err)
	}

	tmpFile.Close()
	return tmpFile.Name(), nil
}

// GetEmbeddedConfig returns the embedded gitleaks configuration
func GetEmbeddedConfig() string {
	return embeddedGitleaksConfig
}
