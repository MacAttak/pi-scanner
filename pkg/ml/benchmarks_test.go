package ml

import (
	"fmt"
	"testing"

	"github.com/MacAttak/pi-scanner/pkg/ml/inference"
	"github.com/MacAttak/pi-scanner/pkg/ml/models"
	"github.com/MacAttak/pi-scanner/pkg/ml/tokenization"
)

// BenchmarkTokenization benchmarks the tokenization process
func BenchmarkTokenization(b *testing.B) {
	// Skip if tokenizer not available
	b.Skip("Tokenizer benchmarks require actual tokenizer files and full implementation")
}

// BenchmarkModelInference benchmarks the model inference process
func BenchmarkModelInference(b *testing.B) {
	b.Skip("Model inference benchmarks require full implementation")
}

// BenchmarkEndToEndValidation benchmarks the complete validation pipeline
func BenchmarkEndToEndValidation(b *testing.B) {
	b.Skip("End-to-end benchmarks require full implementation")
}

// BenchmarkMemoryUsage benchmarks memory allocation patterns
func BenchmarkMemoryUsage(b *testing.B) {
	b.Skip("Memory usage benchmarks require full implementation")
}

// Example benchmark that can run with current implementation
func BenchmarkONNXRuntimeInitialization(b *testing.B) {
	for i := 0; i < b.N; i++ {
		runtime := inference.NewONNXRuntime()
		err := runtime.Initialize()
		if err != nil {
			b.Fatalf("Failed to initialize ONNX runtime: %v", err)
		}
		runtime.Cleanup()
	}
}

// BenchmarkTokenizerConfig benchmarks tokenizer configuration
func BenchmarkTokenizerConfig(b *testing.B) {
	configs := []tokenization.TokenizerConfig{
		tokenization.DefaultTokenizerConfig(),
		{ModelName: "bert-base-uncased", MaxLength: 256},
		{ModelName: "roberta-base", MaxLength: 512},
	}

	b.Run("ConfigValidation", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			for _, config := range configs {
				if config.MaxLength <= 0 {
					b.Error("Invalid max length")
				}
			}
		}
	})
}

// BenchmarkPICandidate benchmarks PI candidate creation
func BenchmarkPICandidate(b *testing.B) {
	b.Run("CreateCandidate", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = models.PICandidate{
				Text:     fmt.Sprintf("123-45-%04d", i%10000),
				Type:     "SSN",
				Context:  fmt.Sprintf("The SSN 123-45-%04d was found", i%10000),
				StartPos: i * 10,
				EndPos:   i*10 + 11,
			}
		}
	})

	b.Run("BatchCandidates", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			candidates := make([]models.PICandidate, 100)
			for j := 0; j < 100; j++ {
				candidates[j] = models.PICandidate{
					Text:     fmt.Sprintf("123-45-%04d", j),
					Type:     "SSN",
					Context:  fmt.Sprintf("The SSN 123-45-%04d was found", j),
					StartPos: j * 50,
					EndPos:   j*50 + 11,
				}
			}
			_ = candidates
		}
	})
}

// BenchmarkModelConfig benchmarks model configuration operations
func BenchmarkModelConfig(b *testing.B) {
	b.Run("CreateConfig", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = models.DeBERTaConfig{
				ModelPath:           "/path/to/model.onnx",
				TokenizerPath:       "/path/to/tokenizer.json",
				MaxLength:           512,
				BatchSize:           32,
				NumThreads:          4,
				UseGPU:              false,
				ConfidenceThreshold: 0.8,
			}
		}
	})
}

// BenchmarkValidationResult benchmarks validation result creation
func BenchmarkValidationResult(b *testing.B) {
	b.Run("CreateResult", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = models.ValidationResult{
				IsValid:     i%2 == 0,
				Confidence:  float32(i%100) / 100.0,
				PIType:      "SSN",
				Explanation: fmt.Sprintf("Validation result for item %d", i),
			}
		}
	})

	b.Run("BatchResults", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			results := make([]models.ValidationResult, 50)
			for j := 0; j < 50; j++ {
				results[j] = models.ValidationResult{
					IsValid:     j%3 != 0,
					Confidence:  float32(j%100) / 100.0,
					PIType:      "EMAIL",
					Explanation: fmt.Sprintf("Batch validation result %d", j),
				}
			}
			_ = results
		}
	})
}

// BenchmarkConcurrentOperations benchmarks concurrent operations
func BenchmarkConcurrentOperations(b *testing.B) {
	b.Run("ConcurrentConfigCreation", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			i := 0
			for pb.Next() {
				_ = inference.ModelRunnerConfig{
					ModelPath:     fmt.Sprintf("/path/to/model-%d.onnx", i),
					TokenizerPath: fmt.Sprintf("/path/to/tokenizer-%d.json", i),
					MaxWorkers:    4,
					QueueSize:     100,
				}
				i++
			}
		})
	})
}