package ast

import (
	"context"
	"path/filepath"
	"strings"
	"sync"

	"github.com/MacAttak/pi-scanner/pkg/discovery"
)

// RepositoryStructure represents the analyzed structure of a repository
type RepositoryStructure struct {
	RootPath        string
	PrimaryLanguage string
	Languages       map[Language]int    // Count of files per language
	HighRiskZones   map[string][]string // Zone name -> file paths
	FileContexts    map[string]*FileContext
	Dependencies    map[string][]string // Package dependencies
	mu              sync.RWMutex
}

// FileContext contains AST-derived context for a specific file
type FileContext struct {
	FilePath      string
	Language      Language
	RiskLevel     RiskLevel
	RiskZone      string
	IsTestFile    bool
	IsConfigFile  bool
	CodeStructure *CodeStructure
	Imports       []string
	Classes       []string
	Methods       []string
}

// AnalyzeRepository performs repository-wide AST analysis
func (a *Analyzer) AnalyzeRepository(ctx context.Context, rootPath string, files []discovery.FileResult) (*RepositoryStructure, error) {
	repo := &RepositoryStructure{
		RootPath:      rootPath,
		Languages:     make(map[Language]int),
		HighRiskZones: make(map[string][]string),
		FileContexts:  make(map[string]*FileContext),
		Dependencies:  make(map[string][]string),
	}

	// Analyze files concurrently
	type result struct {
		file    discovery.FileResult
		context *FileContext
		err     error
	}

	// Create worker pool
	numWorkers := a.config.ConcurrentAnalysis
	if numWorkers <= 0 {
		numWorkers = 4
	}

	jobs := make(chan discovery.FileResult, len(files))
	results := make(chan result, len(files))

	// Start workers
	var wg sync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for file := range jobs {
				select {
				case <-ctx.Done():
					return
				default:
					fileCtx, err := a.analyzeFileForRepo(ctx, file)
					results <- result{file: file, context: fileCtx, err: err}
				}
			}
		}()
	}

	// Queue jobs
	for _, file := range files {
		if file.IsBinary {
			continue
		}
		jobs <- file
	}
	close(jobs)

	// Wait for completion
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results
	for res := range results {
		if res.err != nil {
			// Log error but continue
			continue
		}

		if res.context != nil {
			repo.mu.Lock()
			repo.FileContexts[res.file.Path] = res.context

			// Update language counts
			if res.context.Language != "" {
				repo.Languages[res.context.Language]++
			}

			// Track high-risk zones
			if res.context.RiskLevel == RiskLevelHigh || res.context.RiskLevel == RiskLevelCritical {
				repo.HighRiskZones[res.context.RiskZone] = append(repo.HighRiskZones[res.context.RiskZone], res.file.Path)
			}

			// Track dependencies
			for _, imp := range res.context.Imports {
				pkg := extractPackageName(imp)
				if pkg != "" && !isStandardLibrary(pkg) {
					repo.Dependencies[pkg] = append(repo.Dependencies[pkg], res.file.Path)
				}
			}
			repo.mu.Unlock()
		}
	}

	// Determine primary language
	repo.determinePrimaryLanguage()

	return repo, nil
}

// analyzeFileForRepo analyzes a single file for repository context
func (a *Analyzer) analyzeFileForRepo(ctx context.Context, file discovery.FileResult) (*FileContext, error) {
	language := a.detectLanguage(file.Path)
	if language == "" {
		return nil, nil // Skip non-supported files
	}

	fileCtx := &FileContext{
		FilePath:     file.Path,
		Language:     language,
		RiskLevel:    a.DetermineRiskLevel(file.Path),
		RiskZone:     a.DetermineRiskZone(file.Path),
		IsTestFile:   a.isTestFile(file.Path),
		IsConfigFile: a.isConfigFile(file.Path),
	}

	// For now, we'll skip actual AST parsing to focus on integration
	// This would be expanded with tree-sitter parsing
	fileCtx.extractBasicInfo(file.Path)

	return fileCtx, nil
}

// GetFileContext returns the context for a specific file
func (rs *RepositoryStructure) GetFileContext(filePath string) *FileContext {
	rs.mu.RLock()
	defer rs.mu.RUnlock()
	return rs.FileContexts[filePath]
}

// IsTestFile checks if a file is a test file based on repository analysis
func (rs *RepositoryStructure) IsTestFile(filePath string) bool {
	rs.mu.RLock()
	defer rs.mu.RUnlock()

	if ctx, ok := rs.FileContexts[filePath]; ok {
		return ctx.IsTestFile
	}

	// Fallback to path-based detection
	lower := strings.ToLower(filePath)
	return strings.Contains(lower, "test") || strings.Contains(lower, "spec")
}

// determinePrimaryLanguage determines the primary language of the repository
func (rs *RepositoryStructure) determinePrimaryLanguage() {
	maxCount := 0
	for lang, count := range rs.Languages {
		if count > maxCount {
			maxCount = count
			rs.PrimaryLanguage = string(lang)
		}
	}

	// Map language to readable name
	switch rs.PrimaryLanguage {
	case string(LanguageJava):
		if rs.hasFramework("spring") {
			rs.PrimaryLanguage = "Java Spring Boot"
		} else {
			rs.PrimaryLanguage = "Java"
		}
	case string(LanguageScala):
		if rs.hasFramework("play") {
			rs.PrimaryLanguage = "Scala Play Framework"
		} else if rs.hasFramework("akka") {
			rs.PrimaryLanguage = "Scala Akka"
		} else {
			rs.PrimaryLanguage = "Scala"
		}
	case string(LanguagePython):
		if rs.hasFramework("django") {
			rs.PrimaryLanguage = "Python Django"
		} else if rs.hasFramework("flask") {
			rs.PrimaryLanguage = "Python Flask"
		} else {
			rs.PrimaryLanguage = "Python"
		}
	}
}

// hasFramework checks if the repository uses a specific framework
func (rs *RepositoryStructure) hasFramework(framework string) bool {
	framework = strings.ToLower(framework)
	for pkg := range rs.Dependencies {
		if strings.Contains(strings.ToLower(pkg), framework) {
			return true
		}
	}
	return false
}

// extractBasicInfo extracts basic information from file path and name
func (fc *FileContext) extractBasicInfo(filePath string) {
	// Extract package/module structure
	dir := filepath.Dir(filePath)
	parts := strings.Split(filepath.ToSlash(dir), "/")

	// Look for common patterns
	for _, part := range parts {
		lower := strings.ToLower(part)
		switch {
		case strings.Contains(lower, "model") || strings.Contains(lower, "entity"):
			fc.Classes = append(fc.Classes, "DataModel")
		case strings.Contains(lower, "service"):
			fc.Classes = append(fc.Classes, "Service")
		case strings.Contains(lower, "controller"):
			fc.Classes = append(fc.Classes, "Controller")
		case strings.Contains(lower, "repository") || strings.Contains(lower, "dao"):
			fc.Classes = append(fc.Classes, "Repository")
		}
	}
}

// Helper functions

func (a *Analyzer) isTestFile(filePath string) bool {
	base := filepath.Base(filePath)
	lower := strings.ToLower(base)

	// Check file name patterns
	testPatterns := []string{
		"test", "spec", "_test.", ".test.", "_spec.", ".spec.",
	}

	for _, pattern := range testPatterns {
		if strings.Contains(lower, pattern) {
			return true
		}
	}

	// Check directory patterns
	dir := strings.ToLower(filepath.Dir(filePath))
	return strings.Contains(dir, "/test/") || strings.Contains(dir, "/tests/") ||
		strings.Contains(dir, "/spec/") || strings.Contains(dir, "/specs/")
}

func (a *Analyzer) isConfigFile(filePath string) bool {
	ext := strings.ToLower(filepath.Ext(filePath))
	configExts := []string{
		".yaml", ".yml", ".json", ".xml", ".properties", ".conf", ".ini", ".toml",
	}

	for _, ce := range configExts {
		if ext == ce {
			return true
		}
	}

	base := strings.ToLower(filepath.Base(filePath))
	return strings.Contains(base, "config") || strings.Contains(base, "settings")
}

func extractPackageName(importPath string) string {
	// Extract package name from import statement
	// This is simplified - real implementation would be language-specific
	importPath = strings.Trim(importPath, "\"'")
	parts := strings.Split(importPath, "/")
	if len(parts) > 0 {
		return parts[0]
	}
	return ""
}

func isStandardLibrary(pkg string) bool {
	// Check if package is from standard library
	// This is simplified - real implementation would be language-specific
	stdLibs := []string{
		"java.", "javax.", "scala.", "python", "os", "sys", "io", "fmt",
		"strings", "net", "http", "time", "encoding", "crypto",
	}

	for _, std := range stdLibs {
		if strings.HasPrefix(pkg, std) {
			return true
		}
	}
	return false
}
