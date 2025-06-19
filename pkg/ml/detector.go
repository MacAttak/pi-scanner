package ml

import (
	"context"
	"fmt"
	"sync"

	"github.com/MacAttak/pi-scanner/pkg/detection"
	"github.com/MacAttak/pi-scanner/pkg/ml/inference"
	"github.com/MacAttak/pi-scanner/pkg/ml/tokenization"
)

// MLDetector implements ML-based PI validation using ONNX Runtime and HuggingFace tokenizers
type MLDetector struct {
	runtime   *inference.ONNXRuntime
	model     *inference.ONNXModel
	tokenizer *tokenization.Tokenizer
	config    MLDetectorConfig
	mu        sync.RWMutex
}

// MLDetectorConfig holds configuration for the ML detector
type MLDetectorConfig struct {
	ModelPath         string                         `json:"model_path"`
	TokenizerModel    string                         `json:"tokenizer_model"`
	ConfidenceThreshold float32                      `json:"confidence_threshold"`
	BatchSize         int                            `json:"batch_size"`
	MaxConcurrent     int                            `json:"max_concurrent"`
	EnableGPU         bool                           `json:"enable_gpu"`
	PITypeConfigs     map[string]PITypeConfig        `json:"pi_type_configs"`
}

// PITypeConfig holds PI-type specific configuration
type PITypeConfig struct {
	Enabled             bool    `json:"enabled"`
	ConfidenceThreshold float32 `json:"confidence_threshold"`
	RequireContext      bool    `json:"require_context"`
}

// DefaultMLDetectorConfig returns default configuration
func DefaultMLDetectorConfig() MLDetectorConfig {
	return MLDetectorConfig{
		ModelPath:           "~/.pi-scanner/models/deberta-pi-validator.onnx",
		TokenizerModel:      "microsoft/deberta-v3-base",
		ConfidenceThreshold: 0.85,
		BatchSize:           32,
		MaxConcurrent:       4,
		EnableGPU:           false,
		PITypeConfigs: map[string]PITypeConfig{
			"TFN": {
				Enabled:             true,
				ConfidenceThreshold: 0.90,
				RequireContext:      true,
			},
			"ABN": {
				Enabled:             true,
				ConfidenceThreshold: 0.85,
				RequireContext:      false,
			},
			"MEDICARE": {
				Enabled:             true,
				ConfidenceThreshold: 0.90,
				RequireContext:      true,
			},
			"BSB": {
				Enabled:             true,
				ConfidenceThreshold: 0.80,
				RequireContext:      false,
			},
		},
	}
}

// NewMLDetector creates a new ML-based detector
func NewMLDetector(config MLDetectorConfig) (*MLDetector, error) {
	if config.BatchSize <= 0 {
		config.BatchSize = 32
	}
	if config.MaxConcurrent <= 0 {
		config.MaxConcurrent = 4
	}
	if config.ConfidenceThreshold <= 0 {
		config.ConfidenceThreshold = 0.85
	}

	return &MLDetector{
		config: config,
	}, nil
}

// Initialize sets up the ML detector with runtime and models
func (d *MLDetector) Initialize() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Initialize ONNX Runtime
	d.runtime = inference.NewONNXRuntime()
	err := d.runtime.Initialize()
	if err != nil {
		return fmt.Errorf("failed to initialize ONNX runtime: %w", err)
	}

	// Load the ONNX model
	modelConfig := inference.ModelConfig{
		ModelPath:   d.config.ModelPath,
		InputNames:  []string{"input_ids", "attention_mask", "token_type_ids"},
		OutputNames: []string{"logits"},
		MaxTokens:   512,
		BatchSize:   d.config.BatchSize,
		UseGPU:      d.config.EnableGPU,
		NumThreads:  d.config.MaxConcurrent,
	}

	d.model, err = d.runtime.LoadModelWithConfig(modelConfig)
	if err != nil {
		d.runtime.Cleanup()
		return fmt.Errorf("failed to load ONNX model: %w", err)
	}

	// Initialize tokenizer
	tokenizerConfig := tokenization.TokenizerConfig{
		ModelName:        d.config.TokenizerModel,
		MaxLength:        512,
		Padding:          true,
		Truncation:       true,
		AddSpecialTokens: true,
	}

	d.tokenizer, err = tokenization.NewTokenizer(tokenizerConfig)
	if err != nil {
		d.runtime.Cleanup()
		return fmt.Errorf("failed to create tokenizer: %w", err)
	}

	err = d.tokenizer.Initialize()
	if err != nil {
		d.runtime.Cleanup()
		return fmt.Errorf("failed to initialize tokenizer: %w", err)
	}

	return nil
}

// Close releases all resources
func (d *MLDetector) Close() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	var errs []error

	if d.tokenizer != nil {
		if err := d.tokenizer.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close tokenizer: %w", err))
		}
		d.tokenizer = nil
	}

	if d.model != nil {
		d.model.Destroy()
		d.model = nil
	}

	if d.runtime != nil {
		d.runtime.Cleanup()
		d.runtime = nil
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors closing ML detector: %v", errs)
	}

	return nil
}

// Name returns the detector name
func (d *MLDetector) Name() string {
	return "ml-validator"
}

// Detect validates PI findings using ML
func (d *MLDetector) Detect(ctx context.Context, content []byte, filename string) ([]detection.Finding, error) {
	// ML detector is typically used as a validator, not a primary detector
	// It would be called by the main detection pipeline to validate findings
	return nil, nil
}

// ValidateFinding uses ML to validate a single PI finding
func (d *MLDetector) ValidateFinding(ctx context.Context, finding detection.Finding, fullContent string) (*MLValidationResult, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if d.tokenizer == nil || d.model == nil {
		return nil, fmt.Errorf("ML detector not initialized")
	}

	// Check if this PI type is enabled
	piConfig, exists := d.config.PITypeConfigs[string(finding.Type)]
	if !exists || !piConfig.Enabled {
		return &MLValidationResult{
			IsValid:    false,
			Confidence: 0,
			Reason:     "PI type not configured for ML validation",
		}, nil
	}

	// Extract context around the finding
	contextWindow := 50 // characters before and after
	startOffset := finding.Column - 1
	if startOffset < 0 {
		startOffset = 0
	}
	endOffset := startOffset + len(finding.Match)

	// Tokenize the PI candidate with context
	encoding, err := d.tokenizer.ExtractPIContext(fullContent, startOffset, endOffset, contextWindow)
	if err != nil {
		return nil, fmt.Errorf("failed to tokenize PI context: %w", err)
	}

	// Convert encoding to inference input
	input := inference.InferenceInput{
		InputIDs:      convertToInt64(encoding.IDs),
		AttentionMask: convertToInt64(encoding.AttentionMask),
		TokenTypeIDs:  convertToInt64(encoding.TypeIDs),
	}

	// Run inference
	output, err := d.model.Predict(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to run ML inference: %w", err)
	}

	// Process output
	confidence := output.Confidence[0]
	isValid := confidence >= piConfig.ConfidenceThreshold

	return &MLValidationResult{
		IsValid:      isValid,
		Confidence:   confidence,
		PIType:       string(finding.Type),
		ModelOutput:  output,
		Reason:       d.generateReason(isValid, confidence, piConfig),
	}, nil
}

// ValidateBatch validates multiple findings in batch for efficiency
func (d *MLDetector) ValidateBatch(ctx context.Context, findings []detection.Finding, contents map[string]string) ([]*MLValidationResult, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if d.tokenizer == nil || d.model == nil {
		return nil, fmt.Errorf("ML detector not initialized")
	}

	results := make([]*MLValidationResult, len(findings))
	
	// Process in batches
	for i := 0; i < len(findings); i += d.config.BatchSize {
		end := i + d.config.BatchSize
		if end > len(findings) {
			end = len(findings)
		}
		
		batch := findings[i:end]
		batchResults, err := d.processBatch(ctx, batch, contents)
		if err != nil {
			return nil, fmt.Errorf("failed to process batch: %w", err)
		}
		
		copy(results[i:end], batchResults)
	}
	
	return results, nil
}

// processBatch processes a batch of findings
func (d *MLDetector) processBatch(ctx context.Context, findings []detection.Finding, contents map[string]string) ([]*MLValidationResult, error) {
	// Prepare batch inputs
	candidates := make([]struct {
		Candidate string
		Context   string
		PIType    string
	}, len(findings))
	
	for i, finding := range findings {
		content, exists := contents[finding.File]
		if !exists {
			return nil, fmt.Errorf("content not found for file: %s", finding.File)
		}
		
		// Extract context
		contextWindow := 50
		startOffset := finding.Column - 1
		if startOffset < 0 {
			startOffset = 0
		}
		endOffset := startOffset + len(finding.Match)
		
		contextStart := startOffset - contextWindow
		if contextStart < 0 {
			contextStart = 0
		}
		contextEnd := endOffset + contextWindow
		if contextEnd > len(content) {
			contextEnd = len(content)
		}
		
		candidates[i] = struct {
			Candidate string
			Context   string
			PIType    string
		}{
			Candidate: finding.Match,
			Context:   content[contextStart:contextEnd],
			PIType:    string(finding.Type),
		}
	}
	
	// Tokenize batch
	encodings, err := d.tokenizer.BatchTokenizePICandidates(candidates)
	if err != nil {
		return nil, fmt.Errorf("failed to tokenize batch: %w", err)
	}
	
	// Run inference for each item (batch inference would be more efficient with proper ONNX support)
	results := make([]*MLValidationResult, len(findings))
	for i, encoding := range encodings {
		input := inference.InferenceInput{
			InputIDs:      convertToInt64(encoding.IDs),
			AttentionMask: convertToInt64(encoding.AttentionMask),
			TokenTypeIDs:  convertToInt64(encoding.TypeIDs),
		}
		
		output, err := d.model.Predict(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("failed to run inference for item %d: %w", i, err)
		}
		
		piConfig := d.config.PITypeConfigs[string(findings[i].Type)]
		confidence := output.Confidence[0]
		isValid := confidence >= piConfig.ConfidenceThreshold
		
		results[i] = &MLValidationResult{
			IsValid:      isValid,
			Confidence:   confidence,
			PIType:       string(findings[i].Type),
			ModelOutput:  output,
			Reason:       d.generateReason(isValid, confidence, piConfig),
		}
	}
	
	return results, nil
}

// generateReason generates a human-readable reason for the validation result
func (d *MLDetector) generateReason(isValid bool, confidence float32, config PITypeConfig) string {
	if isValid {
		return fmt.Sprintf("ML validation passed with %.2f%% confidence (threshold: %.2f%%)", 
			confidence*100, config.ConfidenceThreshold*100)
	}
	
	if confidence < 0.5 {
		return fmt.Sprintf("Low confidence (%.2f%%) - likely false positive", confidence*100)
	}
	
	return fmt.Sprintf("Below threshold - confidence: %.2f%%, required: %.2f%%", 
		confidence*100, config.ConfidenceThreshold*100)
}

// MLValidationResult represents the ML validation outcome
type MLValidationResult struct {
	IsValid      bool                        `json:"is_valid"`
	Confidence   float32                     `json:"confidence"`
	PIType       string                      `json:"pi_type"`
	ModelOutput  *inference.InferenceOutput  `json:"model_output,omitempty"`
	Reason       string                      `json:"reason"`
}

// convertToInt64 converts uint32 slice to int64 slice
func convertToInt64(input []uint32) []int64 {
	result := make([]int64, len(input))
	for i, v := range input {
		result[i] = int64(v)
	}
	return result
}

// GetStats returns detector statistics
func (d *MLDetector) GetStats() MLDetectorStats {
	d.mu.RLock()
	defer d.mu.RUnlock()
	
	return MLDetectorStats{
		Initialized:   d.tokenizer != nil && d.model != nil,
		ModelPath:     d.config.ModelPath,
		TokenizerModel: d.config.TokenizerModel,
		BatchSize:     d.config.BatchSize,
		GPUEnabled:    d.config.EnableGPU,
	}
}

// MLDetectorStats represents detector statistics
type MLDetectorStats struct {
	Initialized    bool   `json:"initialized"`
	ModelPath      string `json:"model_path"`
	TokenizerModel string `json:"tokenizer_model"`
	BatchSize      int    `json:"batch_size"`
	GPUEnabled     bool   `json:"gpu_enabled"`
}