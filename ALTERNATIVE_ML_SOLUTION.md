# Alternative ML Solution for PI Scanner

## Executive Summary

Based on the PRD requirements and the challenges with tokenizers/ONNX Runtime, I propose a **pure Go solution** that maintains the multi-stage detection architecture while eliminating CGO dependencies.

## Key Insights from PRD

1. **ML is Stage 2 validation**, not primary detection - it's for "semantic/contextual validation, reducing false positives"
2. **Australian PI types** have strong algorithmic validation (TFN, ABN, Medicare)
3. **Context scoring** is as important as detection
4. **Code-aware detection** is crucial - Spacy failed because it's trained on natural language, not code

## Proposed Architecture

### Stage 1: Enhanced Pattern Detection (Gitleaks + Custom Regex)
Keep as-is - this is our primary detection mechanism

### Stage 2: Pure Go Context-Aware Validation (Replace ML)
Instead of DeBERTa/ONNX, implement:

```go
package validation

type ContextValidator struct {
    codePatterns    *CodePatternAnalyzer
    proximityEngine *ProximityAnalyzer
    syntaxAnalyzer  *SyntaxContextAnalyzer
}

// Analyze code context around potential PI
func (cv *ContextValidator) ValidateFinding(finding Finding) ValidationResult {
    // 1. Code pattern analysis
    codeContext := cv.codePatterns.AnalyzeContext(finding)
    
    // 2. Proximity to other PI
    proximityScore := cv.proximityEngine.ScoreProximity(finding)
    
    // 3. Syntax analysis (variable names, comments, strings)
    syntaxScore := cv.syntaxAnalyzer.AnalyzeSyntax(finding)
    
    return cv.computeConfidence(codeContext, proximityScore, syntaxScore)
}
```

### Stage 3: Australian PI Algorithmic Validation
Keep as-is - this is already pure Go

## Detailed Implementation Plan

### 1. Code-Aware Pattern Analysis

```go
type CodePatternAnalyzer struct {
    testPatterns     []string // "test", "mock", "example", "demo"
    variablePatterns []string // Common variable naming patterns
    commentPatterns  []string // Comment indicators
}

func (cpa *CodePatternAnalyzer) AnalyzeContext(finding Finding) CodeContext {
    return CodeContext{
        IsInTestFile:     cpa.isTestContext(finding.File),
        IsInComment:      cpa.isCommentContext(finding.Line),
        IsVariableName:   cpa.isVariableName(finding.Context),
        IsStringLiteral:  cpa.isStringLiteral(finding.Context),
        IsConfiguration:  cpa.isConfigContext(finding.File),
        SurroundingCode:  cpa.extractCodeWindow(finding),
    }
}
```

### 2. Proximity-Based Validation

```go
type ProximityAnalyzer struct {
    windowSize int
    piTypes    map[PIType]float64 // Weight for each PI type
}

func (pa *ProximityAnalyzer) ScoreProximity(finding Finding) float64 {
    // Find other PI within window
    nearbyFindings := pa.findNearby(finding, pa.windowSize)
    
    // Score based on:
    // - Distance to other PI
    // - Types of nearby PI (name + TFN = high score)
    // - Clustering patterns
    
    return pa.calculateProximityScore(finding, nearbyFindings)
}
```

### 3. Syntax-Aware Analysis

```go
type SyntaxContextAnalyzer struct {
    parser *SimpleCodeParser
}

func (sca *SyntaxContextAnalyzer) AnalyzeSyntax(finding Finding) SyntaxScore {
    // Parse surrounding code
    ast := sca.parser.ParseFragment(finding.Context)
    
    return SyntaxScore{
        IsAssignment:     ast.IsAssignment(),
        IsParameter:      ast.IsParameter(),
        IsHardcoded:      ast.IsHardcodedValue(),
        VariableContext:  ast.GetVariableContext(),
        StringContext:    ast.GetStringContext(),
    }
}
```

### 4. Simple Code Parser (Pure Go)

```go
type SimpleCodeParser struct {
    langPatterns map[string]*LanguagePatterns
}

type LanguagePatterns struct {
    AssignmentRegex  *regexp.Regexp
    ParameterRegex   *regexp.Regexp
    StringRegex      *regexp.Regexp
    CommentRegex     *regexp.Regexp
}

// Support common languages in banking repos
func NewSimpleCodeParser() *SimpleCodeParser {
    return &SimpleCodeParser{
        langPatterns: map[string]*LanguagePatterns{
            ".go":   goPatterns(),
            ".java": javaPatterns(),
            ".py":   pythonPatterns(),
            ".js":   jsPatterns(),
            ".sql":  sqlPatterns(),
        },
    }
}
```

### 5. Confidence Scoring Engine

```go
type ConfidenceEngine struct {
    weights ConfidenceWeights
}

type ConfidenceWeights struct {
    TestFileReduction      float64 // -0.8
    CommentReduction       float64 // -0.6
    VariableNameReduction  float64 // -0.4
    HardcodedBoost         float64 // +0.7
    ProximityBoost         float64 // +0.5
    ValidatedBoost         float64 // +0.9
}

func (ce *ConfidenceEngine) ComputeScore(
    codeContext CodeContext,
    proximityScore float64,
    syntaxScore SyntaxScore,
    validated bool,
) float64 {
    score := 0.5 // baseline
    
    // Apply reductions
    if codeContext.IsInTestFile {
        score += ce.weights.TestFileReduction
    }
    if codeContext.IsInComment {
        score += ce.weights.CommentReduction
    }
    
    // Apply boosts
    if syntaxScore.IsHardcoded {
        score += ce.weights.HardcodedBoost
    }
    if proximityScore > 0.7 {
        score += ce.weights.ProximityBoost
    }
    if validated {
        score += ce.weights.ValidatedBoost
    }
    
    return math.Max(0, math.Min(1, score))
}
```

## Implementation Benefits

### 1. No CGO Dependencies
- Pure Go implementation
- Easy cross-compilation
- No native library management
- Simplified CI/CD

### 2. Code-Aware Detection
- Understands code structure
- Reduces false positives in test files
- Identifies hardcoded vs dynamic values
- Language-specific patterns

### 3. Maintainable & Extensible
- Clear, readable code
- Easy to add new patterns
- Simple to tune weights
- No black-box ML models

### 4. Performance
- Lightweight computation
- Parallelizable
- No model loading overhead
- Predictable memory usage

## Migration Path

### Phase 1: Remove ML Dependencies (Day 1)
```bash
# Remove ML packages
rm -rf pkg/ml/inference
rm -rf pkg/ml/tokenization
rm -rf pkg/ml/models

# Update imports
find . -name "*.go" -exec grep -l "pkg/ml" {} \; | xargs sed -i 's/pkg\/ml/pkg\/validation/g'
```

### Phase 2: Implement Pure Go Validation (Day 2-3)
1. Create `pkg/validation/context_validator.go`
2. Create `pkg/validation/proximity_analyzer.go`
3. Create `pkg/validation/syntax_analyzer.go`
4. Create `pkg/validation/code_parser.go`

### Phase 3: Update Detection Pipeline (Day 4)
```go
// In detector.go
func (d *Detector) Detect(ctx context.Context, content []byte, filename string) ([]Finding, error) {
    // Stage 1: Pattern detection (Gitleaks + regex)
    findings := d.patternDetector.Detect(content, filename)
    
    // Stage 2: Context validation (replaces ML)
    validated := []Finding{}
    for _, f := range findings {
        result := d.contextValidator.ValidateFinding(f)
        if result.Confidence > d.config.MinConfidence {
            f.Confidence = result.Confidence
            f.ValidationDetails = result.Details
            validated = append(validated, f)
        }
    }
    
    // Stage 3: Algorithmic validation
    for i, f := range validated {
        if validator, ok := d.validators[f.Type]; ok {
            isValid, err := validator.Validate(f.Match)
            validated[i].Validated = isValid
            validated[i].ValidationError = err
        }
    }
    
    return validated, nil
}
```

## Testing Strategy

### 1. Create Code-Specific Test Cases
```go
func TestCodeContextDetection(t *testing.T) {
    tests := []struct {
        name     string
        code     string
        expected bool
    }{
        {
            name: "hardcoded TFN in production code",
            code: `const userTFN = "123456782"`, // valid TFN
            expected: true,
        },
        {
            name: "TFN in test file",
            code: `func TestTFN() { tfn := "123456782" }`,
            expected: false, // reduced confidence
        },
        {
            name: "TFN in comment",
            code: `// Example TFN: 123456782`,
            expected: false,
        },
    }
}
```

### 2. Proximity Detection Tests
```go
func TestProximityDetection(t *testing.T) {
    tests := []struct {
        name     string
        code     string
        expected RiskLevel
    }{
        {
            name: "Name + TFN + Address",
            code: `
                user := User{
                    Name: "John Smith",
                    TFN: "123456782",
                    Address: "123 Queen St, Melbourne VIC 3000"
                }
            `,
            expected: RiskLevelCritical,
        },
    }
}
```

## Advantages Over ML Approach

1. **Predictable**: No black-box behavior
2. **Tunable**: Easy to adjust for specific patterns
3. **Fast**: No model loading or inference
4. **Portable**: Works everywhere Go works
5. **Maintainable**: Team can understand and modify

## Disadvantages

1. **Less sophisticated** than transformer models
2. **Requires manual tuning** for new patterns
3. **May need updates** as code patterns evolve

## Conclusion

This pure Go solution maintains the three-stage architecture from the PRD while eliminating problematic dependencies. It's specifically designed for detecting PI in code (not natural language) and provides transparent, tunable detection that can be understood and maintained by the team.

The key insight is that **code has structure** that we can analyze programmatically without needing complex NLP models trained on natural language text.