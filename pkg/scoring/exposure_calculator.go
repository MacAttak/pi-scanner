package scoring

import (
	"strings"
	"time"

	"github.com/MacAttak/pi-scanner/pkg/detection"
)

// ExposureCalculator calculates the exposure level of PI data
type ExposureCalculator struct {
	config *RiskMatrixConfig
}

// NewExposureCalculator creates a new exposure calculator
func NewExposureCalculator(config *RiskMatrixConfig) *ExposureCalculator {
	return &ExposureCalculator{
		config: config,
	}
}

// Calculate computes the exposure score and factors
func (ec *ExposureCalculator) Calculate(input RiskAssessmentInput) (float64, ExposureFactors) {
	factors := ExposureFactors{}

	// Determine repository visibility
	factors.RepositoryVisibility = ec.determineRepositoryVisibility(input)

	// Calculate file accessibility
	factors.FileAccessibility = ec.calculateFileAccessibility(input)

	// Calculate data lifetime
	factors.DataLifetime = ec.calculateDataLifetime(input)

	// Determine encryption status
	factors.EncryptionStatus = ec.determineEncryptionStatus(input)

	// Assess access controls
	factors.AccessControls = ec.assessAccessControls(input)

	// Combine factors into overall exposure score
	exposureScore := ec.combineExposureFactors(factors, input)

	return exposureScore, factors
}

// determineRepositoryVisibility categorizes repository visibility
func (ec *ExposureCalculator) determineRepositoryVisibility(input RiskAssessmentInput) string {
	if input.RepositoryInfo.IsPublic {
		if input.RepositoryInfo.Stars > 1000 {
			return "PUBLIC_HIGH_VISIBILITY"
		} else if input.RepositoryInfo.Stars > 100 {
			return "PUBLIC_MEDIUM_VISIBILITY"
		}
		return "PUBLIC_LOW_VISIBILITY"
	}

	// Private repository visibility depends on organization
	if input.OrganizationInfo.Size == "large" {
		return "PRIVATE_LARGE_ORG"
	} else if input.OrganizationInfo.Size == "medium" {
		return "PRIVATE_MEDIUM_ORG"
	}

	return "PRIVATE_SMALL_ORG"
}

// calculateFileAccessibility determines how accessible the file is
func (ec *ExposureCalculator) calculateFileAccessibility(input RiskAssessmentInput) float64 {
	accessibility := 0.5 // Base accessibility

	// Public repos are highly accessible
	if input.RepositoryInfo.IsPublic {
		accessibility = 0.9
	}

	// Default branch is more accessible
	if input.RepositoryInfo.DefaultBranch == "main" ||
		input.RepositoryInfo.DefaultBranch == "master" {
		accessibility += 0.1
	}

	// Common file paths are more likely to be discovered
	commonPaths := []string{
		"config", "conf", ".env", "settings",
		"src", "app", "lib", "pkg",
	}

	filePath := strings.ToLower(input.FileContext.FilePath)
	for _, common := range commonPaths {
		if strings.Contains(filePath, common) {
			accessibility += 0.05
			break
		}
	}

	// Large files are more noticeable
	if input.FileContext.FileSize > 100*1024 { // > 100KB
		accessibility += 0.05
	}

	// Test files are less likely to be accessed in production
	if input.FileContext.IsTest {
		accessibility *= 0.5
	}

	return ec.normalizeScore(accessibility)
}

// calculateDataLifetime estimates how long the data has been exposed
func (ec *ExposureCalculator) calculateDataLifetime(input RiskAssessmentInput) int {
	// Use last commit date as a proxy for data age
	dataAge := time.Since(input.RepositoryInfo.LastCommit)

	// Convert to days
	days := int(dataAge.Hours() / 24)

	// Cap at 365 days for calculation purposes
	if days > 365 {
		days = 365
	}

	return days
}

// determineEncryptionStatus checks if data appears to be encrypted
func (ec *ExposureCalculator) determineEncryptionStatus(input RiskAssessmentInput) string {
	// Check for encryption indicators in the finding
	match := strings.ToLower(input.Finding.Match)
	context := strings.ToLower(input.Finding.Context)

	// Look for encryption patterns
	encryptionPatterns := []string{
		"encrypted", "encrypt", "aes", "rsa", "hash",
		"bcrypt", "pbkdf2", "sha256", "base64",
	}

	for _, pattern := range encryptionPatterns {
		if strings.Contains(context, pattern) {
			return "ENCRYPTED"
		}
	}

	// Check if the match looks encrypted (high entropy, special chars)
	if looksEncrypted(match) {
		return "POSSIBLY_ENCRYPTED"
	}

	// Check for plain text indicators
	if input.Finding.Validated {
		return "PLAIN_TEXT"
	}

	return "UNKNOWN"
}

// assessAccessControls evaluates the access control strength
func (ec *ExposureCalculator) assessAccessControls(input RiskAssessmentInput) float64 {
	controlStrength := 0.5 // Base control strength

	// Public repos have minimal access control
	if input.RepositoryInfo.IsPublic {
		return 0.1
	}

	// Organizations with security teams have better controls
	if input.OrganizationInfo.HasSecurityTeam {
		controlStrength += 0.2
	}

	// Regulated organizations have stronger controls
	if input.OrganizationInfo.Regulated {
		controlStrength += 0.2
	}

	// CI/CD presence suggests some access control
	if input.RepositoryInfo.HasCICD {
		controlStrength += 0.1
	}

	// Maturity level affects controls
	maturityBonus := map[string]float64{
		"high":   0.2,
		"medium": 0.1,
		"low":    -0.1,
	}

	if bonus, exists := maturityBonus[input.OrganizationInfo.MaturityLevel]; exists {
		controlStrength += bonus
	}

	return ec.normalizeScore(controlStrength)
}

// combineExposureFactors combines all factors into overall exposure score
func (ec *ExposureCalculator) combineExposureFactors(factors ExposureFactors, input RiskAssessmentInput) float64 {
	// Start with file accessibility as base
	baseExposure := factors.FileAccessibility

	// Repository visibility multipliers
	visibilityMultipliers := map[string]float64{
		"PUBLIC_HIGH_VISIBILITY":   1.5,
		"PUBLIC_MEDIUM_VISIBILITY": 1.3,
		"PUBLIC_LOW_VISIBILITY":    1.1,
		"PRIVATE_LARGE_ORG":        0.8,
		"PRIVATE_MEDIUM_ORG":       0.6,
		"PRIVATE_SMALL_ORG":        0.4,
	}

	if multiplier, exists := visibilityMultipliers[factors.RepositoryVisibility]; exists {
		baseExposure *= multiplier
	}

	// Data lifetime increases exposure (longer exposure = higher risk)
	lifetimeMultiplier := 1.0
	if factors.DataLifetime > 30 {
		lifetimeMultiplier = 1.1
	}
	if factors.DataLifetime > 90 {
		lifetimeMultiplier = 1.2
	}
	if factors.DataLifetime > 180 {
		lifetimeMultiplier = 1.3
	}
	if factors.DataLifetime > 365 {
		lifetimeMultiplier = 1.5
	}

	baseExposure *= lifetimeMultiplier

	// Encryption status affects exposure
	encryptionMultipliers := map[string]float64{
		"PLAIN_TEXT":         1.2,
		"UNKNOWN":            1.0,
		"POSSIBLY_ENCRYPTED": 0.6,
		"ENCRYPTED":          0.3,
	}

	if multiplier, exists := encryptionMultipliers[factors.EncryptionStatus]; exists {
		baseExposure *= multiplier
	}

	// Access controls reduce exposure
	baseExposure *= (1.0 - factors.AccessControls*0.5)

	// Additional factors based on PI type
	highExposureTypes := map[detection.PIType]bool{
		detection.PITypeTFN:        true,
		detection.PITypeCreditCard: true,
		detection.PITypeMedicare:   true,
		detection.PITypePassport:   true,
	}

	if highExposureTypes[input.Finding.Type] {
		baseExposure *= 1.2
	}

	// Network effects - more contributors = more exposure
	if input.RepositoryInfo.Contributors > 10 {
		baseExposure *= 1.1
	}
	if input.RepositoryInfo.Contributors > 50 {
		baseExposure *= 1.2
	}

	// Fork network increases exposure
	if input.RepositoryInfo.Forks > 10 {
		baseExposure *= 1.1
	}
	if input.RepositoryInfo.Forks > 100 {
		baseExposure *= 1.3
	}

	return ec.normalizeScore(baseExposure)
}

// looksEncrypted performs basic entropy check to detect encrypted data
func looksEncrypted(s string) bool {
	// Very basic check - in production, use proper entropy calculation

	// Check length (encrypted data is often longer)
	if len(s) < 20 {
		return false
	}

	// Check for high randomness indicators
	upperCount := 0
	lowerCount := 0
	digitCount := 0
	specialCount := 0

	for _, r := range s {
		switch {
		case r >= 'A' && r <= 'Z':
			upperCount++
		case r >= 'a' && r <= 'z':
			lowerCount++
		case r >= '0' && r <= '9':
			digitCount++
		default:
			specialCount++
		}
	}

	// Encrypted data typically has good distribution
	total := float64(len(s))
	hasUpper := float64(upperCount)/total > 0.2
	hasLower := float64(lowerCount)/total > 0.2
	hasDigit := float64(digitCount)/total > 0.2

	// If it has good mix of characters, might be encrypted
	if hasUpper && hasLower && hasDigit {
		return true
	}

	// Check for base64 pattern
	if strings.HasSuffix(s, "=") || strings.HasSuffix(s, "==") {
		return true
	}

	return false
}

// normalizeScore ensures score is within 0-1 range
func (ec *ExposureCalculator) normalizeScore(score float64) float64 {
	if score < 0 {
		return 0
	}
	if score > 1 {
		return 1
	}
	return score
}
