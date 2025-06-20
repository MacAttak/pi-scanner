package evaluation

import (
	"context"
	"fmt"
	"time"

	"github.com/MacAttak/pi-scanner/pkg/detection"
	"github.com/MacAttak/pi-scanner/pkg/testing/benchmark"
)

// DetectorComparator evaluates and compares different detector configurations
type DetectorComparator struct {
	detectors map[string]detection.Detector
	dataset   *benchmark.BenchmarkDataset
}

// NewDetectorComparator creates a new comparator with the given detectors and dataset
func NewDetectorComparator(detectors map[string]detection.Detector, dataset *benchmark.BenchmarkDataset) *DetectorComparator {
	return &DetectorComparator{
		detectors: detectors,
		dataset:   dataset,
	}
}

// Compare evaluates all detectors against the dataset and returns results
func (dc *DetectorComparator) Compare() (map[string]*EvaluationResult, error) {
	results := make(map[string]*EvaluationResult)

	for name, detector := range dc.detectors {
		fmt.Printf("Evaluating detector: %s\n", name)

		startTime := time.Now()
		metrics, err := dc.evaluateDetector(detector)
		executionTime := time.Since(startTime)

		if err != nil {
			return nil, fmt.Errorf("failed to evaluate detector %s: %v", name, err)
		}

		result := &EvaluationResult{
			DetectorName:  name,
			TestSetName:   "Australian PI Benchmark",
			Timestamp:     time.Now(),
			Metrics:       metrics,
			ExecutionTime: executionTime,
			TestCaseCount: len(dc.dataset.AllCases()),
		}

		results[name] = result
		fmt.Printf("  %s\n", metrics.CompactReport())
	}

	return results, nil
}

// evaluateDetector runs a single detector against all test cases
func (dc *DetectorComparator) evaluateDetector(detector detection.Detector) (*EvaluationMetrics, error) {
	metrics := &EvaluationMetrics{}

	for _, testCase := range dc.dataset.AllCases() {
		// Run detection
		findings, err := detector.Detect(
			context.Background(),
			[]byte(testCase.Code),
			testCase.Filename,
		)
		if err != nil {
			return nil, fmt.Errorf("detection failed for test case %s: %v", testCase.ID, err)
		}

		// Check if we detected the expected PI type
		detected := dc.hasRelevantDetection(findings, testCase.PIType)

		// Update confusion matrix
		switch {
		case detected && testCase.IsActualPI:
			metrics.TruePositives++
		case detected && !testCase.IsActualPI:
			metrics.FalsePositives++
		case !detected && !testCase.IsActualPI:
			metrics.TrueNegatives++
		case !detected && testCase.IsActualPI:
			metrics.FalseNegatives++
		}
	}

	return metrics, nil
}

// hasRelevantDetection checks if the findings contain the expected PI type
func (dc *DetectorComparator) hasRelevantDetection(findings []detection.Finding, expectedType detection.PIType) bool {
	for _, finding := range findings {
		if finding.Type == expectedType {
			return true
		}
	}
	return false
}

// EvaluateByContext evaluates performance broken down by context (test, production, comment, etc.)
func (dc *DetectorComparator) EvaluateByContext(detectorName string) (map[string]*EvaluationMetrics, error) {
	detector, exists := dc.detectors[detectorName]
	if !exists {
		return nil, fmt.Errorf("detector %s not found", detectorName)
	}

	results := make(map[string]*EvaluationMetrics)

	// Group test cases by context
	contextGroups := make(map[string][]benchmark.TestCase)
	for _, testCase := range dc.dataset.AllCases() {
		contextGroups[testCase.Context] = append(contextGroups[testCase.Context], testCase)
	}

	// Evaluate each context separately
	for ctxName, testCases := range contextGroups {
		metrics := &EvaluationMetrics{}

		for _, testCase := range testCases {
			findings, err := detector.Detect(
				context.Background(),
				[]byte(testCase.Code),
				testCase.Filename,
			)
			if err != nil {
				return nil, fmt.Errorf("detection failed for test case %s: %v", testCase.ID, err)
			}

			detected := dc.hasRelevantDetection(findings, testCase.PIType)

			switch {
			case detected && testCase.IsActualPI:
				metrics.TruePositives++
			case detected && !testCase.IsActualPI:
				metrics.FalsePositives++
			case !detected && !testCase.IsActualPI:
				metrics.TrueNegatives++
			case !detected && testCase.IsActualPI:
				metrics.FalseNegatives++
			}
		}

		results[ctxName] = metrics
	}

	return results, nil
}

// EvaluateByPIType evaluates performance broken down by PI type
func (dc *DetectorComparator) EvaluateByPIType(detectorName string) (map[string]*EvaluationMetrics, error) {
	detector, exists := dc.detectors[detectorName]
	if !exists {
		return nil, fmt.Errorf("detector %s not found", detectorName)
	}

	results := make(map[string]*EvaluationMetrics)

	// Group test cases by PI type
	typeGroups := make(map[detection.PIType][]benchmark.TestCase)
	for _, testCase := range dc.dataset.AllCases() {
		typeGroups[testCase.PIType] = append(typeGroups[testCase.PIType], testCase)
	}

	// Evaluate each PI type separately
	for piType, testCases := range typeGroups {
		metrics := &EvaluationMetrics{}

		for _, testCase := range testCases {
			findings, err := detector.Detect(
				context.Background(),
				[]byte(testCase.Code),
				testCase.Filename,
			)
			if err != nil {
				return nil, fmt.Errorf("detection failed for test case %s: %v", testCase.ID, err)
			}

			detected := dc.hasRelevantDetection(findings, testCase.PIType)

			switch {
			case detected && testCase.IsActualPI:
				metrics.TruePositives++
			case detected && !testCase.IsActualPI:
				metrics.FalsePositives++
			case !detected && !testCase.IsActualPI:
				metrics.TrueNegatives++
			case !detected && testCase.IsActualPI:
				metrics.FalseNegatives++
			}
		}

		results[string(piType)] = metrics
	}

	return results, nil
}

// GenerateDetailedReport creates a comprehensive evaluation report
func (dc *DetectorComparator) GenerateDetailedReport() (*DetailedReport, error) {
	overall, err := dc.Compare()
	if err != nil {
		return nil, err
	}

	report := &DetailedReport{
		Timestamp:    time.Now(),
		DatasetStats: dc.dataset.Stats(),
		Overall:      overall,
		ByContext:    make(map[string]map[string]*EvaluationMetrics),
		ByPIType:     make(map[string]map[string]*EvaluationMetrics),
	}

	// Generate context and PI type breakdowns for each detector
	for detectorName := range dc.detectors {
		contextResults, err := dc.EvaluateByContext(detectorName)
		if err != nil {
			return nil, err
		}
		report.ByContext[detectorName] = contextResults

		typeResults, err := dc.EvaluateByPIType(detectorName)
		if err != nil {
			return nil, err
		}
		report.ByPIType[detectorName] = typeResults
	}

	// Generate comparison and quality assessments
	var results []EvaluationResult
	for _, result := range overall {
		results = append(results, *result)
	}
	report.Comparison = GenerateComparison(results)

	report.QualityAssessments = make(map[string]*QualityAssessment)
	for detectorName, result := range overall {
		report.QualityAssessments[detectorName] = AssessQuality(result.Metrics)
	}

	return report, nil
}

// DetailedReport represents a comprehensive evaluation report
type DetailedReport struct {
	Timestamp          time.Time                                `json:"timestamp"`
	DatasetStats       benchmark.DatasetStats                   `json:"dataset_stats"`
	Overall            map[string]*EvaluationResult             `json:"overall"`
	ByContext          map[string]map[string]*EvaluationMetrics `json:"by_context"`
	ByPIType           map[string]map[string]*EvaluationMetrics `json:"by_pi_type"`
	Comparison         *ComparisonReport                        `json:"comparison"`
	QualityAssessments map[string]*QualityAssessment            `json:"quality_assessments"`
}

// Summary returns a human-readable summary of the report
func (dr *DetailedReport) Summary() string {
	summary := fmt.Sprintf("PI Detection Quality Assessment Report\n")
	summary += fmt.Sprintf("Generated: %s\n\n", dr.Timestamp.Format("2006-01-02 15:04:05"))

	summary += fmt.Sprintf("Dataset Statistics:\n")
	summary += fmt.Sprintf("- Total test cases: %d\n", dr.DatasetStats.TotalCases)
	summary += fmt.Sprintf("- True positives: %d\n", dr.DatasetStats.TruePositives)
	summary += fmt.Sprintf("- True negatives: %d\n", dr.DatasetStats.TrueNegatives)
	summary += fmt.Sprintf("- Edge cases: %d\n\n", dr.DatasetStats.EdgeCases)

	summary += dr.Comparison.Summary

	summary += "\nQuality Grades:\n"
	for detectorName, assessment := range dr.QualityAssessments {
		summary += fmt.Sprintf("- %s: Grade %s (%.1f%%)\n", detectorName, assessment.Grade, assessment.Score)
	}

	return summary
}
