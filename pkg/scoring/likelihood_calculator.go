package scoring

import (
	"strings"
	"time"

	"github.com/MacAttak/pi-scanner/pkg/detection"
)

// LikelihoodCalculator calculates the likelihood of PI exploitation
type LikelihoodCalculator struct {
	config *RiskMatrixConfig
}

// NewLikelihoodCalculator creates a new likelihood calculator
func NewLikelihoodCalculator(config *RiskMatrixConfig) *LikelihoodCalculator {
	return &LikelihoodCalculator{
		config: config,
	}
}

// Calculate computes the likelihood score and factors
func (lc *LikelihoodCalculator) Calculate(input RiskAssessmentInput) (float64, LikelihoodFactors) {
	factors := LikelihoodFactors{}

	// Calculate exploit complexity
	factors.ExploitComplexity = lc.calculateExploitComplexity(input)

	// Determine access vector
	factors.AccessVector = lc.determineAccessVector(input)

	// Determine authentication requirements
	factors.Authentication = lc.determineAuthentication(input)

	// Count historical incidents
	factors.HistoricalIncidents = input.HistoricalData.PreviousIncidents

	// Assess threat actor capability
	factors.ThreatActorCapability = lc.assessThreatActorCapability(input)

	// Combine factors into overall likelihood score
	likelihoodScore := lc.combineLikelihoodFactors(factors, input)

	return likelihoodScore, factors
}

// calculateExploitComplexity determines how complex it is to exploit the exposure
func (lc *LikelihoodCalculator) calculateExploitComplexity(input RiskAssessmentInput) float64 {
	// Start with base complexity
	complexity := 0.5

	// Public repositories are easier to discover
	if input.RepositoryInfo.IsPublic {
		complexity -= 0.3
	}

	// Test files are less likely to contain real data
	if input.FileContext.IsTest {
		complexity += 0.3
	}

	// Configuration files are often targeted
	if input.FileContext.IsConfiguration {
		complexity -= 0.2
	}

	// Source code files require more analysis
	if input.FileContext.IsSource && !input.FileContext.IsConfiguration {
		complexity += 0.1
	}

	// Well-known file paths are easier to find
	knownPaths := []string{
		"config", "conf", "settings", "env", ".env",
		"credentials", "secrets", "keys", "auth",
	}

	filePath := strings.ToLower(input.FileContext.FilePath)
	for _, known := range knownPaths {
		if strings.Contains(filePath, known) {
			complexity -= 0.1
			break
		}
	}

	// Validated PI is easier to exploit (confirmed real)
	if input.Finding.Validated {
		complexity -= 0.2
	}

	// Multiple co-occurrences make it easier to correlate data
	if len(input.CoOccurrences) > 2 {
		complexity -= 0.1
	}

	// Normalize (lower complexity = higher likelihood)
	return lc.normalizeScore(1.0 - complexity)
}

// determineAccessVector categorizes how the PI can be accessed
func (lc *LikelihoodCalculator) determineAccessVector(input RiskAssessmentInput) string {
	if input.RepositoryInfo.IsPublic {
		return "PUBLIC_NETWORK"
	}

	if input.OrganizationInfo.HasSecurityTeam {
		return "INTERNAL_RESTRICTED"
	}

	return "INTERNAL_NETWORK"
}

// determineAuthentication categorizes authentication requirements
func (lc *LikelihoodCalculator) determineAuthentication(input RiskAssessmentInput) string {
	// Public repos require no authentication
	if input.RepositoryInfo.IsPublic {
		return "NONE"
	}

	// Check for CI/CD (usually has elevated permissions)
	if input.RepositoryInfo.HasCICD {
		return "SINGLE_FACTOR"
	}

	// Regulated organizations likely have better controls
	if input.OrganizationInfo.Regulated {
		return "MULTI_FACTOR"
	}

	return "SINGLE_FACTOR"
}

// assessThreatActorCapability estimates the capability level of potential threat actors
func (lc *LikelihoodCalculator) assessThreatActorCapability(input RiskAssessmentInput) float64 {
	capability := 0.5 // Base capability

	// High-value targets attract more capable actors
	highValueTypes := map[detection.PIType]bool{
		detection.PITypeTFN:        true,
		detection.PITypeCreditCard: true,
		detection.PITypeMedicare:   true,
		detection.PITypePassport:   true,
	}

	if highValueTypes[input.Finding.Type] {
		capability += 0.2
	}

	// Popular repositories attract more attention
	if input.RepositoryInfo.Stars > 1000 {
		capability += 0.1
	}

	// Regulated industries are targeted by sophisticated actors
	if input.OrganizationInfo.Regulated {
		capability += 0.2
	}

	// Recent activity suggests active targeting
	if input.HistoricalData.LastIncidentDate.After(time.Now().AddDate(0, -6, 0)) {
		capability += 0.1
	}

	return lc.normalizeScore(capability)
}

// combineLikelihoodFactors combines all factors into overall likelihood score
func (lc *LikelihoodCalculator) combineLikelihoodFactors(factors LikelihoodFactors, input RiskAssessmentInput) float64 {
	// Base likelihood on exploit complexity
	baseLikelihood := factors.ExploitComplexity

	// Adjust based on access vector
	accessMultipliers := map[string]float64{
		"PUBLIC_NETWORK":      1.5,
		"INTERNAL_NETWORK":    1.0,
		"INTERNAL_RESTRICTED": 0.7,
		"LOCAL":               0.5,
	}

	if multiplier, exists := accessMultipliers[factors.AccessVector]; exists {
		baseLikelihood *= multiplier
	}

	// Adjust based on authentication
	authMultipliers := map[string]float64{
		"NONE":          1.3,
		"SINGLE_FACTOR": 1.0,
		"MULTI_FACTOR":  0.6,
	}

	if multiplier, exists := authMultipliers[factors.Authentication]; exists {
		baseLikelihood *= multiplier
	}

	// Historical incidents increase likelihood
	if factors.HistoricalIncidents > 0 {
		incidentMultiplier := 1.0 + (0.15 * float64(factors.HistoricalIncidents))
		if incidentMultiplier > 2.0 {
			incidentMultiplier = 2.0 // Cap the multiplier
		}
		baseLikelihood *= incidentMultiplier
	}

	// Threat actor capability affects likelihood
	baseLikelihood *= (0.5 + factors.ThreatActorCapability*0.5)

	// Time-based factors
	timeSinceLastCommit := time.Since(input.RepositoryInfo.LastCommit)
	if timeSinceLastCommit > 365*24*time.Hour {
		// Abandoned repos are less likely to be monitored
		baseLikelihood *= 1.2
	}

	// Environmental adjustments
	if input.FileContext.IsProduction {
		baseLikelihood *= lc.config.ProductionMultiplier
	}

	// Sensitive paths increase likelihood
	sensitivePaths := []string{
		"prod", "production", "live",
		"payment", "billing", "financial",
		"customer", "user", "personal",
	}

	filePath := strings.ToLower(input.FileContext.FilePath)
	for _, sensitive := range sensitivePaths {
		if strings.Contains(filePath, sensitive) {
			baseLikelihood *= (1.0 + lc.config.SensitivePathBonus)
			break
		}
	}

	return lc.normalizeScore(baseLikelihood)
}

// normalizeScore ensures score is within 0-1 range
func (lc *LikelihoodCalculator) normalizeScore(score float64) float64 {
	if score < 0 {
		return 0
	}
	if score > 1 {
		return 1
	}
	return score
}
