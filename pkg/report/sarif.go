package report

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/MacAttak/pi-scanner/pkg/detection"
	"github.com/MacAttak/pi-scanner/pkg/scoring"
)

// SARIF version and schema constants
const (
	SARIFVersion = "2.1.0"
	SARIFSchema  = "https://json.schemastore.org/sarif-2.1.0.json"
)

// SARIFReport represents the top-level SARIF log
type SARIFReport struct {
	Version string     `json:"version"`
	Schema  string     `json:"$schema"`
	Runs    []SARIFRun `json:"runs"`
}

// SARIFRun represents a single run of the analysis tool
type SARIFRun struct {
	Tool               SARIFTool                  `json:"tool"`
	Results            []SARIFResult              `json:"results"`
	ArtifactLocations  []SARIFArtifactLocation    `json:"artifacts,omitempty"`
	LogicalLocations   []SARIFLogicalLocation     `json:"logicalLocations,omitempty"`
	Invocations        []SARIFInvocation          `json:"invocations,omitempty"`
	OriginalURIBaseIDs map[string]SARIFURIBaseID  `json:"originalUriBaseIds,omitempty"`
	Properties         map[string]interface{}     `json:"properties,omitempty"`
}

// SARIFTool describes the analysis tool
type SARIFTool struct {
	Driver SARIFToolComponent `json:"driver"`
}

// SARIFToolComponent contains tool details
type SARIFToolComponent struct {
	Name            string                `json:"name"`
	Version         string                `json:"version,omitempty"`
	InformationURI  string                `json:"informationUri,omitempty"`
	Rules           []SARIFRule           `json:"rules,omitempty"`
	Notifications   []SARIFNotification   `json:"notifications,omitempty"`
	SemanticVersion string                `json:"semanticVersion,omitempty"`
	Properties      map[string]interface{} `json:"properties,omitempty"`
}

// SARIFRule represents a static analysis rule
type SARIFRule struct {
	ID                   string                    `json:"id"`
	Name                 string                    `json:"name,omitempty"`
	ShortDescription     SARIFMultiformatMessage   `json:"shortDescription,omitempty"`
	FullDescription      SARIFMultiformatMessage   `json:"fullDescription,omitempty"`
	Help                 SARIFMultiformatMessage   `json:"help,omitempty"`
	DefaultConfiguration SARIFReportingConfiguration `json:"defaultConfiguration,omitempty"`
	Properties           map[string]interface{}     `json:"properties,omitempty"`
}

// SARIFMultiformatMessage contains text in multiple formats
type SARIFMultiformatMessage struct {
	Text     string `json:"text"`
	Markdown string `json:"markdown,omitempty"`
}

// SARIFReportingConfiguration contains rule configuration
type SARIFReportingConfiguration struct {
	Enabled bool    `json:"enabled"`
	Level   string  `json:"level,omitempty"`
	Rank    float64 `json:"rank,omitempty"`
}

// SARIFResult represents a single finding
type SARIFResult struct {
	RuleID              string                  `json:"ruleId"`
	RuleIndex           int                     `json:"ruleIndex,omitempty"`
	Level               string                  `json:"level,omitempty"`
	Message             SARIFMessage            `json:"message"`
	Locations           []SARIFLocation         `json:"locations"`
	PartialFingerprints map[string]string       `json:"partialFingerprints,omitempty"`
	Fingerprints        map[string]string       `json:"fingerprints,omitempty"`
	CodeFlows           []SARIFCodeFlow         `json:"codeFlows,omitempty"`
	Fixes               []SARIFFix              `json:"fixes,omitempty"`
	Properties          map[string]interface{}  `json:"properties,omitempty"`
	Rank                float64                 `json:"rank,omitempty"`
}

// SARIFMessage contains the result message
type SARIFMessage struct {
	Text      string   `json:"text,omitempty"`
	Markdown  string   `json:"markdown,omitempty"`
	ID        string   `json:"id,omitempty"`
	Arguments []string `json:"arguments,omitempty"`
}

// SARIFLocation represents a location in code
type SARIFLocation struct {
	PhysicalLocation SARIFPhysicalLocation  `json:"physicalLocation,omitempty"`
	LogicalLocation  SARIFLogicalLocation   `json:"logicalLocation,omitempty"`
	Message          *SARIFMessage          `json:"message,omitempty"`
	Annotations      []SARIFAnnotation      `json:"annotations,omitempty"`
	Properties       map[string]interface{} `json:"properties,omitempty"`
}

// SARIFPhysicalLocation represents a physical location in a file
type SARIFPhysicalLocation struct {
	ArtifactLocation SARIFArtifactLocation  `json:"artifactLocation"`
	Region           SARIFRegion            `json:"region,omitempty"`
	ContextRegion    SARIFRegion            `json:"contextRegion,omitempty"`
	Properties       map[string]interface{} `json:"properties,omitempty"`
}

// SARIFArtifactLocation represents a file location
type SARIFArtifactLocation struct {
	URI        string                 `json:"uri"`
	URIBaseID  string                 `json:"uriBaseId,omitempty"`
	Index      int                    `json:"index,omitempty"`
	Properties map[string]interface{} `json:"properties,omitempty"`
}

// SARIFRegion represents a region in a file
type SARIFRegion struct {
	StartLine   int                    `json:"startLine,omitempty"`
	StartColumn int                    `json:"startColumn,omitempty"`
	EndLine     int                    `json:"endLine,omitempty"`
	EndColumn   int                    `json:"endColumn,omitempty"`
	CharOffset  int                    `json:"charOffset,omitempty"`
	CharLength  int                    `json:"charLength,omitempty"`
	Snippet     *SARIFContent          `json:"snippet,omitempty"`
	Properties  map[string]interface{} `json:"properties,omitempty"`
}

// SARIFContent represents code content
type SARIFContent struct {
	Text       string                 `json:"text,omitempty"`
	Binary     string                 `json:"binary,omitempty"`
	Properties map[string]interface{} `json:"properties,omitempty"`
}

// SARIFLogicalLocation represents a logical location
type SARIFLogicalLocation struct {
	Name               string                 `json:"name,omitempty"`
	Index              int                    `json:"index,omitempty"`
	FullyQualifiedName string                 `json:"fullyQualifiedName,omitempty"`
	DecoratedName      string                 `json:"decoratedName,omitempty"`
	ParentIndex        int                    `json:"parentIndex,omitempty"`
	Kind               string                 `json:"kind,omitempty"`
	Properties         map[string]interface{} `json:"properties,omitempty"`
}

// SARIFCodeFlow represents code flow
type SARIFCodeFlow struct {
	ThreadFlows []SARIFThreadFlow      `json:"threadFlows"`
	Properties  map[string]interface{} `json:"properties,omitempty"`
}

// SARIFThreadFlow represents thread flow
type SARIFThreadFlow struct {
	ID         string                     `json:"id,omitempty"`
	Message    *SARIFMessage              `json:"message,omitempty"`
	Locations  []SARIFThreadFlowLocation  `json:"locations"`
	Properties map[string]interface{}     `json:"properties,omitempty"`
}

// SARIFThreadFlowLocation represents a location in thread flow
type SARIFThreadFlowLocation struct {
	Location    SARIFLocation          `json:"location,omitempty"`
	Stack       *SARIFStack            `json:"stack,omitempty"`
	Kinds       []string               `json:"kinds,omitempty"`
	Taxa        []string               `json:"taxa,omitempty"`
	Module      string                 `json:"module,omitempty"`
	State       map[string]interface{} `json:"state,omitempty"`
	NestingLevel int                   `json:"nestingLevel,omitempty"`
	ExecutionOrder int                  `json:"executionOrder,omitempty"`
	ExecutionTimeUTC string              `json:"executionTimeUtc,omitempty"`
	Importance   string                `json:"importance,omitempty"`
	Properties   map[string]interface{} `json:"properties,omitempty"`
}

// SARIFStack represents a call stack
type SARIFStack struct {
	Message    *SARIFMessage          `json:"message,omitempty"`
	Frames     []SARIFStackFrame      `json:"frames"`
	Properties map[string]interface{} `json:"properties,omitempty"`
}

// SARIFStackFrame represents a stack frame
type SARIFStackFrame struct {
	Location   SARIFLocation          `json:"location,omitempty"`
	Module     string                 `json:"module,omitempty"`
	ThreadID   int                    `json:"threadId,omitempty"`
	Parameters []string               `json:"parameters,omitempty"`
	Properties map[string]interface{} `json:"properties,omitempty"`
}

// SARIFFix represents a proposed fix
type SARIFFix struct {
	Description       SARIFMessage             `json:"description,omitempty"`
	ArtifactChanges   []SARIFArtifactChange    `json:"artifactChanges"`
	Properties        map[string]interface{}   `json:"properties,omitempty"`
}

// SARIFArtifactChange represents a change to an artifact
type SARIFArtifactChange struct {
	ArtifactLocation SARIFArtifactLocation  `json:"artifactLocation"`
	Replacements     []SARIFReplacement     `json:"replacements"`
	Properties       map[string]interface{} `json:"properties,omitempty"`
}

// SARIFReplacement represents a replacement
type SARIFReplacement struct {
	DeletedRegion SARIFRegion            `json:"deletedRegion"`
	InsertedContent *SARIFContent         `json:"insertedContent,omitempty"`
	Properties      map[string]interface{} `json:"properties,omitempty"`
}

// SARIFAnnotation represents an annotation
type SARIFAnnotation struct {
	Location    SARIFLocation          `json:"location"`
	Message     SARIFMessage           `json:"message"`
	Properties  map[string]interface{} `json:"properties,omitempty"`
}

// SARIFNotification represents a notification
type SARIFNotification struct {
	ID              string                    `json:"id"`
	Name            string                    `json:"name,omitempty"`
	ShortDescription SARIFMultiformatMessage  `json:"shortDescription,omitempty"`
	FullDescription  SARIFMultiformatMessage  `json:"fullDescription,omitempty"`
	Help            SARIFMultiformatMessage   `json:"help,omitempty"`
	DefaultConfiguration SARIFReportingConfiguration `json:"defaultConfiguration,omitempty"`
	Properties      map[string]interface{}    `json:"properties,omitempty"`
}

// SARIFInvocation represents tool invocation details
type SARIFInvocation struct {
	CommandLine         string                   `json:"commandLine,omitempty"`
	Arguments           []string                 `json:"arguments,omitempty"`
	ResponseFiles       []SARIFArtifactLocation  `json:"responseFiles,omitempty"`
	StartTimeUTC        string                   `json:"startTimeUtc,omitempty"`
	EndTimeUTC          string                   `json:"endTimeUtc,omitempty"`
	ExecutionSuccessful bool                     `json:"executionSuccessful"`
	ExitCode            int                      `json:"exitCode,omitempty"`
	WorkingDirectory    SARIFArtifactLocation    `json:"workingDirectory,omitempty"`
	EnvironmentVariables map[string]string       `json:"environmentVariables,omitempty"`
	Account             string                   `json:"account,omitempty"`
	ProcessID           int                      `json:"processId,omitempty"`
	ExecutableLocation  SARIFArtifactLocation    `json:"executableLocation,omitempty"`
	Properties          map[string]interface{}   `json:"properties,omitempty"`
}

// SARIFURIBaseID represents a URI base identifier
type SARIFURIBaseID struct {
	URI         string                 `json:"uri"`
	Description SARIFMultiformatMessage `json:"description,omitempty"`
	Properties  map[string]interface{} `json:"properties,omitempty"`
}

// SARIFExporter handles SARIF report generation
type SARIFExporter struct {
	toolName    string
	toolVersion string
	infoURI     string
	baseURI     string
}

// NewSARIFExporter creates a new SARIF exporter
func NewSARIFExporter(toolName, toolVersion, infoURI string) *SARIFExporter {
	return &SARIFExporter{
		toolName:    toolName,
		toolVersion: toolVersion,
		infoURI:     infoURI,
	}
}

// SetBaseURI sets the base URI for relative paths
func (e *SARIFExporter) SetBaseURI(baseURI string) {
	e.baseURI = baseURI
}

// Export writes findings to SARIF format
func (e *SARIFExporter) Export(w io.Writer, findings []detection.Finding, metadata ExportMetadata) error {
	report := e.createReport(findings, metadata)
	
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}

// ExportWithRiskAssessment exports findings with risk assessment data
func (e *SARIFExporter) ExportWithRiskAssessment(w io.Writer, findings []IntegrationRecord, metadata ExportMetadata) error {
	report := e.createReportWithRisk(findings, metadata)
	
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}

// createReport creates a SARIF report from findings
func (e *SARIFExporter) createReport(findings []detection.Finding, metadata ExportMetadata) *SARIFReport {
	run := SARIFRun{
		Tool: SARIFTool{
			Driver: SARIFToolComponent{
				Name:            e.toolName,
				Version:         e.toolVersion,
				SemanticVersion: e.toolVersion,
				InformationURI:  e.infoURI,
				Rules:           e.createRules(),
				Properties: map[string]interface{}{
					"tags": []string{"security", "privacy", "compliance", "australia"},
				},
			},
		},
		Invocations: []SARIFInvocation{
			{
				StartTimeUTC:        metadata.Timestamp.UTC().Format(time.RFC3339),
				EndTimeUTC:          metadata.Timestamp.Add(metadata.ScanDuration).UTC().Format(time.RFC3339),
				ExecutionSuccessful: true,
				Properties: map[string]interface{}{
					"repository": metadata.Repository,
					"branch":     metadata.Branch,
					"commitHash": metadata.CommitHash,
				},
			},
		},
		Properties: map[string]interface{}{
			"scanID":       metadata.ScanID,
			"scanDuration": metadata.ScanDuration.Seconds(),
		},
	}

	// Convert findings to results
	run.Results = e.convertFindings(findings)
	
	// Set base URI if configured
	if e.baseURI != "" {
		run.OriginalURIBaseIDs = map[string]SARIFURIBaseID{
			"REPO_ROOT": {
				URI: e.baseURI,
				Description: SARIFMultiformatMessage{
					Text: "Repository root directory",
				},
			},
		}
	}

	return &SARIFReport{
		Version: SARIFVersion,
		Schema:  SARIFSchema,
		Runs:    []SARIFRun{run},
	}
}

// createReportWithRisk creates a SARIF report with risk assessment data
func (e *SARIFExporter) createReportWithRisk(findings []IntegrationRecord, metadata ExportMetadata) *SARIFReport {
	// Convert to basic findings first
	basicFindings := make([]detection.Finding, len(findings))
	for i, ir := range findings {
		basicFindings[i] = ir.Finding
	}
	
	report := e.createReport(basicFindings, metadata)
	
	// Enhance results with risk data
	for i, ir := range findings {
		if i < len(report.Runs[0].Results) {
			result := &report.Runs[0].Results[i]
			
			// Add risk assessment properties
			if ir.RiskAssessment != nil {
				result.Properties["riskAssessment"] = map[string]interface{}{
					"overallRisk":     ir.RiskAssessment.OverallRisk,
					"riskLevel":       string(ir.RiskAssessment.RiskLevel),
					"riskCategory":    string(ir.RiskAssessment.RiskCategory),
					"impactScore":     ir.RiskAssessment.ImpactScore,
					"likelihoodScore": ir.RiskAssessment.LikelihoodScore,
					"exposureScore":   ir.RiskAssessment.ExposureScore,
				}
				
				// Set SARIF level based on risk level
				result.Level = e.mapRiskLevelToSARIF(ir.RiskAssessment.RiskLevel)
				result.Rank = ir.RiskAssessment.OverallRisk * 100 // 0-100 scale
				
				// Add compliance flags
				result.Properties["compliance"] = map[string]interface{}{
					"apraReporting":        ir.RiskAssessment.ComplianceFlags.APRAReporting,
					"privacyActBreach":     ir.RiskAssessment.ComplianceFlags.PrivacyActBreach,
					"notifiableDataBreach": ir.RiskAssessment.ComplianceFlags.NotifiableDataBreach,
				}
				
				// Add fixes based on mitigations
				if len(ir.RiskAssessment.Mitigations) > 0 {
					result.Fixes = e.createFixes(ir.Finding, ir.RiskAssessment.Mitigations)
				}
			}
			
			// Add additional properties
			result.Properties["confidenceScore"] = ir.ConfidenceScore
			result.Properties["environment"] = ir.Environment
			result.Properties["proximityInfo"] = ir.ProximityInfo
		}
	}
	
	return report
}

// createRules creates SARIF rules for PI types
func (e *SARIFExporter) createRules() []SARIFRule {
	piTypes := []struct {
		id          string
		name        string
		description string
		help        string
		level       string
		rank        float64
	}{
		{
			id:          "PI001",
			name:        "Tax File Number (TFN)",
			description: "Australian Tax File Number detected",
			help:        "TFNs are highly sensitive personal identifiers. Remove or securely vault these values.",
			level:       "error",
			rank:        100,
		},
		{
			id:          "PI002",
			name:        "Medicare Number",
			description: "Australian Medicare Number detected",
			help:        "Medicare numbers are sensitive health identifiers. Ensure proper protection and compliance with health privacy laws.",
			level:       "error",
			rank:        95,
		},
		{
			id:          "PI003",
			name:        "Australian Business Number (ABN)",
			description: "Australian Business Number detected",
			help:        "While ABNs are public, they may indicate business relationships. Review context for sensitivity.",
			level:       "warning",
			rank:        70,
		},
		{
			id:          "PI004",
			name:        "Bank State Branch (BSB)",
			description: "Australian BSB code detected",
			help:        "BSB codes with account numbers enable financial transactions. Protect accordingly.",
			level:       "warning",
			rank:        80,
		},
		{
			id:          "PI005",
			name:        "Credit Card Number",
			description: "Credit card number detected",
			help:        "Credit card numbers must be protected according to PCI-DSS standards. Never store in plain text.",
			level:       "error",
			rank:        90,
		},
		{
			id:          "PI006",
			name:        "Email Address",
			description: "Email address detected",
			help:        "Email addresses are personal data. Ensure compliance with privacy regulations.",
			level:       "note",
			rank:        40,
		},
		{
			id:          "PI007",
			name:        "Phone Number",
			description: "Phone number detected",
			help:        "Phone numbers are personal identifiers. Consider the context and protect appropriately.",
			level:       "note",
			rank:        50,
		},
		{
			id:          "PI008",
			name:        "Personal Name",
			description: "Personal name detected",
			help:        "Names are personal data. Ensure proper handling according to privacy laws.",
			level:       "note",
			rank:        30,
		},
		{
			id:          "PI009",
			name:        "Physical Address",
			description: "Physical address detected",
			help:        "Addresses are sensitive location data. Protect according to privacy requirements.",
			level:       "warning",
			rank:        60,
		},
		{
			id:          "PI010",
			name:        "Passport Number",
			description: "Passport number detected",
			help:        "Passport numbers are government-issued identifiers. Require strong protection.",
			level:       "error",
			rank:        85,
		},
		{
			id:          "PI011",
			name:        "Driver License",
			description: "Driver license number detected",
			help:        "Driver license numbers are government-issued identifiers. Protect appropriately.",
			level:       "warning",
			rank:        75,
		},
		{
			id:          "PI012",
			name:        "IP Address",
			description: "IP address detected",
			help:        "IP addresses can be personal data in some jurisdictions. Review context.",
			level:       "note",
			rank:        20,
		},
	}

	rules := make([]SARIFRule, len(piTypes))
	for i, pt := range piTypes {
		rules[i] = SARIFRule{
			ID:   pt.id,
			Name: pt.name,
			ShortDescription: SARIFMultiformatMessage{
				Text: pt.description,
			},
			FullDescription: SARIFMultiformatMessage{
				Text:     pt.description,
				Markdown: fmt.Sprintf("**%s**: %s", pt.name, pt.description),
			},
			Help: SARIFMultiformatMessage{
				Text:     pt.help,
				Markdown: fmt.Sprintf("### Remediation\n\n%s", pt.help),
			},
			DefaultConfiguration: SARIFReportingConfiguration{
				Enabled: true,
				Level:   pt.level,
				Rank:    pt.rank,
			},
			Properties: map[string]interface{}{
				"tags": []string{"security", "privacy", "pii"},
				"precision": "high",
			},
		}
	}

	return rules
}

// convertFindings converts detection findings to SARIF results
func (e *SARIFExporter) convertFindings(findings []detection.Finding) []SARIFResult {
	results := make([]SARIFResult, len(findings))
	
	for i, finding := range findings {
		ruleID := e.getRuleID(finding.Type)
		
		// Create location
		location := SARIFLocation{
			PhysicalLocation: SARIFPhysicalLocation{
				ArtifactLocation: SARIFArtifactLocation{
					URI: e.normalizeURI(finding.File),
				},
				Region: SARIFRegion{
					StartLine:   finding.Line,
					StartColumn: finding.Column,
					EndLine:     finding.Line,
					EndColumn:   finding.Column + len(finding.Match),
				},
			},
		}
		
		// Add code snippet if available
		if finding.Context != "" {
			location.PhysicalLocation.Region.Snippet = &SARIFContent{
				Text: finding.Context,
			}
		}
		
		// Create fingerprints for deduplication
		fingerprints := map[string]string{
			"primaryLocationLineHash": e.createFingerprint(finding),
		}
		
		results[i] = SARIFResult{
			RuleID: ruleID,
			Level:  e.getDefaultLevel(finding.Type),
			Message: SARIFMessage{
				Text: fmt.Sprintf("%s detected: %s", 
					getPITypeDisplay(finding.Type),
					maskSensitiveData(finding.Match, string(finding.Type))),
			},
			Locations: []SARIFLocation{location},
			PartialFingerprints: fingerprints,
			Properties: map[string]interface{}{
				"piType":    string(finding.Type),
				"validated": finding.Validated,
				"match":     maskSensitiveData(finding.Match, string(finding.Type)),
			},
		}
		
		// Add rule index for efficiency
		if idx := e.getRuleIndex(finding.Type); idx >= 0 {
			results[i].RuleIndex = idx
		}
	}
	
	return results
}

// createFixes creates SARIF fixes from mitigations
func (e *SARIFExporter) createFixes(finding detection.Finding, mitigations []scoring.Mitigation) []SARIFFix {
	fixes := make([]SARIFFix, 0, len(mitigations))
	
	for _, mitigation := range mitigations {
		if mitigation.Priority == "CRITICAL" || mitigation.Priority == "HIGH" {
			fix := SARIFFix{
				Description: SARIFMessage{
					Text: fmt.Sprintf("%s: %s", mitigation.Title, mitigation.Description),
				},
				ArtifactChanges: []SARIFArtifactChange{
					{
						ArtifactLocation: SARIFArtifactLocation{
							URI: e.normalizeURI(finding.File),
						},
						Replacements: []SARIFReplacement{
							{
								DeletedRegion: SARIFRegion{
									StartLine:   finding.Line,
									StartColumn: finding.Column,
									EndLine:     finding.Line,
									EndColumn:   finding.Column + len(finding.Match),
								},
								InsertedContent: &SARIFContent{
									Text: "/* REDACTED - " + string(finding.Type) + " */",
								},
							},
						},
					},
				},
				Properties: map[string]interface{}{
					"priority": mitigation.Priority,
					"effort":   mitigation.Effort,
					"timeline": mitigation.Timeline,
					"category": mitigation.Category,
				},
			}
			fixes = append(fixes, fix)
		}
	}
	
	return fixes
}

// Helper methods

func (e *SARIFExporter) getRuleID(piType detection.PIType) string {
	ruleMap := map[detection.PIType]string{
		detection.PITypeTFN:           "PI001",
		detection.PITypeMedicare:      "PI002",
		detection.PITypeABN:           "PI003",
		detection.PITypeBSB:           "PI004",
		detection.PITypeCreditCard:    "PI005",
		detection.PITypeEmail:         "PI006",
		detection.PITypePhone:         "PI007",
		detection.PITypeName:          "PI008",
		detection.PITypeAddress:       "PI009",
		detection.PITypePassport:      "PI010",
		detection.PITypeDriverLicense: "PI011",
		detection.PITypeIP:            "PI012",
	}
	
	if id, exists := ruleMap[piType]; exists {
		return id
	}
	return "PI999" // Unknown
}

func (e *SARIFExporter) getRuleIndex(piType detection.PIType) int {
	indexMap := map[detection.PIType]int{
		detection.PITypeTFN:           0,
		detection.PITypeMedicare:      1,
		detection.PITypeABN:           2,
		detection.PITypeBSB:           3,
		detection.PITypeCreditCard:    4,
		detection.PITypeEmail:         5,
		detection.PITypePhone:         6,
		detection.PITypeName:          7,
		detection.PITypeAddress:       8,
		detection.PITypePassport:      9,
		detection.PITypeDriverLicense: 10,
		detection.PITypeIP:            11,
	}
	
	if idx, exists := indexMap[piType]; exists {
		return idx
	}
	return -1
}

func (e *SARIFExporter) getDefaultLevel(piType detection.PIType) string {
	// High sensitivity PI types
	highSensitivity := map[detection.PIType]bool{
		detection.PITypeTFN:        true,
		detection.PITypeMedicare:   true,
		detection.PITypeCreditCard: true,
		detection.PITypePassport:   true,
	}
	
	if highSensitivity[piType] {
		return "error"
	}
	
	// Medium sensitivity
	mediumSensitivity := map[detection.PIType]bool{
		detection.PITypeABN:           true,
		detection.PITypeBSB:           true,
		detection.PITypeAddress:       true,
		detection.PITypeDriverLicense: true,
	}
	
	if mediumSensitivity[piType] {
		return "warning"
	}
	
	return "note"
}

func (e *SARIFExporter) mapRiskLevelToSARIF(level scoring.RiskLevel) string {
	switch level {
	case scoring.RiskLevelCritical:
		return "error"
	case scoring.RiskLevelHigh:
		return "error"
	case scoring.RiskLevelMedium:
		return "warning"
	case scoring.RiskLevelLow:
		return "note"
	default:
		return "none"
	}
}

func (e *SARIFExporter) normalizeURI(filePath string) string {
	// Convert backslashes to forward slashes for consistency
	normalized := strings.ReplaceAll(filePath, "\\", "/")
	
	// Make relative to base URI if configured
	if e.baseURI != "" && strings.HasPrefix(normalized, e.baseURI) {
		normalized = strings.TrimPrefix(normalized, e.baseURI)
		normalized = strings.TrimPrefix(normalized, "/")
	}
	
	// Don't encode the path - SARIF expects readable paths
	// Only encode if it contains special characters that would break JSON
	return normalized
}

func (e *SARIFExporter) createFingerprint(finding detection.Finding) string {
	// Create a unique fingerprint for the finding
	data := fmt.Sprintf("%s:%s:%d:%s", 
		finding.File,
		finding.Type,
		finding.Line,
		finding.Match)
	
	// Simple hash for demonstration - in production use crypto hash
	return fmt.Sprintf("%x", uuid.NewSHA1(uuid.NameSpaceURL, []byte(data)))
}