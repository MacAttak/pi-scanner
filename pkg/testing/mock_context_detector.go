package testing

import (
	"context"
	"strings"

	"github.com/MacAttak/pi-scanner/pkg/detection"
)

// MockContextDetector wraps a base detector and applies simple context filtering
type MockContextDetector struct {
	baseDetector detection.Detector
}

// NewMockContextDetector creates a detector that filters out obvious false positives
func NewMockContextDetector(base detection.Detector) *MockContextDetector {
	return &MockContextDetector{
		baseDetector: base,
	}
}

// Name returns the detector name
func (mcd *MockContextDetector) Name() string {
	return "mock-context-" + mcd.baseDetector.Name()
}

// Detect runs the base detector and then filters results based on context
func (mcd *MockContextDetector) Detect(ctx context.Context, content []byte, filename string) ([]detection.Finding, error) {
	// Run base detection
	findings, err := mcd.baseDetector.Detect(ctx, content, filename)
	if err != nil {
		return nil, err
	}

	// Filter findings based on simple context rules
	var filtered []detection.Finding
	contentStr := string(content)

	for _, finding := range findings {
		if mcd.shouldKeepFinding(finding, contentStr, filename) {
			// Increase confidence for kept findings
			finding.Confidence = 0.9
			filtered = append(filtered, finding)
		}
	}

	return filtered, nil
}

// shouldKeepFinding determines if a finding should be kept based on context
func (mcd *MockContextDetector) shouldKeepFinding(finding detection.Finding, content, filename string) bool {
	// Filter out findings in comments
	if mcd.isInComment(content, finding) {
		return false
	}

	// Filter out test files
	if mcd.isTestFile(filename) {
		return false
	}

	// Filter out mock data
	if mcd.isMockData(content, finding) {
		return false
	}

	return true
}

// isInComment checks if finding is in a comment
func (mcd *MockContextDetector) isInComment(content string, finding detection.Finding) bool {
	lines := strings.Split(content, "\n")
	if finding.Line <= 0 || finding.Line > len(lines) {
		return false
	}

	line := lines[finding.Line-1]
	
	// Check for comment markers
	if strings.Contains(line, "//") {
		commentStart := strings.Index(line, "//")
		if finding.Column > commentStart {
			return true
		}
	}

	return false
}

// isTestFile checks if filename indicates a test file
func (mcd *MockContextDetector) isTestFile(filename string) bool {
	testPatterns := []string{"_test.go", "test.go", "mock_", "fixture_"}
	
	for _, pattern := range testPatterns {
		if strings.Contains(filename, pattern) {
			return true
		}
	}
	
	return false
}

// isMockData checks if the content suggests mock/test data
func (mcd *MockContextDetector) isMockData(content string, finding detection.Finding) bool {
	lines := strings.Split(content, "\n")
	if finding.Line <= 0 || finding.Line > len(lines) {
		return false
	}

	// Check line and surrounding lines for mock indicators
	start := finding.Line - 2
	if start < 0 {
		start = 0
	}
	end := finding.Line + 1
	if end >= len(lines) {
		end = len(lines) - 1
	}

	context := strings.ToLower(strings.Join(lines[start:end+1], " "))
	
	mockIndicators := []string{
		"mock", "test", "example", "sample", "dummy", "placeholder",
		"for testing", "for unit tests", "test data",
	}

	for _, indicator := range mockIndicators {
		if strings.Contains(context, indicator) {
			return true
		}
	}

	return false
}