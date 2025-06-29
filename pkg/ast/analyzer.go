package ast

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
)

// Analyzer provides AST-based code analysis capabilities
type Analyzer struct {
	config *Config
}

// Config contains configuration for AST analysis
type Config struct {
	EnabledLanguages   []Language           `json:"enabled_languages"`
	BankingDomainRules *BankingDomainConfig `json:"banking_domain_rules"`
	MaxFileSize        int64                `json:"max_file_size"`
	ConcurrentAnalysis int                  `json:"concurrent_analysis"`
	CacheEnabled       bool                 `json:"cache_enabled"`
}

// Language represents supported programming languages
type Language string

const (
	LanguageJava   Language = "java"
	LanguageScala  Language = "scala"
	LanguagePython Language = "python"
	LanguageGo     Language = "go"
)

// BankingDomainConfig contains banking-specific analysis rules
type BankingDomainConfig struct {
	HighRiskPatterns   []string             `json:"high_risk_patterns"`
	MediumRiskPatterns []string             `json:"medium_risk_patterns"`
	LowRiskPatterns    []string             `json:"low_risk_patterns"`
	ExcludePatterns    []string             `json:"exclude_patterns"`
	RiskZones          map[string]RiskLevel `json:"risk_zones"`
}

// RiskLevel represents the risk level of a code location
type RiskLevel string

const (
	RiskLevelCritical RiskLevel = "CRITICAL"
	RiskLevelHigh     RiskLevel = "HIGH"
	RiskLevelMedium   RiskLevel = "MEDIUM"
	RiskLevelLow      RiskLevel = "LOW"
	RiskLevelIgnore   RiskLevel = "IGNORE"
)

// AnalysisResult contains the results of AST analysis
type AnalysisResult struct {
	Language      Language       `json:"language"`
	FilePath      string         `json:"file_path"`
	RiskLevel     RiskLevel      `json:"risk_level"`
	RiskZone      string         `json:"risk_zone"`
	CodeStructure *CodeStructure `json:"code_structure"`
	SecurityHints []SecurityHint `json:"security_hints"`
	Dependencies  []Dependency   `json:"dependencies"`
	AnalysisTime  int64          `json:"analysis_time_ms"`
	Error         string         `json:"error,omitempty"`
}

// CodeStructure represents the structural analysis of code
type CodeStructure struct {
	Classes     []ClassInfo    `json:"classes"`
	Methods     []MethodInfo   `json:"methods"`
	Variables   []VariableInfo `json:"variables"`
	Annotations []Annotation   `json:"annotations"`
	Imports     []string       `json:"imports"`
}

// ClassInfo represents information about a class
type ClassInfo struct {
	Name        string   `json:"name"`
	Package     string   `json:"package"`
	LineNumber  int      `json:"line_number"`
	IsPublic    bool     `json:"is_public"`
	Extends     string   `json:"extends,omitempty"`
	Implements  []string `json:"implements,omitempty"`
	Annotations []string `json:"annotations,omitempty"`
}

// MethodInfo represents information about a method
type MethodInfo struct {
	Name        string   `json:"name"`
	Class       string   `json:"class"`
	LineNumber  int      `json:"line_number"`
	IsPublic    bool     `json:"is_public"`
	Parameters  []string `json:"parameters"`
	ReturnType  string   `json:"return_type"`
	Annotations []string `json:"annotations,omitempty"`
}

// VariableInfo represents information about a variable
type VariableInfo struct {
	Name       string `json:"name"`
	Type       string `json:"type"`
	LineNumber int    `json:"line_number"`
	IsConstant bool   `json:"is_constant"`
	Scope      string `json:"scope"`
}

// Annotation represents code annotations/decorators
type Annotation struct {
	Name       string            `json:"name"`
	Parameters map[string]string `json:"parameters,omitempty"`
	LineNumber int               `json:"line_number"`
}

// SecurityHint represents a security-related observation
type SecurityHint struct {
	Type        string `json:"type"`
	Description string `json:"description"`
	LineNumber  int    `json:"line_number"`
	Severity    string `json:"severity"`
}

// Dependency represents a code dependency
type Dependency struct {
	Name    string `json:"name"`
	Type    string `json:"type"` // import, inheritance, composition
	Target  string `json:"target"`
	Package string `json:"package,omitempty"`
}

// NewAnalyzer creates a new AST analyzer
func NewAnalyzer(config *Config) *Analyzer {
	if config == nil {
		config = DefaultConfig()
	}
	return &Analyzer{config: config}
}

// DefaultConfig returns the default AST analysis configuration
func DefaultConfig() *Config {
	return &Config{
		EnabledLanguages: []Language{
			LanguageJava,
			LanguageScala,
			LanguagePython,
		},
		BankingDomainRules: DefaultBankingDomainConfig(),
		MaxFileSize:        1024 * 1024, // 1MB
		ConcurrentAnalysis: 4,
		CacheEnabled:       true,
	}
}

// DefaultBankingDomainConfig returns default banking domain configuration
func DefaultBankingDomainConfig() *BankingDomainConfig {
	return &BankingDomainConfig{
		HighRiskPatterns: []string{
			"*/model/*", "*/entity/*", "*/domain/*",
			"*/dto/*", "*/vo/*", "*/pojo/*",
			"*/customer/*", "*/account/*", "*/payment/*",
			"*/transaction/*", "*/user/*", "*/client/*",
		},
		MediumRiskPatterns: []string{
			"*/service/*", "*/controller/*", "*/api/*",
			"*/handler/*", "*/processor/*", "*/manager/*",
			"*/dao/*", "*/repository/*", "*/gateway/*",
		},
		LowRiskPatterns: []string{
			"*/util/*", "*/helper/*", "*/config/*",
			"*/exception/*", "*/error/*", "*/constant/*",
		},
		ExcludePatterns: []string{
			"*/test/*", "*/tests/*", "*/spec/*", "*/specs/*",
			"*_test.*", "*Test.*", "*Spec.*",
			"*/target/*", "*/build/*", "*/node_modules/*",
			"*.class", "*.jar", "*.war",
		},
		RiskZones: map[string]RiskLevel{
			"customer_data":      RiskLevelCritical,
			"financial_data":     RiskLevelCritical,
			"payment_processing": RiskLevelHigh,
			"authentication":     RiskLevelHigh,
			"business_logic":     RiskLevelMedium,
			"utilities":          RiskLevelLow,
			"tests":              RiskLevelIgnore,
		},
	}
}

// DefaultBankingConfig returns the default configuration for banking domain analysis
func DefaultBankingConfig() *Config {
	config := DefaultConfig()
	config.BankingDomainRules = DefaultBankingDomainConfig()
	return config
}

// AnalyzeFile performs AST analysis on a single file
func (a *Analyzer) AnalyzeFile(ctx context.Context, filePath string, content []byte) (*AnalysisResult, error) {
	language := a.detectLanguage(filePath)
	if !a.isLanguageEnabled(language) {
		return nil, fmt.Errorf("language %s not enabled for analysis", language)
	}

	result := &AnalysisResult{
		Language: language,
		FilePath: filePath,
	}

	// Determine risk level based on file path
	result.RiskLevel = a.DetermineRiskLevel(filePath)
	result.RiskZone = a.DetermineRiskZone(filePath)

	// Skip analysis for ignored files
	if result.RiskLevel == RiskLevelIgnore {
		return result, nil
	}

	// Perform language-specific AST analysis
	switch language {
	case LanguageJava:
		return a.analyzeJava(ctx, filePath, content, result)
	case LanguageScala:
		return a.analyzeScala(ctx, filePath, content, result)
	case LanguagePython:
		return a.analyzePython(ctx, filePath, content, result)
	default:
		return result, fmt.Errorf("unsupported language: %s", language)
	}
}

// detectLanguage determines the programming language from file extension
func (a *Analyzer) detectLanguage(filePath string) Language {
	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".java":
		return LanguageJava
	case ".scala":
		return LanguageScala
	case ".py":
		return LanguagePython
	case ".go":
		return LanguageGo
	default:
		return Language("")
	}
}

// isLanguageEnabled checks if a language is enabled for analysis
func (a *Analyzer) isLanguageEnabled(language Language) bool {
	for _, enabled := range a.config.EnabledLanguages {
		if enabled == language {
			return true
		}
	}
	return false
}

// DetermineRiskLevel determines the risk level based on file path patterns
func (a *Analyzer) DetermineRiskLevel(filePath string) RiskLevel {
	normalizedPath := strings.ToLower(filepath.ToSlash(filePath))

	// Check exclude patterns first
	for _, pattern := range a.config.BankingDomainRules.ExcludePatterns {
		if a.matchesPattern(normalizedPath, strings.ToLower(pattern)) {
			return RiskLevelIgnore
		}
	}

	// Check high risk patterns
	for _, pattern := range a.config.BankingDomainRules.HighRiskPatterns {
		if a.matchesPattern(normalizedPath, strings.ToLower(pattern)) {
			return RiskLevelHigh
		}
	}

	// Check medium risk patterns
	for _, pattern := range a.config.BankingDomainRules.MediumRiskPatterns {
		if a.matchesPattern(normalizedPath, strings.ToLower(pattern)) {
			return RiskLevelMedium
		}
	}

	// Check low risk patterns
	for _, pattern := range a.config.BankingDomainRules.LowRiskPatterns {
		if a.matchesPattern(normalizedPath, strings.ToLower(pattern)) {
			return RiskLevelLow
		}
	}

	// Default to medium risk for unknown patterns
	return RiskLevelMedium
}

// matchesPattern checks if a path matches a pattern (supports wildcards and directory patterns)
func (a *Analyzer) matchesPattern(path, pattern string) bool {
	// Handle wildcard patterns
	if strings.Contains(pattern, "*") {
		// Convert glob pattern to regex-like matching
		if strings.HasPrefix(pattern, "*/") && strings.HasSuffix(pattern, "/*") {
			// Pattern like "*/model/*" - check if path contains "/model/" or starts with "model/"
			inner := pattern[2 : len(pattern)-2] // Remove "*/" and "/*"
			return strings.Contains(path, "/"+inner+"/") || strings.HasPrefix(path, inner+"/")
		}

		if strings.HasPrefix(pattern, "*/") {
			// Pattern like "*/test" - check if path ends with "/test" or contains "/test."
			suffix := pattern[2:] // Remove "*/"
			return strings.Contains(path, "/"+suffix) || strings.HasSuffix(path, "/"+suffix) || strings.HasPrefix(path, suffix+"/")
		}

		if strings.HasSuffix(pattern, "/*") {
			// Pattern like "test/*" - check if path starts with "test/"
			prefix := pattern[:len(pattern)-2] // Remove "/*"
			return strings.HasPrefix(path, prefix+"/") || strings.Contains(path, "/"+prefix+"/")
		}

		// Handle patterns like "*_test.*" or "*Test.*"
		if strings.HasPrefix(pattern, "*") && strings.Contains(pattern, ".") {
			// File name pattern matching
			fileName := filepath.Base(path)
			matched, _ := filepath.Match(pattern, fileName)
			return matched
		}

		// Use filepath.Match for other patterns
		matched, _ := filepath.Match(pattern, path)
		return matched
	}

	// Direct string matching
	return strings.Contains(path, pattern)
}

// DetermineRiskZone categorizes the file into a risk zone
func (a *Analyzer) DetermineRiskZone(filePath string) string {
	normalizedPath := strings.ToLower(filepath.ToSlash(filePath))

	// Check test patterns first (highest priority)
	if strings.Contains(normalizedPath, "test") || strings.Contains(normalizedPath, "spec") ||
		strings.Contains(normalizedPath, "_test.") || strings.Contains(normalizedPath, "test.") {
		return "tests"
	}

	// Banking domain specific classifications
	if strings.Contains(normalizedPath, "customer") || strings.Contains(normalizedPath, "client") {
		return "customer_data"
	}
	if strings.Contains(normalizedPath, "payment") || strings.Contains(normalizedPath, "transaction") || strings.Contains(normalizedPath, "account") {
		return "financial_data"
	}
	if strings.Contains(normalizedPath, "auth") || strings.Contains(normalizedPath, "security") || strings.Contains(normalizedPath, "login") {
		return "authentication"
	}
	if strings.Contains(normalizedPath, "service") || strings.Contains(normalizedPath, "business") || strings.Contains(normalizedPath, "logic") {
		return "business_logic"
	}
	if strings.Contains(normalizedPath, "util") || strings.Contains(normalizedPath, "helper") || strings.Contains(normalizedPath, "config") {
		return "utilities"
	}

	return "general"
}
