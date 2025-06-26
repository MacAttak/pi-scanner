package detection

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDebugSpecificNumbers tests specific edge cases to ensure proper PI type detection
// This test was added to verify fixes for:
// - "123456789" being detected as driver license instead of invalid TFN
// - "1123456701" being detected as driver license instead of rejected Medicare
func TestDebugSpecificNumbers(t *testing.T) {
	tests := []struct {
		name         string
		content      string
		expectedType PIType
		shouldDetect bool
		description  string
	}{
		{
			name:         "Invalid TFN 123456789",
			content:      `tfn := "123456789"`,
			expectedType: PITypeTFN,
			shouldDetect: true, // Should detect as TFN but with low confidence
			description:  "Should detect 123456789 as invalid TFN, not driver license",
		},
		{
			name:         "Invalid Medicare 1123456701",
			content:      `medicare := "1123456701"`,
			expectedType: PITypeMedicare,
			shouldDetect: false, // Should not detect as Medicare (starts with 1)
			description:  "Should reject 1123456701 as invalid Medicare (wrong first digit)",
		},
		{
			name:         "Valid TFN",
			content:      `tfn := "123456782"`,
			expectedType: PITypeTFN,
			shouldDetect: true,
			description:  "Should detect valid TFN with high confidence",
		},
		{
			name:         "Valid Medicare",
			content:      `medicare := "2123456701"`,
			expectedType: PITypeMedicare,
			shouldDetect: true,
			description:  "Should detect valid Medicare number",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			detector := NewDetector()
			findings, err := detector.Detect(context.Background(), []byte(tt.content), "test.go")
			require.NoError(t, err)

			if tt.shouldDetect {
				require.NotEmpty(t, findings, tt.description)

				// Find the specific PI type
				found := false
				for _, f := range findings {
					if f.Type == tt.expectedType {
						found = true
						t.Logf("Found %s: %s (confidence: %.2f, validated: %v)",
							f.Type, f.Match, f.Confidence, f.Validated)

						// Check it's not detected as driver license
						assert.Equal(t, tt.expectedType, f.Type,
							"Should be detected as %s, not %s", tt.expectedType, f.Type)
					}
				}
				assert.True(t, found, "Should find %s in results", tt.expectedType)

				// Ensure no driver license detection for these cases
				for _, f := range findings {
					if f.Type == PITypeDriverLicense {
						t.Errorf("Should not detect as driver license: %s", f.Match)
					}
				}
			} else {
				// Should not detect anything for this content
				assert.Empty(t, findings, tt.description)
			}
		})
	}
}

// TestDriverLicenseSpecificity tests that driver license pattern doesn't catch other PI types
func TestDriverLicenseSpecificity(t *testing.T) {
	detector := NewDetector()
	ctx := context.Background()

	// Numbers that should NOT be detected as driver licenses
	notDriverLicenses := []struct {
		content string
		desc    string
	}{
		{`number := "123456789"`, "9-digit sequential"},
		{`id := "1123456701"`, "10-digit starting with 1"},
		{`abn := "51824753556"`, "11-digit ABN"},
		{`phone := "0412345678"`, "Phone number"},
		{`tfn := "876543210"`, "9-digit number"},
	}

	for _, tc := range notDriverLicenses {
		findings, err := detector.Detect(ctx, []byte(tc.content), "test.go")
		require.NoError(t, err)

		for _, f := range findings {
			assert.NotEqual(t, PITypeDriverLicense, f.Type,
				"%s should not be detected as driver license", tc.desc)
		}
	}
}
