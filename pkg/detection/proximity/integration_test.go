package proximity

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestBasicIntegration tests that the proximity detector components work together
func TestBasicIntegration(t *testing.T) {
	detector := NewProximityDetector()

	testCases := []struct {
		name           string
		content        string
		match          string
		startIndex     int
		endIndex       int
		expectTestData bool
		expectMinScore float64
		expectMaxScore float64
	}{
		{
			name:           "Clear test data",
			content:        "test SSN: 123-45-6789",
			match:          "123-45-6789",
			startIndex:     10,
			endIndex:       21,
			expectTestData: true,
			expectMinScore: 0.0,
			expectMaxScore: 0.2,
		},
		{
			name:           "Clear PI label",
			content:        "SSN: 123-45-6789",
			match:          "123-45-6789",
			startIndex:     5,
			endIndex:       16,
			expectTestData: false,
			expectMinScore: 0.5,
			expectMaxScore: 1.0,
		},
		{
			name:           "Form field",
			content:        `<input name="ssn" value="123-45-6789">`,
			match:          "123-45-6789",
			startIndex:     25,
			endIndex:       36,
			expectTestData: false,
			expectMinScore: 0.6,
			expectMaxScore: 1.0,
		},
		{
			name:           "Variable assignment",
			content:        "var ssn = '123-45-6789'",
			match:          "123-45-6789",
			startIndex:     11,
			endIndex:       22,
			expectTestData: false,
			expectMinScore: 0.8,
			expectMaxScore: 1.0, // Variable with PI name gets high score
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := detector.AnalyzeContext(tc.content, tc.match, tc.startIndex, tc.endIndex)

			assert.Equal(t, tc.expectTestData, result.IsTestData,
				"Test data detection mismatch for case: %s", tc.name)

			assert.GreaterOrEqual(t, result.Score, tc.expectMinScore,
				"Score too low for case: %s (got %f)", tc.name, result.Score)

			assert.LessOrEqual(t, result.Score, tc.expectMaxScore,
				"Score too high for case: %s (got %f)", tc.name, result.Score)

			assert.NotEmpty(t, result.Reason, "Reason should not be empty")

			// Verify that all components are working
			assert.NotNil(t, result.Structure, "Structure analysis should not be nil")
			assert.NotNil(t, result.Semantic, "Semantic analysis should not be nil")
		})
	}
}

func TestComponentsWork(t *testing.T) {
	// Test that individual components work
	t.Run("PatternMatcher", func(t *testing.T) {
		pm := NewPatternMatcher()
		assert.True(t, pm.ContainsTestDataKeywords("test data"))
		assert.False(t, pm.ContainsTestDataKeywords("production data"))

		labels := pm.FindPIContextLabels("SSN: 123-45-6789")
		assert.NotEmpty(t, labels)
	})

	t.Run("ContextAnalyzer", func(t *testing.T) {
		ca := NewContextAnalyzer()
		before, after := ca.ExtractSurroundingText("hello world test", 6, 11, 5)
		assert.Equal(t, "ello ", before) // Window of 5 from position 6 goes back to position 1
		assert.Equal(t, " test", after)

		distance, found := ca.GetWordProximity("SSN: 123456789", "SSN", 5, 14)
		assert.True(t, found)
		assert.Equal(t, 1, distance)
	})

	t.Run("ProximityDetector", func(t *testing.T) {
		pd := NewProximityDetector()

		// Test basic functionality
		result := pd.AnalyzeContext("SSN: 123-45-6789", "123-45-6789", 5, 16)
		assert.NotNil(t, result)
		assert.Greater(t, result.Score, 0.0)
		assert.NotEmpty(t, result.Reason)
	})
}
