package scoring

import (
	"fmt"
	"time"

	"github.com/MacAttak/pi-scanner/pkg/detection"
)

// ScoreAggregator combines individual factor scores into a final confidence score
// and provides detailed breakdowns for audit and regulatory compliance
type ScoreAggregator struct {
	config *AggregatorConfig
}

// AggregatorConfig holds configuration for score aggregation
type AggregatorConfig struct {
	// Aggregation method
	WeightedCombination bool    `json:"weighted_combination"`
	ProximityWeight     float64 `json:"proximity_weight"`
	MLWeight            float64 `json:"ml_weight"`
	ValidationWeight    float64 `json:"validation_weight"`

	// Score bounds
	MaxScore float64 `json:"max_score"`
	MinScore float64 `json:"min_score"`

	// Risk level thresholds (aligned with Australian banking regulations)
	CriticalThreshold float64 `json:"critical_threshold"` // 0.9+
	HighThreshold     float64 `json:"high_threshold"`     // 0.7-0.89
	MediumThreshold   float64 `json:"medium_threshold"`   // 0.4-0.69

	// Regulatory compliance settings
	APRACompliance       bool `json:"apra_compliance"`
	PrivacyActCompliance bool `json:"privacy_act_compliance"`

	// Audit trail settings
	DetailedAuditTrail bool `json:"detailed_audit_trail"`
	IncludeTimestamps  bool `json:"include_timestamps"`
}

// DefaultAggregatorConfig returns the default aggregator configuration
func DefaultAggregatorConfig() *AggregatorConfig {
	return &AggregatorConfig{
		WeightedCombination:  true,
		ProximityWeight:      0.4,
		MLWeight:             0.3,
		ValidationWeight:     0.3,
		MaxScore:             1.0,
		MinScore:             0.0,
		CriticalThreshold:    0.9,
		HighThreshold:        0.7,
		MediumThreshold:      0.4,
		APRACompliance:       true,
		PrivacyActCompliance: true,
		DetailedAuditTrail:   true,
		IncludeTimestamps:    true,
	}
}

// NewScoreAggregator creates a new score aggregator
func NewScoreAggregator(config *AggregatorConfig) (*ScoreAggregator, error) {
	if config == nil {
		config = DefaultAggregatorConfig()
	}

	return &ScoreAggregator{
		config: config,
	}, nil
}

// AggregateScores combines all factor scores into a final confidence score
func (a *ScoreAggregator) AggregateScores(factors FactorScores) float64 {
	var baseScore float64

	if a.config.WeightedCombination {
		// Weighted combination of primary scores
		baseScore = factors.ProximityScore*a.config.ProximityWeight +
			factors.MLScore*a.config.MLWeight +
			factors.ValidationScore*a.config.ValidationWeight
	} else {
		// Simple average of primary scores
		baseScore = (factors.ProximityScore + factors.MLScore + factors.ValidationScore) / 3.0
	}

	// Apply PI type weight
	baseScore *= factors.PITypeWeight

	// Apply environment factor
	baseScore *= factors.EnvironmentScore

	// Apply co-occurrence boost/penalty
	baseScore *= factors.CoOccurrenceScore

	// Ensure score is within bounds
	if baseScore < a.config.MinScore {
		baseScore = a.config.MinScore
	}
	if baseScore > a.config.MaxScore {
		baseScore = a.config.MaxScore
	}

	return baseScore
}

// MapScoreToRiskLevel maps a confidence score to a risk level
func (a *ScoreAggregator) MapScoreToRiskLevel(score float64) RiskLevel {
	if score >= a.config.CriticalThreshold {
		return RiskLevelCritical
	} else if score >= a.config.HighThreshold {
		return RiskLevelHigh
	} else if score >= a.config.MediumThreshold {
		return RiskLevelMedium
	}
	return RiskLevelLow
}

// GenerateScoreBreakdown creates a detailed breakdown of how the score was calculated
func (a *ScoreAggregator) GenerateScoreBreakdown(factors FactorScores, finalScore float64) ScoreBreakdown {
	components := []ScoreComponent{
		{
			Name:        "proximity",
			Score:       factors.ProximityScore,
			Weight:      a.config.ProximityWeight,
			Description: "Context and proximity analysis score",
		},
		{
			Name:        "ml_validation",
			Score:       factors.MLScore,
			Weight:      a.config.MLWeight,
			Description: "Machine learning model confidence score",
		},
		{
			Name:        "algorithmic_validation",
			Score:       factors.ValidationScore,
			Weight:      a.config.ValidationWeight,
			Description: "Algorithmic validation result (checksum, format)",
		},
		{
			Name:        "environment",
			Score:       factors.EnvironmentScore,
			Weight:      1.0, // Multiplicative factor
			Description: "Environment context factor (test vs production)",
		},
		{
			Name:        "co_occurrence",
			Score:       factors.CoOccurrenceScore,
			Weight:      1.0, // Multiplicative factor
			Description: "Co-occurrence with other PI types",
		},
		{
			Name:        "pi_type_weight",
			Score:       factors.PITypeWeight,
			Weight:      1.0, // Multiplicative factor
			Description: "PI type importance weight (regulatory priority)",
		},
	}

	weights := map[string]float64{
		"proximity":              a.config.ProximityWeight,
		"ml_validation":          a.config.MLWeight,
		"algorithmic_validation": a.config.ValidationWeight,
		"environment":            1.0,
		"co_occurrence":          1.0,
		"pi_type_weight":         1.0,
	}

	adjustments := a.calculateAdjustments(factors)

	return ScoreBreakdown{
		FinalScore:  finalScore,
		Components:  components,
		Weights:     weights,
		Adjustments: adjustments,
	}
}

// calculateAdjustments identifies and quantifies score adjustments
func (a *ScoreAggregator) calculateAdjustments(factors FactorScores) []ScoreAdjustment {
	adjustments := []ScoreAdjustment{}

	// Environment adjustments
	if factors.EnvironmentScore < 1.0 {
		penalty := 1.0 - factors.EnvironmentScore
		adjustments = append(adjustments, ScoreAdjustment{
			Type:        "environment_penalty",
			Impact:      -penalty,
			Description: fmt.Sprintf("Environment penalty: %.1f%% reduction", penalty*100),
		})
	} else if factors.EnvironmentScore > 1.0 {
		bonus := factors.EnvironmentScore - 1.0
		adjustments = append(adjustments, ScoreAdjustment{
			Type:        "environment_bonus",
			Impact:      bonus,
			Description: fmt.Sprintf("Production environment bonus: %.1f%% increase", bonus*100),
		})
	}

	// Co-occurrence adjustments
	if factors.CoOccurrenceScore > 1.0 {
		boost := factors.CoOccurrenceScore - 1.0
		adjustments = append(adjustments, ScoreAdjustment{
			Type:        "co_occurrence_boost",
			Impact:      boost,
			Description: fmt.Sprintf("Co-occurrence with other PI types: %.1f%% increase", boost*100),
		})
	}

	// PI type weight adjustments
	if factors.PITypeWeight < 1.0 {
		reduction := 1.0 - factors.PITypeWeight
		adjustments = append(adjustments, ScoreAdjustment{
			Type:        "pi_type_weight",
			Impact:      -reduction,
			Description: fmt.Sprintf("Lower priority PI type: %.1f%% reduction", reduction*100),
		})
	}

	// Validation failure adjustment
	if factors.ValidationScore == 0.0 {
		adjustments = append(adjustments, ScoreAdjustment{
			Type:        "validation_failure",
			Impact:      -0.5, // Significant impact
			Description: "Algorithmic validation failed - likely false positive",
		})
	}

	return adjustments
}

// GenerateAuditTrail creates a comprehensive audit trail for regulatory compliance
func (a *ScoreAggregator) GenerateAuditTrail(factors FactorScores, finalScore float64, piType detection.PIType) []AuditEntry {
	now := time.Now()
	auditTrail := []AuditEntry{}

	// Proximity analysis audit
	auditTrail = append(auditTrail, AuditEntry{
		Component:   "proximity",
		Timestamp:   now,
		Score:       factors.ProximityScore,
		Description: "Proximity and context analysis completed",
		Details: map[string]string{
			"weight": fmt.Sprintf("%.2f", a.config.ProximityWeight),
			"impact": fmt.Sprintf("%.3f", factors.ProximityScore*a.config.ProximityWeight),
			"method": "pattern_matching_with_context_analysis",
		},
	})

	// ML validation audit
	auditTrail = append(auditTrail, AuditEntry{
		Component:   "ml_validation",
		Timestamp:   now.Add(1 * time.Millisecond),
		Score:       factors.MLScore,
		Description: "Machine learning validation completed",
		Details: map[string]string{
			"weight": fmt.Sprintf("%.2f", a.config.MLWeight),
			"impact": fmt.Sprintf("%.3f", factors.MLScore*a.config.MLWeight),
			"model":  "deberta-pi-validator",
		},
	})

	// Algorithmic validation audit
	auditTrail = append(auditTrail, AuditEntry{
		Component:   "algorithmic_validation",
		Timestamp:   now.Add(2 * time.Millisecond),
		Score:       factors.ValidationScore,
		Description: "Algorithmic validation completed",
		Details: map[string]string{
			"weight":    fmt.Sprintf("%.2f", a.config.ValidationWeight),
			"impact":    fmt.Sprintf("%.3f", factors.ValidationScore*a.config.ValidationWeight),
			"algorithm": a.getValidationAlgorithm(piType),
		},
	})

	// Environment analysis audit
	auditTrail = append(auditTrail, AuditEntry{
		Component:   "environment_analysis",
		Timestamp:   now.Add(3 * time.Millisecond),
		Score:       factors.EnvironmentScore,
		Description: "Environment context analysis completed",
		Details: map[string]string{
			"multiplier": fmt.Sprintf("%.3f", factors.EnvironmentScore),
			"impact":     a.getEnvironmentImpactDescription(factors.EnvironmentScore),
		},
	})

	// Co-occurrence analysis audit
	auditTrail = append(auditTrail, AuditEntry{
		Component:   "co_occurrence_analysis",
		Timestamp:   now.Add(4 * time.Millisecond),
		Score:       factors.CoOccurrenceScore,
		Description: "Co-occurrence analysis completed",
		Details: map[string]string{
			"multiplier": fmt.Sprintf("%.3f", factors.CoOccurrenceScore),
			"impact":     a.getCoOccurrenceImpactDescription(factors.CoOccurrenceScore),
		},
	})

	// Final aggregation audit
	auditTrail = append(auditTrail, AuditEntry{
		Component:   "aggregation",
		Timestamp:   now.Add(5 * time.Millisecond),
		Score:       finalScore,
		Description: "Final score aggregation completed",
		Details: map[string]string{
			"final_score":           fmt.Sprintf("%.6f", finalScore),
			"risk_level":            string(a.MapScoreToRiskLevel(finalScore)),
			"aggregation_method":    a.getAggregationMethod(),
			"regulatory_compliance": a.getRegulatoryComplianceStatus(piType),
		},
	})

	return auditTrail
}

// GenerateRegulatoryCompliance creates regulatory compliance information
func (a *ScoreAggregator) GenerateRegulatoryCompliance(piType detection.PIType, riskLevel RiskLevel) RegulatoryCompliance {
	compliance := RegulatoryCompliance{
		APRA:            a.isAPRARelevant(piType),
		PrivacyAct:      a.isPrivacyActRelevant(piType),
		RequiredActions: []ComplianceAction{},
	}

	// Generate required actions based on PI type and risk level
	actions := a.generateComplianceActions(piType, riskLevel)
	compliance.RequiredActions = actions

	return compliance
}

// isAPRARelevant determines if APRA regulations apply to this PI type
func (a *ScoreAggregator) isAPRARelevant(piType detection.PIType) bool {
	if !a.config.APRACompliance {
		return false
	}

	// APRA regulates banking and financial services, including data that could impact financial services
	bankingPITypes := map[detection.PIType]bool{
		detection.PITypeTFN:        true, // Tax File Number - financial identity
		detection.PITypeMedicare:   true, // Medicare - personal identity verification for financial services
		detection.PITypeBSB:        true, // Bank State Branch
		detection.PITypeAccount:    true, // Account numbers
		detection.PITypeCreditCard: true, // Credit card data
		detection.PITypeABN:        true, // Australian Business Number (business banking)
	}

	return bankingPITypes[piType]
}

// isPrivacyActRelevant determines if Privacy Act regulations apply to this PI type
func (a *ScoreAggregator) isPrivacyActRelevant(piType detection.PIType) bool {
	if !a.config.PrivacyActCompliance {
		return false
	}

	// Privacy Act applies to most personal information
	nonPersonalTypes := map[detection.PIType]bool{
		detection.PITypeIP:  true, // IP addresses may not be personal in all contexts
		detection.PITypeABN: true, // Business numbers are not personal
	}

	return !nonPersonalTypes[piType]
}

// generateComplianceActions generates required compliance actions
func (a *ScoreAggregator) generateComplianceActions(piType detection.PIType, riskLevel RiskLevel) []ComplianceAction {
	actions := []ComplianceAction{}
	now := time.Now()

	// Critical and High risk PI requires immediate action
	if riskLevel == RiskLevelCritical || riskLevel == RiskLevelHigh {
		// Immediate notification action
		actions = append(actions, ComplianceAction{
			Type:        "IMMEDIATE_NOTIFICATION",
			Description: "Immediately notify data protection officer and security team",
			Priority:    "CRITICAL",
			Deadline:    now.Add(1 * time.Hour),
		})

		// Risk assessment action
		actions = append(actions, ComplianceAction{
			Type:        "RISK_ASSESSMENT",
			Description: "Conduct comprehensive risk assessment and impact analysis",
			Priority:    "HIGH",
			Deadline:    now.Add(24 * time.Hour),
		})
	}

	// Australian banking regulation actions
	if a.isAPRARelevant(piType) && riskLevel != RiskLevelLow {
		actions = append(actions, ComplianceAction{
			Type:        "BANKING_REGULATION",
			Description: "Review against Australian banking regulations (APRA CPS 234) and implement required security controls",
			Priority:    "HIGH",
			Deadline:    now.Add(72 * time.Hour),
		})
	}

	// Healthcare regulation actions for Medicare
	if piType == detection.PITypeMedicare && riskLevel != RiskLevelLow {
		actions = append(actions, ComplianceAction{
			Type:        "HEALTHCARE_REGULATION",
			Description: "Review against Australian healthcare privacy requirements and Medicare compliance",
			Priority:    "HIGH",
			Deadline:    now.Add(48 * time.Hour),
		})
	}

	// Privacy Act compliance actions
	if a.isPrivacyActRelevant(piType) && (riskLevel == RiskLevelCritical || riskLevel == RiskLevelHigh) {
		actions = append(actions, ComplianceAction{
			Type:        "PRIVACY_ACT_COMPLIANCE",
			Description: "Ensure compliance with Australian Privacy Act 1988 and notifiable data breach scheme",
			Priority:    "HIGH",
			Deadline:    now.Add(30 * 24 * time.Hour), // 30 days
		})
	}

	// Data remediation for all non-low risk findings
	if riskLevel != RiskLevelLow {
		actions = append(actions, ComplianceAction{
			Type:        "DATA_REMEDIATION",
			Description: "Remove or secure exposed personal information and implement access controls",
			Priority:    "MEDIUM",
			Deadline:    now.Add(7 * 24 * time.Hour), // 7 days
		})
	}

	return actions
}

// Helper methods for audit trail details

func (a *ScoreAggregator) getValidationAlgorithm(piType detection.PIType) string {
	algorithms := map[detection.PIType]string{
		detection.PITypeTFN:      "TFN_CHECKSUM",
		detection.PITypeABN:      "ABN_MODULUS_89",
		detection.PITypeMedicare: "MEDICARE_CHECKSUM",
		detection.PITypeBSB:      "BSB_FORMAT",
	}

	if algo, exists := algorithms[piType]; exists {
		return algo
	}
	return "FORMAT_VALIDATION"
}

func (a *ScoreAggregator) getEnvironmentImpactDescription(score float64) string {
	if score < 0.5 {
		return "significant_penalty_test_environment"
	} else if score < 1.0 {
		return "moderate_penalty_non_production"
	} else if score > 1.0 {
		return "production_environment_boost"
	}
	return "neutral_environment"
}

func (a *ScoreAggregator) getCoOccurrenceImpactDescription(score float64) string {
	if score > 1.2 {
		return "high_risk_pi_combination"
	} else if score > 1.0 {
		return "moderate_pi_combination"
	}
	return "no_significant_co_occurrence"
}

func (a *ScoreAggregator) getAggregationMethod() string {
	if a.config.WeightedCombination {
		return "weighted_combination"
	}
	return "simple_average"
}

func (a *ScoreAggregator) getRegulatoryComplianceStatus(piType detection.PIType) string {
	status := []string{}

	if a.isAPRARelevant(piType) {
		status = append(status, "APRA")
	}
	if a.isPrivacyActRelevant(piType) {
		status = append(status, "Privacy_Act")
	}

	if len(status) == 0 {
		return "minimal_regulatory_impact"
	}

	return fmt.Sprintf("applicable: %v", status)
}
