package detection

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDetector_Detect(t *testing.T) {
	tests := []struct {
		name         string
		content      string
		expectedPIs  []string
		expectedType []PIType
		description  string
	}{
		// Australian Tax File Number tests
		{
			name:         "detect TFN in code",
			content:      `const customerTFN = "123456789"`,
			expectedPIs:  []string{"123456789"},
			expectedType: []PIType{PITypeTFN},
			description:  "Should detect valid TFN in string literal",
		},
		{
			name:         "detect TFN with formatting",
			content:      `tfn: "123-456-789"`,
			expectedPIs:  []string{"123-456-789"},
			expectedType: []PIType{PITypeTFN},
			description:  "Should detect TFN with dashes",
		},
		{
			name:         "detect TFN with spaces",
			content:      `user_tfn = "123 456 789"`,
			expectedPIs:  []string{"123 456 789"},
			expectedType: []PIType{PITypeTFN},
			description:  "Should detect TFN with spaces",
		},
		{
			name:         "ignore invalid TFN length",
			content:      `id = "12345678"`, // 8 digits
			expectedPIs:  []string{},
			expectedType: []PIType{},
			description:  "Should not detect 8-digit numbers as TFN",
		},
		// Medicare tests
		{
			name:         "detect Medicare number",
			content:      `medicare: "2123456701"`,
			expectedPIs:  []string{"2123456701"},
			expectedType: []PIType{PITypeMedicare},
			description:  "Should detect valid Medicare number",
		},
		{
			name:         "detect Medicare with IRN",
			content:      `healthCard = "2123456701/1"`,
			expectedPIs:  []string{"2123456701/1"},
			expectedType: []PIType{PITypeMedicare},
			description:  "Should detect Medicare with Individual Reference Number",
		},
		{
			name:         "reject Medicare with invalid first digit",
			content:      `card = "1123456701"`, // First digit must be 2-6
			expectedPIs:  []string{},
			expectedType: []PIType{},
			description:  "Should not detect Medicare starting with 1",
		},
		// ABN tests
		{
			name:         "detect ABN",
			content:      `company_abn = "51824753556"`,
			expectedPIs:  []string{"51824753556"},
			expectedType: []PIType{PITypeABN},
			description:  "Should detect valid 11-digit ABN",
		},
		{
			name:         "detect ABN with spaces",
			content:      `ABN: 51 824 753 556`,
			expectedPIs:  []string{"51 824 753 556"},
			expectedType: []PIType{PITypeABN},
			description:  "Should detect ABN with spaces",
		},
		// BSB tests
		{
			name:         "detect BSB",
			content:      `bsb = "062-001"`,
			expectedPIs:  []string{"062-001"},
			expectedType: []PIType{PITypeBSB},
			description:  "Should detect BSB with dash",
		},
		{
			name:         "detect BSB without dash",
			content:      `bank_bsb: "062001"`,
			expectedPIs:  []string{"062001"},
			expectedType: []PIType{PITypeBSB},
			description:  "Should detect BSB without dash",
		},
		// Email tests
		{
			name:         "detect email address",
			content:      `email: "john.smith@example.com"`,
			expectedPIs:  []string{"john.smith@example.com"},
			expectedType: []PIType{PITypeEmail},
			description:  "Should detect valid email",
		},
		// Phone tests
		{
			name:         "detect Australian mobile",
			content:      `mobile: "0412345678"`,
			expectedPIs:  []string{"0412345678"},
			expectedType: []PIType{PITypePhone},
			description:  "Should detect Australian mobile number",
		},
		{
			name:         "detect Australian landline",
			content:      `phone: "(02) 9123 4567"`,
			expectedPIs:  []string{"(02) 9123 4567"},
			expectedType: []PIType{PITypePhone},
			description:  "Should detect formatted landline",
		},
		// Multiple PI in same content
		{
			name: "detect multiple PI types",
			content: `
				customer := Customer{
					Name: "John Smith",
					TFN: "123456789",
					Email: "john@example.com",
				}
			`,
			expectedPIs:  []string{"John Smith", "123456789", "john@example.com"},
			expectedType: []PIType{PITypeName, PITypeTFN, PITypeEmail},
			description:  "Should detect multiple PI types in struct",
		},
		// Edge cases
		{
			name:         "ignore PI in comments",
			content:      `// Example TFN: 123456789`,
			expectedPIs:  []string{"123456789"},
			expectedType: []PIType{PITypeTFN},
			description:  "Should still detect PI in comments (context analysis comes later)",
		},
		{
			name:         "detect PI in JSON",
			content:      `{"tfn": "123456789", "email": "test@example.com"}`,
			expectedPIs:  []string{"123456789", "test@example.com"},
			expectedType: []PIType{PITypeTFN, PITypeEmail},
			description:  "Should detect PI in JSON format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			detector := NewDetector()
			findings, err := detector.Detect(context.Background(), []byte(tt.content), "test.go")

			require.NoError(t, err)
			assert.Len(t, findings, len(tt.expectedPIs), tt.description)

			// Check each expected PI was found
			for i, expectedPI := range tt.expectedPIs {
				if i < len(findings) {
					assert.Equal(t, expectedPI, findings[i].Match)
					assert.Equal(t, tt.expectedType[i], findings[i].Type)
				}
			}
		})
	}
}

func TestDetector_DetectWithContext(t *testing.T) {
	tests := []struct {
		name            string
		content         string
		filename        string
		expectedRiskMod float32
		description     string
	}{
		{
			name:            "reduce risk for test files",
			content:         `tfn := "123456789"`,
			filename:        "customer_test.go",
			expectedRiskMod: 0.1,
			description:     "Should reduce risk for test files",
		},
		{
			name:            "reduce risk for mock files",
			content:         `mockTFN := "123456789"`,
			filename:        "mock_data.go",
			expectedRiskMod: 0.1,
			description:     "Should reduce risk for mock files",
		},
		{
			name:            "normal risk for production files",
			content:         `customerTFN := "123456789"`,
			filename:        "customer.go",
			expectedRiskMod: 1.0,
			description:     "Should maintain normal risk for production files",
		},
		{
			name:            "reduce risk for test directories",
			content:         `tfn := "123456789"`,
			filename:        "test/fixtures/data.go",
			expectedRiskMod: 0.1,
			description:     "Should reduce risk for files in test directories",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			detector := NewDetector()
			findings, err := detector.Detect(context.Background(), []byte(tt.content), tt.filename)

			require.NoError(t, err)
			require.NotEmpty(t, findings)

			// Check risk modification based on context
			for _, finding := range findings {
				assert.Equal(t, tt.expectedRiskMod, finding.ContextModifier, tt.description)
			}
		})
	}
}

func TestDetector_Performance(t *testing.T) {
	// Large content for performance testing
	largeContent := ""
	for i := 0; i < 1000; i++ {
		largeContent += `
			customer := Customer{
				Name: "John Smith",
				TFN: "123456789",
				Email: "john@example.com",
				Phone: "0412345678",
			}
		`
	}

	detector := NewDetector()

	// This should complete quickly even with large content
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	findings, err := detector.Detect(ctx, []byte(largeContent), "large_file.go")

	require.NoError(t, err)
	assert.NotEmpty(t, findings)
}

func BenchmarkDetector_Detect(b *testing.B) {
	content := `
		customer := Customer{
			Name: "John Smith",
			TFN: "123456789",
			Email: "john@example.com",
			Phone: "0412345678",
			Medicare: "2123456701",
			ABN: "51824753556",
		}
	`

	detector := NewDetector()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = detector.Detect(ctx, []byte(content), "bench.go")
	}
}

func BenchmarkDetector_LargeFile(b *testing.B) {
	// Simulate a large file
	content := ""
	for i := 0; i < 100; i++ {
		content += `const data = "Some regular content without PI information that should be skipped quickly"`
	}
	content += `tfn := "123456789"` // One PI in large file

	detector := NewDetector()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = detector.Detect(ctx, []byte(content), "large.go")
	}
}

func TestDetector_DetectWithValidation(t *testing.T) {
	tests := []struct {
		name               string
		content            string
		expectedValid      bool
		expectedConfidence float32
		description        string
	}{
		{
			name:               "valid TFN with checksum",
			content:            `tfn := "123456782"`, // Valid TFN
			expectedValid:      true,
			expectedConfidence: 0.95,
			description:        "Should increase confidence for valid TFN",
		},
		{
			name:               "invalid TFN checksum",
			content:            `tfn := "123456789"`, // Invalid checksum
			expectedValid:      false,
			expectedConfidence: 0.5,
			description:        "Should decrease confidence for invalid TFN",
		},
		{
			name:               "valid ABN - Telstra",
			content:            `abn := "33051775556"`,
			expectedValid:      true,
			expectedConfidence: 0.95,
			description:        "Should validate real ABN",
		},
		{
			name:               "invalid ABN checksum",
			content:            `abn := "12345678901"`,
			expectedValid:      false,
			expectedConfidence: 0.5,
			description:        "Should detect invalid ABN",
		},
		{
			name:               "valid Medicare",
			content:            `medicare := "2123456701"`,
			expectedValid:      true,
			expectedConfidence: 0.95,
			description:        "Should validate Medicare with correct checksum",
		},
		{
			name:               "valid BSB",
			content:            `bsb := "062-000"`,
			expectedValid:      true,
			expectedConfidence: 0.95,
			description:        "Should validate BSB with valid state code",
		},
		{
			name:               "invalid BSB state",
			content:            `bsb := "068-000"`, // Invalid state digit 8
			expectedValid:      false,
			expectedConfidence: 0.5,
			description:        "Should reject BSB with invalid state",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			detector := NewDetector()
			findings, err := detector.Detect(context.Background(), []byte(tt.content), "test.go")

			require.NoError(t, err)
			require.Len(t, findings, 1, "Should find exactly one PI")

			finding := findings[0]
			assert.Equal(t, tt.expectedValid, finding.Validated, tt.description)
			assert.Equal(t, tt.expectedConfidence, finding.Confidence, "Confidence should match expected")

			if !tt.expectedValid && finding.ValidationError == "" {
				t.Errorf("Expected validation error for invalid PI but got none")
			}
		})
	}
}
