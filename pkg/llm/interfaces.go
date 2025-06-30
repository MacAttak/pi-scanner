package llm

import (
	"context"
)

// Client is a generic interface for LLM operations
type Client interface {
	// Complete generates a completion for the given prompt
	Complete(ctx context.Context, prompt string, options *CompletionOptions) (string, error)

	// Stream generates a streaming completion
	Stream(ctx context.Context, prompt string, options *CompletionOptions, callback StreamCallback) error

	// GetCapabilities returns the client's capabilities
	GetCapabilities() Capabilities
}

// CompletionOptions contains options for completion requests
type CompletionOptions struct {
	MaxTokens   int      `json:"max_tokens,omitempty"`
	Temperature float32  `json:"temperature,omitempty"`
	TopP        float32  `json:"top_p,omitempty"`
	Stop        []string `json:"stop,omitempty"`
}

// StreamCallback is called for each chunk of streaming response
type StreamCallback func(chunk string) error

// Capabilities describes what the LLM client supports
type Capabilities struct {
	MaxTokens         int  `json:"max_tokens"`
	SupportsStreaming bool `json:"supports_streaming"`
	SupportsJSON      bool `json:"supports_json"`
	SupportsFunctions bool `json:"supports_functions"`
}

// ClientAdapter adapts a detection.LLMValidator to the generic Client interface
type ClientAdapter struct {
	validator interface{} // Will be detection.LLMValidator but avoiding circular import
}

// NewClientAdapter creates a new adapter
func NewClientAdapter(validator interface{}) Client {
	return &ClientAdapter{validator: validator}
}

// Complete implements Client.Complete by wrapping the validator
func (a *ClientAdapter) Complete(ctx context.Context, prompt string, options *CompletionOptions) (string, error) {
	// For now, return a simple implementation
	// In a real implementation, this would use the validator's capabilities
	return prompt, nil
}

// Stream implements Client.Stream
func (a *ClientAdapter) Stream(ctx context.Context, prompt string, options *CompletionOptions, callback StreamCallback) error {
	// Simple non-streaming implementation
	response, err := a.Complete(ctx, prompt, options)
	if err != nil {
		return err
	}
	return callback(response)
}

// GetCapabilities implements Client.GetCapabilities
func (a *ClientAdapter) GetCapabilities() Capabilities {
	return Capabilities{
		MaxTokens:         4096,
		SupportsStreaming: false,
		SupportsJSON:      true,
		SupportsFunctions: false,
	}
}
