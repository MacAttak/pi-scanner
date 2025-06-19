package validation

import (
	"regexp"
	"strconv"
	"strings"
)

// TFNValidator validates Australian Tax File Numbers
type TFNValidator struct{}

// Validate checks if the TFN is valid using the official algorithm
func (v *TFNValidator) Validate(value string) (bool, error) {
	// Remove spaces and dashes
	tfn := regexp.MustCompile(`[\s\-]`).ReplaceAllString(value, "")
	
	// Must be 9 digits
	if len(tfn) != 9 {
		return false, nil
	}
	
	// Check all digits
	if !regexp.MustCompile(`^\d{9}$`).MatchString(tfn) {
		return false, nil
	}
	
	// Official TFN weights: 1, 4, 3, 7, 5, 8, 6, 9, 10
	weights := []int{1, 4, 3, 7, 5, 8, 6, 9, 10}
	sum := 0
	
	for i := 0; i < 9; i++ {
		digit, _ := strconv.Atoi(string(tfn[i]))
		sum += digit * weights[i]
	}
	
	// Valid if sum is divisible by 11
	return sum%11 == 0, nil
}

// Type returns the PI type
func (v *TFNValidator) Type() string {
	return "TFN"
}

// Normalize returns normalized TFN
func (v *TFNValidator) Normalize(value string) string {
	// Remove all non-digits
	return regexp.MustCompile(`[^\d]`).ReplaceAllString(value, "")
}

// ABNValidator validates Australian Business Numbers
type ABNValidator struct{}

// Validate checks if the ABN is valid using modulus 89
func (v *ABNValidator) Validate(value string) (bool, error) {
	// Remove spaces and dashes
	abn := regexp.MustCompile(`[\s\-]`).ReplaceAllString(value, "")
	
	// Must be 11 digits
	if len(abn) != 11 {
		return false, nil
	}
	
	// Check all digits
	if !regexp.MustCompile(`^\d{11}$`).MatchString(abn) {
		return false, nil
	}
	
	// Modulus 89 algorithm
	// Step 1: Subtract 1 from first digit
	firstDigit, _ := strconv.Atoi(string(abn[0]))
	firstDigit -= 1
	
	// Step 2: Apply weights: 10, 1, 3, 5, 7, 9, 11, 13, 15, 17, 19
	weights := []int{10, 1, 3, 5, 7, 9, 11, 13, 15, 17, 19}
	sum := firstDigit * weights[0]
	
	for i := 1; i < 11; i++ {
		digit, _ := strconv.Atoi(string(abn[i]))
		sum += digit * weights[i]
	}
	
	// Valid if sum is divisible by 89
	return sum%89 == 0, nil
}

// Type returns the PI type
func (v *ABNValidator) Type() string {
	return "ABN"
}

// Normalize returns normalized ABN
func (v *ABNValidator) Normalize(value string) string {
	return regexp.MustCompile(`[^\d]`).ReplaceAllString(value, "")
}

// MedicareValidator validates Australian Medicare numbers
type MedicareValidator struct{}

// Validate checks if the Medicare number is valid
func (v *MedicareValidator) Validate(value string) (bool, error) {
	// Remove spaces, dashes, and slashes
	medicare := regexp.MustCompile(`[\s\-/]`).ReplaceAllString(value, "")
	
	// Medicare numbers are 10 or 11 digits (with IRN)
	if len(medicare) < 10 || len(medicare) > 11 {
		return false, nil
	}
	
	// First digit must be 2-6
	if medicare[0] < '2' || medicare[0] > '6' {
		return false, nil
	}
	
	// Extract main number (first 8 digits) and check digit (9th digit)
	mainNumber := medicare[:8]
	checkDigit, _ := strconv.Atoi(string(medicare[8]))
	
	// Weights: 1, 3, 7, 9, 1, 3, 7, 9
	weights := []int{1, 3, 7, 9, 1, 3, 7, 9}
	sum := 0
	
	for i := 0; i < 8; i++ {
		digit, _ := strconv.Atoi(string(mainNumber[i]))
		sum += digit * weights[i]
	}
	
	// Check digit should equal sum % 10
	calculatedCheck := sum % 10
	
	return calculatedCheck == checkDigit, nil
}

// Type returns the PI type
func (v *MedicareValidator) Type() string {
	return "MEDICARE"
}

// Normalize returns normalized Medicare number
func (v *MedicareValidator) Normalize(value string) string {
	return regexp.MustCompile(`[^\d]`).ReplaceAllString(value, "")
}

// BSBValidator validates Australian Bank State Branch codes
type BSBValidator struct{}

// Validate checks if the BSB is valid
func (v *BSBValidator) Validate(value string) (bool, error) {
	// Remove dashes and spaces
	bsb := strings.ReplaceAll(value, "-", "")
	bsb = strings.ReplaceAll(bsb, " ", "")
	
	// Must be exactly 6 digits
	if len(bsb) != 6 {
		return false, nil
	}
	
	// Check all digits
	if !regexp.MustCompile(`^\d{6}$`).MatchString(bsb) {
		return false, nil
	}
	
	// BSB format validation:
	// First 2 digits: Bank code
	// 3rd digit: State code (2-7 are valid state codes)
	// Last 3 digits: Branch code
	
	stateDigit := bsb[2]
	if stateDigit < '2' || stateDigit > '7' {
		return false, nil
	}
	
	return true, nil
}

// Type returns the PI type
func (v *BSBValidator) Type() string {
	return "BSB"
}

// Normalize returns normalized BSB in XXX-XXX format
func (v *BSBValidator) Normalize(value string) string {
	clean := regexp.MustCompile(`[^\d]`).ReplaceAllString(value, "")
	if len(clean) == 6 {
		return clean[:3] + "-" + clean[3:]
	}
	return clean
}

// ACNValidator validates Australian Company Numbers
type ACNValidator struct{}

// Validate checks if the ACN is valid
func (v *ACNValidator) Validate(value string) (bool, error) {
	// Remove spaces and dashes
	acn := regexp.MustCompile(`[\s\-]`).ReplaceAllString(value, "")
	
	// Must be 9 digits
	if len(acn) != 9 {
		return false, nil
	}
	
	// Check all digits
	if !regexp.MustCompile(`^\d{9}$`).MatchString(acn) {
		return false, nil
	}
	
	// ACN uses similar algorithm to ABN but with different weights
	// Weights: 8, 7, 6, 5, 4, 3, 2, 1
	weights := []int{8, 7, 6, 5, 4, 3, 2, 1}
	sum := 0
	
	for i := 0; i < 8; i++ {
		digit, _ := strconv.Atoi(string(acn[i]))
		sum += digit * weights[i]
	}
	
	// Calculate check digit
	remainder := sum % 10
	checkDigit := (10 - remainder) % 10
	
	// Compare with last digit
	lastDigit, _ := strconv.Atoi(string(acn[8]))
	
	return checkDigit == lastDigit, nil
}

// Type returns the PI type
func (v *ACNValidator) Type() string {
	return "ACN"
}

// Normalize returns normalized ACN
func (v *ACNValidator) Normalize(value string) string {
	return regexp.MustCompile(`[^\d]`).ReplaceAllString(value, "")
}

// ValidatorRegistry holds all validators
type ValidatorRegistry struct {
	validators map[string]Validator
}

// Validator interface for all PI validators
type Validator interface {
	Validate(value string) (bool, error)
	Type() string
	Normalize(value string) string
}

// NewValidatorRegistry creates a new validator registry
func NewValidatorRegistry() *ValidatorRegistry {
	registry := &ValidatorRegistry{
		validators: make(map[string]Validator),
	}
	
	// Register all validators
	registry.Register(&TFNValidator{})
	registry.Register(&ABNValidator{})
	registry.Register(&MedicareValidator{})
	registry.Register(&BSBValidator{})
	registry.Register(&ACNValidator{})
	
	return registry
}

// Register adds a validator to the registry
func (r *ValidatorRegistry) Register(v Validator) {
	r.validators[v.Type()] = v
}

// Get returns a validator by type
func (r *ValidatorRegistry) Get(piType string) (Validator, bool) {
	v, ok := r.validators[piType]
	return v, ok
}

// ValidateAll validates a value against all validators
func (r *ValidatorRegistry) ValidateAll(value string) (string, bool) {
	for piType, validator := range r.validators {
		if valid, _ := validator.Validate(value); valid {
			return piType, true
		}
	}
	return "", false
}