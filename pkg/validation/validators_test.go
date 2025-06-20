package validation

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTFNValidator(t *testing.T) {
	validator := &TFNValidator{}

	tests := []struct {
		name       string
		value      string
		expected   bool
		normalized string
	}{
		// Valid TFNs (using algorithm to generate valid ones)
		{
			name:       "valid TFN - 123456782",
			value:      "123456782",
			expected:   true,
			normalized: "123456782",
		},
		{
			name:       "valid TFN with spaces",
			value:      "123 456 782",
			expected:   true,
			normalized: "123456782",
		},
		{
			name:       "valid TFN with dashes",
			value:      "123-456-782",
			expected:   true,
			normalized: "123456782",
		},
		{
			name:       "valid TFN - 876543210",
			value:      "876543210",
			expected:   true,
			normalized: "876543210",
		},
		// Invalid TFNs
		{
			name:     "invalid checksum",
			value:    "123456789",
			expected: false,
		},
		{
			name:     "too short",
			value:    "12345678",
			expected: false,
		},
		{
			name:     "too long",
			value:    "1234567890",
			expected: false,
		},
		{
			name:     "contains letters",
			value:    "12345678a",
			expected: false,
		},
		{
			name:     "all zeros",
			value:    "000000000",
			expected: true, // Actually passes checksum: 0*1+0*4+...+0*10 = 0, 0%11=0
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid, err := validator.Validate(tt.value)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, valid)

			if tt.normalized != "" {
				assert.Equal(t, tt.normalized, validator.Normalize(tt.value))
			}
		})
	}
}

func TestABNValidator(t *testing.T) {
	validator := &ABNValidator{}

	tests := []struct {
		name     string
		value    string
		expected bool
	}{
		// Valid ABNs (real examples from public companies)
		{
			name:     "valid ABN - Telstra",
			value:    "33051775556",
			expected: true,
		},
		{
			name:     "valid ABN with spaces",
			value:    "33 051 775 556",
			expected: true,
		},
		{
			name:     "valid ABN - Commonwealth Bank",
			value:    "48123123124",
			expected: true,
		},
		{
			name:     "valid ABN - Woolworths",
			value:    "88000014675",
			expected: true,
		},
		// Invalid ABNs
		{
			name:     "invalid checksum",
			value:    "12345678901",
			expected: false,
		},
		{
			name:     "too short",
			value:    "1234567890",
			expected: false,
		},
		{
			name:     "too long",
			value:    "123456789012",
			expected: false,
		},
		{
			name:     "contains letters",
			value:    "1234567890a",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid, err := validator.Validate(tt.value)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, valid, "ABN validation failed for %s", tt.value)
		})
	}
}

func TestMedicareValidator(t *testing.T) {
	validator := &MedicareValidator{}

	tests := []struct {
		name     string
		value    string
		expected bool
	}{
		// Valid Medicare numbers (generated using algorithm)
		{
			name:     "valid Medicare - starts with 2",
			value:    "2123456701",
			expected: true,
		},
		{
			name:     "valid Medicare with IRN",
			value:    "2123456701/1",
			expected: true,
		},
		{
			name:     "valid Medicare - starts with 3",
			value:    "3123456711", // Corrected checksum
			expected: true,
		},
		{
			name:     "valid Medicare with spaces",
			value:    "2123 45670 1",
			expected: true,
		},
		// Invalid Medicare numbers
		{
			name:     "invalid first digit (1)",
			value:    "1123456701",
			expected: false,
		},
		{
			name:     "invalid first digit (7)",
			value:    "7123456701",
			expected: false,
		},
		{
			name:     "invalid checksum",
			value:    "2123456789",
			expected: false,
		},
		{
			name:     "too short",
			value:    "212345670",
			expected: false,
		},
		{
			name:     "too long",
			value:    "212345670123", // 12 digits
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid, err := validator.Validate(tt.value)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, valid)
		})
	}
}

func TestBSBValidator(t *testing.T) {
	validator := &BSBValidator{}

	tests := []struct {
		name       string
		value      string
		expected   bool
		normalized string
	}{
		// Valid BSBs (real bank branches)
		{
			name:       "valid BSB - Commonwealth Bank Sydney",
			value:      "062-000",
			expected:   true,
			normalized: "062-000",
		},
		{
			name:       "valid BSB - ANZ Melbourne",
			value:      "013-006",
			expected:   true,
			normalized: "013-006",
		},
		{
			name:       "valid BSB without dash",
			value:      "062000",
			expected:   true,
			normalized: "062-000",
		},
		{
			name:       "valid BSB - Westpac Brisbane",
			value:      "034-002",
			expected:   true,
			normalized: "034-002",
		},
		// Invalid BSBs
		{
			name:     "invalid state digit (0)",
			value:    "060-000",
			expected: false,
		},
		{
			name:     "invalid state digit (1)",
			value:    "061-000",
			expected: false,
		},
		{
			name:     "invalid state digit (8)",
			value:    "068-000",
			expected: false,
		},
		{
			name:     "too short",
			value:    "06200",
			expected: false,
		},
		{
			name:     "too long",
			value:    "0620001",
			expected: false,
		},
		{
			name:     "contains letters",
			value:    "06A-000",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid, err := validator.Validate(tt.value)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, valid)

			if tt.normalized != "" && tt.expected {
				assert.Equal(t, tt.normalized, validator.Normalize(tt.value))
			}
		})
	}
}

func TestACNValidator(t *testing.T) {
	validator := &ACNValidator{}

	tests := []struct {
		name     string
		value    string
		expected bool
	}{
		// Valid ACNs (real company numbers)
		{
			name:     "valid ACN - BHP",
			value:    "004028077",
			expected: true,
		},
		{
			name:     "valid ACN with spaces",
			value:    "004 028 077",
			expected: true,
		},
		{
			name:     "valid ACN - Qantas",
			value:    "009661901",
			expected: true,
		},
		// Invalid ACNs
		{
			name:     "invalid checksum",
			value:    "004028078",
			expected: false,
		},
		{
			name:     "too short",
			value:    "00402807",
			expected: false,
		},
		{
			name:     "too long",
			value:    "0040280771",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid, err := validator.Validate(tt.value)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, valid)
		})
	}
}

func TestValidatorRegistry(t *testing.T) {
	registry := NewValidatorRegistry()

	t.Run("registry has all validators", func(t *testing.T) {
		validators := []string{"TFN", "ABN", "MEDICARE", "BSB", "ACN"}
		for _, vType := range validators {
			validator, ok := registry.Get(vType)
			assert.True(t, ok, "Validator %s should be registered", vType)
			assert.NotNil(t, validator)
		}
	})

	t.Run("ValidateAll identifies correct type", func(t *testing.T) {
		tests := []struct {
			value        string
			expectedType string
			shouldMatch  bool
		}{
			{
				value:        "123456782", // Valid TFN
				expectedType: "TFN",
				shouldMatch:  true,
			},
			{
				value:        "33051775556", // Valid ABN (Telstra)
				expectedType: "ABN",
				shouldMatch:  true,
			},
			{
				value:        "2123456701", // Valid Medicare
				expectedType: "MEDICARE",
				shouldMatch:  true,
			},
			{
				value:        "062-000", // Valid BSB
				expectedType: "BSB",
				shouldMatch:  true,
			},
			{
				value:        "004028077", // Valid ACN
				expectedType: "ACN",
				shouldMatch:  true,
			},
			{
				value:        "invalid123",
				expectedType: "",
				shouldMatch:  false,
			},
		}

		for _, tt := range tests {
			piType, valid := registry.ValidateAll(tt.value)
			assert.Equal(t, tt.shouldMatch, valid)
			if tt.shouldMatch {
				assert.Equal(t, tt.expectedType, piType)
			}
		}
	})
}

// TestRealWorldFormats validates that our patterns match real-world formats
func TestRealWorldFormats(t *testing.T) {
	// Test various real-world formatting variations
	tfnValidator := &TFNValidator{}

	t.Run("TFN format variations", func(t *testing.T) {
		validFormats := []string{
			"123456782",   // No formatting
			"123 456 782", // Space separated
			"123-456-782", // Dash separated
			"123.456.782", // Dot separated (less common but seen)
			" 123456782 ", // With surrounding spaces
		}

		for _, format := range validFormats {
			normalized := tfnValidator.Normalize(format)
			assert.Equal(t, "123456782", normalized, "Failed to normalize: %s", format)
		}
	})

	abnValidator := &ABNValidator{}

	t.Run("ABN format variations", func(t *testing.T) {
		validFormats := []string{
			"33051775556",    // No formatting
			"33 051 775 556", // Standard format
			"33-051-775-556", // Dash separated
			"33051775556",    // Continuous
		}

		for _, format := range validFormats {
			normalized := abnValidator.Normalize(format)
			assert.Equal(t, "33051775556", normalized, "Failed to normalize: %s", format)
		}
	})
}

// BenchmarkValidators tests the performance of validators
func BenchmarkTFNValidator(b *testing.B) {
	validator := &TFNValidator{}
	tfn := "123456782"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		validator.Validate(tfn)
	}
}

func BenchmarkABNValidator(b *testing.B) {
	validator := &ABNValidator{}
	abn := "33051775556"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		validator.Validate(abn)
	}
}
