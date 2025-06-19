package proximity

import (
	"fmt"
	"testing"
)

// ExampleProximityDetector demonstrates how to use the proximity detector
func ExampleProximityDetector() {
	detector := NewProximityDetector()
	
	// Example 1: Test data
	content1 := "// Test TFN: 123 456 789 for unit testing"
	result1 := detector.AnalyzeContext(content1, "123 456 789", 12, 23)
	fmt.Printf("Test data - Score: %.2f, Is Test: %t, Reason: %s\n", 
		result1.Score, result1.IsTestData, result1.Reason)
	
	// Example 2: Real PI with label
	content2 := "Customer TFN: 123 456 789"
	result2 := detector.AnalyzeContext(content2, "123 456 789", 14, 25)
	fmt.Printf("Real PI - Score: %.2f, Is Test: %t, Reason: %s\n", 
		result2.Score, result2.IsTestData, result2.Reason)
	
	// Example 3: Form field
	content3 := `<input type="text" name="tfn" value="123 456 789">`
	result3 := detector.AnalyzeContext(content3, "123 456 789", 37, 48)
	fmt.Printf("Form field - Score: %.2f, Context: %s, Reason: %s\n", 
		result3.Score, result3.Context, result3.Reason)
	
	// Output:
	// Test data - Score: 0.10, Is Test: true, Reason: test data indicator
	// Real PI - Score: 0.66, Is Test: false, Reason: PI context label detected
	// Form field - Score: 0.80, Context: form, Reason: form field context
}

func TestExampleUsage(t *testing.T) {
	ExampleProximityDetector()
}