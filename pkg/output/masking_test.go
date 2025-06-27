package output

import (
	"testing"

	"github.com/MacAttak/pi-scanner/pkg/detection"
	"github.com/stretchr/testify/assert"
)

func TestMasker_Mask(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		piType   detection.PIType
		level    MaskingLevel
		expected string
	}{
		// Full masking tests
		{
			name:     "full mask TFN",
			value:    "123456789",
			piType:   detection.PITypeTFN,
			level:    MaskingLevelFull,
			expected: "*********",
		},
		{
			name:     "full mask with structure",
			value:    "123-456-789",
			piType:   detection.PITypeTFN,
			level:    MaskingLevelFull,
			expected: "***-***-***",
		},
		{
			name:     "full mask email",
			value:    "john.doe@example.com",
			piType:   detection.PITypeEmail,
			level:    MaskingLevelFull,
			expected: "****.***@*******.***",
		},

		// Partial masking tests
		{
			name:     "partial mask TFN",
			value:    "123456789",
			piType:   detection.PITypeTFN,
			level:    MaskingLevelPartial,
			expected: "123****89",
		},
		{
			name:     "partial mask Medicare",
			value:    "2234567890",
			piType:   detection.PITypeMedicare,
			level:    MaskingLevelPartial,
			expected: "22******90",
		},
		{
			name:     "partial mask credit card",
			value:    "1234567812345678",
			piType:   detection.PITypeCreditCard,
			level:    MaskingLevelPartial,
			expected: "************5678",
		},
		{
			name:     "partial mask email",
			value:    "john.doe@example.com",
			piType:   detection.PITypeEmail,
			level:    MaskingLevelPartial,
			expected: "jo***@example.com",
		},
		{
			name:     "partial mask phone",
			value:    "0412345678",
			piType:   detection.PITypePhone,
			level:    MaskingLevelPartial,
			expected: "0412****78",
		},
		{
			name:     "partial mask passport",
			value:    "N1234567",
			piType:   detection.PITypePassport,
			level:    MaskingLevelPartial,
			expected: "N*****67",
		},
		{
			name:     "partial mask BSB",
			value:    "123-456",
			piType:   detection.PITypeBSB,
			level:    MaskingLevelPartial,
			expected: "123***",
		},

		// No masking tests
		{
			name:     "no mask TFN",
			value:    "123456789",
			piType:   detection.PITypeTFN,
			level:    MaskingLevelNone,
			expected: "123456789",
		},
		{
			name:     "no mask email",
			value:    "john.doe@example.com",
			piType:   detection.PITypeEmail,
			level:    MaskingLevelNone,
			expected: "john.doe@example.com",
		},

		// Edge cases
		{
			name:     "empty value",
			value:    "",
			piType:   detection.PITypeTFN,
			level:    MaskingLevelPartial,
			expected: "",
		},
		{
			name:     "short value partial",
			value:    "123",
			piType:   detection.PITypeTFN,
			level:    MaskingLevelPartial,
			expected: "***",
		},
		{
			name:     "unknown PI type",
			value:    "somevalue",
			piType:   "UNKNOWN",
			level:    MaskingLevelPartial,
			expected: "s*******e",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			masker := NewMasker(tt.level)
			result := masker.Mask(tt.value, tt.piType)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMasker_MaskFinding(t *testing.T) {
	finding := &detection.Finding{
		Type:          detection.PITypeTFN,
		Match:         "123456789",
		Context:       "The TFN is 123456789 in the system",
		ContextBefore: "User TFN: 123456789",
		ContextAfter:  "TFN 123456789 is stored",
	}

	masker := NewMasker(MaskingLevelPartial)
	masked := masker.MaskFinding(finding)

	// Check that the match is masked
	assert.Equal(t, "123****89", masked.Match)

	// Check that context is masked
	assert.Contains(t, masked.Context, "123****89")
	assert.NotContains(t, masked.Context, "123456789")

	assert.Contains(t, masked.ContextBefore, "123****89")
	assert.NotContains(t, masked.ContextBefore, "123456789")

	assert.Contains(t, masked.ContextAfter, "123****89")
	assert.NotContains(t, masked.ContextAfter, "123456789")

	// Original finding should not be modified
	assert.Equal(t, "123456789", finding.Match)
}

func TestMasker_CustomPattern(t *testing.T) {
	masker := NewMasker(MaskingLevelPartial)

	// Set a custom pattern for TFN
	masker.SetPattern(detection.PITypeTFN, MaskPattern{
		ShowPrefix: 1,
		ShowSuffix: 4,
		MaskChar:   "#",
	})

	result := masker.Mask("123456789", detection.PITypeTFN)
	assert.Equal(t, "1####6789", result)
}

func TestMasker_EmailSpecialHandling(t *testing.T) {
	masker := NewMasker(MaskingLevelPartial)

	tests := []struct {
		email    string
		expected string
	}{
		{"john@example.com", "jo***@example.com"},
		{"a@example.com", "*@example.com"},
		{"admin@company.org", "ad***@company.org"},
		{"not-an-email", "***-**-*****"},
		{"@example.com", "@example.com"},
	}

	for _, tt := range tests {
		t.Run(tt.email, func(t *testing.T) {
			result := masker.Mask(tt.email, detection.PITypeEmail)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMasker_ContextMasking(t *testing.T) {
	masker := NewMasker(MaskingLevelPartial)

	tests := []struct {
		name     string
		context  string
		value    string
		piType   detection.PIType
		expected string
	}{
		{
			name:     "exact match",
			context:  "The TFN is 123456789 in the database",
			value:    "123456789",
			piType:   detection.PITypeTFN,
			expected: "The TFN is 123****89 in the database",
		},
		{
			name:     "with dashes",
			context:  "TFN: 123-456-789 (validated)",
			value:    "123456789",
			piType:   detection.PITypeTFN,
			expected: "TFN: 123****89 (validated)",
		},
		{
			name:     "multiple occurrences",
			context:  "TFN 123456789 appears twice: 123456789",
			value:    "123456789",
			piType:   detection.PITypeTFN,
			expected: "TFN 123****89 appears twice: 123****89",
		},
		{
			name:     "no match",
			context:  "This context has no PI",
			value:    "123456789",
			piType:   detection.PITypeTFN,
			expected: "This context has no PI",
		},
		{
			name:     "empty context",
			context:  "",
			value:    "123456789",
			piType:   detection.PITypeTFN,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := masker.maskContext(tt.context, tt.value, tt.piType)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestValidateMaskingLevel(t *testing.T) {
	tests := []struct {
		input    string
		expected MaskingLevel
		hasError bool
	}{
		{"FULL", MaskingLevelFull, false},
		{"PARTIAL", MaskingLevelPartial, false},
		{"NONE", MaskingLevelNone, false},
		{"invalid", MaskingLevelPartial, true},
		{"", MaskingLevelPartial, true},
		{"full", MaskingLevelPartial, true}, // Case sensitive
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			level, err := ValidateMaskingLevel(tt.input)
			assert.Equal(t, tt.expected, level)
			if tt.hasError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestMaskSensitiveData_Compatibility(t *testing.T) {
	// Test backward compatibility function
	tests := []struct {
		value    string
		piType   string
		expected string
	}{
		{"123456789", "TFN", "123****89"},
		{"john@example.com", "EMAIL", "jo***@example.com"},
		{"1234567812345678", "CREDIT_CARD", "************5678"},
	}

	for _, tt := range tests {
		t.Run(tt.piType, func(t *testing.T) {
			result := MaskSensitiveData(tt.value, tt.piType)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMasker_SetLevel(t *testing.T) {
	masker := NewMasker(MaskingLevelPartial)

	// Test partial masking
	result := masker.Mask("123456789", detection.PITypeTFN)
	assert.Equal(t, "123****89", result)

	// Change to full masking
	masker.SetLevel(MaskingLevelFull)
	result = masker.Mask("123456789", detection.PITypeTFN)
	assert.Equal(t, "*********", result)

	// Change to no masking
	masker.SetLevel(MaskingLevelNone)
	result = masker.Mask("123456789", detection.PITypeTFN)
	assert.Equal(t, "123456789", result)
}

func TestMasker_ComplexValues(t *testing.T) {
	masker := NewMasker(MaskingLevelPartial)

	tests := []struct {
		name     string
		value    string
		piType   detection.PIType
		expected string
	}{
		{
			name:     "address with numbers",
			value:    "123 Main Street, Sydney NSW 2000",
			piType:   detection.PITypeAddress,
			expected: "123Ma*********************",
		},
		{
			name:     "name with special chars",
			value:    "O'Brien-Smith",
			piType:   detection.PITypeName,
			expected: "O*********h",
		},
		{
			name:     "ABN with spaces",
			value:    "12 345 678 901",
			piType:   detection.PITypeABN,
			expected: "12******901",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := masker.Mask(tt.value, tt.piType)
			assert.Equal(t, tt.expected, result)
		})
	}
}
