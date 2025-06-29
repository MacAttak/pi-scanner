package validation

import (
	"strings"
)

// CreditCardValidator validates credit card numbers using the Luhn algorithm
type CreditCardValidator struct{}

// NewCreditCardValidator creates a new credit card validator
func NewCreditCardValidator() *CreditCardValidator {
	return &CreditCardValidator{}
}

// Validate checks if the credit card number is valid
func (v *CreditCardValidator) Validate(value string) (bool, error) {
	// Remove all non-digit characters
	cleaned := v.Normalize(value)

	// Check minimum length (most cards have at least 13 digits)
	if len(cleaned) < 13 || len(cleaned) > 19 {
		return false, nil
	}

	// Validate using Luhn algorithm
	return v.luhnCheck(cleaned), nil
}

// Type returns the PI type this validator handles
func (v *CreditCardValidator) Type() string {
	return "CREDIT_CARD"
}

// Normalize returns a normalized credit card number (digits only)
func (v *CreditCardValidator) Normalize(value string) string {
	// Remove all non-digit characters
	var result strings.Builder
	for _, r := range value {
		if r >= '0' && r <= '9' {
			result.WriteRune(r)
		}
	}
	return result.String()
}

// luhnCheck implements the Luhn algorithm
func (v *CreditCardValidator) luhnCheck(number string) bool {
	if len(number) == 0 {
		return false
	}

	sum := 0
	isEven := false

	// Process digits from right to left
	for i := len(number) - 1; i >= 0; i-- {
		digit := int(number[i] - '0')

		if isEven {
			digit *= 2
			if digit > 9 {
				digit = digit - 9
			}
		}

		sum += digit
		isEven = !isEven
	}

	return sum%10 == 0
}

// GetCardType returns the card type based on the number
func (v *CreditCardValidator) GetCardType(number string) string {
	cleaned := v.Normalize(number)
	if len(cleaned) < 1 {
		return "unknown"
	}

	// Check patterns
	switch {
	case strings.HasPrefix(cleaned, "4"):
		return "visa"
	case strings.HasPrefix(cleaned, "51") || strings.HasPrefix(cleaned, "52") ||
		strings.HasPrefix(cleaned, "53") || strings.HasPrefix(cleaned, "54") ||
		strings.HasPrefix(cleaned, "55"):
		return "mastercard"
	case strings.HasPrefix(cleaned, "34") || strings.HasPrefix(cleaned, "37"):
		return "amex"
	case strings.HasPrefix(cleaned, "6011") || strings.HasPrefix(cleaned, "65"):
		return "discover"
	case strings.HasPrefix(cleaned, "35"):
		return "jcb"
	case strings.HasPrefix(cleaned, "30") || strings.HasPrefix(cleaned, "36") ||
		strings.HasPrefix(cleaned, "38"):
		return "diners"
	default:
		return "unknown"
	}
}

// IsTestCard checks if the card number is a known test card
func (v *CreditCardValidator) IsTestCard(number string) bool {
	cleaned := v.Normalize(number)

	// Common test card numbers
	testCards := []string{
		"4111111111111111", // Visa test
		"4012888888881881", // Visa test
		"5105105105105100", // MasterCard test
		"5555555555554444", // MasterCard test
		"378282246310005",  // Amex test
		"371449635398431",  // Amex test
		"6011111111111117", // Discover test
		"3056930009020004", // Diners test
		"3530111333300000", // JCB test
	}

	for _, testCard := range testCards {
		if cleaned == testCard {
			return true
		}
	}

	return false
}
