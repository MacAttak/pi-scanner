package detection

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
)

// LLMEnhancedDetector wraps a regular detector with LLM validation
type LLMEnhancedDetector struct {
	baseDetector Detector
	llmValidator LLMValidator
	config       *LLMEnhancedConfig
}

// NewLLMEnhancedDetector creates a new LLM-enhanced detector
func NewLLMEnhancedDetector(baseDetector Detector, validator LLMValidator, config *LLMEnhancedConfig) *LLMEnhancedDetector {
	if config == nil {
		config = &LLMEnhancedConfig{
			Enabled:            true,
			ValidateRiskLevels: []RiskLevel{RiskLevelHigh, RiskLevelMedium},
			MaxConcurrency:     3,
			SkipTestFiles:      true,
			ContextLinesBefore: 50,
			ContextLinesAfter:  50,
		}
	}

	return &LLMEnhancedDetector{
		baseDetector: baseDetector,
		llmValidator: validator,
		config:       config,
	}
}

// Detect runs the base detector and enhances findings with LLM validation
func (d *LLMEnhancedDetector) Detect(ctx context.Context, content []byte, filename string) ([]Finding, error) {
	// Run base detection
	findings, err := d.baseDetector.Detect(ctx, content, filename)
	if err != nil {
		return nil, fmt.Errorf("base detection failed: %w", err)
	}

	// Skip LLM validation if disabled
	if !d.config.Enabled || d.llmValidator == nil {
		return findings, nil
	}

	// Skip test files if configured
	if d.config.SkipTestFiles && isTestFile(filename) {
		return findings, nil
	}

	// Enhance findings with LLM validation
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, d.config.MaxConcurrency)

	for i := range findings {
		if !d.shouldValidate(findings[i]) {
			continue
		}

		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			d.validateFinding(ctx, &findings[idx], string(content), filename)
		}(i)
	}

	wg.Wait()
	return findings, nil
}

// Name returns the detector name
func (d *LLMEnhancedDetector) Name() string {
	return fmt.Sprintf("%s-llm-enhanced", d.baseDetector.Name())
}

// validateFinding validates a single finding with LLM
func (d *LLMEnhancedDetector) validateFinding(ctx context.Context, finding *Finding, content, filename string) {
	// Extract context around the finding
	context := ExtractContext(content, *finding, d.config.ContextLinesBefore, d.config.ContextLinesAfter)

	// Create validation request
	req := LLMValidationRequest{
		Finding:    *finding,
		Context:    context,
		FilePath:   filename,
		FileType:   getFileType(filename),
		IsTestFile: isTestFile(filename),
	}

	// Perform validation
	result, err := d.llmValidator.ValidateFinding(ctx, req)
	if err != nil {
		// Log error but don't fail
		return
	}

	// Update finding with LLM results
	finding.LLMValidated = true
	finding.LLMRisk = result.Risk
	finding.LLMExplanation = result.Explanation
	finding.LLMConfidence = result.Confidence
}

// shouldValidate determines if a finding should be validated by LLM
func (d *LLMEnhancedDetector) shouldValidate(finding Finding) bool {
	for _, risk := range d.config.ValidateRiskLevels {
		if finding.RiskLevel == risk {
			return true
		}
	}
	return false
}

// ExtractContext extracts lines of context around a finding
func ExtractContext(content string, finding Finding, linesBefore, linesAfter int) string {
	lines := strings.Split(content, "\n")

	startLine := finding.Line - linesBefore - 1
	if startLine < 0 {
		startLine = 0
	}

	endLine := finding.Line + linesAfter
	if endLine > len(lines) {
		endLine = len(lines)
	}

	contextLines := lines[startLine:endLine]

	// Add line numbers for clarity
	var result []string
	for i, line := range contextLines {
		lineNum := startLine + i + 1
		prefix := "  "
		if lineNum == finding.Line {
			prefix = "> "
		}
		result = append(result, fmt.Sprintf("%s%4d: %s", prefix, lineNum, line))
	}

	return strings.Join(result, "\n")
}

// isTestFile checks if a file is a test file
func isTestFile(filename string) bool {
	lowerName := strings.ToLower(filename)
	baseName := strings.ToLower(filepath.Base(filename))

	testPatterns := []string{
		"_test.go", "_test.py", "test_", ".test.", ".spec.",
		"/test/", "/tests/", "/spec/", "/specs/",
		"_spec.rb", "test.java", "tests.java",
		"spec.scala", "test.scala",
	}

	for _, pattern := range testPatterns {
		if strings.Contains(lowerName, strings.ToLower(pattern)) {
			return true
		}
	}

	// Check if the base filename starts with "Test" (common Java pattern)
	if strings.HasPrefix(baseName, "test") {
		ext := filepath.Ext(baseName)
		// Only consider it a test file if it has a code extension
		codeExts := []string{".java", ".scala", ".kt", ".groovy", ".js", ".ts", ".py", ".go", ".rb"}
		for _, codeExt := range codeExts {
			if ext == codeExt {
				return true
			}
		}
	}

	return false
}

// getFileType returns the file type based on extension
func getFileType(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".go":
		return "go"
	case ".py":
		return "python"
	case ".js", ".jsx", ".ts", ".tsx":
		return "javascript"
	case ".java":
		return "java"
	case ".scala":
		return "scala"
	case ".rb":
		return "ruby"
	case ".rs":
		return "rust"
	case ".c", ".cpp", ".cc", ".cxx", ".h", ".hpp":
		return "cpp"
	case ".cs":
		return "csharp"
	case ".php":
		return "php"
	case ".kt", ".kts":
		return "kotlin"
	case ".swift":
		return "swift"
	case ".m", ".mm":
		return "objc"
	case ".r":
		return "r"
	case ".sh", ".bash", ".zsh":
		return "shell"
	case ".yml", ".yaml":
		return "yaml"
	case ".json":
		return "json"
	case ".xml":
		return "xml"
	case ".sql":
		return "sql"
	default:
		return "unknown"
	}
}
