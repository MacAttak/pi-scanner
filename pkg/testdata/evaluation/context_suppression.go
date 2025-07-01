package evaluation

import (
	"context"
	"fmt"

	"github.com/MacAttak/pi-scanner/pkg/testdata/benchmark"
)

// ContextSuppressionMetrics measures how well a detector suppresses detections in specific contexts
type ContextSuppressionMetrics struct {
	Context                string
	TotalCases             int
	SuppressedDetections   int
	UnsuppressedDetections int
	SuppressionRate        float64
}

// EvaluateContextSuppression measures how well a detector suppresses PI in test/mock/comment contexts
func (dc *DetectorComparator) EvaluateContextSuppression(detectorName string) (map[string]*ContextSuppressionMetrics, error) {
	detector, exists := dc.detectors[detectorName]
	if !exists {
		return nil, fmt.Errorf("detector %s not found", detectorName)
	}

	// For comparison, we need a baseline detector without context filtering
	baselineDetector, exists := dc.detectors["Pattern-Only"]
	if !exists {
		// If no baseline available, we can't measure suppression
		return nil, fmt.Errorf("baseline detector 'Pattern-Only' not found for comparison")
	}

	results := make(map[string]*ContextSuppressionMetrics)

	// Group test cases by context
	contextGroups := make(map[string][]benchmark.TestCase)
	for _, testCase := range dc.dataset.AllCases() {
		contextGroups[testCase.Context] = append(contextGroups[testCase.Context], testCase)
	}

	// Evaluate suppression for each context
	for ctxName, testCases := range contextGroups {
		metrics := &ContextSuppressionMetrics{
			Context:    ctxName,
			TotalCases: len(testCases),
		}

		for _, testCase := range testCases {
			// Get baseline detection (without context filtering)
			baselineFindings, err := baselineDetector.Detect(
				context.Background(),
				[]byte(testCase.Code),
				testCase.Filename,
			)
			if err != nil {
				return nil, fmt.Errorf("baseline detection failed: %v", err)
			}

			// Get context-aware detection
			contextFindings, err := detector.Detect(
				context.Background(),
				[]byte(testCase.Code),
				testCase.Filename,
			)
			if err != nil {
				return nil, fmt.Errorf("context detection failed: %v", err)
			}

			// Check if baseline detected something
			baselineDetected := dc.hasRelevantDetection(baselineFindings, testCase.PIType)
			contextDetected := dc.hasRelevantDetection(contextFindings, testCase.PIType)

			// If baseline detected it but context-aware didn't, it was suppressed
			if baselineDetected && !contextDetected {
				metrics.SuppressedDetections++
			} else if baselineDetected && contextDetected {
				metrics.UnsuppressedDetections++
			}
		}

		// Calculate suppression rate
		totalDetections := metrics.SuppressedDetections + metrics.UnsuppressedDetections
		if totalDetections > 0 {
			metrics.SuppressionRate = float64(metrics.SuppressedDetections) / float64(totalDetections)
		}

		results[ctxName] = metrics
	}

	return results, nil
}

// CalculateTestContextSuppressionRate calculates the overall suppression rate for test-like contexts
func CalculateTestContextSuppressionRate(suppressionMetrics map[string]*ContextSuppressionMetrics) float64 {
	testContexts := []string{"test", "mock", "comment"}

	totalSuppressed := 0
	totalDetections := 0

	for _, ctx := range testContexts {
		if metrics, exists := suppressionMetrics[ctx]; exists {
			totalSuppressed += metrics.SuppressedDetections
			totalDetections += metrics.SuppressedDetections + metrics.UnsuppressedDetections
		}
	}

	if totalDetections == 0 {
		return 0.0
	}

	return float64(totalSuppressed) / float64(totalDetections)
}
