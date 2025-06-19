package inference

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// MockONNXRuntime provides a mock implementation for testing without ONNX Runtime library
type MockONNXRuntime struct {
	initialized bool
	mu          sync.RWMutex
}

// MockTensor implements the Tensor interface for testing
type MockTensor struct {
	data   interface{}
	shape  []int64
	closed bool
}

// MockONNXModel provides a mock implementation for testing
type MockONNXModel struct {
	config  ModelConfig
	tensors []*MockTensor
}

// NewMockONNXRuntime creates a new mock ONNX runtime for testing
func NewMockONNXRuntime() *MockONNXRuntime {
	return &MockONNXRuntime{
		initialized: false,
	}
}

// Initialize mock initialization
func (r *MockONNXRuntime) Initialize() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.initialized {
		return fmt.Errorf("ONNX runtime already initialized")
	}

	r.initialized = true
	return nil
}

// IsInitialized returns whether the mock runtime is initialized
func (r *MockONNXRuntime) IsInitialized() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.initialized
}

// Cleanup mock cleanup
func (r *MockONNXRuntime) Cleanup() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.initialized = false
}

// LoadModel mock model loading
func (r *MockONNXRuntime) LoadModel(modelPath string) (*MockONNXModel, error) {
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

	config := ModelConfig{
		ModelPath:   modelPath,
		InputNames:  []string{"input_ids", "attention_mask"},
		OutputNames: []string{"logits"},
		MaxTokens:   512,
		BatchSize:   1,
	}

	return &MockONNXModel{
		config:  config,
		tensors: make([]*MockTensor, 0),
	}, nil
}

// CreateInputTensors creates mock tensors
func (r *MockONNXRuntime) CreateInputTensors(input InferenceInput) ([]Tensor, error) {
	if len(input.InputIDs) != len(input.AttentionMask) {
		return nil, fmt.Errorf("input_ids and attention_mask length mismatch")
	}

	if len(input.TokenTypeIDs) > 0 && len(input.TokenTypeIDs) != len(input.InputIDs) {
		return nil, fmt.Errorf("token_type_ids length mismatch")
	}

	tensors := make([]Tensor, 0, 3)

	// Create mock tensors
	inputIDsTensor := &MockTensor{
		data:  input.InputIDs,
		shape: []int64{1, int64(len(input.InputIDs))},
	}
	tensors = append(tensors, inputIDsTensor)

	attentionTensor := &MockTensor{
		data:  input.AttentionMask,
		shape: []int64{1, int64(len(input.AttentionMask))},
	}
	tensors = append(tensors, attentionTensor)

	if len(input.TokenTypeIDs) > 0 {
		tokenTypeTensor := &MockTensor{
			data:  input.TokenTypeIDs,
			shape: []int64{1, int64(len(input.TokenTypeIDs))},
		}
		tensors = append(tensors, tokenTypeTensor)
	}

	return tensors, nil
}

// Destroy mock tensor destruction
func (t *MockTensor) Destroy() error {
	if t.closed {
		return fmt.Errorf("tensor already destroyed")
	}
	t.closed = true
	t.data = nil
	t.shape = nil
	return nil
}

// Predict mock prediction
func (m *MockONNXModel) Predict(ctx context.Context, input InferenceInput) (*InferenceOutput, error) {
	if len(input.InputIDs) > m.config.MaxTokens {
		return nil, fmt.Errorf("input exceeds max tokens (%d > %d)", len(input.InputIDs), m.config.MaxTokens)
	}

	// Mock output - simulate PI detection
	output := &InferenceOutput{
		Logits:      [][]float32{{0.2, 0.8}}, // Mock classification scores
		Predictions: []string{"PI"},
		Confidence:  []float32{0.8},
	}

	return output, nil
}

// Destroy mock model cleanup
func (m *MockONNXModel) Destroy() {
	for _, tensor := range m.tensors {
		if tensor != nil {
			tensor.Destroy()
		}
	}
	m.tensors = nil
}