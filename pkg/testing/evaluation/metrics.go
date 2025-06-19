package evaluation

import (
	"fmt"
	"time"
)

// EvaluationMetrics represents standard classification metrics for PI detection
type EvaluationMetrics struct {
	TruePositives  int `json:"true_positives"`  // Correctly identified PI
	FalsePositives int `json:"false_positives"` // Incorrectly flagged as PI
	TrueNegatives  int `json:"true_negatives"`  // Correctly identified as not PI
	FalseNegatives int `json:"false_negatives"` // Missed actual PI
}

// Precision calculates what percentage of detected PI are actually PI
// Precision = TP / (TP + FP)
func (em *EvaluationMetrics) Precision() float64 {
	if em.TruePositives+em.FalsePositives == 0 {
		return 0
	}
	return float64(em.TruePositives) / float64(em.TruePositives+em.FalsePositives)
}

// Recall calculates what percentage of actual PI we detected
// Recall = TP / (TP + FN)
func (em *EvaluationMetrics) Recall() float64 {
	if em.TruePositives+em.FalseNegatives == 0 {
		return 0
	}
	return float64(em.TruePositives) / float64(em.TruePositives+em.FalseNegatives)
}

// F1Score calculates the harmonic mean of precision and recall
// F1 = 2 * (Precision * Recall) / (Precision + Recall)
func (em *EvaluationMetrics) F1Score() float64 {
	p := em.Precision()
	r := em.Recall()
	if p+r == 0 {
		return 0
	}
	return 2 * (p * r) / (p + r)
}

// Specificity calculates what percentage of non-PI we correctly identified
// Specificity = TN / (TN + FP)
func (em *EvaluationMetrics) Specificity() float64 {
	if em.TrueNegatives+em.FalsePositives == 0 {
		return 0
	}
	return float64(em.TrueNegatives) / float64(em.TrueNegatives+em.FalsePositives)
}

// Accuracy calculates overall correctness (less useful for imbalanced datasets)
// Accuracy = (TP + TN) / (TP + TN + FP + FN)
func (em *EvaluationMetrics) Accuracy() float64 {
	total := em.TruePositives + em.TrueNegatives + em.FalsePositives + em.FalseNegatives
	if total == 0 {
		return 0
	}
	return float64(em.TruePositives+em.TrueNegatives) / float64(total)
}

// Total returns the total number of test cases
func (em *EvaluationMetrics) Total() int {
	return em.TruePositives + em.TrueNegatives + em.FalsePositives + em.FalseNegatives
}

// Add combines metrics from another evaluation
func (em *EvaluationMetrics) Add(other *EvaluationMetrics) {
	em.TruePositives += other.TruePositives
	em.FalsePositives += other.FalsePositives
	em.TrueNegatives += other.TrueNegatives
	em.FalseNegatives += other.FalseNegatives
}

// Report generates a detailed performance report
func (em *EvaluationMetrics) Report() string {
	return fmt.Sprintf(`
Performance Metrics:
- Precision: %.2f%% (of detected PI, how many were actual PI)
- Recall:    %.2f%% (of actual PI, how many did we detect)
- F1-Score:  %.2f%% (harmonic mean of precision and recall)
- Specificity: %.2f%% (of non-PI, how many correctly identified)
- Accuracy:  %.2f%% (overall correctness)

Confusion Matrix:
- True Positives:  %d (correctly identified PI)
- False Positives: %d (incorrectly flagged as PI)
- True Negatives:  %d (correctly identified as not PI)
- False Negatives: %d (missed actual PI)

Total Test Cases: %d
    `,
		em.Precision()*100,
		em.Recall()*100,
		em.F1Score()*100,
		em.Specificity()*100,
		em.Accuracy()*100,
		em.TruePositives,
		em.FalsePositives,
		em.TrueNegatives,
		em.FalseNegatives,
		em.Total(),
	)
}

// CompactReport generates a single line summary
func (em *EvaluationMetrics) CompactReport() string {
	return fmt.Sprintf("P:%.1f%% R:%.1f%% F1:%.1f%% (TP:%d FP:%d TN:%d FN:%d)",
		em.Precision()*100,
		em.Recall()*100,
		em.F1Score()*100,
		em.TruePositives,
		em.FalsePositives,
		em.TrueNegatives,
		em.FalseNegatives,
	)
}

// EvaluationResult represents the complete result of an evaluation run
type EvaluationResult struct {
	DetectorName   string             `json:"detector_name"`
	TestSetName    string             `json:"test_set_name"`
	Timestamp      time.Time          `json:"timestamp"`
	Metrics        *EvaluationMetrics `json:"metrics"`
	ExecutionTime  time.Duration      `json:"execution_time"`
	TestCaseCount  int                `json:"test_case_count"`
	Configuration  map[string]any     `json:"configuration,omitempty"`
	Notes          string             `json:"notes,omitempty"`
}

// ComparisonReport compares multiple evaluation results
type ComparisonReport struct {
	Results []EvaluationResult `json:"results"`
	Summary string             `json:"summary"`
}

// GenerateComparison creates a comparison report between multiple results
func GenerateComparison(results []EvaluationResult) *ComparisonReport {
	if len(results) == 0 {
		return &ComparisonReport{
			Results: results,
			Summary: "No results to compare",
		}
	}

	summary := fmt.Sprintf("Comparison of %d detector configurations:\n\n", len(results))

	// Find best performing detector for each metric
	bestPrecision := 0.0
	bestRecall := 0.0
	bestF1 := 0.0
	bestPrecisionDetector := ""
	bestRecallDetector := ""
	bestF1Detector := ""

	for _, result := range results {
		precision := result.Metrics.Precision()
		recall := result.Metrics.Recall()
		f1 := result.Metrics.F1Score()

		if precision > bestPrecision {
			bestPrecision = precision
			bestPrecisionDetector = result.DetectorName
		}
		if recall > bestRecall {
			bestRecall = recall
			bestRecallDetector = result.DetectorName
		}
		if f1 > bestF1 {
			bestF1 = f1
			bestF1Detector = result.DetectorName
		}

		summary += fmt.Sprintf("%s: %s\n", result.DetectorName, result.Metrics.CompactReport())
	}

	summary += fmt.Sprintf(`
Best Performance:
- Precision: %.1f%% (%s)
- Recall:    %.1f%% (%s)
- F1-Score:  %.1f%% (%s)
`,
		bestPrecision*100, bestPrecisionDetector,
		bestRecall*100, bestRecallDetector,
		bestF1*100, bestF1Detector,
	)

	return &ComparisonReport{
		Results: results,
		Summary: summary,
	}
}

// QualityAssessment provides overall quality assessment
type QualityAssessment struct {
	Grade       string  `json:"grade"`        // A, B, C, D, F
	Score       float64 `json:"score"`        // 0-100
	Strengths   []string `json:"strengths"`
	Weaknesses  []string `json:"weaknesses"`
	Recommendation string `json:"recommendation"`
}

// AssessQuality provides an overall quality assessment based on metrics
func AssessQuality(metrics *EvaluationMetrics) *QualityAssessment {
	precision := metrics.Precision()
	recall := metrics.Recall()
	f1 := metrics.F1Score()

	// Calculate weighted score (F1 is most important, then precision, then recall)
	score := (f1 * 0.5) + (precision * 0.3) + (recall * 0.2)

	assessment := &QualityAssessment{
		Score: score * 100,
	}

	// Assign grade based on score
	switch {
	case score >= 0.90:
		assessment.Grade = "A"
	case score >= 0.80:
		assessment.Grade = "B"
	case score >= 0.70:
		assessment.Grade = "C"
	case score >= 0.60:
		assessment.Grade = "D"
	default:
		assessment.Grade = "F"
	}

	// Analyze strengths and weaknesses
	if precision >= 0.85 {
		assessment.Strengths = append(assessment.Strengths, "High precision - few false positives")
	} else if precision <= 0.60 {
		assessment.Weaknesses = append(assessment.Weaknesses, "Low precision - many false positives")
	}

	if recall >= 0.85 {
		assessment.Strengths = append(assessment.Strengths, "High recall - catches most PI")
	} else if recall <= 0.70 {
		assessment.Weaknesses = append(assessment.Weaknesses, "Low recall - missing PI")
	}

	if f1 >= 0.80 {
		assessment.Strengths = append(assessment.Strengths, "Well-balanced precision and recall")
	} else if f1 <= 0.65 {
		assessment.Weaknesses = append(assessment.Weaknesses, "Poor balance between precision and recall")
	}

	// Generate recommendation
	if precision > recall+0.15 {
		assessment.Recommendation = "Consider tuning to improve recall without significantly impacting precision"
	} else if recall > precision+0.15 {
		assessment.Recommendation = "Consider adding validation layers to reduce false positives"
	} else if f1 < 0.70 {
		assessment.Recommendation = "Both precision and recall need improvement - review detection patterns"
	} else {
		assessment.Recommendation = "Performance is good - focus on edge cases and optimization"
	}

	return assessment
}