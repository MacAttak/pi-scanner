package report

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/MacAttak/pi-scanner/pkg/detection"
	"github.com/MacAttak/pi-scanner/pkg/scoring"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSARIFExporter(t *testing.T) {
	exporter := NewSARIFExporter("PI Scanner", "1.0.0", "https://github.com/MacAttak/pi-scanner")

	assert.NotNil(t, exporter)
	assert.Equal(t, "PI Scanner", exporter.toolName)
	assert.Equal(t, "1.0.0", exporter.toolVersion)
	assert.Equal(t, "https://github.com/MacAttak/pi-scanner", exporter.infoURI)
	assert.Empty(t, exporter.baseURI)
}

func TestSARIFExporter_SetBaseURI(t *testing.T) {
	exporter := NewSARIFExporter("PI Scanner", "1.0.0", "")
	exporter.SetBaseURI("/home/user/repo")

	assert.Equal(t, "/home/user/repo", exporter.baseURI)
}

func TestSARIFExporter_Export(t *testing.T) {
	findings := []detection.Finding{
		{
			Type:      detection.PITypeTFN,
			Match:     "123-456-789",
			File:      "src/customer.go",
			Line:      42,
			Column:    10,
			Validated: true,
			Context:   `customerTFN := "123-456-789"`,
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
		ScanDuration: 2 * time.Minute,
		ToolVersion:  "1.0.0",
		Timestamp:    time.Date(2024, 1, 15, 14, 30, 0, 0, time.UTC),
	}

	exporter := NewSARIFExporter("PI Scanner", "1.0.0", "https://github.com/MacAttak/pi-scanner")

	var buf bytes.Buffer
	err := exporter.Export(&buf, findings, metadata)
	require.NoError(t, err)

	// Parse the output
	var report SARIFReport
	err = json.Unmarshal(buf.Bytes(), &report)
	require.NoError(t, err)

	// Validate structure
	assert.Equal(t, SARIFVersion, report.Version)
	assert.Equal(t, SARIFSchema, report.Schema)
	assert.Len(t, report.Runs, 1)

	run := report.Runs[0]

	// Check tool information
	assert.Equal(t, "PI Scanner", run.Tool.Driver.Name)
	assert.Equal(t, "1.0.0", run.Tool.Driver.Version)
	assert.NotEmpty(t, run.Tool.Driver.Rules)

	// Check invocation
	assert.Len(t, run.Invocations, 1)
	assert.Equal(t, "2024-01-15T14:30:00Z", run.Invocations[0].StartTimeUTC)
	assert.True(t, run.Invocations[0].ExecutionSuccessful)

	// Check results
	assert.Len(t, run.Results, 2)

	// First result (TFN)
	result1 := run.Results[0]
	assert.Equal(t, "PI001", result1.RuleID)
	assert.Equal(t, "error", result1.Level)
	assert.Contains(t, result1.Message.Text, "Tax File Number")
	assert.Contains(t, result1.Message.Text, "123****89")
	assert.Len(t, result1.Locations, 1)
	assert.Equal(t, "src/customer.go", result1.Locations[0].PhysicalLocation.ArtifactLocation.URI)
	assert.Equal(t, 42, result1.Locations[0].PhysicalLocation.Region.StartLine)
	assert.Equal(t, 10, result1.Locations[0].PhysicalLocation.Region.StartColumn)
	assert.NotNil(t, result1.Locations[0].PhysicalLocation.Region.Snippet)
	assert.Contains(t, result1.Locations[0].PhysicalLocation.Region.Snippet.Text, "customerTFN")

	// Check properties
	props, ok := result1.Properties["validated"].(bool)
	assert.True(t, ok)
	assert.True(t, props)
}

func TestSARIFExporter_ExportWithRiskAssessment(t *testing.T) {
	findings := []IntegrationRecord{
		{
			Finding: detection.Finding{
				Type:      detection.PITypeTFN,
				Match:     "123-456-789",
				File:      "src/customer.go",
				Line:      42,
				Column:    10,
				Validated: true,
			},
			ConfidenceScore: 0.95,
			Environment:     "production",
			ProximityInfo:   "Near 'customer' keyword",
			RiskAssessment: &scoring.RiskAssessment{
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
				Mitigations: []scoring.Mitigation{
					{
						ID:          "mit-1",
						Title:       "Remove hardcoded TFN",
						Description: "Move TFN to secure environment variables",
						Priority:    "CRITICAL",
						Effort:      "Low",
						Timeline:    "Immediate",
						Category:    "REMEDIATION",
					},
				},
			},
		},
	}

	metadata := ExportMetadata{
		ScanID:       "scan-123",
		Repository:   "test-repo",
		Branch:       "main",
		CommitHash:   "abc123",
		ScanDuration: 2 * time.Minute,
		ToolVersion:  "1.0.0",
		Timestamp:    time.Now(),
	}

	exporter := NewSARIFExporter("PI Scanner", "1.0.0", "https://github.com/MacAttak/pi-scanner")

	var buf bytes.Buffer
	err := exporter.ExportWithRiskAssessment(&buf, findings, metadata)
	require.NoError(t, err)

	// Parse the output
	var report SARIFReport
	err = json.Unmarshal(buf.Bytes(), &report)
	require.NoError(t, err)

	// Check enhanced result
	result := report.Runs[0].Results[0]

	// Check risk assessment in properties
	riskAssessment, ok := result.Properties["riskAssessment"].(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, 0.9, riskAssessment["overallRisk"])
	assert.Equal(t, "CRITICAL", riskAssessment["riskLevel"])
	assert.Equal(t, "IDENTITY_THEFT", riskAssessment["riskCategory"])

	// Check compliance
	compliance, ok := result.Properties["compliance"].(map[string]interface{})
	assert.True(t, ok)
	assert.True(t, compliance["apraReporting"].(bool))
	assert.True(t, compliance["privacyActBreach"].(bool))
	assert.True(t, compliance["notifiableDataBreach"].(bool))

	// Check fixes were added
	assert.Len(t, result.Fixes, 1)
	fix := result.Fixes[0]
	assert.Contains(t, fix.Description.Text, "Remove hardcoded TFN")
	assert.Len(t, fix.ArtifactChanges, 1)
	assert.Equal(t, "src/customer.go", fix.ArtifactChanges[0].ArtifactLocation.URI)

	// Check additional properties
	assert.Equal(t, 0.95, result.Properties["confidenceScore"])
	assert.Equal(t, "production", result.Properties["environment"])
	assert.Equal(t, "Near 'customer' keyword", result.Properties["proximityInfo"])

	// Check rank was set
	assert.Equal(t, 90.0, result.Rank)
}

func TestSARIFExporter_CreateRules(t *testing.T) {
	exporter := NewSARIFExporter("PI Scanner", "1.0.0", "")
	rules := exporter.createRules()

	assert.Len(t, rules, 12) // 12 PI types

	// Check TFN rule
	tfnRule := rules[0]
	assert.Equal(t, "PI001", tfnRule.ID)
	assert.Equal(t, "Tax File Number (TFN)", tfnRule.Name)
	assert.Contains(t, tfnRule.ShortDescription.Text, "Tax File Number")
	assert.Contains(t, tfnRule.Help.Text, "TFNs are highly sensitive")
	assert.True(t, tfnRule.DefaultConfiguration.Enabled)
	assert.Equal(t, "error", tfnRule.DefaultConfiguration.Level)
	assert.Equal(t, 100.0, tfnRule.DefaultConfiguration.Rank)

	// Check tags
	tags, ok := tfnRule.Properties["tags"].([]string)
	assert.True(t, ok)
	assert.Contains(t, tags, "security")
	assert.Contains(t, tags, "privacy")
	assert.Contains(t, tags, "pii")
}

func TestSARIFExporter_NormalizeURI(t *testing.T) {
	tests := []struct {
		name     string
		baseURI  string
		filePath string
		expected string
	}{
		{
			name:     "no base URI",
			baseURI:  "",
			filePath: "src/customer.go",
			expected: "src/customer.go",
		},
		{
			name:     "with base URI - relative path",
			baseURI:  "/home/user/repo",
			filePath: "/home/user/repo/src/customer.go",
			expected: "src/customer.go",
		},
		{
			name:     "backslashes converted",
			baseURI:  "",
			filePath: `src\windows\file.go`,
			expected: "src/windows/file.go",
		},
		{
			name:     "special characters",
			baseURI:  "",
			filePath: "src/file with spaces.go",
			expected: "src/file with spaces.go", // SARIF uses readable paths
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exporter := NewSARIFExporter("Test", "1.0", "")
			exporter.SetBaseURI(tt.baseURI)

			result := exporter.normalizeURI(tt.filePath)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSARIFExporter_GetRuleID(t *testing.T) {
	exporter := NewSARIFExporter("Test", "1.0", "")

	tests := []struct {
		piType   detection.PIType
		expected string
	}{
		{detection.PITypeTFN, "PI001"},
		{detection.PITypeMedicare, "PI002"},
		{detection.PITypeABN, "PI003"},
		{detection.PITypeBSB, "PI004"},
		{detection.PITypeCreditCard, "PI005"},
		{detection.PITypeEmail, "PI006"},
		{detection.PITypePhone, "PI007"},
		{detection.PITypeName, "PI008"},
		{detection.PITypeAddress, "PI009"},
		{detection.PITypePassport, "PI010"},
		{detection.PITypeDriverLicense, "PI011"},
		{detection.PITypeIP, "PI012"},
		{detection.PIType("UNKNOWN"), "PI999"},
	}

	for _, tt := range tests {
		t.Run(string(tt.piType), func(t *testing.T) {
			result := exporter.getRuleID(tt.piType)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSARIFExporter_MapRiskLevelToSARIF(t *testing.T) {
	exporter := NewSARIFExporter("Test", "1.0", "")

	tests := []struct {
		riskLevel scoring.RiskLevel
		expected  string
	}{
		{scoring.RiskLevelCritical, "error"},
		{scoring.RiskLevelHigh, "error"},
		{scoring.RiskLevelMedium, "warning"},
		{scoring.RiskLevelLow, "note"},
		{scoring.RiskLevel("UNKNOWN"), "none"},
	}

	for _, tt := range tests {
		t.Run(string(tt.riskLevel), func(t *testing.T) {
			result := exporter.mapRiskLevelToSARIF(tt.riskLevel)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSARIFReport_ValidJSON(t *testing.T) {
	// Create a minimal valid SARIF report
	report := SARIFReport{
		Version: SARIFVersion,
		Schema:  SARIFSchema,
		Runs: []SARIFRun{
			{
				Tool: SARIFTool{
					Driver: SARIFToolComponent{
						Name:    "Test Tool",
						Version: "1.0.0",
					},
				},
				Results: []SARIFResult{
					{
						RuleID: "TEST001",
						Level:  "warning",
						Message: SARIFMessage{
							Text: "Test finding",
						},
						Locations: []SARIFLocation{
							{
								PhysicalLocation: SARIFPhysicalLocation{
									ArtifactLocation: SARIFArtifactLocation{
										URI: "test.go",
									},
									Region: SARIFRegion{
										StartLine: 1,
									},
								},
							},
						},
					},
				},
			},
		},
	}

	// Marshal to JSON
	jsonData, err := json.MarshalIndent(report, "", "  ")
	require.NoError(t, err)

	// Verify it contains expected elements
	jsonStr := string(jsonData)
	assert.Contains(t, jsonStr, `"version": "2.1.0"`)
	assert.Contains(t, jsonStr, `"$schema": "https://json.schemastore.org/sarif-2.1.0.json"`)
	assert.Contains(t, jsonStr, `"name": "Test Tool"`)
	assert.Contains(t, jsonStr, `"ruleId": "TEST001"`)

	// Verify it can be unmarshaled
	var parsed SARIFReport
	err = json.Unmarshal(jsonData, &parsed)
	require.NoError(t, err)
	assert.Equal(t, report.Version, parsed.Version)
}

func TestSARIFExporter_EmptyFindings(t *testing.T) {
	exporter := NewSARIFExporter("PI Scanner", "1.0.0", "")

	var buf bytes.Buffer
	err := exporter.Export(&buf, []detection.Finding{}, ExportMetadata{
		Timestamp: time.Now(),
	})
	require.NoError(t, err)

	// Parse and verify
	var report SARIFReport
	err = json.Unmarshal(buf.Bytes(), &report)
	require.NoError(t, err)

	assert.Len(t, report.Runs, 1)
	assert.Empty(t, report.Runs[0].Results)
}

func TestSARIFExporter_LargeReport(t *testing.T) {
	// Create many findings
	findings := make([]detection.Finding, 100)
	for i := range findings {
		findings[i] = detection.Finding{
			Type:   detection.PITypeTFN,
			Match:  "123-456-789",
			File:   fmt.Sprintf("src/file%d.go", i),
			Line:   i + 1,
			Column: 10,
		}
	}

	exporter := NewSARIFExporter("PI Scanner", "1.0.0", "")

	var buf bytes.Buffer
	err := exporter.Export(&buf, findings, ExportMetadata{
		Timestamp: time.Now(),
	})
	require.NoError(t, err)

	// Parse and verify
	var report SARIFReport
	err = json.Unmarshal(buf.Bytes(), &report)
	require.NoError(t, err)

	assert.Len(t, report.Runs[0].Results, 100)
}

func TestSARIFExporter_CreateFingerprint(t *testing.T) {
	exporter := NewSARIFExporter("Test", "1.0", "")

	finding1 := detection.Finding{
		Type:   detection.PITypeTFN,
		Match:  "123-456-789",
		File:   "test.go",
		Line:   42,
		Column: 10,
	}

	finding2 := finding1
	finding2.Line = 43 // Different line

	fp1 := exporter.createFingerprint(finding1)
	fp2 := exporter.createFingerprint(finding2)

	assert.NotEmpty(t, fp1)
	assert.NotEmpty(t, fp2)
	assert.NotEqual(t, fp1, fp2) // Different fingerprints for different lines

	// Same finding should produce same fingerprint
	fp3 := exporter.createFingerprint(finding1)
	assert.Equal(t, fp1, fp3)
}

func TestSARIFExporter_PropertiesAndTags(t *testing.T) {
	findings := []detection.Finding{
		{
			Type:      detection.PITypeTFN,
			Match:     "123-456-789",
			File:      "src/customer.go",
			Line:      42,
			Column:    10,
			Validated: true,
		},
	}

	metadata := ExportMetadata{
		ScanID:       "scan-123",
		Repository:   "test-repo",
		Branch:       "main",
		CommitHash:   "abc123",
		ScanDuration: 2 * time.Minute,
		ToolVersion:  "1.0.0",
		Timestamp:    time.Now(),
	}

	exporter := NewSARIFExporter("PI Scanner", "1.0.0", "")

	var buf bytes.Buffer
	err := exporter.Export(&buf, findings, metadata)
	require.NoError(t, err)

	// Parse the output
	var report SARIFReport
	err = json.Unmarshal(buf.Bytes(), &report)
	require.NoError(t, err)

	// Check tool properties
	toolProps := report.Runs[0].Tool.Driver.Properties
	tags, ok := toolProps["tags"].([]interface{})
	assert.True(t, ok)
	assert.Contains(t, tags, "security")
	assert.Contains(t, tags, "australia")

	// Check run properties
	runProps := report.Runs[0].Properties
	assert.Equal(t, "scan-123", runProps["scanID"])
	assert.NotNil(t, runProps["scanDuration"])

	// Check invocation properties
	invProps := report.Runs[0].Invocations[0].Properties
	assert.Equal(t, "test-repo", invProps["repository"])
	assert.Equal(t, "main", invProps["branch"])
	assert.Equal(t, "abc123", invProps["commitHash"])
}

// Example usage test
func ExampleSARIFExporter_Export() {
	findings := []detection.Finding{
		{
			Type:      detection.PITypeTFN,
			Match:     "123-456-789",
			File:      "src/customer.go",
			Line:      42,
			Column:    10,
			Validated: true,
		},
	}

	metadata := ExportMetadata{
		ScanID:       "example-scan",
		Repository:   "example-repo",
		Branch:       "main",
		ToolVersion:  "1.0.0",
		Timestamp:    time.Now(),
		ScanDuration: 30 * time.Second,
	}

	exporter := NewSARIFExporter("PI Scanner", "1.0.0", "https://github.com/MacAttak/pi-scanner")

	var buf bytes.Buffer
	if err := exporter.Export(&buf, findings, metadata); err != nil {
		panic(err)
	}

	// Check if output contains SARIF version
	output := buf.String()
	if strings.Contains(output, `"version": "2.1.0"`) {
		fmt.Println("SARIF report generated successfully")
	}

	// Output: SARIF report generated successfully
}

// Benchmarks
func BenchmarkSARIFExporter_Export(b *testing.B) {
	findings := make([]detection.Finding, 50)
	for i := range findings {
		findings[i] = detection.Finding{
			Type:   detection.PITypeTFN,
			Match:  "123-456-789",
			File:   fmt.Sprintf("src/file%d.go", i),
			Line:   i + 1,
			Column: 10,
		}
	}

	metadata := ExportMetadata{
		Timestamp:    time.Now(),
		ScanDuration: time.Minute,
	}

	exporter := NewSARIFExporter("PI Scanner", "1.0.0", "")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		err := exporter.Export(&buf, findings, metadata)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkSARIFExporter_ExportWithRiskAssessment(b *testing.B) {
	findings := make([]IntegrationRecord, 50)
	for i := range findings {
		findings[i] = IntegrationRecord{
			Finding: detection.Finding{
				Type:   detection.PITypeTFN,
				Match:  "123-456-789",
				File:   fmt.Sprintf("src/file%d.go", i),
				Line:   i + 1,
				Column: 10,
			},
			ConfidenceScore: 0.85,
			RiskAssessment: &scoring.RiskAssessment{
				OverallRisk:  0.8,
				RiskLevel:    scoring.RiskLevelHigh,
				RiskCategory: scoring.RiskCategoryIdentityTheft,
			},
		}
	}

	metadata := ExportMetadata{
		Timestamp:    time.Now(),
		ScanDuration: time.Minute,
	}

	exporter := NewSARIFExporter("PI Scanner", "1.0.0", "")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		err := exporter.ExportWithRiskAssessment(&buf, findings, metadata)
		if err != nil {
			b.Fatal(err)
		}
	}
}
