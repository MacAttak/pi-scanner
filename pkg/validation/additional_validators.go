package validation

import (
	"regexp"
	"strconv"
	"strings"
)

// PassportValidator validates Australian passport numbers
type PassportValidator struct{}

// Validate checks if the passport number is valid
func (v *PassportValidator) Validate(value string) (bool, error) {
	// Remove spaces
	passport := strings.ReplaceAll(value, " ", "")

	// Australian passport format: 1 letter + 7 digits OR 2 letters + 7 digits (newer format)
	// Example: N1234567 or PA1234567

	// Pattern 1: Single letter + 7 digits (older format)
	pattern1 := regexp.MustCompile(`^[A-Z]\d{7}$`)
	// Pattern 2: Two letters + 7 digits (newer format)
	pattern2 := regexp.MustCompile(`^[A-Z]{2}\d{7}$`)

	if pattern1.MatchString(passport) || pattern2.MatchString(passport) {
		// Additional validation: First letter should be valid passport series
		// Valid series: E, K, L, M, N, P, R, S, T, X, Z (for single letter)
		// PA, PB, PC, PD, PE (for two letter format)
		if len(passport) == 8 {
			validSeries := []string{"E", "K", "L", "M", "N", "P", "R", "S", "T", "X", "Z"}
			firstLetter := string(passport[0])
			for _, series := range validSeries {
				if firstLetter == series {
					return true, nil
				}
			}
		} else if len(passport) == 9 && strings.HasPrefix(passport, "P") {
			// New format starts with P
			return true, nil
		}
	}

	return false, nil
}

// Type returns the PI type
func (v *PassportValidator) Type() string {
	return "PASSPORT"
}

// Normalize returns normalized passport number
func (v *PassportValidator) Normalize(value string) string {
	return strings.ToUpper(strings.ReplaceAll(value, " ", ""))
}

// BankAccountValidator validates Australian bank account numbers
type BankAccountValidator struct{}

// Validate checks if the bank account number is valid
func (v *BankAccountValidator) Validate(value string) (bool, error) {
	// Remove spaces and dashes
	account := regexp.MustCompile(`[\s\-]`).ReplaceAllString(value, "")

	// Australian bank accounts are typically 6-10 digits
	// Some banks use different formats:
	// - Commonwealth Bank: 10 digits
	// - ANZ: 9 digits
	// - Westpac: 10 digits
	// - NAB: 9-10 digits

	if len(account) < 6 || len(account) > 10 {
		return false, nil
	}

	// Must be all digits
	if !regexp.MustCompile(`^\d+$`).MatchString(account) {
		return false, nil
	}

	// For 10-digit accounts, we could do check digit validation
	// but Australian banks use different algorithms
	// For now, accept all valid digit sequences

	// For other lengths, accept if all digits
	return true, nil
}

// Type returns the PI type
func (v *BankAccountValidator) Type() string {
	return "BANK_ACCOUNT"
}

// Normalize returns normalized account number
func (v *BankAccountValidator) Normalize(value string) string {
	return regexp.MustCompile(`[^\d]`).ReplaceAllString(value, "")
}

// DriverLicenseValidator validates Australian driver's licenses
type DriverLicenseValidator struct {
	stateValidators map[string]*regexp.Regexp
}

// NewDriverLicenseValidator creates a new driver license validator
func NewDriverLicenseValidator() *DriverLicenseValidator {
	return &DriverLicenseValidator{
		stateValidators: map[string]*regexp.Regexp{
			// NSW: 8 digits OR 2 letters + 6 digits
			"NSW": regexp.MustCompile(`^(\d{8}|[A-Z]{2}\d{6})$`),
			// VIC: 1-10 digits OR 1-9 digits + 1 letter
			"VIC": regexp.MustCompile(`^(\d{1,10}|\d{1,9}[A-Z])$`),
			// QLD: 9 digits (Open licence) OR Customer Reference Number format
			"QLD": regexp.MustCompile(`^\d{9}$`),
			// SA: 1 letter + 6 digits
			"SA": regexp.MustCompile(`^[A-Z]\d{6}$`),
			// WA: 7 digits
			"WA": regexp.MustCompile(`^\d{7}$`),
			// TAS: 7 digits OR 8 alphanumeric (at least one letter)
			"TAS": regexp.MustCompile(`^(\d{7}|[A-Z0-9]{8})$`),
			// NT: 10 digits
			"NT": regexp.MustCompile(`^\d{10}$`),
			// ACT: 10 digits
			"ACT": regexp.MustCompile(`^\d{10}$`),
		},
	}
}

// Validate checks if the license number is valid for any state
func (v *DriverLicenseValidator) Validate(value string) (bool, error) {
	// Clean the value
	cleaned := strings.ToUpper(strings.ReplaceAll(value, " ", ""))
	cleaned = strings.ReplaceAll(cleaned, "-", "")

	// Reject if contains lowercase letters (not normalized)
	if value != "" && strings.ToUpper(value) != value && !strings.Contains(value, " ") && !strings.Contains(value, "-") {
		return false, nil
	}

	// Check against all state patterns
	for _, pattern := range v.stateValidators {
		if pattern.MatchString(cleaned) {
			return true, nil
		}
	}

	return false, nil
}

// ValidateForState checks if the license is valid for a specific state
func (v *DriverLicenseValidator) ValidateForState(value, state string) (bool, error) {
	cleaned := strings.ToUpper(strings.ReplaceAll(value, " ", ""))
	cleaned = strings.ReplaceAll(cleaned, "-", "")

	pattern, exists := v.stateValidators[strings.ToUpper(state)]
	if !exists {
		return false, nil
	}

	return pattern.MatchString(cleaned), nil
}

// Type returns the PI type
func (v *DriverLicenseValidator) Type() string {
	return "DRIVER_LICENSE"
}

// Normalize returns normalized license number
func (v *DriverLicenseValidator) Normalize(value string) string {
	return strings.ToUpper(strings.ReplaceAll(strings.ReplaceAll(value, " ", ""), "-", ""))
}

// ARBNValidator validates Australian Registered Body Numbers
type ARBNValidator struct{}

// Validate checks if the ARBN is valid
func (v *ARBNValidator) Validate(value string) (bool, error) {
	// Remove spaces and dashes
	arbn := regexp.MustCompile(`[\s\-]`).ReplaceAllString(value, "")

	// ARBN is 9 digits, uses same check digit algorithm as ACN
	if len(arbn) != 9 {
		return false, nil
	}

	// Check all digits
	if !regexp.MustCompile(`^\d{9}$`).MatchString(arbn) {
		return false, nil
	}

	// ARBN uses same algorithm as ACN
	// Weights: 8, 7, 6, 5, 4, 3, 2, 1
	weights := []int{8, 7, 6, 5, 4, 3, 2, 1}
	sum := 0

	for i := 0; i < 8; i++ {
		digit, _ := strconv.Atoi(string(arbn[i]))
		sum += digit * weights[i]
	}

	// Calculate check digit
	remainder := sum % 10
	checkDigit := (10 - remainder) % 10

	// Compare with last digit
	lastDigit, _ := strconv.Atoi(string(arbn[8]))

	return checkDigit == lastDigit, nil
}

// Type returns the PI type
func (v *ARBNValidator) Type() string {
	return "ARBN"
}

// Normalize returns normalized ARBN
func (v *ARBNValidator) Normalize(value string) string {
	return regexp.MustCompile(`[^\d]`).ReplaceAllString(value, "")
}

// RegisterAdditionalValidators registers all additional validators
func RegisterAdditionalValidators(registry *ValidatorRegistry) {
	registry.Register(&PassportValidator{})
	registry.Register(&BankAccountValidator{})
	registry.Register(NewDriverLicenseValidator())
	registry.Register(&ARBNValidator{})
}
