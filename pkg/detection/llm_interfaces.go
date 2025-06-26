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
	Finding       Finding   `json:"finding"`
	Context       string    `json:"context"`
	FilePath      string    `json:"file_path"`
	FileType      string    `json:"file_type"`
	IsTestFile    bool      `json:"is_test_file"`
	SurroundingPI []Finding `json:"surrounding_pi,omitempty"`
}

// LLMValidationResult contains the LLM's assessment of a finding
type LLMValidationResult struct {
	Risk        RiskLevel `json:"risk"`
	Explanation string    `json:"explanation"`
	Confidence  float64   `json:"confidence"`
	Timestamp   time.Time `json:"timestamp"`
}
