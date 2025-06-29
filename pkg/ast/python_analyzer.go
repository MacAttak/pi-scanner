package ast

import (
	"context"
	"regexp"
	"strings"
	"time"
)

// analyzePython performs Python-specific AST analysis
func (a *Analyzer) analyzePython(ctx context.Context, filePath string, content []byte, result *AnalysisResult) (*AnalysisResult, error) {
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

	// Parse imports
	a.parsePythonImports(contentStr, result)

	// Parse classes
	a.parsePythonClasses(contentStr, result)

	// Parse functions and methods
	a.parsePythonMethods(contentStr, result)

	// Parse variables
	a.parsePythonVariables(contentStr, result)

	// Analyze security hints
	a.analyzePythonSecurityHints(contentStr, result)

	// Parse dependencies
	a.parsePythonDependencies(contentStr, result)

	return result, nil
}

// parsePythonImports extracts import statements
func (a *Analyzer) parsePythonImports(content string, result *AnalysisResult) {
	// Python import patterns
	importRegex := regexp.MustCompile(`(?m)^(?:from\s+([a-zA-Z_][a-zA-Z0-9_]*(?:\.[a-zA-Z_][a-zA-Z0-9_]*)*)\s+)?import\s+([a-zA-Z_][a-zA-Z0-9_]*(?:\s*,\s*[a-zA-Z_][a-zA-Z0-9_]*)*(?:\s+as\s+[a-zA-Z_][a-zA-Z0-9_]*)?|\*)`)

	matches := importRegex.FindAllStringSubmatch(content, -1)

	for _, match := range matches {
		if len(match) > 2 {
			fromModule := match[1]
			importedItems := match[2]

			var fullImport string
			if fromModule != "" {
				fullImport = fromModule + "." + importedItems
			} else {
				fullImport = importedItems
			}

			result.CodeStructure.Imports = append(result.CodeStructure.Imports, fullImport)

			// Create dependency
			result.Dependencies = append(result.Dependencies, Dependency{
				Name:   fullImport,
				Type:   "import",
				Target: fullImport,
			})
		}
	}
}

// parsePythonClasses extracts class definitions
func (a *Analyzer) parsePythonClasses(content string, result *AnalysisResult) {
	// Python class definition regex
	classRegex := regexp.MustCompile(`(?m)^(\s*)(?:@[a-zA-Z_][a-zA-Z0-9_]*(?:\([^)]*\))?\s*\n\s*)*class\s+([A-Za-z_][A-Za-z0-9_]*)\s*(?:\(([^)]*)\))?\s*:`)

	matches := classRegex.FindAllStringSubmatch(content, -1)

	for _, match := range matches {
		if len(match) > 2 {
			className := match[2]
			inheritance := ""
			if len(match) > 3 && match[3] != "" {
				inheritance = strings.TrimSpace(match[3])
			}

			var extends string
			var implements []string

			if inheritance != "" {
				// Parse inheritance - first class is usually base class
				inheritanceList := strings.Split(inheritance, ",")
				for i, parent := range inheritanceList {
					parent = strings.TrimSpace(parent)
					if i == 0 {
						extends = parent
					} else {
						implements = append(implements, parent)
					}
				}
			}

			// Find line number
			lineNum := a.findLineNumber(content, match[0])

			// Extract decorators for this class
			annotations := a.extractPythonDecorators(content, lineNum)

			// Python classes are public by default (no explicit access modifiers)
			classInfo := ClassInfo{
				Name:        className,
				Package:     "", // Python uses modules, not packages like Java
				LineNumber:  lineNum,
				IsPublic:    !strings.HasPrefix(className, "_"), // Convention: _ prefix indicates internal
				Extends:     extends,
				Implements:  implements,
				Annotations: annotations,
			}

			result.CodeStructure.Classes = append(result.CodeStructure.Classes, classInfo)
		}
	}
}

// parsePythonMethods extracts function and method definitions
func (a *Analyzer) parsePythonMethods(content string, result *AnalysisResult) {
	// Python function/method definition regex
	methodRegex := regexp.MustCompile(`(?m)^(\s*)(?:@[a-zA-Z_][a-zA-Z0-9_]*(?:\([^)]*\))?\s*\n\s*)*(?:async\s+)?def\s+([A-Za-z_][A-Za-z0-9_]*)\s*\(([^)]*)\)\s*(?:->\s*([^:]+))?\s*:`)

	matches := methodRegex.FindAllStringSubmatch(content, -1)

	for _, match := range matches {
		if len(match) > 2 {
			methodName := match[2]
			parametersStr := match[3]
			returnType := ""
			if len(match) > 4 && match[4] != "" {
				returnType = strings.TrimSpace(match[4])
			}

			parameters := a.parsePythonParameters(parametersStr)

			lineNum := a.findLineNumber(content, match[0])
			annotations := a.extractPythonDecorators(content, lineNum)

			// Python functions are public by default
			isPublic := !strings.HasPrefix(methodName, "_") // Convention: _ prefix indicates internal

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

// parsePythonVariables extracts variable and attribute definitions
func (a *Analyzer) parsePythonVariables(content string, result *AnalysisResult) {
	// Python variable assignment regex - class attributes and module-level variables
	variableRegex := regexp.MustCompile(`(?m)^(\s*)([A-Za-z_][A-Za-z0-9_]*)\s*:\s*([A-Za-z_][A-Za-z0-9_\[\],\s]*)\s*(?:=|$)`)

	matches := variableRegex.FindAllStringSubmatch(content, -1)

	for _, match := range matches {
		if len(match) > 3 {
			indentation := match[1]
			varName := match[2]
			varType := strings.TrimSpace(match[3])

			lineNum := a.findLineNumber(content, match[0])

			// Determine scope based on indentation and naming
			scope := "module"
			if len(indentation) > 0 {
				scope = "class"
			}

			// Python constants are typically ALL_CAPS
			isConstant := strings.ToUpper(varName) == varName && len(varName) > 1

			variableInfo := VariableInfo{
				Name:       varName,
				Type:       varType,
				LineNumber: lineNum,
				IsConstant: isConstant,
				Scope:      scope,
			}

			result.CodeStructure.Variables = append(result.CodeStructure.Variables, variableInfo)
		}
	}

	// Also look for simple assignments without type hints
	simpleAssignRegex := regexp.MustCompile(`(?m)^(\s*)([A-Z_][A-Z0-9_]+)\s*=\s*["'][^"']*["']`)
	simpleMatches := simpleAssignRegex.FindAllStringSubmatch(content, -1)

	for _, match := range simpleMatches {
		if len(match) > 2 {
			indentation := match[1]
			varName := match[2]

			lineNum := a.findLineNumber(content, match[0])

			scope := "module"
			if len(indentation) > 0 {
				scope = "class"
			}

			variableInfo := VariableInfo{
				Name:       varName,
				Type:       "str", // Inferred from string literal
				LineNumber: lineNum,
				IsConstant: true, // ALL_CAPS convention
				Scope:      scope,
			}

			result.CodeStructure.Variables = append(result.CodeStructure.Variables, variableInfo)
		}
	}
}

// analyzePythonSecurityHints identifies potential security issues
func (a *Analyzer) analyzePythonSecurityHints(content string, result *AnalysisResult) {
	securityPatterns := map[string]string{
		`(?i)password\s*=\s*["'][^"']+["']`:                     "Hardcoded password detected",
		`(?i)secret\s*=\s*["'][^"']+["']`:                       "Hardcoded secret detected",
		`(?i)\w*secret\w*\s*=\s*["'][^"']+["']`:                 "Hardcoded secret detected",
		`(?i)api[_-]?key\s*=\s*["'][^"']+["']`:                  "Hardcoded API key detected",
		`(?i)token\s*=\s*["'][^"']+["']`:                        "Hardcoded token detected",
		`cursor\.execute\s*\(\s*[f"'][^"']*\{[^}]*\}[^"']*["']`: "Potential SQL injection with f-string",
		`\.format\s*\([^)]*\)\s*\)`:                             "String formatting - check for injection",
		`eval\s*\(`:                                             "Dangerous eval() usage",
		`exec\s*\(`:                                             "Dangerous exec() usage",
		`subprocess\.\w+\([^)]*shell\s*=\s*True`:                "Shell injection risk",
		`os\.system\(`:                                          "Command execution with os.system",
		`pickle\.loads?\(`:                                      "Pickle deserialization - potential code execution",
		`yaml\.load\([^)]*\)`:                                   "Unsafe YAML loading - check for SafeLoader",
		`@app\.route\s*\([^)]*\<[^>]*\>`:                        "Flask route with dynamic parameters",
		`request\.args\[[^]]*\]`:                                "Direct request parameter access",
		`request\.form\[[^]]*\]`:                                "Direct form data access",
		`open\s*\([^)]*mode\s*=\s*["']w`:                        "File write operation",
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
			if strings.Contains(strings.ToLower(description), "dangerous") ||
				strings.Contains(strings.ToLower(description), "code execution") {
				severity = "CRITICAL"
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

// parsePythonDependencies analyzes code dependencies
func (a *Analyzer) parsePythonDependencies(content string, result *AnalysisResult) {
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
					Type:   "multiple_inheritance",
					Target: impl,
				}
				result.Dependencies = append(result.Dependencies, dep)
			}
		}
	}
}

// Helper functions for Python

func (a *Analyzer) parsePythonParameters(paramStr string) []string {
	if strings.TrimSpace(paramStr) == "" {
		return []string{}
	}

	params := strings.Split(paramStr, ",")
	var result []string

	for _, param := range params {
		param = strings.TrimSpace(param)
		if param != "" {
			// Python parameter formats:
			// name, name: type, name = default, name: type = default
			// *args, **kwargs, name=default

			// Remove default values
			if strings.Contains(param, "=") {
				param = strings.Split(param, "=")[0]
				param = strings.TrimSpace(param)
			}

			// Remove type hints
			if strings.Contains(param, ":") {
				param = strings.Split(param, ":")[0]
				param = strings.TrimSpace(param)
			}

			// Skip *args, **kwargs special parameters for now
			if !strings.HasPrefix(param, "*") && param != "" {
				result = append(result, param)
			}
		}
	}

	return result
}

func (a *Analyzer) extractPythonDecorators(content string, lineNum int) []string {
	lines := strings.Split(content, "\n")
	var decorators []string

	// Look backwards from the line to find decorators
	for i := lineNum - 2; i >= 0 && i < len(lines); i-- {
		line := strings.TrimSpace(lines[i])
		if strings.HasPrefix(line, "@") {
			decorators = append([]string{line}, decorators...) // Prepend to maintain order
		} else if line != "" && !strings.HasPrefix(line, "#") {
			break // Stop at non-decorator, non-comment line
		}
	}

	return decorators
}
