package report_test

import (
	"bytes"
	"fmt"
	"time"

	"github.com/MacAttak/pi-scanner/pkg/detection"
	"github.com/MacAttak/pi-scanner/pkg/report"
)

func ExampleCSVExporter_Export() {
	// Create sample records
	records := []report.CSVRecord{
		{
			Timestamp:       time.Date(2024, 1, 15, 14, 30, 0, 0, time.UTC),
			Repository:      "example-repo",
			Branch:          "main",
			FilePath:        "src/customer.go",
			LineNumber:      42,
			PIType:          "TFN",
			PITypeDisplay:   "Tax File Number",
			MaskedMatch:     "123****89",
			Validated:       true,
			ConfidenceScore: 0.95,
			RiskLevel:       "CRITICAL",
		},
	}

	// Create exporter with masked values
	exporter := report.NewCSVExporter(report.WithMaskedValues())

	// Export to buffer
	var buf bytes.Buffer
	if err := exporter.Export(&buf, records); err != nil {
		panic(err)
	}

	// Print first few lines
	lines := bytes.Split(buf.Bytes(), []byte("\n"))
	for i := 0; i < 2 && i < len(lines); i++ {
		fmt.Println(string(lines[i]))
	}

	// Output:
	// Timestamp,Repository,Branch,File Path,Line,Column,PI Type,PI Type Display,Validated,Test Data,Confidence Score,Risk Level,Risk Score,Masked Value,Impact Score,Likelihood Score,Exposure Score,Risk Category,Environment,APRA Relevant,Privacy Act Issue,Notifiable Breach
	// 2024-01-15 14:30:00,example-repo,main,src/customer.go,42,0,TFN,Tax File Number,true,false,0.95,CRITICAL,0.00,123****89,0.00,0.00,0.00,,,false,false,false
}

func ExampleCSVExporter_ExportFindings() {
	// Create sample findings
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

	// Create metadata
	metadata := report.ExportMetadata{
		ScanID:       "scan-123",
		Repository:   "example-repo",
		Branch:       "main",
		CommitHash:   "abc123",
		ScanDuration: 2 * time.Minute,
		ToolVersion:  "1.0.0",
		Timestamp:    time.Date(2024, 1, 15, 14, 30, 0, 0, time.UTC),
	}

	// Create exporter
	exporter := report.NewCSVExporter(report.WithMaskedValues())

	// Export findings
	var buf bytes.Buffer
	if err := exporter.ExportFindings(&buf, findings, metadata); err != nil {
		panic(err)
	}

	fmt.Println("CSV export completed successfully")
	// Output: CSV export completed successfully
}

func ExampleCSVSummaryExporter_ExportSummary() {
	// Create summary data
	summary := report.ScanSummary{
		TotalFindings:  100,
		CriticalCount:  10,
		HighCount:      20,
		MediumCount:    30,
		LowCount:       40,
		ValidatedCount: 60,
	}

	// Create metadata
	metadata := report.ExportMetadata{
		Repository:   "example-repo",
		Branch:       "main",
		Timestamp:    time.Date(2024, 1, 15, 14, 30, 0, 0, time.UTC),
		ScanDuration: 2 * time.Minute,
	}

	// Create summary exporter
	exporter := report.NewCSVSummaryExporter()

	// Export summary
	var buf bytes.Buffer
	if err := exporter.ExportSummary(&buf, summary, metadata); err != nil {
		panic(err)
	}

	// Print first few lines
	lines := bytes.Split(buf.Bytes(), []byte("\n"))
	for i := 0; i < 4 && i < len(lines); i++ {
		fmt.Println(string(lines[i]))
	}

	// Output:
	// Metric,Value,Percentage
	// Repository,example-repo,
	// Branch,main,
	// Scan Date,2024-01-15 14:30:00,
}
