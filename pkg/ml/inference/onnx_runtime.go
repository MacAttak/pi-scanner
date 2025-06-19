package inference

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	ort "github.com/yalue/onnxruntime_go"
)

// ONNXRuntime wraps the ONNX Runtime for ML inference
type ONNXRuntime struct {
	initialized bool
	mu          sync.RWMutex
}

// ModelConfig holds configuration for ONNX model loading
type ModelConfig struct {
	ModelPath    string   `json:"model_path"`
	InputNames   []string `json:"input_names"`
	OutputNames  []string `json:"output_names"`
	MaxTokens    int      `json:"max_tokens"`
	BatchSize    int      `json:"batch_size"`
	UseGPU       bool     `json:"use_gpu"`
	NumThreads   int      `json:"num_threads"`
}

// InferenceInput represents input data for model inference
type InferenceInput struct {
	InputIDs      []int64 `json:"input_ids"`
	AttentionMask []int64 `json:"attention_mask"`
	TokenTypeIDs  []int64 `json:"token_type_ids,omitempty"`
}

// InferenceOutput represents output data from model inference
type InferenceOutput struct {
	Logits      [][]float32 `json:"logits"`
	Predictions []string    `json:"predictions"`
	Confidence  []float32   `json:"confidence"`
}

// ONNXModel wraps an ONNX model session
type ONNXModel struct {
	session    *ort.Session[float32]
	config     ModelConfig
	inputTensors []*ort.Tensor[int64]
	outputTensors []*ort.Tensor[float32]
}

// Tensor represents a generic tensor interface
type Tensor interface {
	Destroy() error
}

// NewONNXRuntime creates a new ONNX runtime instance
func NewONNXRuntime() *ONNXRuntime {
	return &ONNXRuntime{
		initialized: false,
	}
}

// Initialize sets up the ONNX runtime environment
func (r *ONNXRuntime) Initialize() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.initialized {
		return fmt.Errorf("ONNX runtime already initialized")
	}

	// Set up the library path first
	err := InitializeONNXRuntime()
	if err != nil {
		return fmt.Errorf("failed to set up ONNX runtime library: %w", err)
	}

	// Initialize ONNX Runtime environment
	err = ort.InitializeEnvironment()
	if err != nil {
		return fmt.Errorf("failed to initialize ONNX runtime environment: %w", err)
	}

	r.initialized = true
	return nil
}

// IsInitialized returns whether the runtime is initialized
func (r *ONNXRuntime) IsInitialized() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.initialized
}

// Cleanup destroys the ONNX runtime environment
func (r *ONNXRuntime) Cleanup() {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.initialized {
		ort.DestroyEnvironment()
		r.initialized = false
	}
}

// LoadModel loads an ONNX model from the specified path
func (r *ONNXRuntime) LoadModel(modelPath string) (*ONNXModel, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if !r.initialized {
		return nil, fmt.Errorf("ONNX runtime not initialized")
	}

	// Validate model path
	if modelPath == "" {
		return nil, fmt.Errorf("model path cannot be empty")
	}

	if !filepath.IsAbs(modelPath) {
		return nil, fmt.Errorf("model path must be absolute")
	}

	if filepath.Ext(modelPath) != ".onnx" {
		return nil, fmt.Errorf("model file must have .onnx extension")
	}

	if _, err := os.Stat(modelPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to load model: file does not exist: %s", modelPath)
	}

	// Create default config
	config := ModelConfig{
		ModelPath:   modelPath,
		InputNames:  []string{"input_ids", "attention_mask"},
		OutputNames: []string{"logits"},
		MaxTokens:   512,
		BatchSize:   1,
		UseGPU:      false,
		NumThreads:  1,
	}

	return r.LoadModelWithConfig(config)
}

// LoadModelWithConfig loads an ONNX model with specific configuration
func (r *ONNXRuntime) LoadModelWithConfig(config ModelConfig) (*ONNXModel, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if !r.initialized {
		return nil, fmt.Errorf("ONNX runtime not initialized")
	}

	err := config.Validate()
	if err != nil {
		return nil, fmt.Errorf("invalid model config: %w", err)
	}

	// Create input tensors
	inputTensors := make([]*ort.Tensor[int64], len(config.InputNames))
	for i := range inputTensors {
		shape := ort.NewShape(int64(config.BatchSize), int64(config.MaxTokens))
		inputTensors[i], err = ort.NewEmptyTensor[int64](shape)
		if err != nil {
			// Cleanup any created tensors
			for j := 0; j < i; j++ {
				inputTensors[j].Destroy()
			}
			return nil, fmt.Errorf("failed to create input tensor %d: %w", i, err)
		}
	}

	// Create output tensors
	outputTensors := make([]*ort.Tensor[float32], len(config.OutputNames))
	for i := range outputTensors {
		// Output shape depends on model architecture
		// For classification models, typically [batch_size, num_classes]
		shape := ort.NewShape(int64(config.BatchSize), 2) // Binary classification
		outputTensors[i], err = ort.NewEmptyTensor[float32](shape)
		if err != nil {
			// Cleanup any created tensors
			for j := 0; j < i; j++ {
				outputTensors[j].Destroy()
			}
			for _, tensor := range inputTensors {
				tensor.Destroy()
			}
			return nil, fmt.Errorf("failed to create output tensor %d: %w", i, err)
		}
	}

	// Convert to generic tensor slices for session creation
	inputTensorPtrs := make([]*ort.Tensor[int64], len(inputTensors))
	outputTensorPtrs := make([]*ort.Tensor[float32], len(outputTensors))
	copy(inputTensorPtrs, inputTensors)
	copy(outputTensorPtrs, outputTensors)

	// Create ONNX session - this will fail until we add the library
	// For now, we'll create a mock session structure
	session := &ort.Session[float32]{}

	model := &ONNXModel{
		session:       session,
		config:        config,
		inputTensors:  inputTensors,
		outputTensors: outputTensors,
	}

	return model, nil
}

// Validate validates the model configuration
func (c *ModelConfig) Validate() error {
	if c.ModelPath == "" {
		return fmt.Errorf("model path cannot be empty")
	}

	if len(c.InputNames) == 0 {
		return fmt.Errorf("input names cannot be empty")
	}

	if len(c.OutputNames) == 0 {
		return fmt.Errorf("output names cannot be empty")
	}

	if c.MaxTokens <= 0 {
		return fmt.Errorf("max tokens must be positive")
	}

	if c.BatchSize <= 0 {
		return fmt.Errorf("batch size must be positive")
	}

	return nil
}

// CreateInputTensors creates tensors from inference input data
func (r *ONNXRuntime) CreateInputTensors(input InferenceInput) ([]Tensor, error) {
	if len(input.InputIDs) != len(input.AttentionMask) {
		return nil, fmt.Errorf("input_ids and attention_mask length mismatch")
	}

	if len(input.TokenTypeIDs) > 0 && len(input.TokenTypeIDs) != len(input.InputIDs) {
		return nil, fmt.Errorf("token_type_ids length mismatch")
	}

	tensors := make([]Tensor, 0, 3)

	// Create input_ids tensor
	shape := ort.NewShape(1, int64(len(input.InputIDs)))
	inputIDsTensor, err := ort.NewTensor(shape, input.InputIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to create input_ids tensor: %w", err)
	}
	tensors = append(tensors, inputIDsTensor)

	// Create attention_mask tensor
	attentionTensor, err := ort.NewTensor(shape, input.AttentionMask)
	if err != nil {
		inputIDsTensor.Destroy()
		return nil, fmt.Errorf("failed to create attention_mask tensor: %w", err)
	}
	tensors = append(tensors, attentionTensor)

	// Create token_type_ids tensor if provided
	if len(input.TokenTypeIDs) > 0 {
		tokenTypeTensor, err := ort.NewTensor(shape, input.TokenTypeIDs)
		if err != nil {
			inputIDsTensor.Destroy()
			attentionTensor.Destroy()
			return nil, fmt.Errorf("failed to create token_type_ids tensor: %w", err)
		}
		tensors = append(tensors, tokenTypeTensor)
	}

	return tensors, nil
}

// Predict runs inference on the model with the given input
func (m *ONNXModel) Predict(ctx context.Context, input InferenceInput) (*InferenceOutput, error) {
	// Validate input dimensions
	if len(input.InputIDs) > m.config.MaxTokens {
		return nil, fmt.Errorf("input exceeds max tokens (%d > %d)", len(input.InputIDs), m.config.MaxTokens)
	}

	// For now, return mock output since we don't have real ONNX integration yet
	// This will be replaced with actual inference once the ONNX library is integrated
	output := &InferenceOutput{
		Logits:      [][]float32{{0.1, 0.9}}, // Mock classification scores
		Predictions: []string{"PI"},
		Confidence:  []float32{0.9},
	}

	return output, nil
}

// Destroy cleans up the model resources
func (m *ONNXModel) Destroy() {
	if m.session != nil {
		// m.session.Destroy() // Will be enabled when ONNX library is integrated
		m.session = nil
	}

	for _, tensor := range m.inputTensors {
		if tensor != nil {
			tensor.Destroy()
		}
	}

	for _, tensor := range m.outputTensors {
		if tensor != nil {
			tensor.Destroy()
		}
	}

	m.inputTensors = nil
	m.outputTensors = nil
}

// SetSharedLibraryPath sets the path to the ONNX Runtime shared library
func SetSharedLibraryPath(path string) error {
	if path == "" {
		return fmt.Errorf("library path cannot be empty")
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("library file does not exist: %s", path)
	}

	ort.SetSharedLibraryPath(path)
	return nil
}

// GetVersion returns the ONNX Runtime version information
func GetVersion() string {
	// This would return the actual ONNX Runtime version when integrated
	return "1.22.0-mock"
}

// IsGPUAvailable checks if GPU acceleration is available
func IsGPUAvailable() bool {
	// This would check for CUDA/GPU availability when integrated
	return false
}