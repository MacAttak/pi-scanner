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

// Ensure LMStudioClient implements both interfaces
var _ detection.LLMValidator = (*LMStudioClient)(nil)
var _ Client = (*LMStudioClient)(nil)

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

// Complete implements the Client interface for generic completions
func (c *LMStudioClient) Complete(ctx context.Context, prompt string, options *CompletionOptions) (string, error) {
	if options == nil {
		options = &CompletionOptions{
			MaxTokens:   c.config.MaxTokens,
			Temperature: c.config.Temperature,
		}
	}

	resp, err := c.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model: c.config.Model,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleUser,
				Content: prompt,
			},
		},
		MaxTokens:   options.MaxTokens,
		Temperature: options.Temperature,
		TopP:        options.TopP,
		Stop:        options.Stop,
	})

	if err != nil {
		return "", fmt.Errorf("LLM completion failed: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no completion choices returned")
	}

	return resp.Choices[0].Message.Content, nil
}

// Stream implements the Client interface for streaming completions
func (c *LMStudioClient) Stream(ctx context.Context, prompt string, options *CompletionOptions, callback StreamCallback) error {
	// For now, use non-streaming implementation
	response, err := c.Complete(ctx, prompt, options)
	if err != nil {
		return err
	}
	return callback(response)
}

// GetCapabilities implements the Client interface
func (c *LMStudioClient) GetCapabilities() Capabilities {
	return Capabilities{
		MaxTokens:         c.config.MaxTokens,
		SupportsStreaming: true,
		SupportsJSON:      true,
		SupportsFunctions: false,
	}
}

// createValidationPrompt creates the prompt for LLM validation
func (c *LMStudioClient) createValidationPrompt(req detection.LLMValidationRequest) string {
	var sb strings.Builder

	sb.WriteString("## Finding Details\n\n")
	sb.WriteString(fmt.Sprintf("**File Path:** `%s`\n", req.FilePath))
	sb.WriteString(fmt.Sprintf("**File Type:** %s", req.FileType))
	if req.IsTestFile {
		sb.WriteString(" (TEST FILE)")
	}
	sb.WriteString("\n")
	sb.WriteString(fmt.Sprintf("**PI Type:** %s\n", req.Finding.Type))
	sb.WriteString(fmt.Sprintf("**Detected Value:** `%s`\n", req.Finding.Match))
	sb.WriteString(fmt.Sprintf("**Location:** Line %d, Column %d\n", req.Finding.Line, req.Finding.Column))
	sb.WriteString(fmt.Sprintf("**Initial Risk:** %s\n", req.Finding.RiskLevel))
	sb.WriteString(fmt.Sprintf("**Pattern Confidence:** %.2f\n", req.Finding.Confidence))

	// Add validation status if available
	if req.Finding.Validated {
		sb.WriteString("**Format Validation:** Passed\n")
	} else if req.Finding.ValidationError != "" {
		sb.WriteString(fmt.Sprintf("**Format Validation:** Failed (%s)\n", req.Finding.ValidationError))
	}

	// Add test file notice
	if req.IsTestFile {
		sb.WriteString("\n**Note: This is a test file** - PI found in test files is often synthetic test data.\n")
	}

	sb.WriteString("\n## Code Context\n\n")
	sb.WriteString("The following shows the detected PI value in its surrounding code context:\n\n")
	sb.WriteString("```" + req.FileType + "\n")
	sb.WriteString(req.Context)
	sb.WriteString("\n```\n\n")

	// Add specific guidance based on PI type
	sb.WriteString("## Analysis Guidelines\n\n")
	switch req.Finding.Type {
	case detection.PITypeTFN:
		sb.WriteString("- TFNs are 9-digit tax file numbers, highly sensitive under Australian law\n")
		sb.WriteString("- Check if this follows the pattern: XXX XXX XXX\n")
		sb.WriteString("- Sequential numbers (123456789) are often test data\n")
	case detection.PITypeMedicare:
		sb.WriteString("- Medicare numbers are 10-11 digits, very sensitive health identifiers\n")
		sb.WriteString("- Format: XXXX XXXXX X (optional 11th digit)\n")
		sb.WriteString("- Check if the number could be a real Medicare card\n")
	case detection.PITypeDriverLicense:
		sb.WriteString("- Australian driver licenses vary by state\n")
		sb.WriteString("- Often 8-10 digits or alphanumeric combinations\n")
		sb.WriteString("- Context keywords like 'license' or 'DL' increase likelihood\n")
	case detection.PITypeCreditCard:
		sb.WriteString("- Credit card numbers are 13-19 digits with specific patterns\n")
		sb.WriteString("- Test cards often use known test numbers (4111111111111111)\n")
		sb.WriteString("- Check for Luhn algorithm validity\n")
	}

	sb.WriteString("\nPlease analyze this finding using the Chain-of-Thought approach and provide your assessment in JSON format.")

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

const systemPrompt = `You are an expert security analyst specializing in personally identifiable information (PI) detection in code repositories, with deep knowledge of Australian privacy regulations.

Your task is to carefully analyze potential PI findings and determine if they represent actual privacy risks or false positives.

## Analysis Framework

Use a step-by-step Chain-of-Thought approach:

1. **Context Analysis**
   - Examine the file path and name
   - Identify the type of file (test, config, source, docs)
   - Consider the broader module/package purpose

2. **Code Pattern Analysis**
   - Analyze how the value is used in the code
   - Check if it's hardcoded or dynamically generated
   - Look for indicators of test/example data
   - Examine variable names and comments

3. **Data Characteristics**
   - Evaluate if the format matches real PI
   - Check for sequential patterns (e.g., 123456789)
   - Look for obviously fake values (test, example, dummy)
   - Consider if values are realistic

4. **Security Context**
   - Assess exposure risk (logs, APIs, configs)
   - Check if data is properly protected
   - Consider the sensitivity of the specific PI type

## Common False Positive Patterns

- Test files with example data
- Documentation with format examples
- Sequential numbers in tests (123456789)
- Mock/stub implementations
- Validation regex patterns
- Error messages with format placeholders
- Configuration templates

## Real PI Indicators

- Production configuration files
- Hardcoded values in source code
- Database seed files with real data
- Log files with actual user data
- API responses with unmasked PI
- Comments containing real examples

## Response Format

Provide your analysis as JSON:

{
  "risk": "CRITICAL|HIGH|MEDIUM|LOW",
  "explanation": "Clear explanation including: (1) What the code does, (2) Why this is/isn't real PI, (3) Specific evidence from context",
  "confidence": 0.0-1.0
}

## Risk Level Guidelines

- **CRITICAL**: Confirmed real PI that is exposed or improperly handled
- **HIGH**: Very likely real PI requiring immediate attention
- **MEDIUM**: Possibly real PI, requires human review
- **LOW**: Likely false positive (test data, examples, patterns)

## Important Notes

- Be thorough but concise in explanations
- Focus on actionable insights
- When uncertain, lean towards caution
- Consider regulatory compliance requirements
- Prioritize findings that pose real privacy risks`
