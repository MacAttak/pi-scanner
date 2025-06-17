# PII Detection Technologies Research

## Executive Summary

This document provides detailed research findings on technologies for PI/PII detection with a focus on:
1. Gitleaks capabilities and configuration
2. ML/NLP models (particularly DeBERTa) for PII detection
3. Regex pattern libraries for financial and Australian-specific PII
4. Integration approaches for layered detection

All technologies are evaluated for local deployment, accuracy, and suitability for banking/financial data.

## 1. Gitleaks

### Overview
Gitleaks is an open-source SAST (Static Application Security Testing) tool designed to detect and prevent secrets from being committed to git repositories. It uses regular expressions and entropy-based detection.

### Key Capabilities
- **Configuration Format**: Uses TOML format for custom rules
- **Detection Methods**: Regex patterns combined with Shannon entropy calculations
- **Performance**: Designed for high-speed scanning of large repositories
- **Extensibility**: Fully customizable rule sets

### Australian-Specific Configuration

```toml
# Australian Tax File Number (TFN)
[[rules]]
id = "australian_tfn"
title = "Australian Tax File Number"
description = "Detects Australian Tax File Numbers"
regex = '''\b\d{3}[\s-]?\d{3}[\s-]?\d{3}\b'''
keywords = ["tfn", "tax file", "taxfile"]
entropy = 3.5

# Australian Business Number (ABN)
[[rules]]
id = "australian_abn"
title = "Australian Business Number"
description = "Detects Australian Business Numbers"
regex = '''\b\d{2}[\s-]?\d{3}[\s-]?\d{3}[\s-]?\d{3}\b'''
keywords = ["abn", "business number"]

# Australian Medicare Number
[[rules]]
id = "australian_medicare"
title = "Australian Medicare Number"
description = "Detects Australian Medicare Numbers"
regex = '''\b[2-6]\d{3}[\s-]?\d{5}[\s-]?\d{1}\b'''
keywords = ["medicare", "health card"]

# Australian Driver's License (varies by state)
[[rules]]
id = "australian_drivers_license"
title = "Australian Driver's License"
description = "Detects various Australian state driver's license formats"
regex = '''(?i)\b(?:NSW|VIC|QLD|SA|WA|TAS|NT|ACT)[\s-]?(?:\d{6,10}|[A-Z]\d{5,8})\b'''
keywords = ["license", "licence", "driver", "drivers"]
```

### Performance Characteristics
- **Parallel Processing**: Supports concurrent file scanning
- **Memory Efficiency**: Designed for large codebases
- **Speed**: Can scan thousands of files per second
- **Incremental Scanning**: Can focus on changed files only

### Best Practices
1. Start with a minimal rule set and expand gradually
2. Use entropy thresholds to reduce false positives
3. Implement allowlists for known test data
4. Utilize path-based exclusions for test directories
5. Regular expression testing using regex101.com (Golang mode)

## 2. ML/NLP Models for PII Detection

### DeBERTa-v3 for PII Detection

#### Overview
DeBERTa (Decoding-enhanced BERT with Disentangled Attention) v3 is the leading model for PII detection tasks.

#### Performance Metrics
- **Recall Rate**: 98% (critical for PII detection)
- **Model Size**: Various sizes available:
  - DeBERTa-v3-xsmall: 22M parameters (efficient for local deployment)
  - DeBERTa-v3-base: ~184M parameters
  - DeBERTa-v3-large: ~434M parameters
- **Inference Speed**: Varies by model size and hardware

#### Key Advantages
1. **High Recall**: 98% recall ensures most PII is detected
2. **Efficiency**: XSmall variant provides excellent performance with minimal resources
3. **Context Understanding**: Superior to regex for context-dependent PII
4. **Pre-trained Models**: Available on Hugging Face for immediate use

#### Implementation Example
```python
from transformers import pipeline

# Initialize the PII detection pipeline
pii_detector = pipeline(
    "token-classification", 
    "lakshyakh93/deberta_finetuned_pii", 
    device=-1  # CPU deployment
)

# Detect PII
text = "My name is John and I live in California."
results = pii_detector(text, aggregation_strategy="first")
```

#### PII Categories Detected
- Account information (names, numbers, transactions)
- Banking details (BIC, IBAN, crypto addresses)
- Personal information (names, DOB, gender)
- Contact information (email, phone, addresses)
- Financial data (credit cards, CVV, currency)
- Digital identifiers (IP addresses)

### BERT vs RoBERTa Comparison

| Aspect | BERT | RoBERTa | Recommendation |
|--------|------|---------|----------------|
| Accuracy | Good (87-89%) | Better (89-92%) | RoBERTa for financial data |
| Resource Usage | Lower | Higher | BERT for constrained environments |
| Vocabulary | 30K tokens | 50K tokens | RoBERTa for financial terminology |
| Training Data | 16GB | 160GB | RoBERTa more robust |
| False Positives | Higher | Lower | RoBERTa for production |

### Local Deployment Considerations

1. **Hardware Requirements**:
   - Minimum: 4GB RAM for small models
   - Recommended: 8-16GB RAM, GPU optional
   - CPU inference is viable for batch processing

2. **Optimization Strategies**:
   - Use quantization for smaller model sizes
   - Implement batching for throughput
   - Consider model distillation for speed

3. **Financial Data Specific**:
   - Fine-tune on banking-specific datasets
   - Include financial terminology in training
   - Validate on Australian financial formats

## 3. Regex Pattern Libraries

### Australian PII Patterns

#### Tax File Number (TFN)
```regex
# Standard TFN patterns
\b\d{3}[\s-]?\d{3}[\s-]?\d{3}\b
\b\d{2}[\s-]?\d{3}[\s-]?\d{3}\b  # 8-digit variant
```

#### Australian Business Number (ABN)
```regex
\b\d{2}[\s-]?\d{3}[\s-]?\d{3}[\s-]?\d{3}\b
```

#### Medicare Number
```regex
# First digit must be 2-6
\b[2-6]\d{3}[\s-]?\d{5}[\s-]?\d{1-2}?\b
```

### Financial Pattern Best Practices

#### Credit Card Numbers
```regex
\b((4\d{3}|5[1-5]\d{2}|2\d{3}|3[47]\d{1,2})[\s\-]?\d{4,6}[\s\-]?\d{4,6}?([\s\-]\d{3,4})?(\d{3})?)\b
```

#### Bank Account Numbers
```regex
# US (9-17 digits)
\b\d{9,17}\b

# IBAN (up to 34 chars)
\b[A-Z]{2}\d{2}[A-Z0-9]{1,30}\b
```

### Performance Optimization

1. **Use Non-Capturing Groups**: `(?:...)` instead of `(...)`
2. **Anchor Patterns**: Use `\b` word boundaries
3. **Avoid Backtracking**: Use possessive quantifiers `*+`, `++`
4. **Character Classes**: `[0-9]` faster than `\d` in some engines
5. **Compile Once**: Pre-compile regex patterns

### OCR-Defensive Patterns
For scanned documents, replace `\d` with `[\dOIlZEASB]` to handle OCR errors:
```regex
# Standard
\b\d{3}-\d{3}-\d{3}\b

# OCR-Defensive
\b[\dOIlZEASB]{3}-[\dOIlZEASB]{3}-[\dOIlZEASB]{3}\b
```

## 4. Integration Approaches

### Layered Detection Architecture

#### Stage 1: Broad Detection (Gitleaks + Regex)
- **Purpose**: Cast wide net for potential PII
- **Strengths**: Fast, pattern-based, high recall
- **Weaknesses**: Higher false positives

#### Stage 2: ML/NLP Validation (DeBERTa)
- **Purpose**: Context-aware validation
- **Strengths**: Reduces false positives, understands context
- **Weaknesses**: Slower, requires more resources

#### Stage 3: Algorithmic Verification
- **Purpose**: Validate checksums, formats
- **Strengths**: High precision for structured data
- **Weaknesses**: Only works for specific formats

### Implementation Strategy

```python
class LayeredPIIDetector:
    def __init__(self):
        self.regex_detector = RegexDetector()
        self.ml_detector = DeBertaDetector()
        self.validator = AlgorithmicValidator()
    
    def detect(self, text):
        # Stage 1: Regex detection
        candidates = self.regex_detector.find_candidates(text)
        
        # Stage 2: ML validation
        ml_validated = []
        for candidate in candidates:
            if self.ml_detector.validate(candidate):
                ml_validated.append(candidate)
        
        # Stage 3: Algorithmic verification
        verified = []
        for item in ml_validated:
            if self.validator.verify(item):
                verified.append(item)
        
        return self.score_and_rank(verified)
```

### False Positive Reduction Strategies

1. **Scoring System**:
   ```toml
   [[rules]]
   name = "weak_pattern"
   regex = "\\b\\d{5}\\b"
   score = 0.1  # Low confidence
   
   [[rules]]
   name = "strong_pattern"
   regex = "\\bTFN:\\s*\\d{3}-\\d{3}-\\d{3}\\b"
   score = 0.9  # High confidence
   ```

2. **Context Requirements**:
   - Require keywords near patterns
   - Check surrounding text for indicators
   - Validate against known formats

3. **Suppression Rules**:
   - Path-based (test directories)
   - Content-based (synthetic data markers)
   - Entropy thresholds

### Performance Considerations

1. **Parallel Processing**:
   - Process files concurrently
   - Batch ML inference
   - Cache validation results

2. **Sampling Strategies**:
   - Full scan for critical paths
   - Sample large files
   - Skip binary files

3. **Resource Management**:
   - Load models once
   - Reuse compiled regex
   - Implement circuit breakers

## Recommendations

### For Commonwealth Bank Implementation

1. **Start with Gitleaks** configured with Australian-specific rules
2. **Add DeBERTa-v3-base** for context validation (good balance of performance/accuracy)
3. **Implement custom validators** for TFN, ABN, Medicare checksums
4. **Use layered approach** with scoring to reduce false positives
5. **Deploy locally** with following specs:
   - 16GB RAM minimum
   - Multi-core CPU (8+ cores recommended)
   - SSD storage for code repositories
   - GPU optional but beneficial for ML inference

### Quick Start Configuration

1. **Minimal Viable Scanner**:
   - Gitleaks with Australian rules
   - Basic regex patterns
   - Simple scoring system

2. **Production Scanner**:
   - Full layered architecture
   - DeBERTa-v3 for validation
   - Comprehensive Australian patterns
   - Checksum validators
   - Context scoring
   - Path-based suppression

3. **Performance Targets**:
   - <5% false positive rate
   - >95% true positive rate
   - Process 1GB repository in <5 minutes
   - Support incremental scanning

This research provides the foundation for implementing a robust, accurate, and performant PII detection system tailored for Australian banking requirements.