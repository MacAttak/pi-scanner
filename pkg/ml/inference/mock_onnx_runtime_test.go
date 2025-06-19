package inference

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMockONNXRuntimeSetup(t *testing.T) {
	t.Run("InitializeEnvironment", func(t *testing.T) {
		runtime := NewMockONNXRuntime()
		
		err := runtime.Initialize()
		assert.NoError(t, err, "Should initialize mock ONNX runtime environment")
		
		defer runtime.Cleanup()
		
		assert.True(t, runtime.IsInitialized(), "Runtime should be initialized")
	})

	t.Run("FailsToInitializeTwice", func(t *testing.T) {
		runtime := NewMockONNXRuntime()
		
		err := runtime.Initialize()
		require.NoError(t, err)
		defer runtime.Cleanup()
		
		err = runtime.Initialize()
		assert.Error(t, err, "Should fail to initialize twice")
		assert.Contains(t, err.Error(), "already initialized")
	})

	t.Run("CleanupWithoutInitialization", func(t *testing.T) {
		runtime := NewMockONNXRuntime()
		
		// Should not panic or error
		runtime.Cleanup()
		assert.False(t, runtime.IsInitialized())
	})
}

func TestMockONNXTensorOperations(t *testing.T) {
	runtime := NewMockONNXRuntime()
	err := runtime.Initialize()
	require.NoError(t, err)
	defer runtime.Cleanup()

	t.Run("CreateInputTensors", func(t *testing.T) {
		input := InferenceInput{
			InputIDs:      []int64{101, 2023, 2003, 1037, 3231, 102},
			AttentionMask: []int64{1, 1, 1, 1, 1, 1},
		}

		tensors, err := runtime.CreateInputTensors(input)
		assert.NoError(t, err, "Should create input tensors")
		assert.NotNil(t, tensors)
		assert.Len(t, tensors, 2, "Should create 2 tensors")
		
		// Cleanup tensors
		for _, tensor := range tensors {
			if tensor != nil {
				err := tensor.Destroy()
				assert.NoError(t, err, "Should destroy tensor without error")
			}
		}
	})

	t.Run("CreateInputTensorsWithTokenTypes", func(t *testing.T) {
		input := InferenceInput{
			InputIDs:      []int64{101, 2023, 2003, 1037, 3231, 102},
			AttentionMask: []int64{1, 1, 1, 1, 1, 1},
			TokenTypeIDs:  []int64{0, 0, 0, 0, 0, 0},
		}

		tensors, err := runtime.CreateInputTensors(input)
		assert.NoError(t, err, "Should create input tensors")
		assert.NotNil(t, tensors)
		assert.Len(t, tensors, 3, "Should create 3 tensors")
		
		// Cleanup tensors
		for _, tensor := range tensors {
			if tensor != nil {
				err := tensor.Destroy()
				assert.NoError(t, err, "Should destroy tensor without error")
			}
		}
	})

	t.Run("InvalidInputData", func(t *testing.T) {
		input := InferenceInput{
			InputIDs:      []int64{101, 2023, 2003},
			AttentionMask: []int64{1, 1}, // Mismatched length
		}

		tensors, err := runtime.CreateInputTensors(input)
		assert.Error(t, err, "Should fail with mismatched input lengths")
		assert.Nil(t, tensors)
		assert.Contains(t, err.Error(), "mismatch")
	})

	t.Run("InvalidTokenTypeIDs", func(t *testing.T) {
		input := InferenceInput{
			InputIDs:      []int64{101, 2023, 2003},
			AttentionMask: []int64{1, 1, 1},
			TokenTypeIDs:  []int64{0, 0}, // Mismatched length
		}

		tensors, err := runtime.CreateInputTensors(input)
		assert.Error(t, err, "Should fail with mismatched token type IDs")
		assert.Nil(t, tensors)
		assert.Contains(t, err.Error(), "mismatch")
	})
}

func TestMockONNXModelInference(t *testing.T) {
	runtime := NewMockONNXRuntime()
	err := runtime.Initialize()
	require.NoError(t, err)
	defer runtime.Cleanup()

	t.Run("MockModelPrediction", func(t *testing.T) {
		// Create a mock model directly since we can't load real ONNX files
		config := ModelConfig{
			ModelPath:   "/tmp/test.onnx",
			InputNames:  []string{"input_ids", "attention_mask"},
			OutputNames: []string{"logits"},
			MaxTokens:   512,
			BatchSize:   1,
		}

		model := &MockONNXModel{
			config:  config,
			tensors: make([]*MockTensor, 0),
		}

		input := InferenceInput{
			InputIDs:      []int64{101, 2023, 2003, 1037, 3231, 102},
			AttentionMask: []int64{1, 1, 1, 1, 1, 1},
		}

		output, err := model.Predict(context.Background(), input)
		assert.NoError(t, err, "Should predict without error")
		assert.NotNil(t, output, "Output should not be nil")
		assert.NotEmpty(t, output.Logits, "Should have logits")
		assert.NotEmpty(t, output.Predictions, "Should have predictions")
		assert.NotEmpty(t, output.Confidence, "Should have confidence scores")
		assert.Equal(t, "PI", output.Predictions[0], "Should predict PI")
		assert.Greater(t, output.Confidence[0], float32(0.5), "Should have high confidence")

		model.Destroy()
	})

	t.Run("InputTooLong", func(t *testing.T) {
		config := ModelConfig{
			ModelPath:   "/tmp/test.onnx",
			InputNames:  []string{"input_ids", "attention_mask"},
			OutputNames: []string{"logits"},
			MaxTokens:   5, // Limit to 5 tokens
			BatchSize:   1,
		}

		model := &MockONNXModel{
			config:  config,
			tensors: make([]*MockTensor, 0),
		}

		input := InferenceInput{
			InputIDs:      []int64{101, 2023, 2003, 1037, 3231, 102}, // 6 tokens
			AttentionMask: []int64{1, 1, 1, 1, 1, 1},
		}

		output, err := model.Predict(context.Background(), input)
		assert.Error(t, err, "Should fail with input too long")
		assert.Nil(t, output)
		assert.Contains(t, err.Error(), "exceeds max tokens")
	})
}

func TestMockTensorLifecycle(t *testing.T) {
	t.Run("TensorDestroy", func(t *testing.T) {
		tensor := &MockTensor{
			data:  []int64{1, 2, 3},
			shape: []int64{1, 3},
		}

		err := tensor.Destroy()
		assert.NoError(t, err, "Should destroy tensor without error")
		assert.True(t, tensor.closed, "Tensor should be marked as closed")
		assert.Nil(t, tensor.data, "Data should be cleared")
		assert.Nil(t, tensor.shape, "Shape should be cleared")
	})

	t.Run("DoubleTensorDestroy", func(t *testing.T) {
		tensor := &MockTensor{
			data:  []int64{1, 2, 3},
			shape: []int64{1, 3},
		}

		err := tensor.Destroy()
		require.NoError(t, err)

		err = tensor.Destroy()
		assert.Error(t, err, "Should fail to destroy tensor twice")
		assert.Contains(t, err.Error(), "already destroyed")
	})
}

func TestMockONNXErrorHandling(t *testing.T) {
	t.Run("UninitializedRuntime", func(t *testing.T) {
		runtime := NewMockONNXRuntime()
		// Don't initialize
		
		_, err := runtime.LoadModel("/tmp/test.onnx")
		assert.Error(t, err, "Should fail when runtime not initialized")
		assert.Contains(t, err.Error(), "not initialized")
	})

	t.Run("InvalidModelPath", func(t *testing.T) {
		runtime := NewMockONNXRuntime()
		err := runtime.Initialize()
		require.NoError(t, err)
		defer runtime.Cleanup()
		
		invalidPaths := []string{
			"",
			"relative/path/model.onnx",
			"/nonexistent/path/model.onnx",
			"/tmp/invalid-extension.txt",
		}
		
		for _, path := range invalidPaths {
			_, err := runtime.LoadModel(path)
			assert.Error(t, err, "Should fail to load invalid path: %s", path)
		}
	})
}

// Benchmark tests for mock operations
func BenchmarkMockTensorCreation(b *testing.B) {
	runtime := NewMockONNXRuntime()
	err := runtime.Initialize()
	if err != nil {
		b.Fatalf("Failed to initialize runtime: %v", err)
	}
	defer runtime.Cleanup()
	
	input := InferenceInput{
		InputIDs:      make([]int64, 512),
		AttentionMask: make([]int64, 512),
	}
	
	// Fill with dummy data
	for i := range input.InputIDs {
		input.InputIDs[i] = int64(i % 1000)
		input.AttentionMask[i] = 1
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tensors, err := runtime.CreateInputTensors(input)
		if err != nil {
			b.Fatalf("Failed to create tensors: %v", err)
		}
		
		// Cleanup
		for _, tensor := range tensors {
			if tensor != nil {
				err := tensor.Destroy()
				if err != nil {
					b.Fatalf("Failed to destroy tensor: %v", err)
				}
			}
		}
	}
}

func BenchmarkMockInference(b *testing.B) {
	config := ModelConfig{
		ModelPath:   "/tmp/test.onnx",
		InputNames:  []string{"input_ids", "attention_mask"},
		OutputNames: []string{"logits"},
		MaxTokens:   512,
		BatchSize:   1,
	}

	model := &MockONNXModel{
		config:  config,
		tensors: make([]*MockTensor, 0),
	}
	defer model.Destroy()

	input := InferenceInput{
		InputIDs:      make([]int64, 128),
		AttentionMask: make([]int64, 128),
	}
	
	// Fill with dummy data
	for i := range input.InputIDs {
		input.InputIDs[i] = int64(i % 1000)
		input.AttentionMask[i] = 1
	}
	
	ctx := context.Background()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := model.Predict(ctx, input)
		if err != nil {
			b.Fatalf("Failed to predict: %v", err)
		}
	}
}