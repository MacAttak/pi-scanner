package detection

import (
	"context"
	"time"
)

// PIType represents the type of personally identifiable information
type PIType string

const (
	PITypeTFN           PIType = "TFN"
	PITypeMedicare      PIType = "MEDICARE"
	PITypeABN           PIType = "ABN"
	PITypeACN           PIType = "ACN"
	PITypeBSB           PIType = "BSB"
	PITypeEmail         PIType = "EMAIL"
	PITypePhone         PIType = "PHONE"
	PITypeName          PIType = "NAME"
	PITypeAddress       PIType = "ADDRESS"
	PITypeCreditCard    PIType = "CREDIT_CARD"
	PITypeDriverLicense PIType = "DRIVER_LICENSE"
	PITypePassport      PIType = "PASSPORT"
	PITypeAccount       PIType = "ACCOUNT"
	PITypeIP            PIType = "IP_ADDRESS"
)

// RiskLevel represents the severity of a finding
type RiskLevel string

const (
	RiskLevelCritical RiskLevel = "CRITICAL"
	RiskLevelHigh     RiskLevel = "HIGH"
	RiskLevelMedium   RiskLevel = "MEDIUM"
	RiskLevelLow      RiskLevel = "LOW"
)

// Finding represents a detected PI instance
type Finding struct {
	// Core fields
	Type   PIType `json:"type"`
	Match  string `json:"match"`
	File   string `json:"file"`
	Line   int    `json:"line"`
	Column int    `json:"column"`

	// Context
	Context       string `json:"context"`
	ContextBefore string `json:"context_before"`
	ContextAfter  string `json:"context_after"`

	// Risk assessment
	RiskLevel       RiskLevel `json:"risk_level"`
	Confidence      float32   `json:"confidence"`
	ContextModifier float32   `json:"context_modifier"`

	// Validation
	Validated       bool   `json:"validated"`
	ValidationError string `json:"validation_error,omitempty"`

	// Metadata
	DetectedAt   time.Time `json:"detected_at"`
	DetectorName string    `json:"detector_name"`
}

// Detector is the interface for PI detection engines
type Detector interface {
	// Detect analyzes content and returns findings
	Detect(ctx context.Context, content []byte, filename string) ([]Finding, error)

	// Name returns the detector name
	Name() string
}

// PatternMatcher defines the interface for pattern-based detection
type PatternMatcher interface {
	// Match finds all pattern matches in content
	Match(content []byte) []PatternMatch

	// Type returns the PI type this matcher detects
	Type() PIType
}

// PatternMatch represents a regex pattern match
type PatternMatch struct {
	Value      string
	StartIndex int
	EndIndex   int
	Groups     map[string]string
}

// Validator validates specific PI types
type Validator interface {
	// Validate checks if the value is valid for this PI type
	Validate(value string) (bool, error)

	// Type returns the PI type this validator handles
	Type() PIType

	// Normalize returns a normalized version of the value
	Normalize(value string) string
}

// ScanResult represents the complete results of a scan
type ScanResult struct {
	Repository string      `json:"repository"`
	StartTime  time.Time   `json:"start_time"`
	EndTime    time.Time   `json:"end_time"`
	Findings   []Finding   `json:"findings"`
	Summary    ScanSummary `json:"summary"`
	Errors     []ScanError `json:"errors,omitempty"`
}

// ScanSummary provides aggregate statistics
type ScanSummary struct {
	TotalFiles    int               `json:"total_files"`
	ScannedFiles  int               `json:"scanned_files"`
	SkippedFiles  int               `json:"skipped_files"`
	TotalFindings int               `json:"total_findings"`
	ByRiskLevel   map[RiskLevel]int `json:"by_risk_level"`
	ByType        map[PIType]int    `json:"by_type"`
	Duration      time.Duration     `json:"duration"`
}

// ScanError represents an error during scanning
type ScanError struct {
	File  string    `json:"file"`
	Error string    `json:"error"`
	Time  time.Time `json:"time"`
}

// Config holds detection configuration
type Config struct {
	// Pattern matching
	EnableRegex    bool     `yaml:"enable_regex"`
	EnableGitleaks bool     `yaml:"enable_gitleaks"`
	CustomPatterns []string `yaml:"custom_patterns"`

	// Validation
	EnableValidation        bool `yaml:"enable_validation"`
	ValidateChecksums       bool `yaml:"validate_checksums"`
	EnableContextValidation bool `yaml:"enable_context_validation"`

	// Context analysis
	TestPathPatterns []string `yaml:"test_path_patterns"`
	MockPathPatterns []string `yaml:"mock_path_patterns"`
	ExcludePaths     []string `yaml:"exclude_paths"`

	// Confidence thresholds
	MinConfidenceThreshold float32 `yaml:"min_confidence_threshold"`
	ContextConfidenceBoost float32 `yaml:"context_confidence_boost"`

	// Risk scoring
	RiskWeights     map[PIType]int `yaml:"risk_weights"`
	ProximityWindow int            `yaml:"proximity_window"`

	// Performance
	MaxFileSize   int64 `yaml:"max_file_size"`
	MaxWorkers    int   `yaml:"max_workers"`
	EnableCaching bool  `yaml:"enable_cache"`
}

// DefaultConfig returns the default detection configuration
func DefaultConfig() *Config {
	return &Config{
		EnableRegex:             true,
		EnableGitleaks:          true,
		EnableValidation:        true,
		ValidateChecksums:       true,
		EnableContextValidation: true,

		TestPathPatterns: []string{
			// Go test patterns
			"*_test.go",
			"*/test/*",
			"*/tests/*",
			"*/testdata/*",
			"**/fixtures/*",
			"*/spec/*",

			// Java test patterns
			"*Test.java",
			"*Tests.java",
			"*/src/test/*",
			"*/test/java/*",
			"*/test/resources/*",

			// Scala test patterns
			"*Test.scala",
			"*Tests.scala",
			"*Spec.scala",
			"*Suite.scala",
			"*/src/test/*",
			"*/test/scala/*",

			// Python test patterns
			"test_*.py",
			"*_test.py",
			"*/tests/*",
			"*/test/*",
			"test*.py",
			"conftest.py",

			// JavaScript/TypeScript test patterns
			"*.test.js",
			"*.test.ts",
			"*.spec.js",
			"*.spec.ts",
			"*/__tests__/*",
			"*/test/*",
			"*/tests/*",
		},

		MockPathPatterns: []string{
			"*/mock/*",
			"*/mocks/*",
			"mock_*.go",
			"*_mock.go",
		},

		ExcludePaths: []string{
			"vendor/",
			"node_modules/",
			".git/",
			"*.min.js",
			"*.min.css",
		},

		// Confidence settings
		MinConfidenceThreshold: 0.4, // Lower threshold to allow invalid checksums to be detected
		ContextConfidenceBoost: 0.2,

		RiskWeights: map[PIType]int{
			PITypeTFN:        100,
			PITypeMedicare:   90,
			PITypeCreditCard: 90,
			PITypePassport:   80,
			PITypeABN:        60,
			PITypeACN:        60,
			PITypeBSB:        50,
			PITypeAccount:    50,
			PITypeName:       40,
			PITypeAddress:    40,
			PITypePhone:      30,
			PITypeEmail:      20,
			PITypeIP:         10,
		},

		ProximityWindow: 5,
		MaxFileSize:     10 * 1024 * 1024, // 10MB
		MaxWorkers:      0,                // 0 means runtime.NumCPU()
		EnableCaching:   true,
	}
}
