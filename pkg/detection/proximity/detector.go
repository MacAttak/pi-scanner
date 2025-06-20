package proximity

import (
	"strings"

	"github.com/MacAttak/pi-scanner/pkg/detection"
)

// ProximityDetector analyzes the context around potential PI to improve detection accuracy
type ProximityDetector struct {
	patternMatcher *PatternMatcher
	analyzer       *ContextAnalyzer
}

// NewProximityDetector creates a new proximity detector
func NewProximityDetector() *ProximityDetector {
	return &ProximityDetector{
		patternMatcher: NewPatternMatcher(),
		analyzer:       NewContextAnalyzer(),
	}
}

// ProximityResult represents the result of proximity context analysis
type ProximityResult struct {
	Score      float64           `json:"score"`
	Reason     string            `json:"reason"`
	Context    PIContextType     `json:"context"`
	Keywords   []string          `json:"keywords"`
	Structure  StructureAnalysis `json:"structure"`
	Semantic   SemanticAnalysis  `json:"semantic"`
	IsTestData bool              `json:"is_test_data"`
}

// PIContextType represents different types of PI context
type PIContextType string

const (
	PIContextLabel         PIContextType = "label"         // "SSN:", "Tax File Number:"
	PIContextForm          PIContextType = "form"          // HTML forms, input fields
	PIContextDatabase      PIContextType = "database"      // SQL queries, database operations
	PIContextLog           PIContextType = "log"           // Log entries
	PIContextConfig        PIContextType = "config"        // Configuration files
	PIContextVariable      PIContextType = "variable"      // Variable assignments
	PIContextDocumentation PIContextType = "documentation" // Comments, docs
	PIContextProduction    PIContextType = "production"    // Regular production code
	PIContextTest          PIContextType = "test"          // Test/mock/sample data
)

// StructureType represents different content structure types
type StructureType string

const (
	StructureJSON      StructureType = "json"
	StructureXML       StructureType = "xml"
	StructureHTML      StructureType = "html"
	StructureSQL       StructureType = "sql"
	StructureYAML      StructureType = "yaml"
	StructureCode      StructureType = "code"
	StructurePlainText StructureType = "plain_text"
	StructureURL       StructureType = "url"
)

// StructureAnalysis contains information about content structure
type StructureAnalysis struct {
	Type         StructureType `json:"type"`
	NestingLevel int           `json:"nesting_level"`
	ElementType  string        `json:"element_type,omitempty"`
}

// SemanticAnalysis contains semantic context information
type SemanticAnalysis struct {
	Confidence float64  `json:"confidence"`
	Indicators []string `json:"indicators"`
	PITypes    []string `json:"pi_types"`
}

// PIContextInfo contains information about detected PI context
type PIContextInfo struct {
	Type     PIContextType `json:"type"`
	Keywords []string      `json:"keywords"`
	Distance int           `json:"distance"`
}

// AnalyzeContext performs comprehensive context analysis around a potential PI match
func (pd *ProximityDetector) AnalyzeContext(content, match string, startIndex, endIndex int) ProximityResult {
	result := ProximityResult{
		Score:      0.5, // Default neutral score
		Keywords:   []string{},
		IsTestData: false,
	}

	// Check if this is test data
	result.IsTestData = pd.IsTestData("", content, match, startIndex, endIndex)
	if result.IsTestData {
		result.Score = 0.1
		// Determine specific test data type
		contextBefore, contextAfter := pd.analyzer.ExtractSurroundingText(content, startIndex, endIndex, 30)
		fullContext := strings.ToLower(contextBefore + match + contextAfter)

		// Check for test data keywords in priority order
		// Check for more specific indicators first
		if strings.Contains(fullContext, "demo user") || strings.Contains(fullContext, "demo ") {
			result.Reason = "demo data indicator"
		} else if strings.Contains(fullContext, "fake") {
			result.Reason = "fake data indicator"
		} else if strings.Contains(fullContext, "dummy") {
			result.Reason = "dummy data indicator"
		} else if strings.Contains(fullContext, "mock") {
			result.Reason = "mock data indicator"
		} else if strings.Contains(fullContext, "sample") {
			result.Reason = "sample data indicator"
		} else if strings.Contains(fullContext, "example") {
			result.Reason = "example data indicator"
		} else if strings.Contains(fullContext, "test") {
			result.Reason = "test data indicator"
		} else {
			result.Reason = "test data indicator"
		}
		result.Context = PIContextTest
		return result
	}

	// Identify PI context (only if it's not test data)
	contextInfo := pd.IdentifyPIContext(content, match, startIndex, endIndex)
	result.Context = contextInfo.Type
	result.Keywords = contextInfo.Keywords

	// Calculate proximity score based on context
	result.Score = pd.CalculateProximityScore(contextInfo.Distance, contextInfo.Type)

	// Analyze structure
	result.Structure = pd.analyzer.AnalyzeStructure(content, startIndex, endIndex)

	// Perform semantic analysis
	result.Semantic = pd.analyzer.AnalyzeSemanticContext(content, startIndex, endIndex)

	// Combine scores and set reason
	result.Score = pd.combineScores(result.Score, result.Semantic.Confidence, result.Structure)
	result.Reason = pd.generateReason(result.Context, result.Keywords, result.IsTestData)

	return result
}

// IsTestData determines if the match appears to be test/mock/sample data
func (pd *ProximityDetector) IsTestData(filename, content, match string, startIndex, endIndex int) bool {
	// Extract context around the match
	contextBefore, contextAfter := pd.analyzer.ExtractSurroundingText(content, startIndex, endIndex, 30)
	fullContext := contextBefore + match + contextAfter

	// Check for test data keywords in the surrounding context
	return pd.patternMatcher.ContainsTestDataKeywords(fullContext)
}

// IdentifyPIContext identifies the type of context where PI appears
func (pd *ProximityDetector) IdentifyPIContext(content, match string, startIndex, endIndex int) PIContextInfo {
	// Extract larger context for analysis
	contextBefore, contextAfter := pd.analyzer.ExtractSurroundingText(content, startIndex, endIndex, 50)
	fullContext := contextBefore + match + contextAfter

	// Check for PI context labels first (highest priority)
	piLabels := pd.patternMatcher.FindPIContextLabels(fullContext)
	if len(piLabels) > 0 {
		distance := pd.calculateLabelDistance(contextBefore, piLabels)
		// Convert labels to lowercase for consistency
		lowerLabels := make([]string, len(piLabels))
		for i, label := range piLabels {
			lowerLabels[i] = strings.ToLower(label)
		}
		return PIContextInfo{
			Type:     PIContextLabel,
			Keywords: lowerLabels,
			Distance: distance,
		}
	}

	// Check for form field context
	if pd.patternMatcher.IsFormFieldContext(fullContext) {
		return PIContextInfo{
			Type:     PIContextForm,
			Keywords: []string{"form", "input"},
			Distance: 1,
		}
	}

	// Check for database context
	if pd.patternMatcher.IsDatabaseContext(fullContext) {
		return PIContextInfo{
			Type:     PIContextDatabase,
			Keywords: []string{"database", "query"},
			Distance: 1,
		}
	}

	// Check for log context
	if pd.patternMatcher.IsLogContext(fullContext) {
		return PIContextInfo{
			Type:     PIContextLog,
			Keywords: []string{"log"},
			Distance: 1,
		}
	}

	// Check for configuration context
	if pd.patternMatcher.IsConfigurationContext(fullContext) {
		return PIContextInfo{
			Type:     PIContextConfig,
			Keywords: []string{"config"},
			Distance: 1,
		}
	}

	// Check for variable context
	if pd.patternMatcher.IsVariableContext(fullContext) {
		return PIContextInfo{
			Type:     PIContextVariable,
			Keywords: []string{"variable"},
			Distance: 1,
		}
	}

	// Check for documentation context
	if pd.patternMatcher.IsDocumentationContext(fullContext) {
		return PIContextInfo{
			Type:     PIContextDocumentation,
			Keywords: []string{"documentation"},
			Distance: 1,
		}
	}

	// Default to production context
	return PIContextInfo{
		Type:     PIContextProduction,
		Keywords: []string{},
		Distance: 5,
	}
}

// CalculateProximityScore calculates a proximity score based on distance and context type
func (pd *ProximityDetector) CalculateProximityScore(distance int, contextType PIContextType) float64 {
	// Base scores for different context types
	baseScores := map[PIContextType]float64{
		PIContextLabel:         0.9,
		PIContextForm:          0.8,
		PIContextDatabase:      0.8,
		PIContextProduction:    0.8,
		PIContextLog:           0.7,
		PIContextConfig:        0.6,
		PIContextDocumentation: 0.4,
		PIContextVariable:      0.3,
		PIContextTest:          0.1,
	}

	baseScore := baseScores[contextType]
	if baseScore == 0 {
		baseScore = 0.5 // Default score
	}

	// Adjust score based on distance for label context
	if contextType == PIContextLabel && distance > 0 {
		// Closer labels get higher scores
		distanceModifier := 1.0 / float64(distance)
		baseScore = baseScore * (0.7 + 0.3*distanceModifier)
	}

	// Ensure score is within bounds
	if baseScore < 0.0 {
		baseScore = 0.0
	}
	if baseScore > 1.0 {
		baseScore = 1.0
	}

	return baseScore
}

// calculateLabelDistance calculates the distance from the match to the nearest PI label
func (pd *ProximityDetector) calculateLabelDistance(contextBefore string, labels []string) int {
	words := strings.Fields(contextBefore)
	if len(words) == 0 {
		return 1
	}

	// Find the closest label
	minDistance := len(words)
	for _, label := range labels {
		labelWords := strings.Fields(label)
		if len(labelWords) == 0 {
			continue
		}

		// Look for the label in the words
		for i := len(words) - len(labelWords); i >= 0; i-- {
			match := true
			for j, labelWord := range labelWords {
				if i+j >= len(words) || !strings.EqualFold(words[i+j], labelWord) {
					match = false
					break
				}
			}
			if match {
				distance := len(words) - i - len(labelWords) + 1
				if distance < minDistance {
					minDistance = distance
				}
				break
			}
		}
	}

	if minDistance == len(words) {
		return 5 // Default distance if not found
	}

	return minDistance
}

// combineScores combines multiple scores using weighted average
func (pd *ProximityDetector) combineScores(proximityScore, semanticScore float64, structure StructureAnalysis) float64 {
	// Weights for different score components
	proximityWeight := 0.6
	semanticWeight := 0.3
	structureWeight := 0.1

	// Structure bonus/penalty
	structureModifier := 1.0
	switch structure.Type {
	case StructureJSON, StructureXML, StructureHTML:
		structureModifier = 1.1 // Structured data is more likely to be real
	case StructureSQL:
		structureModifier = 1.2 // Database queries are very likely to be real
	case StructureCode:
		structureModifier = 0.9 // Code context might be variable names
	}

	combinedScore := (proximityScore*proximityWeight +
		semanticScore*semanticWeight +
		proximityScore*structureWeight) * structureModifier

	// Ensure score is within bounds
	if combinedScore < 0.0 {
		combinedScore = 0.0
	}
	if combinedScore > 1.0 {
		combinedScore = 1.0
	}

	return combinedScore
}

// generateReason generates a human-readable reason for the score
func (pd *ProximityDetector) generateReason(contextType PIContextType, keywords []string, isTestData bool) string {
	if isTestData {
		// Return specific test data reason based on keywords found
		testKeywords := []string{"test", "example", "mock", "sample", "demo", "fake", "dummy"}

		// First check for exact matches
		for _, testKeyword := range testKeywords {
			for _, keyword := range keywords {
				if strings.ToLower(keyword) == testKeyword {
					return testKeyword + " data indicator"
				}
			}
		}

		// Then check for partial matches
		for _, testKeyword := range testKeywords {
			for _, keyword := range keywords {
				if strings.Contains(strings.ToLower(keyword), testKeyword) {
					return testKeyword + " data indicator"
				}
			}
		}

		if len(keywords) > 0 {
			return strings.ToLower(keywords[0]) + " data indicator"
		}
		return "test data indicator"
	}

	switch contextType {
	case PIContextLabel:
		return "PI context label detected"
	case PIContextForm:
		return "form field context"
	case PIContextDatabase:
		return "database query context"
	case PIContextLog:
		return "log entry context"
	case PIContextConfig:
		return "configuration context"
	case PIContextVariable:
		return "variable assignment context"
	case PIContextDocumentation:
		return "documentation context"
	case PIContextProduction:
		return "production code context"
	default:
		return "general context"
	}
}

// EnhanceFinding enhances a detection finding with proximity analysis
func (pd *ProximityDetector) EnhanceFinding(finding *detection.Finding, content string) {
	if finding == nil {
		return
	}

	// Calculate the actual start/end indices in content
	// This is a simplified approach - in reality, you'd need to map line/column to indices
	startIndex := 0
	endIndex := len(finding.Match)

	// Find the match in content
	if idx := strings.Index(content, finding.Match); idx != -1 {
		startIndex = idx
		endIndex = idx + len(finding.Match)
	}

	// Perform proximity analysis
	result := pd.AnalyzeContext(content, finding.Match, startIndex, endIndex)

	// Update finding with proximity information
	finding.ContextModifier = float32(result.Score)
	finding.Confidence = float32(result.Score) * finding.Confidence

	// Enhance context information
	if result.Context == PIContextTest || result.IsTestData {
		finding.RiskLevel = detection.RiskLevelLow
		finding.ContextModifier = 0.1
	}

	// Update context fields with richer information
	contextBefore, contextAfter := pd.analyzer.ExtractSurroundingText(content, startIndex, endIndex, 25)
	finding.ContextBefore = contextBefore
	finding.ContextAfter = contextAfter
}

// AnalyzeFile performs proximity analysis on an entire file's findings
func (pd *ProximityDetector) AnalyzeFile(filename string, content string, findings []detection.Finding) []detection.Finding {
	enhanced := make([]detection.Finding, len(findings))
	copy(enhanced, findings)

	for i := range enhanced {
		pd.EnhanceFinding(&enhanced[i], content)
	}

	return enhanced
}
