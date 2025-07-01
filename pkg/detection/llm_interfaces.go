package detection

import (
	"context"
	"time"
)

// LLMValidator provides context-aware validation of findings
type LLMValidator interface {
	ValidateFinding(ctx context.Context, req LLMValidationRequest) (*LLMValidationResult, error)
	HealthCheck(ctx context.Context) error
}

// LLMValidationRequest contains all information needed for LLM validation
type LLMValidationRequest struct {
	Finding       Finding     `json:"finding"`
	Context       string      `json:"context"`
	FilePath      string      `json:"file_path"`
	FileType      string      `json:"file_type"`
	IsTestFile    bool        `json:"is_test_file"`
	SurroundingPI []Finding   `json:"surrounding_pi,omitempty"`
	ASTContext    *ASTContext `json:"ast_context,omitempty"`
}

// LLMValidationResult contains the LLM's assessment of a finding
type LLMValidationResult struct {
	Risk        RiskLevel `json:"risk"`
	Explanation string    `json:"explanation"`
	Confidence  float64   `json:"confidence"`
	Timestamp   time.Time `json:"timestamp"`
}

// ASTContext contains structural information from AST analysis
type ASTContext struct {
	// File-level information
	Language     string `json:"language"`
	FileType     string `json:"file_type"`  // e.g., "test", "config", "model", "controller"
	RiskZone     string `json:"risk_zone"`  // e.g., "customer_data", "payment_processing"
	RiskLevel    string `json:"risk_level"` // Critical, High, Medium, Low
	IsTestFile   bool   `json:"is_test_file"`
	IsConfigFile bool   `json:"is_config_file"`

	// Code structure
	Classes      []string `json:"classes,omitempty"`      // Class/type names defined in file
	Methods      []string `json:"methods,omitempty"`      // Method/function names
	Imports      []string `json:"imports,omitempty"`      // Import statements
	Dependencies []string `json:"dependencies,omitempty"` // External dependencies

	// Banking domain context
	BankingDomainIndicators []string `json:"banking_indicators,omitempty"` // e.g., "handles_customer_data", "processes_payments"
	SecurityPatterns        []string `json:"security_patterns,omitempty"`  // e.g., "uses_encryption", "has_authentication"

	// Surrounding code context
	EnclosingClass  string `json:"enclosing_class,omitempty"`  // Class containing the finding
	EnclosingMethod string `json:"enclosing_method,omitempty"` // Method containing the finding
	NearbyComments  string `json:"nearby_comments,omitempty"`  // Relevant comments near the finding
}
