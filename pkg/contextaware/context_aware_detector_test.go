package contextaware

import (
	"context"
	"testing"

	"github.com/MacAttak/pi-scanner/pkg/ast"
	"github.com/MacAttak/pi-scanner/pkg/detection"
	"github.com/MacAttak/pi-scanner/pkg/llm"
)

// MockLLMClient for testing
type MockLLMClient struct {
	responses map[string]string
}

func (m *MockLLMClient) Complete(ctx context.Context, prompt string, options *llm.CompletionOptions) (string, error) {
	// Return predefined responses based on prompt content
	if response, exists := m.responses["default"]; exists {
		return response, nil
	}

	// Default response for validation
	return `{
		"is_valid": true,
		"confidence": 0.85,
		"reasoning": "Based on context analysis",
		"contextual_evidence": ["Located in customer data model", "No test indicators"],
		"false_positive_score": 0.15
	}`, nil
}

func (m *MockLLMClient) Stream(ctx context.Context, prompt string, options *llm.CompletionOptions, callback llm.StreamCallback) error {
	response, _ := m.Complete(ctx, prompt, options)
	return callback(response)
}

func (m *MockLLMClient) GetCapabilities() llm.Capabilities {
	return llm.Capabilities{
		MaxTokens:         4096,
		SupportsStreaming: true,
		SupportsJSON:      true,
		SupportsFunctions: false,
	}
}

func TestContextAwareDetector_BasicFunctionality(t *testing.T) {
	// Create mock base detector
	baseDetector := &MockDetector{
		findings: []detection.Finding{
			{
				Type:       detection.PITypeTFN,
				Match:      "123456789",
				File:       "customer.java",
				Line:       10,
				Column:     20,
				Confidence: 0.9,
			},
		},
		name: "MockDetector",
	}

	// Create mock LLM client
	llmClient := &MockLLMClient{
		responses: make(map[string]string),
	}

	// Create context-aware detector
	config := DefaultContextAwareConfig()
	detector := NewContextAwareDetector(baseDetector, llmClient, config)

	// Test detection
	content := []byte(`
public class Customer {
    private String name;
    private String tfn = "123456789"; // Tax file number
    private String email;
}
`)

	findings, err := detector.Detect(context.Background(), content, "customer.java")
	if err != nil {
		t.Fatalf("Detection failed: %v", err)
	}

	if len(findings) != 1 {
		t.Errorf("Expected 1 finding, got %d", len(findings))
	}
}

func TestContextAwareDetector_TestFileDetection(t *testing.T) {
	baseDetector := &MockDetector{
		findings: []detection.Finding{
			{
				Type:       detection.PITypeTFN,
				Match:      "123456789",
				File:       "customer_test.java",
				Line:       15,
				Column:     10,
				Confidence: 0.9,
			},
		},
		name: "MockDetector",
	}

	llmClient := &MockLLMClient{
		responses: map[string]string{
			"default": `{
				"is_valid": false,
				"confidence": 0.2,
				"reasoning": "Test file with mock data",
				"contextual_evidence": ["Test file detected", "Mock TFN pattern"],
				"false_positive_score": 0.8
			}`,
		},
	}

	config := DefaultContextAwareConfig()
	config.ConfidenceThreshold = 0.5
	detector := NewContextAwareDetector(baseDetector, llmClient, config)

	content := []byte(`
public class CustomerTest {
    @Test
    public void testCustomerValidation() {
        Customer customer = new Customer();
        customer.setTfn("123456789"); // Test TFN
    }
}
`)

	findings, err := detector.Detect(context.Background(), content, "customer_test.java")
	if err != nil {
		t.Fatalf("Detection failed: %v", err)
	}

	// Should filter out test file findings with low confidence
	if len(findings) != 0 {
		t.Errorf("Expected 0 findings for test file, got %d", len(findings))
	}
}

func TestContextAwareDetector_FileTypeDetection(t *testing.T) {
	detector := &ContextAwareDetector{
		config: DefaultContextAwareConfig(),
	}

	tests := []struct {
		filename     string
		expectedType string
		isTest       bool
		isConfig     bool
	}{
		{"CustomerTest.java", "test", true, false},
		{"customer_spec.scala", "test", true, false},
		{"test_payment.py", "test", true, false},
		{"application.properties", "configuration", false, true},
		{"config.yaml", "configuration", false, true},
		{"CustomerModel.java", "data_model", false, false},
		{"PaymentService.scala", "business_logic", false, false},
		{"Helper.java", "source_code", false, false},
	}

	for _, test := range tests {
		fileType := detector.detectFileType(test.filename)
		if fileType != test.expectedType {
			t.Errorf("detectFileType(%s) = %s, expected %s", test.filename, fileType, test.expectedType)
		}

		isTest := detector.isTestFile(test.filename)
		if isTest != test.isTest {
			t.Errorf("isTestFile(%s) = %v, expected %v", test.filename, isTest, test.isTest)
		}

		isConfig := detector.isConfigFile(test.filename)
		if isConfig != test.isConfig {
			t.Errorf("isConfigFile(%s) = %v, expected %v", test.filename, isConfig, test.isConfig)
		}
	}
}

func TestContextAwareDetector_BankingDomainIntegration(t *testing.T) {
	// Test integration with AST analysis for banking domain
	baseDetector := &MockDetector{
		findings: []detection.Finding{
			{
				Type:       detection.PITypeBankAccount,
				Match:      "12-3456 12345678",
				File:       "payment/processor.java",
				Line:       25,
				Column:     15,
				Confidence: 0.8,
			},
		},
		name: "MockDetector",
	}

	llmClient := &MockLLMClient{}

	config := DefaultContextAwareConfig()
	config.BankingDomainMode = true
	detector := NewContextAwareDetector(baseDetector, llmClient, config)

	content := []byte(`
package com.bank.payment;

public class PaymentProcessor {
    public void processPayment(String accountNumber) {
        // Process payment for account 12-3456 12345678
        validateAccount(accountNumber);
    }
}
`)

	ctx := context.Background()
	fileContext := detector.analyzeFileContext(ctx, content, "payment/processor.java")

	// Check AST analysis integration
	if fileContext.Language != "Java" {
		t.Errorf("Expected language Java, got %s", fileContext.Language)
	}

	if fileContext.ASTInfo == nil {
		t.Error("Expected AST info to be populated")
	}
}

func TestContextAwareDetector_SurroundingCodeExtraction(t *testing.T) {
	detector := &ContextAwareDetector{
		config: &ContextAwareConfig{
			ContextWindowSize: 2,
		},
	}

	content := []byte(`Line 1
Line 2
Line 3
Line 4 - Finding here
Line 5
Line 6
Line 7`)

	surroundingCode := detector.extractSurroundingCode(content, 4, 4)

	// Should include lines 2-6 (2 lines before and after)
	expected := `  2: Line 2
  3: Line 3
> 4: Line 4 - Finding here
  5: Line 5
  6: Line 6`

	if surroundingCode != expected {
		t.Errorf("Unexpected surrounding code:\nGot:\n%s\nExpected:\n%s", surroundingCode, expected)
	}
}

func TestContextAwareDetector_RiskAssessment(t *testing.T) {
	detector := &ContextAwareDetector{
		config: DefaultContextAwareConfig(),
	}

	// Test risk assessment for different contexts
	testCases := []struct {
		name              string
		finding           ContextualFinding
		expectedRiskLevel string
	}{
		{
			name: "Test file - low risk",
			finding: ContextualFinding{
				Finding: detection.Finding{Type: detection.PITypeTFN},
				Context: CodeContext{IsTestFile: true},
			},
			expectedRiskLevel: "LOW",
		},
		{
			name: "Critical zone - high risk",
			finding: ContextualFinding{
				Finding: detection.Finding{Type: detection.PITypeTFN},
				Context: CodeContext{
					ASTInfo: &ast.AnalysisResult{
						RiskLevel: ast.RiskLevelCritical,
						RiskZone:  "customer_data",
					},
				},
			},
			expectedRiskLevel: "CRITICAL",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assessment := detector.assessRisk(tc.finding)
			if assessment.RiskLevel != tc.expectedRiskLevel {
				t.Errorf("Expected risk level %s, got %s", tc.expectedRiskLevel, assessment.RiskLevel)
			}
		})
	}
}

func TestContextAwareDetector_ValidationRules(t *testing.T) {
	config := DefaultContextAwareConfig()

	// Check that validation rules are properly configured
	tfnRule, exists := config.ValidationRules["AUSTRALIAN_TAX_FILE_NUMBER"]
	if !exists {
		t.Error("Expected TFN validation rule to exist")
	}

	if tfnRule.MinConfidence != 0.8 {
		t.Errorf("Expected TFN min confidence 0.8, got %f", tfnRule.MinConfidence)
	}

	if len(tfnRule.RequiredContext) == 0 {
		t.Error("Expected TFN to have required context")
	}
}

func TestContextAwareDetector_FalsePositivePatterns(t *testing.T) {
	config := DefaultContextAwareConfig()

	// Check false positive patterns
	expectedPatterns := []string{"test", "example", "mock", "123456789"}

	for _, pattern := range expectedPatterns {
		found := false
		for _, fp := range config.FalsePositivePatterns {
			if fp == pattern {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected false positive pattern '%s' not found", pattern)
		}
	}
}

// MockDetector for testing
type MockDetector struct {
	findings []detection.Finding
	err      error
	name     string
}

func (m *MockDetector) Detect(ctx context.Context, content []byte, filename string) ([]detection.Finding, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.findings, nil
}

func (m *MockDetector) Name() string {
	if m.name == "" {
		return "MockDetector"
	}
	return m.name
}
