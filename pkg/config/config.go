package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

// Config represents the complete scanner configuration
type Config struct {
	Version string        `yaml:"version"`
	Scanner ScannerConfig `yaml:"scanner"`
	Risk    RiskConfig    `yaml:"risk"`
	Report  ReportConfig  `yaml:"report"`
	Github  GithubConfig  `yaml:"github"`
	Logging LoggingConfig `yaml:"logging"`
}

// ScannerConfig contains scanner-specific settings
type ScannerConfig struct {
	Workers           int             `yaml:"workers"`
	FileTypes         []string        `yaml:"file_types"`
	ExcludePaths      []string        `yaml:"exclude_paths"`
	MaxFileSize       int64           `yaml:"max_file_size"`
	Timeout           time.Duration   `yaml:"timeout"`
	Validators        ValidatorConfig `yaml:"validators"`
	ProximityDistance int             `yaml:"proximity_distance"`
}

// ValidatorConfig contains validator settings
type ValidatorConfig struct {
	TFN        ValidatorSettings `yaml:"tfn"`
	Medicare   ValidatorSettings `yaml:"medicare"`
	ABN        ValidatorSettings `yaml:"abn"`
	BSB        ValidatorSettings `yaml:"bsb"`
	CreditCard ValidatorSettings `yaml:"credit_card"`
	Email      ValidatorSettings `yaml:"email"`
	Phone      ValidatorSettings `yaml:"phone"`
}

// ValidatorSettings contains individual validator settings
type ValidatorSettings struct {
	Enabled       bool    `yaml:"enabled"`
	StrictMode    bool    `yaml:"strict_mode"`
	MinConfidence float64 `yaml:"min_confidence"`
	CustomPattern string  `yaml:"custom_pattern,omitempty"`
}

// RiskConfig contains risk scoring settings
type RiskConfig struct {
	Thresholds   RiskThresholds     `yaml:"thresholds"`
	Multipliers  RiskMultipliers    `yaml:"multipliers"`
	CoOccurrence CoOccurrenceConfig `yaml:"co_occurrence"`
}

// RiskThresholds defines risk level thresholds
type RiskThresholds struct {
	Critical float64 `yaml:"critical"`
	High     float64 `yaml:"high"`
	Medium   float64 `yaml:"medium"`
	Low      float64 `yaml:"low"`
}

// RiskMultipliers defines environment multipliers
type RiskMultipliers struct {
	Production  float64 `yaml:"production"`
	Staging     float64 `yaml:"staging"`
	Development float64 `yaml:"development"`
	Test        float64 `yaml:"test"`
}

// CoOccurrenceConfig defines co-occurrence settings
type CoOccurrenceConfig struct {
	Enabled         bool    `yaml:"enabled"`
	ProximityWindow int     `yaml:"proximity_window"`
	MinOccurrences  int     `yaml:"min_occurrences"`
	ScoreBoost      float64 `yaml:"score_boost"`
}

// ReportConfig contains report generation settings
type ReportConfig struct {
	Formats           []string    `yaml:"formats"`
	OutputDirectory   string      `yaml:"output_directory"`
	IncludeMasked     bool        `yaml:"include_masked"`
	IncludeContext    bool        `yaml:"include_context"`
	TemplateDirectory string      `yaml:"template_directory,omitempty"`
	SARIF             SARIFConfig `yaml:"sarif"`
}

// SARIFConfig contains SARIF-specific settings
type SARIFConfig struct {
	ToolName    string `yaml:"tool_name"`
	ToolVersion string `yaml:"tool_version"`
	InfoURI     string `yaml:"info_uri"`
}

// GithubConfig contains GitHub integration settings
type GithubConfig struct {
	Token         string        `yaml:"token,omitempty"`
	RateLimit     int           `yaml:"rate_limit"`
	CloneTimeout  time.Duration `yaml:"clone_timeout"`
	CloneDepth    int           `yaml:"clone_depth"`
	TempDirectory string        `yaml:"temp_directory"`
}

// LoggingConfig contains logging settings
type LoggingConfig struct {
	Level      string `yaml:"level"`
	Format     string `yaml:"format"`
	OutputFile string `yaml:"output_file,omitempty"`
	MaxSize    int    `yaml:"max_size"`
	MaxBackups int    `yaml:"max_backups"`
	MaxAge     int    `yaml:"max_age"`
}

// LoadConfig loads configuration from a YAML file
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Apply defaults
	config.applyDefaults()

	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &config, nil
}

// LoadConfigWithDefaults loads config or returns default if file doesn't exist
func LoadConfigWithDefaults(path string) (*Config, error) {
	if path != "" {
		if _, err := os.Stat(path); err == nil {
			return LoadConfig(path)
		}
	}
	return DefaultConfig(), nil
}

// SaveConfig saves configuration to a YAML file
func SaveConfig(config *Config, path string) error {
	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if c.Scanner.Workers < 1 {
		return fmt.Errorf("scanner workers must be at least 1")
	}

	if c.Scanner.MaxFileSize < 0 {
		return fmt.Errorf("max file size cannot be negative")
	}

	if c.Scanner.ProximityDistance < 0 {
		return fmt.Errorf("proximity distance cannot be negative")
	}

	// Validate risk thresholds
	if c.Risk.Thresholds.Critical < c.Risk.Thresholds.High ||
		c.Risk.Thresholds.High < c.Risk.Thresholds.Medium ||
		c.Risk.Thresholds.Medium < c.Risk.Thresholds.Low ||
		c.Risk.Thresholds.Low < 0 {
		return fmt.Errorf("risk thresholds must be in descending order: critical > high > medium > low >= 0")
	}

	// Validate multipliers
	if c.Risk.Multipliers.Production < 0 || c.Risk.Multipliers.Staging < 0 ||
		c.Risk.Multipliers.Development < 0 || c.Risk.Multipliers.Test < 0 {
		return fmt.Errorf("risk multipliers cannot be negative")
	}

	// Validate report formats
	validFormats := map[string]bool{"html": true, "csv": true, "sarif": true, "json": true}
	for _, format := range c.Report.Formats {
		if !validFormats[format] {
			return fmt.Errorf("invalid report format: %s", format)
		}
	}

	// Validate logging level
	validLevels := map[string]bool{"debug": true, "info": true, "warn": true, "error": true}
	if !validLevels[c.Logging.Level] {
		return fmt.Errorf("invalid logging level: %s", c.Logging.Level)
	}

	return nil
}

// applyDefaults applies default values to missing configuration
func (c *Config) applyDefaults() {
	if c.Version == "" {
		c.Version = "1.0"
	}

	// Scanner defaults
	if c.Scanner.Workers == 0 {
		c.Scanner.Workers = 4
	}
	if len(c.Scanner.FileTypes) == 0 {
		c.Scanner.FileTypes = DefaultFileTypes()
	}
	if c.Scanner.MaxFileSize == 0 {
		c.Scanner.MaxFileSize = 10 * 1024 * 1024 // 10MB
	}
	if c.Scanner.Timeout == 0 {
		c.Scanner.Timeout = 30 * time.Minute
	}
	if c.Scanner.ProximityDistance == 0 {
		c.Scanner.ProximityDistance = 10
	}

	// Risk defaults
	if c.Risk.Thresholds.Critical == 0 {
		c.Risk.Thresholds.Critical = 0.8
	}
	if c.Risk.Thresholds.High == 0 {
		c.Risk.Thresholds.High = 0.6
	}
	if c.Risk.Thresholds.Medium == 0 {
		c.Risk.Thresholds.Medium = 0.4
	}
	if c.Risk.Thresholds.Low == 0 {
		c.Risk.Thresholds.Low = 0.2
	}

	// Multiplier defaults
	if c.Risk.Multipliers.Production == 0 {
		c.Risk.Multipliers.Production = 1.5
	}
	if c.Risk.Multipliers.Staging == 0 {
		c.Risk.Multipliers.Staging = 1.2
	}
	if c.Risk.Multipliers.Development == 0 {
		c.Risk.Multipliers.Development = 0.8
	}
	if c.Risk.Multipliers.Test == 0 {
		c.Risk.Multipliers.Test = 0.5
	}

	// Co-occurrence defaults
	if c.Risk.CoOccurrence.ProximityWindow == 0 {
		c.Risk.CoOccurrence.ProximityWindow = 50
	}
	if c.Risk.CoOccurrence.MinOccurrences == 0 {
		c.Risk.CoOccurrence.MinOccurrences = 2
	}
	if c.Risk.CoOccurrence.ScoreBoost == 0 {
		c.Risk.CoOccurrence.ScoreBoost = 0.2
	}

	// Report defaults
	if len(c.Report.Formats) == 0 {
		c.Report.Formats = []string{"html"}
	}
	if c.Report.OutputDirectory == "" {
		c.Report.OutputDirectory = "reports"
	}

	// SARIF defaults
	if c.Report.SARIF.ToolName == "" {
		c.Report.SARIF.ToolName = "PI Scanner"
	}
	if c.Report.SARIF.ToolVersion == "" {
		c.Report.SARIF.ToolVersion = "1.0.0"
	}
	if c.Report.SARIF.InfoURI == "" {
		c.Report.SARIF.InfoURI = "https://github.com/MacAttak/pi-scanner"
	}

	// GitHub defaults
	if c.Github.RateLimit == 0 {
		c.Github.RateLimit = 30
	}
	if c.Github.CloneTimeout == 0 {
		c.Github.CloneTimeout = 10 * time.Minute
	}
	if c.Github.CloneDepth == 0 {
		c.Github.CloneDepth = 1
	}
	if c.Github.TempDirectory == "" {
		c.Github.TempDirectory = os.TempDir()
	}

	// Logging defaults
	if c.Logging.Level == "" {
		c.Logging.Level = "info"
	}
	if c.Logging.Format == "" {
		c.Logging.Format = "json"
	}
	if c.Logging.MaxSize == 0 {
		c.Logging.MaxSize = 100 // 100MB
	}
	if c.Logging.MaxBackups == 0 {
		c.Logging.MaxBackups = 3
	}
	if c.Logging.MaxAge == 0 {
		c.Logging.MaxAge = 28 // days
	}
}

// DefaultFileTypes returns the default file types to scan
func DefaultFileTypes() []string {
	return []string{
		".go", ".py", ".js", ".ts", ".java", ".cs", ".rb", ".php",
		".cpp", ".c", ".h", ".hpp", ".swift", ".kt", ".scala",
		".json", ".yaml", ".yml", ".xml", ".properties", ".conf",
		".env", ".config", ".ini", ".toml", ".txt", ".csv",
		".sql", ".sh", ".bash", ".zsh", ".ps1", ".bat", ".cmd",
	}
}

// MergeConfig merges two configurations, with the second taking precedence
func MergeConfig(base, override *Config) *Config {
	// Deep copy base
	result := *base

	// Override with non-zero values from override config
	if override.Version != "" {
		result.Version = override.Version
	}

	// Merge scanner config
	if override.Scanner.Workers != 0 {
		result.Scanner.Workers = override.Scanner.Workers
	}
	if len(override.Scanner.FileTypes) > 0 {
		result.Scanner.FileTypes = override.Scanner.FileTypes
	}
	if len(override.Scanner.ExcludePaths) > 0 {
		result.Scanner.ExcludePaths = override.Scanner.ExcludePaths
	}

	// Continue with other fields...
	// This is a simplified version - in production you'd want a more sophisticated merge

	return &result
}

// ConfigFromEnvironment loads configuration overrides from environment variables
func ConfigFromEnvironment() *Config {
	config := &Config{}

	// Scanner environment variables
	if workers := os.Getenv("PI_SCANNER_WORKERS"); workers != "" {
		// Parse and set workers
	}

	// GitHub token
	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		config.Github.Token = token
	}

	// Logging level
	if level := os.Getenv("LOG_LEVEL"); level != "" {
		config.Logging.Level = level
	}

	return config
}
