package config

import (
	_ "embed"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

//go:embed default_config.yaml
var defaultConfigYAML string

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	var config Config
	if err := yaml.Unmarshal([]byte(defaultConfigYAML), &config); err != nil {
		// If embedded config fails, return hardcoded defaults
		return hardcodedDefaults()
	}
	config.applyDefaults()
	return &config
}

// hardcodedDefaults returns hardcoded default configuration as fallback
func hardcodedDefaults() *Config {
	return &Config{
		Version: "1.0",
		Scanner: ScannerConfig{
			Workers:           4,
			FileTypes:         DefaultFileTypes(),
			ExcludePaths:      defaultExcludePaths(),
			MaxFileSize:       10 * 1024 * 1024, // 10MB
			ProximityDistance: 10,
			MLValidation: MLConfig{
				Enabled:             false,
				ConfidenceThreshold: 0.7,
				BatchSize:           32,
			},
			Validators: ValidatorConfig{
				TFN: ValidatorSettings{
					Enabled:       true,
					StrictMode:    true,
					MinConfidence: 0.8,
				},
				Medicare: ValidatorSettings{
					Enabled:       true,
					StrictMode:    true,
					MinConfidence: 0.8,
				},
				ABN: ValidatorSettings{
					Enabled:       true,
					StrictMode:    false,
					MinConfidence: 0.7,
				},
				BSB: ValidatorSettings{
					Enabled:       true,
					StrictMode:    false,
					MinConfidence: 0.7,
				},
				CreditCard: ValidatorSettings{
					Enabled:       true,
					StrictMode:    true,
					MinConfidence: 0.9,
				},
				Email: ValidatorSettings{
					Enabled:       true,
					StrictMode:    false,
					MinConfidence: 0.6,
				},
				Phone: ValidatorSettings{
					Enabled:       true,
					StrictMode:    false,
					MinConfidence: 0.6,
				},
			},
		},
		Risk: RiskConfig{
			Thresholds: RiskThresholds{
				Critical: 0.8,
				High:     0.6,
				Medium:   0.4,
				Low:      0.2,
			},
			Multipliers: RiskMultipliers{
				Production:  1.5,
				Staging:     1.2,
				Development: 0.8,
				Test:        0.5,
			},
			CoOccurrence: CoOccurrenceConfig{
				Enabled:         true,
				ProximityWindow: 50,
				MinOccurrences:  2,
				ScoreBoost:      0.2,
			},
		},
		Report: ReportConfig{
			Formats:         []string{"html", "csv"},
			OutputDirectory: "reports",
			IncludeMasked:   true,
			IncludeContext:  true,
			SARIF: SARIFConfig{
				ToolName:    "PI Scanner",
				ToolVersion: "1.0.0",
				InfoURI:     "https://github.com/MacAttak/pi-scanner",
			},
		},
		Github: GithubConfig{
			RateLimit:   30,
			CloneDepth:  1,
			TempDirectory: "/tmp/pi-scanner",
		},
		Logging: LoggingConfig{
			Level:      "info",
			Format:     "json",
			MaxSize:    100,
			MaxBackups: 3,
			MaxAge:     28,
		},
	}
}

// defaultExcludePaths returns default paths to exclude from scanning
func defaultExcludePaths() []string {
	return []string{
		".git",
		".svn",
		".hg",
		"node_modules",
		"vendor",
		".venv",
		"venv",
		"__pycache__",
		".pytest_cache",
		"dist",
		"build",
		"target",
		"bin",
		"obj",
		".idea",
		".vscode",
		"*.min.js",
		"*.min.css",
		"*.map",
		"*.sum",
		"*.lock",
	}
}

// ExampleConfig generates an example configuration file
func ExampleConfig() (*Config, error) {
	config := DefaultConfig()
	
	// Add some example customizations
	config.Scanner.Workers = 8
	config.Scanner.MLValidation.Enabled = true
	config.Scanner.MLValidation.ModelPath = "models/deberta"
	config.Scanner.MLValidation.TokenizerPath = "models/tokenizer"
	
	config.Report.Formats = []string{"html", "csv", "sarif"}
	
	config.Github.Token = "${GITHUB_TOKEN}" // Placeholder
	
	return config, nil
}

// GenerateExampleConfig writes an example configuration file
func GenerateExampleConfig(path string) error {
	config, err := ExampleConfig()
	if err != nil {
		return fmt.Errorf("failed to generate example config: %w", err)
	}
	
	// Add header comment
	header := `# PI Scanner Configuration
# This is an example configuration file for the PI Scanner.
# Modify values as needed for your environment.
# Environment variables can be used with ${VAR_NAME} syntax.

`
	
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}
	
	fullContent := []byte(header + string(data))
	return os.WriteFile(path, fullContent, 0644)
}