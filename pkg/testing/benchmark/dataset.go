package benchmark

import (
	"github.com/MacAttak/pi-scanner/pkg/detection"
)

// BenchmarkDataset represents a collection of test cases for evaluating PI detection
type BenchmarkDataset struct {
	TruePositives []TestCase // Actual PI in code
	TrueNegatives []TestCase // Code without PI
	EdgeCases     []TestCase // Ambiguous cases
	Synthetic     []TestCase // Generated test data
}

// TestCase represents a single test case for evaluation
type TestCase struct {
	ID          string               `json:"id"`
	Code        string               `json:"code"`
	Language    string               `json:"language"`
	PIType      detection.PIType     `json:"pi_type"`
	IsActualPI  bool                 `json:"is_actual_pi"`
	Context     string               `json:"context"` // prod, test, comment, etc.
	Rationale   string               `json:"rationale"` // Why is this PI or not?
	Filename    string               `json:"filename"`
}

// AllCases returns all test cases from the dataset
func (bd *BenchmarkDataset) AllCases() []TestCase {
	var all []TestCase
	all = append(all, bd.TruePositives...)
	all = append(all, bd.TrueNegatives...)
	all = append(all, bd.EdgeCases...)
	all = append(all, bd.Synthetic...)
	return all
}

// GetCasesByContext returns test cases filtered by context
func (bd *BenchmarkDataset) GetCasesByContext(context string) []TestCase {
	var filtered []TestCase
	for _, testCase := range bd.AllCases() {
		if testCase.Context == context {
			filtered = append(filtered, testCase)
		}
	}
	return filtered
}

// GetCasesByPIType returns test cases filtered by PI type
func (bd *BenchmarkDataset) GetCasesByPIType(piType detection.PIType) []TestCase {
	var filtered []TestCase
	for _, testCase := range bd.AllCases() {
		if testCase.PIType == piType {
			filtered = append(filtered, testCase)
		}
	}
	return filtered
}

// Stats returns statistics about the dataset
func (bd *BenchmarkDataset) Stats() DatasetStats {
	stats := DatasetStats{
		TotalCases:    len(bd.AllCases()),
		TruePositives: len(bd.TruePositives),
		TrueNegatives: len(bd.TrueNegatives),
		EdgeCases:     len(bd.EdgeCases),
		Synthetic:     len(bd.Synthetic),
		ByContext:     make(map[string]int),
		ByPIType:      make(map[string]int),
	}

	for _, testCase := range bd.AllCases() {
		stats.ByContext[testCase.Context]++
		stats.ByPIType[string(testCase.PIType)]++
	}

	return stats
}

// DatasetStats provides statistics about the benchmark dataset
type DatasetStats struct {
	TotalCases    int            `json:"total_cases"`
	TruePositives int            `json:"true_positives"`
	TrueNegatives int            `json:"true_negatives"`
	EdgeCases     int            `json:"edge_cases"`
	Synthetic     int            `json:"synthetic"`
	ByContext     map[string]int `json:"by_context"`
	ByPIType      map[string]int `json:"by_pi_type"`
}