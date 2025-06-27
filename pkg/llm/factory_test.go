package llm

import (
	"context"
	"testing"

	"github.com/MacAttak/pi-scanner/pkg/detection"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewValidatorFromConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  *detection.Config
		wantNil bool
		wantErr bool
	}{
		{
			name: "disabled returns nil",
			config: &detection.Config{
				EnableLLMValidation: false,
			},
			wantNil: true,
			wantErr: false,
		},
		{
			name: "lmstudio provider",
			config: &detection.Config{
				EnableLLMValidation: true,
				LLMProvider:         "lmstudio",
				LLMEndpoint:         "http://localhost:1234/v1",
				LLMModel:            "test-model",
				LLMMaxTokens:        1000,
				LLMTemperature:      0.7,
			},
			wantNil: false,
			wantErr: false,
		},
		{
			name: "empty provider defaults to lmstudio",
			config: &detection.Config{
				EnableLLMValidation: true,
				LLMProvider:         "",
				LLMEndpoint:         "http://localhost:1234/v1",
				LLMModel:            "test-model",
				LLMMaxTokens:        1000,
				LLMTemperature:      0.7,
			},
			wantNil: false,
			wantErr: false,
		},
		{
			name: "unsupported provider",
			config: &detection.Config{
				EnableLLMValidation: true,
				LLMProvider:         "unsupported",
			},
			wantNil: true,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator, err := NewValidatorFromConfig(tt.config)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			if tt.wantNil {
				assert.Nil(t, validator)
			} else {
				assert.NotNil(t, validator)
			}
		})
	}
}

func TestCreateEnhancedDetector(t *testing.T) {
	// Mock base detector
	baseDetector := &mockDetector{name: "base"}

	tests := []struct {
		name     string
		config   *detection.Config
		wantType string
	}{
		{
			name: "disabled returns base detector",
			config: &detection.Config{
				EnableLLMValidation: false,
			},
			wantType: "base",
		},
		{
			name: "enabled returns enhanced detector",
			config: &detection.Config{
				EnableLLMValidation: true,
				LLMProvider:         "lmstudio",
				LLMEndpoint:         "http://localhost:1234/v1",
				LLMModel:            "test-model",
				LLMMaxTokens:        1000,
				LLMTemperature:      0.7,
				LLMValidateRisks:    []detection.RiskLevel{detection.RiskLevelHigh},
			},
			wantType: "base-llm-enhanced",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			detector, err := CreateEnhancedDetector(baseDetector, tt.config)
			require.NoError(t, err)
			assert.NotNil(t, detector)
			assert.Equal(t, tt.wantType, detector.Name())
		})
	}
}

// mockDetector for testing
type mockDetector struct {
	name string
}

func (m *mockDetector) Detect(ctx context.Context, content []byte, filename string) ([]detection.Finding, error) {
	return []detection.Finding{}, nil
}

func (m *mockDetector) Name() string {
	return m.name
}
