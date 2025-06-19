package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/MacAttak/pi-scanner/pkg/detection"
	"github.com/MacAttak/pi-scanner/pkg/discovery"
	"github.com/MacAttak/pi-scanner/pkg/processing"
	"github.com/MacAttak/pi-scanner/pkg/repository"
)

// ScanResult represents the results of scanning a repository
type ScanResult struct {
	Repository   *repository.RepositoryInfo `json:"repository"`
	ScanStarted  time.Time                  `json:"scan_started"`
	ScanFinished time.Time                  `json:"scan_finished"`
	Duration     time.Duration              `json:"duration"`
	FilesScanned int                        `json:"files_scanned"`
	Findings     []detection.Finding        `json:"findings"`
	Stats        ScanStats                  `json:"stats"`
	Error        string                     `json:"error,omitempty"`
}

// ScanStats provides statistics about the scan
type ScanStats struct {
	TotalFiles    int                     `json:"total_files"`
	ScannedFiles  int                     `json:"scanned_files"`
	SkippedFiles  int                     `json:"skipped_files"`
	TotalSize     int64                   `json:"total_size"`
	FindingsByType map[string]int         `json:"findings_by_type"`
	FindingsByRisk map[string]int         `json:"findings_by_risk"`
	ProcessingTime time.Duration          `json:"processing_time"`
}

// runScan performs the actual scanning logic
func runScan(ctx context.Context, repoURL, outputFile string, verbose bool) error {
	result := &ScanResult{
		ScanStarted: time.Now(),
		Stats: ScanStats{
			FindingsByType: make(map[string]int),
			FindingsByRisk: make(map[string]int),
		},
	}
	
	if verbose {
		fmt.Printf("🔍 Starting PI scan of repository: %s\n", repoURL)
	}

	// Step 1: Set up repository manager
	repoConfig := repository.DefaultGitHubConfig()
	repoManager := repository.NewRepositoryManager(repoConfig)
	
	// Check authentication
	if verbose {
		fmt.Printf("🔐 Checking GitHub authentication...\n")
	}
	
	err := repoManager.CheckAuthentication(ctx)
	if err != nil {
		result.Error = fmt.Sprintf("Authentication failed: %v", err)
		return saveResult(result, outputFile)
	}
	
	if verbose {
		fmt.Printf("✅ GitHub authentication successful\n")
	}

	// Step 2: Clone repository
	if verbose {
		fmt.Printf("📥 Cloning repository...\n")
	}
	
	repoInfo, err := repoManager.CloneAndTrack(ctx, repoURL)
	if err != nil {
		result.Error = fmt.Sprintf("Failed to clone repository: %v", err)
		return saveResult(result, outputFile)
	}
	
	result.Repository = repoInfo
	
	// Ensure cleanup happens
	defer func() {
		if verbose {
			fmt.Printf("🧹 Cleaning up cloned repository...\n")
		}
		repoManager.CleanupAll()
	}()
	
	if verbose {
		fmt.Printf("✅ Repository cloned to: %s\n", repoInfo.LocalPath)
		fmt.Printf("📊 Repository info: %d files, %d bytes\n", repoInfo.FileCount, repoInfo.Size)
	}

	// Step 3: Set up detectors
	if verbose {
		fmt.Printf("🔧 Setting up detection pipeline...\n")
	}
	
	var detectors []detection.Detector
	
	// Add pattern detector
	patternDetector := detection.NewDetector()
	detectors = append(detectors, patternDetector)
	
	// Add Gitleaks detector
	gitleaksConfigPath := filepath.Join("configs", "gitleaks.toml")
	if _, err := os.Stat(gitleaksConfigPath); err == nil {
		gitleaksDetector, err := detection.NewGitleaksDetector(gitleaksConfigPath)
		if err != nil {
			if verbose {
				fmt.Printf("⚠️  Gitleaks detector setup failed: %v\n", err)
			}
		} else {
			detectors = append(detectors, gitleaksDetector)
			if verbose {
				fmt.Printf("✅ Gitleaks detector loaded\n")
			}
		}
	} else {
		if verbose {
			fmt.Printf("⚠️  Gitleaks config not found, skipping\n")
		}
	}
	
	if verbose {
		fmt.Printf("✅ %d detectors configured\n", len(detectors))
	}

	// Step 4: Discover files
	if verbose {
		fmt.Printf("🔍 Discovering files to scan...\n")
	}
	
	discoveryConfig := discovery.DefaultConfig()
	fileDiscovery := discovery.NewFileDiscovery(discoveryConfig)
	
	files, err := fileDiscovery.DiscoverFiles(ctx, repoInfo.LocalPath)
	if err != nil {
		result.Error = fmt.Sprintf("File discovery failed: %v", err)
		return saveResult(result, outputFile)
	}
	
	result.Stats.TotalFiles = len(files)
	
	if verbose {
		fmt.Printf("✅ Discovered %d files\n", len(files))
	}

	// Step 5: Set up file processor
	processorConfig := processing.DefaultProcessorConfig()
	processorConfig.NumWorkers = 4 // Reasonable for testing
	
	fileProcessor := processing.NewFileProcessor(processorConfig, detectors)
	
	// Step 6: Create processing jobs
	var jobs []processing.FileJob
	for _, file := range files {
		if file.IsBinary {
			result.Stats.SkippedFiles++
			continue
		}
		
		// Read file content
		content, err := os.ReadFile(file.Path)
		if err != nil {
			if verbose {
				fmt.Printf("⚠️  Could not read file %s: %v\n", file.Path, err)
			}
			result.Stats.SkippedFiles++
			continue
		}
		
		result.Stats.TotalSize += int64(len(content))
		
		job := processing.FileJob{
			FilePath: file.Path,
			Content:  content,
			FileInfo: file,
		}
		
		jobs = append(jobs, job)
	}
	
	result.Stats.ScannedFiles = len(jobs)
	
	if verbose {
		fmt.Printf("📋 Prepared %d files for scanning (%d skipped)\n", len(jobs), result.Stats.SkippedFiles)
	}

	// Step 7: Process files
	if verbose {
		fmt.Printf("🚀 Starting file processing with %d workers...\n", processorConfig.NumWorkers)
	}
	
	processingStart := time.Now()
	
	batchProcessor := processing.NewBatchProcessor(fileProcessor, 50)
	results, err := batchProcessor.ProcessFiles(ctx, jobs)
	if err != nil {
		result.Error = fmt.Sprintf("File processing failed: %v", err)
		return saveResult(result, outputFile)
	}
	
	result.Stats.ProcessingTime = time.Since(processingStart)
	
	if verbose {
		fmt.Printf("✅ Processing completed in %v\n", result.Stats.ProcessingTime)
	}

	// Step 8: Collect and analyze findings
	if verbose {
		fmt.Printf("📊 Analyzing findings...\n")
	}
	
	var allFindings []detection.Finding
	for _, procResult := range results {
		if procResult.Error != nil {
			if verbose {
				fmt.Printf("⚠️  Error processing %s: %v\n", procResult.FilePath, procResult.Error)
			}
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
	
	result.Findings = allFindings
	result.FilesScanned = len(results)
	result.ScanFinished = time.Now()
	result.Duration = result.ScanFinished.Sub(result.ScanStarted)
	
	if verbose {
		fmt.Printf("🎯 Scan Summary:\n")
		fmt.Printf("   • Duration: %v\n", result.Duration)
		fmt.Printf("   • Files scanned: %d\n", result.FilesScanned)
		fmt.Printf("   • Total findings: %d\n", len(allFindings))
		
		if len(result.Stats.FindingsByType) > 0 {
			fmt.Printf("   • Findings by type:\n")
			for piType, count := range result.Stats.FindingsByType {
				fmt.Printf("     - %s: %d\n", piType, count)
			}
		}
		
		if len(result.Stats.FindingsByRisk) > 0 {
			fmt.Printf("   • Findings by risk:\n")
			for risk, count := range result.Stats.FindingsByRisk {
				fmt.Printf("     - %s: %d\n", risk, count)
			}
		}
	}

	// Step 9: Save results
	return saveResult(result, outputFile)
}

// saveResult saves the scan result to a JSON file
func saveResult(result *ScanResult, outputFile string) error {
	// Create output directory if needed
	dir := filepath.Dir(outputFile)
	if dir != "." {
		err := os.MkdirAll(dir, 0755)
		if err != nil {
			return fmt.Errorf("failed to create output directory: %w", err)
		}
	}
	
	// Marshal result to JSON
	jsonData, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal results: %w", err)
	}
	
	// Write to file
	err = os.WriteFile(outputFile, jsonData, 0644)
	if err != nil {
		return fmt.Errorf("failed to write results file: %w", err)
	}
	
	fmt.Printf("✅ Results saved to: %s\n", outputFile)
	return nil
}