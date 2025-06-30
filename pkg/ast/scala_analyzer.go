package ast

import (
	"context"
	"regexp"
	"strings"
	"time"
)

// analyzeScala performs Scala-specific AST analysis
func (a *Analyzer) analyzeScala(ctx context.Context, filePath string, content []byte, result *AnalysisResult) (*AnalysisResult, error) {
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
	a.parseScalaImports(contentStr, result)

	// Parse classes and objects
	a.parseScalaClasses(contentStr, result)

	// Parse methods and functions
	a.parseScalaMethods(contentStr, result)

	// Parse variables and fields
	a.parseScalaVariables(contentStr, result)

	// Analyze security hints
	a.analyzeScalaSecurityHints(contentStr, result)

	// Parse dependencies
	a.parseScalaDependencies(contentStr, result)

	return result, nil
}

// parseScalaImports extracts import statements
func (a *Analyzer) parseScalaImports(content string, result *AnalysisResult) {
	// Scala import patterns
	importRegex := regexp.MustCompile(`(?m)^import\s+([a-zA-Z_][a-zA-Z0-9_]*(?:\.[a-zA-Z_][a-zA-Z0-9_]*)*(?:\.\{[^}]+\}|\.\*|\.\_)?)\s*$`)
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

// parseScalaClasses extracts class, object, and trait definitions
func (a *Analyzer) parseScalaClasses(content string, result *AnalysisResult) {
	// Scala class/object/trait definition regex
	classRegex := regexp.MustCompile(`(?m)^\s*(?:@[A-Za-z][A-Za-z0-9_]*(?:\([^)]*\))?\s*)*(?:(abstract|final|sealed|implicit)\s+)*(?:(class|object|trait|case\s+class|case\s+object)\s+)([A-Za-z_][A-Za-z0-9_]*)\s*(?:\[([^\]]+)\])?\s*(?:\(([^)]*)\))?\s*(?:extends\s+([A-Za-z_][A-Za-z0-9_]*(?:\.[A-Za-z_][A-Za-z0-9_]*)*))?\s*(?:with\s+([A-Za-z_][A-Za-z0-9_]*(?:\.[A-Za-z_][A-Za-z0-9_]*)*(?:\s+with\s+[A-Za-z_][A-Za-z0-9_]*(?:\.[A-Za-z_][A-Za-z0-9_]*)*)*))?\s*\{?`)

	matches := classRegex.FindAllStringSubmatch(content, -1)

	for _, match := range matches {
		if len(match) > 3 {
			modifiers := match[1]
			className := match[3]
			extends := ""
			if len(match) > 6 && match[6] != "" {
				extends = match[6]
			}

			var implements []string
			if len(match) > 7 && match[7] != "" {
				// Parse "with" clauses
				withClause := match[7]
				implements = strings.Split(strings.ReplaceAll(withClause, "with", ","), ",")
				for i, impl := range implements {
					implements[i] = strings.TrimSpace(impl)
				}
			}

			// Find line number
			lineNum := a.findLineNumber(content, match[0])

			// Extract annotations for this class
			annotations := a.extractScalaAnnotations(content, lineNum)

			// Determine if public (Scala default is public)
			isPublic := !strings.Contains(modifiers, "private") && !strings.Contains(modifiers, "protected")

			classInfo := ClassInfo{
				Name:        className,
				Package:     a.extractScalaPackage(content),
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

// parseScalaMethods extracts method and function definitions
func (a *Analyzer) parseScalaMethods(content string, result *AnalysisResult) {
	// Scala method definition regex - covers def, val, var functions
	methodRegex := regexp.MustCompile(`(?m)^\s*(?:@[A-Za-z][A-Za-z0-9_]*(?:\([^)]*\))?\s*)*(?:(override|private|protected|implicit|final)\s+)*(def|val|var)\s+([A-Za-z_][A-Za-z0-9_]*)\s*(?:\[([^\]]+)\])?\s*(?:\(([^)]*)\))?\s*(?::\s*([A-Za-z_][A-Za-z0-9_\[\],\s]*))?\s*=`)

	matches := methodRegex.FindAllStringSubmatch(content, -1)

	for _, match := range matches {
		if len(match) > 3 {
			modifiers := match[1]
			methodName := match[3]

			var parameters []string
			if len(match) > 5 && match[5] != "" {
				parameters = a.parseScalaParameters(match[5])
			}

			returnType := ""
			if len(match) > 6 && match[6] != "" {
				returnType = strings.TrimSpace(match[6])
			}

			lineNum := a.findLineNumber(content, match[0])
			annotations := a.extractScalaAnnotations(content, lineNum)

			// Determine if public (Scala default is public)
			isPublic := !strings.Contains(modifiers, "private") && !strings.Contains(modifiers, "protected")

			methodInfo := MethodInfo{
				Name:        methodName,
				LineNumber:  lineNum,
				IsPublic:    isPublic,
				Parameters:  parameters,
				ReturnType:  returnType,
				Annotations: annotations,
			}

			result.CodeStructure.Methods = append(result.CodeStructure.Methods, methodInfo)
		}
	}
}

// parseScalaVariables extracts field and variable definitions
func (a *Analyzer) parseScalaVariables(content string, result *AnalysisResult) {
	// Scala field definition regex
	fieldRegex := regexp.MustCompile(`(?m)^\s*(?:@[A-Za-z][A-Za-z0-9_]*(?:\([^)]*\))?\s*)*(?:(private|protected|implicit|final|lazy)\s+)*(val|var)\s+([A-Za-z_][A-Za-z0-9_]*)\s*(?::\s*([A-Za-z_][A-Za-z0-9_\[\],\s]*))?\s*=`)

	matches := fieldRegex.FindAllStringSubmatch(content, -1)

	for _, match := range matches {
		if len(match) > 3 {
			modifiers := match[1]
			varType := match[2] // val or var
			varName := match[3]
			explicitType := ""
			if len(match) > 4 && match[4] != "" {
				explicitType = strings.TrimSpace(match[4])
			}

			lineNum := a.findLineNumber(content, match[0])

			// val is immutable (constant), var is mutable
			isConstant := varType == "val"

			variableInfo := VariableInfo{
				Name:       varName,
				Type:       explicitType,
				LineNumber: lineNum,
				IsConstant: isConstant,
				Scope:      a.determineScalaScope(modifiers),
			}

			result.CodeStructure.Variables = append(result.CodeStructure.Variables, variableInfo)
		}
	}
}

// analyzeScalaSecurityHints identifies potential security issues
func (a *Analyzer) analyzeScalaSecurityHints(content string, result *AnalysisResult) {
	securityPatterns := map[string]string{
		`(?i)password\s*=\s*["'][^"']+["']`:                 "Hardcoded password detected",
		`(?i)secret\s*=\s*["'][^"']+["']`:                   "Hardcoded secret detected",
		`(?i)apiKey\s*=\s*["'][^"']+["']`:                   "Hardcoded API key detected",
		`(?i)token\s*=\s*["'][^"']+["']`:                    "Hardcoded token detected",
		`SQL\s*\(\s*s?["'][^"']*\$\{[^}]*\}[^"']*["']\s*\)`: "Potential SQL injection with string interpolation",
		`Runtime\.getRuntime\.exec\(`:                       "Command execution detected",
		`scala\.sys\.process\.[^(]*\(`:                      "Process execution detected",
		`@Entity\s*(?:\([^)]*\))?\s*(?:class|case\s+class)`: "JPA Entity - potential data exposure",
		`@Controller\s*(?:\([^)]*\))?\s*(?:class|object)`:   "Controller class - verify input validation",
		`Future\s*\{[^}]*Thread\.sleep`:                     "Blocking operation in Future",
		`Await\.result\(`:                                   "Blocking await - potential performance issue",
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
			if strings.Contains(strings.ToLower(description), "execution") {
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

// parseScalaDependencies analyzes code dependencies
func (a *Analyzer) parseScalaDependencies(content string, result *AnalysisResult) {
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
			if impl != "" {
				dep := Dependency{
					Name:   class.Name,
					Type:   "trait_mixing",
					Target: impl,
				}
				result.Dependencies = append(result.Dependencies, dep)
			}
		}
	}
}

// Helper functions for Scala

func (a *Analyzer) parseScalaParameters(paramStr string) []string {
	if strings.TrimSpace(paramStr) == "" {
		return []string{}
	}

	params := strings.Split(paramStr, ",")
	var result []string

	for _, param := range params {
		param = strings.TrimSpace(param)
		if param != "" {
			// Scala parameter format: name: Type or name: Type = default
			parts := strings.Split(param, ":")
			if len(parts) >= 1 {
				paramName := strings.TrimSpace(parts[0])
				result = append(result, paramName)
			}
		}
	}

	return result
}

func (a *Analyzer) extractScalaAnnotations(content string, lineNum int) []string {
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

func (a *Analyzer) extractScalaPackage(content string) string {
	packageRegex := regexp.MustCompile(`(?m)^package\s+([a-zA-Z_][a-zA-Z0-9_]*(?:\.[a-zA-Z_][a-zA-Z0-9_]*)*)\s*$`)
	match := packageRegex.FindStringSubmatch(content)

	if len(match) > 1 {
		return match[1]
	}

	return ""
}

func (a *Analyzer) determineScalaScope(modifiers string) string {
	if strings.Contains(modifiers, "private") {
		return "private"
	}
	if strings.Contains(modifiers, "protected") {
		return "protected"
	}
	return "public" // Scala default is public
}
