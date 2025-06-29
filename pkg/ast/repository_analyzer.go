package ast

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// RepositoryAnalyzer performs repository-level analysis
type RepositoryAnalyzer struct {
	config   *Config
	analyzer *Analyzer
}

// RepositoryAnalysis contains the results of repository structure analysis
type RepositoryAnalysis struct {
	RepositoryPath    string                 `json:"repository_path"`
	AnalysisTime      time.Time              `json:"analysis_time"`
	Duration          time.Duration          `json:"duration"`
	TotalFiles        int                    `json:"total_files"`
	AnalyzedFiles     int                    `json:"analyzed_files"`
	SkippedFiles      int                    `json:"skipped_files"`
	LanguageStats     map[Language]int       `json:"language_stats"`
	RiskZoneStats     map[string]int         `json:"risk_zone_stats"`
	RiskLevelStats    map[RiskLevel]int      `json:"risk_level_stats"`
	SecurityHints     []SecurityHint         `json:"security_hints"`
	DependencyGraph   *DependencyGraph       `json:"dependency_graph"`
	RiskZones         []RiskZoneInfo         `json:"risk_zones"`
	BankingCompliance *BankingComplianceInfo `json:"banking_compliance"`
	FileAnalyses      []*AnalysisResult      `json:"file_analyses"`
}

// DependencyGraph represents the dependency structure of the repository
type DependencyGraph struct {
	Nodes []DependencyNode `json:"nodes"`
	Edges []DependencyEdge `json:"edges"`
}

// DependencyNode represents a node in the dependency graph
type DependencyNode struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Type        string    `json:"type"` // class, package, module
	FilePath    string    `json:"file_path"`
	RiskLevel   RiskLevel `json:"risk_level"`
	Connections int       `json:"connections"`
}

// DependencyEdge represents an edge in the dependency graph
type DependencyEdge struct {
	From   string `json:"from"`
	To     string `json:"to"`
	Type   string `json:"type"` // import, inheritance, composition
	Weight int    `json:"weight"`
}

// RiskZoneInfo provides detailed information about risk zones
type RiskZoneInfo struct {
	Zone          string    `json:"zone"`
	RiskLevel     RiskLevel `json:"risk_level"`
	FileCount     int       `json:"file_count"`
	Description   string    `json:"description"`
	FilePaths     []string  `json:"file_paths"`
	SecurityHints int       `json:"security_hints"`
}

// BankingComplianceInfo contains banking domain specific compliance information
type BankingComplianceInfo struct {
	DataModelFiles         []string            `json:"data_model_files"`
	CustomerDataFiles      []string            `json:"customer_data_files"`
	PaymentProcessingFiles []string            `json:"payment_processing_files"`
	AuthenticationFiles    []string            `json:"authentication_files"`
	ComplianceFindings     []ComplianceFinding `json:"compliance_findings"`
	RiskScore              float64             `json:"risk_score"`
	RecommendedActions     []string            `json:"recommended_actions"`
}

// ComplianceFinding represents a banking compliance finding
type ComplianceFinding struct {
	Type        string `json:"type"`
	Severity    string `json:"severity"`
	Description string `json:"description"`
	FilePath    string `json:"file_path"`
	LineNumber  int    `json:"line_number"`
	Regulation  string `json:"regulation"` // GDPR, CCPA, PCI-DSS, etc.
}

// NewRepositoryAnalyzer creates a new repository analyzer
func NewRepositoryAnalyzer(config *Config) *RepositoryAnalyzer {
	return &RepositoryAnalyzer{
		config:   config,
		analyzer: NewAnalyzer(config),
	}
}

// AnalyzeRepository performs comprehensive repository analysis
func (ra *RepositoryAnalyzer) AnalyzeRepository(ctx context.Context, repositoryPath string, filePaths []string) (*RepositoryAnalysis, error) {
	startTime := time.Now()

	analysis := &RepositoryAnalysis{
		RepositoryPath: repositoryPath,
		AnalysisTime:   startTime,
		TotalFiles:     len(filePaths),
		LanguageStats:  make(map[Language]int),
		RiskZoneStats:  make(map[string]int),
		RiskLevelStats: make(map[RiskLevel]int),
		SecurityHints:  []SecurityHint{},
		FileAnalyses:   []*AnalysisResult{},
	}

	// Analyze each file
	for _, filePath := range filePaths {
		if !ra.shouldAnalyzeFile(filePath) {
			analysis.SkippedFiles++
			continue
		}

		// Read file content (this would be passed in from the caller in real implementation)
		content, err := ra.readFile(filePath)
		if err != nil {
			analysis.SkippedFiles++
			continue
		}

		// Perform AST analysis
		result, err := ra.analyzer.AnalyzeFile(ctx, filePath, content)
		if err != nil {
			analysis.SkippedFiles++
			continue
		}

		// Update statistics
		analysis.AnalyzedFiles++
		analysis.LanguageStats[result.Language]++
		analysis.RiskZoneStats[result.RiskZone]++
		analysis.RiskLevelStats[result.RiskLevel]++

		// Collect security hints
		analysis.SecurityHints = append(analysis.SecurityHints, result.SecurityHints...)

		// Store file analysis
		analysis.FileAnalyses = append(analysis.FileAnalyses, result)
	}

	// Build dependency graph
	analysis.DependencyGraph = ra.buildDependencyGraph(analysis.FileAnalyses)

	// Analyze risk zones
	analysis.RiskZones = ra.analyzeRiskZones(analysis.FileAnalyses)

	// Perform banking compliance analysis
	analysis.BankingCompliance = ra.performBankingComplianceAnalysis(analysis.FileAnalyses)

	analysis.Duration = time.Since(startTime)

	return analysis, nil
}

// shouldAnalyzeFile determines if a file should be analyzed
func (ra *RepositoryAnalyzer) shouldAnalyzeFile(filePath string) bool {
	// Check if it's a supported language
	language := ra.analyzer.detectLanguage(filePath)
	if !ra.analyzer.isLanguageEnabled(language) {
		return false
	}

	// Check exclude patterns
	normalizedPath := strings.ToLower(filepath.ToSlash(filePath))
	for _, pattern := range ra.config.BankingDomainRules.ExcludePatterns {
		if matched, _ := filepath.Match(strings.ToLower(pattern), normalizedPath); matched {
			return false
		}
	}

	return true
}

// buildDependencyGraph creates a dependency graph from file analyses
func (ra *RepositoryAnalyzer) buildDependencyGraph(analyses []*AnalysisResult) *DependencyGraph {
	nodeMap := make(map[string]*DependencyNode)
	var edges []DependencyEdge

	// Create nodes for each class/module
	for _, analysis := range analyses {
		for _, class := range analysis.CodeStructure.Classes {
			nodeID := fmt.Sprintf("%s.%s", analysis.FilePath, class.Name)
			node := &DependencyNode{
				ID:        nodeID,
				Name:      class.Name,
				Type:      "class",
				FilePath:  analysis.FilePath,
				RiskLevel: analysis.RiskLevel,
			}
			nodeMap[nodeID] = node
		}

		// Create module-level node if no classes
		if len(analysis.CodeStructure.Classes) == 0 {
			nodeID := analysis.FilePath
			node := &DependencyNode{
				ID:        nodeID,
				Name:      filepath.Base(analysis.FilePath),
				Type:      "module",
				FilePath:  analysis.FilePath,
				RiskLevel: analysis.RiskLevel,
			}
			nodeMap[nodeID] = node
		}
	}

	// Create edges from dependencies
	for _, analysis := range analyses {
		for _, dep := range analysis.Dependencies {
			fromID := analysis.FilePath
			if dep.Name != "" {
				fromID = fmt.Sprintf("%s.%s", analysis.FilePath, dep.Name)
			}

			edge := DependencyEdge{
				From:   fromID,
				To:     dep.Target,
				Type:   dep.Type,
				Weight: 1,
			}
			edges = append(edges, edge)

			// Update connection counts
			if node, exists := nodeMap[fromID]; exists {
				node.Connections++
			}
		}
	}

	// Convert map to slice
	var nodes []DependencyNode
	for _, node := range nodeMap {
		nodes = append(nodes, *node)
	}

	return &DependencyGraph{
		Nodes: nodes,
		Edges: edges,
	}
}

// analyzeRiskZones analyzes and categorizes risk zones
func (ra *RepositoryAnalyzer) analyzeRiskZones(analyses []*AnalysisResult) []RiskZoneInfo {
	zoneMap := make(map[string]*RiskZoneInfo)

	for _, analysis := range analyses {
		zone := analysis.RiskZone

		if zoneInfo, exists := zoneMap[zone]; exists {
			zoneInfo.FileCount++
			zoneInfo.FilePaths = append(zoneInfo.FilePaths, analysis.FilePath)
			zoneInfo.SecurityHints += len(analysis.SecurityHints)
		} else {
			zoneInfo := &RiskZoneInfo{
				Zone:          zone,
				RiskLevel:     analysis.RiskLevel,
				FileCount:     1,
				Description:   ra.getZoneDescription(zone),
				FilePaths:     []string{analysis.FilePath},
				SecurityHints: len(analysis.SecurityHints),
			}
			zoneMap[zone] = zoneInfo
		}
	}

	// Convert map to sorted slice
	var zones []RiskZoneInfo
	for _, zone := range zoneMap {
		zones = append(zones, *zone)
	}

	// Sort by risk level and file count
	sort.Slice(zones, func(i, j int) bool {
		if zones[i].RiskLevel != zones[j].RiskLevel {
			return ra.getRiskLevelWeight(zones[i].RiskLevel) > ra.getRiskLevelWeight(zones[j].RiskLevel)
		}
		return zones[i].FileCount > zones[j].FileCount
	})

	return zones
}

// performBankingComplianceAnalysis performs banking-specific compliance analysis
func (ra *RepositoryAnalyzer) performBankingComplianceAnalysis(analyses []*AnalysisResult) *BankingComplianceInfo {
	compliance := &BankingComplianceInfo{
		DataModelFiles:         []string{},
		CustomerDataFiles:      []string{},
		PaymentProcessingFiles: []string{},
		AuthenticationFiles:    []string{},
		ComplianceFindings:     []ComplianceFinding{},
		RecommendedActions:     []string{},
	}

	var totalRisk float64
	fileCount := 0

	for _, analysis := range analyses {
		fileCount++

		// Calculate risk contribution
		switch analysis.RiskLevel {
		case RiskLevelCritical:
			totalRisk += 4.0
		case RiskLevelHigh:
			totalRisk += 3.0
		case RiskLevelMedium:
			totalRisk += 2.0
		case RiskLevelLow:
			totalRisk += 1.0
		}

		// Categorize files based on banking domain
		switch analysis.RiskZone {
		case "customer_data":
			compliance.CustomerDataFiles = append(compliance.CustomerDataFiles, analysis.FilePath)
		case "financial_data":
			compliance.DataModelFiles = append(compliance.DataModelFiles, analysis.FilePath)
			compliance.PaymentProcessingFiles = append(compliance.PaymentProcessingFiles, analysis.FilePath)
		case "payment_processing":
			compliance.PaymentProcessingFiles = append(compliance.PaymentProcessingFiles, analysis.FilePath)
		case "authentication":
			compliance.AuthenticationFiles = append(compliance.AuthenticationFiles, analysis.FilePath)
		}

		// Analyze security hints for compliance findings
		for _, hint := range analysis.SecurityHints {
			finding := ComplianceFinding{
				Type:        hint.Type,
				Severity:    hint.Severity,
				Description: hint.Description,
				FilePath:    analysis.FilePath,
				LineNumber:  hint.LineNumber,
				Regulation:  ra.determineRegulation(hint),
			}
			compliance.ComplianceFindings = append(compliance.ComplianceFindings, finding)
		}
	}

	// Calculate risk score (0-10 scale)
	if fileCount > 0 {
		compliance.RiskScore = (totalRisk / float64(fileCount)) * 2.5 // Scale to 0-10
	}

	// Generate recommendations
	compliance.RecommendedActions = ra.generateRecommendations(compliance)

	return compliance
}

// Helper functions

func (ra *RepositoryAnalyzer) readFile(filePath string) ([]byte, error) {
	// This would normally read the file from disk
	// For now, return empty content as placeholder
	return []byte{}, nil
}

func (ra *RepositoryAnalyzer) getZoneDescription(zone string) string {
	descriptions := map[string]string{
		"customer_data":      "Files handling customer personal information and sensitive data",
		"financial_data":     "Files processing financial transactions and monetary data",
		"payment_processing": "Files involved in payment processing and financial transactions",
		"authentication":     "Files handling user authentication and security",
		"business_logic":     "Core business logic and processing files",
		"utilities":          "Utility and helper functions",
		"tests":              "Test files and specifications",
		"general":            "General application files",
	}

	if desc, exists := descriptions[zone]; exists {
		return desc
	}
	return "General application files"
}

func (ra *RepositoryAnalyzer) getRiskLevelWeight(level RiskLevel) int {
	switch level {
	case RiskLevelCritical:
		return 4
	case RiskLevelHigh:
		return 3
	case RiskLevelMedium:
		return 2
	case RiskLevelLow:
		return 1
	default:
		return 0
	}
}

func (ra *RepositoryAnalyzer) determineRegulation(hint SecurityHint) string {
	description := strings.ToLower(hint.Description)

	if strings.Contains(description, "personal") || strings.Contains(description, "customer") {
		return "GDPR"
	}
	if strings.Contains(description, "payment") || strings.Contains(description, "financial") {
		return "PCI-DSS"
	}
	if strings.Contains(description, "data") {
		return "CCPA"
	}

	return "General"
}

func (ra *RepositoryAnalyzer) generateRecommendations(compliance *BankingComplianceInfo) []string {
	var recommendations []string

	if len(compliance.CustomerDataFiles) > 0 {
		recommendations = append(recommendations, "Implement data encryption for customer data files")
		recommendations = append(recommendations, "Add access logging for customer data access")
	}

	if len(compliance.PaymentProcessingFiles) > 0 {
		recommendations = append(recommendations, "Ensure PCI-DSS compliance for payment processing")
		recommendations = append(recommendations, "Implement transaction monitoring and anomaly detection")
	}

	if len(compliance.AuthenticationFiles) > 0 {
		recommendations = append(recommendations, "Review authentication mechanisms for security")
		recommendations = append(recommendations, "Implement multi-factor authentication where appropriate")
	}

	highSeverityFindings := 0
	for _, finding := range compliance.ComplianceFindings {
		if finding.Severity == "HIGH" || finding.Severity == "CRITICAL" {
			highSeverityFindings++
		}
	}

	if highSeverityFindings > 0 {
		recommendations = append(recommendations, fmt.Sprintf("Address %d high-severity security findings", highSeverityFindings))
	}

	if compliance.RiskScore > 7.0 {
		recommendations = append(recommendations, "Consider code review and security audit due to high risk score")
	}

	return recommendations
}
