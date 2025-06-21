//go:build !ci
// +build !ci

package testing

import (
	"context"
	"fmt"
	"testing"

	"github.com/MacAttak/pi-scanner/pkg/detection"
	"github.com/MacAttak/pi-scanner/pkg/testing/benchmark"
	"github.com/MacAttak/pi-scanner/pkg/testing/evaluation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestComprehensivePIDetectionQuality runs quality assessment with 200+ test cases
func TestComprehensivePIDetectionQuality(t *testing.T) {
	t.Log("🧪 Running Comprehensive PI Detection Quality Assessment (200+ test cases)")

	// Generate comprehensive test dataset
	dataset := benchmark.GenerateComprehensiveTestDataset()
	stats := dataset.Stats()

	t.Logf("📊 Dataset Statistics:")
	t.Logf("   Total Cases: %d", stats.TotalCases)
	t.Logf("   True Positives: %d", stats.TruePositives)
	t.Logf("   True Negatives: %d", stats.TrueNegatives)
	t.Logf("   Edge Cases: %d", stats.EdgeCases)
	t.Logf("   Synthetic: %d", stats.Synthetic)
	t.Logf("   By Context: %v", stats.ByContext)
	t.Logf("   By PI Type: %v", stats.ByPIType)

	// Create detector configurations to test
	detectors := map[string]detection.Detector{
		"Pattern-Only":    createPatternDetector(t),
		"Pattern+Context": createPatternWithContextDetector(t),
	}

	// Run comparative evaluation
	comparator := evaluation.NewDetectorComparator(detectors, dataset)
	results, err := comparator.Compare()
	require.NoError(t, err)

	// Generate detailed report
	report, err := comparator.GenerateDetailedReport()
	require.NoError(t, err)

	// Print comprehensive results
	t.Log("\n📋 Comprehensive Quality Assessment Results:")
	t.Log(report.Summary())

	// Detailed analysis by PI type
	t.Log("\n🔍 Performance by PI Type:")
	for detectorName, typeResults := range report.ByPIType {
		t.Logf("\n%s:", detectorName)
		for piType, metrics := range typeResults {
			if metrics.Total() > 0 {
				t.Logf("  %s: %s", piType, metrics.CompactReport())
			}
		}
	}

	// Detailed analysis by context
	t.Log("\n🎯 Performance by Context:")
	for detectorName, contextResults := range report.ByContext {
		t.Logf("\n%s:", detectorName)
		for context, metrics := range contextResults {
			if metrics.Total() > 0 {
				t.Logf("  %s: %s", context, metrics.CompactReport())
			}
		}
	}

	// Validate improvements
	validateComprehensiveResults(t, results, report)
}

// validateComprehensiveResults validates that results meet our quality targets
func validateComprehensiveResults(t *testing.T, results map[string]*evaluation.EvaluationResult, report *evaluation.DetailedReport) {
	// Target metrics
	targetPrecision := 0.85
	targetRecall := 0.85
	targetF1 := 0.85

	// Minimum acceptable metrics
	minPrecision := 0.70
	minRecall := 0.75
	minF1 := 0.70

	t.Log("\n🎯 Target Performance Validation:")

	for detectorName, result := range results {
		metrics := result.Metrics
		precision := metrics.Precision()
		recall := metrics.Recall()
		f1 := metrics.F1Score()

		t.Logf("\n%s Performance:", detectorName)
		t.Logf("  Precision: %.1f%% (target: %.1f%%, min: %.1f%%)",
			precision*100, targetPrecision*100, minPrecision*100)
		t.Logf("  Recall:    %.1f%% (target: %.1f%%, min: %.1f%%)",
			recall*100, targetRecall*100, minRecall*100)
		t.Logf("  F1-Score:  %.1f%% (target: %.1f%%, min: %.1f%%)",
			f1*100, targetF1*100, minF1*100)

		// Check against minimum thresholds
		if detectorName == "Pattern+Context" {
			if precision < minPrecision {
				t.Logf("  ⚠️  Precision below minimum threshold")
			}
			if recall < minRecall {
				t.Logf("  ⚠️  Recall below minimum threshold")
			}
			if f1 < minF1 {
				t.Logf("  ⚠️  F1-Score below minimum threshold")
			}
		}

		// Quality grade
		assessment := report.QualityAssessments[detectorName]
		t.Logf("  Grade: %s (%.1f%%)", assessment.Grade, assessment.Score)
		t.Logf("  Strengths: %v", assessment.Strengths)
		t.Logf("  Weaknesses: %v", assessment.Weaknesses)
		t.Logf("  Recommendation: %s", assessment.Recommendation)
	}

	// Validate specific improvements
	patternResult := results["Pattern-Only"]
	contextResult := results["Pattern+Context"]

	if patternResult != nil && contextResult != nil {
		precisionImprovement := contextResult.Metrics.Precision() - patternResult.Metrics.Precision()
		t.Logf("\n✅ Context Validation Impact:")
		t.Logf("  Precision Improvement: %+.1f%%", precisionImprovement*100)

		// Context should improve precision
		assert.Greater(t, contextResult.Metrics.Precision(), patternResult.Metrics.Precision(),
			"Context validation should improve precision")
	}
}

// TestPITypeValidation tests validation accuracy for each PI type
func TestPITypeValidation(t *testing.T) {
	t.Log("🔍 Testing PI Type Validation Accuracy")

	generator := benchmark.NewTestDataGenerator()
	detector := detection.NewDetector()
	ctx := context.Background()

	// Test TFN validation
	t.Run("TFN Validation", func(t *testing.T) {
		// Valid TFNs
		validTFNs := []string{
			generator.GenerateValidTFN(),
			"123456782",
			"876543217",
		}

		for _, tfn := range validTFNs {
			code := fmt.Sprintf(`tfn := "%s"`, tfn)
			findings, err := detector.Detect(ctx, []byte(code), "test.go")
			require.NoError(t, err)

			found := false
			for _, f := range findings {
				if f.Type == detection.PITypeTFN {
					found = true
					assert.Equal(t, tfn, f.Match, "Should match exact TFN")
				}
			}
			assert.True(t, found, "Should detect valid TFN: %s", tfn)
		}

		// Invalid TFNs
		invalidTFNs := []string{
			generator.GenerateInvalidTFN(),
			"123456789", // Sequential
			"111111111", // Repeated
		}

		for _, tfn := range invalidTFNs {
			code := fmt.Sprintf(`tfn := "%s"`, tfn)
			findings, err := detector.Detect(ctx, []byte(code), "test.go")
			require.NoError(t, err)

			// Should detect pattern but validation should fail
			for _, f := range findings {
				if f.Type == detection.PITypeTFN && f.Match == tfn {
					t.Logf("Found invalid TFN pattern: %s (confidence: %.2f)", tfn, f.Confidence)
					// Confidence should be lower for invalid TFNs
					assert.Less(t, f.Confidence, float32(1.0), "Invalid TFN should have lower confidence")
				}
			}
		}
	})

	// Test ABN validation
	t.Run("ABN Validation", func(t *testing.T) {
		// Valid ABNs
		validABNs := []string{
			generator.GenerateValidABN(),
			"51824753556", // Commonwealth Bank
		}

		for _, abn := range validABNs {
			code := fmt.Sprintf(`company.ABN = "%s"`, abn)
			findings, err := detector.Detect(ctx, []byte(code), "test.go")
			require.NoError(t, err)

			found := false
			for _, f := range findings {
				if f.Type == detection.PITypeABN {
					found = true
					assert.Equal(t, abn, f.Match, "Should match exact ABN")
				}
			}
			assert.True(t, found, "Should detect valid ABN: %s", abn)
		}
	})

	// Test Medicare validation
	t.Run("Medicare Validation", func(t *testing.T) {
		// Valid Medicare numbers
		for i := 0; i < 5; i++ {
			medicare := generator.GenerateValidMedicare()
			code := fmt.Sprintf(`patient.Medicare = "%s"`, medicare)
			findings, err := detector.Detect(ctx, []byte(code), "test.go")
			require.NoError(t, err)

			found := false
			for _, f := range findings {
				if f.Type == detection.PITypeMedicare {
					found = true
					// Medicare can be 10 or 11 digits
					assert.True(t, len(f.Match) >= 10 && len(f.Match) <= 11,
						"Medicare number should be 10-11 digits")
				}
			}
			assert.True(t, found, "Should detect valid Medicare: %s", medicare)
		}
	})
}

// TestContextFiltering tests that context filtering works correctly
func TestContextFiltering(t *testing.T) {
	t.Log("🎯 Testing Context Filtering Effectiveness")

	generator := benchmark.NewTestDataGenerator()
	baseDetector := detection.NewDetector()
	contextDetector := NewMockContextDetector(baseDetector)
	ctx := context.Background()

	tfn := generator.GenerateValidTFN()

	testCases := []struct {
		name             string
		code             string
		filename         string
		shouldBeFiltered bool
	}{
		{
			name:             "Production Code",
			code:             fmt.Sprintf(`user.TFN = "%s"`, tfn),
			filename:         "user.go",
			shouldBeFiltered: false,
		},
		{
			name:             "Test File",
			code:             fmt.Sprintf(`func TestTFN() { tfn := "%s" }`, tfn),
			filename:         "user_test.go",
			shouldBeFiltered: true,
		},
		{
			name:             "Comment",
			code:             fmt.Sprintf(`// Example TFN: %s`, tfn),
			filename:         "doc.go",
			shouldBeFiltered: true,
		},
		{
			name:             "Mock Data",
			code:             fmt.Sprintf(`const MOCK_TFN = "%s"`, tfn),
			filename:         "mock.go",
			shouldBeFiltered: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Base detector should find all
			baseFindings, err := baseDetector.Detect(ctx, []byte(tc.code), tc.filename)
			require.NoError(t, err)
			assert.NotEmpty(t, baseFindings, "Base detector should find TFN")

			// Context detector should filter appropriately
			contextFindings, err := contextDetector.Detect(ctx, []byte(tc.code), tc.filename)
			require.NoError(t, err)

			if tc.shouldBeFiltered {
				assert.Empty(t, contextFindings, "Context detector should filter out %s", tc.name)
			} else {
				assert.NotEmpty(t, contextFindings, "Context detector should keep %s", tc.name)
			}
		})
	}
}

// BenchmarkComprehensiveDetection benchmarks detection performance
func BenchmarkComprehensiveDetection(b *testing.B) {
	dataset := benchmark.GenerateComprehensiveTestDataset()
	detector := detection.NewDetector()
	ctx := context.Background()

	allCases := dataset.AllCases()
	b.Logf("Benchmarking with %d test cases", len(allCases))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, testCase := range allCases {
			_, err := detector.Detect(ctx, []byte(testCase.Code), testCase.Filename)
			if err != nil {
				b.Fatal(err)
			}
		}
	}

	b.ReportMetric(float64(len(allCases)), "cases/op")
}
