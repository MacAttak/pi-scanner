package scoring

import (
	"testing"
	"time"

	"github.com/pi-scanner/pi-scanner/pkg/detection"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRiskMatrix(t *testing.T) {
	tests := []struct {
		name    string
		config  *RiskMatrixConfig
		wantErr bool
	}{
		{
			name:    "default config success",
			config:  DefaultRiskMatrixConfig(),
			wantErr: false,
		},
		{
			name:    "nil config uses default",
			config:  nil,
			wantErr: false,
		},
		{
			name: "custom config",
			config: &RiskMatrixConfig{
				UseMultiplicativeModel: false,
				ImpactWeight:          0.5,
				LikelihoodWeight:      0.3,
				ExposureWeight:        0.2,
				APRAAligned:           true,
				PrivacyActAligned:     true,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matrix, err := NewRiskMatrix(tt.config)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, matrix)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, matrix)
				assert.NotNil(t, matrix.impactCalculator)
				assert.NotNil(t, matrix.likelihoodCalculator)
				assert.NotNil(t, matrix.exposureCalculator)
			}
		})
	}
}

func TestRiskMatrix_AssessRisk_TFN(t *testing.T) {
	matrix, err := NewRiskMatrix(DefaultRiskMatrixConfig())
	require.NoError(t, err)

	tests := []struct {
		name             string
		input            RiskAssessmentInput
		expectedRiskRange []float64
		expectedLevel    RiskLevel
		expectedCategory RiskCategory
		description      string
	}{
		{
			name: "TFN in public repo - critical risk",
			input: RiskAssessmentInput{
				Finding: detection.Finding{
					Type:      detection.PITypeTFN,
					Match:     "123-456-789",
					File:      "src/customer.go",
					Validated: true,
				},
				ConfidenceScore: 0.95,
				RepositoryInfo: RepositoryInfo{
					IsPublic:     true,
					Stars:        500,
					Contributors: 20,
					LastCommit:   time.Now().Add(-24 * time.Hour),
				},
				FileContext: FileContext{
					FilePath:     "src/customer.go",
					IsProduction: true,
					IsSource:     true,
				},
				OrganizationInfo: OrganizationInfo{
					Industry:  "banking",
					Regulated: true,
				},
			},
			expectedRiskRange: []float64{0.8, 1.0},
			expectedLevel:    RiskLevelCritical,
			expectedCategory: RiskCategoryIdentityTheft,
			description:      "TFN in public banking repo should be critical risk",
		},
		{
			name: "TFN in test file - low risk",
			input: RiskAssessmentInput{
				Finding: detection.Finding{
					Type:      detection.PITypeTFN,
					Match:     "123-456-789",
					File:      "test/customer_test.go",
					Validated: false,
				},
				ConfidenceScore: 0.3,
				RepositoryInfo: RepositoryInfo{
					IsPublic:   false,
					LastCommit: time.Now(),
				},
				FileContext: FileContext{
					FilePath: "test/customer_test.go",
					IsTest:   true,
				},
				OrganizationInfo: OrganizationInfo{
					Industry: "technology",
				},
			},
			expectedRiskRange: []float64{0.0, 0.39},
			expectedLevel:    RiskLevelLow,
			expectedCategory: RiskCategoryOperational,
			description:      "Invalid TFN in test file should be low risk",
		},
		{
			name: "TFN with credit card co-occurrence - financial fraud risk",
			input: RiskAssessmentInput{
				Finding: detection.Finding{
					Type:      detection.PITypeTFN,
					Match:     "123-456-789",
					File:      "src/payment.go",
					Validated: true,
				},
				ConfidenceScore: 0.9,
				RepositoryInfo: RepositoryInfo{
					IsPublic:   true,
					LastCommit: time.Now(),
				},
				FileContext: FileContext{
					FilePath:     "src/payment.go",
					IsProduction: true,
				},
				CoOccurrences: []detection.Finding{
					{
						Type:  detection.PITypeCreditCard,
						Match: "4111111111111111",
					},
				},
				OrganizationInfo: OrganizationInfo{
					Industry:  "finance",
					Regulated: true,
				},
			},
			expectedRiskRange: []float64{0.8, 1.0},
			expectedLevel:    RiskLevelCritical,
			expectedCategory: RiskCategoryFinancialFraud,
			description:      "TFN with credit card should indicate financial fraud risk",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := matrix.AssessRisk(tt.input)

			require.NoError(t, err)
			assert.NotNil(t, result)
			
			// Check overall risk score
			assert.GreaterOrEqual(t, result.OverallRisk, tt.expectedRiskRange[0], tt.description)
			assert.LessOrEqual(t, result.OverallRisk, tt.expectedRiskRange[1], tt.description)
			
			// Check risk level
			assert.Equal(t, tt.expectedLevel, result.RiskLevel, tt.description)
			
			// Check risk category
			assert.Equal(t, tt.expectedCategory, result.RiskCategory, tt.description)
			
			// Verify component scores
			assert.GreaterOrEqual(t, result.ImpactScore, 0.0)
			assert.LessOrEqual(t, result.ImpactScore, 1.0)
			assert.GreaterOrEqual(t, result.LikelihoodScore, 0.0)
			assert.LessOrEqual(t, result.LikelihoodScore, 1.0)
			assert.GreaterOrEqual(t, result.ExposureScore, 0.0)
			assert.LessOrEqual(t, result.ExposureScore, 1.0)
			
			// Check for mitigations
			assert.NotEmpty(t, result.Mitigations)
			
			// Check compliance flags for regulated industries
			if tt.input.OrganizationInfo.Regulated {
				assert.NotNil(t, result.ComplianceFlags)
			}
		})
	}
}

func TestRiskMatrix_AssessRisk_Medicare(t *testing.T) {
	matrix, err := NewRiskMatrix(DefaultRiskMatrixConfig())
	require.NoError(t, err)

	input := RiskAssessmentInput{
		Finding: detection.Finding{
			Type:      detection.PITypeMedicare,
			Match:     "2234567890",
			File:      "src/healthcare.go",
			Validated: true,
		},
		ConfidenceScore: 0.85,
		RepositoryInfo: RepositoryInfo{
			IsPublic:   true,
			LastCommit: time.Now(),
		},
		FileContext: FileContext{
			FilePath:     "src/healthcare.go",
			IsProduction: true,
		},
		OrganizationInfo: OrganizationInfo{
			Industry:  "healthcare",
			Regulated: true,
		},
	}

	result, err := matrix.AssessRisk(input)

	require.NoError(t, err)
	assert.NotNil(t, result)
	
	// Medicare in healthcare should be high/critical risk
	assert.GreaterOrEqual(t, result.OverallRisk, 0.6)
	assert.Contains(t, []RiskLevel{RiskLevelHigh, RiskLevelCritical}, result.RiskLevel)
	
	// Should have healthcare-specific mitigations
	foundHealthcareMitigation := false
	for _, mit := range result.Mitigations {
		if mit.Category == "COMPLIANCE" || mit.Category == "RESPONSE" {
			foundHealthcareMitigation = true
			break
		}
	}
	assert.True(t, foundHealthcareMitigation, "Should have healthcare-specific mitigations")
}

func TestRiskMatrix_Mitigations(t *testing.T) {
	matrix, err := NewRiskMatrix(DefaultRiskMatrixConfig())
	require.NoError(t, err)

	tests := []struct {
		name              string
		riskLevel         RiskLevel
		category          RiskCategory
		minMitigations    int
		requiredPriorities []string
	}{
		{
			name:              "critical risk mitigations",
			riskLevel:         RiskLevelCritical,
			category:          RiskCategoryIdentityTheft,
			minMitigations:    3,
			requiredPriorities: []string{"CRITICAL", "HIGH"},
		},
		{
			name:              "high risk mitigations",
			riskLevel:         RiskLevelHigh,
			category:          RiskCategoryFinancialFraud,
			minMitigations:    3,
			requiredPriorities: []string{"HIGH"},
		},
		{
			name:              "medium risk mitigations",
			riskLevel:         RiskLevelMedium,
			category:          RiskCategoryOperational,
			minMitigations:    2,
			requiredPriorities: []string{"MEDIUM"},
		},
		{
			name:              "low risk mitigations",
			riskLevel:         RiskLevelLow,
			category:          RiskCategoryOperational,
			minMitigations:    2,
			requiredPriorities: []string{"MEDIUM"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := RiskAssessmentInput{
				Finding: detection.Finding{
					Type: detection.PITypeTFN,
				},
				ConfidenceScore: 0.8,
			}

			mitigations := matrix.generateMitigations(input, tt.riskLevel, tt.category)

			assert.GreaterOrEqual(t, len(mitigations), tt.minMitigations)
			
			// Check for required priority levels
			priorities := make(map[string]bool)
			for _, mit := range mitigations {
				priorities[mit.Priority] = true
				
				// Verify mitigation structure
				assert.NotEmpty(t, mit.ID)
				assert.NotEmpty(t, mit.Title)
				assert.NotEmpty(t, mit.Description)
				assert.NotEmpty(t, mit.Timeline)
				assert.NotEmpty(t, mit.Category)
			}
			
			for _, required := range tt.requiredPriorities {
				assert.True(t, priorities[required], 
					"Should have %s priority mitigation for %s risk", required, tt.riskLevel)
			}
		})
	}
}

func TestRiskMatrix_ComplianceFlags(t *testing.T) {
	matrix, err := NewRiskMatrix(DefaultRiskMatrixConfig())
	require.NoError(t, err)

	tests := []struct {
		name                   string
		piType                 detection.PIType
		riskLevel              RiskLevel
		industry               string
		expectedNotifiable     bool
		expectedAPRA           bool
		expectedPrivacyAct     bool
		minNotifications       int
	}{
		{
			name:               "TFN critical - all compliance triggered",
			piType:             detection.PITypeTFN,
			riskLevel:          RiskLevelCritical,
			industry:           "banking",
			expectedNotifiable: true,
			expectedAPRA:       true,
			expectedPrivacyAct: true,
			minNotifications:   2,
		},
		{
			name:               "Medicare high - privacy act triggered",
			piType:             detection.PITypeMedicare,
			riskLevel:          RiskLevelHigh,
			industry:           "healthcare",
			expectedNotifiable: true,
			expectedAPRA:       true,
			expectedPrivacyAct: true,
			minNotifications:   2,
		},
		{
			name:               "ABN medium - limited compliance",
			piType:             detection.PITypeABN,
			riskLevel:          RiskLevelMedium,
			industry:           "retail",
			expectedNotifiable: false,
			expectedAPRA:       true,
			expectedPrivacyAct: false, // ABN is business data
			minNotifications:   1,
		},
		{
			name:               "Email low - no compliance triggered",
			piType:             detection.PITypeEmail,
			riskLevel:          RiskLevelLow,
			industry:           "technology",
			expectedNotifiable: false,
			expectedAPRA:       false,
			expectedPrivacyAct: false,
			minNotifications:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := RiskAssessmentInput{
				Finding: detection.Finding{
					Type: tt.piType,
				},
				OrganizationInfo: OrganizationInfo{
					Industry: tt.industry,
				},
			}

			flags := matrix.determineComplianceFlags(input, tt.riskLevel)

			assert.Equal(t, tt.expectedNotifiable, flags.NotifiableDataBreach,
				"Notifiable data breach flag mismatch")
			assert.Equal(t, tt.expectedAPRA, flags.APRAReporting,
				"APRA reporting flag mismatch")
			assert.Equal(t, tt.expectedPrivacyAct, flags.PrivacyActBreach,
				"Privacy Act breach flag mismatch")
			assert.GreaterOrEqual(t, len(flags.RequiredNotifications), tt.minNotifications,
				"Should have minimum required notifications")
		})
	}
}

func TestRiskMatrix_MultiplicativeModel(t *testing.T) {
	// Test with multiplicative model
	multConfig := DefaultRiskMatrixConfig()
	multConfig.UseMultiplicativeModel = true
	multMatrix, err := NewRiskMatrix(multConfig)
	require.NoError(t, err)

	// Test with weighted average model
	avgConfig := DefaultRiskMatrixConfig()
	avgConfig.UseMultiplicativeModel = false
	avgMatrix, err := NewRiskMatrix(avgConfig)
	require.NoError(t, err)

	input := RiskAssessmentInput{
		Finding: detection.Finding{
			Type:      detection.PITypeTFN,
			Validated: true,
		},
		ConfidenceScore: 0.9,
		RepositoryInfo: RepositoryInfo{
			IsPublic: true,
		},
		FileContext: FileContext{
			IsProduction: true,
		},
		OrganizationInfo: OrganizationInfo{
			Regulated: true,
		},
	}

	multResult, err := multMatrix.AssessRisk(input)
	require.NoError(t, err)

	avgResult, err := avgMatrix.AssessRisk(input)
	require.NoError(t, err)

	// Both should identify high risk, but scores may differ
	assert.GreaterOrEqual(t, multResult.OverallRisk, 0.6)
	assert.GreaterOrEqual(t, avgResult.OverallRisk, 0.6)
	
	// Multiplicative model should emphasize compounding risks more
	assert.NotEqual(t, multResult.OverallRisk, avgResult.OverallRisk,
		"Different models should produce different scores")
}

func TestRiskMatrix_HistoricalData(t *testing.T) {
	matrix, err := NewRiskMatrix(DefaultRiskMatrixConfig())
	require.NoError(t, err)

	baseInput := RiskAssessmentInput{
		Finding: detection.Finding{
			Type: detection.PITypeCreditCard,
		},
		ConfidenceScore: 0.8,
		RepositoryInfo: RepositoryInfo{
			IsPublic: true,
		},
	}

	// No historical incidents
	noHistoryInput := baseInput
	noHistoryInput.HistoricalData = HistoricalData{
		PreviousIncidents: 0,
	}

	// Multiple historical incidents
	historyInput := baseInput
	historyInput.HistoricalData = HistoricalData{
		PreviousIncidents: 5,
		LastIncidentDate:  time.Now().Add(-30 * 24 * time.Hour),
	}

	noHistoryResult, err := matrix.AssessRisk(noHistoryInput)
	require.NoError(t, err)

	historyResult, err := matrix.AssessRisk(historyInput)
	require.NoError(t, err)

	// Historical incidents should increase risk
	assert.Greater(t, historyResult.OverallRisk, noHistoryResult.OverallRisk,
		"Historical incidents should increase risk score")
	// Likelihood should be higher or equal (might already be at max)
	assert.GreaterOrEqual(t, historyResult.LikelihoodScore, noHistoryResult.LikelihoodScore,
		"Historical incidents should not decrease likelihood score")
}

func TestRiskMatrix_EnvironmentContext(t *testing.T) {
	matrix, err := NewRiskMatrix(DefaultRiskMatrixConfig())
	require.NoError(t, err)

	baseInput := RiskAssessmentInput{
		Finding: detection.Finding{
			Type:      detection.PITypeBSB,
			Validated: true,
		},
		ConfidenceScore: 0.7,
	}

	// Production environment
	prodInput := baseInput
	prodInput.FileContext = FileContext{
		FilePath:     "src/prod/banking.go",
		IsProduction: true,
	}

	// Test environment
	testInput := baseInput
	testInput.FileContext = FileContext{
		FilePath: "test/banking_test.go",
		IsTest:   true,
	}

	prodResult, err := matrix.AssessRisk(prodInput)
	require.NoError(t, err)

	testResult, err := matrix.AssessRisk(testInput)
	require.NoError(t, err)

	// Production should have higher risk than test
	assert.Greater(t, prodResult.OverallRisk, testResult.OverallRisk,
		"Production environment should have higher risk than test")
}

func TestRiskMatrix_EdgeCases(t *testing.T) {
	matrix, err := NewRiskMatrix(DefaultRiskMatrixConfig())
	require.NoError(t, err)

	tests := []struct {
		name    string
		input   RiskAssessmentInput
		wantErr bool
		errMsg  string
	}{
		{
			name: "empty finding type",
			input: RiskAssessmentInput{
				Finding: detection.Finding{
					Type: "",
				},
			},
			wantErr: true,
			errMsg:  "finding type is required",
		},
		{
			name: "invalid confidence score",
			input: RiskAssessmentInput{
				Finding: detection.Finding{
					Type: detection.PITypeTFN,
				},
				ConfidenceScore: 1.5,
			},
			wantErr: true,
			errMsg:  "confidence score must be between 0 and 1",
		},
		{
			name: "minimal valid input",
			input: RiskAssessmentInput{
				Finding: detection.Finding{
					Type: detection.PITypeEmail,
				},
				ConfidenceScore: 0.5,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := matrix.AssessRisk(tt.input)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
			}
		})
	}
}

// Benchmark tests
func BenchmarkRiskMatrix_AssessRisk(b *testing.B) {
	matrix, err := NewRiskMatrix(DefaultRiskMatrixConfig())
	require.NoError(b, err)

	input := RiskAssessmentInput{
		Finding: detection.Finding{
			Type:      detection.PITypeTFN,
			Match:     "123-456-789",
			Validated: true,
		},
		ConfidenceScore: 0.9,
		RepositoryInfo: RepositoryInfo{
			IsPublic:     true,
			Stars:        100,
			Contributors: 10,
			LastCommit:   time.Now(),
		},
		FileContext: FileContext{
			FilePath:     "src/customer.go",
			IsProduction: true,
		},
		OrganizationInfo: OrganizationInfo{
			Industry:  "banking",
			Regulated: true,
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := matrix.AssessRisk(input)
		require.NoError(b, err)
	}
}