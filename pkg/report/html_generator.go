package report

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
)

// HTMLGenerator generates HTML reports
type HTMLGenerator struct {
	templates *template.Template
}

// NewHTMLGenerator creates a new HTML generator
func NewHTMLGenerator() *HTMLGenerator {
	// Create a basic template for now
	// In production, this would load from embedded templates
	tmpl := template.Must(template.New("report").Parse(basicHTMLTemplate))

	return &HTMLGenerator{
		templates: tmpl,
	}
}

// Generate creates an HTML report from template data
func (g *HTMLGenerator) Generate(data *HTMLTemplateData) ([]byte, error) {
	var buf bytes.Buffer

	if err := g.templates.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("template execution failed: %w", err)
	}

	return buf.Bytes(), nil
}

// GenerateJSON generates the template data as JSON (for debugging)
func (g *HTMLGenerator) GenerateJSON(data *HTMLTemplateData) ([]byte, error) {
	return json.MarshalIndent(data, "", "  ")
}

// basicHTMLTemplate is a minimal template for testing
const basicHTMLTemplate = `<!DOCTYPE html>
<html>
<head>
    <title>PI Scanner Report</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; }
        .header { background-color: #f0f0f0; padding: 20px; }
        .summary { margin: 20px 0; }
        .findings { margin: 20px 0; }
        .finding { margin: 10px 0; padding: 10px; border: 1px solid #ddd; }
        .critical { border-color: #d9534f; }
        .high { border-color: #f0ad4e; }
        .medium { border-color: #5bc0de; }
        .low { border-color: #5cb85c; }
    </style>
</head>
<body>
    <div class="header">
        <h1>PI Scanner Report</h1>
        <p>Repository: {{.Repository.Name}}</p>
        <p>Branch: {{.Repository.Branch}}</p>
        <p>Generated: {{.GeneratedAt.Format "2006-01-02 15:04:05"}}</p>
    </div>

    <div class="summary">
        <h2>Summary</h2>
        <p>Total Findings: {{.Summary.TotalFindings}}</p>
        <ul>
            <li>Critical: {{.Summary.CriticalCount}}</li>
            <li>High: {{.Summary.HighCount}}</li>
            <li>Medium: {{.Summary.MediumCount}}</li>
            <li>Low: {{.Summary.LowCount}}</li>
        </ul>
    </div>

    <div class="findings">
        <h2>Findings</h2>

        {{if .CriticalFindings}}
        <h3>Critical Risk</h3>
        {{range .CriticalFindings}}
        <div class="finding critical">
            <p><strong>Type:</strong> {{.TypeDisplay}}</p>
            <p><strong>File:</strong> {{.File}}:{{.Line}}</p>
            <p><strong>Value:</strong> {{.MaskedMatch}}</p>
        </div>
        {{end}}
        {{end}}

        {{if .HighFindings}}
        <h3>High Risk</h3>
        {{range .HighFindings}}
        <div class="finding high">
            <p><strong>Type:</strong> {{.TypeDisplay}}</p>
            <p><strong>File:</strong> {{.File}}:{{.Line}}</p>
            <p><strong>Value:</strong> {{.MaskedMatch}}</p>
        </div>
        {{end}}
        {{end}}

        {{if .MediumFindings}}
        <h3>Medium Risk</h3>
        {{range .MediumFindings}}
        <div class="finding medium">
            <p><strong>Type:</strong> {{.TypeDisplay}}</p>
            <p><strong>File:</strong> {{.File}}:{{.Line}}</p>
            <p><strong>Value:</strong> {{.MaskedMatch}}</p>
        </div>
        {{end}}
        {{end}}

        {{if .LowFindings}}
        <h3>Low Risk</h3>
        {{range .LowFindings}}
        <div class="finding low">
            <p><strong>Type:</strong> {{.TypeDisplay}}</p>
            <p><strong>File:</strong> {{.File}}:{{.Line}}</p>
            <p><strong>Value:</strong> {{.MaskedMatch}}</p>
        </div>
        {{end}}
        {{end}}
    </div>
</body>
</html>`
