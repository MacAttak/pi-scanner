package languages

import (
	"context"
	"fmt"
	"strings"

	"github.com/MacAttak/pi-scanner/pkg/detection"
)

// MultiLanguageTestRunner manages and executes multi-language test cases
type MultiLanguageTestRunner struct {
	detector detection.Detector
}

// NewMultiLanguageTestRunner creates a new test runner
func NewMultiLanguageTestRunner(detector detection.Detector) *MultiLanguageTestRunner {
	return &MultiLanguageTestRunner{
		detector: detector,
	}
}

// GetAllTestCases returns all test cases for all supported languages
func (r *MultiLanguageTestRunner) GetAllTestCases() []MultiLanguageTestCase {
	var allCases []MultiLanguageTestCase
	
	// Add test cases from each language
	allCases = append(allCases, JavaTestCases()...)
	allCases = append(allCases, ScalaTestCases()...)
	allCases = append(allCases, PythonTestCases()...)
	
	return allCases
}

// GetTestCasesByLanguage returns test cases for a specific language
func (r *MultiLanguageTestRunner) GetTestCasesByLanguage(language string) []MultiLanguageTestCase {
	switch strings.ToLower(language) {
	case "java":
		return JavaTestCases()
	case "scala":
		return ScalaTestCases()
	case "python":
		return PythonTestCases()
	default:
		return []MultiLanguageTestCase{}
	}
}

// GetTestCasesByContext returns test cases for a specific context
func (r *MultiLanguageTestRunner) GetTestCasesByContext(context string) []MultiLanguageTestCase {
	allCases := r.GetAllTestCases()
	var filtered []MultiLanguageTestCase
	
	for _, testCase := range allCases {
		if testCase.Context == context {
			filtered = append(filtered, testCase)
		}
	}
	
	return filtered
}

// GetTestCasesByPIType returns test cases for a specific PI type
func (r *MultiLanguageTestRunner) GetTestCasesByPIType(piType detection.PIType) []MultiLanguageTestCase {
	allCases := r.GetAllTestCases()
	var filtered []MultiLanguageTestCase
	
	for _, testCase := range allCases {
		if testCase.PIType == piType {
			filtered = append(filtered, testCase)
		}
	}
	
	return filtered
}

// ExecuteTestCase runs a single test case and returns the result
func (r *MultiLanguageTestRunner) ExecuteTestCase(ctx context.Context, testCase MultiLanguageTestCase) (*MultiLanguageTestResult, error) {
	// Run detection on the test code
	findings, err := r.detector.Detect(ctx, []byte(testCase.Code), testCase.Filename)
	if err != nil {
		return nil, fmt.Errorf("detection failed for test case %s: %w", testCase.ID, err)
	}
	
	result := &MultiLanguageTestResult{
		TestCase: testCase,
		Findings: findings,
	}
	
	// Analyze results
	result.analyzeResults()
	
	return result, nil
}

// ExecuteAllTestCases runs all test cases and returns results
func (r *MultiLanguageTestRunner) ExecuteAllTestCases(ctx context.Context) ([]*MultiLanguageTestResult, error) {
	testCases := r.GetAllTestCases()
	results := make([]*MultiLanguageTestResult, 0, len(testCases))
	
	for _, testCase := range testCases {
		result, err := r.ExecuteTestCase(ctx, testCase)
		if err != nil {
			return nil, err
		}
		results = append(results, result)
	}
	
	return results, nil
}

// MultiLanguageTestResult represents the result of executing a test case
type MultiLanguageTestResult struct {
	TestCase         MultiLanguageTestCase   `json:"test_case"`
	Findings         []detection.Finding     `json:"findings"`
	ExpectedDetected bool                    `json:"expected_detected"`
	ActualDetected   bool                    `json:"actual_detected"`
	Passed           bool                    `json:"passed"`
	FailureReason    string                  `json:"failure_reason,omitempty"`
	PITypeMatches    bool                    `json:"pi_type_matches"`
	ContextCorrect   bool                    `json:"context_correct"`
}

// analyzeResults analyzes the detection results against expectations
func (r *MultiLanguageTestResult) analyzeResults() {
	r.ExpectedDetected = r.TestCase.ExpectedPI
	r.ActualDetected = len(r.Findings) > 0
	
	// Check if detection result matches expectation
	detectionCorrect := r.ExpectedDetected == r.ActualDetected
	
	// If we expected and found PI, check if the type matches
	r.PITypeMatches = true
	if r.ExpectedDetected && r.ActualDetected {
		r.PITypeMatches = false
		for _, finding := range r.Findings {
			if finding.Type == r.TestCase.PIType {
				r.PITypeMatches = true
				break
			}
		}
	}
	
	// Check context appropriateness
	r.ContextCorrect = r.isContextAppropriate()
	
	// Determine if test passed
	r.Passed = detectionCorrect && r.PITypeMatches && r.ContextCorrect
	
	// Set failure reason if test failed
	if !r.Passed {
		if !detectionCorrect {
			if r.ExpectedDetected {
				r.FailureReason = "Expected PI detection but none found"
			} else {
				r.FailureReason = "Unexpected PI detection (false positive)"
			}
		} else if !r.PITypeMatches {
			r.FailureReason = fmt.Sprintf("Expected PI type %s but found different types", r.TestCase.PIType)
		} else if !r.ContextCorrect {
			r.FailureReason = "Context validation failed"
		}
	}
}

// isContextAppropriate checks if the detection is appropriate for the context
func (r *MultiLanguageTestResult) isContextAppropriate() bool {
	// For test contexts, we generally don't want to detect PI (should be filtered)
	if r.TestCase.Context == "test" && r.ActualDetected {
		// Test context should generally not trigger detection
		return false
	}
	
	// For logging contexts, detection is appropriate (security risk)
	if r.TestCase.Context == "logging" && r.ExpectedDetected {
		return r.ActualDetected
	}
	
	// For production contexts, detection should work as expected
	if r.TestCase.Context == "production" {
		return r.ExpectedDetected == r.ActualDetected
	}
	
	// For documentation contexts, usually shouldn't detect (examples only)
	if r.TestCase.Context == "documentation" && r.ActualDetected {
		return false
	}
	
	// Default: rely on basic expectation matching
	return true
}

// MultiLanguageTestSummary provides summary statistics for test results
type MultiLanguageTestSummary struct {
	TotalTests        int                            `json:"total_tests"`
	PassedTests       int                            `json:"passed_tests"`
	FailedTests       int                            `json:"failed_tests"`
	PassRate          float64                        `json:"pass_rate"`
	ByLanguage        map[string]*LanguageSummary    `json:"by_language"`
	ByPIType          map[string]*PITypeSummary      `json:"by_pi_type"`
	ByContext         map[string]*ContextSummary     `json:"by_context"`
	FalsePositives    int                            `json:"false_positives"`
	FalseNegatives    int                            `json:"false_negatives"`
	FailedTestCases   []*MultiLanguageTestResult     `json:"failed_test_cases"`
}

// LanguageSummary provides statistics for a specific language
type LanguageSummary struct {
	TotalTests   int     `json:"total_tests"`
	PassedTests  int     `json:"passed_tests"`
	FailedTests  int     `json:"failed_tests"`
	PassRate     float64 `json:"pass_rate"`
}

// PITypeSummary provides statistics for a specific PI type
type PITypeSummary struct {
	TotalTests      int     `json:"total_tests"`
	PassedTests     int     `json:"passed_tests"`
	FailedTests     int     `json:"failed_tests"`
	PassRate        float64 `json:"pass_rate"`
	DetectionRate   float64 `json:"detection_rate"`
}

// ContextSummary provides statistics for a specific context
type ContextSummary struct {
	TotalTests   int     `json:"total_tests"`
	PassedTests  int     `json:"passed_tests"`
	FailedTests  int     `json:"failed_tests"`
	PassRate     float64 `json:"pass_rate"`
}

// GenerateSummary creates a comprehensive summary of test results
func GenerateSummary(results []*MultiLanguageTestResult) *MultiLanguageTestSummary {
	summary := &MultiLanguageTestSummary{
		TotalTests:      len(results),
		ByLanguage:      make(map[string]*LanguageSummary),
		ByPIType:        make(map[string]*PITypeSummary),
		ByContext:       make(map[string]*ContextSummary),
		FailedTestCases: make([]*MultiLanguageTestResult, 0),
	}
	
	// Initialize counters
	languageStats := make(map[string]*LanguageSummary)
	piTypeStats := make(map[string]*PITypeSummary)
	contextStats := make(map[string]*ContextSummary)
	
	// Process each result
	for _, result := range results {
		if result.Passed {
			summary.PassedTests++
		} else {
			summary.FailedTests++
			summary.FailedTestCases = append(summary.FailedTestCases, result)
			
			// Count false positives and negatives
			if result.ExpectedDetected && !result.ActualDetected {
				summary.FalseNegatives++
			} else if !result.ExpectedDetected && result.ActualDetected {
				summary.FalsePositives++
			}
		}
		
		// Update language statistics
		lang := result.TestCase.Language
		if languageStats[lang] == nil {
			languageStats[lang] = &LanguageSummary{}
		}
		languageStats[lang].TotalTests++
		if result.Passed {
			languageStats[lang].PassedTests++
		} else {
			languageStats[lang].FailedTests++
		}
		
		// Update PI type statistics
		piType := string(result.TestCase.PIType)
		if piTypeStats[piType] == nil {
			piTypeStats[piType] = &PITypeSummary{}
		}
		piTypeStats[piType].TotalTests++
		if result.Passed {
			piTypeStats[piType].PassedTests++
		} else {
			piTypeStats[piType].FailedTests++
		}
		
		// Update context statistics
		context := result.TestCase.Context
		if contextStats[context] == nil {
			contextStats[context] = &ContextSummary{}
		}
		contextStats[context].TotalTests++
		if result.Passed {
			contextStats[context].PassedTests++
		} else {
			contextStats[context].FailedTests++
		}
	}
	
	// Calculate pass rates
	if summary.TotalTests > 0 {
		summary.PassRate = float64(summary.PassedTests) / float64(summary.TotalTests)
	}
	
	// Calculate language pass rates
	for lang, stats := range languageStats {
		if stats.TotalTests > 0 {
			stats.PassRate = float64(stats.PassedTests) / float64(stats.TotalTests)
		}
		summary.ByLanguage[lang] = stats
	}
	
	// Calculate PI type pass rates and detection rates
	for piType, stats := range piTypeStats {
		if stats.TotalTests > 0 {
			stats.PassRate = float64(stats.PassedTests) / float64(stats.TotalTests)
			
			// Calculate detection rate (how often we detect when we should)
			expectedDetections := 0
			actualDetections := 0
			for _, result := range results {
				if string(result.TestCase.PIType) == piType && result.TestCase.ExpectedPI {
					expectedDetections++
					if result.ActualDetected {
						actualDetections++
					}
				}
			}
			if expectedDetections > 0 {
				stats.DetectionRate = float64(actualDetections) / float64(expectedDetections)
			}
		}
		summary.ByPIType[piType] = stats
	}
	
	// Calculate context pass rates
	for context, stats := range contextStats {
		if stats.TotalTests > 0 {
			stats.PassRate = float64(stats.PassedTests) / float64(stats.TotalTests)
		}
		summary.ByContext[context] = stats
	}
	
	return summary
}

// GetSupportedLanguages returns a list of all supported programming languages
func GetSupportedLanguages() []string {
	return []string{"java", "scala", "python"}
}

// GetSupportedContexts returns a list of all test contexts
func GetSupportedContexts() []string {
	return []string{"production", "test", "documentation", "logging", "configuration"}
}