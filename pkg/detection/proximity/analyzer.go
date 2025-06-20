package proximity

import (
	"regexp"
	"strings"
)

// ContextAnalyzer provides utilities for analyzing the context around potential PI matches
type ContextAnalyzer struct {
	wordPattern       *regexp.Regexp
	structurePatterns map[StructureType]*regexp.Regexp
}

// NewContextAnalyzer creates a new context analyzer
func NewContextAnalyzer() *ContextAnalyzer {
	ca := &ContextAnalyzer{
		structurePatterns: make(map[StructureType]*regexp.Regexp),
	}
	ca.compilePatterns()
	return ca
}

// compilePatterns compiles regex patterns for structure detection
func (ca *ContextAnalyzer) compilePatterns() {
	// Word boundary pattern for keyword matching
	ca.wordPattern = regexp.MustCompile(`\b\w+\b`)

	// Structure detection patterns
	ca.structurePatterns[StructureJSON] = regexp.MustCompile(`(?s)[{,]\s*"[^"]*"\s*:\s*"[^"]*"`)
	ca.structurePatterns[StructureXML] = regexp.MustCompile(`(?s)<[^>]+>[^<]*</[^>]+>|<[^>]+/>`)
	ca.structurePatterns[StructureHTML] = regexp.MustCompile(`(?i)(?s)<(input|textarea|select|form)[^>]*>`)
	ca.structurePatterns[StructureSQL] = regexp.MustCompile(`(?i)\b(SELECT|INSERT|UPDATE|DELETE|FROM|WHERE|SET)\b`)
	ca.structurePatterns[StructureYAML] = regexp.MustCompile(`(?m)^\s*\w+\s*:\s*`)
	ca.structurePatterns[StructureURL] = regexp.MustCompile(`(?i)(https?://[^\s<>]+|ftp://[^\s<>]+)`)
	ca.structurePatterns[StructureCode] = regexp.MustCompile(`(?i)\b(var|let|const|function|class|if|for|while|return)\b`)
}

// ExtractSurroundingText extracts text before and after a match within a specified window
func (ca *ContextAnalyzer) ExtractSurroundingText(content string, startIndex, endIndex, windowSize int) (before, after string) {
	if len(content) == 0 || startIndex < 0 || endIndex < 0 {
		return "", ""
	}

	// Swap indices if they're reversed
	if startIndex > endIndex {
		startIndex, endIndex = endIndex, startIndex
	}

	// Ensure indices are valid
	if startIndex > len(content) {
		startIndex = len(content)
	}
	if endIndex > len(content) {
		endIndex = len(content)
	}

	// Calculate before context
	beforeStart := startIndex - windowSize
	if beforeStart < 0 {
		beforeStart = 0
	}
	before = content[beforeStart:startIndex]

	// Calculate after context
	afterEnd := endIndex + windowSize
	if afterEnd > len(content) {
		afterEnd = len(content)
	}
	after = content[endIndex:afterEnd]

	return before, after
}

// GetWordProximity calculates the word distance between a target word and a match position
func (ca *ContextAnalyzer) GetWordProximity(content, targetWord string, startIndex, endIndex int) (distance int, found bool) {
	if len(content) == 0 || len(targetWord) == 0 {
		return -1, false
	}

	// Swap indices if they're reversed
	if startIndex > endIndex {
		startIndex, endIndex = endIndex, startIndex
	}

	// Ensure indices are valid
	if startIndex < 0 {
		startIndex = 0
	}
	if endIndex > len(content) {
		endIndex = len(content)
	}

	// Extract surrounding context (larger window for proximity analysis)
	before, after := ca.ExtractSurroundingText(content, startIndex, endIndex, 100)

	// Find target word in before context
	beforeWords := ca.extractWords(before)
	afterWords := ca.extractWords(after)

	targetLower := strings.ToLower(targetWord)

	// Check before words (reverse order for distance calculation)
	for i := len(beforeWords) - 1; i >= 0; i-- {
		if strings.ToLower(beforeWords[i]) == targetLower {
			return len(beforeWords) - i, true
		}
	}

	// Check after words
	for i, word := range afterWords {
		if strings.ToLower(word) == targetLower {
			return i + 1, true
		}
	}

	return -1, false
}

// CountKeywords counts occurrences of keywords in the text as whole words
func (ca *ContextAnalyzer) CountKeywords(text string, keywords []string) int {
	if len(text) == 0 || len(keywords) == 0 {
		return 0
	}

	textLower := strings.ToLower(text)
	words := ca.extractWords(textLower)
	wordSet := make(map[string]bool)
	for _, word := range words {
		wordSet[word] = true
	}

	count := 0
	for _, keyword := range keywords {
		keywordLower := strings.ToLower(keyword)
		if wordSet[keywordLower] {
			// Count all occurrences of this keyword
			for _, word := range words {
				if word == keywordLower {
					count++
				}
			}
		}
	}

	return count
}

// AnalyzeStructure analyzes the structure type of content around a match
func (ca *ContextAnalyzer) AnalyzeStructure(content string, startIndex, endIndex int) StructureAnalysis {
	if len(content) == 0 {
		return StructureAnalysis{Type: StructurePlainText, NestingLevel: 0}
	}

	// Swap indices if they're reversed
	if startIndex > endIndex {
		startIndex, endIndex = endIndex, startIndex
	}

	// Ensure indices are valid
	if startIndex < 0 {
		startIndex = 0
	}
	if endIndex > len(content) {
		endIndex = len(content)
	}

	// Extract larger context for structure analysis
	before, after := ca.ExtractSurroundingText(content, startIndex, endIndex, 200)

	// Safely extract the match content
	var matchContent string
	if startIndex < len(content) && endIndex <= len(content) && startIndex <= endIndex {
		matchContent = content[startIndex:endIndex]
	}
	context := before + matchContent + after

	// Check structure types in order of priority (most specific first)
	structureOrder := []StructureType{
		StructureURL,  // Check URL first as it should override others
		StructureHTML, // Check HTML before XML to avoid conflicts
		StructureXML,
		StructureJSON,
		StructureSQL,
		StructureCode,
		StructureYAML, // Check YAML last as it can be too general
	}

	for _, structType := range structureOrder {
		if pattern, exists := ca.structurePatterns[structType]; exists && pattern.MatchString(context) {
			nestingLevel := ca.calculateNestingLevel(context, structType)
			elementType := ca.identifyElementType(context, structType)
			return StructureAnalysis{
				Type:         structType,
				NestingLevel: nestingLevel,
				ElementType:  elementType,
			}
		}
	}

	return StructureAnalysis{Type: StructurePlainText, NestingLevel: 0}
}

// CalculateContextWindow calculates the optimal context window size based on content
func (ca *ContextAnalyzer) CalculateContextWindow(content string, startIndex, endIndex, baseWindow int) (beforeWindow, afterWindow int) {
	if len(content) == 0 {
		return 0, 0
	}

	// Ensure indices are valid
	if startIndex < 0 {
		startIndex = 0
	}
	if endIndex > len(content) {
		endIndex = len(content)
	}
	if startIndex > len(content) {
		startIndex = len(content)
	}

	// Calculate available space and return it
	// The baseWindow parameter is used for guidance but doesn't limit the actual window
	beforeWindow = startIndex
	afterWindow = len(content) - endIndex

	return beforeWindow, afterWindow
}

// FindNearestKeyword finds the nearest keyword to a match position
func (ca *ContextAnalyzer) FindNearestKeyword(content string, keywords []string, startIndex, endIndex int) (keyword string, distance int, found bool) {
	if len(content) == 0 || len(keywords) == 0 {
		return "", -1, false
	}

	// Swap indices if they're reversed
	if startIndex > endIndex {
		startIndex, endIndex = endIndex, startIndex
	}

	// Ensure indices are valid
	if startIndex < 0 {
		startIndex = 0
	}
	if endIndex > len(content) {
		endIndex = len(content)
	}

	minDistance := -1
	nearestKeyword := ""

	for _, kw := range keywords {
		dist, f := ca.GetWordProximity(content, kw, startIndex, endIndex)
		if f && (minDistance == -1 || dist < minDistance) {
			minDistance = dist
			nearestKeyword = kw
		}
	}

	if minDistance != -1 {
		return nearestKeyword, minDistance, true
	}

	return "", -1, false
}

// AnalyzeSemanticContext performs semantic analysis of the context
func (ca *ContextAnalyzer) AnalyzeSemanticContext(content string, startIndex, endIndex int) SemanticAnalysis {
	if len(content) == 0 {
		return SemanticAnalysis{Confidence: 0.5, Indicators: []string{}, PITypes: []string{}}
	}

	// Swap indices if they're reversed
	if startIndex > endIndex {
		startIndex, endIndex = endIndex, startIndex
	}

	// Ensure indices are valid
	if startIndex < 0 {
		startIndex = 0
	}
	if endIndex > len(content) {
		endIndex = len(content)
	}

	// Extract context for analysis
	before, after := ca.ExtractSurroundingText(content, startIndex, endIndex, 100)
	context := before + after

	indicators := []string{}
	confidence := 0.5 // Start with neutral confidence

	// Analyze for different semantic indicators
	lowerContext := strings.ToLower(context)

	// PI indicators (positive)
	piIndicators := []string{"customer", "user", "client", "person", "individual", "verification", "authentication", "secure", "private", "confidential"}
	for _, indicator := range piIndicators {
		if strings.Contains(lowerContext, indicator) {
			confidence += 0.1
			indicators = append(indicators, indicator)
		}
	}

	// Check for specific PI type labels
	if strings.Contains(lowerContext, "ssn") {
		indicators = append(indicators, "ssn")
		confidence += 0.2
	}

	// Test indicators (negative)
	testIndicators := []string{"test", "mock", "sample", "demo", "example", "fake", "dummy"}
	for _, indicator := range testIndicators {
		if strings.Contains(lowerContext, indicator) {
			confidence -= 0.2
			indicators = append(indicators, indicator)
		}
	}

	// Database indicators (positive)
	dbIndicators := []string{"database", "table", "query", "select", "insert", "update", "where"}
	foundDB := false
	for _, indicator := range dbIndicators {
		if strings.Contains(lowerContext, indicator) && !foundDB {
			confidence += 0.05
			indicators = append(indicators, "database", "query")
			foundDB = true
		}
	}

	// Form indicators (positive)
	formIndicators := []string{"form", "input", "field", "submit", "validation"}
	foundForm := false
	for _, indicator := range formIndicators {
		if strings.Contains(lowerContext, indicator) && !foundForm {
			confidence += 0.05
			indicators = append(indicators, "form", "input")
			foundForm = true
		}
	}

	// Documentation indicators (negative)
	docIndicators := []string{"documentation", "comment", "example", "reference", "guide"}
	for _, indicator := range docIndicators {
		if strings.Contains(lowerContext, indicator) {
			confidence -= 0.1
			indicators = append(indicators, "documentation")
			break
		}
	}

	// Log indicators (moderate positive)
	logIndicators := []string{"log", "info", "debug", "error", "warn", "trace"}
	foundLog := false
	for _, indicator := range logIndicators {
		if strings.Contains(lowerContext, indicator) && !foundLog {
			confidence += 0.03
			indicators = append(indicators, "log")
			foundLog = true
		}
	}

	// Detect likely PI types based on context
	piTypes := ca.detectPITypes(context)

	// Ensure confidence is within bounds
	if confidence < 0.0 {
		confidence = 0.0
	}
	if confidence > 1.0 {
		confidence = 1.0
	}

	return SemanticAnalysis{
		Confidence: confidence,
		Indicators: ca.removeDuplicates(indicators),
		PITypes:    piTypes,
	}
}

// Helper functions

// extractWords extracts words from text, handling various separators
func (ca *ContextAnalyzer) extractWords(text string) []string {
	if ca.wordPattern == nil {
		// Fallback if pattern not compiled
		return strings.Fields(text)
	}

	matches := ca.wordPattern.FindAllString(text, -1)
	words := make([]string, 0, len(matches))
	for _, match := range matches {
		if len(match) > 0 {
			words = append(words, strings.ToLower(match))
		}
	}
	return words
}

// calculateNestingLevel calculates the nesting level for structured content
func (ca *ContextAnalyzer) calculateNestingLevel(content string, structType StructureType) int {
	switch structType {
	case StructureJSON:
		return ca.countCharacter(content, '{') + ca.countCharacter(content, '[')
	case StructureXML, StructureHTML:
		return strings.Count(content, "<") / 2 // Approximate nesting based on tag count
	case StructureYAML:
		lines := strings.Split(content, "\n")
		maxIndent := 0
		for _, line := range lines {
			indent := 0
			for _, r := range line {
				if r == ' ' || r == '\t' {
					indent++
				} else {
					break
				}
			}
			if indent > maxIndent {
				maxIndent = indent
			}
		}
		return maxIndent / 2 // Assuming 2 spaces per level
	default:
		return 0
	}
}

// identifyElementType identifies the specific element type within a structure
func (ca *ContextAnalyzer) identifyElementType(content string, structType StructureType) string {
	switch structType {
	case StructureHTML:
		if strings.Contains(strings.ToLower(content), "input") {
			return "input"
		} else if strings.Contains(strings.ToLower(content), "textarea") {
			return "textarea"
		} else if strings.Contains(strings.ToLower(content), "select") {
			return "select"
		}
		return "form"
	case StructureSQL:
		contentLower := strings.ToLower(content)
		if strings.Contains(contentLower, "select") {
			return "select"
		} else if strings.Contains(contentLower, "insert") {
			return "insert"
		} else if strings.Contains(contentLower, "update") {
			return "update"
		} else if strings.Contains(contentLower, "delete") {
			return "delete"
		}
		return "query"
	default:
		return ""
	}
}

// countCharacter counts occurrences of a character in text
func (ca *ContextAnalyzer) countCharacter(text string, char rune) int {
	count := 0
	for _, r := range text {
		if r == char {
			count++
		}
	}
	return count
}

// detectPITypes detects likely PI types based on context keywords
func (ca *ContextAnalyzer) detectPITypes(context string) []string {
	contextLower := strings.ToLower(context)
	piTypes := []string{}

	// SSN indicators
	if strings.Contains(contextLower, "ssn") || strings.Contains(contextLower, "social security") {
		piTypes = append(piTypes, "SSN")
	}

	// TFN indicators
	if strings.Contains(contextLower, "tfn") || strings.Contains(contextLower, "tax file") {
		piTypes = append(piTypes, "TFN")
	}

	// Medicare indicators
	if strings.Contains(contextLower, "medicare") {
		piTypes = append(piTypes, "Medicare")
	}

	// Email indicators
	if strings.Contains(contextLower, "email") || strings.Contains(contextLower, "@") {
		piTypes = append(piTypes, "Email")
	}

	// Phone indicators
	if strings.Contains(contextLower, "phone") || strings.Contains(contextLower, "mobile") || strings.Contains(contextLower, "tel") {
		piTypes = append(piTypes, "Phone")
	}

	// Credit card indicators
	if strings.Contains(contextLower, "credit") || strings.Contains(contextLower, "card") || strings.Contains(contextLower, "cc") {
		piTypes = append(piTypes, "Credit Card")
	}

	return piTypes
}

// removeDuplicates removes duplicate strings from a slice
func (ca *ContextAnalyzer) removeDuplicates(items []string) []string {
	seen := make(map[string]bool)
	result := []string{}

	for _, item := range items {
		if !seen[item] {
			seen[item] = true
			result = append(result, item)
		}
	}

	return result
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// max returns the maximum of two integers
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
