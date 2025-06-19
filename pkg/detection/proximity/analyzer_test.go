package proximity

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestContextAnalyzer_ExtractSurroundingText(t *testing.T) {
	analyzer := NewContextAnalyzer()

	testCases := []struct {
		name        string
		content     string
		startIndex  int
		endIndex    int
		windowSize  int
		expectedBefore string
		expectedAfter  string
	}{
		{
			name:           "Normal case",
			content:        "This is a test SSN: 123-45-6789 for validation",
			startIndex:     20,
			endIndex:       31,
			windowSize:     10,
			expectedBefore: "test SSN: ",
			expectedAfter:  " for valid",
		},
		{
			name:           "Start of content",
			content:        "123-45-6789 is a test SSN",
			startIndex:     0,
			endIndex:       11,
			windowSize:     10,
			expectedBefore: "",
			expectedAfter:  " is a test",
		},
		{
			name:           "End of content",
			content:        "The test SSN is 123-45-6789",
			startIndex:     16,
			endIndex:       27,
			windowSize:     10,
			expectedBefore: "st SSN is ",
			expectedAfter:  "",
		},
		{
			name:           "Small window",
			content:        "SSN: 123-45-6789",
			startIndex:     5,
			endIndex:       16,
			windowSize:     3,
			expectedBefore: "N: ",
			expectedAfter:  "",
		},
		{
			name:           "Large window",
			content:        "SSN: 123-45-6789",
			startIndex:     5,
			endIndex:       16,
			windowSize:     50,
			expectedBefore: "SSN: ",
			expectedAfter:  "",
		},
		{
			name:           "Multi-line content",
			content:        "Line 1\nSSN: 123-45-6789\nLine 3",
			startIndex:     12,
			endIndex:       23,
			windowSize:     10,
			expectedBefore: "ne 1\nSSN: ",
			expectedAfter:  "\nLine 3",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			before, after := analyzer.ExtractSurroundingText(tc.content, tc.startIndex, tc.endIndex, tc.windowSize)
			assert.Equal(t, tc.expectedBefore, before, "Before text mismatch")
			assert.Equal(t, tc.expectedAfter, after, "After text mismatch")
		})
	}
}

func TestContextAnalyzer_GetWordProximity(t *testing.T) {
	analyzer := NewContextAnalyzer()

	testCases := []struct {
		name           string
		content        string
		targetWord     string
		startIndex     int
		endIndex       int
		expectedDist   int
		expectedFound  bool
	}{
		{
			name:          "Adjacent word",
			content:       "SSN: 123-45-6789",
			targetWord:    "SSN",
			startIndex:    5,
			endIndex:      16,
			expectedDist:  1,
			expectedFound: true,
		},
		{
			name:          "Word with distance",
			content:       "User SSN number is 123-45-6789",
			targetWord:    "SSN",
			startIndex:    19,
			endIndex:      30,
			expectedDist:  3,
			expectedFound: true,
		},
		{
			name:          "Case insensitive",
			content:       "user ssn number is 123-45-6789",
			targetWord:    "SSN",
			startIndex:    19,
			endIndex:      30,
			expectedDist:  3,
			expectedFound: true,
		},
		{
			name:          "Word not found",
			content:       "user data: 123-45-6789",
			targetWord:    "SSN",
			startIndex:    11,
			endIndex:      22,
			expectedDist:  -1,
			expectedFound: false,
		},
		{
			name:          "Multiple occurrences - closest",
			content:       "SSN test data SSN: 123-45-6789",
			targetWord:    "SSN",
			startIndex:    19,
			endIndex:      30,
			expectedDist:  1,
			expectedFound: true,
		},
		{
			name:          "Word after match",
			content:       "123-45-6789 is SSN data",
			targetWord:    "SSN",
			startIndex:    0,
			endIndex:      11,
			expectedDist:  2,
			expectedFound: true,
		},
		{
			name:          "Punctuation separation",
			content:       "SSN, 123-45-6789",
			targetWord:    "SSN",
			startIndex:    5,
			endIndex:      16,
			expectedDist:  1,
			expectedFound: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			distance, found := analyzer.GetWordProximity(tc.content, tc.targetWord, tc.startIndex, tc.endIndex)
			assert.Equal(t, tc.expectedFound, found, "Found mismatch")
			if found {
				assert.Equal(t, tc.expectedDist, distance, "Distance mismatch")
			}
		})
	}
}

func TestContextAnalyzer_CountKeywords(t *testing.T) {
	analyzer := NewContextAnalyzer()

	testCases := []struct {
		name        string
		content     string
		keywords    []string
		expectedCount int
	}{
		{
			name:          "Single keyword match",
			content:       "This is test data with SSN",
			keywords:      []string{"test"},
			expectedCount: 1,
		},
		{
			name:          "Multiple keyword matches",
			content:       "This is test sample mock data",
			keywords:      []string{"test", "sample", "mock"},
			expectedCount: 3,
		},
		{
			name:          "Case insensitive",
			content:       "This is TEST Sample MOCK data",
			keywords:      []string{"test", "sample", "mock"},
			expectedCount: 3,
		},
		{
			name:          "No matches",
			content:       "This is real production data",
			keywords:      []string{"test", "sample", "mock"},
			expectedCount: 0,
		},
		{
			name:          "Partial matches don't count",
			content:       "This is testing data",
			keywords:      []string{"test"},
			expectedCount: 0, // "testing" should not match "test" as a whole word
		},
		{
			name:          "Multiple occurrences of same keyword",
			content:       "test data for test purposes",
			keywords:      []string{"test"},
			expectedCount: 2,
		},
		{
			name:          "Keywords with punctuation",
			content:       "test, sample, and mock data",
			keywords:      []string{"test", "sample", "mock"},
			expectedCount: 3,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			count := analyzer.CountKeywords(tc.content, tc.keywords)
			assert.Equal(t, tc.expectedCount, count, "Keyword count mismatch")
		})
	}
}

func TestContextAnalyzer_AnalyzeStructure(t *testing.T) {
	analyzer := NewContextAnalyzer()

	testCases := []struct {
		name            string
		content         string
		startIndex      int
		endIndex        int
		expectedType    StructureType
		expectedNesting int
	}{
		{
			name:            "JSON object",
			content:         `{"ssn": "123-45-6789", "name": "John"}`,
			startIndex:      9,
			endIndex:        20,
			expectedType:    StructureJSON,
			expectedNesting: 1,
		},
		{
			name:            "Nested JSON",
			content:         `{"user": {"ssn": "123-45-6789"}}`,
			startIndex:      17,
			endIndex:        28,
			expectedType:    StructureJSON,
			expectedNesting: 2,
		},
		{
			name:            "XML element",
			content:         `<ssn>123-45-6789</ssn>`,
			startIndex:      5,
			endIndex:        16,
			expectedType:    StructureXML,
			expectedNesting: 1,
		},
		{
			name:            "HTML input",
			content:         `<input type="text" value="123-45-6789">`,
			startIndex:      26,
			endIndex:        37,
			expectedType:    StructureHTML,
			expectedNesting: 0, // Single tag, no nesting
		},
		{
			name:            "SQL query",
			content:         "SELECT * FROM users WHERE ssn = '123-45-6789'",
			startIndex:      33,
			endIndex:        44,
			expectedType:    StructureSQL,
			expectedNesting: 0,
		},
		{
			name:            "YAML",
			content:         "user:\n  ssn: 123-45-6789\n  name: John",
			startIndex:      13,
			endIndex:        24,
			expectedType:    StructureYAML,
			expectedNesting: 1,
		},
		{
			name:            "Code block",
			content:         "var ssn = '123-45-6789'; // comment",
			startIndex:      11,
			endIndex:        22,
			expectedType:    StructureCode,
			expectedNesting: 0,
		},
		{
			name:            "Plain text",
			content:         "The SSN number 123-45-6789 belongs to user",
			startIndex:      15,
			endIndex:        26,
			expectedType:    StructurePlainText,
			expectedNesting: 0,
		},
		{
			name:            "URL/Query string",
			content:         "https://api.example.com/user?ssn=123-45-6789",
			startIndex:      33,
			endIndex:        44,
			expectedType:    StructureURL,
			expectedNesting: 0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := analyzer.AnalyzeStructure(tc.content, tc.startIndex, tc.endIndex)
			assert.Equal(t, tc.expectedType, result.Type, "Structure type mismatch")
			assert.Equal(t, tc.expectedNesting, result.NestingLevel, "Nesting level mismatch")
		})
	}
}

func TestContextAnalyzer_CalculateContextWindow(t *testing.T) {
	analyzer := NewContextAnalyzer()

	testCases := []struct {
		name           string
		content        string
		startIndex     int
		endIndex       int
		baseWindow     int
		expectedBefore int
		expectedAfter  int
	}{
		{
			name:           "Normal case",
			content:        "This is a test TFN: 123 456 789 for testing",
			startIndex:     20,
			endIndex:       31,
			baseWindow:     10,
			expectedBefore: 20, // Can go to start
			expectedAfter:  13, // Can go to end
		},
		{
			name:           "Near start",
			content:        "123 456 789 is a test TFN",
			startIndex:     0,
			endIndex:       11,
			baseWindow:     10,
			expectedBefore: 0,  // At start
			expectedAfter:  10, // Normal window
		},
		{
			name:           "Near end",
			content:        "Test SSN: 123-45-6789",
			startIndex:     10,
			endIndex:       21,
			baseWindow:     10,
			expectedBefore: 10, // Normal window
			expectedAfter:  0,  // At end
		},
		{
			name:           "Content shorter than window",
			content:        "123-45-6789",
			startIndex:     0,
			endIndex:       11,
			baseWindow:     20,
			expectedBefore: 0,  // At start
			expectedAfter:  0,  // At end
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			before, after := analyzer.CalculateContextWindow(tc.content, tc.startIndex, tc.endIndex, tc.baseWindow)
			assert.Equal(t, tc.expectedBefore, before, "Before window mismatch")
			assert.Equal(t, tc.expectedAfter, after, "After window mismatch")
		})
	}
}

func TestContextAnalyzer_FindNearestKeyword(t *testing.T) {
	analyzer := NewContextAnalyzer()

	testCases := []struct {
		name           string
		content        string
		keywords       []string
		startIndex     int
		endIndex       int
		expectedKeyword string
		expectedDist   int
		expectedFound  bool
	}{
		{
			name:            "Adjacent keyword",
			content:         "SSN: 123-45-6789",
			keywords:        []string{"SSN", "TFN"},
			startIndex:      5,
			endIndex:        16,
			expectedKeyword: "SSN",
			expectedDist:    1,
			expectedFound:   true,
		},
		{
			name:            "Multiple keywords - closest wins",
			content:         "TFN test SSN: 123-45-6789",
			keywords:        []string{"SSN", "TFN"},
			startIndex:      14,
			endIndex:        25,
			expectedKeyword: "SSN",
			expectedDist:    1,
			expectedFound:   true,
		},
		{
			name:            "Case insensitive match",
			content:         "ssn: 123-45-6789",
			keywords:        []string{"SSN"},
			startIndex:      5,
			endIndex:        16,
			expectedKeyword: "SSN",
			expectedDist:    1,
			expectedFound:   true,
		},
		{
			name:            "No keywords found",
			content:         "data: 123-45-6789",
			keywords:        []string{"SSN", "TFN"},
			startIndex:      6,
			endIndex:        17,
			expectedKeyword: "",
			expectedDist:    -1,
			expectedFound:   false,
		},
		{
			name:            "Keyword after match",
			content:         "123-45-6789 is SSN",
			keywords:        []string{"SSN"},
			startIndex:      0,
			endIndex:        11,
			expectedKeyword: "SSN",
			expectedDist:    2,
			expectedFound:   true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			keyword, distance, found := analyzer.FindNearestKeyword(tc.content, tc.keywords, tc.startIndex, tc.endIndex)
			assert.Equal(t, tc.expectedFound, found, "Found mismatch")
			if found {
				assert.Equal(t, tc.expectedKeyword, keyword, "Keyword mismatch")
				assert.Equal(t, tc.expectedDist, distance, "Distance mismatch")
			}
		})
	}
}

func TestContextAnalyzer_AnalyzeSemanticContext(t *testing.T) {
	analyzer := NewContextAnalyzer()

	testCases := []struct {
		name                string
		content             string
		startIndex          int
		endIndex            int
		expectedConfidence  float64
		expectedIndicators  []string
	}{
		{
			name:               "Strong PI context",
			content:            "Customer SSN: 123-45-6789 for verification",
			startIndex:         14,
			endIndex:           25,
			expectedConfidence: 0.9,
			expectedIndicators: []string{"SSN", "customer", "verification"},
		},
		{
			name:               "Test data context",
			content:            "Test SSN: 123-45-6789 for mock data",
			startIndex:         10,
			endIndex:           21,
			expectedConfidence: 0.1,
			expectedIndicators: []string{"test", "mock"},
		},
		{
			name:               "Database context",
			content:            "SELECT * FROM users WHERE ssn = '123-45-6789'",
			startIndex:         33,
			endIndex:           44,
			expectedConfidence: 0.8,
			expectedIndicators: []string{"database", "query", "ssn"},
		},
		{
			name:               "Documentation context",
			content:            "// Example SSN: 123-45-6789 for reference",
			startIndex:         16,
			endIndex:           27,
			expectedConfidence: 0.1,
			expectedIndicators: []string{"example", "documentation"},
		},
		{
			name:               "Form context",
			content:            `<input name="ssn" value="123-45-6789">`,
			startIndex:         25,
			endIndex:           36,
			expectedConfidence: 0.8,
			expectedIndicators: []string{"form", "input", "ssn"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := analyzer.AnalyzeSemanticContext(tc.content, tc.startIndex, tc.endIndex)
			assert.InDelta(t, tc.expectedConfidence, result.Confidence, 0.1, "Confidence mismatch")
			
			for _, indicator := range tc.expectedIndicators {
				assert.Contains(t, result.Indicators, indicator, 
					"Expected indicator '%s' not found in %v", indicator, result.Indicators)
			}
		})
	}
}

func TestContextAnalyzer_EdgeCases(t *testing.T) {
	analyzer := NewContextAnalyzer()

	testCases := []struct {
		name        string
		content     string
		startIndex  int
		endIndex    int
		description string
	}{
		{
			name:        "Empty content",
			content:     "",
			startIndex:  0,
			endIndex:    0,
			description: "Should handle empty content",
		},
		{
			name:        "Invalid indices",
			content:     "test content",
			startIndex:  15,
			endIndex:    20,
			description: "Should handle indices beyond content length",
		},
		{
			name:        "Reversed indices",
			content:     "test content",
			startIndex:  5,
			endIndex:    2,
			description: "Should handle reversed start/end indices",
		},
		{
			name:        "Very long content",
			content:     strings.Repeat("a", 100000) + "SSN: 123-45-6789",
			startIndex:  100005,
			endIndex:    100016,
			description: "Should handle very long content efficiently",
		},
		{
			name:        "Unicode content",
			content:     "用户SSN: 123-45-6789测试",
			startIndex:  7,
			endIndex:    18,
			description: "Should handle unicode characters",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// These should not panic
			assert.NotPanics(t, func() {
				analyzer.ExtractSurroundingText(tc.content, tc.startIndex, tc.endIndex, 10)
				analyzer.GetWordProximity(tc.content, "SSN", tc.startIndex, tc.endIndex)
				analyzer.CountKeywords(tc.content, []string{"test", "SSN"})
				analyzer.AnalyzeStructure(tc.content, tc.startIndex, tc.endIndex)
				analyzer.CalculateContextWindow(tc.content, tc.startIndex, tc.endIndex, 10)
				analyzer.FindNearestKeyword(tc.content, []string{"SSN"}, tc.startIndex, tc.endIndex)
				analyzer.AnalyzeSemanticContext(tc.content, tc.startIndex, tc.endIndex)
			}, tc.description)
		})
	}
}

func TestContextAnalyzer_Performance(t *testing.T) {
	analyzer := NewContextAnalyzer()
	
	// Create large content for performance testing
	largeContent := strings.Repeat("This is test data with various keywords like SSN, TFN, Medicare, and other PI types. ", 1000)
	largeContent += "SSN: 123-45-6789"
	
	startIndex := len(largeContent) - 11
	endIndex := len(largeContent)
	
	// These operations should complete quickly even with large content
	t.Run("Performance test", func(t *testing.T) {
		// Test that operations complete within reasonable time
		result := analyzer.AnalyzeSemanticContext(largeContent, startIndex, endIndex)
		require.NotNil(t, result)
		assert.GreaterOrEqual(t, result.Confidence, 0.0)
		assert.LessOrEqual(t, result.Confidence, 1.0)
		
		_, found := analyzer.GetWordProximity(largeContent, "SSN", startIndex, endIndex)
		assert.True(t, found, "Should find SSN keyword in large content")
		
		structure := analyzer.AnalyzeStructure(largeContent, startIndex, endIndex)
		assert.NotNil(t, structure)
		assert.Equal(t, StructurePlainText, structure.Type)
	})
}