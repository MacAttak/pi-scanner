package scoring

import (
	"testing"

	"github.com/pi-scanner/pi-scanner/pkg/detection"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewScoreAggregator(t *testing.T) {
	tests := []struct {
		name    string
		config  *AggregatorConfig
		wantErr bool
	}{
		{
			name:    "default config success",
			config:  DefaultAggregatorConfig(),
			wantErr: false,
		},
		{
			name:    "nil config uses default",
			config:  nil,
			wantErr: false,
		},
		{
			name: "custom weights",
			config: &AggregatorConfig{
				WeightedCombination: true,
				ProximityWeight:     0.4,
				MLWeight:           0.3,
				ValidationWeight:   0.3,
				MaxScore:           1.0,
				MinScore:           0.0,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			aggregator, err := NewScoreAggregator(tt.config)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, aggregator)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, aggregator)
				assert.NotNil(t, aggregator.config)
			}
		})
	}
}

func TestScoreAggregator_AggregateScores(t *testing.T) {
	aggregator, err := NewScoreAggregator(DefaultAggregatorConfig())
	require.NoError(t, err)

	tests := []struct {
		name          string
		factors       FactorScores
		expectedRange []float64 // [min, max]
		description   string
	}{
		{
			name: "all high scores",
			factors: FactorScores{
				ProximityScore:   0.9,
				MLScore:         0.95,
				ValidationScore: 1.0,
				EnvironmentScore: 1.0,
				CoOccurrenceScore: 1.2,
				PITypeWeight:    1.0,
			},
			expectedRange: []float64{0.9, 1.0},
			description:   "high scores across all factors should result in high aggregate",
		},
		{
			name: "mixed scores",
			factors: FactorScores{
				ProximityScore:   0.7,
				MLScore:         0.6,
				ValidationScore: 0.8,
				EnvironmentScore: 1.0,
				CoOccurrenceScore: 1.0,
				PITypeWeight:    0.9,
			},
			expectedRange: []float64{0.6, 0.8},
			description:   "mixed scores should result in moderate aggregate",
		},
		{
			name: "test environment penalty",
			factors: FactorScores{
				ProximityScore:   0.8,
				MLScore:         0.9,
				ValidationScore: 1.0,
				EnvironmentScore: 0.2, // test environment
				CoOccurrenceScore: 1.0,
				PITypeWeight:    1.0,
			},
			expectedRange: []float64{0.15, 0.25},
			description:   "test environment should significantly reduce aggregate score",
		},
		{
			name: "validation failure override",
			factors: FactorScores{
				ProximityScore:   0.9,
				MLScore:         0.95,
				ValidationScore: 0.0, // validation failed
				EnvironmentScore: 1.0,
				CoOccurrenceScore: 1.0,
				PITypeWeight:    1.0,
			},
			expectedRange: []float64{0.6, 0.7},
			description:   "validation failure should reduce confidence but not eliminate it",
		},
		{
			name: "co-occurrence boost",
			factors: FactorScores{
				ProximityScore:   0.7,
				MLScore:         0.7,
				ValidationScore: 1.0,
				EnvironmentScore: 1.0,
				CoOccurrenceScore: 1.5, // high co-occurrence boost
				PITypeWeight:    1.0,
			},
			expectedRange: []float64{0.8, 1.0},
			description:   "co-occurrence should boost the aggregate score",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := aggregator.AggregateScores(tt.factors)

			assert.GreaterOrEqual(t, result, tt.expectedRange[0], tt.description)
			assert.LessOrEqual(t, result, tt.expectedRange[1], tt.description)
			assert.GreaterOrEqual(t, result, 0.0, "score should not be negative")
			assert.LessOrEqual(t, result, 1.0, "score should not exceed 1.0")
		})
	}
}

func TestScoreAggregator_MapScoreToRiskLevel(t *testing.T) {
	aggregator, err := NewScoreAggregator(DefaultAggregatorConfig())
	require.NoError(t, err)

	tests := []struct {
		name         string
		score        float64
		expectedRisk RiskLevel
	}{
		{"critical_high", 0.95, RiskLevelCritical},
		{"critical_boundary", 0.90, RiskLevelCritical},
		{"high_upper", 0.89, RiskLevelHigh},
		{"high_middle", 0.8, RiskLevelHigh},
		{"high_boundary", 0.70, RiskLevelHigh},
		{"medium_upper", 0.69, RiskLevelMedium},
		{"medium_middle", 0.5, RiskLevelMedium},
		{"medium_boundary", 0.40, RiskLevelMedium},
		{"low_upper", 0.39, RiskLevelLow},
		{"low_middle", 0.2, RiskLevelLow},
		{"low_zero", 0.0, RiskLevelLow},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := aggregator.MapScoreToRiskLevel(tt.score)
			assert.Equal(t, tt.expectedRisk, result)
		})
	}
}

func TestScoreAggregator_GenerateScoreBreakdown(t *testing.T) {
	aggregator, err := NewScoreAggregator(DefaultAggregatorConfig())
	require.NoError(t, err)

	factors := FactorScores{
		ProximityScore:    0.8,
		MLScore:          0.9,
		ValidationScore:  1.0,
		EnvironmentScore: 0.2, // test environment
		CoOccurrenceScore: 1.3,
		PITypeWeight:     1.0,
	}

	breakdown := aggregator.GenerateScoreBreakdown(factors, 0.65)

	// Verify all components are present
	assert.NotEmpty(t, breakdown.Components, "should have score components")
	assert.NotEmpty(t, breakdown.Weights, "should have weight information")
	assert.NotEmpty(t, breakdown.Adjustments, "should have adjustment information")
	assert.Equal(t, 0.65, breakdown.FinalScore, "should match final score")

	// Check for required components
	componentNames := make(map[string]bool)
	for _, comp := range breakdown.Components {
		componentNames[comp.Name] = true
		assert.GreaterOrEqual(t, comp.Score, 0.0, "component scores should be non-negative")
		assert.LessOrEqual(t, comp.Score, 2.0, "component scores should be reasonable")
		assert.GreaterOrEqual(t, comp.Weight, 0.0, "weights should be non-negative")
		assert.NotEmpty(t, comp.Description, "should have description")
	}

	expectedComponents := []string{"proximity", "ml_validation", "algorithmic_validation", "environment", "co_occurrence", "pi_type_weight"}
	for _, expected := range expectedComponents {
		assert.True(t, componentNames[expected], "should have component: %s", expected)
	}

	// Verify adjustments for test environment
	foundEnvironmentAdjustment := false
	for _, adj := range breakdown.Adjustments {
		if adj.Type == "environment_penalty" {
			foundEnvironmentAdjustment = true
			assert.Less(t, adj.Impact, 0.0, "environment penalty should have negative impact")
		}
	}
	assert.True(t, foundEnvironmentAdjustment, "should have environment penalty adjustment")
}

func TestScoreAggregator_GenerateAuditTrail(t *testing.T) {
	aggregator, err := NewScoreAggregator(DefaultAggregatorConfig())
	require.NoError(t, err)

	factors := FactorScores{
		ProximityScore:    0.8,
		MLScore:          0.9,
		ValidationScore:  1.0,
		EnvironmentScore: 1.0,
		CoOccurrenceScore: 1.2,
		PITypeWeight:     1.0,
	}

	piType := detection.PITypeTFN
	auditTrail := aggregator.GenerateAuditTrail(factors, 0.85, piType)

	// Verify audit trail structure
	assert.NotEmpty(t, auditTrail, "should have audit trail entries")
	assert.GreaterOrEqual(t, len(auditTrail), 4, "should have multiple audit entries")

	// Check audit entry properties
	for _, entry := range auditTrail {
		assert.NotEmpty(t, entry.Component, "should have component name")
		assert.NotEmpty(t, entry.Description, "should have description")
		assert.False(t, entry.Timestamp.IsZero(), "should have timestamp")
		assert.GreaterOrEqual(t, entry.Score, 0.0, "audit scores should be non-negative")
		assert.NotEmpty(t, entry.Details, "should have details map")
	}

	// Verify specific components are audited
	componentsSeen := make(map[string]bool)
	for _, entry := range auditTrail {
		componentsSeen[entry.Component] = true
	}

	expectedComponents := []string{"proximity", "ml_validation", "algorithmic_validation", "aggregation"}
	for _, expected := range expectedComponents {
		assert.True(t, componentsSeen[expected], "should audit component: %s", expected)
	}

	// Check for regulatory compliance details in TFN audit
	foundComplianceDetails := false
	for _, entry := range auditTrail {
		if details, ok := entry.Details["regulatory_compliance"]; ok {
			foundComplianceDetails = true
			assert.NotEmpty(t, details, "should have regulatory compliance details")
		}
	}
	assert.True(t, foundComplianceDetails, "TFN audit should include regulatory compliance details")
}

func TestScoreAggregator_GenerateRegulatoryCompliance(t *testing.T) {
	aggregator, err := NewScoreAggregator(DefaultAggregatorConfig())
	require.NoError(t, err)

	tests := []struct {
		name            string
		piType          detection.PIType
		riskLevel       RiskLevel
		expectedAPRA    bool
		expectedPrivacy bool
		minActions      int
		description     string
	}{
		{
			name:            "TFN critical risk",
			piType:          detection.PITypeTFN,
			riskLevel:       RiskLevelCritical,
			expectedAPRA:    true,
			expectedPrivacy: true,
			minActions:      2,
			description:     "TFN critical should trigger all compliance requirements",
		},
		{
			name:            "Medicare high risk",
			piType:          detection.PITypeMedicare,
			riskLevel:       RiskLevelHigh,
			expectedAPRA:    true,
			expectedPrivacy: true,
			minActions:      2,
			description:     "Medicare high should trigger compliance requirements",
		},
		{
			name:            "ABN medium risk",
			piType:          detection.PITypeABN,
			riskLevel:       RiskLevelMedium,
			expectedAPRA:    true,
			expectedPrivacy: false, // ABN is business data, not personal
			minActions:      1,
			description:     "ABN medium should trigger APRA but not Privacy Act requirements",
		},
		{
			name:            "Email low risk",
			piType:          detection.PITypeEmail,
			riskLevel:       RiskLevelLow,
			expectedAPRA:    false,
			expectedPrivacy: true,
			minActions:      0,
			description:     "Email low risk should have minimal compliance requirements",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compliance := aggregator.GenerateRegulatoryCompliance(tt.piType, tt.riskLevel)

			assert.Equal(t, tt.expectedAPRA, compliance.APRA, tt.description)
			assert.Equal(t, tt.expectedPrivacy, compliance.PrivacyAct, tt.description)
			assert.GreaterOrEqual(t, len(compliance.RequiredActions), tt.minActions, tt.description)

			// Verify action structure
			for _, action := range compliance.RequiredActions {
				assert.NotEmpty(t, action.Type, "action should have type")
				assert.NotEmpty(t, action.Description, "action should have description")
				assert.NotEmpty(t, action.Priority, "action should have priority")
				assert.False(t, action.Deadline.IsZero(), "action should have deadline")
			}

			// Check for Australian-specific requirements
			if tt.piType == detection.PITypeTFN || tt.piType == detection.PITypeMedicare {
				foundAustralianAction := false
				for _, action := range compliance.RequiredActions {
					if action.Type == "BANKING_REGULATION" || action.Type == "HEALTHCARE_REGULATION" {
						foundAustralianAction = true
						assert.Contains(t, action.Description, "Australian", 
							"should mention Australian regulations")
					}
				}
				if tt.riskLevel != RiskLevelLow {
					assert.True(t, foundAustralianAction, 
						"should have Australian-specific action for %s", tt.piType)
				}
			}
		})
	}
}

func TestScoreAggregator_WeightedCombination(t *testing.T) {
	config := DefaultAggregatorConfig()
	config.WeightedCombination = true
	config.ProximityWeight = 0.4
	config.MLWeight = 0.3
	config.ValidationWeight = 0.3

	aggregator, err := NewScoreAggregator(config)
	require.NoError(t, err)

	factors := FactorScores{
		ProximityScore:   0.8,
		MLScore:         0.6,
		ValidationScore: 1.0,
		EnvironmentScore: 1.0,
		CoOccurrenceScore: 1.0,
		PITypeWeight:    1.0,
	}

	result := aggregator.AggregateScores(factors)

	// Expected: (0.8 * 0.4 + 0.6 * 0.3 + 1.0 * 0.3) * 1.0 * 1.0 * 1.0 = 0.8
	expectedWeightedScore := 0.8*0.4 + 0.6*0.3 + 1.0*0.3 // = 0.8
	assert.InDelta(t, expectedWeightedScore, result, 0.1, 
		"weighted combination should follow configured weights")
}

func TestScoreAggregator_EdgeCases(t *testing.T) {
	aggregator, err := NewScoreAggregator(DefaultAggregatorConfig())
	require.NoError(t, err)

	tests := []struct {
		name        string
		factors     FactorScores
		expectedMin float64
		expectedMax float64
		description string
	}{
		{
			name: "all zero scores",
			factors: FactorScores{
				ProximityScore:   0.0,
				MLScore:         0.0,
				ValidationScore: 0.0,
				EnvironmentScore: 0.0,
				CoOccurrenceScore: 0.0,
				PITypeWeight:    0.0,
			},
			expectedMin: 0.0,
			expectedMax: 0.0,
			description: "all zero should result in zero",
		},
		{
			name: "maximum scores",
			factors: FactorScores{
				ProximityScore:   1.0,
				MLScore:         1.0,
				ValidationScore: 1.0,
				EnvironmentScore: 2.0, // boosted
				CoOccurrenceScore: 2.0, // boosted
				PITypeWeight:    1.0,
			},
			expectedMin: 0.9,
			expectedMax: 1.0,
			description: "maximum scores should be capped at 1.0",
		},
		{
			name: "negative environment score",
			factors: FactorScores{
				ProximityScore:   0.8,
				MLScore:         0.8,
				ValidationScore: 0.8,
				EnvironmentScore: -0.1, // invalid negative
				CoOccurrenceScore: 1.0,
				PITypeWeight:    1.0,
			},
			expectedMin: 0.0,
			expectedMax: 0.3,
			description: "should handle negative scores gracefully",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := aggregator.AggregateScores(tt.factors)

			assert.GreaterOrEqual(t, result, tt.expectedMin, tt.description)
			assert.LessOrEqual(t, result, tt.expectedMax, tt.description)
			assert.GreaterOrEqual(t, result, 0.0, "result should not be negative")
			assert.LessOrEqual(t, result, 1.0, "result should not exceed 1.0")
		})
	}
}

func TestScoreAggregator_AustralianBankingCompliance(t *testing.T) {
	aggregator, err := NewScoreAggregator(DefaultAggregatorConfig())
	require.NoError(t, err)

	// Test specific Australian banking regulation requirements
	bankingPITypes := []detection.PIType{
		detection.PITypeTFN,
		detection.PITypeBSB,
		detection.PITypeAccount,
	}

	for _, piType := range bankingPITypes {
		t.Run(string(piType), func(t *testing.T) {
			compliance := aggregator.GenerateRegulatoryCompliance(piType, RiskLevelHigh)

			// Banking PI should always trigger APRA compliance
			assert.True(t, compliance.APRA, 
				"Banking PI type %s should trigger APRA compliance", piType)

			// Should have banking-specific actions
			foundBankingAction := false
			for _, action := range compliance.RequiredActions {
				if action.Type == "BANKING_REGULATION" {
					foundBankingAction = true
					assert.Contains(t, action.Description, "banking", 
						"Banking action should mention banking regulations")
					assert.Equal(t, "HIGH", action.Priority, 
						"Banking actions should be high priority")
				}
			}

			if piType == detection.PITypeTFN || piType == detection.PITypeBSB {
				assert.True(t, foundBankingAction, 
					"Should have banking action for %s", piType)
			}
		})
	}
}

// Benchmark tests for performance validation
func BenchmarkScoreAggregator_AggregateScores(b *testing.B) {
	aggregator, err := NewScoreAggregator(DefaultAggregatorConfig())
	require.NoError(b, err)

	factors := FactorScores{
		ProximityScore:    0.8,
		MLScore:          0.7,
		ValidationScore:  1.0,
		EnvironmentScore: 1.0,
		CoOccurrenceScore: 1.2,
		PITypeWeight:     0.9,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		aggregator.AggregateScores(factors)
	}
}

func BenchmarkScoreAggregator_GenerateScoreBreakdown(b *testing.B) {
	aggregator, err := NewScoreAggregator(DefaultAggregatorConfig())
	require.NoError(b, err)

	factors := FactorScores{
		ProximityScore:    0.8,
		MLScore:          0.7,
		ValidationScore:  1.0,
		EnvironmentScore: 1.0,
		CoOccurrenceScore: 1.2,
		PITypeWeight:     0.9,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		aggregator.GenerateScoreBreakdown(factors, 0.75)
	}
}

func BenchmarkScoreAggregator_GenerateAuditTrail(b *testing.B) {
	aggregator, err := NewScoreAggregator(DefaultAggregatorConfig())
	require.NoError(b, err)

	factors := FactorScores{
		ProximityScore:    0.8,
		MLScore:          0.7,
		ValidationScore:  1.0,
		EnvironmentScore: 1.0,
		CoOccurrenceScore: 1.2,
		PITypeWeight:     0.9,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		aggregator.GenerateAuditTrail(factors, 0.75, detection.PITypeTFN)
	}
}