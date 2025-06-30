package e2e

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPublicRepoE2E tests the scanner against a small public repo
func TestPublicRepoE2E(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	// Build the scanner
	cmd := exec.Command("go", "build", "-o", "pi-scanner-test", "./cmd/pi-scanner")
	cmd.Dir = filepath.Join("..", "..")
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "Failed to build scanner: %s", string(output))
	defer os.Remove(filepath.Join("..", "..", "pi-scanner-test"))

	// Create output directory
	outputDir := t.TempDir()
	outputFile := filepath.Join(outputDir, "results.json")

	// Run scan on a small public repo
	scanCmd := exec.Command("./pi-scanner-test",
		"https://github.com/octocat/Hello-World",
		"--no-input",
		"--verbose")
	scanCmd.Dir = filepath.Join("..", "..")

	var stdout, stderr bytes.Buffer
	scanCmd.Stdout = &stdout
	scanCmd.Stderr = &stderr

	start := time.Now()
	err = scanCmd.Run()
	duration := time.Since(start)

	t.Logf("Scan completed in %v", duration)
	t.Logf("Stdout: %s", stdout.String())
	if stderr.Len() > 0 {
		t.Logf("Stderr: %s", stderr.String())
	}

	// Command should succeed
	require.NoError(t, err, "Scan command should succeed")

	// Output file should exist
	require.FileExists(t, outputFile, "Output file should be created")

	// Parse and validate results
	data, err := os.ReadFile(outputFile)
	require.NoError(t, err, "Should read output file")

	var result map[string]interface{}
	err = json.Unmarshal(data, &result)
	require.NoError(t, err, "Should parse JSON output")

	// Basic validation
	assert.Contains(t, result, "repository", "Result should have repository info")
	assert.Contains(t, result, "scan_started", "Result should have scan start time")
	assert.Contains(t, result, "scan_finished", "Result should have scan end time")
	assert.Contains(t, result, "files_scanned", "Result should have files scanned count")
	assert.Contains(t, result, "findings", "Result should have findings array")

	// This small repo shouldn't have many findings
	findings := result["findings"].([]interface{})
	t.Logf("Found %d potential PI instances", len(findings))

	// Performance check
	assert.Less(t, duration, 30*time.Second, "Small repo should scan quickly")
}

// TestAustralianPIDetectionE2E tests detection of Australian PI patterns
func TestAustralianPIDetectionE2E(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	// Create a test repository with known Australian PI patterns
	testDir := t.TempDir()
	testFile := filepath.Join(testDir, "test_data.go")

	testContent := `package testdata

// Test data for Australian PI detection
const (
	// Valid test TFN
	TestTFN = "123456782"

	// Valid test ABN
	TestABN = "51824753556"

	// Valid test Medicare
	TestMedicare = "2123456701"

	// Valid test BSB
	TestBSB = "062-001"

	// Not PI - just numbers
	Version = "1.2.3"
	Count = "123456789"
)

type Customer struct {
	Name     string
	TFN      string // Tax File Number
	ABN      string // Australian Business Number
	Medicare string // Medicare Number
}

var testCustomer = Customer{
	Name:     "Test User",
	TFN:      "876543210", // Another valid TFN
	ABN:      "88952560394", // Another valid ABN
	Medicare: "2234567805", // Another valid Medicare
}
`

	err := os.WriteFile(testFile, []byte(testContent), 0644)
	require.NoError(t, err)

	// Build scanner if not already built
	scannerPath := filepath.Join("..", "..", "pi-scanner-test")
	if _, err := os.Stat(scannerPath); os.IsNotExist(err) {
		cmd := exec.Command("go", "build", "-o", "pi-scanner-test", "./cmd/pi-scanner")
		cmd.Dir = filepath.Join("..", "..")
		output, err := cmd.CombinedOutput()
		require.NoError(t, err, "Failed to build scanner: %s", string(output))
		defer os.Remove(scannerPath)
	}

	// Run scan
	outputFile := filepath.Join(t.TempDir(), "au-pi-results.json")
	scanCmd := exec.Command("./pi-scanner-test",
		"file://"+testDir,
		"--no-input")
	scanCmd.Dir = filepath.Join("..", "..")

	output, err := scanCmd.CombinedOutput()
	t.Logf("Scan output: %s", string(output))
	require.NoError(t, err, "Scan should succeed")

	// Parse results
	data, err := os.ReadFile(outputFile)
	require.NoError(t, err)

	var result struct {
		Findings []struct {
			Type  string `json:"type"`
			Match string `json:"match"`
			File  string `json:"file"`
			Line  int    `json:"line"`
		} `json:"findings"`
		Stats struct {
			FindingsByType map[string]int `json:"findings_by_type"`
		} `json:"stats"`
	}

	err = json.Unmarshal(data, &result)
	require.NoError(t, err)

	// Verify Australian PI types were detected
	assert.NotEmpty(t, result.Findings, "Should find PI patterns")

	foundTypes := make(map[string]bool)
	for _, finding := range result.Findings {
		foundTypes[finding.Type] = true
		t.Logf("Found %s: %s at line %d", finding.Type, finding.Match, finding.Line)
	}

	// Should detect key Australian PI types
	assert.True(t, foundTypes["TFN"], "Should detect TFN")
	assert.True(t, foundTypes["ABN"], "Should detect ABN")
	assert.True(t, foundTypes["MEDICARE"], "Should detect Medicare")
	assert.True(t, foundTypes["BSB"], "Should detect BSB")

	// Verify counts
	assert.GreaterOrEqual(t, result.Stats.FindingsByType["TFN"], 2, "Should find multiple TFNs")
	assert.GreaterOrEqual(t, result.Stats.FindingsByType["ABN"], 2, "Should find multiple ABNs")
	assert.GreaterOrEqual(t, result.Stats.FindingsByType["MEDICARE"], 2, "Should find multiple Medicare numbers")
}

// TestReportSecurityE2E verifies that reports don't expose unmasked PI
func TestReportSecurityE2E(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	// Create test data with PI
	testDir := t.TempDir()
	testFile := filepath.Join(testDir, "sensitive.go")

	err := os.WriteFile(testFile, []byte(`package main
const RealTFN = "123456782" // This is sensitive!
`), 0644)
	require.NoError(t, err)

	// Build scanner
	scannerPath := filepath.Join("..", "..", "pi-scanner-test")
	if _, err := os.Stat(scannerPath); os.IsNotExist(err) {
		cmd := exec.Command("go", "build", "-o", "pi-scanner-test", "./cmd/pi-scanner")
		cmd.Dir = filepath.Join("..", "..")
		_, err := cmd.CombinedOutput()
		require.NoError(t, err)
		defer os.Remove(scannerPath)
	}

	// Run scan with output
	outputFile := filepath.Join(t.TempDir(), "secure-results.json")
	scanCmd := exec.Command("./pi-scanner-test",
		"file://"+testDir,
		"--no-input")
	scanCmd.Dir = filepath.Join("..", "..")

	_, err = scanCmd.CombinedOutput()
	require.NoError(t, err)

	// Read the output file
	data, err := os.ReadFile(outputFile)
	require.NoError(t, err)

	outputStr := string(data)

	// Verify no unmasked PI in output
	assert.NotContains(t, outputStr, "123456782", "Output should not contain unmasked TFN")

	// Should contain masked version
	assert.Contains(t, outputStr, "123****82", "Output should contain masked TFN")
}
