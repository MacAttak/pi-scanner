package detection

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDriverLicenseDetection(t *testing.T) {
	detector := NewDetector()
	ctx := context.Background()

	tests := []struct {
		name     string
		input    string
		expected []string
		notWant  []string // patterns that should NOT be detected
	}{
		{
			name: "NSW_formats_with_context",
			input: `
				Driver License: 12345678
				NSW License Number: AB123456
				Driver's licence: 87654321
			`,
			expected: []string{"12345678", "AB123456", "87654321"},
		},
		{
			name: "VIC_formats_with_context",
			input: `
				VIC Driver License: 12345678
				Victoria DL: 123456789
				Victorian driver's licence number: 1234567890
			`,
			expected: []string{"12345678", "123456789", "1234567890"},
		},
		{
			name: "QLD_formats_with_context",
			input: `
				QLD Driver License: 123456789
				Queensland licence number: 987654321
			`,
			expected: []string{"123456789", "987654321"},
		},
		{
			name: "SA_formats_with_context",
			input: `
				SA Driver License: A123456
				South Australia DL: B654321
			`,
			expected: []string{"A123456", "B654321"},
		},
		{
			name: "WA_formats_with_context",
			input: `
				WA Driver License: 1234567
				Western Australia licence: 7654321
			`,
			expected: []string{"1234567", "7654321"},
		},
		{
			name: "no_false_positives_for_bare_numbers",
			input: `
				Version: 1
				Count: 42
				Year: 2024
				Port: 8080
				Status: 404
				Bytes: 1024
				ID: 999999999
				Code: 200
				Build: 12345678
			`,
			expected: []string{},
			notWant:  []string{"1", "42", "2024", "8080", "404", "1024", "999999999", "200", "12345678"},
		},
		{
			name: "no_false_positives_for_version_numbers",
			input: `
				version: 1.2.3
				v2.0.0
				release-3.14.159
				build 2024.01.15
			`,
			expected: []string{},
			notWant:  []string{"1", "2", "3", "2.0.0", "3.14.159"},
		},
		{
			name: "mixed_content",
			input: `
				The driver license number is 12345678.
				Version 2.0 released in 2024.
				Error code 404 occurred.
				NSW licence: AB123456
			`,
			expected: []string{"12345678", "AB123456"},
			notWant:  []string{"2", "2024", "404"},
		},
		{
			name: "various_license_keywords",
			input: `
				driver license: 12345678
				driver licence: 87654321
				drivers license: 11111111
				driver's licence: 22222222
				DL: 33333333
				dl number: 44444444
			`,
			expected: []string{"12345678", "87654321", "11111111", "22222222", "33333333", "44444444"},
		},
		{
			name: "no_detection_without_context",
			input: `
				12345678
				87654321
				A123456
				1234567
			`,
			expected: []string{},
			notWant:  []string{"12345678", "87654321", "A123456", "1234567"},
		},
		{
			name: "package_json_identifiers",
			input: `
				"dlv5fgtps": "npm:dlv@1.1.3"
				"node_modules/dlv": {
					"version": "1.1.3",
				}
			`,
			expected: []string{},
			notWant:  []string{"dlv5fgtps", "dlv@1.1.3"},
		},
		{
			name: "test_patterns_excluded",
			input: `
				Test driver license: 11111111
				Example DL: 12345678
				Sample licence number: 00000000
			`,
			expected: []string{},
			notWant:  []string{"11111111", "12345678", "00000000"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			findings, err := detector.Detect(ctx, []byte(tt.input), "test.txt")
			assert.NoError(t, err)

			// Extract driver license findings
			var driverLicenses []string
			for _, f := range findings {
				if f.Type == PITypeDriverLicense {
					driverLicenses = append(driverLicenses, f.Match)
				}
			}

			// Check expected matches
			for _, expected := range tt.expected {
				assert.Contains(t, driverLicenses, expected,
					"Expected to find driver license: %s", expected)
			}

			// Check unwanted matches
			for _, notWant := range tt.notWant {
				assert.NotContains(t, driverLicenses, notWant,
					"Should not detect as driver license: %s", notWant)
			}

			// Verify count
			if len(tt.expected) > 0 {
				assert.Equal(t, len(tt.expected), len(driverLicenses),
					"Expected %d driver licenses, found %d", len(tt.expected), len(driverLicenses))
			}
		})
	}
}

func TestDriverLicenseGitleaksIntegration(t *testing.T) {
	// Test that gitleaks detector also respects context requirements
	gitleaksDetector, err := NewGitleaksDetector("")
	if err != nil {
		t.Skip("Gitleaks detector not available")
	}
	ctx := context.Background()

	tests := []struct {
		name     string
		input    string
		maxCount int // maximum acceptable findings
	}{
		{
			name: "bare_numbers_not_detected",
			input: `
				1
				12
				123
				1234
				12345
				123456
				1234567
				12345678
				123456789
				1234567890
			`,
			maxCount: 0,
		},
		{
			name: "version_numbers_not_detected",
			input: `
				version: 1.2.3
				v2.0.0
				3.14.159
				2024.01.15
			`,
			maxCount: 0,
		},
		{
			name: "http_status_codes_not_detected",
			input: `
				200 OK
				404 Not Found
				500 Internal Server Error
			`,
			maxCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			findings, err := gitleaksDetector.Detect(ctx, []byte(tt.input), "test.txt")
			assert.NoError(t, err)

			// Count driver license findings
			driverLicenseCount := 0
			for _, f := range findings {
				if f.Type == PITypeDriverLicense {
					driverLicenseCount++
					t.Logf("Unexpected driver license finding: %s", f.Match)
				}
			}

			assert.LessOrEqual(t, driverLicenseCount, tt.maxCount,
				"Found %d driver license findings, expected at most %d",
				driverLicenseCount, tt.maxCount)
		})
	}
}
