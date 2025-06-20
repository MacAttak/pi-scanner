package proximity

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestAustralianPIPatterns tests proximity detection for Australian PI types
func TestAustralianPIPatterns(t *testing.T) {
	detector := NewProximityDetector()

	testCases := []struct {
		name            string
		content         string
		match           string
		startIndex      int
		endIndex        int
		expectedScore   float64
		expectTestData  bool
		expectedContext PIContextType
		expectedReason  string
		piType          string
	}{
		// TFN (Tax File Number) patterns
		{
			name:            "TFN with label",
			content:         "Employee TFN: 123 456 789",
			match:           "123 456 789",
			startIndex:      14,
			endIndex:        25,
			expectedScore:   0.9,
			expectTestData:  false,
			expectedContext: PIContextLabel,
			expectedReason:  "PI context label detected",
			piType:          "TFN",
		},
		{
			name:            "Test TFN data",
			content:         "// Mock TFN: 123 456 789 for testing",
			match:           "123 456 789",
			startIndex:      12,
			endIndex:        23,
			expectedScore:   0.1,
			expectTestData:  true,
			expectedContext: PIContextTest,
			expectedReason:  "test data indicator",
			piType:          "TFN",
		},
		{
			name:            "TFN in SQL query",
			content:         "SELECT * FROM employees WHERE tax_file_number = '123456789'",
			match:           "123456789",
			startIndex:      49,
			endIndex:        58,
			expectedScore:   0.8,
			expectTestData:  false,
			expectedContext: PIContextDatabase,
			expectedReason:  "database query context",
			piType:          "TFN",
		},
		{
			name:            "TFN in form",
			content:         `<input type="text" name="tfn" placeholder="Tax File Number" value="123456789">`,
			match:           "123456789",
			startIndex:      68,
			endIndex:        77,
			expectedScore:   0.8,
			expectTestData:  false,
			expectedContext: PIContextForm,
			expectedReason:  "form field context",
			piType:          "TFN",
		},

		// ABN (Australian Business Number) patterns
		{
			name:            "ABN with label",
			content:         "Company ABN: 12 345 678 901",
			match:           "12 345 678 901",
			startIndex:      13,
			endIndex:        27,
			expectedScore:   0.9,
			expectTestData:  false,
			expectedContext: PIContextLabel,
			expectedReason:  "PI context label detected",
			piType:          "ABN",
		},
		{
			name:            "Sample ABN data",
			content:         "const sampleABN = '12345678901'; // for demo purposes",
			match:           "12345678901",
			startIndex:      19,
			endIndex:        30,
			expectedScore:   0.1,
			expectTestData:  true,
			expectedContext: PIContextTest,
			expectedReason:  "test data indicator",
			piType:          "ABN",
		},
		{
			name:            "ABN in config",
			content:         "business_number=12345678901\ncompany_name=Example Corp",
			match:           "12345678901",
			startIndex:      16,
			endIndex:        27,
			expectedScore:   0.1,
			expectTestData:  true,
			expectedContext: PIContextTest,
			expectedReason:  "test data indicator",
			piType:          "ABN",
		},

		// Medicare Number patterns
		{
			name:            "Medicare with label",
			content:         "Patient Medicare No: 2345 67890 1",
			match:           "2345 67890 1",
			startIndex:      21,
			endIndex:        33,
			expectedScore:   0.9,
			expectTestData:  false,
			expectedContext: PIContextLabel,
			expectedReason:  "PI context label detected",
			piType:          "MEDICARE",
		},
		{
			name:            "Test Medicare data",
			content:         "testMedicareNumber = '234567890'\n// Test data only",
			match:           "234567890",
			startIndex:      22,
			endIndex:        31,
			expectedScore:   0.1,
			expectTestData:  true,
			expectedContext: PIContextTest,
			expectedReason:  "test data indicator",
			piType:          "MEDICARE",
		},
		{
			name:            "Medicare in JSON",
			content:         `{"patient_id": 12345, "medicare_number": "234567890"}`,
			match:           "234567890",
			startIndex:      42,
			endIndex:        51,
			expectedScore:   0.8,
			expectTestData:  false,
			expectedContext: PIContextForm,
			expectedReason:  "structured data context",
			piType:          "MEDICARE",
		},

		// BSB (Bank State Branch) patterns
		{
			name:            "BSB with label",
			content:         "Bank BSB: 123-456",
			match:           "123-456",
			startIndex:      10,
			endIndex:        17,
			expectedScore:   0.9,
			expectTestData:  false,
			expectedContext: PIContextLabel,
			expectedReason:  "PI context label detected",
			piType:          "BSB",
		},
		{
			name:            "BSB in banking context",
			content:         "Transfer to account 98765432 at BSB 123-456",
			match:           "123-456",
			startIndex:      37,
			endIndex:        44,
			expectedScore:   0.9,
			expectTestData:  false,
			expectedContext: PIContextLabel,
			expectedReason:  "PI context label detected",
			piType:          "BSB",
		},
		{
			name:            "Example BSB code",
			content:         "// Example BSB code: 123-456 (Commonwealth Bank)",
			match:           "123-456",
			startIndex:      21,
			endIndex:        28,
			expectedScore:   0.1,
			expectTestData:  true,
			expectedContext: PIContextTest,
			expectedReason:  "test data indicator",
			piType:          "BSB",
		},

		// Australian Phone Number patterns
		{
			name:            "Mobile number with label",
			content:         "Contact mobile: 0412 345 678",
			match:           "0412 345 678",
			startIndex:      16,
			endIndex:        28,
			expectedScore:   0.9,
			expectTestData:  false,
			expectedContext: PIContextLabel,
			expectedReason:  "PI context label detected",
			piType:          "PHONE",
		},
		{
			name:            "Test phone number",
			content:         "const dummyPhone = '0412345678'; // Test contact",
			match:           "0412345678",
			startIndex:      20,
			endIndex:        30,
			expectedScore:   0.1,
			expectTestData:  true,
			expectedContext: PIContextTest,
			expectedReason:  "test data indicator",
			piType:          "PHONE",
		},

		// Critical combinations (as mentioned in PRD)
		{
			name:            "Critical combination - name + TFN",
			content:         "John Smith, TFN: 123 456 789, Address: 123 Main St",
			match:           "123 456 789",
			startIndex:      17,
			endIndex:        28,
			expectedScore:   0.9,
			expectTestData:  false,
			expectedContext: PIContextLabel,
			expectedReason:  "PI context label detected",
			piType:          "TFN",
		},
		{
			name:            "Production log entry",
			content:         "[INFO] Processing payment for customer TFN: 123456789",
			match:           "123456789",
			startIndex:      43,
			endIndex:        52,
			expectedScore:   0.9,
			expectTestData:  false,
			expectedContext: PIContextLabel,
			expectedReason:  "PI context label detected",
			piType:          "TFN",
		},
		{
			name:            "Australian address format",
			content:         "Address: 123 Collins Street, Melbourne VIC 3000",
			match:           "123 Collins Street, Melbourne VIC 3000",
			startIndex:      9,
			endIndex:        47,
			expectedScore:   0.8,
			expectTestData:  false,
			expectedContext: PIContextLabel,
			expectedReason:  "PI context label detected",
			piType:          "ADDRESS",
		},
		{
			name:            "Test Australian address",
			content:         "// Sample address: 456 Example St, Sydney NSW 2000",
			match:           "456 Example St, Sydney NSW 2000",
			startIndex:      18,
			endIndex:        50,
			expectedScore:   0.1,
			expectTestData:  true,
			expectedContext: PIContextTest,
			expectedReason:  "test data indicator",
			piType:          "ADDRESS",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := detector.AnalyzeContext(tc.content, tc.match, tc.startIndex, tc.endIndex)

			// Score should be within reasonable range
			assert.InDelta(t, tc.expectedScore, result.Score, 0.2,
				"Score mismatch for %s. Expected: %.2f, Got: %.2f", tc.name, tc.expectedScore, result.Score)

			assert.Equal(t, tc.expectTestData, result.IsTestData,
				"Test data detection mismatch for %s", tc.name)

			assert.Equal(t, tc.expectedContext, result.Context,
				"Context type mismatch for %s", tc.name)

			// Reason should contain key information
			// Accept any test/mock/sample/example/demo/dummy/fake data indicator
			if tc.expectedReason == "test data indicator" {
				assert.Regexp(t, `(test|mock|sample|example|demo|dummy|fake) data indicator`, result.Reason,
					"Reason should contain test data indicator pattern for %s", tc.name)
			} else {
				assert.Contains(t, result.Reason, tc.expectedReason,
					"Reason should contain expected text for %s", tc.name)
			}
		})
	}
}

// TestAustralianPIContextScoring tests context-specific scoring for Australian regulations
func TestAustralianPIContextScoring(t *testing.T) {
	detector := NewProximityDetector()

	// Test cases based on AU banking regulatory requirements
	testCases := []struct {
		name        string
		content     string
		match       string
		startIndex  int
		endIndex    int
		minScore    float64
		maxScore    float64
		description string
	}{
		{
			name:        "Critical - TFN in production database",
			content:     "UPDATE customers SET tax_file_number = '123456789' WHERE id = 12345",
			match:       "123456789",
			startIndex:  40,
			endIndex:    49,
			minScore:    0.7,
			maxScore:    0.9,
			description: "TFN in production database should have high score",
		},
		{
			name:        "Low risk - Test configuration",
			content:     "# Test configuration\ntest_tfn=123456789\ntest_mode=true",
			match:       "123456789",
			startIndex:  26,
			endIndex:    35,
			minScore:    0.05,
			maxScore:    0.15,
			description: "Test data should have very low score",
		},
		{
			name:        "High risk - Customer form submission",
			content:     `<form action="/submit-application"><input name="tax_file_number" value="123456789"></form>`,
			match:       "123456789",
			startIndex:  70,
			endIndex:    79,
			minScore:    0.7,
			maxScore:    0.9,
			description: "Form submission should have high score",
		},
		{
			name:        "Medium risk - Configuration file",
			content:     "app.config.customer_tfn_validation_pattern=123456789",
			match:       "123456789",
			startIndex:  44,
			endIndex:    53,
			minScore:    0.4,
			maxScore:    0.75,
			description: "Configuration should have medium score",
		},
		{
			name:        "Very low risk - Comment/documentation",
			content:     "// TFN format: 9 digits, example: 123456789 (not real)",
			match:       "123456789",
			startIndex:  34,
			endIndex:    43,
			minScore:    0.05,
			maxScore:    0.25,
			description: "Documentation should have low score",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := detector.AnalyzeContext(tc.content, tc.match, tc.startIndex, tc.endIndex)

			assert.GreaterOrEqual(t, result.Score, tc.minScore,
				"Score too low for %s: %s", tc.name, tc.description)
			assert.LessOrEqual(t, result.Score, tc.maxScore,
				"Score too high for %s: %s", tc.name, tc.description)

			t.Logf("%s: Score %.2f (expected %.2f-%.2f) - %s",
				tc.name, result.Score, tc.minScore, tc.maxScore, result.Reason)
		})
	}
}

// TestAustralianPIProximityKeywords tests Australian-specific proximity keywords
func TestAustralianPIProximityKeywords(t *testing.T) {
	detector := NewProximityDetector()

	australianKeywords := map[string][]string{
		"TFN": {
			"tax file number", "tfn", "australian taxation office", "ato",
			"payroll", "employment", "tax", "withholding",
		},
		"ABN": {
			"australian business number", "abn", "business registration",
			"gst", "company", "business", "enterprise",
		},
		"MEDICARE": {
			"medicare", "health insurance", "medical", "healthcare",
			"patient", "doctor", "hospital", "clinic",
		},
		"BSB": {
			"bank state branch", "bsb", "routing", "branch code",
			"bank", "financial institution", "transfer", "deposit",
		},
	}

	for piType, keywords := range australianKeywords {
		for _, keyword := range keywords {
			t.Run(fmt.Sprintf("%s_proximity_%s", piType, keyword), func(t *testing.T) {
				content := fmt.Sprintf("Customer %s details: 123456789", keyword)
				result := detector.AnalyzeContext(content, "123456789", len(content)-9, len(content))

				// Should detect as non-test data with reasonable score
				assert.False(t, result.IsTestData,
					"Should not detect as test data when %s keyword is present", keyword)
				assert.GreaterOrEqual(t, result.Score, 0.5,
					"Score should be reasonable when %s keyword is present", keyword)
			})
		}
	}
}
