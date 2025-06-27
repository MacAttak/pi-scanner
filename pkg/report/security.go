package report

import (
	"bytes"
	"fmt"
	"io"
	"log/slog"

	"github.com/MacAttak/pi-scanner/pkg/detection"
	"github.com/MacAttak/pi-scanner/pkg/output"
)

// SecurityLayer provides security controls for report generation
type SecurityLayer struct {
	outputManager *output.Manager
	config        *SecurityConfig
	logger        *slog.Logger
}

// SecurityConfig configures the security layer
type SecurityConfig struct {
	// EnforceMasking ensures all reports use the configured masking level
	EnforceMasking bool

	// ValidateBeforeOutput checks output for unmasked PI
	ValidateBeforeOutput bool

	// PreventDirectOutput prevents bypassing the security layer
	PreventDirectOutput bool

	// LogAllOperations logs all report generation
	LogAllOperations bool

	// RequireAuthentication requires auth for sensitive operations
	RequireAuthentication bool
}

// DefaultSecurityConfig returns secure defaults
func DefaultSecurityConfig() *SecurityConfig {
	return &SecurityConfig{
		EnforceMasking:        true,
		ValidateBeforeOutput:  true,
		PreventDirectOutput:   true,
		LogAllOperations:      true,
		RequireAuthentication: false, // Would be true in production
	}
}

// NewSecurityLayer creates a new security layer
func NewSecurityLayer(outputManager *output.Manager, config *SecurityConfig, logger *slog.Logger) *SecurityLayer {
	if config == nil {
		config = DefaultSecurityConfig()
	}

	if logger == nil {
		logger = outputManager.GetSafeLogger()
	}

	return &SecurityLayer{
		outputManager: outputManager,
		config:        config,
		logger:        logger,
	}
}

// SecureCSVExporter wraps CSV export with security controls
type SecureCSVExporter struct {
	*CSVExporter
	security *SecurityLayer
}

// NewSecureCSVExporter creates a secure CSV exporter
func (s *SecurityLayer) NewSecureCSVExporter(opts ...CSVExporterOption) *SecureCSVExporter {
	return &SecureCSVExporter{
		CSVExporter: NewCSVExporter(opts...),
		security:    s,
	}
}

// ExportFindings exports findings with security controls
func (e *SecureCSVExporter) ExportFindings(w io.Writer, findings []detection.Finding, metadata ExportMetadata) error {
	// Log the operation
	if e.security.config.LogAllOperations {
		e.security.logger.Info("generating CSV report",
			"finding_count", len(findings),
			"repository", metadata.Repository,
			"branch", metadata.Branch,
		)
	}

	// Prepare findings with masking
	maskedFindings := findings
	if e.security.config.EnforceMasking {
		maskedFindings = e.security.outputManager.PrepareFindings(findings)
	}

	// Generate the report
	var buf bytes.Buffer
	if err := e.CSVExporter.ExportFindings(&buf, maskedFindings, metadata); err != nil {
		return fmt.Errorf("CSV generation failed: %w", err)
	}

	// Validate output if configured
	if e.security.config.ValidateBeforeOutput {
		if err := e.security.outputManager.ValidateOutput(buf.Bytes(), findings); err != nil {
			e.security.logger.Error("output validation failed",
				"error", err,
				"format", "csv",
			)
			return fmt.Errorf("output contains unmasked PI: %w", err)
		}
	}

	// Write validated output
	_, err := w.Write(buf.Bytes())
	return err
}

// SecureHTMLGenerator wraps HTML generation with security controls
type SecureHTMLGenerator struct {
	security *SecurityLayer
}

// NewSecureHTMLGenerator creates a secure HTML generator
func (s *SecurityLayer) NewSecureHTMLGenerator() *SecureHTMLGenerator {
	return &SecureHTMLGenerator{
		security: s,
	}
}

// GenerateReport generates an HTML report with security controls
func (g *SecureHTMLGenerator) GenerateReport(findings []detection.Finding, metadata ExportMetadata) ([]byte, error) {
	// Log the operation
	if g.security.config.LogAllOperations {
		g.security.logger.Info("generating HTML report",
			"finding_count", len(findings),
			"repository", metadata.Repository,
		)
	}

	// Prepare findings with masking
	maskedFindings := findings
	if g.security.config.EnforceMasking {
		maskedFindings = g.security.outputManager.PrepareFindings(findings)
	}

	// Create template data
	templateData := g.createTemplateData(maskedFindings, metadata)

	// Generate HTML
	generator := NewHTMLGenerator()
	output, err := generator.Generate(templateData)
	if err != nil {
		return nil, fmt.Errorf("HTML generation failed: %w", err)
	}

	// Validate output if configured
	if g.security.config.ValidateBeforeOutput {
		if err := g.security.outputManager.ValidateOutput(output, findings); err != nil {
			g.security.logger.Error("output validation failed",
				"error", err,
				"format", "html",
			)
			return nil, fmt.Errorf("output contains unmasked PI: %w", err)
		}
	}

	return output, nil
}

// createTemplateData creates template data from findings
func (g *SecureHTMLGenerator) createTemplateData(findings []detection.Finding, metadata ExportMetadata) *HTMLTemplateData {
	// Group findings by risk level
	var critical, high, medium, low []Finding

	for _, f := range findings {
		htmlFinding := Finding{
			ID:          fmt.Sprintf("finding-%d", f.Line),
			Type:        string(f.Type),
			TypeDisplay: getPITypeDisplay(f.Type),
			RiskLevel:   string(f.RiskLevel),
			File:        f.File,
			Line:        f.Line,
			Column:      f.Column,
			Match:       f.Match, // Already masked by PrepareFindings
			MaskedMatch: f.Match, // Same as Match when pre-masked
			Context:     f.Context,
			Validated:   f.Validated,
		}

		switch f.RiskLevel {
		case detection.RiskLevelCritical:
			critical = append(critical, htmlFinding)
		case detection.RiskLevelHigh:
			high = append(high, htmlFinding)
		case detection.RiskLevelMedium:
			medium = append(medium, htmlFinding)
		case detection.RiskLevelLow:
			low = append(low, htmlFinding)
		}
	}

	return &HTMLTemplateData{
		ReportID:     metadata.ScanID,
		GeneratedAt:  metadata.Timestamp,
		ScanDuration: metadata.ScanDuration.String(),
		ToolVersion:  metadata.ToolVersion,
		Repository: RepositoryInfo{
			Name:       metadata.Repository,
			Branch:     metadata.Branch,
			CommitHash: metadata.CommitHash,
		},
		Summary: ScanSummary{
			TotalFindings: len(findings),
			CriticalCount: len(critical),
			HighCount:     len(high),
			MediumCount:   len(medium),
			LowCount:      len(low),
		},
		CriticalFindings: critical,
		HighFindings:     high,
		MediumFindings:   medium,
		LowFindings:      low,
	}
}

// SecureSARIFExporter wraps SARIF export with security controls
type SecureSARIFExporter struct {
	*SARIFExporter
	security *SecurityLayer
}

// NewSecureSARIFExporter creates a secure SARIF exporter
func (s *SecurityLayer) NewSecureSARIFExporter() *SecureSARIFExporter {
	return &SecureSARIFExporter{
		SARIFExporter: NewSARIFExporter("pi-scanner", "1.0.0", "https://github.com/MacAttak/pi-scanner"),
		security:      s,
	}
}

// Export generates a SARIF report with security controls
func (e *SecureSARIFExporter) Export(w io.Writer, findings []detection.Finding, metadata ExportMetadata) error {
	// Log the operation
	if e.security.config.LogAllOperations {
		e.security.logger.Info("generating SARIF report",
			"finding_count", len(findings),
			"repository", metadata.Repository,
		)
	}

	// Prepare findings with masking
	maskedFindings := findings
	if e.security.config.EnforceMasking {
		maskedFindings = e.security.outputManager.PrepareFindings(findings)
	}

	// Generate the report
	var buf bytes.Buffer
	if err := e.SARIFExporter.Export(&buf, maskedFindings, metadata); err != nil {
		return fmt.Errorf("SARIF generation failed: %w", err)
	}

	// Validate output if configured
	if e.security.config.ValidateBeforeOutput {
		if err := e.security.outputManager.ValidateOutput(buf.Bytes(), findings); err != nil {
			e.security.logger.Error("output validation failed",
				"error", err,
				"format", "sarif",
			)
			return fmt.Errorf("output contains unmasked PI: %w", err)
		}
	}

	// Write validated output
	_, err := w.Write(buf.Bytes())
	return err
}

// ReportFactory creates secure report generators
type ReportFactory struct {
	security *SecurityLayer
}

// NewReportFactory creates a new report factory
func NewReportFactory(outputManager *output.Manager, config *SecurityConfig) *ReportFactory {
	security := NewSecurityLayer(outputManager, config, nil)
	return &ReportFactory{
		security: security,
	}
}

// CreateCSVExporter creates a secure CSV exporter
func (f *ReportFactory) CreateCSVExporter(opts ...CSVExporterOption) *SecureCSVExporter {
	return f.security.NewSecureCSVExporter(opts...)
}

// CreateHTMLGenerator creates a secure HTML generator
func (f *ReportFactory) CreateHTMLGenerator() *SecureHTMLGenerator {
	return f.security.NewSecureHTMLGenerator()
}

// CreateSARIFExporter creates a secure SARIF exporter
func (f *ReportFactory) CreateSARIFExporter() *SecureSARIFExporter {
	return f.security.NewSecureSARIFExporter()
}

// CreateExporter creates an exporter for the specified format
func (f *ReportFactory) CreateExporter(format string, opts ...interface{}) (interface{}, error) {
	switch format {
	case "csv":
		csvOpts := make([]CSVExporterOption, 0)
		for _, opt := range opts {
			if csvOpt, ok := opt.(CSVExporterOption); ok {
				csvOpts = append(csvOpts, csvOpt)
			}
		}
		return f.CreateCSVExporter(csvOpts...), nil

	case "html":
		return f.CreateHTMLGenerator(), nil

	case "sarif":
		return f.CreateSARIFExporter(), nil

	default:
		return nil, fmt.Errorf("unsupported format: %s", format)
	}
}
