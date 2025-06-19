package scoring

import (
	"github.com/MacAttak/pi-scanner/pkg/detection"
)

// ImpactCalculator calculates the potential impact of a PI exposure
type ImpactCalculator struct {
	config *RiskMatrixConfig
}

// NewImpactCalculator creates a new impact calculator
func NewImpactCalculator(config *RiskMatrixConfig) *ImpactCalculator {
	return &ImpactCalculator{
		config: config,
	}
}

// Calculate computes the impact score and factors
func (ic *ImpactCalculator) Calculate(input RiskAssessmentInput) (float64, ImpactFactors) {
	factors := ImpactFactors{}
	
	// Calculate data sensitivity based on PI type
	factors.DataSensitivity = ic.calculateDataSensitivity(input.Finding.Type)
	
	// Estimate record count impact
	factors.RecordCount = ic.estimateRecordCount(input)
	
	// Calculate financial impact
	factors.FinancialImpact = ic.calculateFinancialImpact(input.Finding.Type, factors.RecordCount)
	
	// Calculate regulatory impact
	factors.RegulatoryImpact = ic.calculateRegulatoryImpact(input.Finding.Type, input.OrganizationInfo)
	
	// Calculate reputational impact
	factors.ReputationalImpact = ic.calculateReputationalImpact(input)
	
	// Estimate affected individuals
	factors.AffectedIndividuals = ic.estimateAffectedIndividuals(input, factors.RecordCount)
	
	// Combine factors into overall impact score
	impactScore := ic.combineImpactFactors(factors)
	
	return impactScore, factors
}

// calculateDataSensitivity determines sensitivity level of PI type
func (ic *ImpactCalculator) calculateDataSensitivity(piType detection.PIType) float64 {
	// Australian PI sensitivity levels aligned with Privacy Act
	sensitivityLevels := map[detection.PIType]float64{
		detection.PITypeTFN:           1.0,  // Highest - tax file number
		detection.PITypeMedicare:      0.95, // Very high - health identifier
		detection.PITypeCreditCard:    0.9,  // Very high - financial data
		detection.PITypePassport:      0.9,  // Very high - identity document
		detection.PITypeDriverLicense: 0.85, // High - identity document
		detection.PITypeABN:           0.6,  // Medium - business identifier
		detection.PITypeBSB:           0.7,  // High - banking identifier
		detection.PITypeAccount:       0.8,  // High - financial account
		detection.PITypeName:          0.5,  // Medium - personal identifier
		detection.PITypeAddress:       0.5,  // Medium - personal identifier
		detection.PITypePhone:         0.4,  // Medium-low - contact info
		detection.PITypeEmail:         0.3,  // Low - contact info
		detection.PITypeIP:            0.2,  // Low - technical identifier
	}
	
	if sensitivity, exists := sensitivityLevels[piType]; exists {
		return sensitivity
	}
	
	return 0.5 // Default medium sensitivity
}

// estimateRecordCount estimates the number of records exposed
func (ic *ImpactCalculator) estimateRecordCount(input RiskAssessmentInput) int {
	baseCount := 1 // At least one record
	
	// Check for bulk exposure patterns
	if input.FileContext.IsConfiguration {
		baseCount = 10 // Config files often contain multiple records
	}
	
	if input.FileContext.FileSize > 1024*1024 { // > 1MB
		baseCount = 100 // Large files likely contain many records
	}
	
	// Multiply by co-occurrences (likely same individuals)
	if len(input.CoOccurrences) > 0 {
		baseCount *= len(input.CoOccurrences)
	}
	
	return baseCount
}

// calculateFinancialImpact estimates potential financial impact
func (ic *ImpactCalculator) calculateFinancialImpact(piType detection.PIType, recordCount int) float64 {
	// Base financial impact per record (in relative terms)
	financialImpactPerRecord := map[detection.PIType]float64{
		detection.PITypeCreditCard: 1.0,  // Highest - direct financial access
		detection.PITypeBSB:        0.8,  // High - banking details
		detection.PITypeAccount:    0.8,  // High - account access
		detection.PITypeTFN:        0.7,  // High - tax/identity fraud
		detection.PITypeMedicare:   0.5,  // Medium - healthcare fraud
		detection.PITypePassport:   0.6,  // Medium-high - identity fraud
		detection.PITypeABN:        0.3,  // Low-medium - business impact
		detection.PITypeName:       0.2,  // Low - requires additional info
		detection.PITypeEmail:      0.1,  // Very low - spam/phishing
	}
	
	baseImpact := 0.3 // Default
	if impact, exists := financialImpactPerRecord[piType]; exists {
		baseImpact = impact
	}
	
	// Scale by record count (with diminishing returns)
	scaleFactor := 1.0
	if recordCount > 10 {
		scaleFactor = 1.2
	}
	if recordCount > 100 {
		scaleFactor = 1.5
	}
	if recordCount > 1000 {
		scaleFactor = 2.0
	}
	
	return ic.normalizeScore(baseImpact * scaleFactor)
}

// calculateRegulatoryImpact assesses regulatory compliance impact
func (ic *ImpactCalculator) calculateRegulatoryImpact(piType detection.PIType, orgInfo OrganizationInfo) float64 {
	baseImpact := 0.5
	
	// Higher impact for regulated industries
	if orgInfo.Regulated {
		baseImpact = 0.8
	}
	
	// Australian regulatory requirements
	if ic.config.APRAAligned {
		apraRelevantTypes := map[detection.PIType]bool{
			detection.PITypeTFN:        true,
			detection.PITypeBSB:        true,
			detection.PITypeAccount:    true,
			detection.PITypeCreditCard: true,
		}
		
		if apraRelevantTypes[piType] {
			baseImpact = ic.maxFloat(baseImpact, 0.9)
		}
	}
	
	if ic.config.PrivacyActAligned {
		privacyActTypes := map[detection.PIType]bool{
			detection.PITypeTFN:           true,
			detection.PITypeMedicare:      true,
			detection.PITypeDriverLicense: true,
			detection.PITypePassport:      true,
		}
		
		if privacyActTypes[piType] {
			baseImpact = ic.maxFloat(baseImpact, 0.85)
		}
	}
	
	// Industry-specific impacts
	industryMultipliers := map[string]float64{
		"banking":    1.3,
		"finance":    1.3,
		"healthcare": 1.2,
		"government": 1.2,
		"insurance":  1.1,
		"retail":     1.0,
		"technology": 0.9,
	}
	
	if multiplier, exists := industryMultipliers[orgInfo.Industry]; exists {
		baseImpact *= multiplier
	}
	
	return ic.normalizeScore(baseImpact)
}

// calculateReputationalImpact estimates reputational damage
func (ic *ImpactCalculator) calculateReputationalImpact(input RiskAssessmentInput) float64 {
	baseImpact := 0.5
	
	// Public repositories have higher reputational impact
	if input.RepositoryInfo.IsPublic {
		baseImpact *= ic.config.PublicRepoMultiplier
	}
	
	// Popular repositories have higher impact
	if input.RepositoryInfo.Stars > 100 {
		baseImpact *= 1.2
	}
	if input.RepositoryInfo.Stars > 1000 {
		baseImpact *= 1.5
	}
	
	// Historical incidents increase impact
	if input.HistoricalData.PreviousIncidents > 0 {
		baseImpact *= 1.0 + (0.1 * float64(input.HistoricalData.PreviousIncidents))
	}
	
	// Sensitive PI types have higher reputational impact
	sensitiveTypes := map[detection.PIType]bool{
		detection.PITypeTFN:        true,
		detection.PITypeMedicare:   true,
		detection.PITypeCreditCard: true,
		detection.PITypePassport:   true,
	}
	
	if sensitiveTypes[input.Finding.Type] {
		baseImpact *= 1.3
	}
	
	return ic.normalizeScore(baseImpact)
}

// estimateAffectedIndividuals estimates number of affected individuals
func (ic *ImpactCalculator) estimateAffectedIndividuals(input RiskAssessmentInput, recordCount int) int {
	// Base estimate on record count
	affected := recordCount
	
	// Adjust based on context
	if input.FileContext.IsTest {
		affected = 0 // Test data doesn't affect real individuals
	}
	
	if input.FileContext.IsProduction {
		affected = int(float64(affected) * ic.config.ProductionMultiplier)
	}
	
	// Consider repository reach
	if input.RepositoryInfo.IsPublic {
		// Public repos might have wider exposure
		reachMultiplier := 1 + (input.RepositoryInfo.Forks / 100)
		if reachMultiplier > 10 {
			reachMultiplier = 10 // Cap multiplier
		}
		affected *= reachMultiplier
	}
	
	return affected
}

// combineImpactFactors combines all factors into overall impact score
func (ic *ImpactCalculator) combineImpactFactors(factors ImpactFactors) float64 {
	// Weighted combination of factors
	weights := map[string]float64{
		"data_sensitivity":   0.25,
		"financial_impact":   0.25,
		"regulatory_impact":  0.25,
		"reputational_impact": 0.15,
		"affected_scale":     0.10,
	}
	
	// Scale affected individuals to 0-1 range
	affectedScale := 0.0
	if factors.AffectedIndividuals > 0 {
		if factors.AffectedIndividuals <= 10 {
			affectedScale = 0.3
		} else if factors.AffectedIndividuals <= 100 {
			affectedScale = 0.6
		} else if factors.AffectedIndividuals <= 1000 {
			affectedScale = 0.8
		} else {
			affectedScale = 1.0
		}
	}
	
	impactScore := factors.DataSensitivity*weights["data_sensitivity"] +
		factors.FinancialImpact*weights["financial_impact"] +
		factors.RegulatoryImpact*weights["regulatory_impact"] +
		factors.ReputationalImpact*weights["reputational_impact"] +
		affectedScale*weights["affected_scale"]
	
	return ic.normalizeScore(impactScore)
}

// Helper methods

func (ic *ImpactCalculator) normalizeScore(score float64) float64 {
	if score < 0 {
		return 0
	}
	if score > 1 {
		return 1
	}
	return score
}

func (ic *ImpactCalculator) maxFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}