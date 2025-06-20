package proximity

import (
	"regexp"
	"sort"
	"strings"
)

// PatternMatcher provides methods to identify various patterns that indicate PI context vs test data
type PatternMatcher struct {
	// Compiled regex patterns for performance
	testDataPattern      *regexp.Regexp
	piLabelPattern       *regexp.Regexp
	documentationPattern *regexp.Regexp
	formFieldPattern     *regexp.Regexp
	databasePattern      *regexp.Regexp
	logPattern           *regexp.Regexp
	configPattern        *regexp.Regexp
	variablePattern      *regexp.Regexp
}

// NewPatternMatcher creates a new pattern matcher with compiled patterns
func NewPatternMatcher() *PatternMatcher {
	pm := &PatternMatcher{}
	pm.compilePatterns()
	return pm
}

// compilePatterns compiles all regex patterns for performance
func (pm *PatternMatcher) compilePatterns() {
	// Test data keywords pattern
	testKeywords := []string{
		"test", "TEST", "Test", "testing", "TESTING", "Testing",
		"example", "EXAMPLE", "Example",
		"mock", "MOCK", "Mock", "mocked", "MOCKED", "Mocked",
		"sample", "SAMPLE", "Sample",
		"demo", "DEMO", "Demo",
		"fake", "FAKE", "Fake",
		"dummy", "DUMMY", "Dummy",
		"placeholder", "PLACEHOLDER", "Placeholder",
	}

	// Create pattern that matches test keywords as whole words or with common separators
	testPatterns := make([]string, 0, len(testKeywords)*5)
	for _, keyword := range testKeywords {
		testPatterns = append(testPatterns,
			`\b`+regexp.QuoteMeta(keyword)+`\b`,    // whole word
			`\b`+regexp.QuoteMeta(keyword)+`[_-]`,  // with underscore or dash
			`[_-]`+regexp.QuoteMeta(keyword)+`\b`,  // prefixed with underscore or dash
			regexp.QuoteMeta(keyword)+`Data`,       // camelCase pattern (e.g., mockData)
			`\b`+regexp.QuoteMeta(keyword)+`[A-Z]`, // camelCase prefix (e.g., mockSSN)
		)
	}
	pm.testDataPattern = regexp.MustCompile(`(?i)` + strings.Join(testPatterns, "|"))

	// PI context labels pattern
	piLabels := []string{
		// SSN variations
		"SSN", "ssn", "Social Security Number", "social security number",
		"Social Security No", "social security no",

		// TFN variations
		"TFN", "tfn", "Tax File Number", "tax file number",
		"Tax File No", "tax file no",

		// Medicare variations
		"Medicare No", "medicare no", "Medicare Number", "medicare number",
		"Medicare Card", "medicare card", "Medicare", "medicare",

		// ABN variations
		"ABN", "abn", "Australian Business Number", "australian business number",
		"Business Number", "business number", "Company ABN", "company abn",

		// BSB variations
		"BSB", "bsb", "Bank State Branch", "bank state branch",
		"BSB Code", "bsb code", "Branch Code", "branch code",
		"Routing Code", "routing code", "Bank Code", "bank code",

		// Australian Address variations
		"Address", "address", "Street Address", "street address",
		"Postal Address", "postal address", "Home Address", "home address",
		"Residential Address", "residential address",

		// Credit Card variations
		"Credit Card", "credit card", "CC", "cc", "Card Number", "card number",

		// Phone variations
		"Phone", "phone", "Phone Number", "phone number", "Mobile", "mobile",
		"Tel", "tel", "Telephone", "telephone",

		// Email variations
		"Email", "email", "Email Address", "email address", "E-mail", "e-mail",

		// Driver License variations
		"Driver License", "driver license", "DL", "dl", "License Number", "license number",
		"Drivers License", "drivers license",

		// Passport variations
		"Passport", "passport", "Passport Number", "passport number",
		"Passport No", "passport no",
	}

	// Sort labels by length (longest first) to ensure proper matching precedence
	sort.Slice(piLabels, func(i, j int) bool {
		return len(piLabels[i]) > len(piLabels[j])
	})

	labelPatterns := make([]string, 0, len(piLabels))
	for _, label := range piLabels {
		// Match label followed by colon, equals, or space
		labelPatterns = append(labelPatterns, regexp.QuoteMeta(label)+`\s*[:=\s]`)
	}
	pm.piLabelPattern = regexp.MustCompile(`(?i)` + strings.Join(labelPatterns, "|"))

	// Documentation patterns
	pm.documentationPattern = regexp.MustCompile(`(?s)` + strings.Join([]string{
		`//.*?$`,     // Single line C++ style comments
		`/\*.*?\*/`,  // Multi-line C style comments
		`#.*?$`,      // Hash comments
		`<!--.*?-->`, // HTML comments
		`--.*?$`,     // SQL comments
		`""".*?"""`,  // Python docstrings
		`'''.*?'''`,  // Python docstrings alternate
	}, "|"))

	// Form field patterns
	pm.formFieldPattern = regexp.MustCompile(`(?i)` + strings.Join([]string{
		`<input[^>]*>`,                  // HTML input tags
		`<textarea[^>]*>.*?</textarea>`, // HTML textarea
		`<select[^>]*>.*?</select>`,     // HTML select
		`&\w+=[^&\s]*`,                  // Form data key=value with & prefix
		`\?\w+=[^&\s]*`,                 // Query string parameters
		`^\s*\{.*"\w+"\s*:\s*"[^"]*"`,   // JSON object (more specific pattern)
	}, "|"))

	// Database patterns
	pm.databasePattern = regexp.MustCompile(`(?i)` + strings.Join([]string{
		`\bSELECT\b.*?\bFROM\b`, // SELECT queries
		`\bINSERT\s+INTO\b`,     // INSERT queries
		`\bUPDATE\b.*?\bSET\b`,  // UPDATE queries
		`\bDELETE\s+FROM\b`,     // DELETE queries
		`\bWHERE\b.*?=`,         // WHERE clauses
		`\.where\s*\(`,          // ORM where methods
		`\.filter\s*\(`,         // ORM filter methods
		`\.findOne\s*\(`,        // ORM findOne methods
		`mongodb://`,            // MongoDB connection strings
		`jdbc:`,                 // JDBC URLs
	}, "|"))

	// Log patterns
	pm.logPattern = regexp.MustCompile(`(?i)` + strings.Join([]string{
		`\b(INFO|DEBUG|ERROR|WARN|TRACE|FATAL)\s*[:]\s*`, // Log levels with colon
		`\b(info|debug|error|warn|trace|fatal)\s*[:]\s*`, // Lowercase log levels
		`\d{4}-\d{2}-\d{2}.*?(INFO|DEBUG|ERROR|WARN)`,    // Timestamp with log level
		`logger\.(info|debug|error|warn|trace)`,          // Logger method calls
		`console\.(log|info|debug|error|warn)`,           // Console logging
		`<\d+>.*?:`,                                      // Syslog format
	}, "|"))

	// Configuration patterns
	pm.configPattern = regexp.MustCompile(strings.Join([]string{
		`^\s*[a-z_][a-z0-9_]*\s*=\s*[^=]`,                 // Key=value assignments (lowercase keys)
		`(?i)\w+\.\w+(\.\w+)*\s*=\s*[^=\[]`,               // Properties format like app.config.value= (not array access)
		`^\s*[a-z_][a-z0-9_]*\s*:\s*[^:]`,                 // YAML style key: value (lowercase keys only)
		`(?i)export\s+\w+\s*=`,                             // Environment variable exports
		`(?i)\[\w+\]`,                                      // INI section headers
		`(?i)(default|fallback|initial|config|setting)_[a-z_]+\s*=`, // Default/fallback/config values (lowercase)
		`(?i)\w+_(tfn|ssn|medicare|abn|bsb)_\w+\s*=`,      // Config patterns with PI type names
	}, "|"))

	// Variable patterns
	pm.variablePattern = regexp.MustCompile(`(?i)` + strings.Join([]string{
		`\b(var|let|const)\s+\w+\s*=`,                    // JavaScript variable declarations
		`^\s*[a-zA-Z_]\w*\s*=\s*[^=]`,                    // Simple assignments at start of line (no object properties)
		`(var|string|int|float|double)\s+\w+.*?=`,        // Typed variable declarations (including Go)
		`function\s+\w+\s*\([^)]*=`,                      // Function parameters with defaults
		`\(\s*\w+\s*=`,                                   // Lambda/arrow function parameters
		`const\s*{\s*\w+\s*=`,                            // Destructuring with defaults
		`const\s*\[\s*\w+\s*=`,                           // Array destructuring with defaults
		`(test_|mock|sample_|demo_|fake_|dummy_)\w+\s*=`, // Test variable patterns
	}, "|"))
}

// ContainsTestDataKeywords checks if the text contains keywords that indicate test/mock data
func (pm *PatternMatcher) ContainsTestDataKeywords(text string) bool {
	return pm.testDataPattern.MatchString(text)
}

// FindPIContextLabels finds PI context labels in the text
func (pm *PatternMatcher) FindPIContextLabels(text string) []string {
	matches := pm.piLabelPattern.FindAllString(text, -1)
	labels := make([]string, 0, len(matches))

	for _, match := range matches {
		// Clean up the match by removing trailing separators
		label := strings.TrimSpace(match)
		label = strings.TrimRight(label, ":=")
		label = strings.TrimSpace(label)
		if label != "" {
			labels = append(labels, label)
		}
	}

	// Remove duplicates
	seen := make(map[string]bool)
	unique := make([]string, 0, len(labels))
	for _, label := range labels {
		if !seen[strings.ToLower(label)] {
			seen[strings.ToLower(label)] = true
			unique = append(unique, label)
		}
	}

	return unique
}

// IsDocumentationContext checks if the text appears to be documentation/comments
func (pm *PatternMatcher) IsDocumentationContext(text string) bool {
	return pm.documentationPattern.MatchString(text)
}

// IsFormFieldContext checks if the text appears to be form field related
func (pm *PatternMatcher) IsFormFieldContext(text string) bool {
	return pm.formFieldPattern.MatchString(text)
}

// IsDatabaseContext checks if the text appears to be database query related
func (pm *PatternMatcher) IsDatabaseContext(text string) bool {
	return pm.databasePattern.MatchString(text)
}

// IsLogContext checks if the text appears to be log entry related
func (pm *PatternMatcher) IsLogContext(text string) bool {
	return pm.logPattern.MatchString(text)
}

// IsConfigurationContext checks if the text appears to be configuration related
func (pm *PatternMatcher) IsConfigurationContext(text string) bool {
	return pm.configPattern.MatchString(text)
}

// IsVariableContext checks if the text appears to be variable assignment related
func (pm *PatternMatcher) IsVariableContext(text string) bool {
	return pm.variablePattern.MatchString(text)
}

// GetTestDataConfidence returns a confidence score for test data detection
func (pm *PatternMatcher) GetTestDataConfidence(text string) float64 {
	if !pm.ContainsTestDataKeywords(text) {
		return 0.0
	}

	// Count test keywords
	matches := pm.testDataPattern.FindAllString(text, -1)
	keywordCount := len(matches)

	// More keywords = higher confidence it's test data
	confidence := float64(keywordCount) * 0.3
	if confidence > 1.0 {
		confidence = 1.0
	}

	// Boost confidence for strong test indicators
	strongIndicators := []string{"test", "mock", "fake", "dummy", "sample", "example"}
	for _, indicator := range strongIndicators {
		if strings.Contains(strings.ToLower(text), indicator) {
			confidence += 0.2
			if confidence > 1.0 {
				confidence = 1.0
				break
			}
		}
	}

	return confidence
}

// GetPIContextConfidence returns a confidence score for PI context detection
func (pm *PatternMatcher) GetPIContextConfidence(text string) float64 {
	confidence := 0.0

	// PI labels are strong indicators
	if labels := pm.FindPIContextLabels(text); len(labels) > 0 {
		confidence += 0.8
	}

	// Database context is a strong indicator
	if pm.IsDatabaseContext(text) {
		confidence += 0.6
	}

	// Form context is a strong indicator
	if pm.IsFormFieldContext(text) {
		confidence += 0.6
	}

	// Log context is moderate indicator
	if pm.IsLogContext(text) {
		confidence += 0.4
	}

	// Configuration context is moderate indicator
	if pm.IsConfigurationContext(text) {
		confidence += 0.3
	}

	// Documentation context reduces confidence
	if pm.IsDocumentationContext(text) {
		confidence -= 0.3
	}

	// Variable context reduces confidence (might be variable names)
	if pm.IsVariableContext(text) {
		confidence -= 0.2
	}

	// Ensure confidence is within bounds
	if confidence < 0.0 {
		confidence = 0.0
	}
	if confidence > 1.0 {
		confidence = 1.0
	}

	return confidence
}

// AnalyzeContextType determines the most likely context type
func (pm *PatternMatcher) AnalyzeContextType(text string) PIContextType {
	// Check in order of priority

	// Test data (highest priority for filtering out)
	if pm.ContainsTestDataKeywords(text) {
		return PIContextTest
	}

	// PI labels (highest priority for real PI)
	if len(pm.FindPIContextLabels(text)) > 0 {
		return PIContextLabel
	}

	// Structured contexts
	if pm.IsDatabaseContext(text) {
		return PIContextDatabase
	}

	if pm.IsFormFieldContext(text) {
		return PIContextForm
	}

	if pm.IsLogContext(text) {
		return PIContextLog
	}

	if pm.IsConfigurationContext(text) {
		return PIContextConfig
	}

	// Lower confidence contexts
	if pm.IsVariableContext(text) {
		return PIContextVariable
	}

	if pm.IsDocumentationContext(text) {
		return PIContextDocumentation
	}

	// Default to production context
	return PIContextProduction
}

// ExtractRelevantKeywords extracts keywords that are relevant for context analysis
func (pm *PatternMatcher) ExtractRelevantKeywords(text string) []string {
	keywords := make(map[string]bool)

	// Add PI labels
	for _, label := range pm.FindPIContextLabels(text) {
		keywords[strings.ToLower(label)] = true
	}

	// Add test data keywords if found
	if matches := pm.testDataPattern.FindAllString(text, -1); len(matches) > 0 {
		for _, match := range matches {
			cleaned := strings.ToLower(strings.Trim(match, "_-"))
			keywords[cleaned] = true
		}
	}

	// Add context type indicators
	contextIndicators := map[string][]string{
		"database": {"select", "insert", "update", "delete", "where", "from"},
		"form":     {"input", "form", "field", "value"},
		"log":      {"info", "debug", "error", "warn", "trace"},
		"config":   {"config", "setting", "parameter", "default"},
		"variable": {"var", "let", "const", "variable"},
		"doc":      {"comment", "documentation", "example"},
	}

	textLower := strings.ToLower(text)
	for contextType, indicators := range contextIndicators {
		for _, indicator := range indicators {
			if strings.Contains(textLower, indicator) {
				keywords[contextType] = true
				break
			}
		}
	}

	// Convert to slice
	result := make([]string, 0, len(keywords))
	for keyword := range keywords {
		result = append(result, keyword)
	}

	return result
}
