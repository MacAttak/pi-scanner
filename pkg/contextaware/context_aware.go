package contextaware

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/MacAttak/pi-scanner/pkg/ast"
	"github.com/MacAttak/pi-scanner/pkg/detection"
	"github.com/MacAttak/pi-scanner/pkg/llm"
)

// ContextAwareDetector enhances detection with code context understanding
type ContextAwareDetector struct {
	baseDetector detection.Detector
	llmClient    llm.Client
	astAnalyzer  *ast.Analyzer
	config       *ContextAwareConfig
}

// ContextAwareConfig contains configuration for context-aware detection
type ContextAwareConfig struct {
	EnableLLMValidation   bool                      `json:"enable_llm_validation"`
	EnableASTAnalysis     bool                      `json:"enable_ast_analysis"`
	ConfidenceThreshold   float64                   `json:"confidence_threshold"`
	MaxLLMTokens          int                       `json:"max_llm_tokens"`
	ContextWindowSize     int                       `json:"context_window_size"`
	BankingDomainMode     bool                      `json:"banking_domain_mode"`
	CustomPromptTemplate  string                    `json:"custom_prompt_template"`
	FalsePositivePatterns []string                  `json:"false_positive_patterns"`
	ValidationRules       map[string]ValidationRule `json:"validation_rules"`
}

// ValidationRule defines rules for validating specific PI types
type ValidationRule struct {
	Type            string   `json:"type"`
	RequiredContext []string `json:"required_context"`
	ExcludeContext  []string `json:"exclude_context"`
	MinConfidence   float64  `json:"min_confidence"`
}

// ContextualFinding extends Finding with context information
type ContextualFinding struct {
	detection.Finding
	Context          CodeContext      `json:"context"`
	ValidationResult ValidationResult `json:"validation_result"`
	RiskAssessment   RiskAssessment   `json:"risk_assessment"`
}

// CodeContext provides contextual information about the code
type CodeContext struct {
	Language        string              `json:"language"`
	FileType        string              `json:"file_type"`
	IsTestFile      bool                `json:"is_test_file"`
	IsConfigFile    bool                `json:"is_config_file"`
	SurroundingCode string              `json:"surrounding_code"`
	ASTInfo         *ast.AnalysisResult `json:"ast_info,omitempty"`
	SemanticContext string              `json:"semantic_context"`
}

// ValidationResult contains the result of LLM validation
type ValidationResult struct {
	IsValid            bool     `json:"is_valid"`
	Confidence         float64  `json:"confidence"`
	Reasoning          string   `json:"reasoning"`
	SuggestedType      string   `json:"suggested_type,omitempty"`
	ContextualEvidence []string `json:"contextual_evidence"`
	FalsePositiveScore float64  `json:"false_positive_score"`
}

// RiskAssessment provides risk analysis for the finding
type RiskAssessment struct {
	RiskLevel        string   `json:"risk_level"` // LOW, MEDIUM, HIGH, CRITICAL
	DataSensitivity  string   `json:"data_sensitivity"`
	ExposureContext  string   `json:"exposure_context"`
	Recommendations  []string `json:"recommendations"`
	ComplianceImpact []string `json:"compliance_impact"`
}

// NewContextAwareDetector creates a new context-aware detector
func NewContextAwareDetector(baseDetector detection.Detector, llmClient llm.Client, config *ContextAwareConfig) *ContextAwareDetector {
	if config == nil {
		config = DefaultContextAwareConfig()
	}

	astConfig := ast.DefaultConfig()
	if config.BankingDomainMode {
		// Ensure banking domain configuration is applied
		astConfig.BankingDomainRules = ast.DefaultBankingDomainConfig()
	}

	return &ContextAwareDetector{
		baseDetector: baseDetector,
		llmClient:    llmClient,
		astAnalyzer:  ast.NewAnalyzer(astConfig),
		config:       config,
	}
}

// DefaultContextAwareConfig returns the default configuration
func DefaultContextAwareConfig() *ContextAwareConfig {
	return &ContextAwareConfig{
		EnableLLMValidation: true,
		EnableASTAnalysis:   true,
		ConfidenceThreshold: 0.7,
		MaxLLMTokens:        1000,
		ContextWindowSize:   10, // lines of context
		BankingDomainMode:   true,
		FalsePositivePatterns: []string{
			// Common false positive patterns
			"test", "example", "sample", "demo", "mock",
			"lorem ipsum", "john doe", "jane doe",
			"123456789", "000000000", "111111111",
			"XXXXXXXXX", "****", "____",
		},
		ValidationRules: map[string]ValidationRule{
			"AUSTRALIAN_TAX_FILE_NUMBER": {
				Type:            "TFN",
				RequiredContext: []string{"tax", "tfn", "abn", "customer", "person"},
				ExcludeContext:  []string{"test", "example", "mock"},
				MinConfidence:   0.8,
			},
			"AUSTRALIAN_BUSINESS_NUMBER": {
				Type:            "ABN",
				RequiredContext: []string{"business", "company", "abn", "entity"},
				ExcludeContext:  []string{"test", "sample"},
				MinConfidence:   0.7,
			},
			"AUSTRALIAN_MEDICARE_NUMBER": {
				Type:            "Medicare",
				RequiredContext: []string{"medicare", "health", "patient", "medical"},
				ExcludeContext:  []string{"test", "dummy"},
				MinConfidence:   0.85,
			},
			"BANK_ACCOUNT": {
				Type:            "BankAccount",
				RequiredContext: []string{"account", "bank", "bsb", "payment"},
				ExcludeContext:  []string{"test", "sample"},
				MinConfidence:   0.75,
			},
		},
	}
}

// Detect performs context-aware detection
func (d *ContextAwareDetector) Detect(ctx context.Context, content []byte, filename string) ([]detection.Finding, error) {
	// First, run base detection
	baseFindings, err := d.baseDetector.Detect(ctx, content, filename)
	if err != nil {
		return nil, fmt.Errorf("base detection failed: %w", err)
	}

	// If no findings or enhancements disabled, return base results
	if len(baseFindings) == 0 || (!d.config.EnableLLMValidation && !d.config.EnableASTAnalysis) {
		return baseFindings, nil
	}

	// Analyze file context
	fileContext := d.analyzeFileContext(ctx, content, filename)

	// Enhance findings with context
	contextualFindings := make([]detection.Finding, 0)
	for _, finding := range baseFindings {
		enhanced := d.enhanceFinding(ctx, finding, content, fileContext)

		// Apply confidence threshold
		if enhanced.ValidationResult.Confidence >= d.config.ConfidenceThreshold {
			// Convert back to regular Finding for compatibility
			contextualFindings = append(contextualFindings, enhanced.Finding)
		}
	}

	return contextualFindings, nil
}

// analyzeFileContext analyzes the overall file context
func (d *ContextAwareDetector) analyzeFileContext(ctx context.Context, content []byte, filename string) CodeContext {
	context := CodeContext{
		Language:     d.detectLanguage(filename),
		FileType:     d.detectFileType(filename),
		IsTestFile:   d.isTestFile(filename),
		IsConfigFile: d.isConfigFile(filename),
	}

	// Perform AST analysis if enabled
	if d.config.EnableASTAnalysis && d.astAnalyzer != nil {
		astResult, err := d.astAnalyzer.AnalyzeFile(ctx, filename, content)
		if err == nil {
			context.ASTInfo = astResult
			context.SemanticContext = d.extractSemanticContext(astResult)
		}
	}

	return context
}

// enhanceFinding enhances a finding with contextual information
func (d *ContextAwareDetector) enhanceFinding(ctx context.Context, finding detection.Finding, content []byte, fileContext CodeContext) ContextualFinding {
	enhanced := ContextualFinding{
		Finding: finding,
		Context: fileContext,
	}

	// Extract surrounding code context
	enhanced.Context.SurroundingCode = d.extractSurroundingCode(content, finding.Line, finding.Line)

	// Perform LLM validation if enabled
	if d.config.EnableLLMValidation && d.llmClient != nil {
		enhanced.ValidationResult = d.validateWithLLM(ctx, enhanced)

		// Update confidence based on validation
		if enhanced.ValidationResult.IsValid {
			enhanced.Finding.Confidence = float32(float64(enhanced.Finding.Confidence) * enhanced.ValidationResult.Confidence)
		} else {
			enhanced.Finding.Confidence = float32(float64(enhanced.Finding.Confidence) * (1.0 - enhanced.ValidationResult.FalsePositiveScore))
		}
	}

	// Assess risk
	enhanced.RiskAssessment = d.assessRisk(enhanced)

	return enhanced
}

// validateWithLLM uses the LLM to validate a finding
func (d *ContextAwareDetector) validateWithLLM(ctx context.Context, finding ContextualFinding) ValidationResult {
	prompt := d.buildValidationPrompt(finding)

	response, err := d.llmClient.Complete(ctx, prompt, &llm.CompletionOptions{
		MaxTokens:   d.config.MaxLLMTokens,
		Temperature: 0.1, // Low temperature for consistency
	})

	if err != nil {
		// If LLM fails, return neutral result
		return ValidationResult{
			IsValid:            true,
			Confidence:         0.5,
			Reasoning:          "LLM validation unavailable",
			FalsePositiveScore: 0.5,
		}
	}

	// Parse LLM response
	return d.parseValidationResponse(response)
}

// buildValidationPrompt creates a prompt for LLM validation
func (d *ContextAwareDetector) buildValidationPrompt(finding ContextualFinding) string {
	if d.config.CustomPromptTemplate != "" {
		return d.fillPromptTemplate(d.config.CustomPromptTemplate, finding)
	}

	prompt := fmt.Sprintf(`Analyze this potential %s finding in %s code and determine if it's a genuine sensitive data instance or a false positive.

Finding Details:
- Type: %s
- Value: %s
- File: %s (Test file: %v, Config file: %v)
- Line: %d

Surrounding Code:
%s

Context:
- Language: %s
- Risk Zone: %s
- Semantic Context: %s

Please analyze:
1. Is this genuinely sensitive %s data that should be protected?
2. What contextual clues support or refute this being real sensitive data?
3. What is the confidence level (0-1) that this is a true positive?
4. What is the false positive score (0-1)?

Respond in JSON format:
{
  "is_valid": boolean,
  "confidence": float,
  "reasoning": "detailed explanation",
  "contextual_evidence": ["evidence1", "evidence2"],
  "false_positive_score": float,
  "suggested_type": "corrected type if applicable"
}`,
		finding.Type,
		finding.Context.Language,
		finding.Type,
		finding.Match,
		finding.File,
		finding.Context.IsTestFile,
		finding.Context.IsConfigFile,
		finding.Line,
		finding.Context.SurroundingCode,
		finding.Context.Language,
		d.getRiskZone(finding),
		finding.Context.SemanticContext,
		finding.Type,
	)

	return prompt
}

// Helper methods

func (d *ContextAwareDetector) detectLanguage(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".java":
		return "Java"
	case ".scala":
		return "Scala"
	case ".py":
		return "Python"
	case ".go":
		return "Go"
	case ".js", ".ts":
		return "JavaScript"
	default:
		return "Unknown"
	}
}

func (d *ContextAwareDetector) detectFileType(filename string) string {
	lower := strings.ToLower(filename)

	if strings.Contains(lower, "test") || strings.Contains(lower, "spec") {
		return "test"
	}
	if strings.Contains(lower, "config") || strings.Contains(lower, "properties") {
		return "configuration"
	}
	if strings.Contains(lower, "model") || strings.Contains(lower, "entity") {
		return "data_model"
	}
	if strings.Contains(lower, "service") || strings.Contains(lower, "controller") {
		return "business_logic"
	}

	return "source_code"
}

func (d *ContextAwareDetector) isTestFile(filename string) bool {
	lower := strings.ToLower(filename)
	return strings.Contains(lower, "test") ||
		strings.Contains(lower, "spec") ||
		strings.Contains(lower, "_test.") ||
		strings.Contains(lower, ".test.")
}

func (d *ContextAwareDetector) isConfigFile(filename string) bool {
	lower := strings.ToLower(filename)
	ext := strings.ToLower(filepath.Ext(filename))

	return strings.Contains(lower, "config") ||
		ext == ".properties" ||
		ext == ".yaml" ||
		ext == ".yml" ||
		ext == ".json" ||
		ext == ".xml"
}

func (d *ContextAwareDetector) extractSurroundingCode(content []byte, startLine, endLine int) string {
	lines := strings.Split(string(content), "\n")

	// Convert to 0-based indexing and calculate context window
	contextStart := max(0, startLine-1-d.config.ContextWindowSize)
	contextEnd := min(len(lines), endLine+d.config.ContextWindowSize)

	var contextLines []string
	for i := contextStart; i < contextEnd; i++ {
		prefix := "  "
		if i >= startLine-1 && i <= endLine-1 {
			prefix = "> "
		}
		contextLines = append(contextLines, fmt.Sprintf("%s%d: %s", prefix, i+1, lines[i]))
	}

	return strings.Join(contextLines, "\n")
}

func (d *ContextAwareDetector) extractSemanticContext(astResult *ast.AnalysisResult) string {
	if astResult == nil {
		return "No AST information available"
	}

	var context []string

	// Add risk level and zone
	context = append(context, fmt.Sprintf("Risk Level: %s, Risk Zone: %s", astResult.RiskLevel, astResult.RiskZone))

	// Add class context
	if astResult.CodeStructure != nil && len(astResult.CodeStructure.Classes) > 0 {
		classes := make([]string, 0, len(astResult.CodeStructure.Classes))
		for _, class := range astResult.CodeStructure.Classes {
			classes = append(classes, class.Name)
		}
		context = append(context, fmt.Sprintf("Classes: %s", strings.Join(classes, ", ")))
	}

	// Add security hints
	if len(astResult.SecurityHints) > 0 {
		context = append(context, fmt.Sprintf("Security concerns: %d issues found", len(astResult.SecurityHints)))
	}

	return strings.Join(context, "; ")
}

func (d *ContextAwareDetector) getRiskZone(finding ContextualFinding) string {
	if finding.Context.ASTInfo != nil {
		return finding.Context.ASTInfo.RiskZone
	}
	return "unknown"
}

func (d *ContextAwareDetector) parseValidationResponse(response string) ValidationResult {
	var result ValidationResult

	// Try to parse JSON response
	if err := json.Unmarshal([]byte(response), &result); err != nil {
		// Fallback to simple parsing
		result.IsValid = !strings.Contains(strings.ToLower(response), "false positive")
		result.Confidence = 0.5
		result.Reasoning = response
		result.FalsePositiveScore = 0.5
	}

	return result
}

func (d *ContextAwareDetector) assessRisk(finding ContextualFinding) RiskAssessment {
	assessment := RiskAssessment{
		RiskLevel:       "MEDIUM",
		Recommendations: []string{},
	}

	// Determine risk level based on context
	if finding.Context.IsTestFile {
		assessment.RiskLevel = "LOW"
		assessment.ExposureContext = "Test file - limited exposure"
	} else if finding.Context.ASTInfo != nil {
		switch finding.Context.ASTInfo.RiskLevel {
		case ast.RiskLevelCritical:
			assessment.RiskLevel = "CRITICAL"
		case ast.RiskLevelHigh:
			assessment.RiskLevel = "HIGH"
		case ast.RiskLevelMedium:
			assessment.RiskLevel = "MEDIUM"
		case ast.RiskLevelLow:
			assessment.RiskLevel = "LOW"
		}
		assessment.ExposureContext = fmt.Sprintf("Located in %s zone", finding.Context.ASTInfo.RiskZone)
	}

	// Add data sensitivity based on type
	switch finding.Type {
	case detection.PITypeTFN:
		assessment.DataSensitivity = "HIGHLY_SENSITIVE"
		assessment.ComplianceImpact = []string{"Privacy Act 1988", "ATO Guidelines"}
	case detection.PITypeMedicare:
		assessment.DataSensitivity = "HIGHLY_SENSITIVE"
		assessment.ComplianceImpact = []string{"Healthcare Identifiers Act 2010"}
	case detection.PITypeBankAccount:
		assessment.DataSensitivity = "SENSITIVE"
		assessment.ComplianceImpact = []string{"PCI-DSS", "Banking Code of Practice"}
	default:
		assessment.DataSensitivity = "SENSITIVE"
	}

	// Add recommendations
	if assessment.RiskLevel == "HIGH" || assessment.RiskLevel == "CRITICAL" {
		assessment.Recommendations = append(assessment.Recommendations,
			"Immediate remediation required",
			"Move sensitive data to secure configuration",
			"Implement encryption at rest",
		)
	}

	return assessment
}

func (d *ContextAwareDetector) fillPromptTemplate(template string, finding ContextualFinding) string {
	// Implementation for custom prompt templates
	return template
}

// Name implements the Detector interface
func (d *ContextAwareDetector) Name() string {
	return fmt.Sprintf("ContextAware(%s)", d.baseDetector.Name())
}

// Utility functions
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
