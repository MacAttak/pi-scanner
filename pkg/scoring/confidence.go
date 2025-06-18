package scoring

import (
	"context"
	"fmt"
	"time"

	"github.com/pi-scanner/pi-scanner/pkg/detection"
)

// ConfidenceEngine is the main engine responsible for calculating confidence scores
// by integrating multiple detection methods and providing risk-based scoring
// aligned with Australian banking regulations.
type ConfidenceEngine struct {
	config       *EngineConfig
	factorEngine *FactorEngine
	aggregator   *ScoreAggregator
}

// EngineConfig holds configuration for the confidence engine
type EngineConfig struct {
	// Thresholds
	MinConfidenceThreshold float64 `json:"min_confidence_threshold"`
	MaxConfidenceThreshold float64 `json:"max_confidence_threshold"`
	
	// Integration settings
	EnableProximityScoring bool `json:"enable_proximity_scoring"`
	EnableMLScoring        bool `json:"enable_ml_scoring"`
	EnableValidationScoring bool `json:"enable_validation_scoring"`
	
	// Australian regulatory compliance
	APRACompliance        bool `json:"apra_compliance"`
	PrivacyActCompliance  bool `json:"privacy_act_compliance"`
	BankingRegCompliance  bool `json:"banking_reg_compliance"`
	
	// Risk level thresholds (Australian banking aligned)
	CriticalThreshold float64 `json:"critical_threshold"` // 0.9+
	HighThreshold     float64 `json:"high_threshold"`     // 0.7-0.89
	MediumThreshold   float64 `json:"medium_threshold"`   // 0.4-0.69
	// Low is everything else (0.0-0.39)
	
	// Performance settings
	EnableAuditTrail      bool `json:"enable_audit_trail"`
	EnableDetailedLogging bool `json:"enable_detailed_logging"`
}

// DefaultEngineConfig returns the default configuration optimized for Australian PI detection
func DefaultEngineConfig() *EngineConfig {
	return &EngineConfig{
		MinConfidenceThreshold:  0.0,
		MaxConfidenceThreshold:  1.0,
		EnableProximityScoring:  true,
		EnableMLScoring:        true,
		EnableValidationScoring: true,
		APRACompliance:         true,
		PrivacyActCompliance:   true,
		BankingRegCompliance:   true,
		CriticalThreshold:      0.9,
		HighThreshold:          0.7,
		MediumThreshold:        0.4,
		EnableAuditTrail:       true,
		EnableDetailedLogging:  false,
	}
}

// ScoreInput contains all the input data needed for confidence scoring
type ScoreInput struct {
	// Core finding data
	Finding detection.Finding `json:"finding"`
	Content string           `json:"content"`
	
	// Component scores
	ProximityScore  *ProximityScore  `json:"proximity_score,omitempty"`
	MLScore         *MLScore         `json:"ml_score,omitempty"`
	ValidationScore *ValidationScore `json:"validation_score,omitempty"`
	
	// Context data
	CoOccurrences []CoOccurrence `json:"co_occurrences,omitempty"`
	Environment   string         `json:"environment,omitempty"`
	
	// Metadata
	ScanTimestamp time.Time `json:"scan_timestamp"`
}

// ProximityScore represents proximity detection results
type ProximityScore struct {
	Score    float64  `json:"score"`
	Context  string   `json:"context"`
	Keywords []string `json:"keywords"`
	Distance int      `json:"distance"`
}

// MLScore represents machine learning model results
type MLScore struct {
	Confidence float32 `json:"confidence"`
	PIType     string  `json:"pi_type"`
	IsValid    bool    `json:"is_valid"`
	ModelName  string  `json:"model_name,omitempty"`
}

// ValidationScore represents algorithmic validation results
type ValidationScore struct {
	IsValid    bool    `json:"is_valid"`
	Algorithm  string  `json:"algorithm"`
	Confidence float64 `json:"confidence"`
	Details    string  `json:"details,omitempty"`
}

// CoOccurrence represents other PI types found in proximity
type CoOccurrence struct {
	PIType   detection.PIType `json:"pi_type"`
	Distance int              `json:"distance"`
	Match    string           `json:"match"`
}

// ConfidenceResult represents the final confidence scoring result
type ConfidenceResult struct {
	// Final score and risk assessment
	FinalScore float64   `json:"final_score"`
	RiskLevel  RiskLevel `json:"risk_level"`
	
	// Detailed breakdown
	Breakdown    ScoreBreakdown    `json:"breakdown"`
	AuditTrail   []AuditEntry      `json:"audit_trail"`
	
	// Regulatory compliance
	RegulatoryCompliance RegulatoryCompliance `json:"regulatory_compliance"`
	
	// Metadata
	CalculatedAt time.Time `json:"calculated_at"`
	Version      string    `json:"version"`
}

// RiskLevel represents the risk severity following Australian banking guidelines
type RiskLevel string

const (
	RiskLevelCritical RiskLevel = "CRITICAL" // 0.9+ - Immediate action required
	RiskLevelHigh     RiskLevel = "HIGH"     // 0.7-0.89 - Urgent attention needed
	RiskLevelMedium   RiskLevel = "MEDIUM"   // 0.4-0.69 - Review required
	RiskLevelLow      RiskLevel = "LOW"      // 0.0-0.39 - Monitor
)

// ScoreBreakdown provides detailed information about how the score was calculated
type ScoreBreakdown struct {
	FinalScore  float64             `json:"final_score"`
	Components  []ScoreComponent    `json:"components"`
	Weights     map[string]float64  `json:"weights"`
	Adjustments []ScoreAdjustment   `json:"adjustments"`
}

// ScoreComponent represents an individual scoring component
type ScoreComponent struct {
	Name        string  `json:"name"`
	Score       float64 `json:"score"`
	Weight      float64 `json:"weight"`
	Description string  `json:"description"`
}

// ScoreAdjustment represents modifications to the base score
type ScoreAdjustment struct {
	Type        string  `json:"type"`
	Impact      float64 `json:"impact"`
	Description string  `json:"description"`
}

// AuditEntry represents a single audit trail entry for regulatory compliance
type AuditEntry struct {
	Component   string            `json:"component"`
	Timestamp   time.Time         `json:"timestamp"`
	Score       float64           `json:"score"`
	Description string            `json:"description"`
	Details     map[string]string `json:"details"`
}

// RegulatoryCompliance represents Australian regulatory compliance information
type RegulatoryCompliance struct {
	APRA            bool             `json:"apra_compliance"`
	PrivacyAct      bool             `json:"privacy_act_compliance"`
	RequiredActions []ComplianceAction `json:"required_actions"`
}

// ComplianceAction represents a required action for regulatory compliance
type ComplianceAction struct {
	Type        string    `json:"type"`
	Description string    `json:"description"`
	Priority    string    `json:"priority"`
	Deadline    time.Time `json:"deadline"`
}

// NewConfidenceEngine creates a new confidence scoring engine
func NewConfidenceEngine(config *EngineConfig) (*ConfidenceEngine, error) {
	if config == nil {
		config = DefaultEngineConfig()
	}
	
	// Validate configuration
	if err := validateEngineConfig(config); err != nil {
		return nil, fmt.Errorf("invalid engine configuration: %w", err)
	}
	
	// Create factor engine
	factorEngine, err := NewFactorEngine(nil) // Use default factor config
	if err != nil {
		return nil, fmt.Errorf("failed to create factor engine: %w", err)
	}
	
	// Create score aggregator
	aggregator, err := NewScoreAggregator(nil) // Use default aggregator config
	if err != nil {
		return nil, fmt.Errorf("failed to create score aggregator: %w", err)
	}
	
	return &ConfidenceEngine{
		config:       config,
		factorEngine: factorEngine,
		aggregator:   aggregator,
	}, nil
}

// CalculateScore computes the confidence score for a given PI finding
func (e *ConfidenceEngine) CalculateScore(ctx context.Context, input ScoreInput) (*ConfidenceResult, error) {
	// Validate input
	if err := e.validateInput(input); err != nil {
		return nil, fmt.Errorf("invalid input: %w", err)
	}
	
	startTime := time.Now()
	
	// Calculate individual factor scores
	factors, err := e.calculateFactorScores(input)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate factor scores: %w", err)
	}
	
	// Aggregate scores
	finalScore := e.aggregator.AggregateScores(factors)
	
	// Map to risk level
	riskLevel := e.mapScoreToRiskLevel(finalScore)
	
	// Generate detailed breakdown
	breakdown := e.aggregator.GenerateScoreBreakdown(factors, finalScore)
	
	// Generate audit trail if enabled
	var auditTrail []AuditEntry
	if e.config.EnableAuditTrail {
		auditTrail = e.aggregator.GenerateAuditTrail(factors, finalScore, input.Finding.Type)
	}
	
	// Generate regulatory compliance information
	compliance := e.aggregator.GenerateRegulatoryCompliance(input.Finding.Type, riskLevel)
	
	result := &ConfidenceResult{
		FinalScore:           finalScore,
		RiskLevel:           riskLevel,
		Breakdown:           breakdown,
		AuditTrail:          auditTrail,
		RegulatoryCompliance: compliance,
		CalculatedAt:        startTime,
		Version:             "1.0",
	}
	
	return result, nil
}

// calculateFactorScores computes all individual factor scores
func (e *ConfidenceEngine) calculateFactorScores(input ScoreInput) (FactorScores, error) {
	factors := FactorScores{}
	
	// Proximity factor
	if e.config.EnableProximityScoring {
		factors.ProximityScore = e.factorEngine.CalculateProximityFactor(input.ProximityScore)
	} else {
		factors.ProximityScore = 0.5 // neutral default
	}
	
	// ML factor
	if e.config.EnableMLScoring {
		factors.MLScore = e.factorEngine.CalculateMLFactor(input.MLScore)
	} else {
		factors.MLScore = 0.5 // neutral default
	}
	
	// Validation factor
	if e.config.EnableValidationScoring {
		factors.ValidationScore = e.factorEngine.CalculateValidationFactor(input.ValidationScore)
	} else {
		factors.ValidationScore = 0.0 // no validation data
	}
	
	// Environment factor
	factors.EnvironmentScore = e.factorEngine.CalculateEnvironmentFactor(input.Finding.File, input.Content)
	
	// Co-occurrence factor
	factors.CoOccurrenceScore = e.factorEngine.CalculateCoOccurrenceFactor(input.Finding.Type, input.CoOccurrences)
	
	// PI type weight
	factors.PITypeWeight = e.factorEngine.CalculatePITypeWeight(input.Finding.Type)
	
	return factors, nil
}

// mapScoreToRiskLevel maps a confidence score to a risk level based on Australian banking guidelines
func (e *ConfidenceEngine) mapScoreToRiskLevel(score float64) RiskLevel {
	if score >= e.config.CriticalThreshold {
		return RiskLevelCritical
	} else if score >= e.config.HighThreshold {
		return RiskLevelHigh
	} else if score >= e.config.MediumThreshold {
		return RiskLevelMedium
	}
	return RiskLevelLow
}

// validateInput ensures the input data is valid
func (e *ConfidenceEngine) validateInput(input ScoreInput) error {
	// Check if finding has required fields
	if input.Finding.Type == "" {
		return fmt.Errorf("invalid finding: missing PI type")
	}
	
	if input.Finding.Match == "" {
		return fmt.Errorf("invalid finding: missing match")
	}
	
	// Validate PI type is supported
	supportedTypes := map[detection.PIType]bool{
		detection.PITypeTFN:          true,
		detection.PITypeMedicare:     true,
		detection.PITypeABN:          true,
		detection.PITypeBSB:          true,
		detection.PITypeEmail:        true,
		detection.PITypePhone:        true,
		detection.PITypeName:         true,
		detection.PITypeAddress:      true,
		detection.PITypeCreditCard:   true,
		detection.PITypeDriverLicense: true,
		detection.PITypePassport:     true,
		detection.PITypeAccount:      true,
		detection.PITypeIP:           true,
	}
	
	if !supportedTypes[input.Finding.Type] {
		return fmt.Errorf("unsupported PI type: %s", input.Finding.Type)
	}
	
	return nil
}

// validateEngineConfig validates the engine configuration
func validateEngineConfig(config *EngineConfig) error {
	if config.MinConfidenceThreshold < 0.0 || config.MinConfidenceThreshold > 1.0 {
		return fmt.Errorf("invalid confidence threshold: min=%f", config.MinConfidenceThreshold)
	}
	
	if config.MaxConfidenceThreshold < 0.0 || config.MaxConfidenceThreshold > 1.0 {
		return fmt.Errorf("invalid confidence threshold: max=%f", config.MaxConfidenceThreshold)
	}
	
	if config.MinConfidenceThreshold > config.MaxConfidenceThreshold {
		return fmt.Errorf("min threshold (%f) cannot be greater than max threshold (%f)", 
			config.MinConfidenceThreshold, config.MaxConfidenceThreshold)
	}
	
	// Validate risk level thresholds
	if config.CriticalThreshold <= config.HighThreshold {
		return fmt.Errorf("critical threshold (%f) must be greater than high threshold (%f)", 
			config.CriticalThreshold, config.HighThreshold)
	}
	
	if config.HighThreshold <= config.MediumThreshold {
		return fmt.Errorf("high threshold (%f) must be greater than medium threshold (%f)", 
			config.HighThreshold, config.MediumThreshold)
	}
	
	if config.MediumThreshold < 0.0 {
		return fmt.Errorf("medium threshold (%f) cannot be negative", config.MediumThreshold)
	}
	
	return nil
}

// GetEngineInfo returns information about the confidence engine
func (e *ConfidenceEngine) GetEngineInfo() map[string]interface{} {
	return map[string]interface{}{
		"version":              "1.0",
		"australian_compliant": true,
		"apra_compliance":      e.config.APRACompliance,
		"privacy_act":          e.config.PrivacyActCompliance,
		"banking_regulations":  e.config.BankingRegCompliance,
		"risk_thresholds": map[string]float64{
			"critical": e.config.CriticalThreshold,
			"high":     e.config.HighThreshold,
			"medium":   e.config.MediumThreshold,
		},
		"enabled_features": map[string]bool{
			"proximity_scoring":  e.config.EnableProximityScoring,
			"ml_scoring":        e.config.EnableMLScoring,
			"validation_scoring": e.config.EnableValidationScoring,
			"audit_trail":       e.config.EnableAuditTrail,
		},
	}
}