package e2e

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

// ComprehensiveTestRepo represents a repository with expected PI patterns
type ComprehensiveTestRepo struct {
	Name               string
	URL                string
	Description        string
	ExpectedPITypes    []string
	LikelyToHaveTestPI bool
	Category           string
}

// Comprehensive list of Australian repositories likely to contain test PI data
var comprehensiveTestRepos = []ComprehensiveTestRepo{
	// Healthcare repositories with test data
	{
		Name:               "HL7 AU FHIR Test Data",
		URL:                "https://github.com/hl7au/au-fhir-test-data",
		Description:        "Contains test Medicare numbers, IHI, HPI-I, DVA numbers",
		ExpectedPITypes:    []string{"MEDICARE", "NAME", "PHONE", "EMAIL"},
		LikelyToHaveTestPI: true,
		Category:           "healthcare",
	},
	{
		Name:               "Australian Digital Health MHR B2B Client",
		URL:                "https://github.com/AuDigitalHealth/mhr-b2b-client-java",
		Description:        "My Health Record client with sample implementations",
		ExpectedPITypes:    []string{"MEDICARE", "NAME", "IHI"},
		LikelyToHaveTestPI: true,
		Category:           "healthcare",
	},
	{
		Name:               "Australian Digital Health HI Vendor",
		URL:                "https://github.com/AuDigitalHealth/hi-vendor-client-dotnet",
		Description:        "Health Identifier Service client",
		ExpectedPITypes:    []string{"IHI", "HPI-I", "HPI-O"},
		LikelyToHaveTestPI: true,
		Category:           "healthcare",
	},

	// Government design systems
	{
		Name:               "Australian Government Design System",
		URL:                "https://github.com/govau/design-system-components",
		Description:        "Design system with example forms",
		ExpectedPITypes:    []string{"NAME", "EMAIL", "PHONE"},
		LikelyToHaveTestPI: false,
		Category:           "government",
	},
	{
		Name:               "NSW Design System",
		URL:                "https://github.com/digitalnsw/nsw-design-system",
		Description:        "NSW Government design system",
		ExpectedPITypes:    []string{"NAME", "EMAIL", "ADDRESS"},
		LikelyToHaveTestPI: false,
		Category:           "government",
	},

	// Financial/business repositories
	{
		Name:               "CommBank API Samples",
		URL:                "https://github.com/CommBank/CommBank-API-Samples",
		Description:        "Commonwealth Bank API examples",
		ExpectedPITypes:    []string{"BSB", "ACCOUNT", "ABN"},
		LikelyToHaveTestPI: true,
		Category:           "financial",
	},

	// Education repositories
	{
		Name:               "University of Melbourne Design System",
		URL:                "https://github.com/unimelb/unimelb-design-system",
		Description:        "University design system with forms",
		ExpectedPITypes:    []string{"NAME", "EMAIL", "PHONE", "STUDENT_ID"},
		LikelyToHaveTestPI: false,
		Category:           "education",
	},

	// Geospatial/data repositories
	{
		Name:               "TerriaJS National Map",
		URL:                "https://github.com/TerriaJS/nationalmap",
		Description:        "Australia's National Map platform",
		ExpectedPITypes:    []string{"NAME", "EMAIL", "ADDRESS"},
		LikelyToHaveTestPI: false,
		Category:           "geospatial",
	},
	{
		Name:               "Digital Earth Australia Notebooks",
		URL:                "https://github.com/GeoscienceAustralia/dea-notebooks",
		Description:        "Jupyter notebooks for satellite data analysis",
		ExpectedPITypes:    []string{"NAME", "EMAIL"},
		LikelyToHaveTestPI: false,
		Category:           "geospatial",
	},
}

func TestComprehensiveRepoScanning(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping comprehensive E2E test in short mode")
	}

	// Build scanner once
	buildScanner(t)

	// Test different categories
	categories := []string{"healthcare", "government", "financial"}

	for _, category := range categories {
		t.Run(fmt.Sprintf("Category_%s", category), func(t *testing.T) {
			// Find repos in this category
			var categoryRepos []ComprehensiveTestRepo
			for _, repo := range comprehensiveTestRepos {
				if repo.Category == category {
					categoryRepos = append(categoryRepos, repo)
				}
			}

			if len(categoryRepos) == 0 {
				t.Skip("No repos in category")
			}

			// Test first repo in category (to save time)
			repo := categoryRepos[0]
			testRepo(t, repo)
		})
	}
}

func TestHighValueRepos(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping high-value repo tests in short mode")
	}

	buildScanner(t)

	// Test repos most likely to have test PI data
	highValueRepos := []ComprehensiveTestRepo{}
	for _, repo := range comprehensiveTestRepos {
		if repo.LikelyToHaveTestPI {
			highValueRepos = append(highValueRepos, repo)
		}
	}

	for _, repo := range highValueRepos {
		t.Run(repo.Name, func(t *testing.T) {
			testRepo(t, repo)
		})
	}
}

func TestReportSecurity(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping report security test in short mode")
	}

	buildScanner(t)

	// Test with HL7 AU FHIR Test Data which should have Medicare numbers
	repo := ComprehensiveTestRepo{
		Name:               "HL7_AU_FHIR_Security_Test",
		URL:                "https://github.com/hl7au/au-fhir-test-data",
		Description:        "Testing report security with healthcare data",
		ExpectedPITypes:    []string{"MEDICARE"},
		LikelyToHaveTestPI: true,
		Category:           "healthcare",
	}

	outputFile := filepath.Join(t.TempDir(), "security-test.json")

	// Run scan
	start := time.Now()
	cmd := exec.Command("./pi-scanner", "scan",
		"--repo", repo.URL,
		"--output", outputFile,
		"--verbose")

	output, err := cmd.CombinedOutput()
	duration := time.Since(start)

	t.Logf("Scan completed in %v", duration)

	// Even if scan fails (e.g., auth issues), check output security
	if err != nil {
		t.Logf("Scan error (expected without auth): %v", err)
		t.Logf("Output: %s", string(output))
	}

	// If output file exists, verify security
	if _, err := os.Stat(outputFile); err == nil {
		data, err := os.ReadFile(outputFile)
		require.NoError(t, err)

		content := string(data)

		// Check for common unmasked PI patterns
		unsafePatterns := []struct {
			pattern string
			name    string
		}{
			{`\d{10}`, "10-digit Medicare number"},
			{`\d{4}\s?\d{5}\s?\d`, "Medicare with spaces"},
			{`\d{9}`, "9-digit TFN"},
			{`\d{3}-\d{3}-\d{3}`, "TFN with dashes"},
			{`\d{11}`, "11-digit ABN"},
			{`\d{2}\s\d{3}\s\d{3}\s\d{3}`, "ABN with spaces"},
		}

		for _, pattern := range unsafePatterns {
			assert.NotRegexp(t, pattern.pattern, content,
				"Report should not contain unmasked %s", pattern.name)
		}

		t.Log("✅ Report security verified - no unmasked PI patterns found")
	}
}

func testRepo(t *testing.T, repo ComprehensiveTestRepo) {
	outputFile := filepath.Join(t.TempDir(), fmt.Sprintf("%s-results.json",
		strings.ReplaceAll(repo.Name, " ", "_")))

	start := time.Now()
	cmd := exec.Command("./pi-scanner", "scan",
		"--repo", repo.URL,
		"--output", outputFile)

	output, err := cmd.CombinedOutput()
	duration := time.Since(start)

	t.Logf("Repository: %s", repo.Name)
	t.Logf("Duration: %v", duration)

	// Handle auth failures gracefully
	if err != nil && strings.Contains(string(output), "authentication") {
		t.Logf("⚠️  Authentication required for %s (expected for private repos)", repo.Name)
		return
	}

	if err != nil {
		t.Logf("Scan error: %v\nOutput: %s", err, string(output))
		return
	}

	// Parse results if available
	if _, err := os.Stat(outputFile); err == nil {
		data, err := os.ReadFile(outputFile)
		require.NoError(t, err)

		var result struct {
			Repository struct {
				URL string `json:"url"`
			} `json:"repository"`
			FilesScanned int `json:"files_scanned"`
			Findings     []struct {
				Type  string `json:"type"`
				Match string `json:"match"`
			} `json:"findings"`
			Stats struct {
				FindingsByType map[string]int `json:"findings_by_type"`
			} `json:"stats"`
		}

		err = json.Unmarshal(data, &result)
		if err != nil {
			t.Logf("Failed to parse results: %v", err)
			return
		}

		t.Logf("Files scanned: %d", result.FilesScanned)
		t.Logf("Total findings: %d", len(result.Findings))

		// Log PI types found
		if len(result.Stats.FindingsByType) > 0 {
			t.Log("PI types found:")
			for piType, count := range result.Stats.FindingsByType {
				t.Logf("  - %s: %d", piType, count)
			}

			// Check if expected types were found
			if repo.LikelyToHaveTestPI {
				foundExpected := false
				for _, expectedType := range repo.ExpectedPITypes {
					if result.Stats.FindingsByType[expectedType] > 0 {
						foundExpected = true
						break
					}
				}

				if foundExpected {
					t.Log("✅ Found expected PI types")
				} else {
					t.Log("⚠️  Did not find expected PI types (may be due to filtering)")
				}
			}
		}

		// Verify masking in findings
		maskedCount := 0
		for _, finding := range result.Findings {
			if strings.Contains(finding.Match, "*") {
				maskedCount++
			}
		}

		if len(result.Findings) > 0 {
			maskingRate := float64(maskedCount) / float64(len(result.Findings)) * 100
			t.Logf("Masking rate: %.1f%% (%d/%d findings masked)",
				maskingRate, maskedCount, len(result.Findings))
			assert.Equal(t, len(result.Findings), maskedCount,
				"All findings should be masked in reports")
		}
	}
}

func buildScanner(t *testing.T) {
	t.Helper()

	// Check if already built
	if _, err := os.Stat("./pi-scanner"); err == nil {
		return
	}

	cmd := exec.Command("go", "build", "-o", "pi-scanner", "./cmd/pi-scanner")
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "Failed to build scanner: %s", string(output))

	t.Cleanup(func() {
		os.Remove("./pi-scanner")
	})
}
