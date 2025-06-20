package report

import (
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"time"

	"github.com/MacAttak/pi-scanner/pkg/detection"
	"github.com/MacAttak/pi-scanner/pkg/scoring"
)

// CSVExporter handles CSV report generation
type CSVExporter struct {
	includeContext  bool
	includeMasked   bool
	includeMetadata bool
	dateFormat      string
}

// CSVExporterOption configures the CSV exporter
type CSVExporterOption func(*CSVExporter)

// WithContext includes code context in CSV export
func WithContext() CSVExporterOption {
	return func(e *CSVExporter) {
		e.includeContext = true
	}
}

// WithMaskedValues includes masked PI values in CSV export
func WithMaskedValues() CSVExporterOption {
	return func(e *CSVExporter) {
		e.includeMasked = true
	}
}

// WithMetadata includes additional metadata columns
func WithMetadata() CSVExporterOption {
	return func(e *CSVExporter) {
		e.includeMetadata = true
	}
}

// WithDateFormat sets custom date format
func WithDateFormat(format string) CSVExporterOption {
	return func(e *CSVExporter) {
		e.dateFormat = format
	}
}

// NewCSVExporter creates a new CSV exporter with options
func NewCSVExporter(opts ...CSVExporterOption) *CSVExporter {
	exporter := &CSVExporter{
		dateFormat: "2006-01-02 15:04:05",
	}

	for _, opt := range opts {
		opt(exporter)
	}

	return exporter
}

// CSVRecord represents a single row in the CSV export
type CSVRecord struct {
	// Core fields
	Timestamp     time.Time
	Repository    string
	Branch        string
	CommitHash    string
	FilePath      string
	LineNumber    int
	ColumnNumber  int
	PIType        string
	PITypeDisplay string
	Match         string
	MaskedMatch   string
	Validated     bool
	IsTestData    bool

	// Risk assessment
	ConfidenceScore float64
	RiskLevel       string
	RiskScore       float64
	ImpactScore     float64
	LikelihoodScore float64
	ExposureScore   float64
	RiskCategory    string

	// Context
	CodeContext      string
	ProximityContext string
	Environment      string

	// Compliance
	APRARelevant     bool
	PrivacyActIssue  bool
	NotifiableBreach bool

	// Metadata
	ScanID       string
	ScanDuration time.Duration
	ToolVersion  string
}

// Export writes findings to CSV format
func (e *CSVExporter) Export(w io.Writer, records []CSVRecord) error {
	writer := csv.NewWriter(w)
	defer writer.Flush()

	// Write headers
	headers := e.getHeaders()
	if err := writer.Write(headers); err != nil {
		return fmt.Errorf("failed to write CSV headers: %w", err)
	}

	// Write records
	for _, record := range records {
		row := e.recordToRow(record)
		if err := writer.Write(row); err != nil {
			return fmt.Errorf("failed to write CSV record: %w", err)
		}
	}

	return writer.Error()
}

// ExportFindings converts findings to CSV records and exports them
func (e *CSVExporter) ExportFindings(w io.Writer, findings []detection.Finding, metadata ExportMetadata) error {
	records := make([]CSVRecord, 0, len(findings))

	for _, finding := range findings {
		record := e.findingToRecord(finding, metadata)
		records = append(records, record)
	}

	return e.Export(w, records)
}

// ExportMetadata contains scan metadata for the export
type ExportMetadata struct {
	ScanID       string
	Repository   string
	Branch       string
	CommitHash   string
	ScanDuration time.Duration
	ToolVersion  string
	Timestamp    time.Time
}

// getHeaders returns CSV column headers based on configuration
func (e *CSVExporter) getHeaders() []string {
	headers := []string{
		"Timestamp",
		"Repository",
		"Branch",
		"File Path",
		"Line",
		"Column",
		"PI Type",
		"PI Type Display",
		"Validated",
		"Test Data",
		"Confidence Score",
		"Risk Level",
		"Risk Score",
	}

	if e.includeMasked {
		headers = append(headers, "Masked Value")
	}

	headers = append(headers,
		"Impact Score",
		"Likelihood Score",
		"Exposure Score",
		"Risk Category",
		"Environment",
	)

	if e.includeContext {
		headers = append(headers, "Code Context", "Proximity Context")
	}

	headers = append(headers,
		"APRA Relevant",
		"Privacy Act Issue",
		"Notifiable Breach",
	)

	if e.includeMetadata {
		headers = append(headers,
			"Scan ID",
			"Commit Hash",
			"Scan Duration (seconds)",
			"Tool Version",
		)
	}

	return headers
}

// recordToRow converts a CSVRecord to a CSV row
func (e *CSVExporter) recordToRow(record CSVRecord) []string {
	row := []string{
		record.Timestamp.Format(e.dateFormat),
		record.Repository,
		record.Branch,
		record.FilePath,
		strconv.Itoa(record.LineNumber),
		strconv.Itoa(record.ColumnNumber),
		record.PIType,
		record.PITypeDisplay,
		strconv.FormatBool(record.Validated),
		strconv.FormatBool(record.IsTestData),
		fmt.Sprintf("%.2f", record.ConfidenceScore),
		record.RiskLevel,
		fmt.Sprintf("%.2f", record.RiskScore),
	}

	if e.includeMasked {
		row = append(row, record.MaskedMatch)
	}

	row = append(row,
		fmt.Sprintf("%.2f", record.ImpactScore),
		fmt.Sprintf("%.2f", record.LikelihoodScore),
		fmt.Sprintf("%.2f", record.ExposureScore),
		record.RiskCategory,
		record.Environment,
	)

	if e.includeContext {
		row = append(row, record.CodeContext, record.ProximityContext)
	}

	row = append(row,
		strconv.FormatBool(record.APRARelevant),
		strconv.FormatBool(record.PrivacyActIssue),
		strconv.FormatBool(record.NotifiableBreach),
	)

	if e.includeMetadata {
		row = append(row,
			record.ScanID,
			record.CommitHash,
			fmt.Sprintf("%.2f", record.ScanDuration.Seconds()),
			record.ToolVersion,
		)
	}

	return row
}

// findingToRecord converts a detection.Finding to a CSVRecord
func (e *CSVExporter) findingToRecord(finding detection.Finding, metadata ExportMetadata) CSVRecord {
	record := CSVRecord{
		Timestamp:     metadata.Timestamp,
		Repository:    metadata.Repository,
		Branch:        metadata.Branch,
		CommitHash:    metadata.CommitHash,
		FilePath:      finding.File,
		LineNumber:    finding.Line,
		ColumnNumber:  finding.Column,
		PIType:        string(finding.Type),
		PITypeDisplay: getPITypeDisplay(finding.Type),
		Match:         finding.Match, // Note: In production, this should be masked
		Validated:     finding.Validated,
		IsTestData:    false, // This would come from scoring
		ScanID:        metadata.ScanID,
		ScanDuration:  metadata.ScanDuration,
		ToolVersion:   metadata.ToolVersion,
	}

	// Mask the match value
	record.MaskedMatch = maskSensitiveData(finding.Match, string(finding.Type))

	// Add placeholder values for fields that would come from scoring
	// In a real implementation, these would be populated from the risk assessment
	record.ConfidenceScore = 0.0
	record.RiskLevel = "UNKNOWN"
	record.RiskScore = 0.0
	record.Environment = "unknown"

	return record
}

// getPITypeDisplay returns a human-readable display name for PI types
func getPITypeDisplay(piType detection.PIType) string {
	displays := map[detection.PIType]string{
		detection.PITypeTFN:           "Tax File Number",
		detection.PITypeMedicare:      "Medicare Number",
		detection.PITypeABN:           "Australian Business Number",
		detection.PITypeBSB:           "Bank State Branch",
		detection.PITypeCreditCard:    "Credit Card",
		detection.PITypeEmail:         "Email Address",
		detection.PITypePhone:         "Phone Number",
		detection.PITypeName:          "Personal Name",
		detection.PITypeAddress:       "Physical Address",
		detection.PITypePassport:      "Passport Number",
		detection.PITypeDriverLicense: "Driver License",
		detection.PITypeIP:            "IP Address",
	}

	if display, exists := displays[piType]; exists {
		return display
	}
	return string(piType)
}

// CSVSummaryExporter exports summary statistics in CSV format
type CSVSummaryExporter struct {
	dateFormat string
}

// NewCSVSummaryExporter creates a new summary exporter
func NewCSVSummaryExporter() *CSVSummaryExporter {
	return &CSVSummaryExporter{
		dateFormat: "2006-01-02 15:04:05",
	}
}

// ExportSummary writes summary statistics to CSV
func (e *CSVSummaryExporter) ExportSummary(w io.Writer, summary ScanSummary, metadata ExportMetadata) error {
	writer := csv.NewWriter(w)
	defer writer.Flush()

	// Write headers
	headers := []string{"Metric", "Value", "Percentage"}
	if err := writer.Write(headers); err != nil {
		return fmt.Errorf("failed to write summary headers: %w", err)
	}

	// Write summary rows
	rows := [][]string{
		{"Repository", metadata.Repository, ""},
		{"Branch", metadata.Branch, ""},
		{"Scan Date", metadata.Timestamp.Format(e.dateFormat), ""},
		{"Scan Duration", fmt.Sprintf("%.2f seconds", metadata.ScanDuration.Seconds()), ""},
		{"Total Findings", strconv.Itoa(summary.TotalFindings), "100.0%"},
		{"Critical Risk", strconv.Itoa(summary.CriticalCount), e.percentage(summary.CriticalCount, summary.TotalFindings)},
		{"High Risk", strconv.Itoa(summary.HighCount), e.percentage(summary.HighCount, summary.TotalFindings)},
		{"Medium Risk", strconv.Itoa(summary.MediumCount), e.percentage(summary.MediumCount, summary.TotalFindings)},
		{"Low Risk", strconv.Itoa(summary.LowCount), e.percentage(summary.LowCount, summary.TotalFindings)},
		{"Validated PI", strconv.Itoa(summary.ValidatedCount), e.percentage(summary.ValidatedCount, summary.TotalFindings)},
		{"Test Data", strconv.Itoa(summary.TestDataCount), e.percentage(summary.TestDataCount, summary.TotalFindings)},
	}

	for _, row := range rows {
		if err := writer.Write(row); err != nil {
			return fmt.Errorf("failed to write summary row: %w", err)
		}
	}

	return writer.Error()
}

// percentage calculates percentage as string
func (e *CSVSummaryExporter) percentage(value, total int) string {
	if total == 0 {
		return "0.0%"
	}
	return fmt.Sprintf("%.1f%%", float64(value)/float64(total)*100)
}

// IntegrationRecord represents a record with full scoring integration
type IntegrationRecord struct {
	Finding         detection.Finding
	ConfidenceScore float64
	RiskAssessment  *scoring.RiskAssessment
	Environment     string
	ProximityInfo   string
}

// ConvertIntegrationRecord converts an integrated record to CSV record
func (e *CSVExporter) ConvertIntegrationRecord(ir IntegrationRecord, metadata ExportMetadata) CSVRecord {
	record := e.findingToRecord(ir.Finding, metadata)

	// Update with actual scoring data
	record.ConfidenceScore = ir.ConfidenceScore
	record.Environment = ir.Environment
	record.ProximityContext = ir.ProximityInfo

	if ir.RiskAssessment != nil {
		record.RiskLevel = string(ir.RiskAssessment.RiskLevel)
		record.RiskScore = ir.RiskAssessment.OverallRisk
		record.ImpactScore = ir.RiskAssessment.ImpactScore
		record.LikelihoodScore = ir.RiskAssessment.LikelihoodScore
		record.ExposureScore = ir.RiskAssessment.ExposureScore
		record.RiskCategory = string(ir.RiskAssessment.RiskCategory)

		record.APRARelevant = ir.RiskAssessment.ComplianceFlags.APRAReporting
		record.PrivacyActIssue = ir.RiskAssessment.ComplianceFlags.PrivacyActBreach
		record.NotifiableBreach = ir.RiskAssessment.ComplianceFlags.NotifiableDataBreach
	}

	// Check if it's test data based on environment
	record.IsTestData = ir.Environment == "test" || ir.Environment == "mock"

	return record
}
