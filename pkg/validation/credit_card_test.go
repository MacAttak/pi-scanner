package validation

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreditCardValidator(t *testing.T) {
	validator := NewCreditCardValidator()

	testCases := []struct {
		name     string
		input    string
		expected bool
		cardType string
	}{
		// Valid cards (test numbers that pass Luhn)
		{
			name:     "Valid Visa",
			input:    "4111111111111111",
			expected: true,
			cardType: "visa",
		},
		{
			name:     "Valid Visa with spaces",
			input:    "4111 1111 1111 1111",
			expected: true,
			cardType: "visa",
		},
		{
			name:     "Valid Visa with dashes",
			input:    "4111-1111-1111-1111",
			expected: true,
			cardType: "visa",
		},
		{
			name:     "Valid MasterCard",
			input:    "5105105105105100",
			expected: true,
			cardType: "mastercard",
		},
		{
			name:     "Valid MasterCard with spaces",
			input:    "5105 1051 0510 5100",
			expected: true,
			cardType: "mastercard",
		},
		{
			name:     "Valid Amex",
			input:    "378282246310005",
			expected: true,
			cardType: "amex",
		},
		{
			name:     "Valid Discover",
			input:    "6011111111111117",
			expected: true,
			cardType: "discover",
		},
		{
			name:     "Valid JCB",
			input:    "3530111333300000",
			expected: true,
			cardType: "jcb",
		},
		{
			name:     "Valid Diners",
			input:    "30569309025904",
			expected: true,
			cardType: "diners",
		},

		// Invalid cards
		{
			name:     "Invalid Luhn check",
			input:    "4111111111111112",
			expected: false,
		},
		{
			name:     "Too short",
			input:    "411111111111",
			expected: false,
		},
		{
			name:     "Too long",
			input:    "41111111111111111111",
			expected: false,
		},
		{
			name:     "Non-numeric",
			input:    "4111-abcd-1111-1111",
			expected: false,
		},
		{
			name:     "Empty string",
			input:    "",
			expected: false,
		},
		{
			name:     "Spaces only",
			input:    "    ",
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			valid, err := validator.Validate(tc.input)
			require.NoError(t, err)
			assert.Equal(t, tc.expected, valid, "Validation result mismatch for %s", tc.input)

			// Check card type if valid
			if tc.expected && tc.cardType != "" {
				cardType := validator.GetCardType(tc.input)
				assert.Equal(t, tc.cardType, cardType, "Card type mismatch for %s", tc.input)
			}
		})
	}
}

func TestCreditCardValidator_Normalize(t *testing.T) {
	validator := NewCreditCardValidator()

	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "With spaces",
			input:    "4111 1111 1111 1111",
			expected: "4111111111111111",
		},
		{
			name:     "With dashes",
			input:    "4111-1111-1111-1111",
			expected: "4111111111111111",
		},
		{
			name:     "With mixed separators",
			input:    "4111 1111-1111.1111",
			expected: "4111111111111111",
		},
		{
			name:     "Already normalized",
			input:    "4111111111111111",
			expected: "4111111111111111",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := validator.Normalize(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestCreditCardValidator_IsTestCard(t *testing.T) {
	validator := NewCreditCardValidator()

	testCards := []string{
		"4111111111111111",
		"4012888888881881",
		"5105105105105100",
		"5555555555554444",
		"378282246310005",
		"371449635398431",
		"6011111111111117",
		"3056930009020004",
		"3530111333300000",
	}

	for _, card := range testCards {
		t.Run(card, func(t *testing.T) {
			assert.True(t, validator.IsTestCard(card), "Expected %s to be recognized as test card", card)
		})
	}

	// Non-test cards
	nonTestCards := []string{
		"4111111111111112", // Invalid Luhn
		"5105105105105101", // Different number
		"1234567890123456", // Random
	}

	for _, card := range nonTestCards {
		t.Run(card, func(t *testing.T) {
			assert.False(t, validator.IsTestCard(card), "Expected %s to NOT be recognized as test card", card)
		})
	}
}

func TestLuhnAlgorithm(t *testing.T) {
	validator := NewCreditCardValidator()

	// Test specific Luhn algorithm cases
	testCases := []struct {
		number   string
		expected bool
	}{
		{"79927398713", true},       // Valid Luhn
		{"79927398714", false},      // Invalid Luhn
		{"4532015112830366", true},  // Valid Visa
		{"4532015112830367", false}, // Invalid Visa
		{"6011514433546201", true},  // Valid Discover
		{"6011514433546202", false}, // Invalid Discover
	}

	for _, tc := range testCases {
		t.Run(tc.number, func(t *testing.T) {
			result := validator.luhnCheck(tc.number)
			assert.Equal(t, tc.expected, result, "Luhn check failed for %s", tc.number)
		})
	}
}
