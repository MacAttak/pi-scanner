package scoring

import (
	"regexp"
	"strings"

	"github.com/pi-scanner/pi-scanner/pkg/detection"
)

// FactorEngine calculates individual scoring factors for confidence assessment
type FactorEngine struct {
	config *FactorConfig
	
	// Pre-compiled patterns for performance
	testPatterns         []*regexp.Regexp
	prodPatterns         []*regexp.Regexp
	mockPatterns         []*regexp.Regexp
	documentationPatterns []*regexp.Regexp
}

// FactorConfig holds configuration for factor calculations
type FactorConfig struct {
	// Component weights for score calculation
	ProximityWeight   float64 `json:"proximity_weight"`
	MLWeight         float64 `json:"ml_weight"`
	ValidationWeight float64 `json:"validation_weight"`
	
	// Environment penalties/bonuses
	EnvironmentPenalties map[string]float64 `json:"environment_penalties"`
	EnvironmentBonuses   map[string]float64 `json:"environment_bonuses"`
	
	// Co-occurrence multipliers based on Australian PI risk combinations
	CoOccurrenceMultipliers map[string]map[string]float64 `json:"co_occurrence_multipliers"`
	
	// PI type weights aligned with Australian banking regulations
	PITypeWeights map[detection.PIType]float64 `json:"pi_type_weights"`
	
	// Distance decay factor for co-occurrences
	DistanceDecayFactor float64 `json:"distance_decay_factor"`
	MaxCoOccurrenceBoost float64 `json:"max_co_occurrence_boost"`
}

// DefaultFactorConfig returns the default factor configuration optimized for Australian PI detection
func DefaultFactorConfig() *FactorConfig {
	return &FactorConfig{
		ProximityWeight:   0.4,
		MLWeight:         0.3,
		ValidationWeight: 0.3,
		
		EnvironmentPenalties: map[string]float64{
			"test":          0.2,  // Heavy penalty for test environments
			"mock":          0.1,  // Very heavy penalty for mock data
			"sample":        0.2,  // Heavy penalty for sample data
			"demo":          0.2,  // Heavy penalty for demo data
			"fixture":       0.1,  // Very heavy penalty for fixture data
			"example":       0.3,  // Moderate penalty for examples
			"documentation": 0.5,  // Moderate penalty for docs
			"debug":         0.7,  // Light penalty for debug code
		},
		
		EnvironmentBonuses: map[string]float64{
			"production": 1.2,  // Bonus for production environments
			"prod":       1.2,  // Bonus for prod environments
			"live":       1.2,  // Bonus for live environments
			"release":    1.1,  // Small bonus for release code
		},
		
		// Australian PI co-occurrence risk matrix
		CoOccurrenceMultipliers: map[string]map[string]float64{
			string(detection.PITypeTFN): {
				string(detection.PITypeMedicare): 1.4, // TFN + Medicare = high identity risk
				string(detection.PITypeName):    1.3, // TFN + Name = high identity risk
				string(detection.PITypeAddress): 1.3, // TFN + Address = high identity risk
				string(detection.PITypePhone):   1.2, // TFN + Phone = moderate identity risk
				string(detection.PITypeEmail):   1.1, // TFN + Email = slight identity risk
				string(detection.PITypeABN):     1.2, // TFN + ABN = business identity risk
			},
			string(detection.PITypeMedicare): {
				string(detection.PITypeName):    1.2, // Medicare + Name = healthcare identity
				string(detection.PITypeAddress): 1.2, // Medicare + Address = healthcare identity
				string(detection.PITypePhone):   1.1, // Medicare + Phone = healthcare contact
			},
			string(detection.PITypeBSB): {
				string(detection.PITypeAccount): 1.3, // BSB + Account = complete banking details
				string(detection.PITypeName):    1.2, // BSB + Name = banking identity
				string(detection.PITypeTFN):     1.2, // BSB + TFN = financial identity
			},
			string(detection.PITypeABN): {
				string(detection.PITypeName):    1.2, // ABN + Name = business identity
				string(detection.PITypeAddress): 1.2, // ABN + Address = business location
				string(detection.PITypePhone):   1.1, // ABN + Phone = business contact
			},
			string(detection.PITypeCreditCard): {
				string(detection.PITypeName):    1.3, // CC + Name = financial identity
				string(detection.PITypeAddress): 1.3, // CC + Address = billing identity
				string(detection.PITypePhone):   1.2, // CC + Phone = cardholder contact
			},
		},
		
		// PI type weights based on Australian regulatory requirements
		PITypeWeights: map[detection.PIType]float64{
			detection.PITypeTFN:          1.0,  // Maximum weight - most sensitive under Australian law
			detection.PITypeMedicare:     0.95, // Very high - healthcare data protection
			detection.PITypeCreditCard:   0.9,  // High - financial data protection
			detection.PITypePassport:     0.9,  // High - identity document
			detection.PITypeABN:          0.8,  // Moderate-high - business identification
			detection.PITypeDriverLicense: 0.8,  // Moderate-high - identity document
			detection.PITypeBSB:          0.7,  // Moderate - banking code
			detection.PITypeAccount:      0.7,  // Moderate - account number
			detection.PITypeName:         0.6,  // Moderate - personal identifier
			detection.PITypeAddress:      0.6,  // Moderate - personal identifier
			detection.PITypePhone:        0.5,  // Lower - contact information
			detection.PITypeEmail:        0.4,  // Lower - contact information
			detection.PITypeIP:           0.2,  // Lowest - technical identifier
		},
		
		DistanceDecayFactor:   0.9,  // Exponential decay for distance
		MaxCoOccurrenceBoost: 1.6,  // Cap boost to prevent runaway scores
	}
}

// FactorScores holds the calculated scores for all factors
type FactorScores struct {
	ProximityScore    float64 `json:"proximity_score"`
	MLScore          float64 `json:"ml_score"`
	ValidationScore  float64 `json:"validation_score"`
	EnvironmentScore float64 `json:"environment_score"`
	CoOccurrenceScore float64 `json:"co_occurrence_score"`
	PITypeWeight     float64 `json:"pi_type_weight"`
}

// NewFactorEngine creates a new factor engine with the given configuration
func NewFactorEngine(config *FactorConfig) (*FactorEngine, error) {
	if config == nil {
		config = DefaultFactorConfig()
	}
	
	engine := &FactorEngine{
		config: config,
	}
	
	// Compile patterns for performance
	engine.compilePatterns()
	
	return engine, nil
}

// compilePatterns pre-compiles regex patterns for efficient matching
func (e *FactorEngine) compilePatterns() {
	// Test environment patterns
	testPatterns := []string{
		`(?i)\btest\b`,
		`(?i)\bmock\b`,
		`(?i)\bsample\b`,
		`(?i)\bdemo\b`,
		`(?i)\bfake\b`,
		`(?i)\bdummy\b`,
		`(?i)\bstub\b`,
		`(?i)_test\.`,
		`(?i)test_`,
		`/test/`,
		`/tests/`,
		`/testing/`,
		`/spec/`,
		`/fixtures/`,
		`/mocks/`,
	}
	
	e.testPatterns = make([]*regexp.Regexp, len(testPatterns))
	for i, pattern := range testPatterns {
		e.testPatterns[i] = regexp.MustCompile(pattern)
	}
	
	// Production environment patterns
	prodPatterns := []string{
		`(?i)\bprod\b`,
		`(?i)\bproduction\b`,
		`(?i)\blive\b`,
		`(?i)\brelease\b`,
		`/prod/`,
		`/production/`,
		`/release/`,
		`(?i)prod_`,
		`(?i)production_`,
	}
	
	e.prodPatterns = make([]*regexp.Regexp, len(prodPatterns))
	for i, pattern := range prodPatterns {
		e.prodPatterns[i] = regexp.MustCompile(pattern)
	}
	
	// Mock/fixture patterns
	mockPatterns := []string{
		`(?i)\bfixture\b`,
		`(?i)mock_`,
		`_mock\.`,
		`/fixtures/`,
		`/examples/`,
		`\.fixture\.`,
		`\.mock\.`,
	}
	
	e.mockPatterns = make([]*regexp.Regexp, len(mockPatterns))
	for i, pattern := range mockPatterns {
		e.mockPatterns[i] = regexp.MustCompile(pattern)
	}
	
	// Documentation patterns
	docPatterns := []string{
		`(?i)\bdocs?\b`,
		`(?i)\bdocumentation\b`,
		`(?i)\breadme\b`,
		`(?i)\bguide\b`,
		`(?i)\btutorial\b`,
		`/docs/`,
		`/documentation/`,
		`\.md$`,
		`\.rst$`,
		`//.*example`,
		`#.*example`,
		`<!--.*example`,
	}
	
	e.documentationPatterns = make([]*regexp.Regexp, len(docPatterns))
	for i, pattern := range docPatterns {
		e.documentationPatterns[i] = regexp.MustCompile(pattern)
	}
}

// CalculateProximityFactor calculates the proximity score factor
func (e *FactorEngine) CalculateProximityFactor(proximityData *ProximityScore) float64 {
	if proximityData == nil {
		return 0.5 // neutral default when no proximity data available
	}
	
	baseScore := proximityData.Score
	
	// Apply context-based penalties
	contextPenalties := map[string]float64{
		"test":          0.2,
		"mock":          0.1,
		"sample":        0.2,
		"demo":          0.2,
		"fixture":       0.1,
		"documentation": 0.5,
		"debug":         0.7,
	}
	
	if penalty, exists := contextPenalties[proximityData.Context]; exists {
		baseScore *= penalty
	}
	
	// Ensure score is within bounds
	if baseScore < 0.0 {
		baseScore = 0.0
	}
	if baseScore > 1.0 {
		baseScore = 1.0
	}
	
	return baseScore
}

// CalculateMLFactor calculates the machine learning score factor
func (e *FactorEngine) CalculateMLFactor(mlData *MLScore) float64 {
	if mlData == nil {
		return 0.5 // neutral default when no ML data available
	}
	
	baseScore := float64(mlData.Confidence)
	
	// Heavy penalty for invalid ML predictions
	if !mlData.IsValid {
		baseScore *= 0.2
	}
	
	// Ensure score is within bounds
	if baseScore < 0.0 {
		baseScore = 0.0
	}
	if baseScore > 1.0 {
		baseScore = 1.0
	}
	
	return baseScore
}

// CalculateValidationFactor calculates the algorithmic validation score factor
func (e *FactorEngine) CalculateValidationFactor(validationData *ValidationScore) float64 {
	if validationData == nil {
		return 0.0 // no validation data means we can't validate
	}
	
	// Validation is binary for algorithmic checks (TFN, ABN, Medicare, BSB)
	if validationData.IsValid {
		return validationData.Confidence
	}
	
	return 0.0
}

// CalculateEnvironmentFactor calculates the environment-based score factor
func (e *FactorEngine) CalculateEnvironmentFactor(filename, content string) float64 {
	baseScore := 1.0 // neutral baseline
	
	indicators := e.DetectEnvironmentIndicators(filename, content)
	
	// Apply penalties and bonuses based on detected indicators
	for _, indicator := range indicators {
		if penalty, exists := e.config.EnvironmentPenalties[indicator]; exists {
			baseScore *= penalty
		} else if bonus, exists := e.config.EnvironmentBonuses[indicator]; exists {
			baseScore *= bonus
		}
	}
	
	// Ensure score is within reasonable bounds
	if baseScore < 0.0 {
		baseScore = 0.0
	}
	if baseScore > 2.0 { // Allow some boost but cap it
		baseScore = 2.0
	}
	
	return baseScore
}

// CalculateCoOccurrenceFactor calculates the co-occurrence score factor
func (e *FactorEngine) CalculateCoOccurrenceFactor(piType detection.PIType, coOccurrences []CoOccurrence) float64 {
	if len(coOccurrences) == 0 {
		return 1.0 // neutral when no co-occurrences
	}
	
	baseMultiplier := 1.0
	
	// Get multipliers for this PI type
	typeMultipliers, exists := e.config.CoOccurrenceMultipliers[string(piType)]
	if !exists {
		return 1.0 // no multipliers defined for this type
	}
	
	// Calculate compound multiplier based on co-occurrences
	for _, coOcc := range coOccurrences {
		if multiplier, exists := typeMultipliers[string(coOcc.PIType)]; exists {
			// Apply distance decay
			distanceDecay := 1.0
			if coOcc.Distance > 1 {
				for i := 1; i < coOcc.Distance; i++ {
					distanceDecay *= e.config.DistanceDecayFactor
				}
			}
			
			adjustedMultiplier := 1.0 + (multiplier-1.0)*distanceDecay
			baseMultiplier *= adjustedMultiplier
		}
	}
	
	// Cap the boost to prevent runaway scores
	if baseMultiplier > e.config.MaxCoOccurrenceBoost {
		baseMultiplier = e.config.MaxCoOccurrenceBoost
	}
	
	return baseMultiplier
}

// CalculatePITypeWeight calculates the weight factor based on PI type importance
func (e *FactorEngine) CalculatePITypeWeight(piType detection.PIType) float64 {
	if weight, exists := e.config.PITypeWeights[piType]; exists {
		return weight
	}
	
	// Default weight for unknown PI types
	return 0.5
}

// DetectEnvironmentIndicators detects environment-related indicators in filename and content
func (e *FactorEngine) DetectEnvironmentIndicators(filename, content string) []string {
	indicators := make([]string, 0)
	seen := make(map[string]bool)
	
	// Check filename and content against compiled patterns
	fullText := filename + " " + content
	
	// Test patterns
	for _, pattern := range e.testPatterns {
		if pattern.MatchString(fullText) {
			if !seen["test"] {
				indicators = append(indicators, "test")
				seen["test"] = true
			}
			break
		}
	}
	
	// Production patterns
	for _, pattern := range e.prodPatterns {
		if pattern.MatchString(fullText) {
			if !seen["production"] {
				indicators = append(indicators, "production")
				seen["production"] = true
			}
			break
		}
	}
	
	// Mock/fixture patterns
	for _, pattern := range e.mockPatterns {
		if pattern.MatchString(fullText) {
			patternStr := pattern.String()
			if strings.Contains(patternStr, "mock") && !seen["mock"] {
				indicators = append(indicators, "mock")
				seen["mock"] = true
			}
			if strings.Contains(patternStr, "fixture") && !seen["fixture"] {
				indicators = append(indicators, "fixture")
				seen["fixture"] = true
			}
		}
	}
	
	// Documentation patterns
	for _, pattern := range e.documentationPatterns {
		if pattern.MatchString(fullText) {
			if !seen["docs"] {
				indicators = append(indicators, "docs")
				seen["docs"] = true
			}
			break
		}
	}
	
	// Additional content-based keywords
	contentLower := strings.ToLower(content)
	keywords := map[string]string{
		"sample":   "sample",
		"demo":     "demo",
		"dummy":    "dummy",
		"stub":     "stub",
		"debug":    "debug",
		"dev":      "development",
		"staging":  "staging",
	}
	
	for keyword, indicator := range keywords {
		if strings.Contains(contentLower, keyword) && !seen[indicator] {
			indicators = append(indicators, indicator)
			seen[indicator] = true
		}
	}
	
	return indicators
}

// GetFactorWeights returns the configured factor weights
func (e *FactorEngine) GetFactorWeights() map[string]float64 {
	return map[string]float64{
		"proximity":   e.config.ProximityWeight,
		"ml":         e.config.MLWeight,
		"validation": e.config.ValidationWeight,
	}
}

// GetPITypeWeights returns all PI type weights
func (e *FactorEngine) GetPITypeWeights() map[detection.PIType]float64 {
	return e.config.PITypeWeights
}

// GetEnvironmentPenalties returns all environment penalties
func (e *FactorEngine) GetEnvironmentPenalties() map[string]float64 {
	return e.config.EnvironmentPenalties
}

// GetCoOccurrenceMultipliers returns all co-occurrence multipliers
func (e *FactorEngine) GetCoOccurrenceMultipliers() map[string]map[string]float64 {
	return e.config.CoOccurrenceMultipliers
}