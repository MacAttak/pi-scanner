package ast

import (
	"context"
	"regexp"
	"strings"
	"time"
)

// analyzeJava performs Java-specific AST analysis
func (a *Analyzer) analyzeJava(ctx context.Context, filePath string, content []byte, result *AnalysisResult) (*AnalysisResult, error) {
	startTime := time.Now()
	defer func() {
		result.AnalysisTime = time.Since(startTime).Milliseconds()
	}()

	contentStr := string(content)

	// Initialize result structure
	result.CodeStructure = &CodeStructure{
		Classes:     []ClassInfo{},
		Methods:     []MethodInfo{},
		Variables:   []VariableInfo{},
		Annotations: []Annotation{},
		Imports:     []string{},
	}
	result.SecurityHints = []SecurityHint{}
	result.Dependencies = []Dependency{}

	// Parse package and imports
	a.parseJavaImports(contentStr, result)

	// Parse classes
	a.parseJavaClasses(contentStr, result)

	// Parse methods
	a.parseJavaMethods(contentStr, result)

	// Parse variables and fields
	a.parseJavaVariables(contentStr, result)

	// Analyze security hints
	a.analyzeJavaSecurityHints(contentStr, result)

	// Parse dependencies
	a.parseJavaDependencies(contentStr, result)

	return result, nil
}

// parseJavaImports extracts import statements
func (a *Analyzer) parseJavaImports(content string, result *AnalysisResult) {
	importRegex := regexp.MustCompile(`(?m)^import\s+(?:static\s+)?([a-zA-Z_][a-zA-Z0-9_]*(?:\.[a-zA-Z_][a-zA-Z0-9_]*)*(?:\.\*)?)\s*;`)
	matches := importRegex.FindAllStringSubmatch(content, -1)

	for _, match := range matches {
		if len(match) > 1 {
			result.CodeStructure.Imports = append(result.CodeStructure.Imports, match[1])

			// Create dependency
			result.Dependencies = append(result.Dependencies, Dependency{
				Name:   match[1],
				Type:   "import",
				Target: match[1],
			})
		}
	}
}

// parseJavaClasses extracts class definitions
func (a *Analyzer) parseJavaClasses(content string, result *AnalysisResult) {
	// Class definition regex - supports various modifiers
	classRegex := regexp.MustCompile(`(?m)^\s*(?:@[A-Za-z][A-Za-z0-9_]*(?:\([^)]*\))?\s*)*(?:(public|private|protected|abstract|final|static)\s+)*(?:(class|interface|enum)\s+)([A-Za-z_][A-Za-z0-9_]*)\s*(?:extends\s+([A-Za-z_][A-Za-z0-9_]*(?:\.[A-Za-z_][A-Za-z0-9_]*)*))?\s*(?:implements\s+([A-Za-z_][A-Za-z0-9_]*(?:\.[A-Za-z_][A-Za-z0-9_]*)*(?:\s*,\s*[A-Za-z_][A-Za-z0-9_]*(?:\.[A-Za-z_][A-Za-z0-9_]*)*)*))?\s*\{`)

	matches := classRegex.FindAllStringSubmatch(content, -1)

	for _, match := range matches {
		if len(match) > 3 {
			className := match[3]
			isPublic := strings.Contains(match[1], "public")
			extends := ""
			if len(match) > 4 && match[4] != "" {
				extends = match[4]
			}

			var implements []string
			if len(match) > 5 && match[5] != "" {
				implements = strings.Split(strings.ReplaceAll(match[5], " ", ""), ",")
			}

			// Find line number
			lineNum := a.findLineNumber(content, match[0])

			// Extract annotations for this class
			annotations := a.extractJavaAnnotations(content, lineNum)

			classInfo := ClassInfo{
				Name:        className,
				Package:     a.extractPackage(content),
				LineNumber:  lineNum,
				IsPublic:    isPublic,
				Extends:     extends,
				Implements:  implements,
				Annotations: annotations,
			}

			result.CodeStructure.Classes = append(result.CodeStructure.Classes, classInfo)
		}
	}
}

// parseJavaMethods extracts method definitions
func (a *Analyzer) parseJavaMethods(content string, result *AnalysisResult) {
	// Method definition regex
	methodRegex := regexp.MustCompile(`(?m)^\s*(?:@[A-Za-z][A-Za-z0-9_]*(?:\([^)]*\))?\s*)*(?:(public|private|protected|static|final|abstract|synchronized)\s+)*([A-Za-z_][A-Za-z0-9_<>,\[\]\s]*)\s+([A-Za-z_][A-Za-z0-9_]*)\s*\(([^)]*)\)\s*(?:throws\s+[A-Za-z_][A-Za-z0-9_,\s]*)?(?:\{|;)`)

	matches := methodRegex.FindAllStringSubmatch(content, -1)

	for _, match := range matches {
		if len(match) > 3 {
			modifiers := match[1]
			returnType := strings.TrimSpace(match[2])
			methodName := match[3]
			parameters := a.parseJavaParameters(match[4])

			lineNum := a.findLineNumber(content, match[0])
			annotations := a.extractJavaAnnotations(content, lineNum)

			methodInfo := MethodInfo{
				Name:        methodName,
				LineNumber:  lineNum,
				IsPublic:    strings.Contains(modifiers, "public"),
				Parameters:  parameters,
				ReturnType:  returnType,
				Annotations: annotations,
			}

			result.CodeStructure.Methods = append(result.CodeStructure.Methods, methodInfo)
		}
	}
}

// parseJavaVariables extracts field and variable definitions
func (a *Analyzer) parseJavaVariables(content string, result *AnalysisResult) {
	// Field definition regex
	fieldRegex := regexp.MustCompile(`(?m)^\s*(?:@[A-Za-z][A-Za-z0-9_]*(?:\([^)]*\))?\s*)*(?:(public|private|protected|static|final|volatile|transient)\s+)*([A-Za-z_][A-Za-z0-9_<>,\[\]\s]+)\s+([A-Za-z_][A-Za-z0-9_]*)\s*(?:=\s*[^;]+)?\s*;`)

	matches := fieldRegex.FindAllStringSubmatch(content, -1)

	for _, match := range matches {
		if len(match) > 3 {
			modifiers := match[1]
			varType := strings.TrimSpace(match[2])
			varName := match[3]

			lineNum := a.findLineNumber(content, match[0])

			variableInfo := VariableInfo{
				Name:       varName,
				Type:       varType,
				LineNumber: lineNum,
				IsConstant: strings.Contains(modifiers, "final"),
				Scope:      a.determineScope(modifiers),
			}

			result.CodeStructure.Variables = append(result.CodeStructure.Variables, variableInfo)
		}
	}
}

// analyzeJavaSecurityHints identifies potential security issues
func (a *Analyzer) analyzeJavaSecurityHints(content string, result *AnalysisResult) {
	securityPatterns := map[string]string{
		`(?i)password\s*=\s*["'][^"']+["']`:                        "Hardcoded password detected",
		`(?i)secret\s*=\s*["'][^"']+["']`:                          "Hardcoded secret detected",
		`(?i)api[_-]?[Kk]ey\s*=\s*["'][^"']+["']`:                  "Hardcoded API key detected",
		`(?i)token\s*=\s*["'][^"']+["']`:                           "Hardcoded token detected",
		`\.execute\s*\(\s*["'][^"']*\+`:                            "Potential SQL injection risk",
		`Runtime\.getRuntime\(\)\.exec\(`:                          "Command execution detected",
		`@Entity\s*(?:\([^)]*\))?\s*public\s+class`:                "JPA Entity - potential data exposure",
		`@RequestMapping\s*\([^)]*value\s*=\s*["'][^"']*\{[^}]*\}`: "Dynamic URL mapping - potential security risk",
	}

	for pattern, description := range securityPatterns {
		regex := regexp.MustCompile(pattern)
		matches := regex.FindAllString(content, -1)

		for _, match := range matches {
			lineNum := a.findLineNumber(content, match)
			severity := "MEDIUM"

			// Adjust severity based on pattern type
			if strings.Contains(strings.ToLower(description), "hardcoded") {
				severity = "HIGH"
			}
			if strings.Contains(strings.ToLower(description), "injection") {
				severity = "HIGH"
			}

			hint := SecurityHint{
				Type:        "security_risk",
				Description: description,
				LineNumber:  lineNum,
				Severity:    severity,
			}

			result.SecurityHints = append(result.SecurityHints, hint)
		}
	}
}

// parseJavaDependencies analyzes code dependencies
func (a *Analyzer) parseJavaDependencies(content string, result *AnalysisResult) {
	// Analyze inheritance relationships
	for _, class := range result.CodeStructure.Classes {
		if class.Extends != "" {
			dep := Dependency{
				Name:   class.Name,
				Type:   "inheritance",
				Target: class.Extends,
			}
			result.Dependencies = append(result.Dependencies, dep)
		}

		for _, impl := range class.Implements {
			dep := Dependency{
				Name:   class.Name,
				Type:   "implementation",
				Target: impl,
			}
			result.Dependencies = append(result.Dependencies, dep)
		}
	}
}

// Helper functions

func (a *Analyzer) parseJavaParameters(paramStr string) []string {
	if strings.TrimSpace(paramStr) == "" {
		return []string{}
	}

	params := strings.Split(paramStr, ",")
	var result []string

	for _, param := range params {
		param = strings.TrimSpace(param)
		if param != "" {
			// Extract parameter type and name
			parts := strings.Fields(param)
			if len(parts) >= 2 {
				result = append(result, parts[len(parts)-1]) // Parameter name is usually last
			}
		}
	}

	return result
}

func (a *Analyzer) extractJavaAnnotations(content string, lineNum int) []string {
	lines := strings.Split(content, "\n")
	var annotations []string

	// Look backwards from the line to find annotations
	for i := lineNum - 2; i >= 0 && i < len(lines); i-- {
		line := strings.TrimSpace(lines[i])
		if strings.HasPrefix(line, "@") {
			annotations = append([]string{line}, annotations...) // Prepend to maintain order
		} else if line != "" && !strings.HasPrefix(line, "//") && !strings.HasPrefix(line, "/*") {
			break // Stop at non-annotation, non-comment line
		}
	}

	return annotations
}

func (a *Analyzer) extractPackage(content string) string {
	packageRegex := regexp.MustCompile(`(?m)^package\s+([a-zA-Z_][a-zA-Z0-9_]*(?:\.[a-zA-Z_][a-zA-Z0-9_]*)*)\s*;`)
	match := packageRegex.FindStringSubmatch(content)

	if len(match) > 1 {
		return match[1]
	}

	return ""
}

func (a *Analyzer) determineScope(modifiers string) string {
	if strings.Contains(modifiers, "public") {
		return "public"
	}
	if strings.Contains(modifiers, "private") {
		return "private"
	}
	if strings.Contains(modifiers, "protected") {
		return "protected"
	}
	return "package"
}

func (a *Analyzer) findLineNumber(content, target string) int {
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		if strings.Contains(line, target) {
			return i + 1
		}
	}
	return 1
}
