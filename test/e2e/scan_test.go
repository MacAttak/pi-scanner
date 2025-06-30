package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/MacAttak/pi-scanner/pkg/detection"
	"github.com/MacAttak/pi-scanner/pkg/discovery"
	"github.com/MacAttak/pi-scanner/pkg/llm"
	"github.com/MacAttak/pi-scanner/pkg/output"
	"github.com/MacAttak/pi-scanner/pkg/processing"
	"github.com/MacAttak/pi-scanner/pkg/report"
	"github.com/MacAttak/pi-scanner/pkg/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestScanPublicRepo tests scanning a public repository with test data
func TestScanPublicRepo(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping e2e test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	tempDir := t.TempDir()

	// Test with a small public repo that contains test PI data
	repoURL := "https://github.com/MacAttak/pi-scanner-test-data"

	// Clone repository
	info, cleanup, err := repository.CloneTemporary(ctx, repoURL)
	require.NoError(t, err)
	defer cleanup()

	t.Logf("Cloned repository to: %s", info.LocalPath)

	// Configure detection
	detectorConfig := &detection.Config{
		EnableContextValidation: true,
		ContextLines:            3,
		MinConfidence:           0.7,
		EnableLLMValidation:     false, // Disable for tests
	}

	// Create detector
	detector := detection.NewDetectorWithConfig(detectorConfig)

	// Create file discoverer
	discoverer := discovery.NewFileDiscoverer(discovery.Config{
		SkipBinary:  true,
		SkipDirs:    []string{".git", "node_modules", "vendor"},
		MaxFileSize: 1024 * 1024, // 1MB
	})

	// Create output manager with secure configuration
	outputConfig := &output.Config{
		MaskingLevel:    output.MaskingLevelPartial,
		EnableAudit:     true,
		ValidateOutputs: true,
	}
	outputManager := output.NewManager(outputConfig)

	// Create file processor
	processor := processing.NewFileProcessor(detector, processing.ProcessorConfig{
		MaxWorkers:      4,
		MaxMemoryMB:     100,
		StreamThreshold: 500 * 1024, // 500KB
	})

	// Discover files
	files, err := discoverer.DiscoverFiles(ctx, info.LocalPath)
	require.NoError(t, err)
	assert.NotEmpty(t, files, "Should find files to scan")

	t.Logf("Found %d files to scan", len(files))

	// Process files
	var findings []detection.Finding
	for _, file := range files {
		if ctx.Err() != nil {
			break
		}

		result, err := processor.ProcessFile(ctx, file)
		if err != nil {
			t.Logf("Error processing %s: %v", file.Path, err)
			continue
		}

		if len(result.Findings) > 0 {
			// Mask findings before adding to results
			for i := range result.Findings {
				masked, err := outputManager.MaskFinding(&result.Findings[i])
				require.NoError(t, err)
				findings = append(findings, *masked)
			}
		}
	}

	// Verify we found some test PI
	assert.NotEmpty(t, findings, "Should find test PI patterns")

	// Group findings by type
	findingsByType := make(map[string]int)
	for _, f := range findings {
		findingsByType[string(f.Type)]++
	}

	t.Logf("Found %d PI instances across %d types", len(findings), len(findingsByType))
	for piType, count := range findingsByType {
		t.Logf("  %s: %d", piType, count)
	}

	// Test report generation
	t.Run("report_generation", func(t *testing.T) {
		testReportGeneration(t, findings, info, tempDir, outputManager)
	})

	// Test output security
	t.Run("output_security", func(t *testing.T) {
		testOutputSecurity(t, findings, tempDir)
	})
}

// TestResourceManagement tests resource management during scans
func TestResourceManagement(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping e2e test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// Create processor with resource limits
	detector := detection.NewDetector()
	processor := processing.NewFileProcessor(detector, processing.ProcessorConfig{
		MaxWorkers:      2,          // Limited workers
		MaxMemoryMB:     50,         // Limited memory
		StreamThreshold: 100 * 1024, // Low threshold to test streaming
	})

	// Create a test file with repetitive content
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "large_test.txt")

	// Write test content with PI patterns
	content := strings.Repeat("Test TFN: 123456782\nTest ABN: 51824753556\n", 10000)
	err := os.WriteFile(testFile, []byte(content), 0644)
	require.NoError(t, err)

	// Process the file
	fileInfo := discovery.FileInfo{
		Path: testFile,
		Size: int64(len(content)),
	}

	result, err := processor.ProcessFile(ctx, fileInfo)
	require.NoError(t, err)

	// Should have found patterns despite resource limits
	assert.NotEmpty(t, result.Findings)

	// Verify memory wasn't exceeded (this is a simple check)
	// In production, you'd use proper memory profiling
	assert.True(t, result.Stats.MemoryUsage < 100*1024*1024, "Memory usage should stay under limit")
}

// Helper functions

func testReportGeneration(t *testing.T, findings []detection.Finding, repoInfo *repository.RepositoryInfo, outputDir string, outputManager *output.Manager) {
	formats := []string{"json", "csv", "html"}

	for _, format := range formats {
		t.Run(format, func(t *testing.T) {
			reportPath := filepath.Join(outputDir, fmt.Sprintf("report.%s", format))

			// Create report data
			reportData := &report.Data{
				Repository: repoInfo,
				Findings:   findings,
				Summary: report.Summary{
					TotalFindings:  len(findings),
					FilesScanned:   100, // Mock value
					FindingsByType: make(map[string]int),
					FindingsByRisk: make(map[string]int),
					ScanDuration:   time.Minute,
				},
			}

			// Count findings
			for _, f := range findings {
				reportData.Summary.FindingsByType[string(f.Type)]++
				reportData.Summary.FindingsByRisk[string(f.RiskLevel)]++
			}

			// Generate report based on format
			var content []byte
			var err error

			switch format {
			case "json":
				gen := report.NewJSONGenerator(outputManager)
				content, err = gen.Generate(reportData)
			case "csv":
				gen := report.NewCSVGenerator(outputManager)
				content, err = gen.Generate(reportData)
			case "html":
				gen := report.NewHTMLGenerator(outputManager)
				content, err = gen.Generate(reportData)
			}

			require.NoError(t, err)
			require.NotEmpty(t, content)

			// Write report
			err = os.WriteFile(reportPath, content, 0644)
			require.NoError(t, err)

			// Verify report content
			switch format {
			case "json":
				var jsonData map[string]interface{}
				err = json.Unmarshal(content, &jsonData)
				assert.NoError(t, err)
				assert.Contains(t, jsonData, "findings")
				assert.Contains(t, jsonData, "summary")

			case "html":
				htmlStr := string(content)
				assert.Contains(t, htmlStr, "<html")
				assert.Contains(t, htmlStr, "PI Scanner Report")
				// Verify masking
				assert.NotContains(t, htmlStr, "123456782") // Full TFN
				assert.Contains(t, htmlStr, "123****82")    // Masked TFN

			case "csv":
				csvStr := string(content)
				assert.Contains(t, csvStr, "Type,Risk,File")
			}
		})
	}
}

func testOutputSecurity(t *testing.T, findings []detection.Finding, outputDir string) {
	// Known test PI values that should be masked
	testPIValues := map[string]string{
		"123456782":   "TFN",
		"51824753556": "ABN",
		"2123456701":  "Medicare",
		"062001":      "BSB",
		"004028077":   "ACN",
	}

	// Check all files in output directory
	err := filepath.Walk(outputDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		contentStr := string(content)

		// Check for unmasked PI values
		for piValue, piType := range testPIValues {
			if strings.Contains(contentStr, piValue) {
				// Check if it's properly masked
				validMaskings := []string{
					piValue[:3] + "***",   // Simple mask
					piValue[:3] + "****",  // Longer mask
					piValue[:3] + "*****", // Even longer
				}

				isMasked := false
				for _, masked := range validMaskings {
					if strings.Contains(contentStr, masked) {
						isMasked = true
						break
					}
				}

				// Special case: full value might be in a context showing it's masked
				if strings.Contains(contentStr, fmt.Sprintf(`"match":"%s***`, piValue[:3])) {
					isMasked = true
				}

				if !isMasked {
					t.Errorf("Found unmasked %s value '%s' in file %s", piType, piValue, path)
				}
			}
		}

		return nil
	})

	require.NoError(t, err)
}
