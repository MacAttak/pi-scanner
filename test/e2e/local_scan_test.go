package e2e

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/MacAttak/pi-scanner/pkg/detection"
	"github.com/MacAttak/pi-scanner/pkg/discovery"
	"github.com/MacAttak/pi-scanner/pkg/output"
	"github.com/MacAttak/pi-scanner/pkg/processing"
	"github.com/MacAttak/pi-scanner/pkg/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestLocalScan tests scanning local test data
func TestLocalScan(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Get test data directory
	testDataDir := filepath.Join("..", "testdata", "repos", "test-pi-data")
	absPath, err := filepath.Abs(testDataDir)
	require.NoError(t, err)

	// Verify test data exists
	_, err = os.Stat(filepath.Join(absPath, "test_data.go"))
	require.NoError(t, err, "Test data file should exist")

	// Configure detection
	detectorConfig := &detection.Config{
		EnableContextValidation: true,
		ContextLines:            3,
		MinConfidence:           0.5, // Lower for test data
		EnableLLMValidation:     false,
	}

	// Create components
	detector := detection.NewDetectorWithConfig(detectorConfig)

	discoverer := discovery.NewFileDiscoverer(discovery.Config{
		SkipBinary:  true,
		SkipDirs:    []string{".git"},
		MaxFileSize: 1024 * 1024,
	})

	outputManager := output.NewManager(&output.Config{
		MaskingLevel:    output.MaskingLevelPartial,
		EnableAudit:     true,
		ValidateOutputs: true,
	})

	processor := processing.NewFileProcessor(detector, processing.ProcessorConfig{
		MaxWorkers:      2,
		MaxMemoryMB:     100,
		StreamThreshold: 500 * 1024,
	})

	// Discover files
	files, err := discoverer.DiscoverFiles(ctx, absPath)
	require.NoError(t, err)
	assert.NotEmpty(t, files, "Should find test files")

	// Process files
	var findings []detection.Finding
	for _, file := range files {
		result, err := processor.ProcessFile(ctx, file)
		if err != nil {
			t.Logf("Error processing %s: %v", file.Path, err)
			continue
		}

		for _, finding := range result.Findings {
			// Apply masking
			masked, err := outputManager.MaskFinding(&finding)
			require.NoError(t, err)
			findings = append(findings, *masked)
		}
	}

	// Verify findings
	require.NotEmpty(t, findings, "Should find PI patterns in test data")

	// Check that we found expected PI types
	foundTypes := make(map[detection.PIType]bool)
	for _, f := range findings {
		foundTypes[f.Type] = true

		// Verify masking is applied
		assert.NotEqual(t, f.Match, f.MatchDisplay, "Match should be masked")
		assert.Contains(t, f.MatchDisplay, "*", "Masked value should contain asterisks")
	}

	// Verify we found major PI types
	expectedTypes := []detection.PIType{
		detection.PITypeTFN,
		detection.PITypeABN,
		detection.PITypeMedicare,
		detection.PITypeBSB,
	}

	for _, piType := range expectedTypes {
		assert.True(t, foundTypes[piType], "Should find %s in test data", piType)
	}

	t.Logf("Found %d PI instances across %d types", len(findings), len(foundTypes))
}

// TestMaskingIntegrity verifies masking works correctly
func TestMaskingIntegrity(t *testing.T) {
	testCases := []struct {
		name         string
		piType       detection.PIType
		value        string
		maskingLevel output.MaskingLevel
		expected     string
	}{
		{
			name:         "TFN partial masking",
			piType:       detection.PITypeTFN,
			value:        "123456782",
			maskingLevel: output.MaskingLevelPartial,
			expected:     "123****82",
		},
		{
			name:         "TFN full masking",
			piType:       detection.PITypeTFN,
			value:        "123456782",
			maskingLevel: output.MaskingLevelFull,
			expected:     "*********",
		},
		{
			name:         "ABN partial masking",
			piType:       detection.PITypeABN,
			value:        "51824753556",
			maskingLevel: output.MaskingLevelPartial,
			expected:     "518*****556",
		},
		{
			name:         "Medicare partial masking",
			piType:       detection.PITypeMedicare,
			value:        "2123456701",
			maskingLevel: output.MaskingLevelPartial,
			expected:     "212*****01",
		},
		{
			name:         "Email partial masking",
			piType:       detection.PITypeEmail,
			value:        "john.doe@example.com",
			maskingLevel: output.MaskingLevelPartial,
			expected:     "jo***@example.com",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			outputManager := output.NewManager(&output.Config{
				MaskingLevel: tc.maskingLevel,
			})

			finding := &detection.Finding{
				Type:  tc.piType,
				Match: tc.value,
			}

			masked, err := outputManager.MaskFinding(finding)
			require.NoError(t, err)
			assert.Equal(t, tc.expected, masked.MatchDisplay)
		})
	}
}

// TestReportSecurityE2E verifies reports don't contain unmasked PI
func TestReportSecurityE2E(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()

	// Create test finding with sensitive data
	findings := []detection.Finding{
		{
			Type:      detection.PITypeTFN,
			Match:     "123456782",
			Line:      10,
			Column:    15,
			FilePath:  "test.go",
			RiskLevel: detection.RiskLevelHigh,
			Context:   "var tfn = '123456782' // customer TFN",
		},
		{
			Type:      detection.PITypeABN,
			Match:     "51824753556",
			Line:      20,
			Column:    10,
			FilePath:  "config.go",
			RiskLevel: detection.RiskLevelMedium,
			Context:   "DefaultABN: '51824753556'",
		},
	}

	// Test with different masking levels
	maskingLevels := []output.MaskingLevel{
		output.MaskingLevelFull,
		output.MaskingLevelPartial,
	}

	for _, level := range maskingLevels {
		t.Run(string(level), func(t *testing.T) {
			outputManager := output.NewManager(&output.Config{
				MaskingLevel:    level,
				ValidateOutputs: true,
			})

			// Mask findings
			maskedFindings := make([]detection.Finding, len(findings))
			for i, f := range findings {
				masked, err := outputManager.MaskFinding(&f)
				require.NoError(t, err)
				maskedFindings[i] = *masked
			}

			// Verify no unmasked values
			for i, masked := range maskedFindings {
				original := findings[i]

				// Match should be masked
				assert.NotEqual(t, original.Match, masked.MatchDisplay)
				assert.NotContains(t, masked.MatchDisplay, original.Match)

				// Context should be masked too
				assert.NotContains(t, masked.Context, original.Match)
			}
		})
	}
}
