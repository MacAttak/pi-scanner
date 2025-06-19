package report

import (
	"bytes"
	"fmt"
	"html/template"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetHTMLTemplate(t *testing.T) {
	tests := []struct {
		name    string
		wantErr bool
	}{
		{
			name:    "successfully load template",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpl, err := GetHTMLTemplate()

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, tmpl)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, tmpl)
				
				// Verify all sub-templates are loaded
				assert.NotNil(t, tmpl.Lookup("styles"))
				assert.NotNil(t, tmpl.Lookup("scripts"))
			}
		})
	}
}

func TestHTMLTemplateRendering(t *testing.T) {
	tmpl, err := GetHTMLTemplate()
	require.NoError(t, err)

	data := HTMLTemplateData{
		ReportID:     "test-123",
		GeneratedAt:  time.Now(),
		ScanDuration: "1m 30s",
		ToolVersion:  "1.0.0",
		Repository: RepositoryInfo{
			Name:           "test-repo",
			URL:            "https://github.com/test/repo",
			Branch:         "main",
			CommitHash:     "abc123",
			LastCommitDate: time.Now().Add(-24 * time.Hour),
			FilesScanned:   100,
			LinesScanned:   10000,
		},
		Summary: ScanSummary{
			TotalFindings:  25,
			CriticalCount:  2,
			HighCount:      5,
			MediumCount:    8,
			LowCount:       10,
			UniqueTypes:    []string{"TFN", "Medicare", "ABN"},
			TopRisks:       []string{"Identity theft", "Financial fraud"},
			TestDataCount:  3,
			ValidatedCount: 15,
		},
		CriticalFindings: []Finding{
			{
				ID:              "finding-1",
				Type:            "TFN",
				TypeDisplay:     "Tax File Number",
				RiskLevel:       "CRITICAL",
				ConfidenceScore: 0.95,
				File:            "src/customer.go",
				Line:            42,
				Column:          10,
				Match:           "123-456-789",
				MaskedMatch:     "123****89",
				Context:         "customerTFN := \"123-456-789\"",
				Validated:       true,
				IsTestData:      false,
				RiskAssessment: RiskAssessmentInfo{
					OverallRisk:     0.9,
					ImpactScore:     0.95,
					LikelihoodScore: 0.85,
					ExposureScore:   0.9,
					RiskCategory:    "IDENTITY_THEFT",
					Factors:         []string{"Public repository", "Production code"},
				},
				Mitigations: []Mitigation{
					{
						Title:       "Remove hardcoded TFN",
						Description: "Move TFN to secure environment variables",
						Priority:    "CRITICAL",
						Effort:      "Low",
						Timeline:    "Immediate",
					},
				},
			},
		},
		Statistics: Statistics{
			TypeDistribution: map[string]int{
				"TFN":      10,
				"Medicare": 8,
				"ABN":      7,
			},
			RiskDistribution: map[string]int{
				"CRITICAL": 2,
				"HIGH":     5,
				"MEDIUM":   8,
				"LOW":      10,
			},
			TopAffectedFiles: []FileStats{
				{
					Path:          "src/customer.go",
					FindingsCount: 5,
					RiskScore:     0.85,
				},
			},
			ValidationStats: ValidationStats{
				TotalChecked:   25,
				ValidCount:     15,
				InvalidCount:   10,
				ValidationRate: 0.6,
			},
			EnvironmentStats: EnvironmentStats{
				ProductionFindings: 20,
				TestFindings:       3,
				MockFindings:       1,
				ConfigFindings:     1,
			},
		},
		Compliance: ComplianceInfo{
			APRACompliant:       false,
			PrivacyActCompliant: false,
			NotifiableBreaches:  2,
			RequiredNotifications: []string{
				"Office of the Australian Information Commissioner (OAIC)",
				"Australian Prudential Regulation Authority (APRA)",
			},
			ComplianceActions: []ComplianceAction{
				{
					Type:        "NOTIFICATION",
					Description: "Notify OAIC within 72 hours",
					Priority:    "CRITICAL",
					Deadline:    time.Now().Add(72 * time.Hour),
					Regulation:  "Privacy Act 1988",
				},
			},
		},
	}

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, data)
	require.NoError(t, err)

	html := buf.String()
	
	// Verify key elements are present
	assert.Contains(t, html, "PI Scanner Report")
	assert.Contains(t, html, "test-repo")
	assert.Contains(t, html, "Executive Summary")
	assert.Contains(t, html, "Critical Risk")
	assert.Contains(t, html, "Australian Regulatory Compliance")
	assert.Contains(t, html, "123****89") // Masked TFN
	assert.Contains(t, html, "Risk Analysis")
	assert.Contains(t, html, "riskChart") // Chart element
	assert.Contains(t, html, "typeChart") // Chart element
	
	// Verify CSS is included
	assert.Contains(t, html, "summary-card")
	assert.Contains(t, html, "finding-card")
	
	// Verify JavaScript is included
	assert.Contains(t, html, "chart.js") // CDN link uses lowercase
	assert.Contains(t, html, "createRiskDistributionChart")
}

func TestMaskSensitiveData(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		piType   string
		expected string
	}{
		{
			name:     "mask TFN",
			value:    "123456789",
			piType:   "TFN",
			expected: "123****89",
		},
		{
			name:     "mask Medicare",
			value:    "2234567890",
			piType:   "MEDICARE",
			expected: "22******90",
		},
		{
			name:     "mask credit card",
			value:    "4111111111111111",
			piType:   "CREDIT_CARD",
			expected: "************1111",
		},
		{
			name:     "mask email",
			value:    "john.doe@example.com",
			piType:   "EMAIL",
			expected: "jo***@example.com",
		},
		{
			name:     "mask phone",
			value:    "0412345678",
			piType:   "PHONE",
			expected: "0412****78",
		},
		{
			name:     "empty value",
			value:    "",
			piType:   "TFN",
			expected: "",
		},
		{
			name:     "short value",
			value:    "12",
			piType:   "TFN",
			expected: "**",
		},
		{
			name:     "unknown type",
			value:    "sensitive",
			piType:   "UNKNOWN",
			expected: "s*******e",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := maskSensitiveData(tt.value, tt.piType)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestHTMLTemplateFunctions(t *testing.T) {
	funcMap := GetTemplateFuncMap()

	tests := []struct {
		name     string
		template string
		data     interface{}
		expected string
	}{
		{
			name:     "formatTime",
			template: `{{formatTime .}}`,
			data:     time.Date(2024, 1, 15, 14, 30, 0, 0, time.UTC),
			expected: "15 Jan 2024 14:30:00 UTC",
		},
		{
			name:     "formatDate",
			template: `{{formatDate .}}`,
			data:     time.Date(2024, 1, 15, 14, 30, 0, 0, time.UTC),
			expected: "15 Jan 2024",
		},
		{
			name:     "formatPercent",
			template: `{{formatPercent .}}`,
			data:     0.756,
			expected: "75.6%",
		},
		{
			name:     "formatScore",
			template: `{{formatScore .}}`,
			data:     0.8567,
			expected: "0.86",
		},
		{
			name:     "riskLevelClass",
			template: `{{riskLevelClass .}}`,
			data:     "CRITICAL",
			expected: "risk-critical",
		},
		{
			name:     "riskLevelIcon",
			template: `{{riskLevelIcon .}}`,
			data:     "HIGH",
			expected: "🔴",
		},
		{
			name:     "piTypeIcon",
			template: `{{piTypeIcon .}}`,
			data:     "TFN",
			expected: "🆔",
		},
		{
			name:     "jsonify",
			template: `{{jsonify .}}`,
			data:     map[string]int{"a": 1, "b": 2},
			expected: `{&#34;a&#34;:1,&#34;b&#34;:2}`, // HTML template escapes quotes
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a new template for each test
			testTmpl := template.New(tt.name).Funcs(funcMap)
			funcTmpl, err := testTmpl.Parse(tt.template)
			require.NoError(t, err)

			var buf bytes.Buffer
			err = funcTmpl.Execute(&buf, tt.data)
			require.NoError(t, err)

			assert.Equal(t, tt.expected, strings.TrimSpace(buf.String()))
		})
	}
}

func TestHTMLTemplateValidation(t *testing.T) {
	// Test that template compiles without errors
	tmpl, err := GetHTMLTemplate()
	require.NoError(t, err)
	require.NotNil(t, tmpl)

	// Test with minimal data
	minimalData := HTMLTemplateData{
		ReportID:    "minimal",
		GeneratedAt: time.Now(),
		ToolVersion: "1.0.0",
		Repository: RepositoryInfo{
			Name: "test",
		},
		Summary: ScanSummary{},
	}

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, minimalData)
	assert.NoError(t, err, "Template should handle minimal data without errors")
}

func TestHTMLTemplateComplexData(t *testing.T) {
	tmpl, err := GetHTMLTemplate()
	require.NoError(t, err)

	// Create complex data with multiple findings
	var findings []Finding
	for i := 0; i < 50; i++ {
		findings = append(findings, Finding{
			ID:              fmt.Sprintf("finding-%d", i),
			Type:            "TFN",
			TypeDisplay:     "Tax File Number",
			RiskLevel:       "MEDIUM",
			ConfidenceScore: 0.7,
			File:            fmt.Sprintf("src/file%d.go", i),
			Line:            i + 1,
			Column:          10,
			Match:           "123-456-789",
			MaskedMatch:     "123****89",
			Validated:       i%2 == 0,
			IsTestData:      i%5 == 0,
		})
	}

	data := HTMLTemplateData{
		ReportID:       "complex-test",
		GeneratedAt:    time.Now(),
		ScanDuration:   "5m 30s",
		ToolVersion:    "1.0.0",
		MediumFindings: findings,
		Repository: RepositoryInfo{
			Name:         "complex-repo",
			FilesScanned: 500,
			LinesScanned: 50000,
		},
		Summary: ScanSummary{
			TotalFindings: 50,
			MediumCount:   50,
		},
		Statistics: Statistics{
			TypeDistribution: map[string]int{
				"TFN":      20,
				"Medicare": 15,
				"ABN":      10,
				"BSB":      5,
			},
		},
	}

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, data)
	require.NoError(t, err)

	html := buf.String()
	assert.Contains(t, html, "50 findings - click to expand")
	assert.Contains(t, html, "complex-repo")
}

func BenchmarkHTMLTemplateRendering(b *testing.B) {
	tmpl, err := GetHTMLTemplate()
	require.NoError(b, err)

	// Create realistic data
	findings := make([]Finding, 20)
	for i := range findings {
		findings[i] = Finding{
			ID:              fmt.Sprintf("finding-%d", i),
			Type:            "TFN",
			TypeDisplay:     "Tax File Number",
			RiskLevel:       "HIGH",
			ConfidenceScore: 0.85,
			File:            fmt.Sprintf("src/file%d.go", i),
			Line:            100 + i,
			Column:          10,
			Match:           "123-456-789",
			MaskedMatch:     "123****89",
			Validated:       true,
		}
	}

	data := HTMLTemplateData{
		ReportID:      "bench-test",
		GeneratedAt:   time.Now(),
		ToolVersion:   "1.0.0",
		Repository:    RepositoryInfo{Name: "bench-repo", FilesScanned: 1000},
		Summary:       ScanSummary{TotalFindings: 100, HighCount: 20},
		HighFindings:  findings,
		Statistics: Statistics{
			TypeDistribution: map[string]int{
				"TFN": 40, "Medicare": 30, "ABN": 20, "BSB": 10,
			},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		err := tmpl.Execute(&buf, data)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func TestHTMLTemplateEscaping(t *testing.T) {
	tmpl, err := GetHTMLTemplate()
	require.NoError(t, err)

	// Test with potentially dangerous input
	data := HTMLTemplateData{
		ReportID:    "xss-test",
		GeneratedAt: time.Now(),
		ToolVersion: "1.0.0",
		Repository: RepositoryInfo{
			Name: "<script>alert('xss')</script>",
			URL:  "javascript:alert('xss')",
		},
		CriticalFindings: []Finding{
			{
				ID:          "xss-finding",
				Type:        "TFN",
				TypeDisplay: "Tax File Number",
				File:        "<img src=x onerror=alert('xss')>",
				Context:     "<script>alert('context')</script>",
				MaskedMatch: "123****89",
			},
		},
		Summary: ScanSummary{
			CriticalCount: 1,
			TotalFindings: 1,
		},
	}

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, data)
	require.NoError(t, err)

	html := buf.String()
	
	// Debug: print a snippet to see what's happening
	if strings.Contains(html, "onerror") {
		idx := strings.Index(html, "onerror")
		start := idx - 50
		if start < 0 {
			start = 0
		}
		end := idx + 50
		if end > len(html) {
			end = len(html)
		}
		t.Logf("Found 'onerror' at position %d: %s", idx, html[start:end])
	}
	
	// Verify XSS attempts are escaped - check for actual dangerous content
	assert.NotContains(t, html, "<script>alert('xss')</script>")
	assert.NotContains(t, html, "javascript:alert('xss')")
	
	// The onerror attribute might be in escaped form in the file path
	// Check that the actual XSS attempt is not executable
	assert.NotRegexp(t, `<img[^>]+onerror\s*=`, html, "Should not contain executable onerror attribute")
	
	// Verify escaped versions are present
	assert.Contains(t, html, "&lt;script&gt;")
	assert.Contains(t, html, "&lt;img")
}

func TestHTMLTemplateAccount(t *testing.T) {
	// Test the Account PI type doesn't exist in template functions
	tmpl, err := GetHTMLTemplate()
	require.NoError(t, err)

	// Account type should get default icon
	var buf bytes.Buffer
	testTmpl, err := tmpl.New("test").Parse(`{{piTypeIcon "ACCOUNT"}}`)
	require.NoError(t, err)
	
	err = testTmpl.ExecuteTemplate(&buf, "test", nil)
	require.NoError(t, err)
	
	assert.Equal(t, "📄", buf.String()) // Default icon
}