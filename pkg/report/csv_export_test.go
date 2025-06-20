package report

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/MacAttak/pi-scanner/pkg/detection"
	"github.com/MacAttak/pi-scanner/pkg/scoring"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCSVExporter(t *testing.T) {
	tests := []struct {
		name    string
		options []CSVExporterOption
		check   func(t *testing.T, e *CSVExporter)
	}{
		{
			name:    "default configuration",
			options: nil,
			check: func(t *testing.T, e *CSVExporter) {
				assert.False(t, e.includeContext)
				assert.False(t, e.includeMasked)
				assert.False(t, e.includeMetadata)
				assert.Equal(t, "2006-01-02 15:04:05", e.dateFormat)
			},
		},
		{
			name:    "with context option",
			options: []CSVExporterOption{WithContext()},
			check: func(t *testing.T, e *CSVExporter) {
				assert.True(t, e.includeContext)
			},
		},
		{
			name:    "with masked values option",
			options: []CSVExporterOption{WithMaskedValues()},
			check: func(t *testing.T, e *CSVExporter) {
				assert.True(t, e.includeMasked)
			},
		},
		{
			name:    "with metadata option",
			options: []CSVExporterOption{WithMetadata()},
			check: func(t *testing.T, e *CSVExporter) {
				assert.True(t, e.includeMetadata)
			},
		},
		{
			name:    "with custom date format",
			options: []CSVExporterOption{WithDateFormat("2006-01-02")},
			check: func(t *testing.T, e *CSVExporter) {
				assert.Equal(t, "2006-01-02", e.dateFormat)
			},
		},
		{
			name: "with all options",
			options: []CSVExporterOption{
				WithContext(),
				WithMaskedValues(),
				WithMetadata(),
				WithDateFormat("Jan 2, 2006"),
			},
			check: func(t *testing.T, e *CSVExporter) {
				assert.True(t, e.includeContext)
				assert.True(t, e.includeMasked)
				assert.True(t, e.includeMetadata)
				assert.Equal(t, "Jan 2, 2006", e.dateFormat)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exporter := NewCSVExporter(tt.options...)
			tt.check(t, exporter)
		})
	}
}

func TestCSVExporter_Export(t *testing.T) {
	timestamp := time.Date(2024, 1, 15, 14, 30, 0, 0, time.UTC)

	records := []CSVRecord{
		{
			Timestamp:        timestamp,
			Repository:       "test-repo",
			Branch:           "main",
			CommitHash:       "abc123",
			FilePath:         "src/customer.go",
			LineNumber:       42,
			ColumnNumber:     10,
			PIType:           "TFN",
			PITypeDisplay:    "Tax File Number",
			Match:            "123-456-789",
			MaskedMatch:      "123****89",
			Validated:        true,
			IsTestData:       false,
			ConfidenceScore:  0.95,
			RiskLevel:        "CRITICAL",
			RiskScore:        0.9,
			ImpactScore:      0.95,
			LikelihoodScore:  0.85,
			ExposureScore:    0.9,
			RiskCategory:     "IDENTITY_THEFT",
			Environment:      "production",
			APRARelevant:     true,
			PrivacyActIssue:  true,
			NotifiableBreach: true,
		},
		{
			Timestamp:        timestamp,
			Repository:       "test-repo",
			Branch:           "main",
			CommitHash:       "abc123",
			FilePath:         "test/test_data.go",
			LineNumber:       10,
			ColumnNumber:     5,
			PIType:           "MEDICARE",
			PITypeDisplay:    "Medicare Number",
			Match:            "2234567890",
			MaskedMatch:      "22******90",
			Validated:        false,
			IsTestData:       true,
			ConfidenceScore:  0.3,
			RiskLevel:        "LOW",
			RiskScore:        0.2,
			ImpactScore:      0.3,
			LikelihoodScore:  0.2,
			ExposureScore:    0.1,
			RiskCategory:     "OPERATIONAL",
			Environment:      "test",
			APRARelevant:     false,
			PrivacyActIssue:  false,
			NotifiableBreach: false,
		},
	}

	tests := []struct {
		name     string
		exporter *CSVExporter
		validate func(t *testing.T, output string)
	}{
		{
			name:     "basic export",
			exporter: NewCSVExporter(),
			validate: func(t *testing.T, output string) {
				lines := strings.Split(strings.TrimSpace(output), "\n")
				assert.GreaterOrEqual(t, len(lines), 3) // Header + 2 records

				// Check header
				assert.Contains(t, lines[0], "Timestamp")
				assert.Contains(t, lines[0], "Repository")
				assert.Contains(t, lines[0], "Risk Level")
				assert.NotContains(t, lines[0], "Masked Value")
				assert.NotContains(t, lines[0], "Code Context")

				// Check first record
				assert.Contains(t, lines[1], "2024-01-15 14:30:00")
				assert.Contains(t, lines[1], "test-repo")
				assert.Contains(t, lines[1], "CRITICAL")
				assert.Contains(t, lines[1], "0.95")
			},
		},
		{
			name:     "export with masked values",
			exporter: NewCSVExporter(WithMaskedValues()),
			validate: func(t *testing.T, output string) {
				lines := strings.Split(strings.TrimSpace(output), "\n")

				// Check header includes masked value
				assert.Contains(t, lines[0], "Masked Value")

				// Check masked values in records
				assert.Contains(t, lines[1], "123****89")
				assert.Contains(t, lines[2], "22******90")
			},
		},
		{
			name:     "export with context",
			exporter: NewCSVExporter(WithContext()),
			validate: func(t *testing.T, output string) {
				lines := strings.Split(strings.TrimSpace(output), "\n")

				// Check header includes context columns
				assert.Contains(t, lines[0], "Code Context")
				assert.Contains(t, lines[0], "Proximity Context")
			},
		},
		{
			name:     "export with metadata",
			exporter: NewCSVExporter(WithMetadata()),
			validate: func(t *testing.T, output string) {
				lines := strings.Split(strings.TrimSpace(output), "\n")

				// Check header includes metadata columns
				assert.Contains(t, lines[0], "Scan ID")
				assert.Contains(t, lines[0], "Tool Version")
			},
		},
		{
			name:     "export with custom date format",
			exporter: NewCSVExporter(WithDateFormat("2006-01-02")),
			validate: func(t *testing.T, output string) {
				lines := strings.Split(strings.TrimSpace(output), "\n")

				// Check date format in records
				assert.Contains(t, lines[1], "2024-01-15")
				assert.NotContains(t, lines[1], "14:30:00")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := tt.exporter.Export(&buf, records)
			require.NoError(t, err)

			tt.validate(t, buf.String())

			// Validate CSV format
			reader := csv.NewReader(strings.NewReader(buf.String()))
			records, err := reader.ReadAll()
			require.NoError(t, err)
			assert.Equal(t, 3, len(records)) // Header + 2 data rows
		})
	}
}

func TestCSVExporter_ExportFindings(t *testing.T) {
	findings := []detection.Finding{
		{
			Type:      detection.PITypeTFN,
			Match:     "123-456-789",
			File:      "src/customer.go",
			Line:      42,
			Column:    10,
			Validated: true,
		},
		{
			Type:      detection.PITypeMedicare,
			Match:     "2234567890",
			File:      "src/health.go",
			Line:      20,
			Column:    5,
			Validated: false,
		},
	}

	metadata := ExportMetadata{
		ScanID:       "scan-123",
		Repository:   "test-repo",
		Branch:       "main",
		CommitHash:   "abc123",
		ScanDuration: 5 * time.Minute,
		ToolVersion:  "1.0.0",
		Timestamp:    time.Now(),
	}

	exporter := NewCSVExporter(WithMaskedValues())

	var buf bytes.Buffer
	err := exporter.ExportFindings(&buf, findings, metadata)
	require.NoError(t, err)

	// Validate output
	reader := csv.NewReader(&buf)
	records, err := reader.ReadAll()
	require.NoError(t, err)

	assert.Equal(t, 3, len(records)) // Header + 2 findings

	// Check headers
	headers := records[0]
	assert.Contains(t, headers, "PI Type")
	assert.Contains(t, headers, "Masked Value")

	// Check first finding
	firstFinding := records[1]
	typeIndex := indexOf(headers, "PI Type")
	assert.Equal(t, "TFN", firstFinding[typeIndex])
}

func TestCSVSummaryExporter_ExportSummary(t *testing.T) {
	summary := ScanSummary{
		TotalFindings:  100,
		CriticalCount:  10,
		HighCount:      20,
		MediumCount:    30,
		LowCount:       40,
		ValidatedCount: 60,
		TestDataCount:  15,
	}

	metadata := ExportMetadata{
		Repository:   "test-repo",
		Branch:       "main",
		Timestamp:    time.Date(2024, 1, 15, 14, 30, 0, 0, time.UTC),
		ScanDuration: 2*time.Minute + 30*time.Second,
	}

	exporter := NewCSVSummaryExporter()

	var buf bytes.Buffer
	err := exporter.ExportSummary(&buf, summary, metadata)
	require.NoError(t, err)

	// Validate output
	reader := csv.NewReader(&buf)
	records, err := reader.ReadAll()
	require.NoError(t, err)

	// Verify structure
	assert.Greater(t, len(records), 10) // Header + multiple summary rows

	// Check headers
	assert.Equal(t, []string{"Metric", "Value", "Percentage"}, records[0])

	// Check some key metrics
	foundTotal := false
	foundCritical := false
	for _, record := range records[1:] {
		if record[0] == "Total Findings" {
			foundTotal = true
			assert.Equal(t, "100", record[1])
			assert.Equal(t, "100.0%", record[2])
		}
		if record[0] == "Critical Risk" {
			foundCritical = true
			assert.Equal(t, "10", record[1])
			assert.Equal(t, "10.0%", record[2])
		}
	}
	assert.True(t, foundTotal, "Should have Total Findings row")
	assert.True(t, foundCritical, "Should have Critical Risk row")
}

func TestCSVExporter_ConvertIntegrationRecord(t *testing.T) {
	finding := detection.Finding{
		Type:      detection.PITypeTFN,
		Match:     "123-456-789",
		File:      "src/customer.go",
		Line:      42,
		Column:    10,
		Validated: true,
	}

	riskAssessment := &scoring.RiskAssessment{
		OverallRisk:     0.9,
		RiskLevel:       scoring.RiskLevelCritical,
		ImpactScore:     0.95,
		LikelihoodScore: 0.85,
		ExposureScore:   0.9,
		RiskCategory:    scoring.RiskCategoryIdentityTheft,
		ComplianceFlags: scoring.ComplianceFlags{
			APRAReporting:        true,
			PrivacyActBreach:     true,
			NotifiableDataBreach: true,
		},
	}

	ir := IntegrationRecord{
		Finding:         finding,
		ConfidenceScore: 0.95,
		RiskAssessment:  riskAssessment,
		Environment:     "production",
		ProximityInfo:   "Near 'customer' keyword",
	}

	metadata := ExportMetadata{
		ScanID:      "scan-123",
		Repository:  "test-repo",
		Branch:      "main",
		Timestamp:   time.Now(),
		ToolVersion: "1.0.0",
	}

	exporter := NewCSVExporter()
	record := exporter.ConvertIntegrationRecord(ir, metadata)

	// Verify conversion
	assert.Equal(t, "TFN", record.PIType)
	assert.Equal(t, "Tax File Number", record.PITypeDisplay)
	assert.Equal(t, 0.95, record.ConfidenceScore)
	assert.Equal(t, "CRITICAL", record.RiskLevel)
	assert.Equal(t, 0.9, record.RiskScore)
	assert.Equal(t, "IDENTITY_THEFT", record.RiskCategory)
	assert.Equal(t, "production", record.Environment)
	assert.Equal(t, "Near 'customer' keyword", record.ProximityContext)
	assert.True(t, record.APRARelevant)
	assert.True(t, record.PrivacyActIssue)
	assert.True(t, record.NotifiableBreach)
	assert.False(t, record.IsTestData)
}

func TestCSVExporter_TestDataDetection(t *testing.T) {
	tests := []struct {
		name         string
		environment  string
		expectedTest bool
	}{
		{
			name:         "test environment",
			environment:  "test",
			expectedTest: true,
		},
		{
			name:         "mock environment",
			environment:  "mock",
			expectedTest: true,
		},
		{
			name:         "production environment",
			environment:  "production",
			expectedTest: false,
		},
		{
			name:         "unknown environment",
			environment:  "unknown",
			expectedTest: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ir := IntegrationRecord{
				Finding:     detection.Finding{Type: detection.PITypeTFN},
				Environment: tt.environment,
			}

			exporter := NewCSVExporter()
			record := exporter.ConvertIntegrationRecord(ir, ExportMetadata{})

			assert.Equal(t, tt.expectedTest, record.IsTestData)
		})
	}
}

func TestGetPITypeDisplay(t *testing.T) {
	tests := []struct {
		piType   detection.PIType
		expected string
	}{
		{detection.PITypeTFN, "Tax File Number"},
		{detection.PITypeMedicare, "Medicare Number"},
		{detection.PITypeABN, "Australian Business Number"},
		{detection.PITypeBSB, "Bank State Branch"},
		{detection.PITypeCreditCard, "Credit Card"},
		{detection.PITypeEmail, "Email Address"},
		{detection.PITypePhone, "Phone Number"},
		{detection.PITypeName, "Personal Name"},
		{detection.PITypeAddress, "Physical Address"},
		{detection.PITypePassport, "Passport Number"},
		{detection.PITypeDriverLicense, "Driver License"},
		{detection.PITypeIP, "IP Address"},
		{detection.PIType("UNKNOWN"), "UNKNOWN"},
	}

	for _, tt := range tests {
		t.Run(string(tt.piType), func(t *testing.T) {
			result := getPITypeDisplay(tt.piType)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCSVExporter_LargeExport(t *testing.T) {
	// Test with a large number of records
	records := make([]CSVRecord, 1000)
	for i := range records {
		records[i] = CSVRecord{
			Timestamp:       time.Now(),
			Repository:      "large-repo",
			FilePath:        fmt.Sprintf("src/file%d.go", i),
			LineNumber:      i + 1,
			PIType:          "TFN",
			PITypeDisplay:   "Tax File Number",
			ConfidenceScore: 0.8,
			RiskLevel:       "HIGH",
		}
	}

	exporter := NewCSVExporter()
	var buf bytes.Buffer

	err := exporter.Export(&buf, records)
	require.NoError(t, err)

	// Validate output
	reader := csv.NewReader(&buf)
	allRecords, err := reader.ReadAll()
	require.NoError(t, err)

	assert.Equal(t, 1001, len(allRecords)) // Header + 1000 records
}

func TestCSVExporter_SpecialCharacters(t *testing.T) {
	// Test handling of special characters in CSV
	records := []CSVRecord{
		{
			Timestamp:   time.Now(),
			Repository:  `repo with "quotes"`,
			FilePath:    `path/with,comma.go`,
			PIType:      "TFN",
			CodeContext: "Line with\nnewline",
			Environment: `env with "special" chars`,
			RiskLevel:   "HIGH",
		},
	}

	exporter := NewCSVExporter(WithContext())
	var buf bytes.Buffer

	err := exporter.Export(&buf, records)
	require.NoError(t, err)

	// Validate CSV can be parsed correctly
	reader := csv.NewReader(&buf)
	allRecords, err := reader.ReadAll()
	require.NoError(t, err)

	assert.Equal(t, 2, len(allRecords)) // Header + 1 record

	// Check that special characters are preserved
	headers := allRecords[0]
	repoIndex := indexOf(headers, "Repository")
	pathIndex := indexOf(headers, "File Path")

	record := allRecords[1]
	assert.Contains(t, record[repoIndex], "quotes")
	assert.Contains(t, record[pathIndex], "comma")
}

func TestCSVExporter_EmptyExport(t *testing.T) {
	// Test exporting empty records
	exporter := NewCSVExporter()
	var buf bytes.Buffer

	err := exporter.Export(&buf, []CSVRecord{})
	require.NoError(t, err)

	// Should still have headers
	reader := csv.NewReader(&buf)
	records, err := reader.ReadAll()
	require.NoError(t, err)

	assert.Equal(t, 1, len(records)) // Just headers
	assert.Contains(t, records[0], "Timestamp")
}

// Helper function
func indexOf(slice []string, item string) int {
	for i, v := range slice {
		if v == item {
			return i
		}
	}
	return -1
}

// Benchmarks
func BenchmarkCSVExporter_Export(b *testing.B) {
	records := make([]CSVRecord, 100)
	for i := range records {
		records[i] = CSVRecord{
			Timestamp:       time.Now(),
			Repository:      "bench-repo",
			FilePath:        fmt.Sprintf("src/file%d.go", i),
			LineNumber:      i + 1,
			PIType:          "TFN",
			ConfidenceScore: 0.8,
			RiskLevel:       "HIGH",
		}
	}

	exporter := NewCSVExporter(WithMaskedValues(), WithContext())

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		err := exporter.Export(&buf, records)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCSVExporter_LargeExport(b *testing.B) {
	records := make([]CSVRecord, 10000)
	for i := range records {
		records[i] = CSVRecord{
			Timestamp:       time.Now(),
			Repository:      "bench-repo",
			FilePath:        fmt.Sprintf("src/file%d.go", i),
			LineNumber:      i + 1,
			PIType:          "TFN",
			ConfidenceScore: 0.8,
			RiskLevel:       "HIGH",
		}
	}

	exporter := NewCSVExporter()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		err := exporter.Export(&buf, records)
		if err != nil {
			b.Fatal(err)
		}
	}
}
