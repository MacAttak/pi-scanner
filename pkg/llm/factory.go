package llm

import (
	"fmt"
	"time"

	"github.com/MacAttak/pi-scanner/pkg/detection"
)

// NewValidatorFromConfig creates an LLM validator from detection config
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

	switch config.LLMProvider {
	case "lmstudio", "":
		return NewLMStudioClient(llmConfig)
	default:
		return nil, fmt.Errorf("unsupported LLM provider: %s", config.LLMProvider)
	}
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
