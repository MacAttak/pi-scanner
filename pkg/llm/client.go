package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/MacAttak/pi-scanner/pkg/detection"
	openai "github.com/sashabaranov/go-openai"
)

// LMStudioClient implements the LLMValidator interface for LM Studio
type LMStudioClient struct {
	client *openai.Client
	config Config
}

// Config holds LLM client configuration
type Config struct {
	Enabled     bool
	Provider    string
	Endpoint    string
	Model       string
	APIKey      string
	MaxTokens   int
	Temperature float32
	Timeout     time.Duration
}

// NewLMStudioClient creates a new LM Studio client
func NewLMStudioClient(config Config) (*LMStudioClient, error) {
	if config.Endpoint == "" {
		config.Endpoint = "http://localhost:1234/v1"
	}
	if config.APIKey == "" {
		config.APIKey = "lm-studio"
	}
	if config.Model == "" {
		config.Model = "qwen2.5-coder-7b-instruct"
	}
	if config.MaxTokens == 0 {
		config.MaxTokens = 1000
	}
	if config.Temperature == 0 {
		config.Temperature = 0.3
	}
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}

	// Validate endpoint security
	validator := NewEndpointValidator(DefaultValidationConfig())
	if err := validator.ValidateConfig(config); err != nil {
		return nil, fmt.Errorf("invalid LLM configuration: %w", err)
	}

	// Additional provider-specific validation
	if err := validator.ValidateProviderConfig(config.Provider, config); err != nil {
		return nil, fmt.Errorf("provider validation failed: %w", err)
	}

	clientConfig := openai.DefaultConfig(config.APIKey)
	clientConfig.BaseURL = config.Endpoint

	return &LMStudioClient{
		client: openai.NewClientWithConfig(clientConfig),
		config: config,
	}, nil
}

// ValidateFinding validates a PI finding using LLM
func (c *LMStudioClient) ValidateFinding(ctx context.Context, req detection.LLMValidationRequest) (*detection.LLMValidationResult, error) {
	// Create the prompt
	prompt := c.createValidationPrompt(req)

	// Create chat completion request
	chatReq := openai.ChatCompletionRequest{
		Model:       c.config.Model,
		Temperature: c.config.Temperature,
		MaxTokens:   c.config.MaxTokens,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleSystem,
				Content: systemPrompt,
			},
			{
				Role:    openai.ChatMessageRoleUser,
				Content: prompt,
			},
		},
	}

	// Set timeout
	ctxWithTimeout, cancel := context.WithTimeout(ctx, c.config.Timeout)
	defer cancel()

	// Make the request
	resp, err := c.client.CreateChatCompletion(ctxWithTimeout, chatReq)
	if err != nil {
		return nil, fmt.Errorf("LLM request failed: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no response from LLM")
	}

	// Parse the response
	return c.parseResponse(resp.Choices[0].Message.Content)
}

// HealthCheck verifies the LLM service is accessible
func (c *LMStudioClient) HealthCheck(ctx context.Context) error {
	// Simple health check - list models
	_, err := c.client.ListModels(ctx)
	return err
}

// createValidationPrompt creates the prompt for LLM validation
func (c *LMStudioClient) createValidationPrompt(req detection.LLMValidationRequest) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("Analyze this potential PI finding in %s code:\n\n", req.FileType))
	sb.WriteString(fmt.Sprintf("File: %s\n", req.FilePath))
	sb.WriteString(fmt.Sprintf("PI Type: %s\n", req.Finding.Type))
	sb.WriteString(fmt.Sprintf("Match: %s\n", req.Finding.Match))
	sb.WriteString(fmt.Sprintf("Line: %d\n", req.Finding.Line))
	sb.WriteString(fmt.Sprintf("Initial Risk: %s\n", req.Finding.RiskLevel))

	if req.IsTestFile {
		sb.WriteString("\nNote: This is a test file.\n")
	}

	sb.WriteString("\nCode Context:\n```\n")
	sb.WriteString(req.Context)
	sb.WriteString("\n```\n")

	sb.WriteString("\nProvide your assessment in JSON format.")

	return sb.String()
}

// parseResponse parses the LLM response into a validation result
func (c *LMStudioClient) parseResponse(content string) (*detection.LLMValidationResult, error) {
	// Extract JSON from the response (handle markdown code blocks)
	jsonStr := content
	if idx := strings.Index(content, "```json"); idx >= 0 {
		start := idx + 7
		if end := strings.Index(content[start:], "```"); end >= 0 {
			jsonStr = content[start : start+end]
		}
	} else if idx := strings.Index(content, "{"); idx >= 0 {
		// Try to find raw JSON
		if end := strings.LastIndex(content, "}"); end >= idx {
			jsonStr = content[idx : end+1]
		}
	}

	// Parse the JSON response
	var result detection.LLMValidationResult
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return nil, fmt.Errorf("failed to parse LLM response: %w", err)
	}

	result.Timestamp = time.Now()
	return &result, nil
}

const systemPrompt = `You are a security expert reviewing code for Australian personally identifiable information (PI).

Your task is to validate whether a detected pattern is truly PI or a false positive based on the code context.

Consider:
1. Is this actual PI or just example/test data?
2. Is it hardcoded or dynamically generated?
3. What is the surrounding code doing?
4. Is this in production code or test/documentation?

Respond with JSON only:
{
  "risk": "CRITICAL|HIGH|MEDIUM|LOW",
  "explanation": "Brief explanation of your assessment",
  "confidence": 0.0-1.0
}

Risk levels:
- CRITICAL: Real PI that is hardcoded or exposed
- HIGH: Likely real PI that needs attention
- MEDIUM: Possible PI, needs review
- LOW: Likely false positive or test data

Be conservative - when in doubt, maintain or slightly reduce the risk level.`
