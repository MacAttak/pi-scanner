package inference

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/MacAttak/pi-scanner/pkg/ml/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestModelRunner_New(t *testing.T) {
	tests := []struct {
		name      string
		config    ModelRunnerConfig
		wantError bool
		errorMsg  string
	}{
		{
			name: "valid config",
			config: ModelRunnerConfig{
				ModelPath:     "/path/to/model.onnx",
				TokenizerPath: "/path/to/tokenizer.json",
				MaxWorkers:    4,
				QueueSize:     100,
			},
			wantError: false,
		},
		{
			name: "missing model path",
			config: ModelRunnerConfig{
				TokenizerPath: "/path/to/tokenizer.json",
				MaxWorkers:    4,
				QueueSize:     100,
			},
			wantError: true,
			errorMsg:  "model path cannot be empty",
		},
		{
			name: "invalid max workers",
			config: ModelRunnerConfig{
				ModelPath:     "/path/to/model.onnx",
				TokenizerPath: "/path/to/tokenizer.json",
				MaxWorkers:    0,
				QueueSize:     100,
			},
			wantError: true,
			errorMsg:  "max workers must be positive",
		},
		{
			name: "auto-configure workers",
			config: ModelRunnerConfig{
				ModelPath:     "/path/to/model.onnx",
				TokenizerPath: "/path/to/tokenizer.json",
				MaxWorkers:    -1, // Auto-configure
				QueueSize:     100,
			},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runner, err := NewModelRunner(tt.config)
			if tt.wantError {
				assert.Error(t, err)
				assert.Nil(t, runner)
				if tt.errorMsg != "" && err != nil {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, runner)
			}
		})
	}
}

func TestModelRunner_StartStop(t *testing.T) {
	// Create temporary files for testing
	tempDir := t.TempDir()
	modelPath := filepath.Join(tempDir, "model.onnx")
	tokenizerPath := filepath.Join(tempDir, "tokenizer.json")
	
	require.NoError(t, createDummyFile(modelPath))
	require.NoError(t, createDummyFile(tokenizerPath))

	config := ModelRunnerConfig{
		ModelPath:     modelPath,
		TokenizerPath: tokenizerPath,
		MaxWorkers:    2,
		QueueSize:     10,
	}

	runner, err := NewModelRunner(config)
	require.NoError(t, err)

	// Start the runner
	err = runner.Start()
	assert.NoError(t, err)
	assert.True(t, runner.IsRunning())

	// Starting again should fail
	err = runner.Start()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already running")

	// Stop the runner
	err = runner.Stop()
	assert.NoError(t, err)
	assert.False(t, runner.IsRunning())

	// Stopping again should be safe
	err = runner.Stop()
	assert.NoError(t, err)
}

func TestModelRunner_ValidatePI(t *testing.T) {
	// Create temporary files
	tempDir := t.TempDir()
	modelPath := filepath.Join(tempDir, "model.onnx")
	tokenizerPath := filepath.Join(tempDir, "tokenizer.json")
	
	require.NoError(t, createDummyFile(modelPath))
	require.NoError(t, createDummyFile(tokenizerPath))

	config := ModelRunnerConfig{
		ModelPath:     modelPath,
		TokenizerPath: tokenizerPath,
		MaxWorkers:    2,
		QueueSize:     10,
		Timeout:       5 * time.Second,
	}

	runner, err := NewModelRunner(config)
	require.NoError(t, err)

	// Mock the model for testing
	// Note: In real tests, we'd use a proper mock
	runner.model = &mockDeBERTaModel{
		returnValid: true,
		confidence:  0.95,
	}

	err = runner.Start()
	require.NoError(t, err)
	defer runner.Stop()

	ctx := context.Background()

	t.Run("validate single PI", func(t *testing.T) {
		request := ValidationRequest{
			Candidate: models.PICandidate{
				Text:    "123-45-6789",
				Type:    "SSN",
				Context: "SSN: 123-45-6789",
			},
		}

		result, err := runner.ValidatePI(ctx, request)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.True(t, result.IsValid)
		assert.Equal(t, float32(0.95), result.Confidence)
	})

	t.Run("validate with timeout", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(ctx, 10*time.Millisecond)
		defer cancel()

		// Configure mock to delay
		mock := runner.model.(*mockDeBERTaModel)
		mock.delay = 100 * time.Millisecond

		request := ValidationRequest{
			Candidate: models.PICandidate{
				Text: "test",
				Type: "SSN",
			},
		}

		_, err := runner.ValidatePI(ctx, request)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "context deadline exceeded")

		// Reset delay
		mock.delay = 0
	})

	t.Run("runner not started", func(t *testing.T) {
		runner.Stop()

		request := ValidationRequest{
			Candidate: models.PICandidate{
				Text: "test",
				Type: "SSN",
			},
		}

		_, err := runner.ValidatePI(ctx, request)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not running")
	})
}

func TestModelRunner_BatchValidatePI(t *testing.T) {
	// Create temporary files
	tempDir := t.TempDir()
	modelPath := filepath.Join(tempDir, "model.onnx")
	tokenizerPath := filepath.Join(tempDir, "tokenizer.json")
	
	require.NoError(t, createDummyFile(modelPath))
	require.NoError(t, createDummyFile(tokenizerPath))

	config := ModelRunnerConfig{
		ModelPath:     modelPath,
		TokenizerPath: tokenizerPath,
		MaxWorkers:    4,
		QueueSize:     20,
		BatchSize:     5,
	}

	runner, err := NewModelRunner(config)
	require.NoError(t, err)

	// Mock the model
	runner.model = &mockDeBERTaModel{
		returnValid: true,
		confidence:  0.85,
	}

	err = runner.Start()
	require.NoError(t, err)
	defer runner.Stop()

	ctx := context.Background()

	requests := []ValidationRequest{
		{
			Candidate: models.PICandidate{
				Text: "123-45-6789",
				Type: "SSN",
			},
		},
		{
			Candidate: models.PICandidate{
				Text: "john@example.com",
				Type: "EMAIL",
			},
		},
		{
			Candidate: models.PICandidate{
				Text: "4111111111111111",
				Type: "CREDIT_CARD",
			},
		},
	}

	results, err := runner.BatchValidatePI(ctx, requests)
	assert.NoError(t, err)
	assert.Len(t, results, len(requests))

	for i, result := range results {
		assert.NotNil(t, result)
		assert.True(t, result.IsValid)
		assert.Equal(t, requests[i].Candidate.Type, result.PIType)
	}
}

func TestModelRunner_GetStats(t *testing.T) {
	// Create temporary files
	tempDir := t.TempDir()
	modelPath := filepath.Join(tempDir, "model.onnx")
	tokenizerPath := filepath.Join(tempDir, "tokenizer.json")
	
	require.NoError(t, createDummyFile(modelPath))
	require.NoError(t, createDummyFile(tokenizerPath))

	config := ModelRunnerConfig{
		ModelPath:     modelPath,
		TokenizerPath: tokenizerPath,
		MaxWorkers:    2,
		QueueSize:     10,
	}

	runner, err := NewModelRunner(config)
	require.NoError(t, err)

	// Mock the model
	runner.model = &mockDeBERTaModel{
		returnValid: true,
		confidence:  0.9,
	}

	err = runner.Start()
	require.NoError(t, err)
	defer runner.Stop()

	ctx := context.Background()

	// Process some requests
	for i := 0; i < 5; i++ {
		request := ValidationRequest{
			Candidate: models.PICandidate{
				Text: fmt.Sprintf("test-%d", i),
				Type: "SSN",
			},
		}
		_, err := runner.ValidatePI(ctx, request)
		assert.NoError(t, err)
	}

	// Get stats
	stats := runner.GetStats()
	assert.Equal(t, int64(5), stats.TotalRequests)
	assert.Equal(t, int64(5), stats.SuccessfulRequests)
	assert.Equal(t, int64(0), stats.FailedRequests)
	assert.True(t, stats.AverageLatency > 0)
}

func TestModelRunner_HealthCheck(t *testing.T) {
	// Create temporary files
	tempDir := t.TempDir()
	modelPath := filepath.Join(tempDir, "model.onnx")
	tokenizerPath := filepath.Join(tempDir, "tokenizer.json")
	
	require.NoError(t, createDummyFile(modelPath))
	require.NoError(t, createDummyFile(tokenizerPath))

	config := ModelRunnerConfig{
		ModelPath:     modelPath,
		TokenizerPath: tokenizerPath,
		MaxWorkers:    2,
		QueueSize:     10,
	}

	runner, err := NewModelRunner(config)
	require.NoError(t, err)

	// Before starting
	health := runner.HealthCheck()
	assert.False(t, health.Healthy)
	assert.Contains(t, health.Message, "not running")

	// Mock the model
	runner.model = &mockDeBERTaModel{
		returnValid: true,
		confidence:  0.9,
	}

	// After starting
	err = runner.Start()
	require.NoError(t, err)
	defer runner.Stop()

	health = runner.HealthCheck()
	assert.True(t, health.Healthy)
	assert.Equal(t, "running", health.Status)
	assert.Equal(t, 2, health.WorkerCount)
	assert.Equal(t, 0, health.QueueDepth)

	// Perform a validation to test queue depth
	ctx := context.Background()
	request := ValidationRequest{
		Candidate: models.PICandidate{
			Text: "test",
			Type: "SSN",
		},
	}
	
	_, err = runner.ValidatePI(ctx, request)
	assert.NoError(t, err)
}

// Mock implementation for testing
type mockDeBERTaModel struct {
	returnValid bool
	confidence  float32
	delay       time.Duration
}

func (m *mockDeBERTaModel) ValidatePI(ctx context.Context, candidate models.PICandidate) (*models.ValidationResult, error) {
	if m.delay > 0 {
		select {
		case <-time.After(m.delay):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	return &models.ValidationResult{
		IsValid:     m.returnValid,
		Confidence:  m.confidence,
		PIType:      candidate.Type,
		Explanation: "Mock validation",
	}, nil
}

func (m *mockDeBERTaModel) BatchValidatePI(ctx context.Context, candidates []models.PICandidate) ([]*models.ValidationResult, error) {
	results := make([]*models.ValidationResult, len(candidates))
	for i, candidate := range candidates {
		result, err := m.ValidatePI(ctx, candidate)
		if err != nil {
			return nil, err
		}
		results[i] = result
	}
	return results, nil
}

func (m *mockDeBERTaModel) Initialize() error {
	return nil
}

func (m *mockDeBERTaModel) Close() error {
	return nil
}

// Helper function
func createDummyFile(path string) error {
	return os.WriteFile(path, []byte("dummy content"), 0644)
}