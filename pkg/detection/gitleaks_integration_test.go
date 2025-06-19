package detection

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGitleaksDetector_WithCustomConfig(t *testing.T) {
	// Use our custom config file
	configPath := filepath.Join("..", "..", "configs", "gitleaks.toml")
	
	detector, err := NewGitleaksDetector(configPath)
	require.NoError(t, err)

	tests := []struct {
		name         string
		content      string
		filename     string
		expectFindings bool
		expectedType string
		description  string
	}{
		{
			name:         "detect AWS access key",
			content:      `AWS_ACCESS_KEY_ID = "AKIAIOSFODNN7EXAMPLE"`,
			filename:     "config.env",
			expectFindings: true,
			expectedType: "aws-access-key-id",
			description:  "Should detect AWS access key",
		},
		{
			name:         "detect Australian TFN",
			content:      `tfn = "123456782"`,
			filename:     "customer.go",
			expectFindings: true,
			expectedType: "australian-tfn",
			description:  "Should detect TFN",
		},
		{
			name:         "detect Australian ABN",
			content:      `abn = "33051775556"`,
			filename:     "business.py",
			expectFindings: true,
			expectedType: "australian-abn", 
			description:  "Should detect ABN",
		},
		{
			name:         "detect Medicare number",
			content:      `medicare = "2123456701"`,
			filename:     "health.js",
			expectFindings: true,
			expectedType: "australian-medicare",
			description:  "Should detect Medicare number",
		},
		{
			name:         "detect BSB",
			content:      `bsb = "062-000"`,
			filename:     "banking.rb",
			expectFindings: true,
			expectedType: "australian-bsb",
			description:  "Should detect BSB",
		},
		{
			name:         "no findings in normal code",
			content:      `print("Hello, world!")`,
			filename:     "hello.py",
			expectFindings: false,
			description:  "Should not detect PI in normal code",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			findings, err := detector.Detect(context.Background(), []byte(tt.content), tt.filename)
			require.NoError(t, err)
			
			if tt.expectFindings {
				assert.NotEmpty(t, findings, tt.description)
				if len(findings) > 0 {
					assert.Equal(t, tt.filename, findings[0].File)
					assert.Equal(t, "gitleaks-detector", findings[0].DetectorName)
					assert.Greater(t, findings[0].Line, 0, "Line should be set")
					assert.GreaterOrEqual(t, findings[0].Column, 0, "Column should be set")
					// Check that we mapped the rule correctly
					if tt.expectedType != "" {
						// Rule IDs get mapped to our PI types
						expectedMapping := map[string]string{
							"aws-access-key-id": "AWS_ACCESS_KEY",
							"australian-tfn": "TFN",
							"australian-abn": "ABN", 
							"australian-medicare": "MEDICARE",
							"australian-bsb": "BSB",
						}
						if expected, ok := expectedMapping[tt.expectedType]; ok {
							assert.Contains(t, string(findings[0].Type), expected)
						}
					}
				}
			} else {
				assert.Empty(t, findings, tt.description)
			}
		})
	}
}

func TestGitleaksDetector_ContextModifier(t *testing.T) {
	configPath := filepath.Join("..", "..", "configs", "gitleaks.toml")
	detector, err := NewGitleaksDetector(configPath)
	require.NoError(t, err)

	tests := []struct {
		name            string
		content         string
		filename        string
		expectedModifier float32
		description     string
	}{
		{
			name:            "test file gets reduced risk",
			content:         `AWS_ACCESS_KEY_ID = "AKIAIOSFODNN7EXAMPLE"`,
			filename:        "test_config.py",
			expectedModifier: 0.1,
			description:     "Test files should have reduced risk",
		},
		{
			name:            "mock file gets reduced risk",
			content:         `AWS_ACCESS_KEY_ID = "AKIAIOSFODNN7EXAMPLE"`,
			filename:        "mock_data.go",
			expectedModifier: 0.1,
			description:     "Mock files should have reduced risk",
		},
		{
			name:            "example file gets reduced risk",
			content:         `AWS_ACCESS_KEY_ID = "AKIAIOSFODNN7EXAMPLE"`,
			filename:        "example.js",
			expectedModifier: 0.3,
			description:     "Example files should have reduced risk",
		},
		{
			name:            "production file has normal risk",
			content:         `AWS_ACCESS_KEY_ID = "AKIAIOSFODNN7EXAMPLE"`,
			filename:        "config.prod.env",
			expectedModifier: 1.0,
			description:     "Production files should have normal risk",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			findings, err := detector.Detect(context.Background(), []byte(tt.content), tt.filename)
			require.NoError(t, err)
			require.NotEmpty(t, findings, "Should find the secret")
			
			assert.Equal(t, tt.expectedModifier, findings[0].ContextModifier, tt.description)
		})
	}
}

// Test that combines pattern detector and Gitleaks for comprehensive detection
func TestCombinedDetection(t *testing.T) {
	// Create both detectors
	patternDetector := NewDetector()
	
	configPath := filepath.Join("..", "..", "configs", "gitleaks.toml")
	gitleaksDetector, err := NewGitleaksDetector(configPath)
	require.NoError(t, err)
	
	// Content with both API secrets and Australian PI
	content := `
		# Configuration file
		AWS_ACCESS_KEY_ID = "AKIAIOSFODNN7EXAMPLE"
		tfn = "123456782"
		email = "john.doe@example.com"
		medicare = "2123456701"
	`
	
	// Get findings from both detectors
	patternFindings, err := patternDetector.Detect(context.Background(), []byte(content), "config.env")
	require.NoError(t, err)
	
	gitleaksFindings, err := gitleaksDetector.Detect(context.Background(), []byte(content), "config.env")
	require.NoError(t, err)
	
	// Pattern detector should find TFN, email, and Medicare
	assert.GreaterOrEqual(t, len(patternFindings), 3, "Pattern detector should find multiple PI")
	
	// Gitleaks should find AWS key, TFN, and Medicare  
	assert.GreaterOrEqual(t, len(gitleaksFindings), 3, "Gitleaks should find secrets and PI")
	
	// Combined findings should cover all detected items
	allFindings := append(patternFindings, gitleaksFindings...)
	assert.GreaterOrEqual(t, len(allFindings), 5, "Combined should find all items")
	
	// Verify we have different detector names
	detectorNames := make(map[string]bool)
	for _, finding := range allFindings {
		detectorNames[finding.DetectorName] = true
	}
	assert.Contains(t, detectorNames, "pattern-detector")
	assert.Contains(t, detectorNames, "gitleaks-detector")
}