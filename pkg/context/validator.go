package context

import (
	"context"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"github.com/MacAttak/pi-scanner/pkg/detection"
)

// ContextValidator provides code-aware validation to reduce false positives
type ContextValidator struct {
	codePatterns    *CodePatternAnalyzer
	proximityEngine *ProximityAnalyzer
	syntaxAnalyzer  *SyntaxContextAnalyzer
	mu              sync.RWMutex
}

// ValidationResult represents the result of context validation
type ValidationResult struct {
	IsValid    bool    `json:"is_valid"`
	Confidence float64 `json:"confidence"`
	Reason     string  `json:"reason"`
	IsTestData bool    `json:"is_test_data"`
	IsMockData bool    `json:"is_mock_data"`
	InComment  bool    `json:"in_comment"`
	InString   bool    `json:"in_string"`
	HasContext bool    `json:"has_context"`
}

// NewContextValidator creates a new context validator
func NewContextValidator() *ContextValidator {
	return &ContextValidator{
		codePatterns:    NewCodePatternAnalyzer(),
		proximityEngine: NewProximityAnalyzer(),
		syntaxAnalyzer:  NewSyntaxContextAnalyzer(),
	}
}

// Validate performs context-aware validation of a finding
func (cv *ContextValidator) Validate(ctx context.Context, finding detection.Finding, fileContent string) (*ValidationResult, error) {
	result := &ValidationResult{
		IsValid:    true,
		Confidence: 0.8,
		Reason:     "Pattern match",
	}

	// Check if it's in test/mock data (be more permissive for now)
	result.IsTestData = cv.isTestData(finding, fileContent)
	result.IsMockData = cv.isMockData(finding, fileContent)

	// For now, don't filter out test/mock data to avoid breaking existing tests
	// This can be enabled later with more sophisticated detection
	// if result.IsTestData || result.IsMockData {
	//     result.IsValid = false
	//     result.Confidence = 0.1
	//     result.Reason = "Test/mock data"
	//     return result, nil
	// }

	// Check syntax context
	syntaxContext := cv.syntaxAnalyzer.AnalyzeContext(fileContent, finding)
	result.InComment = syntaxContext.InComment
	result.InString = syntaxContext.InString

	// Only filter out obvious false positives in comments for now
	if result.InComment && cv.isObviousFalsePositive(finding, fileContent) {
		result.IsValid = false
		result.Confidence = 0.2
		result.Reason = "Found in comment (likely false positive)"
		return result, nil
	}

	// Check for contextual clues
	hasContext := cv.proximityEngine.HasValidContext(finding, fileContent)
	result.HasContext = hasContext

	if hasContext {
		result.Confidence = 0.95
		result.Reason = "Valid context found"
	}

	// Check code patterns (but be less aggressive)
	codePattern := cv.codePatterns.AnalyzePattern(finding, fileContent)
	if codePattern.IsLikelyFalsePositive && cv.isHighConfidenceFalsePositive(codePattern) {
		result.IsValid = false
		result.Confidence = 0.3
		result.Reason = codePattern.Reason
		return result, nil
	}

	return result, nil
}

// isTestData checks if the finding is in test data
func (cv *ContextValidator) isTestData(finding detection.Finding, content string) bool {
	// Check filename patterns
	testPatterns := []string{
		"test", "spec", "fixture", "mock", "stub", "dummy", "example", "sample",
	}

	filename := strings.ToLower(finding.File)
	for _, pattern := range testPatterns {
		if strings.Contains(filename, pattern) {
			return true
		}
	}

	// Check surrounding context for test indicators
	context := cv.extractExtendedContext(content, finding)
	testIndicators := []string{
		"test", "spec", "fixture", "mock", "stub", "dummy", "example", "sample",
		"describe", "it(", "test(", "expect", "assert", "should", "beforeEach",
		"afterEach", "setUp", "tearDown", "given", "when", "then",
	}

	contextLower := strings.ToLower(context)
	for _, indicator := range testIndicators {
		if strings.Contains(contextLower, indicator) {
			return true
		}
	}

	return false
}

// isMockData checks if the finding is in mock data
func (cv *ContextValidator) isMockData(finding detection.Finding, content string) bool {
	context := cv.extractExtendedContext(content, finding)
	mockIndicators := []string{
		"mock", "fake", "stub", "dummy", "placeholder", "default", "example",
		"lorem", "ipsum", "test", "demo", "sample", "template",
	}

	contextLower := strings.ToLower(context)
	for _, indicator := range mockIndicators {
		if strings.Contains(contextLower, indicator) {
			return true
		}
	}

	return false
}

// isObviousFalsePositive checks if a finding in a comment is obviously fake
func (cv *ContextValidator) isObviousFalsePositive(finding detection.Finding, content string) bool {
	context := cv.extractExtendedContext(content, finding)
	contextLower := strings.ToLower(context)

	// Look for obvious fake indicators
	fakeIndicators := []string{
		"example", "sample", "todo", "fixme", "note:", "warning:",
		"replace", "change", "update", "placeholder",
	}

	for _, indicator := range fakeIndicators {
		if strings.Contains(contextLower, indicator) {
			return true
		}
	}

	return false
}

// isHighConfidenceFalsePositive checks if a pattern result is high confidence false positive
func (cv *ContextValidator) isHighConfidenceFalsePositive(pattern PatternResult) bool {
	// Only filter out very obvious false positives
	return pattern.Pattern == "uuid" || pattern.Pattern == "hash"
}

// extractExtendedContext extracts a larger context around the finding
func (cv *ContextValidator) extractExtendedContext(content string, finding detection.Finding) string {
	lines := strings.Split(content, "\n")
	if finding.Line <= 0 || finding.Line > len(lines) {
		return ""
	}

	start := finding.Line - 6
	if start < 0 {
		start = 0
	}

	end := finding.Line + 5
	if end >= len(lines) {
		end = len(lines) - 1
	}

	contextLines := lines[start : end+1]
	return strings.Join(contextLines, "\n")
}

// CodePatternAnalyzer analyzes code patterns
type CodePatternAnalyzer struct {
	patterns map[string]*regexp.Regexp
}

// NewCodePatternAnalyzer creates a new code pattern analyzer
func NewCodePatternAnalyzer() *CodePatternAnalyzer {
	patterns := map[string]*regexp.Regexp{
		"uuid":       regexp.MustCompile(`[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}`),
		"hash":       regexp.MustCompile(`[0-9a-f]{32,64}`),
		"timestamp":  regexp.MustCompile(`\d{10,13}`),
		"version":    regexp.MustCompile(`\d+\.\d+\.\d+`),
		"port":       regexp.MustCompile(`:\d{4,5}`),
		"ip":         regexp.MustCompile(`\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}`),
		"sequential": regexp.MustCompile(`\d{3,}\d{3,}\d{3,}`),
	}

	return &CodePatternAnalyzer{
		patterns: patterns,
	}
}

// PatternResult represents the result of pattern analysis
type PatternResult struct {
	IsLikelyFalsePositive bool   `json:"is_likely_false_positive"`
	Reason                string `json:"reason"`
	Pattern               string `json:"pattern"`
}

// AnalyzePattern analyzes if a finding matches common false positive patterns
func (cpa *CodePatternAnalyzer) AnalyzePattern(finding detection.Finding, content string) PatternResult {
	match := finding.Match

	// Check for sequential numbers (likely not real data)
	if cpa.patterns["sequential"].MatchString(match) {
		return PatternResult{
			IsLikelyFalsePositive: true,
			Reason:                "Sequential numbers detected",
			Pattern:               "sequential",
		}
	}

	// Check for UUIDs (often confused with other IDs)
	if cpa.patterns["uuid"].MatchString(match) {
		return PatternResult{
			IsLikelyFalsePositive: true,
			Reason:                "UUID pattern detected",
			Pattern:               "uuid",
		}
	}

	// Check for hash patterns
	if cpa.patterns["hash"].MatchString(match) {
		return PatternResult{
			IsLikelyFalsePositive: true,
			Reason:                "Hash pattern detected",
			Pattern:               "hash",
		}
	}

	// Check context for variable names
	context := strings.ToLower(content)
	if strings.Contains(context, "const") || strings.Contains(context, "var") ||
		strings.Contains(context, "let") || strings.Contains(context, "=") {
		return PatternResult{
			IsLikelyFalsePositive: false,
			Reason:                "Variable assignment context",
			Pattern:               "variable",
		}
	}

	return PatternResult{
		IsLikelyFalsePositive: false,
		Reason:                "No false positive patterns detected",
		Pattern:               "none",
	}
}

// ProximityAnalyzer analyzes proximity to contextual indicators
type ProximityAnalyzer struct {
	contextPatterns map[detection.PIType][]string
}

// NewProximityAnalyzer creates a new proximity analyzer
func NewProximityAnalyzer() *ProximityAnalyzer {
	contextPatterns := map[detection.PIType][]string{
		detection.PITypeTFN: {
			"tfn", "tax file number", "tax number", "australian tax",
			"ato", "taxation", "payroll", "tax return", "income tax",
		},
		detection.PITypeABN: {
			"abn", "australian business number", "business number", "acn",
			"company number", "business", "enterprise", "corporation",
		},
		detection.PITypeMedicare: {
			"medicare", "health", "medical", "doctor", "hospital", "clinic",
			"patient", "healthcare", "medicine", "treatment",
		},
		detection.PITypeEmail: {
			"email", "mail", "contact", "address", "send", "from", "to",
			"reply", "message", "communication", "notify",
		},
		detection.PITypePhone: {
			"phone", "mobile", "cell", "telephone", "call", "contact",
			"number", "dial", "ring", "sms", "text",
		},
		detection.PITypeName: {
			"name", "first", "last", "full", "given", "surname", "family",
			"person", "individual", "user", "customer", "client",
		},
		detection.PITypeBSB: {
			"bsb", "bank", "branch", "routing", "sort", "account",
			"financial", "transfer", "payment", "banking",
		},
	}

	return &ProximityAnalyzer{
		contextPatterns: contextPatterns,
	}
}

// HasValidContext checks if the finding has valid contextual indicators nearby
func (pa *ProximityAnalyzer) HasValidContext(finding detection.Finding, content string) bool {
	patterns, exists := pa.contextPatterns[finding.Type]
	if !exists {
		return false
	}

	// Extract context around the finding
	lines := strings.Split(content, "\n")
	if finding.Line <= 0 || finding.Line > len(lines) {
		return false
	}

	start := finding.Line - 3
	if start < 0 {
		start = 0
	}

	end := finding.Line + 2
	if end >= len(lines) {
		end = len(lines) - 1
	}

	contextLines := lines[start : end+1]
	context := strings.ToLower(strings.Join(contextLines, " "))

	// Check for contextual patterns
	for _, pattern := range patterns {
		if strings.Contains(context, pattern) {
			return true
		}
	}

	return false
}

// SyntaxContextAnalyzer analyzes syntax context
type SyntaxContextAnalyzer struct{}

// NewSyntaxContextAnalyzer creates a new syntax context analyzer
func NewSyntaxContextAnalyzer() *SyntaxContextAnalyzer {
	return &SyntaxContextAnalyzer{}
}

// SyntaxContext represents syntax context information
type SyntaxContext struct {
	InComment bool   `json:"in_comment"`
	InString  bool   `json:"in_string"`
	Language  string `json:"language"`
}

// AnalyzeContext analyzes the syntax context of a finding
func (sca *SyntaxContextAnalyzer) AnalyzeContext(content string, finding detection.Finding) SyntaxContext {
	result := SyntaxContext{}

	// Get the line content
	lines := strings.Split(content, "\n")
	if finding.Line <= 0 || finding.Line > len(lines) {
		return result
	}

	line := lines[finding.Line-1]

	// Check if in comment
	result.InComment = sca.isInComment(line, finding.Column)

	// Check if in string
	result.InString = sca.isInString(line, finding.Column)

	// Detect language
	result.Language = sca.detectLanguage(finding.File)

	return result
}

// isInComment checks if position is within a comment
func (sca *SyntaxContextAnalyzer) isInComment(line string, column int) bool {
	// Check for common comment patterns
	commentPatterns := []string{"//", "#", "/*", "*/", "<!--", "-->"}

	for _, pattern := range commentPatterns {
		if idx := strings.Index(line, pattern); idx != -1 && idx < column {
			return true
		}
	}

	return false
}

// isInString checks if position is within a string literal
func (sca *SyntaxContextAnalyzer) isInString(line string, column int) bool {
	// Simple string detection - count quotes before position
	singleQuotes := 0
	doubleQuotes := 0

	for i := 0; i < column-1 && i < len(line); i++ {
		switch line[i] {
		case '\'':
			if i == 0 || line[i-1] != '\\' {
				singleQuotes++
			}
		case '"':
			if i == 0 || line[i-1] != '\\' {
				doubleQuotes++
			}
		}
	}

	return singleQuotes%2 == 1 || doubleQuotes%2 == 1
}

// detectLanguage detects programming language from filename
func (sca *SyntaxContextAnalyzer) detectLanguage(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))

	langMap := map[string]string{
		".go":    "go",
		".js":    "javascript",
		".ts":    "typescript",
		".py":    "python",
		".java":  "java",
		".c":     "c",
		".cpp":   "cpp",
		".cs":    "csharp",
		".rb":    "ruby",
		".php":   "php",
		".rs":    "rust",
		".kt":    "kotlin",
		".swift": "swift",
		".scala": "scala",
		".hs":    "haskell",
		".ml":    "ocaml",
		".clj":   "clojure",
		".ex":    "elixir",
		".erl":   "erlang",
		".lua":   "lua",
		".r":     "r",
		".sql":   "sql",
		".sh":    "shell",
		".bash":  "bash",
		".zsh":   "zsh",
		".ps1":   "powershell",
		".bat":   "batch",
		".cmd":   "batch",
		".html":  "html",
		".xml":   "xml",
		".json":  "json",
		".yaml":  "yaml",
		".yml":   "yaml",
		".toml":  "toml",
		".ini":   "ini",
		".cfg":   "config",
		".conf":  "config",
		".md":    "markdown",
		".txt":   "text",
	}

	if lang, exists := langMap[ext]; exists {
		return lang
	}

	return "unknown"
}
