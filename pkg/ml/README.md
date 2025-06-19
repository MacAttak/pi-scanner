# ML Components for PI Scanner

This package provides machine learning capabilities for the PI Scanner, including:

- Model downloading from HuggingFace
- DeBERTa model integration for PI validation
- Tokenization support
- ONNX Runtime integration
- Concurrent model inference with worker pools

## Architecture

```
ml/
├── models/           # Model management and DeBERTa implementation
│   ├── downloader.go # Downloads models from HuggingFace
│   └── deberta.go    # DeBERTa model wrapper
├── inference/        # ONNX Runtime integration
│   ├── onnx_runtime.go # ONNX Runtime wrapper
│   └── model_runner.go # Concurrent inference runner
└── tokenization/     # Text tokenization
    └── tokenizer.go  # HuggingFace tokenizer wrapper
```

## Usage Example

```go
package main

import (
    "context"
    "log"
    
    "github.com/pi-scanner/pi-scanner/pkg/ml/models"
    "github.com/pi-scanner/pi-scanner/pkg/ml/inference"
)

func main() {
    // Download models
    downloader, err := models.NewModelDownloader("/tmp/models")
    if err != nil {
        log.Fatal(err)
    }
    
    modelPath, tokenizerPath, err := downloader.GetDeBERTaModel(context.Background())
    if err != nil {
        log.Fatal(err)
    }
    
    // Create model runner
    config := inference.ModelRunnerConfig{
        ModelPath:     modelPath,
        TokenizerPath: tokenizerPath,
        MaxWorkers:    4,
        QueueSize:     100,
    }
    
    runner, err := inference.NewModelRunner(config)
    if err != nil {
        log.Fatal(err)
    }
    
    // Start the runner
    if err := runner.Start(); err != nil {
        log.Fatal(err)
    }
    defer runner.Stop()
    
    // Validate PI
    request := inference.ValidationRequest{
        Candidate: models.PICandidate{
            Text:    "123-45-6789",
            Type:    "SSN",
            Context: "Customer SSN is 123-45-6789",
        },
    }
    
    result, err := runner.ValidatePI(context.Background(), request)
    if err != nil {
        log.Fatal(err)
    }
    
    log.Printf("Is valid PI: %v (confidence: %.2f%%)", 
        result.IsValid, result.Confidence*100)
}
```

## Testing

Run tests with the tokenizer library path set:

```bash
export CGO_LDFLAGS="-L$(pwd)/lib"
go test ./pkg/ml/...
```

## Model Integration Status

- [x] Model downloader utility
- [x] DeBERTa model wrapper structure
- [x] Tokenizer integration
- [x] Basic ONNX Runtime structure
- [x] Concurrent model runner
- [ ] Actual ONNX model loading (requires ONNX runtime library)
- [ ] Real inference implementation
- [ ] Model fine-tuning capabilities
- [ ] Performance optimizations

## Dependencies

- HuggingFace Tokenizers (C library)
- ONNX Runtime (not yet integrated)
- DeBERTa-v3-base model from HuggingFace

## Notes

The current implementation provides the structure and interfaces for ML integration.
The actual ONNX runtime integration requires the ONNX runtime shared library to be
installed on the system. Model inference is currently mocked for testing purposes.