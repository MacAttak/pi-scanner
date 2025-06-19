package scoring

import (
	"fmt"
	"strings"
	"time"

	"github.com/MacAttak/pi-scanner/pkg/detection"
)

// RiskMatrix provides a multi-dimensional risk assessment framework
// aligned with Australian banking and privacy regulations
type RiskMatrix struct {
	config *RiskMatrixConfig
	
	// Risk dimension calculators
	impactCalculator     *ImpactCalculator
	likelihoodCalculator *LikelihoodCalculator
	exposureCalculator   *ExposureCalculator
}

// RiskMatrixConfig holds configuration for risk matrix calculations
type RiskMatrixConfig struct {
	// Risk calculation method
	UseMultiplicativeModel bool    `json:"use_multiplicative_model"`
	ImpactWeight          float64 `json:"impact_weight"`
	LikelihoodWeight      float64 `json:"likelihood_weight"`
	ExposureWeight        float64 `json:"exposure_weight"`
	
	// Australian regulatory alignments
	APRAAligned        bool `json:"apra_aligned"`
	PrivacyActAligned  bool `json:"privacy_act_aligned"`
	
	// Risk thresholds
	CriticalThreshold float64 `json:"critical_threshold"` // 0.8+
	HighThreshold     float64 `json:"high_threshold"`     // 0.6-0.79
	MediumThreshold   float64 `json:"medium_threshold"`   // 0.4-0.59
	LowThreshold      float64 `json:"low_threshold"`      // 0.2-0.39
	
	// Context modifiers
	ProductionMultiplier float64 `json:"production_multiplier"`
	PublicRepoMultiplier float64 `json:"public_repo_multiplier"`
	SensitivePathBonus   float64 `json:"sensitive_path_bonus"`
}

// DefaultRiskMatrixConfig returns the default risk matrix configuration
func DefaultRiskMatrixConfig() *RiskMatrixConfig {
	return &RiskMatrixConfig{
		UseMultiplicativeModel: true,
		ImpactWeight:          0.4,
		LikelihoodWeight:      0.3,
		ExposureWeight:        0.3,
		APRAAligned:           true,
		PrivacyActAligned:     true,
		CriticalThreshold:     0.8,
		HighThreshold:         0.6,
		MediumThreshold:       0.4,
		LowThreshold:          0.2,
		ProductionMultiplier:  1.5,
		PublicRepoMultiplier:  1.3,
		SensitivePathBonus:    0.2,
	}
}

// RiskAssessment represents a comprehensive risk assessment result
type RiskAssessment struct {
	// Core risk scores
	OverallRisk      float64 `json:"overall_risk"`
	ImpactScore      float64 `json:"impact_score"`
	LikelihoodScore  float64 `json:"likelihood_score"`
	ExposureScore    float64 `json:"exposure_score"`
	
	// Risk level and category
	RiskLevel        RiskLevel    `json:"risk_level"`
	RiskCategory     RiskCategory `json:"risk_category"`
	
	// Detailed breakdown
	ImpactFactors     ImpactFactors     `json:"impact_factors"`
	LikelihoodFactors LikelihoodFactors `json:"likelihood_factors"`
	ExposureFactors   ExposureFactors   `json:"exposure_factors"`
	
	// Mitigation recommendations
	Mitigations      []Mitigation      `json:"mitigations"`
	ComplianceFlags  ComplianceFlags   `json:"compliance_flags"`
	
	// Metadata
	AssessedAt       time.Time         `json:"assessed_at"`
	AssessmentID     string            `json:"assessment_id"`
}

// RiskCategory represents the category of risk
type RiskCategory string

const (
	RiskCategoryIdentityTheft      RiskCategory = "IDENTITY_THEFT"
	RiskCategoryFinancialFraud     RiskCategory = "FINANCIAL_FRAUD"
	RiskCategoryPrivacyBreach      RiskCategory = "PRIVACY_BREACH"
	RiskCategoryRegulatoryBreach   RiskCategory = "REGULATORY_BREACH"
	RiskCategoryReputationalDamage RiskCategory = "REPUTATIONAL_DAMAGE"
	RiskCategoryOperational        RiskCategory = "OPERATIONAL"
)

// ImpactFactors represents factors contributing to impact assessment
type ImpactFactors struct {
	DataSensitivity       float64 `json:"data_sensitivity"`
	RecordCount          int     `json:"record_count"`
	FinancialImpact      float64 `json:"financial_impact"`
	RegulatoryImpact     float64 `json:"regulatory_impact"`
	ReputationalImpact   float64 `json:"reputational_impact"`
	AffectedIndividuals  int     `json:"affected_individuals"`
}

// LikelihoodFactors represents factors contributing to likelihood assessment
type LikelihoodFactors struct {
	ExploitComplexity    float64 `json:"exploit_complexity"`
	AccessVector         string  `json:"access_vector"`
	Authentication       string  `json:"authentication"`
	HistoricalIncidents  int     `json:"historical_incidents"`
	ThreatActorCapability float64 `json:"threat_actor_capability"`
}

// ExposureFactors represents factors contributing to exposure assessment
type ExposureFactors struct {
	RepositoryVisibility string  `json:"repository_visibility"`
	FileAccessibility    float64 `json:"file_accessibility"`
	DataLifetime        int     `json:"data_lifetime_days"`
	EncryptionStatus    string  `json:"encryption_status"`
	AccessControls      float64 `json:"access_controls"`
}

// Mitigation represents a recommended mitigation action
type Mitigation struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Priority    string    `json:"priority"`
	Effort      string    `json:"effort"`
	Timeline    string    `json:"timeline"`
	Category    string    `json:"category"`
	Compliance  []string  `json:"compliance"`
}

// ComplianceFlags indicates regulatory compliance requirements
type ComplianceFlags struct {
	NotifiableDataBreach bool     `json:"notifiable_data_breach"`
	APRAReporting       bool     `json:"apra_reporting"`
	PrivacyActBreach    bool     `json:"privacy_act_breach"`
	GDPRApplicable      bool     `json:"gdpr_applicable"`
	RequiredNotifications []string `json:"required_notifications"`
}

// NewRiskMatrix creates a new risk matrix calculator
func NewRiskMatrix(config *RiskMatrixConfig) (*RiskMatrix, error) {
	if config == nil {
		config = DefaultRiskMatrixConfig()
	}
	
	return &RiskMatrix{
		config:               config,
		impactCalculator:     NewImpactCalculator(config),
		likelihoodCalculator: NewLikelihoodCalculator(config),
		exposureCalculator:   NewExposureCalculator(config),
	}, nil
}

// AssessRisk performs a comprehensive risk assessment
func (rm *RiskMatrix) AssessRisk(input RiskAssessmentInput) (*RiskAssessment, error) {
	if err := rm.validateInput(input); err != nil {
		return nil, fmt.Errorf("invalid risk assessment input: %w", err)
	}
	
	// Calculate individual risk dimensions
	impactScore, impactFactors := rm.impactCalculator.Calculate(input)
	likelihoodScore, likelihoodFactors := rm.likelihoodCalculator.Calculate(input)
	exposureScore, exposureFactors := rm.exposureCalculator.Calculate(input)
	
	// Calculate overall risk score
	overallRisk := rm.calculateOverallRisk(impactScore, likelihoodScore, exposureScore)
	
	// Determine risk level and category
	riskLevel := rm.mapScoreToRiskLevel(overallRisk)
	riskCategory := rm.determineRiskCategory(input, impactFactors)
	
	// Generate mitigations
	mitigations := rm.generateMitigations(input, riskLevel, riskCategory)
	
	// Determine compliance flags
	complianceFlags := rm.determineComplianceFlags(input, riskLevel)
	
	return &RiskAssessment{
		OverallRisk:       overallRisk,
		ImpactScore:       impactScore,
		LikelihoodScore:   likelihoodScore,
		ExposureScore:     exposureScore,
		RiskLevel:         riskLevel,
		RiskCategory:      riskCategory,
		ImpactFactors:     impactFactors,
		LikelihoodFactors: likelihoodFactors,
		ExposureFactors:   exposureFactors,
		Mitigations:       mitigations,
		ComplianceFlags:   complianceFlags,
		AssessedAt:        time.Now(),
		AssessmentID:      rm.generateAssessmentID(),
	}, nil
}

// RiskAssessmentInput contains all input data for risk assessment
type RiskAssessmentInput struct {
	// Finding details
	Finding         detection.Finding `json:"finding"`
	ConfidenceScore float64          `json:"confidence_score"`
	
	// Context information
	RepositoryInfo  RepositoryInfo   `json:"repository_info"`
	FileContext     FileContext      `json:"file_context"`
	OrganizationInfo OrganizationInfo `json:"organization_info"`
	
	// Additional context
	CoOccurrences   []detection.Finding `json:"co_occurrences"`
	HistoricalData  HistoricalData      `json:"historical_data"`
}

// RepositoryInfo contains repository context information
type RepositoryInfo struct {
	IsPublic        bool   `json:"is_public"`
	Stars           int    `json:"stars"`
	Forks           int    `json:"forks"`
	Contributors    int    `json:"contributors"`
	LastCommit      time.Time `json:"last_commit"`
	DefaultBranch   string `json:"default_branch"`
	HasCICD         bool   `json:"has_cicd"`
}

// FileContext contains file-specific context
type FileContext struct {
	FilePath        string `json:"file_path"`
	FileSize        int64  `json:"file_size"`
	IsProduction    bool   `json:"is_production"`
	IsTest          bool   `json:"is_test"`
	IsConfiguration bool   `json:"is_configuration"`
	IsSource        bool   `json:"is_source"`
	Language        string `json:"language"`
}

// OrganizationInfo contains organization context
type OrganizationInfo struct {
	Industry        string `json:"industry"`
	Size            string `json:"size"`
	Regulated       bool   `json:"regulated"`
	HasSecurityTeam bool   `json:"has_security_team"`
	MaturityLevel   string `json:"maturity_level"`
}

// HistoricalData contains historical incident information
type HistoricalData struct {
	PreviousIncidents   int       `json:"previous_incidents"`
	LastIncidentDate    time.Time `json:"last_incident_date"`
	RemediationTime     int       `json:"avg_remediation_days"`
	FalsePositiveRate   float64   `json:"false_positive_rate"`
}

// calculateOverallRisk combines individual risk scores
func (rm *RiskMatrix) calculateOverallRisk(impact, likelihood, exposure float64) float64 {
	if rm.config.UseMultiplicativeModel {
		// Multiplicative model emphasizes compounding risks
		baseRisk := impact * likelihood * exposure
		
		// Apply weights to balance the multiplication
		weightedRisk := baseRisk * 
			(rm.config.ImpactWeight + rm.config.LikelihoodWeight + rm.config.ExposureWeight)
		
		// Normalize to 0-1 range
		return rm.normalizeScore(weightedRisk)
	}
	
	// Weighted average model
	return impact*rm.config.ImpactWeight +
		likelihood*rm.config.LikelihoodWeight +
		exposure*rm.config.ExposureWeight
}

// mapScoreToRiskLevel maps risk score to risk level
func (rm *RiskMatrix) mapScoreToRiskLevel(score float64) RiskLevel {
	if score >= rm.config.CriticalThreshold {
		return RiskLevelCritical
	} else if score >= rm.config.HighThreshold {
		return RiskLevelHigh
	} else if score >= rm.config.MediumThreshold {
		return RiskLevelMedium
	} else if score >= rm.config.LowThreshold {
		return RiskLevelLow
	}
	return RiskLevelLow
}

// determineRiskCategory determines the primary risk category
func (rm *RiskMatrix) determineRiskCategory(input RiskAssessmentInput, impact ImpactFactors) RiskCategory {
	piType := input.Finding.Type
	
	// Financial PI types
	financialTypes := map[detection.PIType]bool{
		detection.PITypeTFN:        true,
		detection.PITypeCreditCard: true,
		detection.PITypeBSB:        true,
		detection.PITypeAccount:    true,
	}
	
	// Identity PI types
	identityTypes := map[detection.PIType]bool{
		detection.PITypeTFN:           true,
		detection.PITypeMedicare:      true,
		detection.PITypePassport:      true,
		detection.PITypeDriverLicense: true,
	}
	
	// Check co-occurrences for financial fraud
	hasFinancialCoOccurrence := false
	for _, coOcc := range input.CoOccurrences {
		if financialTypes[coOcc.Type] {
			hasFinancialCoOccurrence = true
			break
		}
	}
	
	// Check for financial fraud risk (TFN + financial data)
	if financialTypes[piType] && (hasFinancialCoOccurrence || impact.FinancialImpact > 0.7) {
		return RiskCategoryFinancialFraud
	}
	
	// Check for identity theft risk (multiple identity elements)
	if identityTypes[piType] && (len(input.CoOccurrences) >= 1 || input.RepositoryInfo.IsPublic) {
		return RiskCategoryIdentityTheft
	}
	
	// For test files, default to operational
	if input.FileContext.IsTest {
		return RiskCategoryOperational
	}
	
	// Check for regulatory breach (regulated industries)
	if input.OrganizationInfo.Regulated && impact.RegulatoryImpact > 0.6 {
		return RiskCategoryRegulatoryBreach
	}
	
	// Check for privacy breach
	if impact.AffectedIndividuals > 100 {
		return RiskCategoryPrivacyBreach
	}
	
	// Default to operational risk
	return RiskCategoryOperational
}

// generateMitigations creates specific mitigation recommendations
func (rm *RiskMatrix) generateMitigations(input RiskAssessmentInput, level RiskLevel, category RiskCategory) []Mitigation {
	mitigations := []Mitigation{}
	
	// Immediate actions for critical/high risk
	if level == RiskLevelCritical || level == RiskLevelHigh {
		mitigations = append(mitigations, Mitigation{
			ID:          "MIT-001",
			Title:       "Immediate Data Removal",
			Description: "Remove or redact the exposed personal information immediately",
			Priority:    "CRITICAL",
			Effort:      "LOW",
			Timeline:    "Within 2 hours",
			Category:    "REMEDIATION",
			Compliance:  []string{"APRA", "Privacy Act"},
		})
		
		mitigations = append(mitigations, Mitigation{
			ID:          "MIT-002",
			Title:       "Access Control Implementation",
			Description: "Implement strict access controls on affected files and repositories",
			Priority:    "HIGH",
			Effort:      "MEDIUM",
			Timeline:    "Within 24 hours",
			Category:    "PREVENTION",
			Compliance:  []string{"APRA CPS 234"},
		})
	}
	
	// Category-specific mitigations
	switch category {
	case RiskCategoryFinancialFraud:
		mitigations = append(mitigations, Mitigation{
			ID:          "MIT-FIN-001",
			Title:       "Financial Monitoring",
			Description: "Implement monitoring for potential fraudulent transactions",
			Priority:    "HIGH",
			Effort:      "HIGH",
			Timeline:    "Within 72 hours",
			Category:    "DETECTION",
			Compliance:  []string{"APRA", "Banking Code"},
		})
		
	case RiskCategoryIdentityTheft:
		mitigations = append(mitigations, Mitigation{
			ID:          "MIT-ID-001",
			Title:       "Identity Protection Services",
			Description: "Offer identity protection services to affected individuals",
			Priority:    "HIGH",
			Effort:      "MEDIUM",
			Timeline:    "Within 1 week",
			Category:    "RESPONSE",
			Compliance:  []string{"Privacy Act", "Consumer Protection"},
		})
		
	case RiskCategoryRegulatoryBreach:
		mitigations = append(mitigations, Mitigation{
			ID:          "MIT-REG-001",
			Title:       "Regulatory Notification",
			Description: "Notify relevant regulatory bodies as required",
			Priority:    "CRITICAL",
			Effort:      "MEDIUM",
			Timeline:    "Within 72 hours",
			Category:    "COMPLIANCE",
			Compliance:  []string{"Notifiable Data Breach Scheme"},
		})
	}
	
	// General security improvements
	mitigations = append(mitigations, Mitigation{
		ID:          "MIT-GEN-001",
		Title:       "Security Training",
		Description: "Conduct security awareness training focusing on data protection",
		Priority:    "MEDIUM",
		Effort:      "MEDIUM",
		Timeline:    "Within 30 days",
		Category:    "PREVENTION",
		Compliance:  []string{"APRA CPS 234", "ISO 27001"},
	})
	
	// Code scanning improvements
	mitigations = append(mitigations, Mitigation{
		ID:          "MIT-GEN-002",
		Title:       "Automated Scanning",
		Description: "Implement automated PI scanning in CI/CD pipeline",
		Priority:    "MEDIUM",
		Effort:      "HIGH",
		Timeline:    "Within 60 days",
		Category:    "PREVENTION",
		Compliance:  []string{"DevSecOps Best Practice"},
	})
	
	return mitigations
}

// determineComplianceFlags determines applicable compliance requirements
func (rm *RiskMatrix) determineComplianceFlags(input RiskAssessmentInput, level RiskLevel) ComplianceFlags {
	flags := ComplianceFlags{
		RequiredNotifications: []string{},
	}
	
	// Notifiable data breach scheme (Privacy Act)
	if rm.config.PrivacyActAligned && (level == RiskLevelCritical || level == RiskLevelHigh) {
		personalInfoTypes := map[detection.PIType]bool{
			detection.PITypeTFN:           true,
			detection.PITypeMedicare:      true,
			detection.PITypeDriverLicense: true,
			detection.PITypePassport:      true,
			detection.PITypeCreditCard:    true,
		}
		
		if personalInfoTypes[input.Finding.Type] {
			flags.NotifiableDataBreach = true
			flags.PrivacyActBreach = true
			flags.RequiredNotifications = append(flags.RequiredNotifications, 
				"Office of the Australian Information Commissioner (OAIC)")
		}
	}
	
	// APRA reporting requirements - expanded to include all financial services relevant PI
	if rm.config.APRAAligned {
		// APRA oversees banks, credit unions, building societies, insurance companies, and superannuation funds
		// Medicare data is relevant for insurance companies under APRA
		apraRelevantTypes := map[detection.PIType]bool{
			detection.PITypeTFN:        true,  // Financial identity
			detection.PITypeBSB:        true,  // Banking
			detection.PITypeAccount:    true,  // Banking
			detection.PITypeCreditCard: true,  // Financial services
			detection.PITypeMedicare:   true,  // Insurance/health services
			detection.PITypeABN:        true,  // Business banking
		}
		
		if apraRelevantTypes[input.Finding.Type] && level != RiskLevelLow {
			flags.APRAReporting = true
			flags.RequiredNotifications = append(flags.RequiredNotifications,
				"Australian Prudential Regulation Authority (APRA)")
		}
	}
	
	// Check for GDPR applicability (if dealing with EU residents)
	if strings.Contains(strings.ToLower(input.OrganizationInfo.Industry), "global") ||
		strings.Contains(strings.ToLower(input.OrganizationInfo.Industry), "international") {
		flags.GDPRApplicable = true
		if level == RiskLevelCritical || level == RiskLevelHigh {
			flags.RequiredNotifications = append(flags.RequiredNotifications,
				"Relevant EU Data Protection Authority (if applicable)")
		}
	}
	
	return flags
}

// validateInput validates the risk assessment input
func (rm *RiskMatrix) validateInput(input RiskAssessmentInput) error {
	if input.Finding.Type == "" {
		return fmt.Errorf("finding type is required")
	}
	
	if input.ConfidenceScore < 0 || input.ConfidenceScore > 1 {
		return fmt.Errorf("confidence score must be between 0 and 1")
	}
	
	return nil
}

// normalizeScore ensures score is within 0-1 range
func (rm *RiskMatrix) normalizeScore(score float64) float64 {
	if score < 0 {
		return 0
	}
	if score > 1 {
		return 1
	}
	return score
}

// generateAssessmentID creates a unique assessment ID
func (rm *RiskMatrix) generateAssessmentID() string {
	return fmt.Sprintf("RISK-%d-%s", time.Now().Unix(), generateRandomString(6))
}

// generateRandomString generates a random string of specified length
func generateRandomString(length int) string {
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, length)
	for i := range result {
		result[i] = charset[time.Now().UnixNano()%int64(len(charset))]
	}
	return string(result)
}