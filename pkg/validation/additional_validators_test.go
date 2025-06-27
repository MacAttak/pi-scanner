package validation

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPassportValidator(t *testing.T) {
	v := &PassportValidator{}

	tests := []struct {
		name     string
		value    string
		expected bool
	}{
		// Valid old format passports
		{"valid old format N", "N1234567", true},
		{"valid old format E", "E1234567", true},
		{"valid old format K", "K1234567", true},
		{"valid old format L", "L1234567", true},
		{"valid old format M", "M1234567", true},
		{"valid old format P", "P1234567", true},
		{"valid old format R", "R1234567", true},
		{"valid old format S", "S1234567", true},
		{"valid old format T", "T1234567", true},
		{"valid old format X", "X1234567", true},
		{"valid old format Z", "Z1234567", true},

		// Valid new format passports
		{"valid new format PA", "PA1234567", true},
		{"valid new format PB", "PB1234567", true},
		{"valid new format PC", "PC1234567", true},
		{"valid new format PD", "PD1234567", true},
		{"valid new format PE", "PE1234567", true},

		// Invalid formats
		{"invalid letter old format", "A1234567", false},
		{"invalid letter old format B", "B1234567", false},
		{"invalid new format", "AB1234567", false},
		{"too short", "N123456", false},
		{"too long", "N12345678", false},
		{"lowercase", "n1234567", false},
		{"with spaces", "N 1234567", true}, // Should be normalized
		{"letters in number part", "N123456A", false},
		{"all letters", "NABCDEFG", false},
		{"all numbers", "12345678", false},
		{"empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid, err := v.Validate(tt.value)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, valid)
		})
	}
}

func TestPassportValidator_Normalize(t *testing.T) {
	v := &PassportValidator{}

	assert.Equal(t, "N1234567", v.Normalize("n1234567"))
	assert.Equal(t, "N1234567", v.Normalize("N 1234567"))
	assert.Equal(t, "PA1234567", v.Normalize("pa1234567"))
}

func TestBankAccountValidator(t *testing.T) {
	v := &BankAccountValidator{}

	tests := []struct {
		name     string
		value    string
		expected bool
	}{
		// Valid accounts
		{"valid 6 digit", "123456", true},
		{"valid 7 digit", "1234567", true},
		{"valid 8 digit", "12345678", true},
		{"valid 9 digit", "123456789", true},
		{"valid 10 digit", "1234567890", true},
		{"valid with spaces", "12 34 56 78", true},
		{"valid with dashes", "123-456-789", true},

		// Invalid accounts
		{"too short", "12345", false},
		{"too long", "12345678901", false},
		{"letters", "12345678A", false},
		{"empty", "", false},
		{"special chars", "123456@789", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid, err := v.Validate(tt.value)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, valid)
		})
	}
}

func TestBankAccountValidator_CheckDigit(t *testing.T) {
	v := &BankAccountValidator{}

	// Test specific 10-digit accounts with known check digits
	tests := []struct {
		name     string
		account  string
		expected bool
	}{
		// Since we're not validating check digits for bank accounts
		// (different banks use different algorithms), all valid length
		// accounts with digits should pass
		{"10 digit account 1", "1234567890", true},
		{"10 digit account 2", "9876543217", true},
		{"10 digit account 3", "1234567899", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid, err := v.Validate(tt.account)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, valid)
		})
	}
}

func TestDriverLicenseValidator(t *testing.T) {
	v := NewDriverLicenseValidator()

	tests := []struct {
		name     string
		value    string
		expected bool
	}{
		// NSW formats
		{"NSW 8 digits", "12345678", true},
		{"NSW 2 letters 6 digits", "AB123456", true},

		// VIC formats
		{"VIC 1 digit", "1", true},
		{"VIC 5 digits", "12345", true},
		{"VIC 10 digits", "1234567890", true},
		{"VIC 9 digits 1 letter", "123456789A", true},

		// QLD format
		{"QLD 9 digits", "123456789", true},

		// SA format
		{"SA letter 6 digits", "S123456", true},

		// WA format
		{"WA 7 digits", "1234567", true},

		// TAS formats
		{"TAS 7 digits", "1234567", true},
		{"TAS 8 alphanumeric", "AB12CD34", true},

		// NT format
		{"NT 10 digits", "1234567890", true},

		// ACT format
		{"ACT 10 digits", "1234567890", true},

		// Invalid formats
		{"too many letters NSW", "ABC12345", true}, // Actually valid for TAS (8 alphanumeric)
		{"invalid SA format", "12345S6", false},
		{"lowercase letters", "ab123456", false},
		{"with spaces", "AB 123 456", true}, // Should be normalized
		{"with dashes", "AB-123-456", true}, // Should be normalized
		{"empty", "", false},
		{"special chars", "AB@123456", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid, err := v.Validate(tt.value)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, valid)
		})
	}
}

func TestDriverLicenseValidator_ValidateForState(t *testing.T) {
	v := NewDriverLicenseValidator()

	tests := []struct {
		name     string
		value    string
		state    string
		expected bool
	}{
		{"NSW valid 8 digits", "12345678", "NSW", true},
		{"NSW valid letters", "AB123456", "NSW", true},
		{"NSW invalid for VIC", "AB123456", "VIC", false},
		{"VIC valid with letter", "123456789A", "VIC", true},
		{"VIC invalid for NSW", "123456789A", "NSW", false},
		{"invalid state", "12345678", "INVALID", false},
		{"lowercase state", "12345678", "nsw", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid, err := v.ValidateForState(tt.value, tt.state)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, valid)
		})
	}
}

func TestARBNValidator(t *testing.T) {
	v := &ARBNValidator{}

	tests := []struct {
		name     string
		value    string
		expected bool
	}{
		// Valid ARBNs (using same check digit algorithm as ACN)
		{"valid ARBN 1", "000000019", true}, // Known valid ACN/ARBN
		{"valid ARBN 2", "010499966", true}, // Another valid example
		{"valid with spaces", "000 000 019", true},
		{"valid with dashes", "000-000-019", true},

		// Invalid ARBNs
		{"invalid check digit", "123456789", false},
		{"too short", "12345678", false},
		{"too long", "1234567890", false},
		{"letters", "12345678A", false},
		{"empty", "", false},
		{"all zeros", "000000000", true}, // Actually valid ARBN (check digit is correct)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid, err := v.Validate(tt.value)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, valid)
		})
	}
}

func TestARBNValidator_CheckDigitAlgorithm(t *testing.T) {
	v := &ARBNValidator{}

	// Test the check digit algorithm specifically
	// ARBN uses same algorithm as ACN
	// Example: 12 345 678 5
	// Weights: 8  7  6  5  4  3  2  1
	// Sum: 1*8 + 2*7 + 3*6 + 4*5 + 5*4 + 6*3 + 7*2 + 8*1 = 120
	// Check digit: (10 - (120 % 10)) % 10 = 0, but last digit is 5
	// This means we need a different example

	// Let's use: 000 000 019
	// Sum: 0*8 + 0*7 + 0*6 + 0*5 + 0*4 + 0*3 + 1*2 + 9*1 = 11
	// Check digit: (10 - (11 % 10)) % 10 = 9
	arbn := "000000019"
	valid, err := v.Validate(arbn)
	assert.NoError(t, err)
	assert.True(t, valid, "ARBN 000000019 should be valid")
}

func TestValidatorRegistry_WithAdditionalValidators(t *testing.T) {
	registry := NewValidatorRegistry()

	// Check that all validators are registered
	validators := []string{
		"TFN", "ABN", "MEDICARE", "BSB", "ACN",
		"PASSPORT", "BANK_ACCOUNT", "DRIVER_LICENSE", "ARBN",
	}

	for _, validatorType := range validators {
		t.Run(validatorType, func(t *testing.T) {
			v, exists := registry.Get(validatorType)
			require.True(t, exists, "Validator %s should be registered", validatorType)
			assert.NotNil(t, v)
			assert.Equal(t, validatorType, v.Type())
		})
	}
}

func TestValidatorRegistry_ValidateAll_WithNewTypes(t *testing.T) {
	registry := NewValidatorRegistry()

	tests := []struct {
		value        string
		expectedType string
		shouldMatch  bool
	}{
		// Existing types
		{"123456782", "TFN", true},
		{"51824753556", "ABN", true},
		{"2123456701", "MEDICARE", true},
		{"123-456", "BSB", true},
		{"000000019", "ACN", true},

		// New types
		{"N1234567", "PASSPORT", true},
		{"PA1234567", "PASSPORT", true},
		{"123456", "BANK_ACCOUNT", true},
		{"12345678", "BANK_ACCOUNT", true}, // 8-digit bank account number
		{"S123456", "DRIVER_LICENSE", true},
		{"000000019", "ARBN", true}, // Same as ACN - will match first

		// Invalid
		{"invalid", "", false},
		{"", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			piTypes, valid := registry.ValidateAll(tt.value)
			if tt.shouldMatch {
				assert.True(t, valid)
				// Verify the expected type is in the matches
				assert.Contains(t, piTypes, tt.expectedType,
					"Expected %s to be identified as %s, but got %v",
					tt.value, tt.expectedType, piTypes)
			} else {
				assert.False(t, valid)
				assert.Empty(t, piTypes)
			}
		})
	}
}
