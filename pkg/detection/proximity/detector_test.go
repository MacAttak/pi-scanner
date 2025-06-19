package proximity

import (
	"strings"
	"testing"

	"github.com/MacAttak/pi-scanner/pkg/detection"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProximityDetector_AnalyzeContext(t *testing.T) {
	detector := NewProximityDetector()

	testCases := []struct {
		name           string
		content        string
		match          string
		startIndex     int
		endIndex       int
		expectedScore  float64
		expectedReason string
	}{
		{
			name:           "Test data with test keyword",
			content:        "// Test data: SSN: 123-45-6789",
			match:          "123-45-6789",
			startIndex:     17,
			endIndex:       28,
			expectedScore:  0.1,
			expectedReason: "test data indicator",
		},
		{
			name:           "Example data with example keyword",
			content:        "Example TFN: 123456789 for documentation",
			match:          "123456789",
			startIndex:     13,
			endIndex:       22,
			expectedScore:  0.1,
			expectedReason: "example data indicator",
		},
		{
			name:           "Mock data with mock keyword",
			content:        "const mockMedicare = '2345678901'",
			match:          "2345678901",
			startIndex:     22,
			endIndex:       32,
			expectedScore:  0.1,
			expectedReason: "mock data indicator",
		},
		{
			name:           "Sample data indicator",
			content:        "Sample ABN: 12345678901",
			match:          "12345678901",
			startIndex:     12,
			endIndex:       23,
			expectedScore:  0.1,
			expectedReason: "sample data indicator",
		},
		{
			name:           "Demo data indicator",
			content:        "Demo user email: demo@example.com",
			match:          "demo@example.com",
			startIndex:     17,
			endIndex:       33,
			expectedScore:  0.1,
			expectedReason: "demo data indicator",
		},
		{
			name:           "Variable name vs actual data",
			content:        "var ssn_number = '123-45-6789'",
			match:          "123-45-6789",
			startIndex:     18,
			endIndex:       29,
			expectedScore:  0.3,
			expectedReason: "variable assignment context",
		},
		{
			name:           "PI context label - SSN",
			content:        "SSN: 123-45-6789",
			match:          "123-45-6789",
			startIndex:     5,
			endIndex:       16,
			expectedScore:  0.9,
			expectedReason: "PI context label detected",
		},
		{
			name:           "PI context label - TFN",
			content:        "Tax File Number: 123456789",
			match:          "123456789",
			startIndex:     17,
			endIndex:       26,
			expectedScore:  0.9,
			expectedReason: "PI context label detected",
		},
		{
			name:           "PI context label - Medicare",
			content:        "Medicare No: 2345678901/1",
			match:          "2345678901/1",
			startIndex:     13,
			endIndex:       25,
			expectedScore:  0.9,
			expectedReason: "PI context label detected",
		},
		{
			name:           "Documentation comment",
			content:        "// Customer SSN 123-45-6789 should be encrypted",
			match:          "123-45-6789",
			startIndex:     16,
			endIndex:       27,
			expectedScore:  0.4,
			expectedReason: "documentation context",
		},
		{
			name:           "Code comment",
			content:        "/* Store the TFN 123456789 securely */",
			match:          "123456789",
			startIndex:     17,
			endIndex:       26,
			expectedScore:  0.4,
			expectedReason: "documentation context",
		},
		{
			name:           "Form field context",
			content:        `<input type="text" name="ssn" value="123-45-6789">`,
			match:          "123-45-6789",
			startIndex:     39,
			endIndex:       50,
			expectedScore:  0.8,
			expectedReason: "form field context",
		},
		{
			name:           "Database query context",
			content:        "SELECT * FROM users WHERE ssn = '123-45-6789'",
			match:          "123-45-6789",
			startIndex:     33,
			endIndex:       44,
			expectedScore:  0.8,
			expectedReason: "database query context",
		},
		{
			name:           "Log entry context",
			content:        "INFO: Processing user with TFN: 123456789",
			match:          "123456789",
			startIndex:     32,
			endIndex:       41,
			expectedScore:  0.7,
			expectedReason: "log entry context",
		},
		{
			name:           "Configuration value",
			content:        "default_tfn=123456789",
			match:          "123456789",
			startIndex:     12,
			endIndex:       21,
			expectedScore:  0.6,
			expectedReason: "configuration context",
		},
		{
			name:           "JSON field",
			content:        `{"ssn": "123-45-6789", "name": "John"}`,
			match:          "123-45-6789",
			startIndex:     9,
			endIndex:       20,
			expectedScore:  0.8,
			expectedReason: "structured data context",
		},
		{
			name:           "Regular code context",
			content:        "user.processPayment(123-45-6789)",
			match:          "123-45-6789",
			startIndex:     20,
			endIndex:       31,
			expectedScore:  0.8,
			expectedReason: "production code context",
		},
		{
			name:           "Fake/invalid prefix",
			content:        "fake_ssn = '123-45-6789'",
			match:          "123-45-6789",
			startIndex:     12,
			endIndex:       23,
			expectedScore:  0.1,
			expectedReason: "fake data indicator",
		},
		{
			name:           "Invalid/dummy prefix",
			content:        "dummy_medicare_number = '2345678901'",
			match:          "2345678901",
			startIndex:     25,
			endIndex:       35,
			expectedScore:  0.1,
			expectedReason: "dummy data indicator",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := detector.AnalyzeContext(tc.content, tc.match, tc.startIndex, tc.endIndex)
			
			assert.InDelta(t, tc.expectedScore, result.Score, 0.1, 
				"Expected score %f, got %f for case: %s", tc.expectedScore, result.Score, tc.name)
			assert.Contains(t, result.Reason, tc.expectedReason,
				"Expected reason to contain '%s', got '%s'", tc.expectedReason, result.Reason)
		})
	}
}

func TestProximityDetector_IsTestData(t *testing.T) {
	detector := NewProximityDetector()

	testCases := []struct {
		name           string
		content        string
		match          string
		startIndex     int
		endIndex       int
		expectedResult bool
	}{
		{
			name:           "Test file path",
			content:        "123-45-6789",
			match:          "123-45-6789",
			startIndex:     0,
			endIndex:       11,
			expectedResult: false, // Note: This would be determined by file path analysis
		},
		{
			name:           "Test keyword in content",
			content:        "test SSN: 123-45-6789",
			match:          "123-45-6789",
			startIndex:     10,
			endIndex:       21,
			expectedResult: true,
		},
		{
			name:           "Example keyword",
			content:        "Example: 123-45-6789",
			match:          "123-45-6789",
			startIndex:     9,
			endIndex:       20,
			expectedResult: true,
		},
		{
			name:           "Mock keyword",
			content:        "mockSSN = '123-45-6789'",
			match:          "123-45-6789",
			startIndex:     11,
			endIndex:       22,
			expectedResult: true,
		},
		{
			name:           "Sample keyword",
			content:        "sample data: 123-45-6789",
			match:          "123-45-6789",
			startIndex:     13,
			endIndex:       24,
			expectedResult: true,
		},
		{
			name:           "Demo keyword",
			content:        "demo_ssn = '123-45-6789'",
			match:          "123-45-6789",
			startIndex:     12,
			endIndex:       23,
			expectedResult: true,
		},
		{
			name:           "Fake keyword",
			content:        "fake SSN: 123-45-6789",
			match:          "123-45-6789",
			startIndex:     10,
			endIndex:       21,
			expectedResult: true,
		},
		{
			name:           "Dummy keyword",
			content:        "dummy data 123-45-6789",
			match:          "123-45-6789",
			startIndex:     11,
			endIndex:       22,
			expectedResult: true,
		},
		{
			name:           "Regular production code",
			content:        "user.ssn = '123-45-6789'",
			match:          "123-45-6789",
			startIndex:     12,
			endIndex:       23,
			expectedResult: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := detector.IsTestData("", tc.content, tc.match, tc.startIndex, tc.endIndex)
			assert.Equal(t, tc.expectedResult, result, "Case: %s", tc.name)
		})
	}
}

func TestProximityDetector_IdentifyPIContext(t *testing.T) {
	detector := NewProximityDetector()

	testCases := []struct {
		name             string
		content          string
		match            string
		startIndex       int
		endIndex         int
		expectedContext  PIContextType
		expectedKeywords []string
	}{
		{
			name:             "SSN label context",
			content:          "SSN: 123-45-6789",
			match:            "123-45-6789",
			startIndex:       5,
			endIndex:         16,
			expectedContext:  PIContextLabel,
			expectedKeywords: []string{"SSN"},
		},
		{
			name:             "Tax File Number context",
			content:          "Tax File Number: 123456789",
			match:            "123456789",
			startIndex:       17,
			endIndex:         26,
			expectedContext:  PIContextLabel,
			expectedKeywords: []string{"Tax File Number"},
		},
		{
			name:             "Medicare context",
			content:          "Medicare No: 2345678901",
			match:            "2345678901",
			startIndex:       13,
			endIndex:         23,
			expectedContext:  PIContextLabel,
			expectedKeywords: []string{"Medicare No"},
		},
		{
			name:             "Form field context",
			content:          `<input name="ssn" value="123-45-6789">`,
			match:            "123-45-6789",
			startIndex:       25,
			endIndex:         36,
			expectedContext:  PIContextForm,
			expectedKeywords: []string{"input", "ssn"},
		},
		{
			name:             "Database context",
			content:          "SELECT * FROM users WHERE ssn = '123-45-6789'",
			match:            "123-45-6789",
			startIndex:       33,
			endIndex:         44,
			expectedContext:  PIContextDatabase,
			expectedKeywords: []string{"SELECT", "WHERE", "ssn"},
		},
		{
			name:             "Log context",
			content:          "INFO: User 123-45-6789 logged in",
			match:            "123-45-6789",
			startIndex:       11,
			endIndex:         22,
			expectedContext:  PIContextLog,
			expectedKeywords: []string{"INFO"},
		},
		{
			name:             "Configuration context",
			content:          "default_ssn=123-45-6789",
			match:            "123-45-6789",
			startIndex:       12,
			endIndex:         23,
			expectedContext:  PIContextConfig,
			expectedKeywords: []string{"default_ssn"},
		},
		{
			name:             "Variable assignment context",
			content:          "var ssn = '123-45-6789'",
			match:            "123-45-6789",
			startIndex:       11,
			endIndex:         22,
			expectedContext:  PIContextVariable,
			expectedKeywords: []string{"var", "ssn"},
		},
		{
			name:             "Documentation context",
			content:          "// Customer SSN 123-45-6789 for reference",
			match:            "123-45-6789",
			startIndex:       16,
			endIndex:         27,
			expectedContext:  PIContextDocumentation,
			expectedKeywords: []string{"Customer", "SSN"},
		},
		{
			name:             "Generic production context",
			content:          "processUser(123-45-6789)",
			match:            "123-45-6789",
			startIndex:       12,
			endIndex:         23,
			expectedContext:  PIContextProduction,
			expectedKeywords: []string{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := detector.IdentifyPIContext(tc.content, tc.match, tc.startIndex, tc.endIndex)
			assert.Equal(t, tc.expectedContext, result.Type, "Case: %s", tc.name)
			
			for _, keyword := range tc.expectedKeywords {
				assert.Contains(t, result.Keywords, keyword, 
					"Expected keyword '%s' in %v for case: %s", keyword, result.Keywords, tc.name)
			}
		})
	}
}

func TestProximityDetector_CalculateProximityScore(t *testing.T) {
	detector := NewProximityDetector()

	testCases := []struct {
		name          string
		proximity     int
		contextType   PIContextType
		expectedScore float64
	}{
		{
			name:          "Adjacent to PI label",
			proximity:     1,
			contextType:   PIContextLabel,
			expectedScore: 0.9,
		},
		{
			name:          "Close to PI label",
			proximity:     3,
			contextType:   PIContextLabel,
			expectedScore: 0.8,
		},
		{
			name:          "Far from PI label",
			proximity:     10,
			contextType:   PIContextLabel,
			expectedScore: 0.6,
		},
		{
			name:          "Form field context",
			proximity:     5,
			contextType:   PIContextForm,
			expectedScore: 0.8,
		},
		{
			name:          "Database context",
			proximity:     5,
			contextType:   PIContextDatabase,
			expectedScore: 0.8,
		},
		{
			name:          "Log context",
			proximity:     5,
			contextType:   PIContextLog,
			expectedScore: 0.7,
		},
		{
			name:          "Variable context",
			proximity:     5,
			contextType:   PIContextVariable,
			expectedScore: 0.3,
		},
		{
			name:          "Documentation context",
			proximity:     5,
			contextType:   PIContextDocumentation,
			expectedScore: 0.4,
		},
		{
			name:          "Production context",
			proximity:     5,
			contextType:   PIContextProduction,
			expectedScore: 0.8,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			score := detector.CalculateProximityScore(tc.proximity, tc.contextType)
			assert.InDelta(t, tc.expectedScore, score, 0.1, "Case: %s", tc.name)
		})
	}
}

func TestProximityDetector_IntegrationWithDetection(t *testing.T) {
	detector := NewProximityDetector()

	testCases := []struct {
		name            string
		content         string
		filename        string
		piType          detection.PIType
		expectedFindings int
		expectedScores   []float64
	}{
		{
			name:             "Test file with test data",
			content:          "// Test SSN: 123-45-6789\nconst testTFN = '123456789'",
			filename:         "user_test.go",
			piType:           detection.PITypeTFN,
			expectedFindings: 2,
			expectedScores:   []float64{0.1, 0.1}, // Both should be marked as test data
		},
		{
			name:             "Production file with PI labels",
			content:          "SSN: 123-45-6789\nTax File Number: 987654321",
			filename:         "user.go",
			piType:           detection.PITypeTFN,
			expectedFindings: 2,
			expectedScores:   []float64{0.9, 0.9}, // Both should have high scores
		},
		{
			name:             "Mixed context",
			content:          "// Example SSN: 123-45-6789\nuser.ssn = '987-65-4321'",
			filename:         "user.go",
			piType:           detection.PITypeTFN,
			expectedFindings: 2,
			expectedScores:   []float64{0.1, 0.8}, // Example vs production
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			findings := []ProximityResult{}
			
			// Simulate finding PI matches in content
			// This would normally be done by the main detector
			// For testing, we'll manually create findings
			
			if tc.name == "Test file with test data" {
				findings = append(findings, 
					detector.AnalyzeContext(tc.content, "123-45-6789", 12, 23),
					detector.AnalyzeContext(tc.content, "123456789", 40, 49),
				)
			} else if tc.name == "Production file with PI labels" {
				findings = append(findings,
					detector.AnalyzeContext(tc.content, "123-45-6789", 5, 16),
					detector.AnalyzeContext(tc.content, "987654321", 34, 43),
				)
			} else if tc.name == "Mixed context" {
				findings = append(findings,
					detector.AnalyzeContext(tc.content, "123-45-6789", 15, 26),
					detector.AnalyzeContext(tc.content, "987-65-4321", 40, 51),
				)
			}
			
			require.Len(t, findings, tc.expectedFindings)
			
			for i, expectedScore := range tc.expectedScores {
				assert.InDelta(t, expectedScore, findings[i].Score, 0.1,
					"Finding %d score mismatch for case: %s", i, tc.name)
			}
		})
	}
}

func TestProximityDetector_EdgeCases(t *testing.T) {
	detector := NewProximityDetector()

	testCases := []struct {
		name        string
		content     string
		match       string
		startIndex  int
		endIndex    int
		description string
	}{
		{
			name:        "Empty content",
			content:     "",
			match:       "123-45-6789",
			startIndex:  0,
			endIndex:    11,
			description: "Should handle empty content gracefully",
		},
		{
			name:        "Match at beginning",
			content:     "123-45-6789 is a SSN",
			match:       "123-45-6789",
			startIndex:  0,
			endIndex:    11,
			description: "Should handle match at beginning of content",
		},
		{
			name:        "Match at end",
			content:     "The SSN is 123-45-6789",
			match:       "123-45-6789",
			startIndex:  11,
			endIndex:    22,
			description: "Should handle match at end of content",
		},
		{
			name:        "Very long content",
			content:     strings.Repeat("a", 10000) + "SSN: 123-45-6789",
			match:       "123-45-6789",
			startIndex:  10005,
			endIndex:    10016,
			description: "Should handle very long content efficiently",
		},
		{
			name:        "Special characters",
			content:     "SSN: 123-45-6789 (encrypted: §†∆ø¬)",
			match:       "123-45-6789",
			startIndex:  5,
			endIndex:    16,
			description: "Should handle special characters in context",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Should not panic and should return reasonable results
			result := detector.AnalyzeContext(tc.content, tc.match, tc.startIndex, tc.endIndex)
			assert.NotNil(t, result, tc.description)
			assert.GreaterOrEqual(t, result.Score, 0.0, "Score should be non-negative")
			assert.LessOrEqual(t, result.Score, 1.0, "Score should not exceed 1.0")
		})
	}
}