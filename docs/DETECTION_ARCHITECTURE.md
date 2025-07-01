# PI Detection Architecture

## Overview

The GitHub PI Scanner implements a sophisticated two-phase detection system for identifying personally identifiable information (PI) in source code. This document explains the architecture and how different components work together.

## Two-Phase Detection Architecture

### Phase 1: Pattern Detection
The first phase casts a wide net using pattern matching to identify potential PI. This phase prioritizes high recall (catching all potential PI) over precision.

### Phase 2: Context Validation  
The second phase uses contextual analysis and optional LLM validation to reduce false positives while maintaining detection accuracy.

## Detection Components

### 1. Base Detector (`detection.NewDetector()`)
The foundational detector that implements pattern matching for Australian PI types:
- **Pattern Matching**: Uses regex patterns to identify potential PI
- **Validation**: Applies checksum algorithms (e.g., modulo 11 for TFN, modulo 89 for ABN)
- **Risk Assessment**: Adjusts risk levels based on file context (test files get lower risk)
- **No Suppression**: Does NOT suppress findings - only adjusts risk levels

Key behaviors:
- Test files: Risk level reduced to LOW
- Production files: Risk level increased to HIGH/CRITICAL
- Configuration files: Risk level set to HIGH
- Multiple PI in proximity: Risk level increased

### 2. Context-Aware Wrappers

#### MockContextDetector (`testutil.NewMockContextDetector()`)
A wrapper that adds context-based suppression:
- **Suppresses findings in**:
  - Test files (`*_test.go`, `test_*.py`, etc.)
  - Mock files (`mock_*.go`, `*_mock.py`)
  - Comments (lines containing `//`)
  - Files with mock indicators in surrounding context
- **Used for**: Testing and demonstrating context suppression capabilities

#### LLMEnhancedDetector (`detection.NewLLMEnhancedDetector()`)
The most sophisticated detector that uses AI for context analysis:
- **LLM Validation**: Sends findings to an LLM for contextual validation
- **AST Context**: Includes code structure information in validation
- **Smart Suppression**: LLM determines if PI is real or test/example data
- **Configurable**: Can be disabled or configured with different LLM endpoints

### 3. Gitleaks Integration
An alternative detector that uses Gitleaks rules:
- **Rule-Based**: Uses TOML configuration for pattern definitions
- **Custom Rules**: Includes Australian PI patterns
- **Context Modifiers**: Applies similar risk adjustments as base detector

## Context Analysis Layers

### 1. File Path Context
Analyzed by checking file paths for patterns:
- Test directories: `/test/`, `/tests/`, `/spec/`
- Test files: `*_test.go`, `*.test.js`, `test_*.py`
- Mock files: `mock_*.go`, `*_mock.py`
- Documentation: `*.md`, `/docs/`

### 2. Code Context
Analyzed by examining surrounding code:
- Variable names: `testTFN`, `mockData`, `exampleABN`
- Comments: PI in comments gets lower risk
- Test patterns: `assert`, `expect`, test function names

### 3. AST Context (When Available)
Structural analysis provides:
- Risk zones: `customer_data`, `payment_processing`, `authentication`
- Code constructs: Classes, methods, imports
- File type: Test vs. production code

## Risk Level Definitions

- **CRITICAL**: PI in production code that could be exposed (API responses, logs)
- **HIGH**: PI in production code  
- **MEDIUM**: PI in configuration or with some mitigating context
- **LOW**: PI in test files, examples, or documentation

## Usage Examples

### Basic Detection (No Suppression)
```go
detector := detection.NewDetector()
findings, _ := detector.Detect(ctx, content, filename)
// Returns all findings with adjusted risk levels
```

### With Context Suppression
```go
baseDetector := detection.NewDetector()
contextDetector := testutil.NewMockContextDetector(baseDetector)
findings, _ := contextDetector.Detect(ctx, content, filename)
// Test file findings are suppressed
```

### With LLM Validation
```go
config := detection.LLMConfig{Enabled: true}
detector := detection.NewLLMEnhancedDetector(baseDetector, llmClient, config)
findings, _ := detector.Detect(ctx, content, filename)
// LLM validates each finding's context
```

## Best Practices

1. **Choose the Right Detector**:
   - Use base detector for maximum sensitivity
   - Use MockContextDetector for testing
   - Use LLMEnhancedDetector for production with LLM available

2. **Interpret Risk Levels**:
   - Don't ignore LOW risk findings - review them
   - Focus remediation on HIGH/CRITICAL findings
   - Consider context when evaluating MEDIUM risk

3. **Testing**:
   - Test with various file types and contexts
   - Verify both detection and suppression behavior
   - Use realistic test data that matches production patterns
