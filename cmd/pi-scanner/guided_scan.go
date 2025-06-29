package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/MacAttak/pi-scanner/pkg/ast"
	"github.com/MacAttak/pi-scanner/pkg/detection"
	"github.com/MacAttak/pi-scanner/pkg/discovery"
	"github.com/MacAttak/pi-scanner/pkg/llm"
	"github.com/MacAttak/pi-scanner/pkg/output"
	"github.com/MacAttak/pi-scanner/pkg/processing"
	"github.com/MacAttak/pi-scanner/pkg/report"
	"github.com/MacAttak/pi-scanner/pkg/repository"
)

// PatternScanResult represents the results of pattern-based scanning
type PatternScanResult struct {
	Repository   *repository.RepositoryInfo `json:"repository"`
	ScanStarted  time.Time                  `json:"scan_started"`
	ScanFinished time.Time                  `json:"scan_finished"`
	Duration     time.Duration              `json:"duration"`
	FilesScanned int                        `json:"files_scanned"`
	Findings     []detection.Finding        `json:"findings"` // Masked findings for reports
	RawFindings  []detection.Finding        `json:"-"`        // Unmasked findings for validation (not exported)
	Stats        ScanStats                  `json:"stats"`
	ReportDir    string                     `json:"report_directory"`
	Error        string                     `json:"error,omitempty"`
}

// ScanStats provides statistics about the scan
type ScanStats struct {
	TotalFiles     int            `json:"total_files"`
	ScannedFiles   int            `json:"scanned_files"`
	SkippedFiles   int            `json:"skipped_files"`
	TotalSize      int64          `json:"total_size"`
	FindingsByType map[string]int `json:"findings_by_type"`
	FindingsByRisk map[string]int `json:"findings_by_risk"`
	ProcessingTime time.Duration  `json:"processing_time"`
}

// ValidationResult represents the results of LLM validation
type ValidationResult struct {
	OriginalScan    *PatternScanResult  `json:"original_scan"`
	ValidationStart time.Time           `json:"validation_started"`
	ValidationEnd   time.Time           `json:"validation_finished"`
	Duration        time.Duration       `json:"duration"`
	TotalFindings   int                 `json:"total_findings"`
	ValidatedCount  int                 `json:"validated_count"`
	Findings        []detection.Finding `json:"findings"`
	Stats           ValidationStats     `json:"stats"`
}

// ValidationStats provides validation statistics
type ValidationStats struct {
	ByRiskChange map[string]int `json:"by_risk_change"`
	Downgraded   int            `json:"downgraded"`
	Upgraded     int            `json:"upgraded"`
	Confirmed    int            `json:"confirmed"`
}

// ScanResourceManager manages the lifecycle of all resources for a complete scan operation
type ScanResourceManager struct {
	ctx         context.Context
	cancel      context.CancelFunc
	repoManager *repository.RepositoryManager
	repoInfo    *repository.RepositoryInfo
	cleanupFns  []func() error
	mu          sync.Mutex
}

// NewScanResourceManager creates a new resource manager for scan operations
func NewScanResourceManager(ctx context.Context, repoURL string) (*ScanResourceManager, error) {
	// Create cancellable context for the entire scan operation
	scanCtx, cancel := context.WithCancel(ctx)

	srm := &ScanResourceManager{
		ctx:        scanCtx,
		cancel:     cancel,
		cleanupFns: make([]func() error, 0),
	}

	// Set up repository manager
	repoConfig := repository.DefaultGitHubConfig()
	srm.repoManager = repository.NewRepositoryManager(repoConfig)

	// Check authentication
	if err := srm.repoManager.CheckAuthentication(scanCtx); err != nil {
		cancel()
		return nil, fmt.Errorf("GitHub authentication failed: %w", err)
	}

	// Clone repository
	repoInfo, err := srm.repoManager.CloneAndTrack(scanCtx, repoURL)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to clone repository: %w", err)
	}
	srm.repoInfo = repoInfo

	// Register cleanup function for repository
	srm.addCleanupFn(func() error {
		return srm.repoManager.CleanupAll()
	})

	return srm, nil
}

// GetRepositoryInfo returns the repository information for use by scan phases
func (srm *ScanResourceManager) GetRepositoryInfo() *repository.RepositoryInfo {
	return srm.repoInfo
}

// GetContext returns the scan context for cancellation/timeout handling
func (srm *ScanResourceManager) GetContext() context.Context {
	return srm.ctx
}

// addCleanupFn registers a cleanup function to be called during resource cleanup
func (srm *ScanResourceManager) addCleanupFn(fn func() error) {
	srm.mu.Lock()
	defer srm.mu.Unlock()
	srm.cleanupFns = append(srm.cleanupFns, fn)
}

// Cleanup releases all resources managed by this resource manager
func (srm *ScanResourceManager) Cleanup() error {
	// Cancel the context first to signal all operations to stop
	srm.cancel()

	srm.mu.Lock()
	defer srm.mu.Unlock()

	var firstError error

	// Run cleanup functions in reverse order (LIFO)
	for i := len(srm.cleanupFns) - 1; i >= 0; i-- {
		if err := srm.cleanupFns[i](); err != nil && firstError == nil {
			firstError = err
		}
	}

	return firstError
}

// runPatternScan performs pattern-based scanning and returns results
func runPatternScan(ctx context.Context, resourceManager *ScanResourceManager, repoURL string) (*PatternScanResult, string, error) {
	result := &PatternScanResult{
		ScanStarted: time.Now(),
		Stats: ScanStats{
			FindingsByType: make(map[string]int),
			FindingsByRisk: make(map[string]int),
		},
	}

	// Create report manager and directory
	reportManager := report.NewManager("")
	reportDir, err := reportManager.CreateReportDirectory(repoURL)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create report directory: %w", err)
	}
	result.ReportDir = reportDir

	// Get repository info from resource manager
	repoInfo := resourceManager.GetRepositoryInfo()
	result.Repository = repoInfo

	// Set up detectors (pattern-based only)
	detectionConfig := detection.DefaultConfig()
	detectionConfig.EnableLLMValidation = false // Never use LLM in phase 1

	var detectors []detection.Detector

	// Add pattern detector
	patternDetector := detection.NewDetector()
	detectors = append(detectors, patternDetector)

	// Add Gitleaks detector
	gitleaksDetector, err := detection.NewGitleaksDetector("")
	if err != nil {
		if verbose {
			fmt.Printf("⚠️  Gitleaks detector setup failed: %v\n", err)
		}
	} else {
		detectors = append(detectors, gitleaksDetector)
	}

	// Discover files
	fmt.Printf("📂 Discovering files...")
	discoveryConfig := discovery.DefaultConfig()
	fileDiscovery := discovery.NewFileDiscovery(discoveryConfig)

	files, err := fileDiscovery.DiscoverFiles(ctx, repoInfo.LocalPath)
	if err != nil {
		fmt.Printf(" ❌\n")
		result.Error = fmt.Sprintf("File discovery failed: %v", err)
		if err := savePatternResult(result, reportManager.GetPhase1Path(reportDir)); err != nil {
			fmt.Printf("Warning: failed to save pattern result: %v\n", err)
		}
		return result, reportDir, fmt.Errorf("file discovery failed: %w", err)
	}
	fmt.Printf(" ✅ (%d files)\n", len(files))

	// Analyze code structure (AST)
	var repoStructure *ast.RepositoryStructure
	fmt.Printf("🔍 Analyzing code structure...")
	astAnalyzer := ast.NewAnalyzer(ast.DefaultBankingConfig())
	repoStructure, err = astAnalyzer.AnalyzeRepository(ctx, repoInfo.LocalPath, files)
	if err != nil {
		// AST analysis failure is not fatal - continue with pattern detection
		fmt.Printf(" ⚠️  (continuing without AST analysis)\n")
		if verbose {
			fmt.Printf("   AST analysis error: %v\n", err)
		}
	} else {
		fmt.Printf(" ✅\n")
		// Display repository insights
		if repoStructure.PrimaryLanguage != "" {
			fmt.Printf("   - Detected: %s application\n", repoStructure.PrimaryLanguage)
		}
		if len(repoStructure.HighRiskZones) > 0 {
			zones := []string{}
			for zone := range repoStructure.HighRiskZones {
				zones = append(zones, zone)
			}
			fmt.Printf("   - High-risk zones: %s\n", strings.Join(zones, ", "))
		}
		testCount := 0
		for _, f := range files {
			if repoStructure.IsTestFile(f.Path) {
				testCount++
			}
		}
		fmt.Printf("   - Test files: %d (will be marked lower risk)\n", testCount)
	}

	result.Stats.TotalFiles = len(files)

	// Set up file processor
	processorConfig := processing.DefaultProcessorConfig()
	processorConfig.NumWorkers = 8 // More workers since no LLM bottleneck

	fileProcessor := processing.NewFileProcessor(processorConfig, detectors)

	// Create processing jobs
	var jobs []processing.FileJob
	for _, file := range files {
		if file.IsBinary {
			result.Stats.SkippedFiles++
			continue
		}

		// Read file content
		content, err := os.ReadFile(file.Path)
		if err != nil {
			result.Stats.SkippedFiles++
			continue
		}

		result.Stats.TotalSize += int64(len(content))

		job := processing.FileJob{
			FilePath: file.Path,
			Content:  content,
			FileInfo: file,
		}

		// Add AST context if available
		if repoStructure != nil {
			job.ASTContext = repoStructure.GetFileContext(file.Path)
		}

		jobs = append(jobs, job)
	}

	result.Stats.ScannedFiles = len(jobs)

	// Process files with progress
	fmt.Printf("🔍 Scanning for PI patterns...\n")
	processingStart := time.Now()

	// Create a simple progress tracker
	processed := 0
	progressDone := make(chan bool)
	go func() {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if processed < len(jobs) {
					rate := float64(processed) / time.Since(processingStart).Seconds()
					remaining := float64(len(jobs)-processed) / rate
					fmt.Printf("\r   Processing: [%s] %d/%d files | %.0f files/sec | ~%.0fs remaining",
						progressBar(processed, len(jobs), 30),
						processed, len(jobs), rate, remaining)
				}
			case <-progressDone:
				fmt.Printf("\r   Processing: [%s] %d/%d files | Complete!                     \n",
					progressBar(len(jobs), len(jobs), 30),
					len(jobs), len(jobs))
				return
			}
		}
	}()

	batchProcessor := processing.NewBatchProcessor(fileProcessor, 100)
	results, err := batchProcessor.ProcessFiles(ctx, jobs)
	processed = len(results)
	progressDone <- true

	if err != nil {
		result.Error = fmt.Sprintf("File processing failed: %v", err)
		if err := savePatternResult(result, reportManager.GetPhase1Path(reportDir)); err != nil {
			fmt.Printf("Warning: failed to save pattern result: %v\n", err)
		}
		return result, reportDir, fmt.Errorf("file processing failed: %w", err)
	}

	result.Stats.ProcessingTime = time.Since(processingStart)

	// Collect findings
	var allFindings []detection.Finding
	for _, procResult := range results {
		if procResult.Error != nil {
			continue
		}

		for _, finding := range procResult.Findings {
			allFindings = append(allFindings, finding)

			// Update statistics
			piType := string(finding.Type)
			result.Stats.FindingsByType[piType]++

			riskLevel := string(finding.RiskLevel)
			result.Stats.FindingsByRisk[riskLevel]++
		}
	}

	// Store raw unmasked findings for LLM validation
	result.RawFindings = allFindings
	result.FilesScanned = len(results)
	result.ScanFinished = time.Now()
	result.Duration = result.ScanFinished.Sub(result.ScanStarted)

	// Apply masking only for report output
	outputConfig := &output.Config{
		MaskingLevel:         getMaskingLevel(),
		EnableAuditLogging:   false,
		WarnOnInsecureConfig: false,
	}

	outputManager, err := output.NewManager(outputConfig, nil)
	if err != nil {
		return result, reportDir, fmt.Errorf("failed to create output manager: %w", err)
	}

	// Create masked findings for reports while preserving raw findings for validation
	result.Findings = outputManager.PrepareFindings(allFindings)

	// Save results
	phase1Path := reportManager.GetPhase1Path(reportDir)
	if err := savePatternResult(result, phase1Path); err != nil {
		return result, reportDir, err
	}

	// Save summary
	summaryPath := reportManager.GetSummaryPath(reportDir)
	if err := saveSummary(result, summaryPath); err != nil {
		return result, reportDir, err
	}

	return result, reportDir, nil
}

// runLLMValidation performs LLM validation on findings
func runLLMValidation(ctx context.Context, scanResult *PatternScanResult, scope string, reportDir string) (*ValidationResult, error) {
	// Check LLM availability
	llmConfig := llm.Config{
		Enabled:     true,
		Provider:    "lmstudio",
		Endpoint:    "http://localhost:1234/v1",
		Model:       "qwen2.5-coder-7b-instruct",
		APIKey:      "lm-studio",
		MaxTokens:   1000,
		Temperature: 0.3,
		Timeout:     30 * time.Second,
	}

	llmClient, err := llm.NewLMStudioClient(llmConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create LLM client: %w", err)
	}

	// Test LLM connection
	fmt.Printf("🔍 Checking LLM availability...")
	testCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	err = llmClient.HealthCheck(testCtx)
	cancel()

	if err != nil {
		fmt.Printf(" ❌\n")
		return nil, fmt.Errorf("LLM service not available: %w", err)
	}
	fmt.Printf(" ✅\n")

	// Filter findings based on scope using raw unmasked findings
	toValidate := filterFindingsByScope(scanResult.RawFindings, scope)
	if len(toValidate) == 0 {
		fmt.Println("No findings to validate based on selected scope.")
		return nil, nil
	}

	fmt.Printf("🤖 Validating %d findings with AI...\n", len(toValidate))

	// Create validation result
	result := &ValidationResult{
		OriginalScan:    scanResult,
		ValidationStart: time.Now(),
		TotalFindings:   len(scanResult.RawFindings),
		ValidatedCount:  len(toValidate),
		Stats: ValidationStats{
			ByRiskChange: make(map[string]int),
		},
	}

	// Perform validation with progress tracking using raw unmasked findings
	validatedFindings, err := validateWithProgress(ctx, llmClient, toValidate, scanResult.RawFindings)
	if err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	result.Findings = validatedFindings
	result.ValidationEnd = time.Now()
	result.Duration = result.ValidationEnd.Sub(result.ValidationStart)

	// Calculate statistics
	calculateValidationStats(result, scanResult.Findings)

	// Apply masking before saving
	outputConfig := &output.Config{
		MaskingLevel:         getMaskingLevel(),
		EnableAuditLogging:   false,
		WarnOnInsecureConfig: false,
	}

	outputManager, err := output.NewManager(outputConfig, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create output manager: %w", err)
	}

	result.Findings = outputManager.PrepareFindings(result.Findings)

	// Save results
	reportManager := report.NewManager("")
	phase2Path := reportManager.GetPhase2Path(reportDir)

	jsonData, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal results: %w", err)
	}

	if err := os.WriteFile(phase2Path, jsonData, 0644); err != nil {
		return nil, fmt.Errorf("failed to write results: %w", err)
	}

	return result, nil
}

// Helper functions

func getMaskingLevel() output.MaskingLevel {
	switch strings.ToUpper(maskingLevel) {
	case "FULL":
		return output.MaskingLevelFull
	case "NONE":
		return output.MaskingLevelNone
	default:
		return output.MaskingLevelPartial
	}
}

func savePatternResult(result *PatternScanResult, outputFile string) error {
	jsonData, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal results: %w", err)
	}

	err = os.WriteFile(outputFile, jsonData, 0644)
	if err != nil {
		return fmt.Errorf("failed to write results file: %w", err)
	}

	return nil
}

func saveSummary(result *PatternScanResult, summaryFile string) error {
	var content string

	content += "PI Scanner Report\n"
	content += "================\n\n"
	content += fmt.Sprintf("Repository: %s\n", result.Repository.URL)
	content += fmt.Sprintf("Scan Date: %s\n", result.ScanStarted.Format("2006-01-02 15:04:05"))
	content += fmt.Sprintf("Duration: %.1fs\n", result.Duration.Seconds())
	content += fmt.Sprintf("Files Scanned: %d\n\n", result.FilesScanned)

	content += fmt.Sprintf("Total Findings: %d\n\n", len(result.Findings))

	content += "By Risk Level:\n"
	content += fmt.Sprintf("  CRITICAL: %d\n", result.Stats.FindingsByRisk["CRITICAL"])
	content += fmt.Sprintf("  HIGH: %d\n", result.Stats.FindingsByRisk["HIGH"])
	content += fmt.Sprintf("  MEDIUM: %d\n", result.Stats.FindingsByRisk["MEDIUM"])
	content += fmt.Sprintf("  LOW: %d\n\n", result.Stats.FindingsByRisk["LOW"])

	content += "By PI Type:\n"
	for piType, count := range result.Stats.FindingsByType {
		content += fmt.Sprintf("  %s: %d\n", piType, count)
	}

	return os.WriteFile(summaryFile, []byte(content), 0644)
}

func displayScanSummary(result *PatternScanResult) {
	fmt.Printf("\n📊 Pattern Scan Results\n")
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	if result.Repository != nil {
		fmt.Printf("Repository: %s\n", result.Repository.URL)
	}
	fmt.Printf("Files scanned: %d\n", result.FilesScanned)
	fmt.Printf("Time taken: %.1fs\n\n", result.Duration.Seconds())

	total := len(result.Findings)
	if total == 0 {
		fmt.Printf("✨ No PI findings detected!\n")
		return
	}

	fmt.Printf("Findings by Risk Level:\n")
	fmt.Printf("  🔴 CRITICAL: %d\n", result.Stats.FindingsByRisk["CRITICAL"])
	fmt.Printf("  🟠 HIGH: %d\n", result.Stats.FindingsByRisk["HIGH"])
	fmt.Printf("  🟡 MEDIUM: %d\n", result.Stats.FindingsByRisk["MEDIUM"])
	fmt.Printf("  🟢 LOW: %d\n\n", result.Stats.FindingsByRisk["LOW"])

	fmt.Printf("PI Types Detected:\n")
	for piType, count := range result.Stats.FindingsByType {
		fmt.Printf("  • %s: %d\n", piType, count)
	}
}

func displayValidationSummary(result *ValidationResult) {
	fmt.Printf("\n✅ Validation Complete\n")
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	fmt.Printf("Validated: %d findings\n", result.ValidatedCount)
	fmt.Printf("Time taken: %.1fs\n\n", result.Duration.Seconds())

	fmt.Printf("Risk Assessment Changes:\n")
	fmt.Printf("  ⬇️  Downgraded to lower risk: %d\n", result.Stats.Downgraded)
	fmt.Printf("  ⬆️  Upgraded to higher risk: %d\n", result.Stats.Upgraded)
	fmt.Printf("  ✓  Confirmed at same risk: %d\n", result.Stats.Confirmed)

	if len(result.Stats.ByRiskChange) > 0 && verbose {
		fmt.Printf("\nDetailed Risk Changes:\n")
		for change, count := range result.Stats.ByRiskChange {
			fmt.Printf("  • %s: %d\n", change, count)
		}
	}
}

func filterFindingsByScope(findings []detection.Finding, scope string) []detection.Finding {
	var filtered []detection.Finding

	for _, f := range findings {
		switch scope {
		case "all":
			filtered = append(filtered, f)
		case "high":
			if f.RiskLevel == detection.RiskLevelCritical || f.RiskLevel == detection.RiskLevelHigh {
				filtered = append(filtered, f)
			}
		case "high-medium":
			if f.RiskLevel == detection.RiskLevelCritical ||
				f.RiskLevel == detection.RiskLevelHigh ||
				f.RiskLevel == detection.RiskLevelMedium {
				filtered = append(filtered, f)
			}
		}
	}

	return filtered
}

func validateWithProgress(ctx context.Context, llmClient *llm.LMStudioClient, toValidate, allFindings []detection.Finding) ([]detection.Finding, error) {
	// Create a map for quick lookup
	validateMap := make(map[string]bool)
	for _, f := range toValidate {
		key := fmt.Sprintf("%s:%d:%s", f.File, f.Line, f.Match)
		validateMap[key] = true
	}

	// Create enhanced detector for validation
	detectionConfig := detection.DefaultConfig()
	detectionConfig.EnableLLMValidation = true
	detectionConfig.LLMValidateRisks = []detection.RiskLevel{
		detection.RiskLevelCritical,
		detection.RiskLevelHigh,
		detection.RiskLevelMedium,
		detection.RiskLevelLow,
	}

	// Create base detector that returns our existing findings
	baseDetector := &replayDetector{
		findings: allFindings,
		filter:   validateMap,
	}

	enhancedDetector := detection.NewLLMEnhancedDetector(baseDetector, llmClient, &detection.LLMEnhancedConfig{
		Enabled:            true,
		ValidateRiskLevels: detectionConfig.LLMValidateRisks,
		MaxConcurrency:     20,
		SkipTestFiles:      false,
	})

	// Set up progress tracking
	enhancedDetector.SetProgressCallback(func(p, t int, rate float64) {
		remaining := float64(t-p) / rate
		fmt.Printf("\r   Validating: [%s] %d/%d (%.0f%%) | %.1f/min | ~%.0f min remaining",
			progressBar(p, t, 30), p, t, float64(p)/float64(t)*100, rate, remaining)
	})

	// Group findings by file
	fileGroups := make(map[string][]detection.Finding)
	for _, f := range allFindings {
		fileGroups[f.File] = append(fileGroups[f.File], f)
	}

	// Process each file through the enhanced detector
	resultMap := make(map[string]detection.Finding)

	// First add all original findings
	for _, f := range allFindings {
		key := fmt.Sprintf("%s:%d:%s", f.File, f.Line, f.Match)
		resultMap[key] = f
	}

	// Process files that have findings to validate
	for file, findings := range fileGroups {
		// Check if any findings in this file need validation
		hasValidationTarget := false
		for _, f := range findings {
			key := fmt.Sprintf("%s:%d:%s", f.File, f.Line, f.Match)
			if validateMap[key] {
				hasValidationTarget = true
				break
			}
		}

		if !hasValidationTarget {
			continue
		}

		// Read file content
		content, err := os.ReadFile(file)
		if err != nil {
			if verbose {
				fmt.Printf("\nWarning: Could not read file %s: %v\n", file, err)
			}
			continue
		}

		// Run enhanced detection (which will do LLM validation)
		enhancedFindings, err := enhancedDetector.Detect(ctx, content, file)
		if err != nil {
			if verbose {
				fmt.Printf("\nWarning: Validation error for %s: %v\n", file, err)
			}
			continue
		}

		// Update results with enhanced findings
		for _, f := range enhancedFindings {
			key := fmt.Sprintf("%s:%d:%s", f.File, f.Line, f.Match)
			if f.LLMValidated {
				resultMap[key] = f
			}
		}
	}

	fmt.Println() // Clear progress line

	// Convert back to slice
	var result []detection.Finding
	for _, f := range resultMap {
		result = append(result, f)
	}

	return result, nil
}

func calculateValidationStats(result *ValidationResult, originalFindings []detection.Finding) {
	// Create maps for comparison
	originalMap := make(map[string]detection.Finding)
	for _, f := range originalFindings {
		key := fmt.Sprintf("%s:%d:%s", f.File, f.Line, f.Match)
		originalMap[key] = f
	}

	for _, f := range result.Findings {
		if !f.LLMValidated {
			continue
		}

		key := fmt.Sprintf("%s:%d:%s", f.File, f.Line, f.Match)
		if orig, ok := originalMap[key]; ok {
			if f.LLMRisk < orig.RiskLevel {
				result.Stats.Downgraded++
			} else if f.LLMRisk > orig.RiskLevel {
				result.Stats.Upgraded++
			} else {
				result.Stats.Confirmed++
			}

			change := fmt.Sprintf("%s→%s", orig.RiskLevel, f.LLMRisk)
			result.Stats.ByRiskChange[change]++
		}
	}
}

func progressBar(current, total, width int) string {
	if total == 0 {
		return strings.Repeat("░", width)
	}

	filled := (current * width) / total
	if filled > width {
		filled = width
	}

	bar := strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
	return bar
}

// replayDetector returns existing findings for a file
type replayDetector struct {
	findings []detection.Finding
	filter   map[string]bool
}

func (d *replayDetector) Detect(ctx context.Context, content []byte, filename string) ([]detection.Finding, error) {
	var result []detection.Finding
	for _, f := range d.findings {
		if f.File == filename {
			key := fmt.Sprintf("%s:%d:%s", f.File, f.Line, f.Match)
			// Only return findings that need validation
			if d.filter[key] {
				result = append(result, f)
			}
		}
	}
	return result, nil
}

func (d *replayDetector) Name() string {
	return "replay"
}
