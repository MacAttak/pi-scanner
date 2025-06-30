package detection

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestFalsePositives verifies detection behavior for ambiguous patterns
// In the two-phase architecture, most patterns are detected for LLM disambiguation
// Only obviously synthetic patterns (repeated/sequential digits) are suppressed
func TestFalsePositives(t *testing.T) {
	detector := NewDetector()

	falsePositives := []struct {
		category    string
		description string
		content     string
		context     string // Additional context about why this shouldn't be detected
	}{
		// Order numbers and IDs that look like TFNs
		{
			category:    "Order Numbers",
			description: "Order number that matches TFN pattern",
			content:     "Order #123456782 has been processed",
			context:     "Order numbers often have 9 digits but aren't TFNs",
		},
		{
			category:    "Order Numbers",
			description: "Invoice number with TFN-like pattern",
			content:     "Invoice: 865432108 dated 2024-01-01",
			context:     "Invoice numbers shouldn't trigger TFN detection",
		},
		{
			category:    "Order Numbers",
			description: "Reference number with spaces",
			content:     "Reference: 123 456 782 for your records",
			context:     "Generic reference numbers aren't TFNs",
		},

		// Sequential and synthetic patterns
		{
			category:    "Sequential Numbers",
			description: "Sequential digits",
			content:     "Test data: 123456789",
			context:     "Sequential numbers are commonly used in tests",
		},
		{
			category:    "Sequential Numbers",
			description: "Repeated digits",
			content:     "Default value: 111111111",
			context:     "Repeated digits are synthetic",
		},
		{
			category:    "Sequential Numbers",
			description: "Counting pattern",
			content:     "IDs from 987654321 to 987654330",
			context:     "Sequential IDs in ranges",
		},

		// Phone numbers that might match ABN patterns
		{
			category:    "Phone Numbers",
			description: "International phone with 11 digits",
			content:     "Call us at 61412345678",
			context:     "International phone numbers can have 11 digits like ABNs",
		},
		{
			category:    "Phone Numbers",
			description: "Phone with country code",
			content:     "Phone: +61 412 345 678",
			context:     "Formatted phone numbers shouldn't match ABN",
		},

		// Timestamps and dates
		{
			category:    "Timestamps",
			description: "Unix timestamp",
			content:     "Created at: 1234567890",
			context:     "10-digit timestamps might look like Medicare numbers",
		},
		{
			category:    "Timestamps",
			description: "Date in numeric format",
			content:     "Date: 20240115123456",
			context:     "Numeric date formats shouldn't trigger detection",
		},

		// Version numbers and builds
		{
			category:    "Version Numbers",
			description: "Software version",
			content:     "Version 2.12.345.6789",
			context:     "Version numbers with dots shouldn't match",
		},
		{
			category:    "Version Numbers",
			description: "Build number",
			content:     "Build: 2023456789",
			context:     "Build numbers starting with year",
		},

		// Database IDs and UUIDs
		{
			category:    "Database IDs",
			description: "Numeric database ID",
			content:     "user_id: 5182475355",
			context:     "Database IDs are not ABNs",
		},
		{
			category:    "Database IDs",
			description: "Partial UUID",
			content:     "Session: 123456-789012",
			context:     "UUID segments shouldn't match BSB",
		},

		// Code examples and documentation
		{
			category:    "Documentation",
			description: "Example in comment",
			content:     "// Example TFN: 123-456-789",
			context:     "Documentation examples should be ignored",
		},
		{
			category:    "Documentation",
			description: "Format specification",
			content:     "TFN format is XXX-XXX-XXX where X is a digit",
			context:     "Format descriptions aren't actual PI",
		},
		{
			category:    "Documentation",
			description: "Markdown example",
			content:     "```\nExample: 123456782\n```",
			context:     "Code blocks in documentation",
		},

		// Mathematical and scientific data
		{
			category:    "Scientific Data",
			description: "Coordinates",
			content:     "Location: -33.865143, 151.209900",
			context:     "GPS coordinates with decimal places",
		},
		{
			category:    "Scientific Data",
			description: "Measurements",
			content:     "Reading: 123.456.789 units",
			context:     "Scientific measurements with dots",
		},

		// File paths and names
		{
			category:    "File Paths",
			description: "File with numeric name",
			content:     "/tmp/backup_20240628_123456789.tar.gz",
			context:     "Backup files with timestamps",
		},
		{
			category:    "File Paths",
			description: "Directory with numbers",
			content:     "Path: /var/log/2024/06/28/123456/",
			context:     "Directory structures with dates",
		},

		// Network and system identifiers
		{
			category:    "Network IDs",
			description: "Port ranges",
			content:     "Listening on ports 062001-062010",
			context:     "Port ranges that look like BSB",
		},
		{
			category:    "Network IDs",
			description: "MAC address segment",
			content:     "MAC: 00:40:28:07:70:01",
			context:     "MAC addresses have numeric segments",
		},

		// Product codes and SKUs
		{
			category:    "Product Codes",
			description: "Product SKU",
			content:     "SKU: NSW-12345678",
			context:     "Product codes with state prefixes",
		},
		{
			category:    "Product Codes",
			description: "Barcode",
			content:     "EAN: 9312345678901",
			context:     "Barcodes have many digits",
		},

		// Hash values and checksums
		{
			category:    "Hashes",
			description: "Partial hash",
			content:     "Checksum: 123456789abcdef",
			context:     "Hash values starting with numbers",
		},
		{
			category:    "Hashes",
			description: "CRC value",
			content:     "CRC32: 865432108",
			context:     "Checksum values are not TFNs",
		},

		// Test data generators
		{
			category:    "Test Data",
			description: "Lorem ipsum with numbers",
			content:     "Lorem ipsum 123456789 dolor sit amet",
			context:     "Lorem ipsum generators sometimes include numbers",
		},
		{
			category:    "Test Data",
			description: "Faker/mock data",
			content:     "mock.TFN = '123456789' // Invalid checksum",
			context:     "Mock data with explicit invalid values",
		},

		// Edge cases with prefixes/suffixes
		{
			category:    "Edge Cases",
			description: "Number with alphabetic prefix",
			content:     "ID: A123456789",
			context:     "Alphanumeric IDs shouldn't match",
		},
		{
			category:    "Edge Cases",
			description: "Number with alphabetic suffix",
			content:     "Code: 123456789B",
			context:     "Alphanumeric codes shouldn't match",
		},
		{
			category:    "Edge Cases",
			description: "Number in URL",
			content:     "https://example.com/user/123456789",
			context:     "URL segments aren't PI",
		},
	}

	for _, category := range []string{"Order Numbers", "Sequential Numbers", "Phone Numbers",
		"Timestamps", "Version Numbers", "Database IDs", "Documentation", "Scientific Data",
		"File Paths", "Network IDs", "Product Codes", "Hashes", "Test Data", "Edge Cases"} {

		t.Run(category, func(t *testing.T) {
			for _, fp := range falsePositives {
				if fp.category != category {
					continue
				}

				t.Run(fp.description, func(t *testing.T) {
					findings, err := detector.Detect(context.Background(), []byte(fp.content), "test.txt")
					require.NoError(t, err)

					// Two-phase architecture: We expect most patterns to be detected
					// Even synthetic patterns are detected but with failed validation
					if len(findings) > 0 {
						t.Logf("[%s] Detected for LLM validation: %s - %v findings",
							fp.category, fp.description, len(findings))

						// Synthetic patterns should have low confidence due to failed validation
						if fp.category == "Sequential Numbers" &&
							(fp.description == "Sequential digits" || fp.description == "Repeated digits") {
							for _, finding := range findings {
								assert.True(t, finding.Confidence <= 0.5,
									"Synthetic pattern should have low confidence: %s",
									fp.description)
							}
						}
					}
				})
			}
		})
	}
}

// TestFalsePositiveRegression tests specific false positives reported in production
func TestFalsePositiveRegression(t *testing.T) {
	detector := NewDetector()

	// These are actual false positives that were reported and should not be detected
	regressionTests := []struct {
		name            string
		content         string
		issue           string
		shouldNotDetect PIType
	}{
		{
			name:            "GitHub Issue Numbers",
			content:         "Fixed in PR #123456789",
			issue:           "GitHub PR/Issue numbers detected as TFN",
			shouldNotDetect: PITypeTFN,
		},
		{
			name:            "Jira Ticket IDs",
			content:         "PROJ-123456789",
			issue:           "Jira tickets with numeric suffixes",
			shouldNotDetect: PITypeTFN,
		},
		{
			name:            "Docker Image Tags",
			content:         "image:v2.12.3456789",
			issue:           "Version tags detected as Medicare",
			shouldNotDetect: PITypeMedicare,
		},
		{
			name:            "AWS Account IDs",
			content:         "Account: 123456789012",
			issue:           "12-digit AWS account IDs",
			shouldNotDetect: PITypeABN,
		},
		{
			name:            "Kubernetes Timestamps",
			content:         "creationTimestamp: 2024-06-28T12:34:56.789Z",
			issue:           "K8s timestamps with milliseconds",
			shouldNotDetect: PITypeTFN,
		},
	}

	for _, rt := range regressionTests {
		t.Run(rt.name, func(t *testing.T) {
			findings, err := detector.Detect(context.Background(), []byte(rt.content), "test.txt")
			require.NoError(t, err)

			for _, finding := range findings {
				assert.NotEqual(t, rt.shouldNotDetect, finding.Type,
					"Regression: %s\nContent: %s", rt.issue, rt.content)
			}
		})
	}
}

// TestContextualFalsePositives tests false positives that depend on context
func TestContextualFalsePositives(t *testing.T) {
	detector := NewDetector()

	tests := []struct {
		name     string
		content  string
		filename string
		context  string
	}{
		{
			name:     "Test file with sequential numbers",
			content:  "testTFN := '123456789'",
			filename: "user_test.go",
			context:  "Test files should have reduced risk",
		},
		{
			name:     "Mock data file",
			content:  "export const MOCK_ABN = '12345678901'",
			filename: "mocks/data.js",
			context:  "Mock directories indicate test data",
		},
		{
			name:     "Example in README",
			content:  "Example usage: --tfn 123456789",
			filename: "README.md",
			context:  "Documentation files contain examples",
		},
		{
			name:     "Config template",
			content:  "default_abn: XXXXXXXXXXXX # 11 digits",
			filename: "config.template.yml",
			context:  "Template files show format, not real data",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			findings, err := detector.Detect(context.Background(), []byte(tt.content), tt.filename)
			require.NoError(t, err)

			// Context should reduce risk level
			for _, finding := range findings {
				assert.True(t,
					finding.RiskLevel <= RiskLevelMedium,
					"High risk in test context: %s\nFile: %s",
					tt.context, tt.filename)
			}
		})
	}
}
