package detection

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Mock LLM validator for testing
type mockLLMValidator struct {
	validateFunc func(ctx context.Context, req LLMValidationRequest) (*LLMValidationResult, error)
	healthFunc   func(ctx context.Context) error
}

func (m *mockLLMValidator) ValidateFinding(ctx context.Context, req LLMValidationRequest) (*LLMValidationResult, error) {
	if m.validateFunc != nil {
		return m.validateFunc(ctx, req)
	}
	return &LLMValidationResult{
		Risk:        RiskLevelLow,
		Explanation: "Mock validation",
		Confidence:  0.8,
		Timestamp:   time.Now(),
	}, nil
}

func (m *mockLLMValidator) HealthCheck(ctx context.Context) error {
	if m.healthFunc != nil {
		return m.healthFunc(ctx)
	}
	return nil
}

func TestNewLLMEnhancedDetector(t *testing.T) {
	baseDetector := &mockDetector{name: "test-detector"}
	validator := &mockLLMValidator{}

	tests := []struct {
		name   string
		config *LLMEnhancedConfig
	}{
		{
			name:   "with default config",
			config: nil,
		},
		{
			name: "with custom config",
			config: &LLMEnhancedConfig{
				Enabled:            true,
				ValidateRiskLevels: []RiskLevel{RiskLevelHigh},
				MaxConcurrency:     5,
				SkipTestFiles:      false,
				ContextLinesBefore: 10,
				ContextLinesAfter:  10,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			detector := NewLLMEnhancedDetector(baseDetector, validator, tt.config)
			assert.NotNil(t, detector)
			assert.Equal(t, "test-detector-llm-enhanced", detector.Name())
			assert.NotNil(t, detector.config)
		})
	}
}

func TestLLMEnhancedDetector_Detect(t *testing.T) {
	content := []byte(`package main

func main() {
	// Test data
	tfn := "123456782"
	medicare := "2123456781"
}`)

	findings := []Finding{
		{
			Type:       PITypeTFN,
			Match:      "123456782",
			Line:       5,
			RiskLevel:  RiskLevelHigh,
			Confidence: 0.9,
		},
		{
			Type:       PITypeMedicare,
			Match:      "2123456781",
			Line:       6,
			RiskLevel:  RiskLevelMedium,
			Confidence: 0.8,
		},
	}

	baseDetector := &mockDetector{
		detectFunc: func(ctx context.Context, content []byte, filename string) ([]Finding, error) {
			// Return copies to avoid mutation
			result := make([]Finding, len(findings))
			copy(result, findings)
			return result, nil
		},
	}

	tests := []struct {
		name           string
		validator      *mockLLMValidator
		config         *LLMEnhancedConfig
		filename       string
		expectEnhanced bool
	}{
		{
			name: "successful validation",
			validator: &mockLLMValidator{
				validateFunc: func(ctx context.Context, req LLMValidationRequest) (*LLMValidationResult, error) {
					return &LLMValidationResult{
						Risk:        RiskLevelLow,
						Explanation: "Test data",
						Confidence:  0.9,
					}, nil
				},
			},
			config: &LLMEnhancedConfig{
				Enabled:            true,
				ValidateRiskLevels: []RiskLevel{RiskLevelHigh, RiskLevelMedium},
				MaxConcurrency:     2,
			},
			filename:       "main.go",
			expectEnhanced: true,
		},
		{
			name:      "disabled LLM",
			validator: &mockLLMValidator{},
			config: &LLMEnhancedConfig{
				Enabled: false,
			},
			filename:       "main.go",
			expectEnhanced: false,
		},
		{
			name:      "skip test files",
			validator: &mockLLMValidator{},
			config: &LLMEnhancedConfig{
				Enabled:       true,
				SkipTestFiles: true,
			},
			filename:       "main_test.go",
			expectEnhanced: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			detector := NewLLMEnhancedDetector(baseDetector, tt.validator, tt.config)

			result, err := detector.Detect(context.Background(), content, tt.filename)
			require.NoError(t, err)
			assert.Len(t, result, 2)

			if tt.expectEnhanced {
				// Check that findings were enhanced
				for _, f := range result {
					assert.True(t, f.LLMValidated)
					assert.NotEmpty(t, f.LLMExplanation)
				}
			} else {
				// Check that findings were not enhanced
				for _, f := range result {
					assert.False(t, f.LLMValidated)
				}
			}
		})
	}
}

func TestExtractContext(t *testing.T) {
	content := `line 1
line 2
line 3
line 4
line 5
line 6
line 7
line 8
line 9
line 10`

	finding := Finding{Line: 5}

	tests := []struct {
		name        string
		linesBefore int
		linesAfter  int
		expected    string
	}{
		{
			name:        "extract with context",
			linesBefore: 2,
			linesAfter:  2,
			expected:    "     3: line 3\n     4: line 4\n>    5: line 5\n     6: line 6\n     7: line 7",
		},
		{
			name:        "extract at start",
			linesBefore: 10,
			linesAfter:  2,
			expected:    "     1: line 1\n     2: line 2\n     3: line 3\n     4: line 4\n>    5: line 5\n     6: line 6\n     7: line 7",
		},
		{
			name:        "extract at end",
			linesBefore: 2,
			linesAfter:  10,
			expected:    "     3: line 3\n     4: line 4\n>    5: line 5\n     6: line 6\n     7: line 7\n     8: line 8\n     9: line 9\n    10: line 10",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractContext(content, finding, tt.linesBefore, tt.linesAfter)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsTestFile(t *testing.T) {
	tests := []struct {
		filename string
		expected bool
	}{
		{"main_test.go", true},
		{"test_main.go", true},
		{"main.test.js", true},
		{"main.spec.js", true},
		{"/path/to/test/file.go", true},
		{"/path/to/tests/file.py", true},
		{"TestMain.java", true},
		{"MainTests.java", true},
		{"main_spec.rb", true},
		{"main.go", false},
		{"production.py", false},
		{"config.yaml", false},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			result := isTestFile(tt.filename)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetFileType(t *testing.T) {
	tests := []struct {
		filename string
		expected string
	}{
		{"main.go", "go"},
		{"script.py", "python"},
		{"app.js", "javascript"},
		{"Main.java", "java"},
		{"config.yaml", "yaml"},
		{"data.json", "json"},
		{"unknown.xyz", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			result := getFileType(tt.filename)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Mock detector for testing
type mockDetector struct {
	name       string
	detectFunc func(ctx context.Context, content []byte, filename string) ([]Finding, error)
}

func (m *mockDetector) Detect(ctx context.Context, content []byte, filename string) ([]Finding, error) {
	if m.detectFunc != nil {
		return m.detectFunc(ctx, content, filename)
	}
	return []Finding{}, nil
}

func (m *mockDetector) Name() string {
	return m.name
}
