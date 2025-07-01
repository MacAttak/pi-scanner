package detection

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// LLMEnhancedDetector wraps a regular detector with LLM validation
type LLMEnhancedDetector struct {
	baseDetector Detector
	llmValidator LLMValidator
	config       *LLMEnhancedConfig
	progress     *ValidationProgress
}

// ValidationProgress tracks LLM validation progress
type ValidationProgress struct {
	mu        sync.Mutex
	total     int
	processed int
	skipped   int
	startTime time.Time
	callback  func(processed, total int, rate float64)
}

// NewLLMEnhancedDetector creates a new LLM-enhanced detector
func NewLLMEnhancedDetector(baseDetector Detector, validator LLMValidator, config *LLMEnhancedConfig) *LLMEnhancedDetector {
	if config == nil {
		config = &LLMEnhancedConfig{
			Enabled:            true,
			ValidateRiskLevels: []RiskLevel{RiskLevelCritical, RiskLevelHigh, RiskLevelMedium, RiskLevelLow},
			MaxConcurrency:     20,    // Increased for better throughput
			SkipTestFiles:      false, // Never skip test files - critical for DLP compliance
			ContextLinesBefore: 50,
			ContextLinesAfter:  50,
		}
	}

	return &LLMEnhancedDetector{
		baseDetector: baseDetector,
		llmValidator: validator,
		config:       config,
		progress: &ValidationProgress{
			startTime: time.Now(),
		},
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

	// First, apply smart filtering
	toValidate := make([]int, 0)
	for i := range findings {
		if d.shouldValidate(findings[i], filename) {
			toValidate = append(toValidate, i)
		}
	}

	// Set up progress tracking
	d.progress.mu.Lock()
	d.progress.total = len(toValidate)
	d.progress.processed = 0
	d.progress.skipped = len(findings) - len(toValidate)
	d.progress.mu.Unlock()

	// Enhance findings with LLM validation
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, d.config.MaxConcurrency)

	for _, idx := range toValidate {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			// Get dynamic context size based on risk
			linesBefore, linesAfter := d.getContextSize(findings[i].RiskLevel)
			d.validateFinding(ctx, &findings[i], string(content), filename, linesBefore, linesAfter)

			// Update progress
			d.updateProgress()
		}(idx)
	}

	wg.Wait()
	return findings, nil
}

// Name returns the detector name
func (d *LLMEnhancedDetector) Name() string {
	return fmt.Sprintf("%s-llm-enhanced", d.baseDetector.Name())
}

// validateFinding validates a single finding with LLM
func (d *LLMEnhancedDetector) validateFinding(ctx context.Context, finding *Finding, content, filename string, linesBefore, linesAfter int) {
	// Extract context around the finding
	context := ExtractContext(content, *finding, linesBefore, linesAfter)

	// Create validation request
	req := LLMValidationRequest{
		Finding:    *finding,
		Context:    context,
		FilePath:   filename,
		FileType:   getFileType(filename),
		IsTestFile: isTestFile(filename),
		ASTContext: finding.ASTContext, // Pass AST context from finding
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
func (d *LLMEnhancedDetector) shouldValidate(finding Finding, filename string) bool {
	// Skip obvious false positives
	if d.isObviousFalsePositive(finding, filename) {
		return false
	}

	// Check risk level
	for _, risk := range d.config.ValidateRiskLevels {
		if finding.RiskLevel == risk {
			return true
		}
	}
	return false
}

// isObviousFalsePositive checks for patterns that are clearly not real PI
func (d *LLMEnhancedDetector) isObviousFalsePositive(finding Finding, filename string) bool {
	// Only skip very obvious false positives based on content, not file type
	// Let LLM decide for most cases

	// Skip common example values that are definitely not real PI
	examplePatterns := []string{
		"john doe", "jane doe", "john smith", "jane smith",
		"example@email.com", "test@test.com", "user@example.com",
		"foo@bar.com", "admin@example.com",
	}

	matchLower := strings.ToLower(finding.Match)
	for _, pattern := range examplePatterns {
		if matchLower == pattern || strings.Contains(matchLower, pattern) {
			return true
		}
	}

	// Skip sequential test patterns
	if finding.Type == PITypeTFN || finding.Type == PITypeDriverLicense {
		if finding.Match == "123456789" || finding.Match == "000000000" ||
			finding.Match == "111111111" || finding.Match == "999999999" {
			return true
		}
	}

	return false
}

// getContextSize returns dynamic context size based on risk level
func (d *LLMEnhancedDetector) getContextSize(risk RiskLevel) (before, after int) {
	switch risk {
	case RiskLevelCritical, RiskLevelHigh:
		return 50, 50
	case RiskLevelMedium:
		return 30, 30
	default: // LOW
		return 20, 20
	}
}

// updateProgress updates validation progress
func (d *LLMEnhancedDetector) updateProgress() {
	d.progress.mu.Lock()
	defer d.progress.mu.Unlock()

	d.progress.processed++

	if d.progress.callback != nil {
		elapsed := time.Since(d.progress.startTime).Minutes()
		rate := float64(d.progress.processed) / elapsed
		d.progress.callback(d.progress.processed, d.progress.total, rate)
	}
}

// SetProgressCallback sets the progress callback function
func (d *LLMEnhancedDetector) SetProgressCallback(callback func(processed, total int, rate float64)) {
	d.progress.mu.Lock()
	defer d.progress.mu.Unlock()
	d.progress.callback = callback
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
