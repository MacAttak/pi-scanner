package llm

import (
	"fmt"
	"strings"
	"time"

	"github.com/MacAttak/pi-scanner/pkg/detection"
)

// ValidatorFactory creates LLM validators with proper security validation
type ValidatorFactory struct {
	endpointValidator *EndpointValidator
	providerRegistry  map[string]ProviderConstructor
}

// ProviderConstructor is a function that creates an LLM validator
type ProviderConstructor func(config Config) (detection.LLMValidator, error)

// GetDefaultFactory returns the default validator factory with security enforcement
func GetDefaultFactory() *ValidatorFactory {
	return NewValidatorFactory()
}

// NewValidatorFactory creates a new validator factory with security enforcement
func NewValidatorFactory() *ValidatorFactory {
	// Create endpoint validator with strict defaults
	endpointValidator := NewEndpointValidator(DefaultValidationConfig())

	factory := &ValidatorFactory{
		endpointValidator: endpointValidator,
		providerRegistry:  make(map[string]ProviderConstructor),
	}

	// Register default providers
	factory.RegisterProvider("lmstudio", func(cfg Config) (detection.LLMValidator, error) {
		return NewLMStudioClient(cfg)
	})
	factory.RegisterProvider("lm-studio", func(cfg Config) (detection.LLMValidator, error) {
		return NewLMStudioClient(cfg)
	})

	// Future providers can be registered here
	// factory.RegisterProvider("ollama", NewOllamaClient)

	return factory
}

// RegisterProvider registers a new LLM provider
func (f *ValidatorFactory) RegisterProvider(name string, constructor ProviderConstructor) {
	f.providerRegistry[strings.ToLower(name)] = constructor
}

// CreateValidator creates an LLM validator with security validation
func (f *ValidatorFactory) CreateValidator(config Config) (detection.LLMValidator, error) {
	// Ensure provider is set
	if config.Provider == "" {
		config.Provider = "lmstudio" // Default provider
	}

	// Validate the configuration
	if err := f.endpointValidator.ValidateConfig(config); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	// Validate provider-specific requirements
	if err := f.endpointValidator.ValidateProviderConfig(config.Provider, config); err != nil {
		return nil, fmt.Errorf("provider validation failed: %w", err)
	}

	// Get the provider constructor
	constructor, exists := f.providerRegistry[strings.ToLower(config.Provider)]
	if !exists {
		return nil, fmt.Errorf("unknown provider: %s", config.Provider)
	}

	// Create the validator
	return constructor(config)
}

// NewValidatorFromConfig creates an LLM validator from detection config with security validation
func NewValidatorFromConfig(config *detection.Config) (detection.LLMValidator, error) {
	if !config.EnableLLMValidation {
		return nil, nil
	}

	llmConfig := Config{
		Enabled:     config.EnableLLMValidation,
		Provider:    config.LLMProvider,
		Endpoint:    config.LLMEndpoint,
		Model:       config.LLMModel,
		APIKey:      config.LLMAPIKey,
		MaxTokens:   config.LLMMaxTokens,
		Temperature: config.LLMTemperature,
		Timeout:     30 * time.Second,
	}

	// Use factory with security validation
	factory := GetDefaultFactory()
	return factory.CreateValidator(llmConfig)
}

// CreateEnhancedDetector wraps a detector with LLM validation if enabled
func CreateEnhancedDetector(baseDetector detection.Detector, config *detection.Config) (detection.Detector, error) {
	if !config.EnableLLMValidation {
		return baseDetector, nil
	}

	validator, err := NewValidatorFromConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create LLM validator: %w", err)
	}

	if validator == nil {
		return baseDetector, nil
	}

	enhancedConfig := &detection.LLMEnhancedConfig{
		Enabled:            true,
		ValidateRiskLevels: config.LLMValidateRisks,
		MaxConcurrency:     3,
		SkipTestFiles:      true,
		ContextLinesBefore: 50,
		ContextLinesAfter:  50,
	}

	return detection.NewLLMEnhancedDetector(baseDetector, validator, enhancedConfig), nil
}

// SecureLLMConfig provides a secure configuration builder
type SecureLLMConfig struct {
	config Config
}

// NewSecureLLMConfig creates a new secure configuration with safe defaults
func NewSecureLLMConfig() *SecureLLMConfig {
	return &SecureLLMConfig{
		config: Config{
			Provider:    "lmstudio",
			Endpoint:    "http://localhost:1234/v1",
			APIKey:      "lm-studio",
			Model:       "qwen2.5-coder-7b-instruct",
			MaxTokens:   1000,
			Temperature: 0.3,
			Timeout:     30 * time.Second,
		},
	}
}

// WithProvider sets the provider (validated against allowed list)
func (c *SecureLLMConfig) WithProvider(provider string) *SecureLLMConfig {
	// Only allow local providers
	allowedProviders := []string{"lmstudio", "lm-studio", "ollama"}
	provider = strings.ToLower(provider)

	for _, allowed := range allowedProviders {
		if provider == allowed {
			c.config.Provider = provider
			break
		}
	}
	return c
}

// WithModel sets the model name
func (c *SecureLLMConfig) WithModel(model string) *SecureLLMConfig {
	c.config.Model = model
	return c
}

// WithMaxTokens sets max tokens (clamped to safe range)
func (c *SecureLLMConfig) WithMaxTokens(tokens int) *SecureLLMConfig {
	if tokens < 100 {
		tokens = 100
	} else if tokens > 4096 {
		tokens = 4096
	}
	c.config.MaxTokens = tokens
	return c
}

// WithTemperature sets temperature (clamped to valid range)
func (c *SecureLLMConfig) WithTemperature(temp float32) *SecureLLMConfig {
	if temp < 0 {
		temp = 0
	} else if temp > 2 {
		temp = 2
	}
	c.config.Temperature = temp
	return c
}

// Build returns the configuration
func (c *SecureLLMConfig) Build() Config {
	return c.config
}
