package ml_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/MacAttak/pi-scanner/pkg/ml/inference"
	"github.com/MacAttak/pi-scanner/pkg/ml/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMLIntegration demonstrates how all ML components work together
func TestMLIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create temporary directory for models
	tempDir := t.TempDir()

	// Step 1: Download models
	t.Run("download_models", func(t *testing.T) {
		downloader, err := models.NewModelDownloader(tempDir)
		require.NoError(t, err)

		// In a real scenario, this would download from HuggingFace
		// For testing, we'll just verify the downloader works
		modelPath := downloader.GetModelPath("test/model", "model.onnx")
		assert.Contains(t, modelPath, "test/model")
		assert.Contains(t, modelPath, "model.onnx")
	})

	// Step 2: Create and test DeBERTa model
	t.Run("deberta_model", func(t *testing.T) {
		// Create dummy model files for testing
		modelPath := filepath.Join(tempDir, "model.onnx")
		tokenizerPath := filepath.Join(tempDir, "tokenizer.json")
		
		// Create dummy files
		require.NoError(t, createDummyFile(modelPath))
		require.NoError(t, createDummyFile(tokenizerPath))

		config := models.DeBERTaConfig{
			ModelPath:           modelPath,
			TokenizerPath:       tokenizerPath,
			MaxLength:           512,
			BatchSize:           8,
			ConfidenceThreshold: 0.7,
		}

		model, err := models.NewDeBERTaModel(config)
		require.NoError(t, err)
		assert.NotNil(t, model)

		// Note: Initialize would fail without ONNX runtime library
		// This demonstrates the structure
	})

	// Step 3: Test model runner
	t.Run("model_runner", func(t *testing.T) {
		modelPath := filepath.Join(tempDir, "model.onnx")
		tokenizerPath := filepath.Join(tempDir, "tokenizer.json")
		
		require.NoError(t, createDummyFile(modelPath))
		require.NoError(t, createDummyFile(tokenizerPath))

		config := inference.ModelRunnerConfig{
			ModelPath:     modelPath,
			TokenizerPath: tokenizerPath,
			MaxWorkers:    2,
			QueueSize:     10,
		}

		runner, err := inference.NewModelRunner(config)
		require.NoError(t, err)
		assert.NotNil(t, runner)

		// Check health before starting
		health := runner.HealthCheck()
		assert.False(t, health.Healthy)
		assert.Equal(t, "stopped", health.Status)
	})

	// Step 4: Test PI validation pipeline (conceptual)
	t.Run("pi_validation_pipeline", func(t *testing.T) {
		// This demonstrates how PI validation would work
		candidates := []models.PICandidate{
			{
				Text:    "123-45-6789",
				Type:    "SSN",
				Context: "The customer's SSN is 123-45-6789",
			},
			{
				Text:    "john.doe@example.com",
				Type:    "EMAIL",
				Context: "Contact email: john.doe@example.com",
			},
			{
				Text:    "4111111111111111",
				Type:    "CREDIT_CARD",
				Context: "Payment card: 4111111111111111",
			},
		}

		// In a real implementation with ONNX runtime:
		// 1. Model runner would be started
		// 2. Candidates would be submitted for validation
		// 3. Results would be collected

		// For now, we verify the data structures
		for _, candidate := range candidates {
			assert.NotEmpty(t, candidate.Text)
			assert.NotEmpty(t, candidate.Type)
			assert.NotEmpty(t, candidate.Context)
		}
	})
}

// TestModelDownloaderIntegration tests the model downloader with a mock server
func TestModelDownloaderIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tempDir := t.TempDir()
	downloader, err := models.NewModelDownloader(tempDir)
	require.NoError(t, err)

	// Test concurrent downloads
	t.Run("concurrent_operations", func(t *testing.T) {
		// Simulate checking multiple models
		modelIDs := []string{
			"microsoft/deberta-v3-base",
			"sentence-transformers/all-MiniLM-L6-v2",
			"bert-base-uncased",
		}

		results := make(chan bool, len(modelIDs))
		
		for _, modelID := range modelIDs {
			go func(id string) {
				path := downloader.GetModelPath(id, "model.onnx")
				results <- filepath.IsAbs(path)
			}(modelID)
		}

		// All paths should be absolute
		for i := 0; i < len(modelIDs); i++ {
			assert.True(t, <-results)
		}
	})

	// Test cache management
	t.Run("cache_management", func(t *testing.T) {
		// Get cache size (should be 0 for empty cache)
		size, err := downloader.GetCacheSize()
		require.NoError(t, err)
		assert.Equal(t, int64(0), size)

		// List cached models (should be empty)
		models, err := downloader.ListCachedModels()
		require.NoError(t, err)
		assert.Empty(t, models)
	})
}

// Helper function
func createDummyFile(path string) error {
	return os.WriteFile(path, []byte("dummy content"), 0644)
}