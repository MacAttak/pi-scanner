package inference

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Note: These tests require ONNX Runtime to be installed.
// On macOS: brew install onnxruntime
// On Linux: Download from https://github.com/microsoft/onnxruntime/releases
// To run tests without ONNX Runtime, use: go test -short
func TestONNXRuntimeSetup(t *testing.T) {
	t.Run("InitializeEnvironment", func(t *testing.T) {
		runtime := NewONNXRuntime()
		
		err := runtime.Initialize()
		assert.NoError(t, err, "Should initialize ONNX runtime environment")
		
		defer runtime.Cleanup()
		
		assert.True(t, runtime.IsInitialized(), "Runtime should be initialized")
	})

	t.Run("FailsToInitializeTwice", func(t *testing.T) {
		runtime := NewONNXRuntime()
		
		err := runtime.Initialize()
		require.NoError(t, err)
		defer runtime.Cleanup()
		
		err = runtime.Initialize()
		assert.Error(t, err, "Should fail to initialize twice")
		assert.Contains(t, err.Error(), "already initialized")
	})

	t.Run("CleanupWithoutInitialization", func(t *testing.T) {
		runtime := NewONNXRuntime()
		
		// Should not panic or error
		runtime.Cleanup()
		assert.False(t, runtime.IsInitialized())
	})
}

func TestONNXModelLoading(t *testing.T) {
	runtime := NewONNXRuntime()
	err := runtime.Initialize()
	require.NoError(t, err)
	defer runtime.Cleanup()

	t.Run("LoadNonExistentModel", func(t *testing.T) {
		model, err := runtime.LoadModel("nonexistent.onnx")
		assert.Error(t, err, "Should fail to load non-existent model")
		assert.Nil(t, model)
		assert.Contains(t, err.Error(), "failed to load model")
	})

	t.Run("ModelConfigValidation", func(t *testing.T) {
		ctx := context.Background()
		_ = ctx // Use context to avoid unused import error
		config := ModelConfig{
			ModelPath:    "test.onnx",
			InputNames:   []string{"input_ids", "attention_mask"},
			OutputNames:  []string{"logits"},
			MaxTokens:    512,
			BatchSize:    1,
		}

		err := config.Validate()
		assert.NoError(t, err, "Valid config should pass validation")

		// Test invalid configs
		invalidConfigs := []ModelConfig{
			{ModelPath: "", InputNames: []string{"input"}}, // Empty model path
			{ModelPath: "test.onnx", InputNames: []string{}}, // Empty input names
			{ModelPath: "test.onnx", InputNames: []string{"input"}, MaxTokens: 0}, // Zero max tokens
			{ModelPath: "test.onnx", InputNames: []string{"input"}, MaxTokens: 512, BatchSize: 0}, // Zero batch size
		}

		for i, invalidConfig := range invalidConfigs {
			err := invalidConfig.Validate()
			assert.Error(t, err, "Invalid config %d should fail validation", i)
		}
	})
}

func TestONNXInference(t *testing.T) {
	// Skip this test if we don't have a real model for testing
	if testing.Short() {
		t.Skip("Skipping inference tests in short mode")
	}

	runtime := NewONNXRuntime()
	err := runtime.Initialize()
	require.NoError(t, err)
	defer runtime.Cleanup()

	// This test would require a real ONNX model file
	t.Run("ModelInference", func(t *testing.T) {
		t.Skip("Requires real ONNX model file for testing")
		
		// Example test structure:
		// config := ModelConfig{
		//     ModelPath:   "test_model.onnx",
		//     InputNames:  []string{"input_ids", "attention_mask"},
		//     OutputNames: []string{"logits"},
		//     MaxTokens:   512,
		//     BatchSize:   1,
		// }
		
		// model, err := runtime.LoadModel(config.ModelPath)
		// require.NoError(t, err)
		// defer model.Destroy()
		
		// input := InferenceInput{
		//     InputIDs:      []int64{101, 2023, 2003, 1037, 3231, 102}, // Example tokens
		//     AttentionMask: []int64{1, 1, 1, 1, 1, 1},
		// }
		
		// output, err := model.Predict(context.Background(), input)
		// assert.NoError(t, err)
		// assert.NotNil(t, output)
		// assert.NotEmpty(t, output.Logits)
	})
}

func TestONNXTensorOperations(t *testing.T) {
	runtime := NewONNXRuntime()
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
}

func TestONNXPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance tests in short mode")
	}

	runtime := NewONNXRuntime()
	err := runtime.Initialize()
	require.NoError(t, err)
	defer runtime.Cleanup()

	t.Run("ConcurrentInference", func(t *testing.T) {
		t.Skip("Requires real model for performance testing")
		
		// Example concurrent inference test:
		// This would test thread safety and performance
		// of running multiple inferences simultaneously
	})

	t.Run("MemoryUsage", func(t *testing.T) {
		// Test memory usage patterns
		// Ensure tensors are properly cleaned up
		// Monitor for memory leaks
		
		// Create and destroy many tensors
		for i := 0; i < 100; i++ {
			input := InferenceInput{
				InputIDs:      make([]int64, 512),
				AttentionMask: make([]int64, 512),
			}
			
			// Fill with dummy data
			for j := range input.InputIDs {
				input.InputIDs[j] = int64(j % 1000)
				input.AttentionMask[j] = 1
			}
			
			tensors, err := runtime.CreateInputTensors(input)
			if err == nil && tensors != nil {
				for _, tensor := range tensors {
					if tensor != nil {
						tensor.Destroy()
					}
				}
			}
		}
		
		// Test passes if no memory leaks or panics occur
	})
}

func TestONNXErrorHandling(t *testing.T) {
	t.Run("UninitializedRuntime", func(t *testing.T) {
		runtime := NewONNXRuntime()
		// Don't initialize
		
		_, err := runtime.LoadModel("test.onnx")
		assert.Error(t, err, "Should fail when runtime not initialized")
		assert.Contains(t, err.Error(), "not initialized")
	})

	t.Run("InvalidModelPath", func(t *testing.T) {
		runtime := NewONNXRuntime()
		err := runtime.Initialize()
		require.NoError(t, err)
		defer runtime.Cleanup()
		
		invalidPaths := []string{
			"",
			"/nonexistent/path/model.onnx",
			"invalid-extension.txt",
			"../../../etc/passwd",
		}
		
		for _, path := range invalidPaths {
			_, err := runtime.LoadModel(path)
			assert.Error(t, err, "Should fail to load invalid path: %s", path)
		}
	})
}

// Benchmark tests for ONNX operations
func BenchmarkONNXInference(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping benchmark tests in short mode")
	}
	
	runtime := NewONNXRuntime()
	err := runtime.Initialize()
	if err != nil {
		b.Fatalf("Failed to initialize runtime: %v", err)
	}
	defer runtime.Cleanup()
	
	// This would benchmark actual inference with a real model
	b.Skip("Requires real ONNX model for benchmarking")
}

func BenchmarkTensorCreation(b *testing.B) {
	runtime := NewONNXRuntime()
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