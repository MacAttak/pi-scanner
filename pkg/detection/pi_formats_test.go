package detection

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestComprehensivePIFormats tests all variations of Australian PI formats
func TestComprehensivePIFormats(t *testing.T) {
	// Use gitleaks detector which has all the Australian PI rules
	detector, err := NewGitleaksDetectorAuto()
	require.NoError(t, err)

	tests := []struct {
		name         string
		piType       PIType
		validFormats []struct {
			input       string
			description string
			normalized  string // Expected normalized form
		}
		invalidFormats []struct {
			input       string
			description string
		}
	}{
		{
			name:   "Tax File Number (TFN)",
			piType: PITypeTFN,
			validFormats: []struct {
				input       string
				description string
				normalized  string
			}{
				{"123456782", "Basic 9-digit format", "123456782"},
				{"123-456-782", "Hyphenated format", "123456782"},
				{"123 456 782", "Space-separated format", "123456782"},
				{"865 432 108", "Another valid TFN", "865432108"},
				{"TFN: 123456782", "With prefix label", "123456782"},
				{"tfn=123456782", "Variable assignment", "123456782"},
				{`"123456782"`, "In quotes", "123456782"},
				{"'123456782'", "In single quotes", "123456782"},
			},
			invalidFormats: []struct {
				input       string
				description string
			}{
				{"12345678", "Too few digits (8)"},
				{"1234567890", "Too many digits (10)"},
				{"123456789", "Invalid checksum"},
				{"000000000", "All zeros"},
				{"111111111", "All ones (synthetic)"},
				{"123456789A", "Contains letter"},
				{"12-34-56-789", "Wrong formatting"},
				{"ORDER123456782", "Part of order number"},
			},
		},
		{
			name:   "Medicare Number",
			piType: PITypeMedicare,
			validFormats: []struct {
				input       string
				description string
				normalized  string
			}{
				{"2123456701", "Basic 10-digit format", "2123456701"},
				{"2123 4567 0 1", "Standard card format", "2123456701"},
				{"2123-4567-0-1", "Hyphenated format", "2123456701"},
				{"2123456701/1", "With Individual Reference Number", "2123456701"},
				{"3987654321", "Starting with 3", "3987654321"},
				{"4123456789", "Starting with 4", "4123456789"},
				{"5234567890", "Starting with 5", "5234567890"},
				{"6345678901", "Starting with 6", "6345678901"},
			},
			invalidFormats: []struct {
				input       string
				description string
			}{
				{"1123456789", "Invalid first digit (1)"},
				{"7123456789", "Invalid first digit (7)"},
				{"212345678", "Too few digits (9)"},
				{"21234567890", "Too many digits (11)"},
				{"2123456789", "Invalid checksum"},
				{"0123456789", "Starting with 0"},
				{"9123456789", "Starting with 9"},
				{"212345678A", "Contains letter"},
			},
		},
		{
			name:   "Australian Business Number (ABN)",
			piType: PITypeABN,
			validFormats: []struct {
				input       string
				description string
				normalized  string
			}{
				{"51824753556", "Basic 11-digit format", "51824753556"},
				{"51 824 753 556", "Space-separated format", "51824753556"},
				{"51-824-753-556", "Hyphenated format", "51824753556"},
				{"ABN: 51 824 753 556", "With prefix", "51824753556"},
				{"88952560394", "Another valid ABN", "88952560394"},
				{"20509179503", "Valid ABN", "20509179503"},
			},
			invalidFormats: []struct {
				input       string
				description string
			}{
				{"5182475355", "Too few digits (10)"},
				{"518247535566", "Too many digits (12)"},
				{"51824753557", "Invalid modulus 89 check"},
				{"00000000000", "All zeros"},
				{"11111111111", "All ones"},
				{"5182475355A", "Contains letter"},
				{"10824753556", "Invalid first digits"},
			},
		},
		{
			name:   "Bank State Branch (BSB)",
			piType: PITypeBSB,
			validFormats: []struct {
				input       string
				description string
				normalized  string
			}{
				{"062-001", "Standard format", "062001"},
				{"062001", "No separator", "062001"},
				{"062 001", "Space separator", "062001"},
				{"123-456", "Valid BSB", "123456"},
				{"BSB: 062-001", "With prefix", "062001"},
				{"bsb=062001", "Variable assignment", "062001"},
			},
			invalidFormats: []struct {
				input       string
				description string
			}{
				{"06-2001", "Wrong format"},
				{"0620-01", "Wrong format"},
				{"062-0001", "Too many digits"},
				{"06-001", "Too few digits"},
				{"A62-001", "Contains letter"},
				{"062-00A", "Contains letter"},
				{"000-000", "All zeros (might be valid?)"},
			},
		},
		{
			name:   "Australian Company Number (ACN)",
			piType: PITypeACN,
			validFormats: []struct {
				input       string
				description string
				normalized  string
			}{
				{"004028077", "Basic 9-digit format", "004028077"},
				{"004 028 077", "Space-separated", "004028077"},
				{"004-028-077", "Hyphenated", "004028077"},
				{"ACN 004 028 077", "With prefix", "004028077"},
				{"123456785", "Valid ACN", "123456785"},
			},
			invalidFormats: []struct {
				input       string
				description string
			}{
				{"00402807", "Too few digits (8)"},
				{"0040280770", "Too many digits (10)"},
				{"004028078", "Invalid check digit"},
				{"000000000", "All zeros"},
				{"00402807A", "Contains letter"},
			},
		},
		{
			name:   "Driver's License",
			piType: PITypeDriverLicense,
			validFormats: []struct {
				input       string
				description string
				normalized  string
			}{
				// NSW (8 digits or alphanumeric)
				{"12345678", "NSW numeric format", "12345678"},
				{"NSW License: 12345678", "NSW with prefix", "12345678"},

				// VIC (up to 10 digits)
				{"123456789", "VIC 9-digit format", "123456789"},
				{"1234567890", "VIC 10-digit format", "1234567890"},

				// QLD (9 digits)
				{"123456789", "QLD format", "123456789"},

				// WA (7 characters)
				{"1234567", "WA format", "1234567"},
			},
			invalidFormats: []struct {
				input       string
				description string
			}{
				{"123", "Too short"},
				{"12345678901", "Too long (11 digits)"},
				{"LICENSE123", "Wrong prefix format"},
			},
		},
		{
			name:   "Bank Account",
			piType: PITypeBankAccount,
			validFormats: []struct {
				input       string
				description string
				normalized  string
			}{
				{"account 12345678", "With account keyword", ""},
				{"Account: 123456789", "With Account prefix", ""},
				{"account=1234567890", "Variable assignment", ""},
				{"bank account 12345678", "With bank account", ""},
				{"acct: 123456789", "With acct abbreviation", ""},
			},
			invalidFormats: []struct {
				input       string
				description string
			}{
				{"12345678", "Standalone number without context"},
				{"account 12345", "Too short (5 digits)"},
				{"account 12345678901", "Too long (11 digits)"},
				{"account 1234567A", "Contains letter"},
				{"12-345-678", "Formatted without keyword"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test valid formats
			for _, valid := range tt.validFormats {
				t.Run(fmt.Sprintf("Valid_%s", valid.description), func(t *testing.T) {
					content := fmt.Sprintf("test content with %s in it", valid.input)
					findings, err := detector.Detect(context.Background(), []byte(content), "test.txt")
					require.NoError(t, err)

					// Should detect at least one finding of the correct type
					found := false
					for _, finding := range findings {
						if finding.Type == tt.piType {
							found = true
							// Verify normalization if applicable
							if valid.normalized != "" {
								normalized := normalizePI(finding.Match)
								assert.Equal(t, valid.normalized, normalized,
									"Expected normalized value %s but got %s",
									valid.normalized, normalized)
							}
							break
						}
					}

					assert.True(t, found, "Failed to detect %s in format: %s",
						tt.piType, valid.input)
				})
			}

			// Test invalid formats
			for _, invalid := range tt.invalidFormats {
				t.Run(fmt.Sprintf("Invalid_%s", invalid.description), func(t *testing.T) {
					content := fmt.Sprintf("test content with %s in it", invalid.input)
					findings, err := detector.Detect(context.Background(), []byte(content), "test.txt")
					require.NoError(t, err)

					// Should not detect as this PI type
					for _, finding := range findings {
						if finding.Type == tt.piType && finding.Match == invalid.input {
							t.Errorf("Incorrectly detected %s as %s: %s",
								invalid.input, tt.piType, invalid.description)
						}
					}
				})
			}
		})
	}
}

// TestPIFormatEdgeCases tests edge cases and boundary conditions
func TestPIFormatEdgeCases(t *testing.T) {
	detector := NewDetector()

	tests := []struct {
		name        string
		content     string
		shouldFind  map[PIType]int // Expected count for each PI type
		description string
	}{
		{
			name:    "Multiple PI in single line",
			content: "Customer TFN: 123456782, ABN: 51824753556, Medicare: 2123456701",
			shouldFind: map[PIType]int{
				PITypeTFN:      1,
				PITypeABN:      1,
				PITypeMedicare: 1,
			},
			description: "Should detect all three PI types in one line",
		},
		{
			name: "PI at line boundaries",
			content: `Start of line 123456782
51824753556 end of line
Middle has 2123456701 here`,
			shouldFind: map[PIType]int{
				PITypeTFN:      1,
				PITypeABN:      1,
				PITypeMedicare: 1,
			},
			description: "Should detect PI at start, end, and middle of lines",
		},
		{
			name:    "PI with mixed case context",
			content: "CUSTOMER TFN: 123456782, abn: 51824753556, MeDiCaRe: 2123456701",
			shouldFind: map[PIType]int{
				PITypeTFN:      1,
				PITypeABN:      1,
				PITypeMedicare: 1,
			},
			description: "Should be case-insensitive for context",
		},
		{
			name: "PI in various data structures",
			content: `{
				"tfn": "123456782",
				"abn": [51824753556],
				"medicare": '2123456701'
			}`,
			shouldFind: map[PIType]int{
				PITypeTFN:      1,
				PITypeABN:      1,
				PITypeMedicare: 1,
			},
			description: "Should detect PI in JSON-like structures",
		},
		{
			name:    "PI with special characters nearby",
			content: "(TFN:123456782) {ABN=51824753556} [Medicare#2123456701]",
			shouldFind: map[PIType]int{
				PITypeTFN:      1,
				PITypeABN:      1,
				PITypeMedicare: 1,
			},
			description: "Should detect PI surrounded by special characters",
		},
		{
			name:    "Overlapping patterns",
			content: "ID: 123456782 might look like TFN but also like an account",
			shouldFind: map[PIType]int{
				PITypeTFN: 1,
				// Might also detect as potential account number
			},
			description: "Should handle overlapping pattern matches",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			findings, err := detector.Detect(context.Background(), []byte(tt.content), "test.txt")
			require.NoError(t, err)

			// Count findings by type
			foundCounts := make(map[PIType]int)
			for _, finding := range findings {
				foundCounts[finding.Type]++
			}

			// Verify expected counts
			for piType, expectedCount := range tt.shouldFind {
				actualCount := foundCounts[piType]
				assert.Equal(t, expectedCount, actualCount,
					"%s: expected %d %s findings, got %d",
					tt.description, expectedCount, piType, actualCount)
			}
		})
	}
}

// normalizePI removes formatting from PI values for comparison
func normalizePI(value string) string {
	// Remove common separators
	normalized := value
	for _, sep := range []string{"-", " ", "/", "(", ")", ".", ","} {
		normalized = strings.ReplaceAll(normalized, sep, "")
	}
	return strings.TrimSpace(normalized)
}
