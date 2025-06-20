package languages

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/MacAttak/pi-scanner/pkg/detection"
)

func TestMultiLanguageTestCases(t *testing.T) {
	// Create detector with default configuration
	detector := detection.NewDetector()
	runner := NewMultiLanguageTestRunner(detector)
	
	// Test each language separately
	t.Run("Java", func(t *testing.T) {
		testLanguage(t, runner, "java")
	})
	
	t.Run("Scala", func(t *testing.T) {
		testLanguage(t, runner, "scala")
	})
	
	t.Run("Python", func(t *testing.T) {
		testLanguage(t, runner, "python")
	})
}

func testLanguage(t *testing.T, runner *MultiLanguageTestRunner, language string) {
	testCases := runner.GetTestCasesByLanguage(language)
	require.NotEmpty(t, testCases, "Should have test cases for %s", language)
	
	ctx := context.Background()
	var results []*MultiLanguageTestResult
	
	// Execute all test cases for this language
	for _, testCase := range testCases {
		t.Run(testCase.ID, func(t *testing.T) {
			result, err := runner.ExecuteTestCase(ctx, testCase)
			require.NoError(t, err, "Test case execution should not fail")
			
			// Log detailed information for failed tests
			if !result.Passed {
				t.Logf("FAILED - %s: %s", testCase.ID, result.FailureReason)
				t.Logf("Expected PI: %v, Actual PI: %v", result.ExpectedDetected, result.ActualDetected)
				t.Logf("PI Type matches: %v, Context correct: %v", result.PITypeMatches, result.ContextCorrect)
				if len(result.Findings) > 0 {
					t.Logf("Findings: %d", len(result.Findings))
					for i, finding := range result.Findings {
						t.Logf("  %d: Type=%s, Match=%s, Confidence=%.2f", i+1, finding.Type, finding.Match, finding.Confidence)
					}
				}
			}
			
			results = append(results, result)
		})
	}
	
	// Generate summary for this language
	summary := GenerateSummary(results)
	
	t.Logf("%s Language Summary:", language)
	t.Logf("  Total tests: %d", summary.TotalTests)
	t.Logf("  Passed: %d (%.1f%%)", summary.PassedTests, summary.PassRate*100)
	t.Logf("  Failed: %d", summary.FailedTests)
	t.Logf("  False positives: %d", summary.FalsePositives)
	t.Logf("  False negatives: %d", summary.FalseNegatives)
	
	// Log failed test cases for analysis
	if len(summary.FailedTestCases) > 0 {
		t.Logf("  Failed test cases:")
		for _, failed := range summary.FailedTestCases {
			t.Logf("    - %s: %s", failed.TestCase.ID, failed.FailureReason)
		}
	}
	
	// Assert minimum pass rate (adjust as needed based on current performance)
	minPassRate := 0.70 // 70% minimum pass rate
	assert.GreaterOrEqual(t, summary.PassRate, minPassRate, 
		"Pass rate should be at least %.1f%% for %s", minPassRate*100, language)
}

func TestMultiLanguageCodeConstructFiltering(t *testing.T) {
	detector := detection.NewDetector()
	runner := NewMultiLanguageTestRunner(detector)
	
	// Test that code constructs are properly filtered out across all languages
	falsePositiveCases := []string{
		"java-false-name-001", "java-false-name-002", "java-false-name-003",
		"scala-false-name-001", "scala-false-name-002", "scala-false-name-003", "scala-false-name-004",
		"python-false-name-001", "python-false-name-002", "python-false-name-003", "python-false-name-004",
	}
	
	ctx := context.Background()
	
	for _, caseID := range falsePositiveCases {
		t.Run(caseID, func(t *testing.T) {
			// Find the test case
			allCases := runner.GetAllTestCases()
			var testCase *MultiLanguageTestCase
			for _, tc := range allCases {
				if tc.ID == caseID {
					testCase = &tc
					break
				}
			}
			require.NotNil(t, testCase, "Test case %s should exist", caseID)
			
			// Execute the test case
			result, err := runner.ExecuteTestCase(ctx, *testCase)
			require.NoError(t, err)
			
			// Should not detect PI in code constructs
			assert.False(t, result.ActualDetected, 
				"Should not detect PI in code construct for case %s: %s", caseID, testCase.Rationale)
			
			if result.ActualDetected {
				t.Logf("Unexpected detection in %s:", caseID)
				for _, finding := range result.Findings {
					t.Logf("  Found: %s (type: %s, confidence: %.2f)", finding.Match, finding.Type, finding.Confidence)
				}
			}
		})
	}
}

func TestMultiLanguageAustralianPIDetection(t *testing.T) {
	detector := detection.NewDetector()
	runner := NewMultiLanguageTestRunner(detector)
	
	// Test Australian PI types across all languages
	australianPITypes := []detection.PIType{
		detection.PITypeTFN,
		detection.PITypeMedicare,
		detection.PITypeABN,
		detection.PITypeBSB,
		detection.PITypeACN,
	}
	
	ctx := context.Background()
	
	for _, piType := range australianPITypes {
		t.Run(string(piType), func(t *testing.T) {
			testCases := runner.GetTestCasesByPIType(piType)
			
			if len(testCases) == 0 {
				t.Skipf("No test cases for PI type %s", piType)
			}
			
			var results []*MultiLanguageTestResult
			for _, testCase := range testCases {
				if testCase.ExpectedPI { // Only test positive cases
					result, err := runner.ExecuteTestCase(ctx, testCase)
					require.NoError(t, err)
					results = append(results, result)
				}
			}
			
			// Calculate detection rate for this PI type
			detected := 0
			for _, result := range results {
				if result.ActualDetected && result.PITypeMatches {
					detected++
				}
			}
			
			if len(results) > 0 {
				detectionRate := float64(detected) / float64(len(results))
				t.Logf("%s detection rate: %.1f%% (%d/%d)", piType, detectionRate*100, detected, len(results))
				
				// Assert minimum detection rate for Australian PI types
				minDetectionRate := 0.80 // 80% minimum
				assert.GreaterOrEqual(t, detectionRate, minDetectionRate,
					"Detection rate for %s should be at least %.1f%%", piType, minDetectionRate*100)
			}
		})
	}
}

func TestMultiLanguageContextFiltering(t *testing.T) {
	detector := detection.NewDetector()
	runner := NewMultiLanguageTestRunner(detector)
	
	// Test that test context cases are properly filtered
	testContextCases := runner.GetTestCasesByContext("test")
	require.NotEmpty(t, testContextCases, "Should have test context cases")
	
	ctx := context.Background()
	
	suppressedCount := 0
	totalTestCases := 0
	
	for _, testCase := range testContextCases {
		if !testCase.ExpectedPI { // Should be suppressed
			result, err := runner.ExecuteTestCase(ctx, testCase)
			require.NoError(t, err)
			
			if !result.ActualDetected {
				suppressedCount++
			} else {
				t.Logf("Test context not properly suppressed in %s: found %d findings", 
					testCase.ID, len(result.Findings))
			}
			totalTestCases++
		}
	}
	
	if totalTestCases > 0 {
		suppressionRate := float64(suppressedCount) / float64(totalTestCases)
		t.Logf("Test context suppression rate: %.1f%% (%d/%d)", 
			suppressionRate*100, suppressedCount, totalTestCases)
		
		// Assert minimum suppression rate for test contexts
		minSuppressionRate := 0.70 // 70% minimum
		assert.GreaterOrEqual(t, suppressionRate, minSuppressionRate,
			"Test context suppression rate should be at least %.1f%%", minSuppressionRate*100)
	}
}

func TestMultiLanguageTestCaseValidation(t *testing.T) {
	runner := NewMultiLanguageTestRunner(nil) // Don't need detector for validation
	allCases := runner.GetAllTestCases()
	
	// Validate test case structure
	seenIDs := make(map[string]bool)
	
	for _, testCase := range allCases {
		// Check for duplicate IDs
		assert.False(t, seenIDs[testCase.ID], "Duplicate test case ID: %s", testCase.ID)
		seenIDs[testCase.ID] = true
		
		// Validate required fields
		assert.NotEmpty(t, testCase.ID, "Test case ID should not be empty")
		assert.NotEmpty(t, testCase.Language, "Language should not be empty")
		assert.NotEmpty(t, testCase.Filename, "Filename should not be empty")
		assert.NotEmpty(t, testCase.Code, "Code should not be empty")
		assert.NotEmpty(t, testCase.Context, "Context should not be empty")
		assert.NotEmpty(t, testCase.Rationale, "Rationale should not be empty")
		
		// Validate language values
		supportedLanguages := GetSupportedLanguages()
		assert.Contains(t, supportedLanguages, testCase.Language, 
			"Language %s should be supported", testCase.Language)
		
		// Validate context values
		supportedContexts := GetSupportedContexts()
		assert.Contains(t, supportedContexts, testCase.Context,
			"Context %s should be supported", testCase.Context)
		
		// Validate filename extensions match language
		switch testCase.Language {
		case "java":
			assert.True(t, strings.HasSuffix(testCase.Filename, ".java"),
				"Java test case %s should have .java extension", testCase.ID)
		case "scala":
			assert.True(t, strings.HasSuffix(testCase.Filename, ".scala"),
				"Scala test case %s should have .scala extension", testCase.ID)
		case "python":
			assert.True(t, strings.HasSuffix(testCase.Filename, ".py"),
				"Python test case %s should have .py extension", testCase.ID)
		}
	}
	
	t.Logf("Validated %d test cases across %d languages", 
		len(allCases), len(GetSupportedLanguages()))
}

func TestMultiLanguageTestCoverage(t *testing.T) {
	runner := NewMultiLanguageTestRunner(nil)
	
	// Ensure we have good coverage across languages and PI types
	for _, language := range GetSupportedLanguages() {
		t.Run(language, func(t *testing.T) {
			cases := runner.GetTestCasesByLanguage(language)
			assert.GreaterOrEqual(t, len(cases), 10, 
				"Should have at least 10 test cases for %s", language)
			
			// Check coverage of PI types
			piTypeCoverage := make(map[detection.PIType]int)
			for _, testCase := range cases {
				piTypeCoverage[testCase.PIType]++
			}
			
			// Should cover at least major Australian PI types
			majorPITypes := []detection.PIType{
				detection.PITypeTFN,
				detection.PITypeMedicare,
				detection.PITypeABN,
				detection.PITypeName,
			}
			
			for _, piType := range majorPITypes {
				assert.Greater(t, piTypeCoverage[piType], 0,
					"Should have test cases for %s in %s", piType, language)
			}
		})
	}
}