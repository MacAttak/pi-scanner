package report

import (
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"strings"
	"time"
)

//go:embed templates/*.html templates/*.css templates/*.js
var templatesFS embed.FS

// HTMLTemplateData represents the data structure for HTML report generation
type HTMLTemplateData struct {
	// Report metadata
	ReportID     string    `json:"report_id"`
	GeneratedAt  time.Time `json:"generated_at"`
	ScanDuration string    `json:"scan_duration"`
	ToolVersion  string    `json:"tool_version"`

	// Repository information
	Repository RepositoryInfo `json:"repository"`

	// Scan summary
	Summary ScanSummary `json:"summary"`

	// Findings by risk level
	CriticalFindings []Finding `json:"critical_findings"`
	HighFindings     []Finding `json:"high_findings"`
	MediumFindings   []Finding `json:"medium_findings"`
	LowFindings      []Finding `json:"low_findings"`

	// Statistics and charts data
	Statistics Statistics `json:"statistics"`

	// Compliance information
	Compliance ComplianceInfo `json:"compliance"`
}

// RepositoryInfo contains repository details
type RepositoryInfo struct {
	Name           string    `json:"name"`
	URL            string    `json:"url"`
	Branch         string    `json:"branch"`
	CommitHash     string    `json:"commit_hash"`
	LastCommitDate time.Time `json:"last_commit_date"`
	FilesScanned   int       `json:"files_scanned"`
	LinesScanned   int       `json:"lines_scanned"`
}

// ScanSummary provides high-level scan results
type ScanSummary struct {
	TotalFindings  int      `json:"total_findings"`
	CriticalCount  int      `json:"critical_count"`
	HighCount      int      `json:"high_count"`
	MediumCount    int      `json:"medium_count"`
	LowCount       int      `json:"low_count"`
	UniqueTypes    []string `json:"unique_types"`
	TopRisks       []string `json:"top_risks"`
	TestDataCount  int      `json:"test_data_count"`
	ValidatedCount int      `json:"validated_count"`
}

// Finding represents a single PI detection finding
type Finding struct {
	ID              string             `json:"id"`
	Type            string             `json:"type"`
	TypeDisplay     string             `json:"type_display"`
	RiskLevel       string             `json:"risk_level"`
	ConfidenceScore float64            `json:"confidence_score"`
	File            string             `json:"file"`
	Line            int                `json:"line"`
	Column          int                `json:"column"`
	Match           string             `json:"match"`
	MaskedMatch     string             `json:"masked_match"`
	Context         string             `json:"context"`
	Validated       bool               `json:"validated"`
	IsTestData      bool               `json:"is_test_data"`
	RiskAssessment  RiskAssessmentInfo `json:"risk_assessment"`
	Mitigations     []Mitigation       `json:"mitigations"`
}

// RiskAssessmentInfo contains risk scoring details
type RiskAssessmentInfo struct {
	OverallRisk     float64  `json:"overall_risk"`
	ImpactScore     float64  `json:"impact_score"`
	LikelihoodScore float64  `json:"likelihood_score"`
	ExposureScore   float64  `json:"exposure_score"`
	RiskCategory    string   `json:"risk_category"`
	Factors         []string `json:"factors"`
}

// Mitigation represents a recommended mitigation action
type Mitigation struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Priority    string `json:"priority"`
	Effort      string `json:"effort"`
	Timeline    string `json:"timeline"`
}

// Statistics contains scan statistics
type Statistics struct {
	// PI type distribution
	TypeDistribution map[string]int `json:"type_distribution"`

	// Risk level distribution
	RiskDistribution map[string]int `json:"risk_distribution"`

	// File type distribution
	FileTypeDistribution map[string]int `json:"file_type_distribution"`

	// Top affected files
	TopAffectedFiles []FileStats `json:"top_affected_files"`

	// Validation statistics
	ValidationStats ValidationStats `json:"validation_stats"`

	// Environment statistics
	EnvironmentStats EnvironmentStats `json:"environment_stats"`
}

// FileStats represents statistics for a single file
type FileStats struct {
	Path          string  `json:"path"`
	FindingsCount int     `json:"findings_count"`
	RiskScore     float64 `json:"risk_score"`
}

// ValidationStats contains validation statistics
type ValidationStats struct {
	TotalChecked   int     `json:"total_checked"`
	ValidCount     int     `json:"valid_count"`
	InvalidCount   int     `json:"invalid_count"`
	ValidationRate float64 `json:"validation_rate"`
}

// EnvironmentStats contains environment-based statistics
type EnvironmentStats struct {
	ProductionFindings int `json:"production_findings"`
	TestFindings       int `json:"test_findings"`
	MockFindings       int `json:"mock_findings"`
	ConfigFindings     int `json:"config_findings"`
}

// ComplianceInfo contains regulatory compliance information
type ComplianceInfo struct {
	APRACompliant         bool               `json:"apra_compliant"`
	PrivacyActCompliant   bool               `json:"privacy_act_compliant"`
	NotifiableBreaches    int                `json:"notifiable_breaches"`
	RequiredNotifications []string           `json:"required_notifications"`
	ComplianceActions     []ComplianceAction `json:"compliance_actions"`
}

// ComplianceAction represents a required compliance action
type ComplianceAction struct {
	Type        string    `json:"type"`
	Description string    `json:"description"`
	Priority    string    `json:"priority"`
	Deadline    time.Time `json:"deadline"`
	Regulation  string    `json:"regulation"`
}

// GetTemplateFuncMap returns the template function map
func GetTemplateFuncMap() template.FuncMap {
	return template.FuncMap{
		"formatTime": func(t time.Time) string {
			return t.Format("2 Jan 2006 15:04:05 MST")
		},
		"formatDate": func(t time.Time) string {
			return t.Format("2 Jan 2006")
		},
		"formatPercent": func(val float64) string {
			return fmt.Sprintf("%.1f%%", val*100)
		},
		"formatScore": func(val float64) string {
			return fmt.Sprintf("%.2f", val)
		},
		"maskPI": func(val string, piType string) string {
			return maskSensitiveData(val, piType)
		},
		"riskLevelClass": func(level string) string {
			switch level {
			case "CRITICAL":
				return "risk-critical"
			case "HIGH":
				return "risk-high"
			case "MEDIUM":
				return "risk-medium"
			case "LOW":
				return "risk-low"
			default:
				return "risk-unknown"
			}
		},
		"riskLevelIcon": func(level string) string {
			switch level {
			case "CRITICAL":
				return "⚠️"
			case "HIGH":
				return "🔴"
			case "MEDIUM":
				return "🟡"
			case "LOW":
				return "🟢"
			default:
				return "❓"
			}
		},
		"piTypeIcon": func(piType string) string {
			icons := map[string]string{
				"TFN":            "🆔",
				"MEDICARE":       "🏥",
				"ABN":            "🏢",
				"BSB":            "🏦",
				"CREDIT_CARD":    "💳",
				"EMAIL":          "📧",
				"PHONE":          "📱",
				"NAME":           "👤",
				"ADDRESS":        "🏠",
				"PASSPORT":       "📔",
				"DRIVER_LICENSE": "🚗",
			}
			if icon, exists := icons[piType]; exists {
				return icon
			}
			return "📄"
		},
		"jsonify": func(v interface{}) template.JS {
			b, err := json.Marshal(v)
			if err != nil {
				return template.JS("{}")
			}
			return template.JS(b)
		},
	}
}

// GetHTMLTemplate returns the parsed HTML template
func GetHTMLTemplate() (*template.Template, error) {
	funcMap := GetTemplateFuncMap()

	tmplContent, err := templatesFS.ReadFile("templates/report.html")
	if err != nil {
		return nil, fmt.Errorf("failed to read template: %w", err)
	}

	tmpl, err := template.New("report").Funcs(funcMap).Parse(string(tmplContent))
	if err != nil {
		return nil, fmt.Errorf("failed to parse template: %w", err)
	}

	// Parse CSS
	cssContent, err := templatesFS.ReadFile("templates/styles.css")
	if err != nil {
		return nil, fmt.Errorf("failed to read CSS: %w", err)
	}

	_, err = tmpl.New("styles").Parse(string(cssContent))
	if err != nil {
		return nil, fmt.Errorf("failed to parse CSS: %w", err)
	}

	// Parse JS
	jsContent, err := templatesFS.ReadFile("templates/scripts.js")
	if err != nil {
		return nil, fmt.Errorf("failed to read JS: %w", err)
	}

	_, err = tmpl.New("scripts").Parse(string(jsContent))
	if err != nil {
		return nil, fmt.Errorf("failed to parse JS: %w", err)
	}

	return tmpl, nil
}

// maskSensitiveData masks PI data for display
func maskSensitiveData(value string, piType string) string {
	if len(value) == 0 {
		return value
	}

	switch piType {
	case "TFN":
		// Show first 3 and last 2 digits
		if len(value) >= 9 {
			return value[:3] + "****" + value[len(value)-2:]
		}
	case "MEDICARE":
		// Show first 2 and last 2 digits
		if len(value) >= 10 {
			return value[:2] + "******" + value[len(value)-2:]
		}
	case "CREDIT_CARD":
		// Show last 4 digits only
		if len(value) >= 12 {
			return "************" + value[len(value)-4:]
		}
	case "EMAIL":
		// Show first 2 chars and domain
		parts := strings.Split(value, "@")
		if len(parts) == 2 && len(parts[0]) > 2 {
			return parts[0][:2] + "***@" + parts[1]
		}
	case "PHONE":
		// Show area code and last 2 digits
		if len(value) >= 10 {
			return value[:4] + "****" + value[len(value)-2:]
		}
	}

	// Default masking - show first and last character
	if len(value) > 2 {
		return value[:1] + strings.Repeat("*", len(value)-2) + value[len(value)-1:]
	}

	return strings.Repeat("*", len(value))
}
