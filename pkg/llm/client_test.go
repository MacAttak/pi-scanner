package llm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/MacAttak/pi-scanner/pkg/detection"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewLMStudioClient(t *testing.T) {
	tests := []struct {
		name   string
		config Config
		want   Config
	}{
		{
			name:   "default config",
			config: Config{},
			want: Config{
				Endpoint:    "http://localhost:1234/v1",
				APIKey:      "lm-studio",
				Model:       "qwen2.5-coder-7b-instruct",
				MaxTokens:   1000,
				Temperature: 0.3,
				Timeout:     30 * time.Second,
			},
		},
		{
			name: "custom config",
			config: Config{
				Endpoint:    "http://localhost:8080/v1",
				APIKey:      "custom-key",
				Model:       "custom-model",
				MaxTokens:   500,
				Temperature: 0.5,
				Timeout:     60 * time.Second,
			},
			want: Config{
				Endpoint:    "http://localhost:8080/v1",
				APIKey:      "custom-key",
				Model:       "custom-model",
				MaxTokens:   500,
				Temperature: 0.5,
				Timeout:     60 * time.Second,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewLMStudioClient(tt.config)
			require.NoError(t, err)
			assert.NotNil(t, client)
			assert.Equal(t, tt.want.Model, client.config.Model)
			assert.Equal(t, tt.want.MaxTokens, client.config.MaxTokens)
			assert.Equal(t, tt.want.Temperature, client.config.Temperature)
		})
	}
}

func TestCreateValidationPrompt(t *testing.T) {
	client := &LMStudioClient{
		config: Config{Model: "test-model"},
	}

	req := detection.LLMValidationRequest{
		Finding: detection.Finding{
			Type:      detection.PITypeTFN,
			Match:     "123456782",
			Line:      10,
			RiskLevel: detection.RiskLevelHigh,
		},
		Context:    "  9: // Test data\n> 10: tfn := '123456782'\n  11: // More code",
		FilePath:   "/test/file.go",
		FileType:   "go",
		IsTestFile: true,
	}

	prompt := client.createValidationPrompt(req)

	assert.Contains(t, prompt, "**File Type:** go (TEST FILE)")
	assert.Contains(t, prompt, "**PI Type:** TFN")
	assert.Contains(t, prompt, "**Detected Value:** `123456782`")
	assert.Contains(t, prompt, "**Location:** Line 10, Column 0")
	assert.Contains(t, prompt, "**Initial Risk:** HIGH")
	assert.Contains(t, prompt, "Note: This is a test file")
	assert.Contains(t, prompt, req.Context)
}

func TestParseResponse(t *testing.T) {
	client := &LMStudioClient{}

	tests := []struct {
		name    string
		content string
		want    *detection.LLMValidationResult
		wantErr bool
	}{
		{
			name:    "json in markdown block",
			content: "Here's my analysis:\n```json\n{\n  \"risk\": \"LOW\",\n  \"explanation\": \"This is test data\",\n  \"confidence\": 0.9\n}\n```",
			want: &detection.LLMValidationResult{
				Risk:        detection.RiskLevelLow,
				Explanation: "This is test data",
				Confidence:  0.9,
			},
		},
		{
			name:    "raw json",
			content: `{"risk": "HIGH", "explanation": "Real PI", "confidence": 0.95}`,
			want: &detection.LLMValidationResult{
				Risk:        detection.RiskLevelHigh,
				Explanation: "Real PI",
				Confidence:  0.95,
			},
		},
		{
			name:    "invalid json",
			content: "This is not JSON",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := client.parseResponse(tt.content)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want.Risk, result.Risk)
			assert.Equal(t, tt.want.Explanation, result.Explanation)
			assert.Equal(t, tt.want.Confidence, result.Confidence)
		})
	}
}

func TestValidateFinding(t *testing.T) {
	// Mock server for testing
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/chat/completions" {
			response := map[string]interface{}{
				"choices": []map[string]interface{}{
					{
						"message": map[string]string{
							"content": `{"risk": "LOW", "explanation": "Test data in test file", "confidence": 0.85}`,
						},
					},
				},
			}
			if err := json.NewEncoder(w).Encode(response); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client, err := NewLMStudioClient(Config{
		Endpoint: server.URL + "/v1",
		APIKey:   "test",
		Model:    "test-model",
	})
	require.NoError(t, err)

	req := detection.LLMValidationRequest{
		Finding: detection.Finding{
			Type:      detection.PITypeTFN,
			Match:     "123456782",
			RiskLevel: detection.RiskLevelHigh,
		},
		Context:    "test context",
		FilePath:   "/test/file_test.go",
		FileType:   "go",
		IsTestFile: true,
	}

	result, err := client.ValidateFinding(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, detection.RiskLevelLow, result.Risk)
	assert.Equal(t, "Test data in test file", result.Explanation)
	assert.Equal(t, 0.85, result.Confidence)
}

func TestHealthCheck(t *testing.T) {
	// Mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/models" {
			response := map[string]interface{}{
				"data": []map[string]string{
					{"id": "test-model"},
				},
			}
			if err := json.NewEncoder(w).Encode(response); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client, err := NewLMStudioClient(Config{
		Endpoint: server.URL + "/v1",
	})
	require.NoError(t, err)

	err = client.HealthCheck(context.Background())
	assert.NoError(t, err)
}

func TestCreateValidationPrompt_WithASTContext(t *testing.T) {
	client, err := NewLMStudioClient(Config{})
	require.NoError(t, err)

	req := detection.LLMValidationRequest{
		Finding: detection.Finding{
			Type:       detection.PITypeTFN,
			Match:      "123456782",
			Line:       25,
			Column:     10,
			RiskLevel:  detection.RiskLevelHigh,
			Confidence: 0.9,
			Validated:  true,
		},
		Context:    "func ProcessPayment() {\n>   tfn := \"123456782\"\n}",
		FilePath:   "/src/payment/processor.go",
		FileType:   "go",
		IsTestFile: false,
		ASTContext: &detection.ASTContext{
			Language:                "go",
			FileType:                "payment",
			RiskZone:                "payment_processing",
			RiskLevel:               "Critical",
			IsTestFile:              false,
			IsConfigFile:            false,
			Classes:                 []string{"PaymentProcessor"},
			Methods:                 []string{"ProcessPayment", "ValidateTransaction"},
			Imports:                 []string{"crypto/aes", "encoding/json"},
			BankingDomainIndicators: []string{"processes_payments", "handles_transactions"},
			SecurityPatterns:        []string{"uses_encryption"},
			EnclosingClass:          "PaymentProcessor",
			EnclosingMethod:         "ProcessPayment",
			NearbyComments:          "// Process customer payment with TFN for tax reporting",
		},
	}

	prompt := client.createValidationPrompt(req)

	// Verify prompt contains AST context information
	assert.Contains(t, prompt, "Structural Context (AST Analysis)")
	assert.Contains(t, prompt, "Language:** go")
	assert.Contains(t, prompt, "Risk Zone:** payment_processing")
	assert.Contains(t, prompt, "Risk Level: Critical")
	assert.Contains(t, prompt, "Classes:** PaymentProcessor")
	assert.Contains(t, prompt, "Within Class:** PaymentProcessor")
	assert.Contains(t, prompt, "Within Method:** ProcessPayment")
	assert.Contains(t, prompt, "Banking Indicators:** processes_payments, handles_transactions")
	assert.Contains(t, prompt, "Security Patterns:** uses_encryption")
	assert.Contains(t, prompt, "Relevant Comments:**")
	assert.Contains(t, prompt, "Process customer payment with TFN for tax reporting")

	// Verify other parts of prompt are still present
	assert.Contains(t, prompt, "PI Type:** TFN")
	assert.Contains(t, prompt, "Detected Value:** `123456782`")
	assert.Contains(t, prompt, "Location:** Line 25, Column 10")
}

func TestCreateValidationPrompt_WithoutASTContext(t *testing.T) {
	client, err := NewLMStudioClient(Config{})
	require.NoError(t, err)

	req := detection.LLMValidationRequest{
		Finding: detection.Finding{
			Type:       detection.PITypeEmail,
			Match:      "test@example.com",
			Line:       10,
			Column:     5,
			RiskLevel:  detection.RiskLevelMedium,
			Confidence: 0.8,
		},
		Context:    "// Contact: test@example.com",
		FilePath:   "/docs/readme.md",
		FileType:   "markdown",
		IsTestFile: false,
		ASTContext: nil, // No AST context
	}

	prompt := client.createValidationPrompt(req)

	// Verify prompt does not contain AST context section
	assert.NotContains(t, prompt, "Structural Context (AST Analysis)")
	assert.NotContains(t, prompt, "Risk Zone:**")
	assert.NotContains(t, prompt, "Within Method:**")

	// Verify other parts of prompt are present
	assert.Contains(t, prompt, "PI Type:** EMAIL")
	assert.Contains(t, prompt, "Detected Value:** `test@example.com`")
	assert.Contains(t, prompt, "File Type:** markdown")
}

func TestCreateValidationPrompt_PartialASTContext(t *testing.T) {
	client, err := NewLMStudioClient(Config{})
	require.NoError(t, err)

	req := detection.LLMValidationRequest{
		Finding: detection.Finding{
			Type:      detection.PITypeCreditCard,
			Match:     "4111111111111111",
			RiskLevel: detection.RiskLevelHigh,
		},
		Context:    "const testCard = \"4111111111111111\"",
		FilePath:   "/test/fixtures.js",
		FileType:   "javascript",
		IsTestFile: true,
		ASTContext: &detection.ASTContext{
			Language:     "javascript",
			RiskZone:     "tests",
			RiskLevel:    "Low",
			IsTestFile:   true,
			IsConfigFile: false,
			// Only partial context - no classes, methods, etc.
		},
	}

	prompt := client.createValidationPrompt(req)

	// Verify AST context section is present
	assert.Contains(t, prompt, "Structural Context (AST Analysis)")
	assert.Contains(t, prompt, "Language:** javascript")
	assert.Contains(t, prompt, "Risk Zone:** tests")

	// Verify empty fields are not included
	assert.NotContains(t, prompt, "Classes:**")
	assert.NotContains(t, prompt, "Within Method:**")
	assert.NotContains(t, prompt, "Banking Indicators:**")
}
