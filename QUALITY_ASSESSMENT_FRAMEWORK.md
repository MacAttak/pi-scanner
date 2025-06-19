# PI Scanner Quality Assessment Framework

## Executive Summary

To properly evaluate our PI detection quality, we need a comprehensive testing framework that measures:
- **Precision**: How many detected PI are actual PI (minimize false positives)
- **Recall**: How many actual PI did we find (minimize false negatives)  
- **F1-Score**: Harmonic mean of precision and recall
- **NOT Accuracy**: Due to class imbalance (99.9% of code is not PI)

## Current Performance Baselines

### Gitleaks Performance (from research):
- **Precision**: 46% (2nd best among secret scanners)
- **Recall**: 86-88% (best among secret scanners)
- **F1-Score**: 60%

### Key Insight:
Gitleaks has **excellent recall** but **moderate precision** - meaning it finds most secrets but has false positives. This is where our Stage 2 validation becomes critical.

## Proposed Testing Framework

### 1. Create Benchmark Dataset

```go
// pkg/testing/benchmark/dataset.go
type BenchmarkDataset struct {
    TruePositives  []TestCase // Actual PI in code
    TrueNegatives  []TestCase // Code without PI
    EdgeCases      []TestCase // Ambiguous cases
    Synthetic      []TestCase // Generated test data
}

type TestCase struct {
    ID          string
    Code        string
    Language    string
    PIType      detection.PIType
    IsActualPI  bool
    Context     string // prod, test, comment, etc.
    Rationale   string // Why is this PI or not?
}
```

### 2. Australian PI Test Cases

```go
// pkg/testing/benchmark/australian_pi.go
func GenerateAustralianPITestCases() []TestCase {
    return []TestCase{
        // TRUE POSITIVES - Actual PI in production code
        {
            ID:         "au-tfn-001",
            Code:       `const DEFAULT_TFN = "123456782"`, // Valid TFN
            PIType:     detection.PITypeTFN,
            IsActualPI: true,
            Context:    "production",
            Rationale:  "Hardcoded valid TFN in production constant",
        },
        {
            ID:         "au-tfn-002",
            Code:       `user.TaxFileNumber = "876543217"`, // Valid TFN
            PIType:     detection.PITypeTFN,
            IsActualPI: true,
            Context:    "production",
            Rationale:  "Valid TFN assigned to user object",
        },
        
        // FALSE POSITIVES - Look like PI but aren't
        {
            ID:         "au-tfn-003",
            Code:       `// Example TFN: 123456782`,
            PIType:     detection.PITypeTFN,
            IsActualPI: false,
            Context:    "comment",
            Rationale:  "TFN in comment for documentation",
        },
        {
            ID:         "au-tfn-004",
            Code:       `func TestTFNValidation() { tfn := "123456782" }`,
            PIType:     detection.PITypeTFN,
            IsActualPI: false,
            Context:    "test",
            Rationale:  "Valid TFN but in test file",
        },
        
        // EDGE CASES
        {
            ID:         "au-medicare-001",
            Code:       `validateMedicare("2428778132")`, // Valid Medicare
            PIType:     detection.PITypeMedicare,
            IsActualPI: false, // It's being validated, not stored
            Context:    "validation",
            Rationale:  "Medicare number in validation function",
        },
        
        // CO-OCCURRENCE (Critical Risk)
        {
            ID:         "au-multi-001",
            Code:       `
                customer := Customer{
                    Name: "John Smith",
                    TFN: "123456782",
                    Address: "123 Queen St, Melbourne",
                }`,
            PIType:     detection.PITypeTFN,
            IsActualPI: true,
            Context:    "production",
            Rationale:  "Multiple PI types together = critical risk",
        },
    }
}
```

### 3. Evaluation Metrics Implementation

```go
// pkg/testing/evaluation/metrics.go
type EvaluationMetrics struct {
    TruePositives  int
    FalsePositives int
    TrueNegatives  int
    FalseNegatives int
}

func (em *EvaluationMetrics) Precision() float64 {
    if em.TruePositives + em.FalsePositives == 0 {
        return 0
    }
    return float64(em.TruePositives) / float64(em.TruePositives + em.FalsePositives)
}

func (em *EvaluationMetrics) Recall() float64 {
    if em.TruePositives + em.FalseNegatives == 0 {
        return 0
    }
    return float64(em.TruePositives) / float64(em.TruePositives + em.FalseNegatives)
}

func (em *EvaluationMetrics) F1Score() float64 {
    p := em.Precision()
    r := em.Recall()
    if p + r == 0 {
        return 0
    }
    return 2 * (p * r) / (p + r)
}

func (em *EvaluationMetrics) Report() string {
    return fmt.Sprintf(`
Performance Metrics:
- Precision: %.2f%% (of detected PI, how many were actual PI)
- Recall:    %.2f%% (of actual PI, how many did we detect)
- F1-Score:  %.2f%% (harmonic mean)

Confusion Matrix:
- True Positives:  %d (correctly identified PI)
- False Positives: %d (incorrectly flagged as PI)
- True Negatives:  %d (correctly identified as not PI)
- False Negatives: %d (missed actual PI)
    `, 
        em.Precision()*100, 
        em.Recall()*100, 
        em.F1Score()*100,
        em.TruePositives,
        em.FalsePositives,
        em.TrueNegatives,
        em.FalseNegatives,
    )
}
```

### 4. Comparative Testing Framework

```go
// pkg/testing/evaluation/comparator.go
type DetectorComparator struct {
    detectors map[string]detection.Detector
    dataset   *BenchmarkDataset
}

func (dc *DetectorComparator) Compare() map[string]*EvaluationMetrics {
    results := make(map[string]*EvaluationMetrics)
    
    for name, detector := range dc.detectors {
        metrics := &EvaluationMetrics{}
        
        for _, testCase := range dc.dataset.AllCases() {
            findings, _ := detector.Detect(
                context.Background(), 
                []byte(testCase.Code), 
                testCase.ID + ".go",
            )
            
            detected := len(findings) > 0
            
            switch {
            case detected && testCase.IsActualPI:
                metrics.TruePositives++
            case detected && !testCase.IsActualPI:
                metrics.FalsePositives++
            case !detected && !testCase.IsActualPI:
                metrics.TrueNegatives++
            case !detected && testCase.IsActualPI:
                metrics.FalseNegatives++
            }
        }
        
        results[name] = metrics
    }
    
    return results
}
```

### 5. Real-World Testing Approach

```go
// pkg/testing/realworld/scanner.go
type RealWorldTester struct {
    scanner      *Scanner
    groundTruth  map[string][]GroundTruthPI
}

type GroundTruthPI struct {
    File     string
    Line     int
    PIType   detection.PIType
    Value    string
    Verified bool // Manually verified as actual PI
}

// Test against manually annotated repositories
func (rwt *RealWorldTester) TestRepository(repoPath string) *EvaluationMetrics {
    // 1. Run scanner
    findings := rwt.scanner.ScanDirectory(repoPath)
    
    // 2. Compare against ground truth
    metrics := &EvaluationMetrics{}
    
    // Check each finding
    for _, finding := range findings {
        if rwt.isInGroundTruth(finding) {
            metrics.TruePositives++
        } else {
            metrics.FalsePositives++
        }
    }
    
    // Check for missed PI
    for _, truth := range rwt.groundTruth[repoPath] {
        if !rwt.wasDetected(truth, findings) {
            metrics.FalseNegatives++
        }
    }
    
    return metrics
}
```

## Testing Strategy

### Phase 1: Baseline Testing (Gitleaks Only)
```bash
# Test Gitleaks alone
go test -run TestGitleaksBaseline

Expected Results:
- Precision: ~45-50%
- Recall: ~85-90%
- F1-Score: ~60%
```

### Phase 2: With Context Validation
```bash
# Test Gitleaks + Context Validation
go test -run TestWithContextValidation

Target Results:
- Precision: >75% (reduce false positives)
- Recall: >80% (maintain high recall)
- F1-Score: >77%
```

### Phase 3: With Australian Validation
```bash
# Test full pipeline
go test -run TestFullPipeline

Target Results:
- Precision: >85% (algorithmic validation)
- Recall: >80% (maintain detection)
- F1-Score: >82%
```

## Key Testing Scenarios

### 1. Code Context Tests
```go
testCases := []struct {
    name     string
    code     string
    expected bool
}{
    {
        "production_hardcoded_tfn",
        `const USER_TFN = "123456782"`,
        true, // Should detect
    },
    {
        "test_file_tfn",
        `func TestTFN() { tfn := "123456782" }`,
        false, // Should suppress
    },
    {
        "comment_tfn",
        `// TFN format: 123456782`,
        false, // Should suppress
    },
    {
        "invalid_tfn",
        `tfn := "123456789"`, // Invalid checksum
        false, // Should reject in validation
    },
}
```

### 2. Co-occurrence Tests
```go
{
    "critical_risk_combination",
    `user := User{
        Name: "John Smith",
        TFN: "123456782",
        Medicare: "2428778132",
        Address: "123 Queen St",
    }`,
    true, // Critical risk
}
```

### 3. Language-Specific Tests
Test across Go, Java, Python, JavaScript, SQL to ensure patterns work.

## Synthetic Data Generation

```go
// pkg/testing/synthetic/generator.go
type SyntheticPIGenerator struct {
    faker *faker.Faker
}

func (spg *SyntheticPIGenerator) GenerateTestCase() TestCase {
    piType := spg.randomPIType()
    context := spg.randomContext()
    
    return TestCase{
        Code:       spg.generateCode(piType, context),
        PIType:     piType,
        IsActualPI: context == "production",
        Context:    context,
    }
}

func (spg *SyntheticPIGenerator) GenerateValidTFN() string {
    // Generate valid TFN with correct checksum
    base := spg.faker.RandomInt(10000000, 99999999)
    checksum := calculateTFNChecksum(base)
    return fmt.Sprintf("%d%d", base, checksum)
}
```

## Success Criteria

### Minimum Viable Performance
- **Precision**: >70% (reduce false positives from Gitleaks baseline)
- **Recall**: >80% (maintain high detection rate)
- **F1-Score**: >75%

### Target Performance
- **Precision**: >85%
- **Recall**: >85%
- **F1-Score**: >85%

### By PI Type
- **TFN/ABN/Medicare**: >95% precision (due to checksum validation)
- **Names/Addresses**: >70% precision (harder to validate)
- **Critical Combinations**: >90% precision

## Continuous Improvement

```go
// pkg/testing/feedback/collector.go
type FeedbackCollector struct {
    db *sql.DB
}

func (fc *FeedbackCollector) RecordFeedback(finding Finding, isCorrect bool, reason string) {
    // Store user feedback on findings
    // Use to improve patterns and weights
}

func (fc *FeedbackCollector) GenerateReport() FeedbackReport {
    // Analyze patterns in false positives/negatives
    // Suggest rule improvements
}
```

## Conclusion

This framework allows us to:
1. **Measure objectively** using standard metrics
2. **Compare approaches** (Gitleaks vs Gitleaks+Validation)
3. **Identify weaknesses** through detailed analysis
4. **Improve iteratively** based on feedback

The key insight is that **Gitleaks provides excellent recall** but needs our validation layers to improve precision, especially for code-specific contexts.