package detection

import (
	"context"
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/MacAttak/pi-scanner/pkg/validation"
	"github.com/nyaruka/phonenumbers"
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

	// TWO-PHASE ARCHITECTURE DESIGN:
	// This detector implements Phase 1 of a two-phase PI detection system:
	// Phase 1 (Pattern Detection): Cast a wide net to catch all potential PI
	// Phase 2 (LLM Validation): Use AI to disambiguate true PI from false positives
	//
	// Key principles:
	// 1. We allow overlapping matches (e.g., "ID: 123456782" matches both TFN and generic patterns)
	// 2. We minimize false positive suppression (only suppress very obvious synthetic data)
	// 3. Context is calculated but not used for suppression - it's passed to the LLM
	// 4. Multiple patterns matching the same text is expected and desired for LLM context

	// Apply each matcher
	for _, matcher := range d.matchers {
		matches := matcher.Match(content)

		// Debug logging
		// if len(matches) > 0 && matcher.Type() == PITypeTFN {
		//     fmt.Printf("DEBUG: TFN matcher found %d matches\n", len(matches))
		//     for _, m := range matches {
		//         fmt.Printf("  Match: %s, Valid: %v\n", m.Value, m.ValidationPassed)
		//     }
		// }

		for _, match := range matches {

			// Calculate line and column
			line, column := d.getPosition(contentStr, match.StartIndex)

			// Extract context
			contextBefore, contextAfter := d.extractContext(contentStr, match.StartIndex, match.EndIndex)

			// Set base confidence based on pattern validation
			baseConfidence := float32(0.8)
			if !match.ValidationPassed {
				baseConfidence = 0.5 // Lower confidence for patterns that fail validation
			}

			finding := Finding{
				Type:            matcher.Type(),
				Match:           match.Value,
				File:            filename,
				Line:            line,
				Column:          column,
				Context:         match.Value,
				ContextBefore:   contextBefore,
				ContextAfter:    contextAfter,
				DetectedAt:      time.Now(),
				DetectorName:    d.Name(),
				Confidence:      baseConfidence,
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
			baseRisk := d.calculateRiskLevel(finding.Type)
			finding.RiskLevel = baseRisk

			// Adjust risk level based on context
			adjustedRisk := d.adjustRiskLevelForContext(baseRisk, filename, contentStr, finding.Line)
			finding.RiskLevel = adjustedRisk

			// Debug logging for TFN
			// if finding.Type == PITypeTFN {
			//     weight := d.config.RiskWeights[PITypeTFN]
			//     fmt.Printf("DEBUG TFN: weight=%d, baseRisk=%s, adjustedRisk=%s, file=%s\n",
			//         weight, baseRisk, adjustedRisk, filename)
			// }

			// if strings.Contains(filename, "PaymentService") {
			//	fmt.Printf("DEBUG: After assignment - finding.Type=%s, baseRisk=%s, adjustedRisk=%s, finding.RiskLevel=%s\n",
			//		finding.Type, baseRisk, adjustedRisk, finding.RiskLevel)
			// }

			// Apply context validation and confidence-based filtering
			include := d.shouldIncludeFinding(ctx, finding, contentStr)
			// if finding.Type == PITypeTFN {
			//     fmt.Printf("DEBUG: TFN shouldInclude = %v, confidence = %f\n", include, finding.Confidence)
			// }
			if include {
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
// Order matters: more specific patterns should come before generic ones
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

			// For pattern matching, we accept any 11-digit number that looks like an ABN
			// Checksum validation will happen later via the validation registry
			return true
		},
	})

	// Medicare matcher with enhanced validation
	d.matchers = append(d.matchers, &regexMatcher{
		pattern: `\b[2-6]\d{3}[\s\-]?\d{5}[\s\-]?\d{1}(?:/\d)?\b`,
		piType:  PITypeMedicare,
		d:       d,
		validator: func(match string) bool {
			return d.isValidAustralianMedicare(match)
		},
	})

	// TFN matcher - exactly 9 digits with proper checksum validation
	d.matchers = append(d.matchers, &regexMatcher{
		pattern: `\b\d{3}[\s\-]?\d{3}[\s\-]?\d{3}\b`,
		piType:  PITypeTFN,
		d:       d,
		validator: func(match string) bool {
			return d.isValidAustralianTFN(match)
		},
	})

	// BSB matcher - enhanced validation with bank code ranges
	// Requires BSB context to avoid false positives from random 6-digit numbers
	d.matchers = append(d.matchers, &regexMatcher{
		pattern: `(?i)(?:bsb|bank\s*state\s*branch|bank_bsb)[\s:#=]*["']?\d{3}[\-\s]?\d{3}["']?\b`,
		piType:  PITypeBSB,
		d:       d,
		extractor: func(match string) string {
			// Extract just the BSB number
			bsbRe := regexp.MustCompile(`\d{3}[\-\s]?\d{3}`)
			if bsb := bsbRe.FindString(match); bsb != "" {
				return bsb
			}
			return match
		},
		validator: func(match string) bool {
			return d.isValidAustralianBSB(match)
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
			// For pattern matching, we accept any 9-digit number that looks like an ACN
			// Checksum validation will happen later via the validation registry
			return len(clean) == 9
		},
	})

	// Credit Card matcher - supports major card types with Luhn validation
	d.matchers = append(d.matchers, &regexMatcher{
		pattern: `\b(?:4[0-9]{3}[\s\-]?[0-9]{4}[\s\-]?[0-9]{4}[\s\-]?[0-9]{4}|5[1-5][0-9]{2}[\s\-]?[0-9]{4}[\s\-]?[0-9]{4}[\s\-]?[0-9]{4}|3[47][0-9]{2}[\s\-]?[0-9]{6}[\s\-]?[0-9]{5}|3(?:0[0-5]|[68][0-9])[0-9][\s\-]?[0-9]{6}[\s\-]?[0-9]{4}|6(?:011|5[0-9]{2})[\s\-]?[0-9]{4}[\s\-]?[0-9]{4}[\s\-]?[0-9]{4}|(?:2131|1800|35\d{3})[\s\-]?\d{4}[\s\-]?\d{4}[\s\-]?\d{3})\b`,
		piType:  PITypeCreditCard,
		d:       d,
		validator: func(match string) bool {
			return d.isValidCreditCard(match)
		},
	})

	// Phone matcher (Australian formats) - Enhanced with libphonenumber validation
	d.matchers = append(d.matchers, &regexMatcher{
		pattern: `(?:\+61[\s.-]?[0-9](?:[\s.-]?[0-9]){8,9}|\b0[2-9](?:[\s.-]?[0-9]){8}\b|\([0-9]{2}\)\s*[0-9]{4}\s*[0-9]{4}|\b1[38]00[\s.-]?[0-9]{3}[\s.-]?[0-9]{3}\b|\+61[\s.-]?4[0-9](?:[\s.-]?[0-9]){7})`,
		piType:  PITypePhone,
		d:       d,
		validator: func(match string) bool {
			return d.isValidAustralianPhone(match)
		},
	})

	// Email matcher
	d.matchers = append(d.matchers, &regexMatcher{
		pattern: `\b[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}\b`,
		piType:  PITypeEmail,
		d:       d,
	})

	// Driver License matcher - moved before name matcher to ensure proper priority
	// Enhanced state-specific validation to reduce false positives
	// More specific patterns to avoid matching other PI types
	// Only match alphanumeric patterns or very specific numeric patterns with context
	d.matchers = append(d.matchers, &regexMatcher{
		pattern: `\b(?:[A-Z]\d{6,8}|[A-Z]{2}\d{5}|S\d{6}|(?i)(?:driver[\s\-]*)?(?:license|licence|driv\.?lic\.?|dl)[\s:#-]*(?:number[\s:#-]*)?\d{7,9}|(?i)(?:id|ref)[\s:#-]*[A-Z]\d{5,8}|(?i)(?:id|ref)[\s:#-]*[A-Z]{2}\d{5})\b`,
		piType:  PITypeDriverLicense,
		d:       d,
		extractor: func(match string) string {
			// If it's a pure alphanumeric pattern, return as is
			if regexp.MustCompile(`^[A-Z]\d{6,8}$|^[A-Z]{2}\d{5}$|^S\d{6}$`).MatchString(match) {
				return match
			}

			// Extract alphanumeric license patterns after context words
			alphaNumRe := regexp.MustCompile(`[A-Z]\d{5,8}|[A-Z]{2}\d{5}|S\d{6}`)
			if alphaNum := alphaNumRe.FindString(match); alphaNum != "" {
				return alphaNum
			}

			// Otherwise extract the numeric part after the context
			numRe := regexp.MustCompile(`\d{7,9}`)
			if num := numRe.FindString(match); num != "" {
				// Preserve the original match for validation but return the number
				return num
			}
			return match
		},
		validator: func(match string) bool {
			// For pure alphanumeric patterns, validate directly
			if regexp.MustCompile(`^[A-Z]\d{6,8}$|^[A-Z]{2}\d{5}$|^S\d{6}$`).MatchString(match) {
				return d.isValidAustralianDriverLicense(match)
			}
			// For numeric patterns that were matched with context, they're already validated by the regex
			// Just check basic validity
			if regexp.MustCompile(`^\d{7,9}$`).MatchString(match) {
				// Skip test patterns
				isTest := isTestDriverLicense(match)
				// fmt.Printf("DEBUG DL: match=%s, isTest=%v, returning=%v\n", match, isTest, !isTest)
				return !isTest
			}
			return d.isValidAustralianDriverLicense(match)
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

	// Australian Address matcher
	// Matches: Unit/Number Street Name, Suburb STATE Postcode
	// Examples: "123 Queen Street, Melbourne VIC 3000", "Unit 4/56 Kings Road, Sydney NSW 2000"
	d.matchers = append(d.matchers, &regexMatcher{
		pattern: `(?i)\b(?:unit\s*\d+[/-]?)?\d{1,5}\s+[A-Z][a-z]+(?:\s+[A-Z][a-z]+)*\s+(?:Street|St|Road|Rd|Avenue|Ave|Drive|Dr|Lane|Ln|Place|Pl|Court|Ct|Crescent|Cres|Parade|Pde|Boulevard|Blvd|Highway|Hwy|Terrace|Tce|Way|Circuit|Cct)\b(?:\s*,\s*[A-Z][a-z]+(?:\s+[A-Z][a-z]+)*\s+(?:NSW|VIC|QLD|SA|WA|TAS|NT|ACT)\s+\d{4}\b)?`,
		piType:  PITypeAddress,
		d:       d,
		validator: func(match string) bool {
			// Validate Australian address format
			return d.isValidAustralianAddress(match)
		},
	})

	// Australian Passport matcher
	// Format: Letter followed by 7 digits (e.g., A1234567, M9876543)
	d.matchers = append(d.matchers, &regexMatcher{
		pattern: `\b[A-Z]\d{7}\b`,
		piType:  PITypePassport,
		d:       d,
		validator: func(match string) bool {
			// Australian passports start with specific letters
			validPrefixes := []byte{'A', 'B', 'C', 'D', 'E', 'F', 'G', 'H', 'J', 'K', 'L', 'M', 'N', 'P', 'R', 'S', 'T', 'U', 'V', 'W', 'X', 'Y', 'Z'}
			firstChar := match[0]
			for _, prefix := range validPrefixes {
				if firstChar == prefix {
					return true
				}
			}
			return false
		},
	})

	// Combined BSB + Account matcher (must come before individual matchers)
	// Matches BSB followed by account number with account context
	d.matchers = append(d.matchers, &regexMatcher{
		pattern: `(?i)(?:account|acc|acct)[\s:#\-]*\d{3}[\-\s]?\d{3}[\s]+\d{6,10}\b`,
		piType:  PITypeBankAccount,
		d:       d,
		extractor: func(match string) string {
			// Extract just the numbers part
			numRe := regexp.MustCompile(`\d{3}[\-\s]?\d{3}[\s]+\d{6,10}`)
			if nums := numRe.FindString(match); nums != "" {
				return nums
			}
			return match
		},
		validator: func(match string) bool {
			// Extract numeric part
			numRe := regexp.MustCompile(`(\d{3}[\-\s]?\d{3})[\s]+(\d{6,10})`)
			matches := numRe.FindStringSubmatch(match)
			if len(matches) != 3 {
				return false
			}

			// Validate BSB part
			bsb := strings.ReplaceAll(matches[1], "-", "")
			bsb = strings.ReplaceAll(bsb, " ", "")
			if len(bsb) != 6 {
				return false
			}

			// Validate account part
			account := matches[2]
			if len(account) < 6 || len(account) > 10 {
				return false
			}

			return true
		},
	})

	// Bank Account matcher
	// Matches account numbers (6-10 digits) with appropriate context
	d.matchers = append(d.matchers, &regexMatcher{
		pattern: `(?i)\b(?:(?:account|acct|acc)[\s\-]*(?:number|no\.?)?[\s:#\-=]*|bank[\s\-]*account[\s:#\-=]*)\d{6,10}\b`,
		piType:  PITypeBankAccount,
		d:       d,
		extractor: func(match string) string {
			// Extract just the numeric part
			numRe := regexp.MustCompile(`\d{6,10}`)
			if num := numRe.FindString(match); num != "" {
				return num
			}
			return match
		},
		validator: func(match string) bool {
			// Basic validation for account numbers
			clean := strings.ReplaceAll(match, " ", "")
			clean = strings.ReplaceAll(clean, "-", "")

			// Must be 6-10 digits
			if len(clean) < 6 || len(clean) > 10 {
				return false
			}

			// Must be all digits
			for _, ch := range clean {
				if ch < '0' || ch > '9' {
					return false
				}
			}

			// Skip obvious test patterns
			if clean == "12345678" || clean == "123456789" || clean == "1234567890" {
				return false
			}

			return true
		},
	})

	// Generic bank account matcher for assignment contexts
	// Matches variable assignments containing account-like numbers
	d.matchers = append(d.matchers, &regexMatcher{
		pattern: `(?i)(?:TEST_)?ACCOUNT\s*[=:]\s*["']?\d{6,10}["']?`,
		piType:  PITypeBankAccount,
		d:       d,
		extractor: func(match string) string {
			// Extract just the numeric part
			numRe := regexp.MustCompile(`\d{6,10}`)
			if num := numRe.FindString(match); num != "" {
				return num
			}
			return match
		},
		validator: func(match string) bool {
			// Extract numeric part
			numRe := regexp.MustCompile(`\d{6,10}`)
			num := numRe.FindString(match)
			if num == "" {
				return false
			}

			// Skip synthetic patterns
			if d.isSyntheticPattern(num) {
				return false
			}

			return true
		},
	})

	// SWIFT/BIC Code matcher
	// Format: 8 or 11 alphanumeric characters (e.g., ANZBNZ22, ANZBNZ22MEL)
	d.matchers = append(d.matchers, &regexMatcher{
		pattern: `\b[A-Z]{6}[A-Z0-9]{2}(?:[A-Z0-9]{3})?\b`,
		piType:  PIType("SWIFT_BIC"),
		d:       d,
		validator: func(match string) bool {
			// SWIFT codes are 8 or 11 characters
			if len(match) != 8 && len(match) != 11 {
				return false
			}

			// First 6 characters must be letters (bank code + country)
			for i := 0; i < 6; i++ {
				if match[i] < 'A' || match[i] > 'Z' {
					return false
				}
			}

			// Characters 5-6 should be a valid country code
			// For now, accept any 2-letter combination

			return true
		},
	})

	// IBAN matcher
	// International Bank Account Number - various country formats
	d.matchers = append(d.matchers, &regexMatcher{
		pattern: `\b[A-Z]{2}\d{2}(?:[\s]?[A-Z0-9]{1,4})+\b`,
		piType:  PIType("IBAN"),
		d:       d,
		extractor: func(match string) string {
			// Return the original match with spaces
			return match
		},
		validator: func(match string) bool {
			// Remove spaces
			clean := strings.ReplaceAll(match, " ", "")

			// IBAN length varies by country (15-34 characters)
			if len(clean) < 15 || len(clean) > 34 {
				return false
			}

			// First 2 chars must be country code (letters)
			if clean[0] < 'A' || clean[0] > 'Z' || clean[1] < 'A' || clean[1] > 'Z' {
				return false
			}

			// Next 2 must be check digits
			if clean[2] < '0' || clean[2] > '9' || clean[3] < '0' || clean[3] > '9' {
				return false
			}

			// Country-specific validation (simplified)
			countryCode := clean[0:2]
			expectedLength := map[string]int{
				"AD": 24, "AE": 23, "AT": 20, "AZ": 28, "BA": 20, "BE": 16,
				"BG": 22, "BH": 22, "BR": 29, "CH": 21, "CR": 22, "CY": 28,
				"CZ": 24, "DE": 22, "DK": 18, "DO": 28, "EE": 20, "ES": 24,
				"FI": 18, "FO": 18, "FR": 27, "GB": 22, "GE": 22, "GI": 23,
				"GL": 18, "GR": 27, "GT": 28, "HR": 21, "HU": 28, "IE": 22,
				"IL": 23, "IS": 26, "IT": 27, "JO": 30, "KW": 30, "KZ": 20,
				"LB": 28, "LI": 21, "LT": 20, "LU": 20, "LV": 21, "MC": 27,
				"MD": 24, "ME": 22, "MK": 19, "MR": 27, "MT": 31, "MU": 30,
				"NL": 18, "NO": 15, "PK": 24, "PL": 28, "PS": 29, "PT": 25,
				"QA": 29, "RO": 24, "RS": 22, "SA": 24, "SE": 24, "SI": 19,
				"SK": 24, "SM": 27, "TN": 24, "TR": 26, "XK": 20,
			}

			if expLen, ok := expectedLength[countryCode]; ok {
				return len(clean) == expLen
			}

			// Unknown country code - accept if within valid range
			return true
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
	filenameLower := strings.ToLower(filename)

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

	// Check if it's a documentation file
	if strings.HasSuffix(filenameLower, ".md") ||
		strings.Contains(filenameLower, "/docs/") ||
		strings.Contains(filenameLower, "/doc/") ||
		strings.Contains(filenameLower, "/examples/") ||
		strings.Contains(filenameLower, "/example/") ||
		strings.HasPrefix(filenameLower, "docs/") ||
		strings.HasPrefix(filenameLower, "examples/") ||
		strings.HasPrefix(filenameLower, "readme") ||
		strings.HasPrefix(filenameLower, "contributing") {
		return 0.3
	}

	return 1.0
}

// calculateRiskLevel determines risk level based on PI type
func (d *detector) calculateRiskLevel(piType PIType) RiskLevel {
	weight, exists := d.config.RiskWeights[piType]
	if !exists {
		// Default weight if not found
		weight = 50
	}

	// Debug: Log BSB weight
	// if piType == PITypeBSB {
	//	// For debugging only - uncomment if needed
	//	// fmt.Printf("DEBUG: BSB piType='%s', weight=%d, exists=%v\n", piType, weight, exists)
	// }

	switch {
	case weight >= 90:
		return RiskLevelHigh
	case weight >= 70:
		return RiskLevelMedium
	default:
		return RiskLevelLow
	}
}

// adjustRiskLevelForContext adjusts risk level based on file path, content context, and proximity
func (d *detector) adjustRiskLevelForContext(baseRisk RiskLevel, filename, content string, line int) RiskLevel {
	// Get context modifier for file path
	contextModifier := d.getContextModifier(filename)

	// Check for multiple PI types in proximity (within 5 lines)
	proximityRisk := d.assessProximityRisk(content, line)

	// Check content context for risk indicators
	contentContext := d.assessContentContext(content, line)

	// Start with base risk
	adjustedRisk := baseRisk

	// Debug output for production files
	isProdFile := strings.Contains(strings.ToLower(filename), "prod") ||
		strings.Contains(strings.ToLower(filename), "production") ||
		(strings.Contains(filename, "/main/") && !strings.Contains(filename, "/test/"))
	isConfigFile := strings.Contains(strings.ToLower(filename), "config") ||
		strings.Contains(strings.ToLower(filename), ".env") ||
		strings.Contains(strings.ToLower(filename), "secret") ||
		strings.Contains(strings.ToLower(filename), ".properties")

	// For debugging - uncomment if needed
	// if strings.Contains(filename, "PaymentService") {
	//	fmt.Printf("DEBUG: filename=%s, isProdFile=%v, isConfigFile=%v, contextModifier=%f, baseRisk=%s\n",
	//	    filename, isProdFile, isConfigFile, contextModifier, baseRisk)
	// }

	// Context-based adjustments
	switch {
	case contextModifier <= 0.1: // Test files
		// Lower risk for test files, but not too much
		if adjustedRisk == RiskLevelHigh {
			adjustedRisk = RiskLevelMedium
		} else if adjustedRisk == RiskLevelMedium {
			adjustedRisk = RiskLevelLow
		}
		// RiskLevelLow stays low

	case contextModifier <= 0.3: // Example/doc files
		// Moderate risk for documentation
		if adjustedRisk == RiskLevelHigh {
			adjustedRisk = RiskLevelMedium
		}
		// Medium stays medium, low stays low

	case isConfigFile:
		// Increase risk for config/environment files
		if adjustedRisk == RiskLevelLow {
			adjustedRisk = RiskLevelMedium
		} else if adjustedRisk == RiskLevelMedium {
			adjustedRisk = RiskLevelHigh
		}
		// High stays high

	case isProdFile:
		// Increase risk for production files
		// src/main/java is production code (not test)
		// if strings.Contains(filename, "PaymentService") {
		//	fmt.Printf("DEBUG: In isProdFile case, adjustedRisk before=%s\n", adjustedRisk)
		// }
		if adjustedRisk == RiskLevelLow {
			adjustedRisk = RiskLevelHigh
		} else if adjustedRisk == RiskLevelMedium {
			adjustedRisk = RiskLevelHigh
		} else if adjustedRisk == RiskLevelHigh {
			adjustedRisk = RiskLevelCritical
		}
		// if strings.Contains(filename, "PaymentService") {
		//	fmt.Printf("DEBUG: In isProdFile case, adjustedRisk after=%s\n", adjustedRisk)
		// }
	}

	// Content context adjustments
	switch contentContext {
	case "database":
		// Database operations with PI are critical risk
		adjustedRisk = RiskLevelCritical
	case "api_response":
		// API responses exposing PI are critical
		adjustedRisk = RiskLevelCritical
	case "comment":
		// Comments with PI in production code should maintain their risk level
		// Only downgrade if it's already been identified as test/example file
		if contextModifier <= 0.3 && adjustedRisk.Compare(RiskLevelMedium) > 0 {
			adjustedRisk = RiskLevelMedium
		}
	case "test_data":
		// Test data patterns are lower risk
		if adjustedRisk.Compare(RiskLevelLow) > 0 {
			adjustedRisk = RiskLevelLow
		}
	}

	// Proximity risk adjustments
	if proximityRisk && adjustedRisk.Compare(RiskLevelHigh) < 0 {
		// Multiple PI in proximity increases risk
		if adjustedRisk == RiskLevelLow {
			adjustedRisk = RiskLevelMedium
		} else if adjustedRisk == RiskLevelMedium {
			adjustedRisk = RiskLevelHigh
		}
	}

	// Special case: If this is a HIGH risk PI (like TFN) and we detected proximity risk,
	// maintain the HIGH risk level
	if baseRisk == RiskLevelHigh && proximityRisk {
		if adjustedRisk.Compare(RiskLevelHigh) < 0 {
			adjustedRisk = RiskLevelHigh
		}
	}

	return adjustedRisk
}

// assessProximityRisk checks for multiple PI types within proximity
func (d *detector) assessProximityRisk(content string, centerLine int) bool {
	lines := strings.Split(content, "\n")
	if centerLine <= 0 || centerLine > len(lines) {
		return false
	}

	// Check 5 lines before and after
	start := max(0, centerLine-6)
	end := min(len(lines), centerLine+5)

	proximityContent := strings.Join(lines[start:end], "\n")
	proximityContentLower := strings.ToLower(proximityContent)

	// Count different PI and PII indicators
	piIndicators := 0

	// PI indicators
	piTypes := []string{"tfn", "abn", "medicare", "bsb", "account", "credit card", "driver license", "passport"}
	for _, indicator := range piTypes {
		if strings.Contains(proximityContentLower, indicator) {
			piIndicators++
		}
	}

	// PII indicators that increase risk when combined with PI
	piiTypes := []string{"name", "email", "phone", "address", "date of birth", "dob"}
	piiCount := 0
	for _, indicator := range piiTypes {
		if strings.Contains(proximityContentLower, indicator) {
			piiCount++
		}
	}

	// If we have PII with PI, that increases the risk
	if piIndicators > 0 && piiCount > 0 {
		piIndicators += piiCount
	}

	// Multiple PI types in proximity indicate higher risk
	return piIndicators >= 3
}

// assessContentContext analyzes the content context around the finding
func (d *detector) assessContentContext(content string, line int) string {
	// Extract 3 lines before and after
	context := d.extractLineContext(content, line, 3)
	contextLower := strings.ToLower(context)

	// Database context
	if strings.Contains(contextLower, "insert into") ||
		strings.Contains(contextLower, "update") ||
		strings.Contains(contextLower, "select") ||
		strings.Contains(contextLower, "create table") {
		return "database"
	}

	// API response context
	if strings.Contains(contextLower, "return") &&
		(strings.Contains(contextLower, "{") || strings.Contains(contextLower, "response")) {
		return "api_response"
	}

	// Comment context
	if strings.Contains(context, "//") || strings.Contains(context, "/*") || strings.Contains(context, "#") {
		return "comment"
	}

	// Test data patterns
	if strings.Contains(contextLower, "test") ||
		strings.Contains(contextLower, "mock") ||
		strings.Contains(contextLower, "example") ||
		strings.Contains(contextLower, "dummy") {
		return "test_data"
	}

	// Check for synthetic data patterns (repeated/sequential)
	// Extract just the finding value from the line
	if d.looksLikeSyntheticData(context) {
		return "test_data"
	}

	return "normal"
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// max returns the maximum of two integers
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
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

	// Debug: uncomment to trace name validation
	// fmt.Printf("isValidPersonName: checking '%s' (lower: '%s')\n", name, nameLower)

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

		// License/ID related terms that should be handled by other matchers
		"driver license", "driver licence", "driving license", "driving licence",
		"license number", "licence number", "license id", "licence id",
	}

	// Check against programming terms
	for _, term := range programmingTerms {
		if nameLower == term || strings.Contains(nameLower, term) {
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
	// Passed all filters - likely a real person name
	return !allCaps
}

// shouldIncludeFinding determines if a finding should be included based on context validation and confidence thresholds
func (d *detector) shouldIncludeFinding(ctx context.Context, finding Finding, fileContent string) bool {
	// For test files (low context modifier), we still want to detect PI
	// but we'll rely more on context validation rather than confidence thresholds

	// Apply advanced context validation if enabled
	if d.config.EnableContextValidation {
		isValid := d.validateContext(finding, fileContent)
		if !isValid {
			return false
		}
	}

	// Check base confidence threshold (without context modifier)
	// This ensures we don't filter out legitimate findings just because they're in test files
	minConfidence := d.getMinimumConfidenceThreshold(finding)
	// Include the finding
	return finding.Confidence >= minConfidence
}

// getMinimumConfidenceThreshold returns the minimum confidence threshold for a finding
func (d *detector) getMinimumConfidenceThreshold(finding Finding) float32 {
	// Use the configured minimum threshold if set
	if d.config.MinConfidenceThreshold > 0 {
		return d.config.MinConfidenceThreshold
	}

	// Default thresholds based on context
	// Note: We don't want to filter out findings in test files too aggressively
	// as they might be testing real PI detection
	return 0.3 // Default threshold allows most findings through
}

// validateContext performs simplified context validation
func (d *detector) validateContext(finding Finding, fileContent string) bool {
	// DEBUG
	// if finding.Type == PITypeTFN {
	//     fmt.Printf("DEBUG validateContext: TFN in file %s\n", finding.File)
	// }
	// For test files, we're less strict about context validation
	// since tests often contain real PI examples for testing purposes
	isTestFile := finding.ContextModifier <= 0.1

	// Skip test context validation for test files entirely
	// Test files often contain real PI for testing purposes
	if isTestFile {
		// Don't apply test context suppression to files already identified as test files
		// Continue with other validations
	} else {
		// Check if finding is in test data context
		// Don't suppress for documentation files (README, docs, etc)
		isDocFile := strings.HasSuffix(strings.ToLower(finding.File), ".md") ||
			strings.Contains(strings.ToLower(finding.File), "doc")
		if !isDocFile && d.isInTestContext(finding, fileContent) {
			return false // Suppress findings in test contexts (but not in test files or docs)
		}
	}

	// In two-phase architecture, we don't suppress potential PI in comments
	// The LLM will determine if these are real PI or just examples

	// Check if finding looks like mock/dummy data
	// Don't apply mock context suppression to test files
	if !isTestFile && d.isInMockContext(finding, fileContent) {
		return false // Suppress mock data
	}

	// Check for specific false positive patterns
	if d.isFalsePositiveContext(finding, fileContent) {
		return false // Suppress known false positives
	}

	return true // Include by default
}

// isInTestContext checks if the finding is in a test-related context
func (d *detector) isInTestContext(finding Finding, content string) bool {
	// Extract context around the finding
	context := d.extractLineContext(content, finding.Line, 3)
	contextLower := strings.ToLower(context)

	// Test framework keywords - be more specific to avoid false positives
	// Don't include generic "test" as it matches too many variable names
	testKeywords := []string{
		"@test", "describe(", "it(", "it should", "expect(", "assert", "should(",
		"beforeeach", "aftereach", "setup(", "teardown(", "given(", "when(", "then(",
		"@parameterizedtest", "@valueSource", "unittest", "pytest", "test.skip",
		"scalatest", "wordspec", "funspec", "junit", "testng", "mockito",
		"jest.fn", "jest.mock", "test(", "test {", "test(\"", "test('",
		"spec(", "spec {", "spec(\"", "spec('",
	}

	for _, keyword := range testKeywords {
		if strings.Contains(contextLower, keyword) {
			return true
		}
	}

	return false
}

// isInCommentExample checks if finding is in a comment that appears to be an example
// UNUSED: In two-phase architecture, we detect examples for LLM validation
// func (d *detector) isInCommentExample(finding Finding, content string) bool {
// 	line := d.getLineContent(content, finding.Line)

// 	// Check if line contains comment markers
// 	if strings.Contains(line, "//") || strings.Contains(line, "#") ||
// 		strings.Contains(line, "/*") || strings.Contains(line, "*/") {

// 		lineLower := strings.ToLower(line)
// 		// More specific keywords that indicate it's definitely not real PI
// 		exampleKeywords := []string{
// 			"example:", "sample:", "todo", "fixme", "note:", "warning:",
// 			"replace with", "change to", "update this", "placeholder", "format:",
// 			"test example", "sample data", "dummy value",
// 		}

// 		for _, keyword := range exampleKeywords {
// 			if strings.Contains(lineLower, keyword) {
// 				return true
// 			}
// 		}
// 	}

// 	return false
// }

// isInMockContext checks if finding appears to be mock or dummy data
func (d *detector) isInMockContext(finding Finding, content string) bool {
	context := d.extractLineContext(content, finding.Line, 2)
	contextLower := strings.ToLower(context)

	mockKeywords := []string{
		"fakename", "stub", "dummy", "placeholder",
		"lorem", "ipsum", "demo data", "sample data", "template",
		"test_data", "mock_data", "dummy_data", "fake_data", "example_data",
		"testdata", "testfactory", "mockdata",
	}

	for _, keyword := range mockKeywords {
		if strings.Contains(contextLower, keyword) {
			return true
		}
	}

	return false
}

// looksLikeSyntheticData checks if the context contains synthetic-looking data patterns
func (d *detector) looksLikeSyntheticData(context string) bool {
	// Look for patterns like:
	// - Multiple sequential numbers: "123456789", "123456790", "123456791"
	// - Repeated digits: "111111111", "222222"
	// - Arrays/lists of similar numbers

	// Check for array/list patterns with multiple similar numbers
	if strings.Contains(context, `"111`) || strings.Contains(context, `"222`) ||
		strings.Contains(context, `"333`) || strings.Contains(context, `"444`) ||
		strings.Contains(context, `"555`) || strings.Contains(context, `"666`) ||
		strings.Contains(context, `"777`) || strings.Contains(context, `"888`) ||
		strings.Contains(context, `"999`) || strings.Contains(context, `"000`) {
		return true
	}

	// Check for sequential patterns in arrays or lists
	// Look for patterns like {"123456789", "123456790", "123456791"}
	numbers := regexp.MustCompile(`\d{9}`).FindAllString(context, -1)
	if len(numbers) >= 2 {
		// Check if all numbers are sequential
		sequential := true
		for i := 1; i < len(numbers); i++ {
			prev, _ := strconv.Atoi(numbers[i-1])
			curr, _ := strconv.Atoi(numbers[i])
			// Allow gaps of 1 or -1 for ascending/descending sequences
			if curr != prev+1 && curr != prev-1 {
				sequential = false
				break
			}
		}
		if sequential {
			return true
		}

		// Also check if all numbers are very similar (differ by < 10)
		similar := true
		first, _ := strconv.Atoi(numbers[0])
		for i := 1; i < len(numbers); i++ {
			curr, _ := strconv.Atoi(numbers[i])
			diff := curr - first
			if diff < 0 {
				diff = -diff
			}
			if diff > 10 {
				similar = false
				break
			}
		}
		if similar && len(numbers) >= 3 {
			return true
		}
	}

	return false
}

// isSyntheticPattern checks if a numeric string is likely synthetic/test data
func (d *detector) isSyntheticPattern(s string) bool {
	if len(s) == 0 {
		return false
	}

	// Check for all same digit (111111, 222222, etc)
	allSame := true
	firstChar := s[0]
	for i := 1; i < len(s); i++ {
		if s[i] != firstChar {
			allSame = false
			break
		}
	}
	if allSame {
		return true
	}

	// Check for sequential patterns (123456789, 987654321)
	sequential := true
	ascending := true
	descending := true

	for i := 1; i < len(s); i++ {
		prev := int(s[i-1] - '0')
		curr := int(s[i] - '0')

		if curr != prev+1 {
			ascending = false
		}
		if curr != prev-1 {
			descending = false
		}
		if !ascending && !descending {
			sequential = false
			break
		}
	}

	return sequential
}

// isFalsePositiveContext checks for known false positive patterns
func (d *detector) isFalsePositiveContext(finding Finding, content string) bool {
	// Get the line content
	line := d.getLineContent(content, finding.Line)
	lineLower := strings.ToLower(line)

	// Check for URL context (numbers in URLs are not PI)
	if strings.Contains(line, "http://") || strings.Contains(line, "https://") ||
		strings.Contains(line, ".com/") || strings.Contains(line, ".org/") {
		// Check if the match is part of a URL path
		beforeMatch := ""
		if finding.Column > 1 {
			beforeMatch = line[:finding.Column-1]
		}
		if strings.HasSuffix(beforeMatch, "/") || strings.HasSuffix(beforeMatch, "user/") {
			return true
		}
	}

	// Check for PR/Issue number context
	// More specific checks to avoid over-filtering
	beforeMatch := ""
	if finding.Column > 1 {
		beforeMatch = line[:finding.Column-1]
	}

	// Check if the match is preceded by common false positive indicators
	fpPrefixes := []string{
		"PR #", "PR#", "Issue #", "issue #", "ticket #",
		"Order #", "order #", "Invoice #", "invoice #",
		"Build #", "build #", "Reference #", "ref #",
		"JIRA-", "PROJ-",
	}

	for _, prefix := range fpPrefixes {
		if strings.HasSuffix(beforeMatch, prefix) || strings.HasSuffix(beforeMatch, strings.TrimSpace(prefix)) {
			return true
		}
	}

	// Also check the full line for these patterns
	if regexp.MustCompile(`(?i)(order|invoice|reference|build)\s*#\s*` + regexp.QuoteMeta(finding.Match)).MatchString(line) {
		return true
	}

	// Check for version number context
	if regexp.MustCompile(`(?i)version:?\s*\d+\.\d+`).MatchString(line) {
		return true
	}

	// Check for Lorem ipsum context
	if strings.Contains(lineLower, "lorem ipsum") {
		return true
	}

	// Check for sequential number ranges (e.g., "IDs from 123456789 to 123456799")
	if regexp.MustCompile(`(?i)(ids?|numbers?)\s+from\s+\d+\s+to\s+\d+`).MatchString(line) {
		return true
	}

	// Check for build numbers starting with year
	if regexp.MustCompile(`(?i)build:?\s*(19|20)\d{2}\d+`).MatchString(line) {
		return true
	}

	// In two-phase architecture, we detect examples for LLM validation
	// Only suppress if it's very obviously documentation (e.g., format descriptions)
	// if strings.Contains(lineLower, "// example") || strings.Contains(lineLower, "# example") {
	//     return true
	// }

	// Check for timestamp patterns
	if strings.Contains(lineLower, "timestamp") ||
		strings.Contains(lineLower, "created:") || strings.Contains(lineLower, "modified:") {
		return true
	}

	// Check for hash/checksum context
	if strings.Contains(lineLower, "checksum") || strings.Contains(lineLower, "hash") ||
		strings.Contains(lineLower, "crc") || strings.Contains(lineLower, "md5") ||
		strings.Contains(lineLower, "sha") {
		return true
	}

	// Credit card false positives
	if finding.Type == PITypeCreditCard {
		// Check for order number context
		falsePositivePatterns := []string{
			"order", "order#", "order #", "invoice", "invoice#",
			"transaction", "transaction#", "receipt", "receipt#",
			"reference", "ref#", "uuid", "guid",
		}

		for _, pattern := range falsePositivePatterns {
			if strings.Contains(lineLower, pattern) {
				// Additional check: is it prefixed with # or other order indicators?
				beforeMatch := ""
				if finding.Column > 1 {
					beforeMatch = line[:finding.Column-1]
				}
				if strings.HasSuffix(beforeMatch, "#") ||
					strings.HasSuffix(beforeMatch, "Order ") {
					return true
				}
			}
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

	contextSlice := lines[start : end+1]
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

			// Apply extractor if present
			extractedValue := value
			if m.extractor != nil {
				extractedValue = m.extractor(value)
				if extractedValue == "" {
					continue
				}
			}

			// Note: We no longer skip matches that fail validation here
			// This allows us to track all pattern matches and handle validation
			// results differently based on the PI type (e.g., skip driver licenses
			// that fail validation, but keep TFNs with low confidence)

			// Check validator but include match regardless
			validationPassed := true
			if m.validator != nil {
				validationPassed = m.validator(extractedValue)
			}

			matches = append(matches, PatternMatch{
				Value:            extractedValue,
				StartIndex:       match[0],
				EndIndex:         match[1],
				ValidationPassed: validationPassed,
			})
		}
	}

	return matches
}

// Type returns the PI type this matcher detects
func (m *regexMatcher) Type() PIType {
	return m.piType
}

// isValidAustralianAddress validates if a matched string is a legitimate Australian address
func (d *detector) isValidAustralianAddress(address string) bool {
	// Basic validation: must contain at least street name and number
	if len(address) < 5 {
		return false
	}

	// Extract and validate postcode if present
	postcodePattern := regexp.MustCompile(`\b(\d{4})\b`)
	postcodeMatches := postcodePattern.FindStringSubmatch(address)
	if len(postcodeMatches) > 1 {
		postcode := postcodeMatches[1]
		if !isValidAustralianPostcode(postcode) {
			return false
		}
	}

	// Check for valid Australian state abbreviations if present
	statePattern := regexp.MustCompile(`\b(NSW|VIC|QLD|SA|WA|TAS|NT|ACT)\b`)
	stateMatches := statePattern.FindStringSubmatch(address)
	if len(stateMatches) > 1 && len(postcodeMatches) > 1 {
		state := stateMatches[1]
		postcode := postcodeMatches[1]
		if !isValidStatePostcodeCombination(state, postcode) {
			return false
		}
	}

	// Check for common Australian street types
	streetTypePattern := regexp.MustCompile(`(?i)\b(Street|St|Road|Rd|Avenue|Ave|Drive|Dr|Lane|Ln|Place|Pl|Court|Ct|Crescent|Cres|Parade|Pde|Boulevard|Blvd|Highway|Hwy|Terrace|Tce|Way|Circuit|Cct|Close|Cl|Grove|Gr|Esplanade|Esp|Walk|Wk|Row|Rise|Ridge|View|Vw|Park|Gardens|Gdns|Square|Sq|Mall|Promenade|Prom)\b`)
	if !streetTypePattern.MatchString(address) {
		return false
	}

	// Reject obvious test patterns
	testPatterns := []string{
		"123 Test Street", "999 Fake Road", "111 Sample Avenue",
		"123 Example Road", "456 Demo Street", "789 Mock Avenue",
	}
	addressLower := strings.ToLower(address)
	for _, testPattern := range testPatterns {
		if strings.Contains(addressLower, strings.ToLower(testPattern)) {
			return false
		}
	}

	return true
}

// isValidAustralianPostcode validates Australian postcode ranges
func isValidAustralianPostcode(postcode string) bool {
	if len(postcode) != 4 {
		return false
	}

	code, err := strconv.Atoi(postcode)
	if err != nil {
		return false
	}

	// Valid Australian postcode ranges
	validRanges := [][]int{
		{200, 299},   // ACT (LVRs and PO Boxes)
		{800, 899},   // NT
		{900, 999},   // NT (LVRs and PO Boxes)
		{1000, 1999}, // NSW (LVRs and PO Boxes)
		{2000, 2599}, // NSW
		{2600, 2618}, // ACT
		{2619, 2899}, // NSW
		{2900, 2920}, // ACT
		{2921, 2999}, // NSW
		{3000, 3996}, // VIC
		{4000, 4999}, // QLD
		{5000, 5799}, // SA
		{5800, 5999}, // SA (LVRs and PO Boxes)
		{6000, 6797}, // WA
		{6800, 6999}, // WA (LVRs and PO Boxes)
		{7000, 7799}, // TAS
		{7800, 7999}, // TAS (LVRs and PO Boxes)
		{8000, 8999}, // VIC (LVRs and PO Boxes)
		{9000, 9999}, // QLD (LVRs and PO Boxes)
	}

	for _, validRange := range validRanges {
		if code >= validRange[0] && code <= validRange[1] {
			return true
		}
	}

	return false
}

// isValidStatePostcodeCombination validates state and postcode combinations
func isValidStatePostcodeCombination(state, postcode string) bool {
	code, err := strconv.Atoi(postcode)
	if err != nil {
		return false
	}

	switch state {
	case "NSW":
		return (code >= 1000 && code <= 1999) || (code >= 2000 && code <= 2599) || (code >= 2619 && code <= 2899) || (code >= 2921 && code <= 2999)
	case "ACT":
		return (code >= 200 && code <= 299) || (code >= 2600 && code <= 2618) || (code >= 2900 && code <= 2920)
	case "VIC":
		return (code >= 3000 && code <= 3996) || (code >= 8000 && code <= 8999)
	case "QLD":
		return (code >= 4000 && code <= 4999) || (code >= 9000 && code <= 9999)
	case "SA":
		return (code >= 5000 && code <= 5799) || (code >= 5800 && code <= 5999)
	case "WA":
		return (code >= 6000 && code <= 6797) || (code >= 6800 && code <= 6999)
	case "TAS":
		return (code >= 7000 && code <= 7799) || (code >= 7800 && code <= 7999)
	case "NT":
		return (code >= 800 && code <= 899) || (code >= 900 && code <= 999)
	}

	return false
}

// isValidAustralianPhone validates Australian phone numbers using libphonenumber
func (d *detector) isValidAustralianPhone(phoneStr string) bool {
	// Parse the phone number for Australia
	num, err := phonenumbers.Parse(phoneStr, "AU")
	if err != nil {
		// Try parsing without a default region for international numbers
		num, err = phonenumbers.Parse(phoneStr, "")
		if err != nil {
			return false
		}
	}

	// Check if it's a valid number
	if !phonenumbers.IsValidNumber(num) {
		return false
	}

	// Check if it's specifically Australian
	region := phonenumbers.GetRegionCodeForNumber(num)
	if region != "AU" {
		return false
	}

	// Additional checks to prevent confusion with other PI types
	formattedNumber := phonenumbers.Format(num, phonenumbers.E164)

	// Reject if it looks like an ABN (would be +61 followed by specific patterns)
	// ABNs don't start with typical phone prefixes
	if strings.HasPrefix(formattedNumber, "+61") {
		// Extract national number
		nationalNumber := strings.TrimPrefix(formattedNumber, "+61")

		// Australian mobile numbers start with 4
		// Landlines start with 2, 3, 7, 8 depending on state
		// Service numbers start with 13 or 18
		if len(nationalNumber) >= 1 {
			firstDigit := nationalNumber[0]
			// Valid Australian phone number prefixes
			if firstDigit == '2' || firstDigit == '3' || firstDigit == '4' ||
				firstDigit == '7' || firstDigit == '8' ||
				(len(nationalNumber) >= 2 && (nationalNumber[:2] == "13" || nationalNumber[:2] == "18")) {
				return true
			}
		}
	}

	return false
}

// isValidAustralianTFN validates Australian Tax File Numbers using modulo 11 checksum
func (d *detector) isValidAustralianTFN(tfnStr string) bool {
	// Remove spaces and dashes
	clean := regexp.MustCompile(`[\s\-]`).ReplaceAllString(tfnStr, "")

	// Must be exactly 9 digits and not start with 0
	if len(clean) != 9 || clean[0] == '0' {
		return false
	}

	// Check for synthetic patterns (all same digit, sequential)
	if d.isSyntheticPattern(clean) {
		return false
	}

	// Convert to array of integers
	digits := make([]int, 9)
	for i, char := range clean {
		digit := int(char - '0')
		if digit < 0 || digit > 9 {
			return false
		}
		digits[i] = digit
	}

	// TFN validation weights: [1, 4, 3, 7, 5, 8, 6, 9, 10]
	weights := []int{1, 4, 3, 7, 5, 8, 6, 9, 10}

	// Calculate weighted sum
	sum := 0
	for i := 0; i < 9; i++ {
		sum += digits[i] * weights[i]
	}

	// Valid TFN if sum is divisible by 11
	return sum%11 == 0
}

// isValidAustralianMedicare validates Australian Medicare numbers using proper checksum and IRN validation
func (d *detector) isValidAustralianMedicare(medicareStr string) bool {
	// Remove spaces, dashes, and issue number
	clean := regexp.MustCompile(`[\s\-/]`).ReplaceAllString(medicareStr, "")

	// Extract medicare number and IRN
	var medicare, irn string
	if len(clean) >= 10 {
		medicare = clean[:10]
		if len(clean) >= 11 {
			irn = clean[10:]
		}
	} else {
		return false
	}

	// First digit must be 2-6
	if medicare[0] < '2' || medicare[0] > '6' {
		return false
	}

	// Validate IRN (Individual Reference Number) if present
	if irn != "" {
		irnNum, err := strconv.Atoi(irn)
		if err != nil || irnNum < 1 || irnNum > 9 {
			return false
		}
	}

	// Convert medicare number to array of integers
	digits := make([]int, 10)
	for i, char := range medicare {
		digit := int(char - '0')
		if digit < 0 || digit > 9 {
			return false
		}
		digits[i] = digit
	}

	// Medicare checksum validation using weights [1, 3, 7, 9, 1, 3, 7, 9]
	// Only use first 8 digits for checksum, 9th digit is the check digit
	weights := []int{1, 3, 7, 9, 1, 3, 7, 9}

	// Calculate weighted sum of first 8 digits
	sum := 0
	for i := 0; i < 8; i++ {
		sum += digits[i] * weights[i]
	}

	// Check digit should equal sum % 10
	checkDigit := sum % 10
	return checkDigit == digits[8]
}

// isValidAustralianBSB validates Australian Bank State Branch codes
func (d *detector) isValidAustralianBSB(bsbStr string) bool {
	// Remove dashes and spaces
	clean := regexp.MustCompile(`[\s\-]`).ReplaceAllString(bsbStr, "")

	// Must be exactly 6 digits
	if len(clean) != 6 {
		return false
	}

	// Check for synthetic patterns (all same digit)
	if d.isSyntheticPattern(clean) {
		return false
	}

	// Convert to integer for range checking
	bsbNum, err := strconv.Atoi(clean)
	if err != nil {
		return false
	}

	// Enhanced BSB validation with known bank ranges
	// Based on APCA BSB directory structure

	// Reserve Bank of Australia: 001-000 to 009-999
	if bsbNum >= 1000 && bsbNum <= 9999 {
		return true
	}

	// Commonwealth Bank: 060-000 to 069-999, 062-000 to 064-999
	if (bsbNum >= 60000 && bsbNum <= 69999) || (bsbNum >= 62000 && bsbNum <= 64999) {
		return true
	}

	// Westpac Banking Corporation: 030-000 to 039-999, 732-000 to 739-999
	if (bsbNum >= 30000 && bsbNum <= 39999) || (bsbNum >= 732000 && bsbNum <= 739999) {
		return true
	}

	// Australia and New Zealand Banking Group: 010-000 to 019-999
	if bsbNum >= 10000 && bsbNum <= 19999 {
		return true
	}

	// National Australia Bank: 080-000 to 089-999
	if bsbNum >= 80000 && bsbNum <= 89999 {
		return true
	}

	// Credit unions and building societies: 800-000 to 839-999
	if bsbNum >= 800000 && bsbNum <= 839999 {
		return true
	}

	// Other financial institutions: Various ranges
	// Bendigo Bank: 633-000 to 633-999
	if bsbNum >= 633000 && bsbNum <= 633999 {
		return true
	}

	// ING Bank: 923-000 to 923-999
	if bsbNum >= 923000 && bsbNum <= 923999 {
		return true
	}

	// Macquarie Bank: 182-000 to 182-999
	if bsbNum >= 182000 && bsbNum <= 182999 {
		return true
	}

	// Basic format validation as fallback
	// First digit should be 0-9 (expanded from original 0-7)
	// Check for obvious invalid patterns
	firstDigit := clean[0]
	if firstDigit < '0' || firstDigit > '9' {
		return false
	}

	// Reject obvious test patterns
	if clean == "000000" || clean == "111111" || clean == "222222" ||
		clean == "333333" || clean == "444444" || clean == "555555" ||
		clean == "666666" || clean == "777777" || clean == "888888" ||
		clean == "999999" {
		return false
	}

	// If it doesn't match known ranges but has valid format, be conservative
	// Return false to reduce false positives unless we're confident it's a real BSB
	return false
}

// isValidAustralianDriverLicense validates Australian driver licenses with state-specific formats
func (d *detector) isValidAustralianDriverLicense(licenseStr string) bool {
	license := strings.TrimSpace(licenseStr)

	// Check if the license string contains context keywords (from the regex pattern)
	lowerLicense := strings.ToLower(license)
	hasContext := strings.Contains(lowerLicense, "license") ||
		strings.Contains(lowerLicense, "licence") ||
		strings.Contains(lowerLicense, "driver") ||
		strings.Contains(lowerLicense, "dl")

	// Remove all non-alphanumeric for checking
	alphaNumeric := regexp.MustCompile(`[^A-Za-z0-9]`).ReplaceAllString(license, "")
	digits := regexp.MustCompile(`[^\d]`).ReplaceAllString(license, "")

	// Exclude numbers that are clearly other PI types
	if len(digits) == 9 {
		// Could be TFN or ACN - exclude sequential/repeated patterns
		if digits == "123456789" || digits == "987654321" {
			return false
		}
		// Check for repeated digits
		firstDigit := digits[0]
		allSame := true
		for i := 0; i < len(digits); i++ {
			if digits[i] != firstDigit {
				allSame = false
				break
			}
		}
		if allSame {
			return false
		}
	}
	if len(digits) == 10 {
		// Could be Medicare number - be more cautious
		// Medicare numbers start with 2-6, but we'll exclude all 10-digit numbers
		// that start with 1-6 to be safe
		if digits[0] >= '1' && digits[0] <= '6' {
			return false
		}
	}
	if len(digits) == 11 {
		// Could be ABN
		return false
	}

	// Exclude phone numbers first
	if len(license) > 0 && (license[0] == '0' || strings.HasPrefix(license, "+61")) {
		return false
	}
	if strings.HasPrefix(digits, "1300") || strings.HasPrefix(digits, "1800") ||
		strings.HasPrefix(digits, "04") || strings.HasPrefix(digits, "614") {
		return false
	}

	// State-specific validation based on research

	// NSW: 8-digit numeric format
	if regexp.MustCompile(`^\d{8}$`).MatchString(alphaNumeric) {
		// For pure numeric patterns without context, be VERY restrictive
		if !hasContext {
			return false // Don't match pure 8-digit numbers without context
		}

		// Exclude obvious test patterns
		if isTestDriverLicense(alphaNumeric) {
			return false
		}
		// NSW licenses typically don't start with 0 or 1
		if alphaNumeric[0] == '0' || alphaNumeric[0] == '1' {
			return false
		}
		// Be more restrictive - require some entropy (not all same digit)
		firstDigit := alphaNumeric[0]
		allSame := true
		for i := 1; i < len(alphaNumeric); i++ {
			if alphaNumeric[i] != firstDigit {
				allSame = false
				break
			}
		}
		if allSame {
			return false
		}
		// NSW licenses are typically in specific ranges - be conservative
		// This is a heuristic to reduce false positives
		num, _ := strconv.Atoi(alphaNumeric)
		// NSW licenses are typically in higher ranges
		return num >= 20000000
	}

	// VIC: 9 digits exactly (based on more specific research)
	// This prevents matching with TFNs and other 9-digit numbers
	if regexp.MustCompile(`^\d{9}$`).MatchString(alphaNumeric) {
		// For pure numeric patterns without context, be VERY restrictive
		if !hasContext {
			return false // Don't match pure 9-digit numbers without context
		}

		if isTestDriverLicense(alphaNumeric) {
			return false
		}
		// Additional validation to prevent matching other PI types
		// VIC licenses typically don't start with 0 or 1
		if alphaNumeric[0] == '0' || alphaNumeric[0] == '1' {
			return false
		}
		// Exclude patterns that look like TFNs (sequential or structured)
		if isSequentialNumber(alphaNumeric) {
			return false
		}
		return true
	}

	// SA: 1 letter + 6 digits (7 characters total)
	if regexp.MustCompile(`^[A-Z]\d{6}$`).MatchString(alphaNumeric) {
		return true
	}

	// WA: Various formats (not fully specified)
	if regexp.MustCompile(`^\d{7}$`).MatchString(alphaNumeric) {
		// For pure numeric patterns without context, be restrictive
		if !hasContext {
			return false
		}
		return !isTestDriverLicense(alphaNumeric)
	}

	// TAS: 1 letter + 6 digits OR 2 letters + 5 digits
	if regexp.MustCompile(`^[A-Z]\d{6}$`).MatchString(alphaNumeric) ||
		regexp.MustCompile(`^[A-Z]{2}\d{5}$`).MatchString(alphaNumeric) {
		return true
	}

	// NT: Following general Australian pattern
	if regexp.MustCompile(`^\d{7,8}$`).MatchString(alphaNumeric) {
		// For pure numeric patterns without context, be restrictive
		if !hasContext {
			return false
		}
		return !isTestDriverLicense(alphaNumeric)
	}

	// ACT: Following general Australian pattern
	if regexp.MustCompile(`^\d{7,8}$`).MatchString(alphaNumeric) {
		// For pure numeric patterns without context, be restrictive
		if !hasContext {
			return false
		}
		return !isTestDriverLicense(alphaNumeric)
	}

	return false
}

// isTestDriverLicense checks for obvious test patterns in driver licenses
func isTestDriverLicense(license string) bool {
	testPatterns := []string{
		"00000000", "11111111", "22222222", "33333333", "44444444",
		"55555555", "66666666", "77777777", "88888888", "99999999",
		"12345678", "87654321", "00000007", "0000000", "1111111",
		"2222222", "3333333", "4444444", "5555555", "6666666",
		"7777777", "8888888", "9999999", "1234567", "7654321",
		"123456789", "987654321",
	}

	for _, pattern := range testPatterns {
		if license == pattern {
			return true
		}
	}

	// Also check for sequential patterns
	return isSequentialNumber(license)
}

// isValidCreditCard validates credit card numbers using the validator
func (d *detector) isValidCreditCard(cardStr string) bool {
	// Get credit card validator from registry
	validator, ok := d.validators.Get("CREDIT_CARD")
	if !ok {
		return false
	}

	// Validate using the credit card validator (includes Luhn check)
	valid, _ := validator.Validate(cardStr)
	return valid
}

// isSequentialNumber checks if a number is sequential (like 123456789)
func isSequentialNumber(num string) bool {
	if len(num) < 3 {
		return false
	}

	// Check for ascending sequence
	ascending := true
	for i := 1; i < len(num); i++ {
		if num[i] != num[i-1]+1 {
			ascending = false
			break
		}
	}

	// Check for descending sequence
	descending := true
	for i := 1; i < len(num); i++ {
		if num[i] != num[i-1]-1 {
			descending = false
			break
		}
	}

	return ascending || descending
}
