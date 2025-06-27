package output

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/MacAttak/pi-scanner/pkg/detection"
)

// MaskingLevel represents the level of masking to apply
type MaskingLevel string

const (
	// MaskingLevelFull completely redacts the value
	MaskingLevelFull MaskingLevel = "FULL"

	// MaskingLevelPartial shows some characters for verification
	MaskingLevelPartial MaskingLevel = "PARTIAL"

	// MaskingLevelNone shows the complete value (use with caution)
	MaskingLevelNone MaskingLevel = "NONE"
)

// Masker handles masking of sensitive data
type Masker struct {
	level          MaskingLevel
	customPatterns map[detection.PIType]MaskPattern
}

// MaskPattern defines how to mask a specific PI type
type MaskPattern struct {
	ShowPrefix    int    // Number of characters to show at start
	ShowSuffix    int    // Number of characters to show at end
	MaskChar      string // Character to use for masking
	PreserveChars string // Characters to preserve (e.g., "@" in emails)
}

// DefaultMaskPatterns returns the default masking patterns for each PI type
func DefaultMaskPatterns() map[detection.PIType]MaskPattern {
	return map[detection.PIType]MaskPattern{
		detection.PITypeTFN: {
			ShowPrefix: 3,
			ShowSuffix: 2,
			MaskChar:   "*",
		},
		detection.PITypeMedicare: {
			ShowPrefix: 2,
			ShowSuffix: 2,
			MaskChar:   "*",
		},
		detection.PITypeCreditCard: {
			ShowPrefix: 0,
			ShowSuffix: 4,
			MaskChar:   "*",
		},
		detection.PITypeEmail: {
			ShowPrefix:    2,
			ShowSuffix:    0,
			MaskChar:      "*",
			PreserveChars: "@.",
		},
		detection.PITypePhone: {
			ShowPrefix: 4, // Area code
			ShowSuffix: 2,
			MaskChar:   "*",
		},
		detection.PITypePassport: {
			ShowPrefix: 1, // Letter prefix
			ShowSuffix: 2,
			MaskChar:   "*",
		},
		detection.PITypeDriverLicense: {
			ShowPrefix: 2,
			ShowSuffix: 2,
			MaskChar:   "*",
		},
		detection.PITypeABN: {
			ShowPrefix: 2,
			ShowSuffix: 3,
			MaskChar:   "*",
		},
		detection.PITypeARBN: {
			ShowPrefix: 2,
			ShowSuffix: 3,
			MaskChar:   "*",
		},
		detection.PITypeBSB: {
			ShowPrefix: 3,
			ShowSuffix: 0,
			MaskChar:   "*",
		},
		detection.PITypeBankAccount: {
			ShowPrefix: 0,
			ShowSuffix: 4,
			MaskChar:   "*",
		},
		// Default pattern for other types
		detection.PITypeName: {
			ShowPrefix: 1,
			ShowSuffix: 1,
			MaskChar:   "*",
		},
		detection.PITypeAddress: {
			ShowPrefix: 5,
			ShowSuffix: 0,
			MaskChar:   "*",
		},
	}
}

// NewMasker creates a new masker with the specified level
func NewMasker(level MaskingLevel) *Masker {
	return &Masker{
		level:          level,
		customPatterns: DefaultMaskPatterns(),
	}
}

// SetPattern sets a custom masking pattern for a specific PI type
func (m *Masker) SetPattern(piType detection.PIType, pattern MaskPattern) {
	m.customPatterns[piType] = pattern
}

// SetLevel changes the masking level
func (m *Masker) SetLevel(level MaskingLevel) {
	m.level = level
}

// Mask applies masking to a sensitive value based on its type
func (m *Masker) Mask(value string, piType detection.PIType) string {
	switch m.level {
	case MaskingLevelNone:
		return value
	case MaskingLevelFull:
		return m.fullMask(value)
	case MaskingLevelPartial:
		return m.partialMask(value, piType)
	default:
		// Default to partial masking for safety
		return m.partialMask(value, piType)
	}
}

// MaskFinding masks the sensitive data in a finding
func (m *Masker) MaskFinding(finding *detection.Finding) detection.Finding {
	masked := *finding
	masked.Match = m.Mask(finding.Match, finding.Type)

	// Also mask any PI that might appear in context
	if finding.Context != "" {
		masked.Context = m.maskContext(finding.Context, finding.Match, finding.Type)
	}
	if finding.ContextBefore != "" {
		masked.ContextBefore = m.maskContext(finding.ContextBefore, finding.Match, finding.Type)
	}
	if finding.ContextAfter != "" {
		masked.ContextAfter = m.maskContext(finding.ContextAfter, finding.Match, finding.Type)
	}

	return masked
}

// fullMask completely redacts the value
func (m *Masker) fullMask(value string) string {
	if value == "" {
		return ""
	}

	// Preserve structure (spaces, dashes, etc) but mask all alphanumeric
	var result strings.Builder
	for _, ch := range value {
		if unicode.IsLetter(ch) || unicode.IsDigit(ch) {
			result.WriteString("*")
		} else {
			result.WriteRune(ch)
		}
	}

	return result.String()
}

// partialMask applies type-specific partial masking
func (m *Masker) partialMask(value string, piType detection.PIType) string {
	if value == "" {
		return ""
	}

	// Get the pattern for this PI type
	pattern, exists := m.customPatterns[piType]
	if !exists {
		// Use default pattern
		pattern = MaskPattern{
			ShowPrefix: 1,
			ShowSuffix: 1,
			MaskChar:   "*",
		}
	}

	// Special handling for email addresses
	if piType == detection.PITypeEmail {
		return m.maskEmail(value, pattern)
	}

	// Clean the value (remove non-alphanumeric except preserved chars)
	cleaned := m.cleanValue(value, pattern.PreserveChars)

	// If the value is too short, mask it entirely
	if len(cleaned) <= pattern.ShowPrefix+pattern.ShowSuffix {
		return strings.Repeat(pattern.MaskChar, len(cleaned))
	}

	// Build the masked value
	var result strings.Builder

	// Add prefix
	if pattern.ShowPrefix > 0 {
		result.WriteString(cleaned[:pattern.ShowPrefix])
	}

	// Add masked middle
	middleLen := len(cleaned) - pattern.ShowPrefix - pattern.ShowSuffix
	result.WriteString(strings.Repeat(pattern.MaskChar, middleLen))

	// Add suffix
	if pattern.ShowSuffix > 0 {
		result.WriteString(cleaned[len(cleaned)-pattern.ShowSuffix:])
	}

	return result.String()
}

// maskEmail applies special masking for email addresses
func (m *Masker) maskEmail(email string, pattern MaskPattern) string {
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		// Not a valid email format, use default masking
		return m.fullMask(email)
	}

	localPart := parts[0]
	domain := parts[1]

	// Clean the local part (remove dots and special chars except those preserved)
	cleanLocal := ""
	for _, ch := range localPart {
		if unicode.IsLetter(ch) || unicode.IsDigit(ch) {
			cleanLocal += string(ch)
		}
	}

	// Mask local part - always show 3 asterisks for consistency with existing code
	if len(cleanLocal) <= pattern.ShowPrefix {
		localPart = strings.Repeat(pattern.MaskChar, len(cleanLocal))
	} else {
		localPart = cleanLocal[:pattern.ShowPrefix] + "***"
	}

	return localPart + "@" + domain
}

// cleanValue removes non-alphanumeric characters except those specified
func (m *Masker) cleanValue(value string, preserveChars string) string {
	var result strings.Builder
	for _, ch := range value {
		if unicode.IsLetter(ch) || unicode.IsDigit(ch) || strings.ContainsRune(preserveChars, ch) {
			result.WriteRune(ch)
		}
	}
	return result.String()
}

// maskContext masks occurrences of the sensitive value in context
func (m *Masker) maskContext(context string, sensitiveValue string, piType detection.PIType) string {
	if context == "" || sensitiveValue == "" {
		return context
	}

	maskedValue := m.Mask(sensitiveValue, piType)

	// Replace all occurrences of the sensitive value with the masked version
	result := context

	// Try exact match first
	if strings.Contains(result, sensitiveValue) {
		result = strings.ReplaceAll(result, sensitiveValue, maskedValue)
		return result // If exact match found, we're done
	}

	// Clean the sensitive value to get just alphanumeric
	cleanSensitive := ""
	for _, ch := range sensitiveValue {
		if unicode.IsLetter(ch) || unicode.IsDigit(ch) {
			cleanSensitive += string(ch)
		}
	}

	// Try common variations
	// First, try variations that maintain the same length
	variations := []string{
		strings.ReplaceAll(sensitiveValue, " ", "-"),
		strings.ReplaceAll(sensitiveValue, "-", " "),
		strings.ReplaceAll(sensitiveValue, " ", ""),
		strings.ReplaceAll(sensitiveValue, "-", ""),
	}

	// For TFN specifically, check for common formats
	if piType == detection.PITypeTFN && len(cleanSensitive) == 9 {
		variations = append(variations,
			cleanSensitive[:3]+"-"+cleanSensitive[3:6]+"-"+cleanSensitive[6:],
			cleanSensitive[:3]+" "+cleanSensitive[3:6]+" "+cleanSensitive[6:],
		)
	}

	for _, variant := range variations {
		if variant != "" && variant != sensitiveValue && strings.Contains(result, variant) {
			result = strings.ReplaceAll(result, variant, maskedValue)
		}
	}

	return result
}

// ValidateMaskingLevel validates that a masking level is valid
func ValidateMaskingLevel(level string) (MaskingLevel, error) {
	switch MaskingLevel(level) {
	case MaskingLevelFull, MaskingLevelPartial, MaskingLevelNone:
		return MaskingLevel(level), nil
	default:
		return MaskingLevelPartial, fmt.Errorf("invalid masking level: %s", level)
	}
}

// MaskSensitiveData is a compatibility function that uses partial masking
// This maintains backward compatibility with existing code
func MaskSensitiveData(value string, piType string) string {
	masker := NewMasker(MaskingLevelPartial)
	return masker.Mask(value, detection.PIType(piType))
}
