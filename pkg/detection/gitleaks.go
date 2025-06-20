package detection

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/spf13/viper"
	"github.com/zricethezav/gitleaks/v8/config"
	"github.com/zricethezav/gitleaks/v8/detect"
	"github.com/zricethezav/gitleaks/v8/report"
)

// gitleaksDetector wraps Gitleaks for PI detection
type gitleaksDetector struct {
	detector *detect.Detector
	config   config.Config
}

// NewGitleaksDetector creates a new Gitleaks-based detector
func NewGitleaksDetector(configPath string) (Detector, error) {
	// Read config file
	viperConfig := viper.New()
	viperConfig.SetConfigFile(configPath)
	viperConfig.SetConfigType("toml")

	if err := viperConfig.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read gitleaks config: %w", err)
	}

	// Create ViperConfig struct
	var vc config.ViperConfig
	if err := viperConfig.Unmarshal(&vc); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Translate to Config
	cfg, err := vc.Translate()
	if err != nil {
		return nil, fmt.Errorf("failed to translate config: %w", err)
	}

	detector := detect.NewDetector(cfg)
	detector.Verbose = false
	detector.Redact = 0 // 0 means don't redact secrets

	return &gitleaksDetector{
		detector: detector,
		config:   cfg,
	}, nil
}

// NewGitleaksDetectorWithDefaults creates a detector with default config + Australian rules
func NewGitleaksDetectorWithDefaults() (Detector, error) {
	// Create a temporary config file with our rules
	tmpFile, err := os.CreateTemp("", "gitleaks-*.toml")
	if err != nil {
		return nil, err
	}
	defer os.Remove(tmpFile.Name())

	// Write our config
	configContent := `[extend]
useDefault = true

[[rules]]
id = "australian-tfn"
description = "Australian Tax File Number"
regex = '''\b\d{3}[\s\-]?\d{3}[\s\-]?\d{3}\b'''
keywords = ["tfn", "tax file", "tax_file"]

[[rules]]
id = "australian-abn"
description = "Australian Business Number"  
regex = '''\b\d{2}[\s]?\d{3}[\s]?\d{3}[\s]?\d{3}\b'''
keywords = ["abn", "business number", "business_number"]

[[rules]]
id = "australian-medicare"
description = "Australian Medicare Number"
regex = '''\b[2-6]\d{3}[\s\-]?\d{5}[\s\-]?\d{1}(?:/\d)?\b'''
keywords = ["medicare", "health card", "health_card"]

[[rules]]
id = "australian-bsb"
description = "Australian Bank State Branch"
regex = '''\b\d{3}[\-]?\d{3}\b'''
keywords = ["bsb", "bank state", "branch"]
`
	if _, err := tmpFile.WriteString(configContent); err != nil {
		return nil, err
	}
	tmpFile.Close()

	return NewGitleaksDetector(tmpFile.Name())
}

// Name returns the detector name
func (g *gitleaksDetector) Name() string {
	return "gitleaks-detector"
}

// Detect analyzes content using Gitleaks
func (g *gitleaksDetector) Detect(ctx context.Context, content []byte, filename string) ([]Finding, error) {
	// Check context cancellation
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	// Create a fragment to scan
	fragment := detect.Fragment{
		Raw:      string(content),
		FilePath: filename,
	}

	// Run detection
	results := g.detector.Detect(fragment)

	// Convert Gitleaks findings to our format
	findings := make([]Finding, 0, len(results))
	for _, result := range results {
		finding := g.convertFinding(result, filename)
		findings = append(findings, finding)
	}

	return findings, nil
}

// convertFinding converts a Gitleaks finding to our format
func (g *gitleaksDetector) convertFinding(finding report.Finding, filename string) Finding {
	// Map Gitleaks rule IDs to our PI types
	piType := g.mapRuleToType(finding.RuleID)

	// Calculate line and column from finding (Gitleaks uses 0-based indexing)
	line := finding.StartLine + 1 // Convert to 1-based indexing
	column := finding.StartColumn + 1

	// Determine risk level based on rule
	riskLevel := g.calculateRiskLevel(finding)

	// Extract match value - Gitleaks provides the secret
	match := finding.Secret
	if match == "" && finding.Match != "" {
		match = finding.Match
	}

	return Finding{
		Type:            piType,
		Match:           match,
		File:            filename,
		Line:            line,
		Column:          column,
		Context:         finding.Line,
		ContextBefore:   "", // Gitleaks doesn't provide this
		ContextAfter:    "",
		RiskLevel:       riskLevel,
		Confidence:      0.9, // High confidence for Gitleaks rules
		ContextModifier: g.getContextModifier(filename),
		Validated:       false, // Gitleaks doesn't validate checksums
		DetectedAt:      time.Now(),
		DetectorName:    g.Name(),
	}
}

// mapRuleToType maps Gitleaks rule IDs to our PI types
func (g *gitleaksDetector) mapRuleToType(ruleID string) PIType {
	// Map specific rules first, then generic patterns
	switch ruleID {
	case "australian-tfn":
		return PITypeTFN
	case "australian-abn":
		return PITypeABN
	case "australian-medicare":
		return PITypeMedicare
	case "australian-bsb":
		return PITypeBSB
	case "australian-acn":
		return PIType("ACN")
	case "australian-passport":
		return PITypePassport
	case "australian-phone-mobile", "australian-phone-landline":
		return PITypePhone
	case "australian-drivers-license-nsw":
		return PITypeDriverLicense
	case "aws-access-key-id":
		return PIType("AWS_ACCESS_KEY")
	case "github-pat":
		return PIType("GITHUB_TOKEN")
	case "private-key":
		return PIType("PRIVATE_KEY")
	default:
		// For other rules, map by pattern matching
		switch {
		case strings.Contains(ruleID, "tfn") || strings.Contains(ruleID, "tax-file"):
			return PITypeTFN
		case strings.Contains(ruleID, "abn") || strings.Contains(ruleID, "business-number"):
			return PITypeABN
		case strings.Contains(ruleID, "medicare"):
			return PITypeMedicare
		case strings.Contains(ruleID, "bsb") || strings.Contains(ruleID, "bank-state"):
			return PITypeBSB
		case strings.Contains(ruleID, "email"):
			return PITypeEmail
		case strings.Contains(ruleID, "phone"):
			return PITypePhone
		case strings.Contains(ruleID, "credit") || strings.Contains(ruleID, "card"):
			return PITypeCreditCard
		case strings.Contains(ruleID, "passport"):
			return PITypePassport
		case strings.Contains(ruleID, "driver") || strings.Contains(ruleID, "license"):
			return PITypeDriverLicense
		case strings.Contains(ruleID, "ip") || strings.Contains(ruleID, "address"):
			return PITypeIP
		default:
			// For API keys and secrets, map to a generic type
			return PIType("SECRET_" + strings.ToUpper(ruleID))
		}
	}
}

// calculateRiskLevel determines risk based on the finding
func (g *gitleaksDetector) calculateRiskLevel(finding report.Finding) RiskLevel {
	// Check tags for severity hints
	for _, tag := range finding.Tags {
		switch strings.ToLower(tag) {
		case "critical", "high-risk":
			return RiskLevelCritical
		case "high":
			return RiskLevelHigh
		case "medium":
			return RiskLevelMedium
		case "low":
			return RiskLevelLow
		}
	}

	// Default risk levels based on rule type
	if strings.Contains(finding.RuleID, "key") || strings.Contains(finding.RuleID, "token") {
		return RiskLevelHigh
	}

	return RiskLevelMedium
}

// getContextModifier returns risk modifier based on file context
func (g *gitleaksDetector) getContextModifier(filename string) float32 {
	// Reduce risk for test files
	lowerName := strings.ToLower(filename)
	if strings.Contains(lowerName, "test") ||
		strings.Contains(lowerName, "spec") ||
		strings.Contains(lowerName, "mock") {
		return 0.1
	}

	// Reduce risk for example/sample files
	if strings.Contains(lowerName, "example") ||
		strings.Contains(lowerName, "sample") {
		return 0.3
	}

	return 1.0
}

// CreateAustralianPIRules creates Gitleaks rules for Australian PI
func CreateAustralianPIRules() []config.Rule {
	rules := []config.Rule{
		{
			RuleID:      "australian-tfn",
			Description: "Australian Tax File Number",
			Regex:       regexp.MustCompile(`\b\d{3}[\s\-]?\d{3}[\s\-]?\d{3}\b`),
			Tags:        []string{"PII", "TFN", "Australia"},
		},
		{
			RuleID:      "australian-abn",
			Description: "Australian Business Number",
			Regex:       regexp.MustCompile(`\b\d{2}[\s]?\d{3}[\s]?\d{3}[\s]?\d{3}\b`),
			Tags:        []string{"PII", "ABN", "Australia"},
		},
		{
			RuleID:      "australian-medicare",
			Description: "Australian Medicare Number",
			Regex:       regexp.MustCompile(`\b[2-6]\d{3}[\s\-]?\d{5}[\s\-]?\d{1}(?:/\d)?\b`),
			Tags:        []string{"PII", "Medicare", "Australia"},
		},
		{
			RuleID:      "australian-bsb",
			Description: "Australian Bank State Branch",
			Regex:       regexp.MustCompile(`\b\d{3}[\-]?\d{3}\b`),
			Tags:        []string{"PII", "BSB", "Australia", "Financial"},
		},
		{
			RuleID:      "australian-acn",
			Description: "Australian Company Number",
			Regex:       regexp.MustCompile(`\b\d{3}[\s]?\d{3}[\s]?\d{3}\b`),
			Tags:        []string{"PII", "ACN", "Australia"},
		},
	}

	return rules
}
