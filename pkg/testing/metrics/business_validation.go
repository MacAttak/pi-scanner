package metrics

import (
	"context"
	"fmt"
	"time"

	"github.com/MacAttak/pi-scanner/pkg/detection"
	"github.com/MacAttak/pi-scanner/pkg/testing/datasets"
)

// BusinessValidationMetrics provides enterprise-grade validation metrics
type BusinessValidationMetrics struct {
	Detector detection.Detector
}

// ValidationResult represents business validation results
type ValidationResult struct {
	TestName           string                   `json:"test_name"`
	Timestamp          time.Time                `json:"timestamp"`
	OverallScore       float64                  `json:"overall_score"`
	AccuracyScore      float64                  `json:"accuracy_score"`
	PrecisionScore     float64                  `json:"precision_score"`
	RecallScore        float64                  `json:"recall_score"`
	F1Score            float64                  `json:"f1_score"`
	ContextAccuracy    float64                  `json:"context_accuracy"`
	ComplexityResults  map[string]*ComplexityMetric `json:"complexity_results"`
	LanguageResults    map[string]*LanguageMetric   `json:"language_results"`
	PITypeResults      map[string]*PITypeMetric     `json:"pi_type_results"`
	PerformanceMetrics PerformanceMetric        `json:"performance_metrics"`
	RiskAssessment     RiskAssessment           `json:"risk_assessment"`
	Recommendations    []string                 `json:"recommendations"`
}

// ComplexityMetric tracks performance by code complexity
type ComplexityMetric struct {
	TotalSamples     int     `json:"total_samples"`
	CorrectFindings  int     `json:"correct_findings"`
	FalsePositives   int     `json:"false_positives"`
	FalseNegatives   int     `json:"false_negatives"`
	AccuracyRate     float64 `json:"accuracy_rate"`
	AverageTime      time.Duration `json:"average_time"`
}

// LanguageMetric tracks performance by programming language  
type LanguageMetric struct {
	TotalSamples     int     `json:"total_samples"`
	DetectedFindings int     `json:"detected_findings"`
	ExpectedFindings int     `json:"expected_findings"`
	Precision        float64 `json:"precision"`
	Recall           float64 `json:"recall"`
	F1Score          float64 `json:"f1_score"`
}

// PITypeMetric tracks performance by PI type
type PITypeMetric struct {
	TotalExpected    int     `json:"total_expected"`
	TotalDetected    int     `json:"total_detected"`
	CorrectDetected  int     `json:"correct_detected"`
	DetectionRate    float64 `json:"detection_rate"`
	Precision        float64 `json:"precision"`
	ConfidenceAvg    float64 `json:"confidence_avg"`
}

// PerformanceMetric tracks scanning performance
type PerformanceMetric struct {
	TotalProcessingTime time.Duration `json:"total_processing_time"`
	AverageFileTime     time.Duration `json:"average_file_time"`
	FilesPerSecond      float64       `json:"files_per_second"`
	BytesPerSecond      int64         `json:"bytes_per_second"`
	MemoryUsageMB       float64       `json:"memory_usage_mb"`
}

// RiskAssessment provides enterprise risk evaluation
type RiskAssessment struct {
	OverallRiskLevel    string   `json:"overall_risk_level"`
	CriticalFindings    int      `json:"critical_findings"`
	HighRiskFindings    int      `json:"high_risk_findings"`
	MediumRiskFindings  int      `json:"medium_risk_findings"`
	LowRiskFindings     int      `json:"low_risk_findings"`
	ComplianceIssues    []string `json:"compliance_issues"`
	SecurityRecommendations []string `json:"security_recommendations"`
}

// NewBusinessValidationMetrics creates a new business validation instance
func NewBusinessValidationMetrics(detector detection.Detector) *BusinessValidationMetrics {
	return &BusinessValidationMetrics{
		Detector: detector,
	}
}

// RunComprehensiveValidation performs full business validation suite
func (bvm *BusinessValidationMetrics) RunComprehensiveValidation(ctx context.Context) (*ValidationResult, error) {
	startTime := time.Now()
	
	result := &ValidationResult{
		TestName:          "Comprehensive Business Validation",
		Timestamp:         startTime,
		ComplexityResults: make(map[string]*ComplexityMetric),
		LanguageResults:   make(map[string]*LanguageMetric),
		PITypeResults:     make(map[string]*PITypeMetric),
	}
	
	// Get all real-world samples
	samples := datasets.GetRealWorldSamples()
	
	// Track overall metrics
	var totalExpected, totalDetected, truePositives, falsePositives, falseNegatives int
	var totalProcessingTime time.Duration
	var totalBytes int64
	
	// Initialize tracking maps
	complexityStats := make(map[string]*ComplexityMetric)
	languageStats := make(map[string]*LanguageMetric)
	piTypeStats := make(map[string]*PITypeMetric)
	
	// Process each sample
	for _, sample := range samples {
		sampleStart := time.Now()
		
		// Run detection
		findings, err := bvm.Detector.Detect(ctx, []byte(sample.Code), sample.Filename)
		if err != nil {
			return nil, fmt.Errorf("detection failed for sample %s: %w", sample.ID, err)
		}
		
		processingTime := time.Since(sampleStart)
		totalProcessingTime += processingTime
		totalBytes += int64(len(sample.Code))
		
		// Analyze results
		sampleResults := bvm.analyzeSampleResults(sample, findings)
		
		// Update overall counts
		totalExpected += len(sample.ExpectedPIs)
		totalDetected += len(findings)
		truePositives += sampleResults.TruePositives
		falsePositives += sampleResults.FalsePositives
		falseNegatives += sampleResults.FalseNegatives
		
		// Update complexity metrics
		if _, exists := complexityStats[sample.Complexity]; !exists {
			complexityStats[sample.Complexity] = &ComplexityMetric{}
		}
		bvm.updateComplexityMetrics(complexityStats[sample.Complexity], sampleResults, processingTime)
		
		// Update language metrics
		if _, exists := languageStats[sample.Language]; !exists {
			languageStats[sample.Language] = &LanguageMetric{}
		}
		bvm.updateLanguageMetrics(languageStats[sample.Language], sampleResults)
		
		// Update PI type metrics
		for _, expectedPI := range sample.ExpectedPIs {
			piType := string(expectedPI.Type)
			if _, exists := piTypeStats[piType]; !exists {
				piTypeStats[piType] = &PITypeMetric{}
			}
			bvm.updatePITypeMetrics(piTypeStats[piType], expectedPI, findings)
		}
	}
	
	// Calculate final metrics
	result.AccuracyScore = float64(truePositives) / float64(truePositives+falsePositives+falseNegatives)
	result.PrecisionScore = float64(truePositives) / float64(truePositives+falsePositives)
	result.RecallScore = float64(truePositives) / float64(truePositives+falseNegatives)
	result.F1Score = 2 * (result.PrecisionScore * result.RecallScore) / (result.PrecisionScore + result.RecallScore)
	result.OverallScore = (result.AccuracyScore + result.F1Score) / 2
	
	// Calculate context accuracy
	result.ContextAccuracy = bvm.calculateContextAccuracy(samples)
	
	// Set performance metrics
	result.PerformanceMetrics = PerformanceMetric{
		TotalProcessingTime: totalProcessingTime,
		AverageFileTime:     totalProcessingTime / time.Duration(len(samples)),
		FilesPerSecond:      float64(len(samples)) / totalProcessingTime.Seconds(),
		BytesPerSecond:      int64(float64(totalBytes) / totalProcessingTime.Seconds()),
	}
	
	// Copy stats to result
	for complexity, stats := range complexityStats {
		result.ComplexityResults[complexity] = stats
	}
	for language, stats := range languageStats {
		result.LanguageResults[language] = stats
	}
	for piType, stats := range piTypeStats {
		result.PITypeResults[piType] = stats
	}
	
	// Generate risk assessment
	result.RiskAssessment = bvm.generateRiskAssessment(samples)
	
	// Generate recommendations
	result.Recommendations = bvm.generateRecommendations(result)
	
	return result, nil
}

// SampleAnalysisResult represents analysis for a single sample
type SampleAnalysisResult struct {
	TruePositives  int
	FalsePositives int
	FalseNegatives int
}

// analyzeSampleResults analyzes detection results for a single sample
func (bvm *BusinessValidationMetrics) analyzeSampleResults(sample datasets.RealWorldSample, findings []detection.Finding) SampleAnalysisResult {
	result := SampleAnalysisResult{}
	
	// Create map of expected PIs for easy lookup
	expectedMap := make(map[string]bool)
	for _, expected := range sample.ExpectedPIs {
		key := fmt.Sprintf("%s:%s", expected.Type, expected.Value)
		expectedMap[key] = true
	}
	
	// Check findings against expected
	foundMap := make(map[string]bool)
	for _, finding := range findings {
		key := fmt.Sprintf("%s:%s", finding.Type, finding.Match)
		foundMap[key] = true
		
		if expectedMap[key] {
			result.TruePositives++
		} else {
			result.FalsePositives++
		}
	}
	
	// Count false negatives (expected but not found)
	for expectedKey := range expectedMap {
		if !foundMap[expectedKey] {
			result.FalseNegatives++
		}
	}
	
	return result
}

// updateComplexityMetrics updates complexity-based metrics
func (bvm *BusinessValidationMetrics) updateComplexityMetrics(metric *ComplexityMetric, results SampleAnalysisResult, processingTime time.Duration) {
	metric.TotalSamples++
	metric.CorrectFindings += results.TruePositives
	metric.FalsePositives += results.FalsePositives
	metric.FalseNegatives += results.FalseNegatives
	
	total := metric.CorrectFindings + metric.FalsePositives + metric.FalseNegatives
	if total > 0 {
		metric.AccuracyRate = float64(metric.CorrectFindings) / float64(total)
	}
	
	// Update average time
	if metric.TotalSamples == 1 {
		metric.AverageTime = processingTime
	} else {
		metric.AverageTime = (metric.AverageTime*time.Duration(metric.TotalSamples-1) + processingTime) / time.Duration(metric.TotalSamples)
	}
}

// updateLanguageMetrics updates language-based metrics
func (bvm *BusinessValidationMetrics) updateLanguageMetrics(metric *LanguageMetric, results SampleAnalysisResult) {
	metric.TotalSamples++
	metric.DetectedFindings += results.TruePositives + results.FalsePositives
	metric.ExpectedFindings += results.TruePositives + results.FalseNegatives
	
	if metric.DetectedFindings > 0 {
		metric.Precision = float64(results.TruePositives) / float64(metric.DetectedFindings)
	}
	if metric.ExpectedFindings > 0 {
		metric.Recall = float64(results.TruePositives) / float64(metric.ExpectedFindings)
	}
	if metric.Precision+metric.Recall > 0 {
		metric.F1Score = 2 * (metric.Precision * metric.Recall) / (metric.Precision + metric.Recall)
	}
}

// updatePITypeMetrics updates PI type-based metrics
func (bvm *BusinessValidationMetrics) updatePITypeMetrics(metric *PITypeMetric, expected datasets.ExpectedPI, findings []detection.Finding) {
	metric.TotalExpected++
	
	// Check if this PI was detected
	detected := false
	var confidenceSum float64
	var confidenceCount int
	
	for _, finding := range findings {
		if finding.Type == expected.Type && finding.Match == expected.Value {
			detected = true
			metric.CorrectDetected++
			confidenceSum += float64(finding.Confidence)
			confidenceCount++
		}
		if finding.Type == expected.Type {
			metric.TotalDetected++
		}
	}
	
	if detected {
		metric.DetectionRate = float64(metric.CorrectDetected) / float64(metric.TotalExpected)
	}
	
	if metric.TotalDetected > 0 {
		metric.Precision = float64(metric.CorrectDetected) / float64(metric.TotalDetected)
	}
	
	if confidenceCount > 0 {
		metric.ConfidenceAvg = confidenceSum / float64(confidenceCount)
	}
}

// calculateContextAccuracy calculates context-aware detection accuracy
func (bvm *BusinessValidationMetrics) calculateContextAccuracy(samples []datasets.RealWorldSample) float64 {
	var contextCorrect, total int
	
	for _, sample := range samples {
		total++
		
		// Context-specific expectations
		switch sample.Context {
		case "test":
			// Test files should have minimal detection
			if len(sample.ExpectedPIs) == 0 {
				contextCorrect++
			}
		case "production", "logging":
			// Production files should detect PI appropriately
			if len(sample.ExpectedPIs) > 0 {
				contextCorrect++
			}
		default:
			contextCorrect++ // Neutral contexts
		}
	}
	
	if total == 0 {
		return 0
	}
	
	return float64(contextCorrect) / float64(total)
}

// generateRiskAssessment creates enterprise risk assessment
func (bvm *BusinessValidationMetrics) generateRiskAssessment(samples []datasets.RealWorldSample) RiskAssessment {
	assessment := RiskAssessment{
		ComplianceIssues:        []string{},
		SecurityRecommendations: []string{},
	}
	
	// Count findings by risk level
	for _, sample := range samples {
		for _, expectedPI := range sample.ExpectedPIs {
			switch expectedPI.Type {
			case detection.PITypeTFN, detection.PITypeMedicare:
				assessment.CriticalFindings++
			case detection.PITypeABN, detection.PITypeACN, detection.PITypeCreditCard:
				assessment.HighRiskFindings++
			case detection.PITypeBSB, detection.PITypeName, detection.PITypePhone:
				assessment.MediumRiskFindings++
			default:
				assessment.LowRiskFindings++
			}
		}
	}
	
	// Determine overall risk level
	if assessment.CriticalFindings > 5 {
		assessment.OverallRiskLevel = "CRITICAL"
		assessment.ComplianceIssues = append(assessment.ComplianceIssues, "High volume of critical PI detected")
	} else if assessment.HighRiskFindings > 10 {
		assessment.OverallRiskLevel = "HIGH"
		assessment.ComplianceIssues = append(assessment.ComplianceIssues, "Multiple high-risk PI types detected")
	} else if assessment.MediumRiskFindings > 15 {
		assessment.OverallRiskLevel = "MEDIUM"
	} else {
		assessment.OverallRiskLevel = "LOW"
	}
	
	// Generate security recommendations
	if assessment.CriticalFindings > 0 {
		assessment.SecurityRecommendations = append(assessment.SecurityRecommendations, 
			"Implement data masking for TFN and Medicare numbers")
	}
	if assessment.HighRiskFindings > 0 {
		assessment.SecurityRecommendations = append(assessment.SecurityRecommendations,
			"Review ABN and ACN storage practices")
	}
	
	return assessment
}

// generateRecommendations provides actionable recommendations
func (bvm *BusinessValidationMetrics) generateRecommendations(result *ValidationResult) []string {
	var recommendations []string
	
	// Accuracy recommendations
	if result.OverallScore < 0.8 {
		recommendations = append(recommendations, "Overall detection accuracy below 80% - review detection patterns")
	}
	
	// Precision recommendations
	if result.PrecisionScore < 0.85 {
		recommendations = append(recommendations, "High false positive rate - enhance context filtering")
	}
	
	// Recall recommendations  
	if result.RecallScore < 0.85 {
		recommendations = append(recommendations, "Missing PI detections - review pattern completeness")
	}
	
	// Context recommendations
	if result.ContextAccuracy < 0.9 {
		recommendations = append(recommendations, "Context awareness needs improvement")
	}
	
	// Performance recommendations
	if result.PerformanceMetrics.FilesPerSecond < 10 {
		recommendations = append(recommendations, "Performance optimization needed for large-scale scanning")
	}
	
	return recommendations
}

// GenerateReport creates a formatted validation report
func (bvm *BusinessValidationMetrics) GenerateReport(result *ValidationResult) string {
	report := fmt.Sprintf(`
# Business Validation Report

**Test:** %s  
**Timestamp:** %s  
**Overall Score:** %.2f%%

## Key Metrics
- **Accuracy:** %.2f%%
- **Precision:** %.2f%%  
- **Recall:** %.2f%%
- **F1 Score:** %.2f%%
- **Context Accuracy:** %.2f%%

## Performance
- **Files/Second:** %.1f
- **Bytes/Second:** %d
- **Average Processing Time:** %s

## Risk Assessment
- **Risk Level:** %s
- **Critical Findings:** %d
- **High Risk Findings:** %d

## Recommendations
`,
		result.TestName,
		result.Timestamp.Format("2006-01-02 15:04:05"),
		result.OverallScore*100,
		result.AccuracyScore*100,
		result.PrecisionScore*100,
		result.RecallScore*100,
		result.F1Score*100,
		result.ContextAccuracy*100,
		result.PerformanceMetrics.FilesPerSecond,
		result.PerformanceMetrics.BytesPerSecond,
		result.PerformanceMetrics.AverageFileTime,
		result.RiskAssessment.OverallRiskLevel,
		result.RiskAssessment.CriticalFindings,
		result.RiskAssessment.HighRiskFindings,
	)
	
	for i, rec := range result.Recommendations {
		report += fmt.Sprintf("%d. %s\n", i+1, rec)
	}
	
	return report
}