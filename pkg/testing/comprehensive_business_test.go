package testing

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/MacAttak/pi-scanner/pkg/detection"
	"github.com/MacAttak/pi-scanner/pkg/testing/datasets"
	"github.com/MacAttak/pi-scanner/pkg/testing/metrics"
)

func TestComprehensiveBusinessValidation(t *testing.T) {
	// Create detector with production configuration
	detector := detection.NewDetector()
	
	// Create business validation metrics
	validator := metrics.NewBusinessValidationMetrics(detector)
	
	// Run comprehensive validation
	ctx := context.Background()
	result, err := validator.RunComprehensiveValidation(ctx)
	require.NoError(t, err, "Business validation should complete successfully")
	
	// Assert minimum quality thresholds for enterprise deployment
	t.Run("Overall Quality Metrics", func(t *testing.T) {
		assert.GreaterOrEqual(t, result.OverallScore, 0.80, 
			"Overall score should be at least 80% for enterprise deployment")
		
		assert.GreaterOrEqual(t, result.AccuracyScore, 0.85, 
			"Accuracy should be at least 85% for production use")
		
		assert.GreaterOrEqual(t, result.PrecisionScore, 0.80, 
			"Precision should be at least 80% to minimize false positives")
		
		assert.GreaterOrEqual(t, result.RecallScore, 0.75, 
			"Recall should be at least 75% to catch most PI")
		
		assert.GreaterOrEqual(t, result.F1Score, 0.77, 
			"F1 score should balance precision and recall effectively")
	})
	
	t.Run("Context Awareness", func(t *testing.T) {
		assert.GreaterOrEqual(t, result.ContextAccuracy, 0.85, 
			"Context-aware detection should be at least 85% accurate")
	})
	
	t.Run("Performance Requirements", func(t *testing.T) {
		assert.GreaterOrEqual(t, result.PerformanceMetrics.FilesPerSecond, 5.0, 
			"Should process at least 5 files per second")
		
		assert.LessOrEqual(t, result.PerformanceMetrics.AverageFileTime.Milliseconds(), int64(200), 
			"Average file processing should be under 200ms")
	})
	
	t.Run("Language-Specific Performance", func(t *testing.T) {
		// Java should have excellent performance
		if javaStats, exists := result.LanguageResults["java"]; exists {
			assert.GreaterOrEqual(t, javaStats.F1Score, 0.80, 
				"Java detection should have F1 score >= 80%")
		}
		
		// Python should have good performance  
		if pythonStats, exists := result.LanguageResults["python"]; exists {
			assert.GreaterOrEqual(t, pythonStats.F1Score, 0.75, 
				"Python detection should have F1 score >= 75%")
		}
		
		// Scala should have good performance
		if scalaStats, exists := result.LanguageResults["scala"]; exists {
			assert.GreaterOrEqual(t, scalaStats.F1Score, 0.75, 
				"Scala detection should have F1 score >= 75%")
		}
	})
	
	t.Run("PI Type Detection Rates", func(t *testing.T) {
		// Critical PI types should have high detection rates
		criticalTypes := []string{"TFN", "MEDICARE", "CREDIT_CARD"}
		for _, piType := range criticalTypes {
			if stats, exists := result.PITypeResults[piType]; exists {
				assert.GreaterOrEqual(t, stats.DetectionRate, 0.90, 
					"Critical PI type %s should have detection rate >= 90%", piType)
			}
		}
		
		// High-risk PI types should have good detection rates
		highRiskTypes := []string{"ABN", "ACN", "BSB"}
		for _, piType := range highRiskTypes {
			if stats, exists := result.PITypeResults[piType]; exists {
				assert.GreaterOrEqual(t, stats.DetectionRate, 0.80, 
					"High-risk PI type %s should have detection rate >= 80%", piType)
			}
		}
	})
	
	t.Run("Complexity Handling", func(t *testing.T) {
		// Simple code should have excellent accuracy
		if simpleStats, exists := result.ComplexityResults["simple"]; exists {
			assert.GreaterOrEqual(t, simpleStats.AccuracyRate, 0.90, 
				"Simple code should have accuracy >= 90%")
		}
		
		// Medium complexity should have good accuracy
		if mediumStats, exists := result.ComplexityResults["medium"]; exists {
			assert.GreaterOrEqual(t, mediumStats.AccuracyRate, 0.80, 
				"Medium complexity code should have accuracy >= 80%")
		}
		
		// Complex code should have acceptable accuracy
		if complexStats, exists := result.ComplexityResults["complex"]; exists {
			assert.GreaterOrEqual(t, complexStats.AccuracyRate, 0.70, 
				"Complex code should have accuracy >= 70%")
		}
	})
	
	t.Run("Risk Assessment", func(t *testing.T) {
		// Risk level should be categorized appropriately
		assert.Contains(t, []string{"LOW", "MEDIUM", "HIGH", "CRITICAL"}, 
			result.RiskAssessment.OverallRiskLevel, 
			"Risk level should be properly categorized")
		
		// Should provide actionable recommendations
		assert.NotEmpty(t, result.Recommendations, 
			"Should provide actionable recommendations")
		
		// Should identify compliance issues if present
		if result.RiskAssessment.CriticalFindings > 0 {
			assert.NotEmpty(t, result.RiskAssessment.ComplianceIssues, 
				"Should identify compliance issues when critical findings exist")
		}
	})
	
	// Log comprehensive results for analysis
	t.Logf("Business Validation Results:")
	t.Logf("  Overall Score: %.2f%%", result.OverallScore*100)
	t.Logf("  Accuracy: %.2f%%", result.AccuracyScore*100)
	t.Logf("  Precision: %.2f%%", result.PrecisionScore*100)
	t.Logf("  Recall: %.2f%%", result.RecallScore*100)
	t.Logf("  F1 Score: %.2f%%", result.F1Score*100)
	t.Logf("  Context Accuracy: %.2f%%", result.ContextAccuracy*100)
	t.Logf("  Performance: %.1f files/sec", result.PerformanceMetrics.FilesPerSecond)
	t.Logf("  Risk Level: %s", result.RiskAssessment.OverallRiskLevel)
	
	// Log detailed breakdown
	t.Logf("Language Results:")
	for lang, stats := range result.LanguageResults {
		t.Logf("  %s: Precision=%.2f%%, Recall=%.2f%%, F1=%.2f%%", 
			lang, stats.Precision*100, stats.Recall*100, stats.F1Score*100)
	}
	
	t.Logf("PI Type Results:")
	for piType, stats := range result.PITypeResults {
		t.Logf("  %s: Detection=%.2f%%, Precision=%.2f%%, Confidence=%.2f", 
			piType, stats.DetectionRate*100, stats.Precision*100, stats.ConfidenceAvg)
	}
	
	// Log recommendations
	t.Logf("Recommendations:")
	for i, rec := range result.Recommendations {
		t.Logf("  %d. %s", i+1, rec)
	}
	
	// Generate and log full report
	report := validator.GenerateReport(result)
	t.Logf("Full Business Validation Report:\n%s", report)
}

func TestRealWorldDatasetCoverage(t *testing.T) {
	samples := datasets.GetRealWorldSamples()
	
	t.Run("Dataset Completeness", func(t *testing.T) {
		assert.GreaterOrEqual(t, len(samples), 5, 
			"Should have at least 5 real-world samples")
		
		// Check language coverage
		languages := make(map[string]bool)
		complexities := make(map[string]bool)
		contexts := make(map[string]bool)
		
		for _, sample := range samples {
			languages[sample.Language] = true
			complexities[sample.Complexity] = true
			contexts[sample.Context] = true
		}
		
		assert.GreaterOrEqual(t, len(languages), 3, 
			"Should cover at least 3 programming languages")
		
		assert.GreaterOrEqual(t, len(complexities), 2, 
			"Should cover at least 2 complexity levels")
		
		assert.GreaterOrEqual(t, len(contexts), 2, 
			"Should cover at least 2 different contexts")
	})
	
	t.Run("Sample Quality", func(t *testing.T) {
		for _, sample := range samples {
			assert.NotEmpty(t, sample.ID, "Sample should have unique ID")
			assert.NotEmpty(t, sample.Description, "Sample should have description")
			assert.NotEmpty(t, sample.Code, "Sample should have code content")
			assert.NotEmpty(t, sample.Filename, "Sample should have filename")
			assert.NotEmpty(t, sample.Context, "Sample should have context")
			assert.NotEmpty(t, sample.Complexity, "Sample should have complexity")
			
			// Validate expected PIs have required fields
			for _, expectedPI := range sample.ExpectedPIs {
				assert.NotEmpty(t, expectedPI.Type, "Expected PI should have type")
				assert.NotEmpty(t, expectedPI.Value, "Expected PI should have value")
				assert.Greater(t, expectedPI.Confidence, 0.0, "Expected PI should have confidence")
			}
		}
	})
}

func TestBusinessMetricsDetectionAccuracy(t *testing.T) {
	detector := detection.NewDetector()
	ctx := context.Background()
	
	// Test specific high-value scenarios
	t.Run("Government Service Integration", func(t *testing.T) {
		samples := datasets.GetSamplesByContext("production")
		
		var totalExpected, totalDetected int
		for _, sample := range samples {
			if sample.Description == "Government service integration with TFN validation" {
				findings, err := detector.Detect(ctx, []byte(sample.Code), sample.Filename)
				require.NoError(t, err)
				
				totalExpected += len(sample.ExpectedPIs)
				totalDetected += len(findings)
				
				// Should detect the TFN in government service
				assert.Greater(t, len(findings), 0, "Should detect PI in government service")
			}
		}
		
		if totalExpected > 0 {
			detectionRate := float64(totalDetected) / float64(totalExpected)
			assert.GreaterOrEqual(t, detectionRate, 0.8, 
				"Government service samples should have >= 80% detection rate")
		}
	})
	
	t.Run("Banking System Integration", func(t *testing.T) {
		samples := datasets.GetSamplesByLanguage("scala")
		
		for _, sample := range samples {
			if sample.Description == "Banking system with BSB and account numbers" {
				findings, err := detector.Detect(ctx, []byte(sample.Code), sample.Filename)
				require.NoError(t, err)
				
				// Should detect BSB numbers
				bsbFound := false
				for _, finding := range findings {
					if finding.Type == detection.PITypeBSB {
						bsbFound = true
						break
					}
				}
				assert.True(t, bsbFound, "Should detect BSB numbers in banking system")
			}
		}
	})
	
	t.Run("Healthcare System", func(t *testing.T) {
		samples := datasets.GetSamplesByLanguage("python")
		
		for _, sample := range samples {
			if sample.Description == "Healthcare management system with Medicare numbers" {
				findings, err := detector.Detect(ctx, []byte(sample.Code), sample.Filename)
				require.NoError(t, err)
				
				// Should detect Medicare numbers
				medicareFound := false
				for _, finding := range findings {
					if finding.Type == detection.PITypeMedicare {
						medicareFound = true
						break
					}
				}
				assert.True(t, medicareFound, "Should detect Medicare numbers in healthcare system")
			}
		}
	})
}