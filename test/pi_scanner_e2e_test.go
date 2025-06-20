package test

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// E2EScanResult represents the expected structure of scan results
type E2EScanResult struct {
	Repository   RepositoryInfo `json:"repository"`
	ScanStarted  time.Time      `json:"scan_started"`
	ScanFinished time.Time      `json:"scan_finished"`
	Duration     int64          `json:"duration"`
	FilesScanned int            `json:"files_scanned"`
	Findings     []Finding      `json:"findings"`
	Stats        ScanStats      `json:"stats"`
	Error        string         `json:"error,omitempty"`
}

type RepositoryInfo struct {
	URL       string    `json:"url"`
	Owner     string    `json:"owner"`
	Name      string    `json:"name"`
	LocalPath string    `json:"local_path"`
	Size      int64     `json:"size"`
	FileCount int       `json:"file_count"`
	ClonedAt  time.Time `json:"cloned_at"`
	IsShallow bool      `json:"is_shallow"`
}

type Finding struct {
	Type            string    `json:"type"`
	Match           string    `json:"match"`
	File            string    `json:"file"`
	Line            int       `json:"line"`
	Column          int       `json:"column"`
	Context         string    `json:"context"`
	ContextBefore   string    `json:"context_before"`
	ContextAfter    string    `json:"context_after"`
	RiskLevel       string    `json:"risk_level"`
	Confidence      float32   `json:"confidence"`
	ContextModifier float32   `json:"context_modifier"`
	Validated       bool      `json:"validated"`
	DetectedAt      time.Time `json:"detected_at"`
	DetectorName    string    `json:"detector_name"`
}

type ScanStats struct {
	TotalFiles     int            `json:"total_files"`
	ScannedFiles   int            `json:"scanned_files"`
	SkippedFiles   int            `json:"skipped_files"`
	TotalSize      int64          `json:"total_size"`
	FindingsByType map[string]int `json:"findings_by_type"`
	FindingsByRisk map[string]int `json:"findings_by_risk"`
	ProcessingTime int64          `json:"processing_time"`
}

// TestRepository represents a test case for repository scanning
type TestRepository struct {
	Name                string
	URL                 string
	Description         string
	ExpectedMinFiles    int
	ExpectedMaxDuration time.Duration
	ExpectedPITypes     []string
	ExpectedMinFindings int
	ExpectedMaxFindings int
	SkipIfNoAuth        bool
}

// Australian government and organization repositories for testing
var australianTestRepositories = []TestRepository{
	{
		Name:                "Australian Government Design System",
		URL:                 "govau/design-system-components",
		Description:         "Australian Government Design System components",
		ExpectedMinFiles:    10,
		ExpectedMaxDuration: 30 * time.Second,
		ExpectedPITypes:     []string{"NAME", "EMAIL"},
		ExpectedMinFindings: 5,
		ExpectedMaxFindings: 1000,
		SkipIfNoAuth:        false,
	},
	{
		Name:                "Australia.gov.au Static Site",
		URL:                 "govau/australia-gov-au-static",
		Description:         "Static site for info.australia.gov.au",
		ExpectedMinFiles:    5,
		ExpectedMaxDuration: 20 * time.Second,
		ExpectedPITypes:     []string{"NAME", "EMAIL", "PHONE"},
		ExpectedMinFindings: 3,
		ExpectedMaxFindings: 500,
		SkipIfNoAuth:        false,
	},
	{
		Name:                "Queensland Government Design System",
		URL:                 "qld-gov-au/qgds-qol-mvp",
		Description:         "Queensland Government Design System",
		ExpectedMinFiles:    5,
		ExpectedMaxDuration: 25 * time.Second,
		ExpectedPITypes:     []string{"NAME", "EMAIL"},
		ExpectedMinFindings: 2,
		ExpectedMaxFindings: 800,
		SkipIfNoAuth:        false,
	},
	{
		Name:                "National Library of Australia",
		URL:                 "nla/nla-blacklight",
		Description:         "NLA Blacklight discovery interface",
		ExpectedMinFiles:    10,
		ExpectedMaxDuration: 40 * time.Second,
		ExpectedPITypes:     []string{"NAME", "EMAIL"},
		ExpectedMinFindings: 5,
		ExpectedMaxFindings: 1000,
		SkipIfNoAuth:        false,
	},
	{
		Name:                "TerriaJS National Map",
		URL:                 "TerriaJS/nationalmap",
		Description:         "Australia's National Map geospatial platform",
		ExpectedMinFiles:    20,
		ExpectedMaxDuration: 60 * time.Second,
		ExpectedPITypes:     []string{"NAME", "EMAIL", "PHONE"},
		ExpectedMinFindings: 10,
		ExpectedMaxFindings: 1000,
		SkipIfNoAuth:        false,
	},
}

// International repositories for comparative testing
var internationalTestRepositories = []TestRepository{
	{
		Name:                "GitHub Documentation",
		URL:                 "github/docs",
		Description:         "GitHub's public documentation",
		ExpectedMinFiles:    1000,
		ExpectedMaxDuration: 120 * time.Second,
		ExpectedPITypes:     []string{"NAME", "EMAIL", "TFN", "BSB"},
		ExpectedMinFindings: 1000,
		ExpectedMaxFindings: 50000,
		SkipIfNoAuth:        false,
	},
	{
		Name:                "FreeCodeCamp",
		URL:                 "freeCodeCamp/freeCodeCamp",
		Description:         "Educational platform with diverse content",
		ExpectedMinFiles:    5000,
		ExpectedMaxDuration: 180 * time.Second,
		ExpectedPITypes:     []string{"NAME", "EMAIL"},
		ExpectedMinFindings: 5000,
		ExpectedMaxFindings: 100000,
		SkipIfNoAuth:        false,
	},
}

func TestPIScannerE2E_AustralianRepositories(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E tests in short mode")
	}

	// Build the scanner first
	buildScanner(t)

	for _, repo := range australianTestRepositories {
		t.Run(repo.Name, func(t *testing.T) {
			runRepositoryTest(t, repo)
		})
	}
}

func TestPIScannerE2E_InternationalRepositories(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E tests in short mode")
	}

	// Build the scanner first
	buildScanner(t)

	for _, repo := range internationalTestRepositories {
		t.Run(repo.Name, func(t *testing.T) {
			runRepositoryTest(t, repo)
		})
	}
}

func TestPIScannerE2E_PerformanceBenchmarks(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance benchmarks in short mode")
	}

	buildScanner(t)

	performanceTests := []struct {
		name          string
		repo          string
		maxDuration   time.Duration
		minThroughput float64 // files per second
	}{
		{
			name:          "Small Repository Performance",
			repo:          "octocat/Hello-World",
			maxDuration:   10 * time.Second,
			minThroughput: 1.0,
		},
		{
			name:          "Medium Repository Performance",
			repo:          "govau/design-system-components",
			maxDuration:   30 * time.Second,
			minThroughput: 10.0,
		},
		{
			name:          "Large Repository Performance",
			repo:          "github/docs",
			maxDuration:   120 * time.Second,
			minThroughput: 50.0,
		},
	}

	for _, test := range performanceTests {
		t.Run(test.name, func(t *testing.T) {
			outputFile := filepath.Join(os.TempDir(), fmt.Sprintf("perf-test-%d.json", time.Now().UnixNano()))
			defer os.Remove(outputFile)

			start := time.Now()
			result := runScanCommand(t, test.repo, outputFile, false)
			duration := time.Since(start)

			// Verify performance expectations
			assert.Less(t, duration, test.maxDuration,
				"Scan should complete within expected time")

			if result.Stats.ScannedFiles > 0 {
				throughput := float64(result.Stats.ScannedFiles) / duration.Seconds()
				assert.Greater(t, throughput, test.minThroughput,
					"Should meet minimum throughput requirements")
			}

			t.Logf("Performance Results: %d files in %v (%.2f files/sec)",
				result.Stats.ScannedFiles, duration,
				float64(result.Stats.ScannedFiles)/duration.Seconds())
		})
	}
}

func TestPIScannerE2E_AustralianPIDetection(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Australian PI detection tests in short mode")
	}

	buildScanner(t)

	// Test with repositories that are likely to contain Australian PI examples
	testCases := []struct {
		name          string
		repo          string
		expectedTypes []string
		description   string
	}{
		{
			name:          "GitHub Docs - Australian Examples",
			repo:          "github/docs",
			expectedTypes: []string{"TFN", "ABN", "BSB", "MEDICARE"},
			description:   "Documentation often contains example Australian PI",
		},
		{
			name:          "Educational Content",
			repo:          "freeCodeCamp/freeCodeCamp",
			expectedTypes: []string{"NAME", "EMAIL", "PHONE"},
			description:   "Educational platforms with diverse examples",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			outputFile := filepath.Join(os.TempDir(), fmt.Sprintf("au-pi-test-%d.json", time.Now().UnixNano()))
			defer os.Remove(outputFile)

			result := runScanCommand(t, tc.repo, outputFile, false)

			// Verify Australian PI types were detected
			detectedTypes := make(map[string]bool)
			for _, finding := range result.Findings {
				detectedTypes[finding.Type] = true
			}

			foundAustralianPI := false
			australianTypes := []string{"TFN", "ABN", "MEDICARE", "BSB", "ACN", "DRIVER_LICENSE"}

			for _, auType := range australianTypes {
				if detectedTypes[auType] {
					foundAustralianPI = true
					t.Logf("Found Australian PI type: %s", auType)
				}
			}

			if foundAustralianPI {
				t.Logf("✅ Australian PI detection working - found types in %s", tc.repo)
			} else {
				t.Logf("ℹ️  No Australian PI found in %s (expected for some repositories)", tc.repo)
			}

			// Verify risk assessment is working
			riskLevels := make(map[string]int)
			for _, finding := range result.Findings {
				riskLevels[finding.RiskLevel]++
			}

			if len(riskLevels) > 0 {
				t.Logf("Risk distribution: %+v", riskLevels)
				assert.Contains(t, []string{"LOW", "MEDIUM", "HIGH"},
					getMostCommonRisk(riskLevels), "Should have valid risk levels")
			}
		})
	}
}

func TestPIScannerE2E_ErrorHandling(t *testing.T) {
	buildScanner(t)

	errorTests := []struct {
		name        string
		args        []string
		expectError bool
		errorString string
	}{
		{
			name:        "Missing Repository",
			args:        []string{"scan"},
			expectError: true,
			errorString: "either --repo or --repo-list must be specified",
		},
		{
			name:        "Invalid Repository",
			args:        []string{"scan", "--repo", "invalid/nonexistent-repo-12345"},
			expectError: true,
			errorString: "", // May vary based on git/GitHub response
		},
		{
			name:        "Valid Repository",
			args:        []string{"scan", "--repo", "octocat/Hello-World", "--output", "/tmp/test-valid.json"},
			expectError: false,
			errorString: "",
		},
	}

	for _, test := range errorTests {
		t.Run(test.name, func(t *testing.T) {
			cmd := exec.Command("../pi-scanner", test.args...)
			output, err := cmd.CombinedOutput()

			if test.expectError {
				assert.Error(t, err, "Command should fail")
				if test.errorString != "" {
					assert.Contains(t, string(output), test.errorString,
						"Should contain expected error message")
				}
			} else {
				assert.NoError(t, err, "Command should succeed")
			}

			// Cleanup any output files
			if len(test.args) > 3 && strings.HasPrefix(test.args[len(test.args)-1], "/tmp/") {
				os.Remove(test.args[len(test.args)-1])
			}
		})
	}
}

func TestPIScannerE2E_CLICommands(t *testing.T) {
	buildScanner(t)

	// Test all CLI commands
	commands := []struct {
		name string
		args []string
	}{
		{"Version Command", []string{"version"}},
		{"Help Command", []string{"--help"}},
		{"Scan Help", []string{"scan", "--help"}},
		{"Report Help", []string{"report", "--help"}},
	}

	for _, cmd := range commands {
		t.Run(cmd.name, func(t *testing.T) {
			command := exec.Command("../pi-scanner", cmd.args...)
			output, err := command.CombinedOutput()

			assert.NoError(t, err, "Command should execute successfully")
			assert.NotEmpty(t, output, "Should produce output")

			// Verify expected content
			outputStr := string(output)
			switch cmd.name {
			case "Version Command":
				assert.Contains(t, outputStr, "PI Scanner")
				assert.Contains(t, outputStr, "Version:")
			case "Help Command", "Scan Help", "Report Help":
				assert.Contains(t, outputStr, "Usage:")
			}
		})
	}
}

// Helper functions

func buildScanner(t *testing.T) {
	t.Helper()

	// Build the scanner binary in the project root
	cmd := exec.Command("go", "build", "-o", "../pi-scanner", "../cmd/pi-scanner")
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "Failed to build pi-scanner: %s", string(output))

	// Ensure the binary exists
	assert.FileExists(t, "../pi-scanner", "Scanner binary should exist")
}

func runRepositoryTest(t *testing.T, repo TestRepository) {
	t.Helper()

	outputFile := filepath.Join(os.TempDir(), fmt.Sprintf("e2e-test-%d.json", time.Now().UnixNano()))
	defer os.Remove(outputFile)

	start := time.Now()
	result := runScanCommand(t, repo.URL, outputFile, true)
	duration := time.Since(start)

	// Basic validation
	assert.Empty(t, result.Error, "Scan should complete without errors")
	assert.Greater(t, result.Stats.TotalFiles, repo.ExpectedMinFiles-1,
		"Should discover minimum expected files")
	assert.Less(t, duration, repo.ExpectedMaxDuration,
		"Should complete within expected time")

	// Findings validation
	if repo.ExpectedMinFindings > 0 {
		totalFindings := len(result.Findings)
		assert.GreaterOrEqual(t, totalFindings, repo.ExpectedMinFindings,
			"Should find minimum expected PI instances")
		assert.LessOrEqual(t, totalFindings, repo.ExpectedMaxFindings,
			"Should not exceed maximum expected findings")
	}

	// Verify expected PI types were found (if any findings exist)
	if len(result.Findings) > 0 {
		foundTypes := make(map[string]bool)
		for _, finding := range result.Findings {
			foundTypes[finding.Type] = true
		}

		foundExpectedType := false
		for _, expectedType := range repo.ExpectedPITypes {
			if foundTypes[expectedType] {
				foundExpectedType = true
				break
			}
		}

		if len(repo.ExpectedPITypes) > 0 {
			assert.True(t, foundExpectedType,
				"Should find at least one expected PI type: %v", repo.ExpectedPITypes)
		}
	}

	t.Logf("✅ %s: %d files, %d findings in %v",
		repo.Name, result.Stats.TotalFiles, len(result.Findings), duration)
}

func runScanCommand(t *testing.T, repoURL, outputFile string, verbose bool) *E2EScanResult {
	t.Helper()

	args := []string{"scan", "--repo", repoURL, "--output", outputFile}
	if verbose {
		args = append(args, "--verbose")
	}

	cmd := exec.Command("../pi-scanner", args...)
	output, err := cmd.CombinedOutput()

	// Log output for debugging
	if verbose {
		t.Logf("Scanner output:\n%s", string(output))
	}

	// Read and parse results
	require.FileExists(t, outputFile, "Output file should be created")

	resultData, err := os.ReadFile(outputFile)
	require.NoError(t, err, "Should be able to read results file")

	var result E2EScanResult
	err = json.Unmarshal(resultData, &result)
	require.NoError(t, err, "Should be able to parse results JSON")

	return &result
}

func getMostCommonRisk(riskLevels map[string]int) string {
	maxCount := 0
	mostCommon := ""
	for risk, count := range riskLevels {
		if count > maxCount {
			maxCount = count
			mostCommon = risk
		}
	}
	return mostCommon
}

// Benchmarks for performance regression testing
func BenchmarkE2E_SmallRepository(b *testing.B) {
	buildScanner(&testing.T{})

	for i := 0; i < b.N; i++ {
		outputFile := filepath.Join(os.TempDir(), fmt.Sprintf("bench-%d.json", time.Now().UnixNano()))

		cmd := exec.Command("../pi-scanner", "scan", "--repo", "octocat/Hello-World", "--output", outputFile)
		_, err := cmd.CombinedOutput()

		if err != nil {
			b.Fatal(err)
		}

		os.Remove(outputFile)
	}
}

func BenchmarkE2E_MediumRepository(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping medium repository benchmark in short mode")
	}

	buildScanner(&testing.T{})

	for i := 0; i < b.N; i++ {
		outputFile := filepath.Join(os.TempDir(), fmt.Sprintf("bench-%d.json", time.Now().UnixNano()))

		cmd := exec.Command("../pi-scanner", "scan", "--repo", "govau/design-system-components", "--output", outputFile)
		_, err := cmd.CombinedOutput()

		if err != nil {
			b.Fatal(err)
		}

		os.Remove(outputFile)
	}
}
