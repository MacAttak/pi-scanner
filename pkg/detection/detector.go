package detection

import (
	"context"
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/MacAttak/pi-scanner/pkg/validation"
)

// detector implements the Detector interface
type detector struct {
	config     *Config
	matchers   []PatternMatcher
	mu         sync.RWMutex
	compiled   map[string]*regexp.Regexp
	validators *validation.ValidatorRegistry
}

// NewDetector creates a new detector with default configuration
func NewDetector() Detector {
	return NewDetectorWithConfig(DefaultConfig())
}

// NewDetectorWithConfig creates a new detector with custom configuration
func NewDetectorWithConfig(config *Config) Detector {
	d := &detector{
		config:     config,
		matchers:   []PatternMatcher{},
		compiled:   make(map[string]*regexp.Regexp),
		validators: validation.NewValidatorRegistry(),
	}
	
	// Initialize pattern matchers
	d.initializeMatchers()
	
	return d
}

// Name returns the detector name
func (d *detector) Name() string {
	return "pattern-detector"
}

// Detect analyzes content and returns findings
func (d *detector) Detect(ctx context.Context, content []byte, filename string) ([]Finding, error) {
	// Check context cancellation
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	
	// Check file size limit
	if d.config.MaxFileSize > 0 && int64(len(content)) > d.config.MaxFileSize {
		return nil, fmt.Errorf("file too large: %d bytes (max: %d)", len(content), d.config.MaxFileSize)
	}
	
	// Skip excluded paths
	if d.shouldExclude(filename) {
		return nil, nil
	}
	
	findings := []Finding{}
	contentStr := string(content)
	
	// Track which positions have been matched to avoid overlaps
	type matchRange struct {
		start, end int
		piType     PIType
	}
	var matchedRanges []matchRange
	
	// Apply each matcher
	for _, matcher := range d.matchers {
		matches := matcher.Match(content)
		
		for _, match := range matches {
			// Check if this match overlaps with an existing match
			overlaps := false
			for _, existing := range matchedRanges {
				if (match.StartIndex >= existing.start && match.StartIndex < existing.end) ||
				   (match.EndIndex > existing.start && match.EndIndex <= existing.end) {
					overlaps = true
					break
				}
			}
			
			if overlaps {
				continue
			}
			
			// Add to matched ranges
			matchedRanges = append(matchedRanges, matchRange{
				start:  match.StartIndex,
				end:    match.EndIndex,
				piType: matcher.Type(),
			})
			
			// Calculate line and column
			line, column := d.getPosition(contentStr, match.StartIndex)
			
			// Extract context
			contextBefore, contextAfter := d.extractContext(contentStr, match.StartIndex, match.EndIndex)
			
			finding := Finding{
				Type:           matcher.Type(),
				Match:          match.Value,
				File:           filename,
				Line:           line,
				Column:         column,
				Context:        match.Value,
				ContextBefore:  contextBefore,
				ContextAfter:   contextAfter,
				DetectedAt:     time.Now(),
				DetectorName:   d.Name(),
				Confidence:     0.8, // Base confidence for pattern match
				ContextModifier: d.getContextModifier(filename),
			}
			
			// Validate if enabled and validator exists
			if d.config.ValidateChecksums {
				if validator, ok := d.validators.Get(string(finding.Type)); ok {
					valid, err := validator.Validate(finding.Match)
					finding.Validated = valid
					if err != nil {
						finding.ValidationError = err.Error()
					}
					
					// Increase confidence if validated
					if valid {
						finding.Confidence = 0.95
					} else {
						// Decrease confidence if validation fails
						finding.Confidence = 0.5
						if err == nil {
							finding.ValidationError = "Checksum validation failed"
						}
					}
				}
			}
			
			// Set initial risk level based on type
			finding.RiskLevel = d.calculateRiskLevel(finding.Type)
			
			// Apply context validation and confidence-based filtering
			if d.shouldIncludeFinding(ctx, finding, contentStr) {
				findings = append(findings, finding)
			}
		}
	}
	
	// Sort findings by position for consistent ordering
	sort.Slice(findings, func(i, j int) bool {
		if findings[i].Line != findings[j].Line {
			return findings[i].Line < findings[j].Line
		}
		return findings[i].Column < findings[j].Column
	})
	
	return findings, nil
}

// initializeMatchers sets up all pattern matchers
func (d *detector) initializeMatchers() {
	// ABN matcher - 11 digits (check first to avoid TFN confusion)
	d.matchers = append(d.matchers, &regexMatcher{
		pattern: `\b\d{2}[\s]?\d{3}[\s]?\d{3}[\s]?\d{3}\b`,
		piType:  PITypeABN,
		d:       d,
		validator: func(match string) bool {
			// Remove spaces
			clean := strings.ReplaceAll(match, " ", "")
			// Must be exactly 11 digits
			if len(clean) != 11 {
				return false
			}
			// Exclude phone numbers that might look like ABNs
			// International phone: 61 followed by 9 digits starting with 4
			if strings.HasPrefix(clean, "614") {
				return false
			}
			
			// Validate using correct ABN algorithm (mod 89)
			weights := []int{10, 1, 3, 5, 7, 9, 11, 13, 15, 17, 19}
			sum := 0
			// Subtract 1 from first digit and multiply by weight[0]
			firstDigit := int(clean[0] - '0')
			sum += (firstDigit - 1) * weights[0]
			// Add remaining digits with their weights
			for i := 1; i < 11; i++ {
				digit := int(clean[i] - '0')
				sum += digit * weights[i]
			}
			return sum%89 == 0
		},
	})
	
	// Medicare matcher
	d.matchers = append(d.matchers, &regexMatcher{
		pattern: `\b[2-6]\d{3}[\s\-]?\d{5}[\s\-]?\d{1}(?:/\d)?\b`,
		piType:  PITypeMedicare,
		d:       d,
		validator: func(match string) bool {
			// Remove spaces, dashes, and issue number
			clean := regexp.MustCompile(`[\s\-/]`).ReplaceAllString(match, "")
			// Extract first 10 digits (ignore issue number if present)
			if len(clean) < 10 {
				return false
			}
			medicare := clean[:10]
			
			// First digit must be 2-6
			if medicare[0] < '2' || medicare[0] > '6' {
				return false
			}
			
			// Validate using correct Medicare algorithm (mod 10)
			weights := []int{1, 3, 7, 9, 1, 3, 7, 9}
			sum := 0
			for i := 0; i < 8; i++ {
				digit := int(medicare[i] - '0')
				sum += digit * weights[i]
			}
			expectedCheckDigit := sum % 10
			actualCheckDigit := int(medicare[8] - '0')
			return actualCheckDigit == expectedCheckDigit
		},
	})
	
	// TFN matcher - exactly 9 digits (after ABN to avoid confusion)
	d.matchers = append(d.matchers, &regexMatcher{
		pattern: `\b\d{3}[\s\-]?\d{3}[\s\-]?\d{3}\b`,
		piType:  PITypeTFN,
		d:       d,
		validator: func(match string) bool {
			// Remove spaces and dashes
			clean := regexp.MustCompile(`[\s\-]`).ReplaceAllString(match, "")
			// Must be exactly 9 digits and not start with 0
			if len(clean) != 9 || clean[0] == '0' {
				return false
			}
			
			// Validate using correct TFN algorithm (mod 11)
			weights := []int{1, 4, 3, 7, 5, 8, 6, 9, 10}
			sum := 0
			for i, c := range clean {
				digit := int(c - '0')
				sum += digit * weights[i]
			}
			return sum%11 == 0
		},
	})
	
	// BSB matcher - exactly 6 digits with optional hyphen
	d.matchers = append(d.matchers, &regexMatcher{
		pattern: `\b\d{3}[\-]?\d{3}\b`,
		piType:  PITypeBSB,
		d:       d,
		validator: func(match string) bool {
			// Remove dashes and spaces
			clean := regexp.MustCompile(`[\s\-]`).ReplaceAllString(match, "")
			// Must be exactly 6 digits
			if len(clean) != 6 {
				return false
			}
			// Check for valid BSB range (first digit should be 0-7)
			if clean[0] < '0' || clean[0] > '7' {
				return false
			}
			return true
		},
	})
	
	// ACN matcher - exactly 9 digits with ACN context
	// Must check for ACN-specific context to differentiate from TFN
	d.matchers = append(d.matchers, &regexMatcher{
		pattern: `(?i)(?:acn[:\s]*|company\s*acn[:\s]*|australian\s*company\s*number[:\s]*|findByACN\s*\(|// .*acn[:\s]*)\s*["']?\d{3}[\s]?\d{3}[\s]?\d{3}["']?`,
		piType:  PITypeACN,
		d:       d,
		extractor: func(match string) string {
			// Extract just the number part
			numRe := regexp.MustCompile(`\d{3}[\s]?\d{3}[\s]?\d{3}`)
			if num := numRe.FindString(match); num != "" {
				return num
			}
			return ""
		},
		validator: func(match string) bool {
			// Remove spaces
			clean := strings.ReplaceAll(match, " ", "")
			// Must be exactly 9 digits
			if len(clean) != 9 {
				return false
			}
			
			// Validate using correct ACN algorithm (modified mod 10)
			weights := []int{8, 7, 6, 5, 4, 3, 2, 1}
			sum := 0
			for i := 0; i < 8; i++ {
				digit := int(clean[i] - '0')
				sum += digit * weights[i]
			}
			// Calculate expected check digit
			remainder := sum % 10
			expectedCheckDigit := (10 - remainder) % 10
			actualCheckDigit := int(clean[8] - '0')
			return actualCheckDigit == expectedCheckDigit
		},
	})
	
	// Phone matcher (Australian formats) - MUST BE BEFORE driver license to avoid conflicts
	d.matchers = append(d.matchers, &regexMatcher{
		pattern: `(?:\+61[\s.-]?[2-9]\d{8}|\b0[2-9](?:[\s.-]?\d){8}\b|\(\d{2}\)\s*\d{4}\s*\d{4}|\b1[38]00[\s.-]?\d{3}[\s.-]?\d{3}\b)`,
		piType:  PITypePhone,
		d:       d,
		validator: func(match string) bool {
			// Remove all non-digits
			digits := regexp.MustCompile(`[^\d]`).ReplaceAllString(match, "")
			// Check for valid Australian phone formats
			// Mobile: 10 digits starting with 04 or +614
			// Landline: 10 digits starting with 02-09
			// 1300/1800: 10 digits
			if len(digits) == 10 {
				return true
			}
			// International format with country code
			if len(digits) == 11 && strings.HasPrefix(digits, "61") {
				return true
			}
			return false
		},
	})
	
	// Email matcher
	d.matchers = append(d.matchers, &regexMatcher{
		pattern: `\b[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}\b`,
		piType:  PITypeEmail,
		d:       d,
	})
	
	// Driver License matcher - state-specific patterns (AFTER phone to avoid conflicts)
	// NSW/QLD: 8 digits
	// VIC: 8-10 digits  
	// SA: Letter + 6 digits
	// WA: 7 digits
	// TAS: 7 digits or 2 letters + 5 digits
	d.matchers = append(d.matchers, &regexMatcher{
		pattern: `\b(?:[A-Z]\d{6}|[A-Z]{2}\d{5}|\d{7,10})\b`,
		piType:  PITypeDriverLicense,
		d:       d,
		validator: func(match string) bool {
			// Remove all non-digits for checking
			digits := regexp.MustCompile(`[^\d]`).ReplaceAllString(match, "")
			
			// Exclude phone numbers - they start with 0, +61, or 1300/1800
			if match[0] == '0' || strings.HasPrefix(match, "+61") || 
			   strings.HasPrefix(digits, "1300") || strings.HasPrefix(digits, "1800") ||
			   strings.HasPrefix(digits, "04") { // Mobile numbers
				return false
			}
			
			// Check driver license formats
			if len(match) >= 7 && len(match) <= 10 {
				// Numeric formats (NSW/QLD/VIC/WA/TAS)
				if regexp.MustCompile(`^\d+$`).MatchString(match) {
					return true
				}
				// SA format: Letter + 6 digits
				if len(match) == 7 && match[0] >= 'A' && match[0] <= 'Z' {
					return true
				}
				// TAS format: 2 letters + 5 digits
				if len(match) == 7 && match[0] >= 'A' && match[0] <= 'Z' && match[1] >= 'A' && match[1] <= 'Z' {
					return true
				}
			}
			return false
		},
	})
	
	// Name matcher with context-aware filtering for code scanning
	// Only detects names in appropriate contexts (comments, strings, documentation)
	d.matchers = append(d.matchers, &regexMatcher{
		pattern: `\b[A-Z][a-z]{2,}\s+[A-Z][a-z]{2,}(?:\s+[A-Z][a-z]{2,})?\b`,
		piType:  PITypeName,
		d:       d,
		validator: func(match string) bool {
			// Context-aware validation for code scanning
			return d.isValidPersonName(match)
		},
	})
}

// shouldExclude checks if a file should be excluded from scanning
func (d *detector) shouldExclude(filename string) bool {
	for _, pattern := range d.config.ExcludePaths {
		if matched, _ := filepath.Match(pattern, filename); matched {
			return true
		}
		if strings.Contains(filename, pattern) {
			return true
		}
	}
	return false
}

// getContextModifier returns a risk modifier based on file context
func (d *detector) getContextModifier(filename string) float32 {
	// Check if it's a test file
	for _, pattern := range d.config.TestPathPatterns {
		if matched, _ := filepath.Match(pattern, filename); matched {
			return 0.1
		}
		if strings.Contains(filename, strings.Trim(pattern, "*/")) {
			return 0.1
		}
	}
	
	// Check if it's a mock file
	for _, pattern := range d.config.MockPathPatterns {
		if matched, _ := filepath.Match(pattern, filename); matched {
			return 0.1
		}
		if strings.Contains(filename, strings.Trim(pattern, "*/")) {
			return 0.1
		}
	}
	
	return 1.0
}

// calculateRiskLevel determines risk level based on PI type
func (d *detector) calculateRiskLevel(piType PIType) RiskLevel {
	weight := d.config.RiskWeights[piType]
	
	switch {
	case weight >= 90:
		return RiskLevelHigh
	case weight >= 60:
		return RiskLevelMedium
	default:
		return RiskLevelLow
	}
}

// getPosition calculates line and column from byte index
func (d *detector) getPosition(content string, index int) (line, column int) {
	line = 1
	column = 1
	
	for i := 0; i < index && i < len(content); i++ {
		if content[i] == '\n' {
			line++
			column = 1
		} else {
			column++
		}
	}
	
	return line, column
}

// extractContext extracts surrounding context
func (d *detector) extractContext(content string, start, end int) (before, after string) {
	contextSize := 50
	
	// Extract before context
	beforeStart := start - contextSize
	if beforeStart < 0 {
		beforeStart = 0
	}
	before = content[beforeStart:start]
	
	// Extract after context
	afterEnd := end + contextSize
	if afterEnd > len(content) {
		afterEnd = len(content)
	}
	after = content[end:afterEnd]
	
	return before, after
}

// isValidPersonName performs context-aware validation for person names in code
// Returns false for code constructs, technical terms, and non-person names
func (d *detector) isValidPersonName(name string) bool {
	// Convert to lowercase for comparison
	nameLower := strings.ToLower(name)
	
	// Filter out common programming language constructs
	programmingTerms := []string{
		// Java/Scala constructs
		"user service", "data processor", "http client", "rest controller",
		"service impl", "dao impl", "entity manager", "session factory",
		"application context", "bean factory", "proxy factory",
		"model mapper", "object mapper", "json parser", "xml parser",
		"connection pool", "thread pool", "memory pool",
		"cache manager", "queue manager", "file manager",
		"config loader", "property loader", "class loader",
		"event handler", "message handler", "error handler",
		"request processor", "response builder", "query builder",
		"validation service", "security service", "auth service",
		"payment service", "notification service",
		
		// Python constructs
		"user manager", "data handler", "api client", "base model",
		"view controller", "form validator", "signal handler",
		"middleware handler", "context processor", "template loader",
		"database router", "cache backend", "storage backend",
		"task scheduler", "message broker", "event dispatcher",
		
		// Generic technical terms
		"system admin", "database admin", "network admin",
		"super user", "guest user", "admin user", "test user",
		"default config", "base config", "local config",
		"dev environment", "test environment", "prod environment",
		"error message", "success message", "warning message",
		"status code", "response code", "error code",
		"api key", "access token", "refresh token",
		"session id", "request id", "transaction id",
		
		// Brand/Technology names
		"java spring", "react native", "angular material",
		"node express", "django rest", "spring boot",
		"apache kafka", "redis cache", "mongo db",
		"elastic search", "rabbit mq", "amazon aws",
		"google cloud", "microsoft azure", "docker container",
		
		// Common false positives
		"lorem ipsum", "foo bar", "hello world",
		"test data", "sample data", "mock data",
		"dummy data", "fake data", "example data",
	}
	
	// Check against programming terms
	for _, term := range programmingTerms {
		if strings.Contains(nameLower, term) {
			return false
		}
	}
	
	// Filter out obvious code patterns
	codePatterns := []string{
		"service", "manager", "handler", "processor", "controller",
		"factory", "builder", "parser", "loader", "validator",
		"client", "server", "proxy", "adapter", "wrapper",
		"helper", "utility", "config", "settings", "options",
		"context", "session", "request", "response", "model",
		"entity", "repository", "dao", "dto", "vo",
		"impl", "abstract", "base", "default", "custom",
		"exception", "error", "warning", "info", "debug",
		"test", "mock", "stub", "fake", "dummy",
	}
	
	for _, pattern := range codePatterns {
		if strings.Contains(nameLower, pattern) {
			return false
		}
	}
	
	// Filter out single character names or very short names
	parts := strings.Fields(name)
	for _, part := range parts {
		if len(part) < 3 {
			return false
		}
	}
	
	// Filter out names with numbers or special characters
	for _, char := range name {
		if char >= '0' && char <= '9' {
			return false
		}
		if char != ' ' && !((char >= 'A' && char <= 'Z') || (char >= 'a' && char <= 'z')) {
			return false
		}
	}
	
	// Additional validation: check if it looks like a real name
	// Real names typically don't have all caps or unusual patterns
	allCaps := true
	for _, char := range name {
		if char >= 'a' && char <= 'z' {
			allCaps = false
			break
		}
	}
	if allCaps {
		return false
	}
	
	// Passed all filters - likely a real person name
	return true
}

// shouldIncludeFinding determines if a finding should be included based on context validation and confidence thresholds
func (d *detector) shouldIncludeFinding(ctx context.Context, finding Finding, fileContent string) bool {
	// Apply context modifier to confidence
	adjustedConfidence := finding.Confidence * finding.ContextModifier
	
	// Set minimum confidence threshold based on context
	minConfidence := d.getMinimumConfidenceThreshold(finding)
	
	// If adjusted confidence is below threshold, exclude the finding
	if adjustedConfidence < minConfidence {
		return false
	}
	
	// Apply advanced context validation if enabled
	if d.config.EnableContextValidation {
		isValid := d.validateContext(finding, fileContent)
		if !isValid {
			return false
		}
	}
	
	// Default behavior: include if adjusted confidence meets threshold
	return adjustedConfidence >= minConfidence
}

// getMinimumConfidenceThreshold returns the minimum confidence threshold for a finding
func (d *detector) getMinimumConfidenceThreshold(finding Finding) float32 {
	// Check if it's a test file context (should have higher threshold)
	if finding.ContextModifier <= 0.1 {
		return 0.9 // Very high threshold for test contexts (effectively filters most out)
	}
	
	// Check if it's a mock file context
	if finding.ContextModifier <= 0.2 {
		return 0.8 // High threshold for mock contexts
	}
	
	// Different thresholds based on PI type criticality
	switch finding.Type {
	case PITypeTFN, PITypeMedicare, PITypeCreditCard:
		return 0.4 // Lower threshold for high-risk PI types (don't want to miss them)
	case PITypeABN, PITypeACN, PITypeBSB:
		return 0.4 // Medium-risk PI types
	case PITypeName, PITypeEmail, PITypePhone:
		return 0.3 // Lower-risk PI types can have lower threshold
	default:
		return 0.4 // Default threshold
	}
}

// validateContext performs simplified context validation
func (d *detector) validateContext(finding Finding, fileContent string) bool {
	// Check if finding is in test data context
	if d.isInTestContext(finding, fileContent) {
		return false // Suppress findings in test contexts
	}
	
	// Check if finding is in comment and looks like example data
	if d.isInCommentExample(finding, fileContent) {
		return false // Suppress obvious examples in comments
	}
	
	// Check if finding looks like mock/dummy data
	if d.isInMockContext(finding, fileContent) {
		return false // Suppress mock data
	}
	
	return true // Include by default
}

// isInTestContext checks if the finding is in a test-related context
func (d *detector) isInTestContext(finding Finding, content string) bool {
	// Extract context around the finding
	context := d.extractLineContext(content, finding.Line, 3)
	contextLower := strings.ToLower(context)
	
	// Test framework keywords
	testKeywords := []string{
		"test", "spec", "describe", "it(", "expect", "assert", "should",
		"beforeeach", "aftereach", "setup", "teardown", "given", "when", "then",
		"@test", "@parameterizedtest", "@valueSource", "unittest", "pytest",
		"scalatest", "wordspec", "funspec", "junit", "testng", "mockito",
	}
	
	for _, keyword := range testKeywords {
		if strings.Contains(contextLower, keyword) {
			return true
		}
	}
	
	return false
}

// isInCommentExample checks if finding is in a comment that appears to be an example
func (d *detector) isInCommentExample(finding Finding, content string) bool {
	line := d.getLineContent(content, finding.Line)
	
	// Check if line contains comment markers
	if strings.Contains(line, "//") || strings.Contains(line, "#") || 
	   strings.Contains(line, "/*") || strings.Contains(line, "*/") {
		
		lineLower := strings.ToLower(line)
		exampleKeywords := []string{
			"example", "sample", "todo", "fixme", "note:", "warning:",
			"replace", "change", "update", "placeholder", "format:",
		}
		
		for _, keyword := range exampleKeywords {
			if strings.Contains(lineLower, keyword) {
				return true
			}
		}
	}
	
	return false
}

// isInMockContext checks if finding appears to be mock or dummy data
func (d *detector) isInMockContext(finding Finding, content string) bool {
	context := d.extractLineContext(content, finding.Line, 2)
	contextLower := strings.ToLower(context)
	
	mockKeywords := []string{
		"mock", "fake", "stub", "dummy", "placeholder", 
		"lorem", "ipsum", "demo", "sample", "template", 
		"test_", "mock_", "dummy_", "fake_", "example_",
		"testdata", "testfactory", "mockdata",
	}
	
	for _, keyword := range mockKeywords {
		if strings.Contains(contextLower, keyword) {
			return true
		}
	}
	
	return false
}

// extractLineContext extracts context lines around a specific line number
func (d *detector) extractLineContext(content string, lineNum int, contextLines int) string {
	lines := strings.Split(content, "\n")
	if lineNum <= 0 || lineNum > len(lines) {
		return ""
	}
	
	start := lineNum - contextLines - 1
	if start < 0 {
		start = 0
	}
	
	end := lineNum + contextLines - 1
	if end >= len(lines) {
		end = len(lines) - 1
	}
	
	contextSlice := lines[start:end+1]
	return strings.Join(contextSlice, "\n")
}

// getLineContent returns the content of a specific line
func (d *detector) getLineContent(content string, lineNum int) string {
	lines := strings.Split(content, "\n")
	if lineNum <= 0 || lineNum > len(lines) {
		return ""
	}
	return lines[lineNum-1]
}

// getRegexp returns a compiled regex, using cache
func (d *detector) getRegexp(pattern string) (*regexp.Regexp, error) {
	d.mu.RLock()
	if re, ok := d.compiled[pattern]; ok {
		d.mu.RUnlock()
		return re, nil
	}
	d.mu.RUnlock()
	
	d.mu.Lock()
	defer d.mu.Unlock()
	
	// Double-check after acquiring write lock
	if re, ok := d.compiled[pattern]; ok {
		return re, nil
	}
	
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, err
	}
	
	d.compiled[pattern] = re
	return re, nil
}

// regexMatcher implements PatternMatcher using regex
type regexMatcher struct {
	pattern   string
	piType    PIType
	d         *detector
	validator func(string) bool
	extractor func(string) string // Optional function to extract the actual value from the match
}

// Match finds all pattern matches in content
func (m *regexMatcher) Match(content []byte) []PatternMatch {
	re, err := m.d.getRegexp(m.pattern)
	if err != nil {
		return nil
	}
	
	var matches []PatternMatch
	allMatches := re.FindAllIndex(content, -1)
	
	for _, match := range allMatches {
		if len(match) >= 2 {
			value := string(content[match[0]:match[1]])
			
			// Apply validator if present
			// Apply extractor if present
			extractedValue := value
			if m.extractor != nil {
				extractedValue = m.extractor(value)
				if extractedValue == "" {
					continue
				}
			}
			
			// Apply validator if present
			if m.validator != nil && !m.validator(extractedValue) {
				continue
			}
			
			matches = append(matches, PatternMatch{
				Value:      extractedValue,
				StartIndex: match[0],
				EndIndex:   match[1],
			})
		}
	}
	
	return matches
}

// Type returns the PI type this matcher detects
func (m *regexMatcher) Type() PIType {
	return m.piType
}