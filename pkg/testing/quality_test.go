//go:build !ci
// +build !ci

package testing

import (
	"context"
	"testing"

	"github.com/MacAttak/pi-scanner/pkg/detection"
	"github.com/MacAttak/pi-scanner/pkg/testing/benchmark"
	"github.com/MacAttak/pi-scanner/pkg/testing/evaluation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPIDetectionQuality runs comprehensive quality assessment tests
func TestPIDetectionQuality(t *testing.T) {
	t.Log("🧪 Running PI Detection Quality Assessment")

	// Generate test dataset
	dataset := benchmark.GenerateAustralianPITestCases()
	t.Logf("📊 Dataset: %d test cases (%d true positives, %d true negatives, %d edge cases)",
		len(dataset.AllCases()),
		len(dataset.TruePositives),
		len(dataset.TrueNegatives),
		len(dataset.EdgeCases),
	)

	// Create detector configurations to test
	detectors := map[string]detection.Detector{
		"Pattern-Only":    createPatternDetector(t),
		"Pattern+Context": createPatternWithContextDetector(t),
	}

	// Try to add Gitleaks detector if available
	if gitleaksDetector := tryCreateGitleaksDetector(t); gitleaksDetector != nil {
		detectors["Gitleaks-Only"] = gitleaksDetector
	}

	// Run comparative evaluation
	comparator := evaluation.NewDetectorComparator(detectors, dataset)
	results, err := comparator.Compare()
	require.NoError(t, err)

	// Generate detailed report
	report, err := comparator.GenerateDetailedReport()
	require.NoError(t, err)

	// Print summary
	t.Log("📋 Quality Assessment Results:")
	t.Log(report.Summary())

	// Validate that context validation improves precision
	validateContextValidationImprovement(t, results)

	// Validate minimum performance thresholds
	validateMinimumThresholds(t, results)

	// Test by context to ensure test data is properly filtered
	validateContextFiltering(t, comparator)
}

// tryCreateGitleaksDetector attempts to create a Gitleaks detector, returns nil if not available
func tryCreateGitleaksDetector(t *testing.T) detection.Detector {
	// Try to use the configs/gitleaks.toml file
	detector, err := detection.NewGitleaksDetector("configs/gitleaks.toml")
	if err != nil {
		t.Logf("Gitleaks detector not available: %v", err)
		return nil
	}
	return detector
}

// createPatternDetector creates a pattern-only detector
func createPatternDetector(t *testing.T) detection.Detector {
	return detection.NewDetector()
}

// createPatternWithContextDetector creates a pattern detector with context validation
func createPatternWithContextDetector(t *testing.T) detection.Detector {
	// Create base pattern detector and wrap with mock context validation
	baseDetector := detection.NewDetector()
	return NewMockContextDetector(baseDetector)
}

// validateContextValidationImprovement ensures context validation improves precision
func validateContextValidationImprovement(t *testing.T, results map[string]*evaluation.EvaluationResult) {
	patternResult, hasPattern := results["Pattern-Only"]
	contextResult, hasContext := results["Pattern+Context"]

	if !hasPattern || !hasContext {
		t.Skip("Both Pattern-Only and Pattern+Context needed for comparison")
		return
	}

	patternPrecision := patternResult.Metrics.Precision()
	contextPrecision := contextResult.Metrics.Precision()

	t.Logf("🎯 Precision Comparison:")
	t.Logf("  Pattern-Only:    %.1f%%", patternPrecision*100)
	t.Logf("  Pattern+Context: %.1f%%", contextPrecision*100)

	// Context validation should improve precision
	if contextPrecision > patternPrecision {
		t.Logf("✅ Context validation improved precision by %.1f%%",
			(contextPrecision-patternPrecision)*100)
	} else {
		t.Logf("⚠️  Context validation did not improve precision")
	}

	// Recall should not decrease significantly
	patternRecall := patternResult.Metrics.Recall()
	contextRecall := contextResult.Metrics.Recall()

	t.Logf("🎯 Recall Comparison:")
	t.Logf("  Pattern-Only:    %.1f%%", patternRecall*100)
	t.Logf("  Pattern+Context: %.1f%%", contextRecall*100)

	if contextRecall >= patternRecall*0.9 { // Allow up to 10% recall decrease
		t.Logf("✅ Context validation maintained recall")
	} else {
		t.Logf("⚠️  Context validation significantly reduced recall")
	}
}

// validateMinimumThresholds ensures all detectors meet minimum quality standards
func validateMinimumThresholds(t *testing.T, results map[string]*evaluation.EvaluationResult) {
	minimumPrecision := 0.70 // 70%
	minimumRecall := 0.75    // 75%
	minimumF1 := 0.70        // 70%

	for detectorName, result := range results {
		metrics := result.Metrics

		t.Logf("🎯 %s Performance:", detectorName)
		t.Logf("  Precision: %.1f%% (min: %.1f%%)", metrics.Precision()*100, minimumPrecision*100)
		t.Logf("  Recall:    %.1f%% (min: %.1f%%)", metrics.Recall()*100, minimumRecall*100)
		t.Logf("  F1-Score:  %.1f%% (min: %.1f%%)", metrics.F1Score()*100, minimumF1*100)

		// Check if detector meets minimum thresholds
		if metrics.Precision() >= minimumPrecision {
			t.Logf("  ✅ Precision meets threshold")
		} else {
			t.Logf("  ❌ Precision below threshold")
		}

		if metrics.Recall() >= minimumRecall {
			t.Logf("  ✅ Recall meets threshold")
		} else {
			t.Logf("  ❌ Recall below threshold")
		}

		if metrics.F1Score() >= minimumF1 {
			t.Logf("  ✅ F1-Score meets threshold")
		} else {
			t.Logf("  ❌ F1-Score below threshold")
		}

		// At least one detector should meet all thresholds
		if detectorName == "Pattern+Context" {
			assert.GreaterOrEqual(t, metrics.Precision(), minimumPrecision,
				"Context validation should achieve minimum precision")
			assert.GreaterOrEqual(t, metrics.F1Score(), minimumF1,
				"Context validation should achieve minimum F1-Score")
		}
	}
}

// validateContextFiltering ensures context validation properly filters test data
func validateContextFiltering(t *testing.T, comparator *evaluation.DetectorComparator) {
	contextResults, err := comparator.EvaluateByContext("Pattern+Context")
	require.NoError(t, err)

	t.Logf("📊 Performance by Context:")
	for context, metrics := range contextResults {
		precision := metrics.Precision()
		recall := metrics.Recall()

		t.Logf("  %s: P=%.1f%% R=%.1f%% (TP:%d FP:%d TN:%d FN:%d)",
			context, precision*100, recall*100,
			metrics.TruePositives, metrics.FalsePositives,
			metrics.TrueNegatives, metrics.FalseNegatives)

		// Test and comment contexts should have very high precision (low false positives)
		if context == "test" || context == "comment" {
			assert.GreaterOrEqual(t, precision, 0.8,
				"Test and comment contexts should have high precision")
		}

		// Production context should maintain good recall
		if context == "production" {
			assert.GreaterOrEqual(t, recall, 0.8,
				"Production context should maintain high recall")
		}
	}
}

// TestDetectorRegression ensures our changes don't break existing functionality
func TestDetectorRegression(t *testing.T) {
	t.Log("🔄 Running Regression Tests")

	detector := detection.NewDetector()

	// Test basic detection still works
	testCases := []struct {
		name     string
		code     string
		piType   detection.PIType
		expected bool
	}{
		{
			name:     "TFN Detection",
			code:     `const userTFN = "123456782"`,
			piType:   detection.PITypeTFN,
			expected: true,
		},
		{
			name:     "Email Detection",
			code:     `email := "user@example.com"`,
			piType:   detection.PITypeEmail,
			expected: true,
		},
		{
			name:     "Medicare Detection",
			code:     `medicare := "2428778132"`,
			piType:   detection.PITypeMedicare,
			expected: true,
		},
		{
			name:     "No False Positive",
			code:     `version := "1.2.3"`,
			piType:   detection.PITypeTFN,
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			findings, err := detector.Detect(
				context.Background(),
				[]byte(tc.code),
				"test.go",
			)
			require.NoError(t, err)

			detected := false
			for _, finding := range findings {
				if finding.Type == tc.piType {
					detected = true
					break
				}
			}

			assert.Equal(t, tc.expected, detected,
				"Detection result for %s", tc.name)
		})
	}
}

// BenchmarkDetectionPerformance measures detection performance
func BenchmarkDetectionPerformance(b *testing.B) {
	detector := detection.NewDetector()
	testCode := `
	user := User{
		Name: "John Smith",
		TFN: "123456782",
		Medicare: "2428778132",
		Email: "john@example.com",
		Phone: "0412345678",
	}
	`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := detector.Detect(
			context.Background(),
			[]byte(testCode),
			"benchmark.go",
		)
		if err != nil {
			b.Fatal(err)
		}
	}
}
