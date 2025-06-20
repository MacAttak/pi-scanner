package scoring

import (
	"testing"

	"github.com/MacAttak/pi-scanner/pkg/detection"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewFactorEngine(t *testing.T) {
	tests := []struct {
		name    string
		config  *FactorConfig
		wantErr bool
	}{
		{
			name:    "default config success",
			config:  DefaultFactorConfig(),
			wantErr: false,
		},
		{
			name:    "nil config uses default",
			config:  nil,
			wantErr: false,
		},
		{
			name: "custom weights",
			config: &FactorConfig{
				ProximityWeight:  0.4,
				MLWeight:         0.3,
				ValidationWeight: 0.3,
				EnvironmentPenalties: map[string]float64{
					"test": 0.8,
					"mock": 0.9,
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine, err := NewFactorEngine(tt.config)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, engine)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, engine)
				assert.NotNil(t, engine.config)
			}
		})
	}
}

func TestFactorEngine_CalculateProximityFactor(t *testing.T) {
	engine, err := NewFactorEngine(DefaultFactorConfig())
	require.NoError(t, err)

	tests := []struct {
		name          string
		proximityData *ProximityScore
		expected      float64
		description   string
	}{
		{
			name: "high confidence proximity with label context",
			proximityData: &ProximityScore{
				Score:    0.9,
				Context:  "label",
				Keywords: []string{"tfn", "tax", "file", "number"},
				Distance: 1,
			},
			expected:    0.9,
			description: "should maintain high score for label context",
		},
		{
			name: "form context with medium confidence",
			proximityData: &ProximityScore{
				Score:    0.7,
				Context:  "form",
				Keywords: []string{"input", "form"},
				Distance: 2,
			},
			expected:    0.7,
			description: "form context should be treated normally",
		},
		{
			name: "test context should be heavily penalized",
			proximityData: &ProximityScore{
				Score:    0.8,
				Context:  "test",
				Keywords: []string{"test", "mock", "sample"},
				Distance: 1,
			},
			expected:    0.16, // 0.8 * 0.2 (test penalty)
			description: "test context should apply heavy penalty",
		},
		{
			name: "documentation context penalty",
			proximityData: &ProximityScore{
				Score:    0.6,
				Context:  "documentation",
				Keywords: []string{"example", "docs"},
				Distance: 3,
			},
			expected:    0.3, // 0.6 * 0.5 (documentation penalty)
			description: "documentation should apply moderate penalty",
		},
		{
			name:          "nil proximity data",
			proximityData: nil,
			expected:      0.5, // default neutral score
			description:   "should return default score when no proximity data",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := engine.CalculateProximityFactor(tt.proximityData)
			assert.InDelta(t, tt.expected, result, 0.01, tt.description)
		})
	}
}

func TestFactorEngine_CalculateMLFactor(t *testing.T) {
	engine, err := NewFactorEngine(DefaultFactorConfig())
	require.NoError(t, err)

	tests := []struct {
		name        string
		mlData      *MLScore
		expected    float64
		description string
	}{
		{
			name: "high confidence valid ML prediction",
			mlData: &MLScore{
				Confidence: 0.95,
				PIType:     "TFN",
				IsValid:    true,
			},
			expected:    0.95,
			description: "high confidence valid predictions should be maintained",
		},
		{
			name: "moderate confidence valid prediction",
			mlData: &MLScore{
				Confidence: 0.75,
				PIType:     "ABN",
				IsValid:    true,
			},
			expected:    0.75,
			description: "moderate confidence should be maintained",
		},
		{
			name: "invalid ML prediction",
			mlData: &MLScore{
				Confidence: 0.8,
				PIType:     "TFN",
				IsValid:    false,
			},
			expected:    0.16, // 0.8 * 0.2 (invalid prediction penalty)
			description: "invalid ML predictions should be heavily penalized",
		},
		{
			name: "low confidence prediction",
			mlData: &MLScore{
				Confidence: 0.3,
				PIType:     "MEDICARE",
				IsValid:    true,
			},
			expected:    0.3,
			description: "low confidence should be maintained (already low)",
		},
		{
			name:        "nil ML data",
			mlData:      nil,
			expected:    0.5, // default neutral score
			description: "should return default when no ML data",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := engine.CalculateMLFactor(tt.mlData)
			assert.InDelta(t, tt.expected, result, 0.01, tt.description)
		})
	}
}

func TestFactorEngine_CalculateValidationFactor(t *testing.T) {
	engine, err := NewFactorEngine(DefaultFactorConfig())
	require.NoError(t, err)

	tests := []struct {
		name           string
		validationData *ValidationScore
		expected       float64
		description    string
	}{
		{
			name: "valid TFN checksum",
			validationData: &ValidationScore{
				IsValid:    true,
				Algorithm:  "TFN_CHECKSUM",
				Confidence: 1.0,
			},
			expected:    1.0,
			description: "valid checksum should give maximum confidence",
		},
		{
			name: "invalid TFN checksum",
			validationData: &ValidationScore{
				IsValid:    false,
				Algorithm:  "TFN_CHECKSUM",
				Confidence: 0.0,
			},
			expected:    0.0,
			description: "invalid checksum should give zero confidence",
		},
		{
			name: "valid ABN modulus",
			validationData: &ValidationScore{
				IsValid:    true,
				Algorithm:  "ABN_MODULUS_89",
				Confidence: 1.0,
			},
			expected:    1.0,
			description: "valid ABN should give maximum confidence",
		},
		{
			name: "valid Medicare checksum",
			validationData: &ValidationScore{
				IsValid:    true,
				Algorithm:  "MEDICARE_CHECKSUM",
				Confidence: 1.0,
			},
			expected:    1.0,
			description: "valid Medicare should give maximum confidence",
		},
		{
			name: "valid BSB format",
			validationData: &ValidationScore{
				IsValid:    true,
				Algorithm:  "BSB_FORMAT",
				Confidence: 0.9, // Format validation is slightly less certain than checksums
			},
			expected:    0.9,
			description: "BSB format validation should reflect confidence level",
		},
		{
			name:           "nil validation data",
			validationData: nil,
			expected:       0.0,
			description:    "should return zero when no validation data",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := engine.CalculateValidationFactor(tt.validationData)
			assert.InDelta(t, tt.expected, result, 0.01, tt.description)
		})
	}
}

func TestFactorEngine_CalculateEnvironmentFactor(t *testing.T) {
	engine, err := NewFactorEngine(DefaultFactorConfig())
	require.NoError(t, err)

	tests := []struct {
		name        string
		filename    string
		content     string
		expected    float64
		description string
	}{
		{
			name:        "production file path",
			filename:    "src/prod/customer.go",
			content:     "func processCustomer() {\n    // Production code\n}",
			expected:    1.2, // production boost
			description: "production paths should boost confidence",
		},
		{
			name:        "test file path",
			filename:    "src/customer_test.go",
			content:     "func TestCustomer() {\n    // Test code\n}",
			expected:    0.2, // test penalty
			description: "test files should heavily penalize confidence",
		},
		{
			name:        "mock file path",
			filename:    "test/mocks/customer_mock.go",
			content:     "// Mock implementation",
			expected:    0.02, // test * mock penalty (0.2 * 0.1)
			description: "mock files should heavily penalize confidence",
		},
		{
			name:        "fixture data",
			filename:    "test/fixtures/sample_data.json",
			content:     "{\n    \"sample\": \"data\"\n}",
			expected:    0.004, // test * fixture * sample penalty (0.2 * 0.1 * 0.2)
			description: "fixture files should heavily penalize confidence",
		},
		{
			name:        "documentation file",
			filename:    "docs/api.md",
			content:     "# API Documentation\nExample TFN: 123-456-789",
			expected:    1.0, // Only .md extension but no penalty keywords detected
			description: "documentation should moderately penalize confidence",
		},
		{
			name:        "regular source file",
			filename:    "src/customer.go",
			content:     "func processCustomer() {\n    // Regular code\n}",
			expected:    1.0, // neutral
			description: "regular source files should be neutral",
		},
		{
			name:        "config file with prod indicators",
			filename:    "config/production.yaml",
			content:     "env: production\ndatabase:\n  host: prod-db",
			expected:    1.2, // production boost
			description: "production config should slightly boost confidence",
		},
		{
			name:        "test content keywords",
			filename:    "src/service.go",
			content:     "// This is sample data for testing\nvar testTFN = \"123-456-789\"",
			expected:    0.04, // sample * testing keyword penalty (0.2 * 0.2)
			description: "test keywords in content should penalize confidence",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := engine.CalculateEnvironmentFactor(tt.filename, tt.content)
			assert.InDelta(t, tt.expected, result, 0.01, tt.description)
		})
	}
}

func TestFactorEngine_CalculateCoOccurrenceFactor(t *testing.T) {
	engine, err := NewFactorEngine(DefaultFactorConfig())
	require.NoError(t, err)

	tests := []struct {
		name          string
		piType        detection.PIType
		coOccurrences []CoOccurrence
		expected      float64
		description   string
	}{
		{
			name:   "TFN with Medicare - high risk combination",
			piType: detection.PITypeTFN,
			coOccurrences: []CoOccurrence{
				{
					PIType:   detection.PITypeMedicare,
					Distance: 2,
					Match:    "2234567890",
				},
			},
			expected:    1.36, // 1.4 * 0.9^1 (distance decay for distance 2)
			description: "TFN + Medicare should significantly boost risk",
		},
		{
			name:   "TFN with Name and Phone - identity cluster",
			piType: detection.PITypeTFN,
			coOccurrences: []CoOccurrence{
				{
					PIType:   detection.PITypeName,
					Distance: 1,
					Match:    "John Smith",
				},
				{
					PIType:   detection.PITypePhone,
					Distance: 3,
					Match:    "0412345678",
				},
			},
			expected:    1.51, // compound: (1 + 0.3*1.0) * (1 + 0.2*0.81) = 1.3 * 1.162
			description: "multiple identity elements should compound risk",
		},
		{
			name:   "Medicare with Name - moderate risk",
			piType: detection.PITypeMedicare,
			coOccurrences: []CoOccurrence{
				{
					PIType:   detection.PITypeName,
					Distance: 1,
					Match:    "Jane Doe",
				},
			},
			expected:    1.2, // moderate boost
			description: "Medicare + Name should moderately boost risk",
		},
		{
			name:   "BSB with Account Number - financial cluster",
			piType: detection.PITypeBSB,
			coOccurrences: []CoOccurrence{
				{
					PIType:   detection.PITypeAccount,
					Distance: 1,
					Match:    "123456789",
				},
			},
			expected:    1.3, // financial data boost
			description: "BSB + Account should boost financial risk",
		},
		{
			name:   "distant co-occurrence - reduced impact",
			piType: detection.PITypeTFN,
			coOccurrences: []CoOccurrence{
				{
					PIType:   detection.PITypeName,
					Distance: 10, // far apart
					Match:    "John Smith",
				},
			},
			expected:    1.116, // 1.3 * 0.9^9 (distance decay)
			description: "distant co-occurrences should have reduced impact",
		},
		{
			name:          "no co-occurrences",
			piType:        detection.PITypeTFN,
			coOccurrences: []CoOccurrence{},
			expected:      1.0, // neutral
			description:   "no co-occurrences should be neutral",
		},
		{
			name:   "multiple high-risk combinations",
			piType: detection.PITypeTFN,
			coOccurrences: []CoOccurrence{
				{
					PIType:   detection.PITypeMedicare,
					Distance: 1,
					Match:    "2234567890",
				},
				{
					PIType:   detection.PITypeName,
					Distance: 2,
					Match:    "John Smith",
				},
				{
					PIType:   detection.PITypeAddress,
					Distance: 3,
					Match:    "123 Main St",
				},
			},
			expected:    1.6, // high compound risk (capped)
			description: "multiple high-risk combinations should compound but be capped",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := engine.CalculateCoOccurrenceFactor(tt.piType, tt.coOccurrences)
			assert.InDelta(t, tt.expected, result, 0.01, tt.description)
		})
	}
}

func TestFactorEngine_CalculatePITypeWeight(t *testing.T) {
	engine, err := NewFactorEngine(DefaultFactorConfig())
	require.NoError(t, err)

	tests := []struct {
		name        string
		piType      detection.PIType
		expected    float64
		description string
	}{
		{
			name:        "TFN - highest priority",
			piType:      detection.PITypeTFN,
			expected:    1.0, // maximum weight
			description: "TFN should have maximum weight under Australian regulations",
		},
		{
			name:        "Medicare - high priority",
			piType:      detection.PITypeMedicare,
			expected:    0.95,
			description: "Medicare should have very high weight",
		},
		{
			name:        "Credit Card - high priority",
			piType:      detection.PITypeCreditCard,
			expected:    0.9,
			description: "Credit cards should have high weight",
		},
		{
			name:        "ABN - moderate-high priority",
			piType:      detection.PITypeABN,
			expected:    0.8,
			description: "ABN should have moderate-high weight",
		},
		{
			name:        "BSB - moderate priority",
			piType:      detection.PITypeBSB,
			expected:    0.7,
			description: "BSB should have moderate weight",
		},
		{
			name:        "Phone - lower priority",
			piType:      detection.PITypePhone,
			expected:    0.5,
			description: "Phone should have lower weight",
		},
		{
			name:        "Email - lower priority",
			piType:      detection.PITypeEmail,
			expected:    0.4,
			description: "Email should have lower weight",
		},
		{
			name:        "IP Address - lowest priority",
			piType:      detection.PITypeIP,
			expected:    0.2,
			description: "IP addresses should have lowest weight",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := engine.CalculatePITypeWeight(tt.piType)
			assert.InDelta(t, tt.expected, result, 0.01, tt.description)
		})
	}
}

func TestFactorEngine_DetectEnvironmentIndicators(t *testing.T) {
	engine, err := NewFactorEngine(DefaultFactorConfig())
	require.NoError(t, err)

	tests := []struct {
		name        string
		filename    string
		content     string
		expected    []string
		description string
	}{
		{
			name:        "production file with prod keywords",
			filename:    "src/prod/service.go",
			content:     "// Production service\nconst PROD_URL = \"https://api.prod.example.com\"",
			expected:    []string{"production", "docs"},
			description: "should detect production indicators in path and content",
		},
		{
			name:        "test file with multiple test indicators",
			filename:    "test/customer_test.go",
			content:     "func TestCustomer() {\n    mockData := \"sample\"\n    // Test implementation\n}",
			expected:    []string{"test", "sample"},
			description: "should detect test indicators in path and content",
		},
		{
			name:        "fixture file",
			filename:    "fixtures/data.json",
			content:     "{\n    \"example\": \"dummy data\"\n}",
			expected:    []string{"test", "dummy"},
			description: "should detect test indicator in path and dummy in content",
		},
		{
			name:        "documentation file",
			filename:    "docs/README.md",
			content:     "# Documentation\nExample usage:\n```\napi_key = \"demo-key\"\n```",
			expected:    []string{"test", "docs", "demo"},
			description: "should detect documentation indicators",
		},
		{
			name:        "regular source file - no indicators",
			filename:    "src/service.go",
			content:     "func processData() {\n    return data\n}",
			expected:    []string{},
			description: "regular files should have no environment indicators",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := engine.DetectEnvironmentIndicators(tt.filename, tt.content)
			assert.ElementsMatch(t, tt.expected, result, tt.description)
		})
	}
}

func TestFactorEngine_AustralianComplianceWeights(t *testing.T) {
	engine, err := NewFactorEngine(DefaultFactorConfig())
	require.NoError(t, err)

	// Test that Australian-specific PI types have appropriate weights
	australianTypes := []detection.PIType{
		detection.PITypeTFN,
		detection.PITypeMedicare,
		detection.PITypeABN,
		detection.PITypeBSB,
	}

	for _, piType := range australianTypes {
		t.Run(string(piType), func(t *testing.T) {
			weight := engine.CalculatePITypeWeight(piType)

			// Australian PI types should have high weights (>= 0.7)
			assert.GreaterOrEqual(t, weight, 0.7,
				"Australian PI type %s should have high weight for regulatory compliance", piType)

			// TFN should have the highest weight
			if piType == detection.PITypeTFN {
				assert.Equal(t, 1.0, weight, "TFN should have maximum weight")
			}
		})
	}
}

func TestFactorEngine_EdgeCases(t *testing.T) {
	engine, err := NewFactorEngine(DefaultFactorConfig())
	require.NoError(t, err)

	tests := []struct {
		name        string
		testFunc    func() float64
		expected    float64
		description string
	}{
		{
			name: "nil proximity score",
			testFunc: func() float64 {
				return engine.CalculateProximityFactor(nil)
			},
			expected:    0.5,
			description: "should return default neutral score for nil proximity",
		},
		{
			name: "nil ML score",
			testFunc: func() float64 {
				return engine.CalculateMLFactor(nil)
			},
			expected:    0.5,
			description: "should return default neutral score for nil ML data",
		},
		{
			name: "nil validation score",
			testFunc: func() float64 {
				return engine.CalculateValidationFactor(nil)
			},
			expected:    0.0,
			description: "should return zero for nil validation (no validation data available)",
		},
		{
			name: "empty filename and content",
			testFunc: func() float64 {
				return engine.CalculateEnvironmentFactor("", "")
			},
			expected:    1.0,
			description: "should return neutral score for empty inputs",
		},
		{
			name: "unknown PI type weight",
			testFunc: func() float64 {
				return engine.CalculatePITypeWeight(detection.PIType("UNKNOWN"))
			},
			expected:    0.5,
			description: "should return default weight for unknown PI types",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.testFunc()
			assert.InDelta(t, tt.expected, result, 0.01, tt.description)
		})
	}
}

// Benchmark tests for performance validation
func BenchmarkFactorEngine_CalculateProximityFactor(b *testing.B) {
	engine, err := NewFactorEngine(DefaultFactorConfig())
	require.NoError(b, err)

	proximityData := &ProximityScore{
		Score:    0.8,
		Context:  "production",
		Keywords: []string{"customer", "tfn"},
		Distance: 2,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		engine.CalculateProximityFactor(proximityData)
	}
}

func BenchmarkFactorEngine_CalculateEnvironmentFactor(b *testing.B) {
	engine, err := NewFactorEngine(DefaultFactorConfig())
	require.NoError(b, err)

	filename := "src/customer.go"
	content := "func processCustomer() {\n    tfn := \"123-456-789\"\n    // Process customer data\n}"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		engine.CalculateEnvironmentFactor(filename, content)
	}
}

func BenchmarkFactorEngine_CalculateCoOccurrenceFactor(b *testing.B) {
	engine, err := NewFactorEngine(DefaultFactorConfig())
	require.NoError(b, err)

	coOccurrences := []CoOccurrence{
		{
			PIType:   detection.PITypeMedicare,
			Distance: 2,
			Match:    "2234567890",
		},
		{
			PIType:   detection.PITypeName,
			Distance: 1,
			Match:    "John Smith",
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		engine.CalculateCoOccurrenceFactor(detection.PITypeTFN, coOccurrences)
	}
}
