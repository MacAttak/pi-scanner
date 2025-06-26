# LLM Integration for Enhanced PI Detection

## Overview

The PI Scanner now supports LLM-based validation to reduce false positives in PI detection. This feature uses a local LLM (via LM Studio) to analyze the context around detected patterns and provide more accurate risk assessments.

## How It Works

1. **Base Detection**: The scanner first runs traditional pattern matching to find potential PI
2. **Context Extraction**: For medium and high-risk findings, the scanner extracts surrounding code context
3. **LLM Validation**: The context is sent to a local LLM for analysis
4. **Risk Adjustment**: Based on the LLM's assessment, findings may be downgraded or confirmed

## Setup

### 1. Install LM Studio

Download and install [LM Studio](https://lmstudio.ai/) for your platform.

### 2. Download a Model

Recommended models for code analysis:
- **Qwen2.5-Coder-7B-Instruct** (recommended) - Best balance of speed and accuracy
- **DeepSeek-Coder-6.7B** - Good alternative
- **Llama-3.2-3B-Instruct** - Lighter option for faster processing

### 3. Start LM Studio Server

1. Load your chosen model in LM Studio
2. Start the local server (default: http://localhost:1234)
3. Verify it's running by visiting http://localhost:1234/v1/models

## Usage

### Command Line

```bash
# Basic usage with LLM validation
pi-scanner scan --repo https://github.com/user/repo --enable-llm

# Specify a different model
pi-scanner scan --repo https://github.com/user/repo \
  --enable-llm \
  --llm-model "deepseek-coder-6.7b"

# Use a different endpoint
pi-scanner scan --repo https://github.com/user/repo \
  --enable-llm \
  --llm-endpoint "http://192.168.1.100:1234/v1"
```

### Configuration File

Create a `config.yaml`:

```yaml
enable_llm_validation: true
llm_provider: "lmstudio"
llm_endpoint: "http://localhost:1234/v1"
llm_model: "qwen2.5-coder-7b-instruct"
llm_max_tokens: 1000
llm_temperature: 0.3
llm_validate_risks:
  - HIGH
  - MEDIUM
```

Then use:
```bash
pi-scanner scan --repo https://github.com/user/repo --config config.yaml
```

## Example Output

Without LLM:
```json
{
  "type": "TFN",
  "match": "123456782",
  "risk_level": "HIGH",
  "confidence": 0.8,
  "context": "tfn = '123456782' # Test TFN"
}
```

With LLM:
```json
{
  "type": "TFN",
  "match": "123456782",
  "risk_level": "HIGH",
  "confidence": 0.8,
  "context": "tfn = '123456782' # Test TFN",
  "llm_validated": true,
  "llm_risk": "LOW",
  "llm_explanation": "This appears to be test data based on the comment '# Test TFN' and the variable name suggesting it's for testing purposes.",
  "llm_confidence": 0.9
}
```

## Performance Considerations

- LLM validation adds 0.5-2 seconds per finding
- Only medium and high-risk findings are validated by default
- Validation runs concurrently (max 3 requests by default)
- Test files are skipped to improve performance

## Troubleshooting

### LM Studio Connection Issues

```bash
# Check if LM Studio is running
curl http://localhost:1234/v1/models

# Test with a simple completion
curl http://localhost:1234/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "qwen2.5-coder-7b-instruct",
    "messages": [{"role": "user", "content": "Hello"}]
  }'
```

### Slow Performance

1. Use a smaller model (e.g., Llama-3.2-3B)
2. Reduce context lines in configuration
3. Limit validation to HIGH risk only
4. Ensure LM Studio has GPU acceleration enabled

## Future Enhancements

- Support for cloud LLM providers (OpenAI, Anthropic)
- Batch processing for better performance
- Custom prompts for specific PI types
- Fine-tuned models for Australian PI detection
