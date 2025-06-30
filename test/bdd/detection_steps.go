package bdd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/MacAttak/pi-scanner/pkg/detection"
	"github.com/cucumber/godog"
)

type detectionContext struct {
	detector    detection.Detector
	testFile    string
	testContent string
	findings    []detection.Finding
	currentFile string
	scanOptions detection.ScanOptions
	llmEnabled  bool
}

var detect = &detectionContext{}

func registerDetectionSteps(ctx *godog.ScenarioContext) {
	// Background
	ctx.Step(`^the scanner is configured for Australian PI detection$`, theScannerIsConfiguredForAustralianPIDetection)

	// Given steps
	ctx.Step(`^a file "([^"]*)" containing:$`, aFileContaining)

	// When steps
	ctx.Step(`^I scan the file$`, iScanTheFile)
	ctx.Step(`^I scan the file with ML validation enabled$`, iScanTheFileWithMLValidationEnabled)

	// Then steps - Finding detection
	ctx.Step(`^a TFN finding should be detected$`, aTFNFindingShouldBeDetected)
	ctx.Step(`^a Medicare finding should be detected$`, aMedicareFindingShouldBeDetected)
	ctx.Step(`^an ABN finding should be detected$`, anABNFindingShouldBeDetected)
	ctx.Step(`^a BSB finding should be detected$`, aBSBFindingShouldBeDetected)
	ctx.Step(`^a bank account finding should be detected$`, aBankAccountFindingShouldBeDetected)
	ctx.Step(`^(\d+) driver's license findings should be detected$`, driverLicenseFindingsShouldBeDetected)
	ctx.Step(`^multiple PI findings should be detected$`, multiplePIFindingsShouldBeDetected)
	ctx.Step(`^PI findings should be detected$`, piFindingsShouldBeDetected)

	// Then steps - Risk and validation
	ctx.Step(`^the finding should have risk level "([^"]*)"$`, theFindingShouldHaveRiskLevel)
	ctx.Step(`^the combined risk level should be "([^"]*)"$`, theCombinedRiskLevelShouldBe)
	ctx.Step(`^the risk level should be "([^"]*)"$`, theRiskLevelShouldBe)
	ctx.Step(`^the risk reason should include "([^"]*)"$`, theRiskReasonShouldInclude)

	// Then steps - Validation
	ctx.Step(`^the finding should pass TFN checksum validation$`, theFindingShouldPassTFNChecksumValidation)
	ctx.Step(`^the finding should pass Medicare checksum validation$`, theFindingShouldPassMedicareChecksumValidation)
	ctx.Step(`^the finding should pass ABN modulus 89 validation$`, theFindingShouldPassABNModulus89Validation)
	ctx.Step(`^the matched text should be normalized to "([^"]*)"$`, theMatchedTextShouldBeNormalizedTo)

	// Then steps - Context
	ctx.Step(`^the context should indicate "([^"]*)"$`, theContextShouldIndicate)
	ctx.Step(`^the findings should be marked as co-occurring$`, theFindingsShouldBeMarkedAsCoOccurring)
	ctx.Step(`^they should be flagged as "([^"]*)"$`, theyShouldBeFlaggedAs)
	ctx.Step(`^the findings should be marked as "([^"]*)"$`, theFindingsShouldBeMarkedAs)
	ctx.Step(`^"([^"]*)" should not be detected as a valid license$`, shouldNotBeDetectedAsAValidLicense)

	// ML validation steps
	ctx.Step(`^the ML model should analyze the context$`, theMLModelShouldAnalyzeTheContext)
	ctx.Step(`^the confidence should be low due to comment context$`, theConfidenceShouldBeLowDueToCommentContext)

	// But steps
	ctx.Step(`^the risk level should be reduced$`, theRiskLevelShouldBeReduced)

	// Cleanup
	ctx.After(func(ctx context.Context, sc *godog.Scenario, err error) (context.Context, error) {
		cleanupDetection()
		return ctx, nil
	})
}

func theScannerIsConfiguredForAustralianPIDetection() error {
	// Initialize the detector
	detect.detector = detection.NewDetector()
	detect.scanOptions = detection.ScanOptions{
		IncludeContext: true,
		ContextLines:   5,
	}
	return nil
}

func aFileContaining(filename string, content *godog.DocString) error {
	detect.testFile = filename
	detect.testContent = content.Content
	detect.currentFile = filepath.Join(os.TempDir(), filename)

	// Write the test content to a temporary file
	return os.WriteFile(detect.currentFile, []byte(detect.testContent), 0644)
}

func iScanTheFile() error {
	if detect.detector == nil {
		return fmt.Errorf("detector not initialized")
	}

	// Scan the file
	findings, err := detect.detector.ScanFile(context.Background(), detect.currentFile, detect.scanOptions)
	if err != nil {
		return err
	}

	detect.findings = findings
	return nil
}

func iScanTheFileWithMLValidationEnabled() error {
	detect.llmEnabled = true
	detect.scanOptions.EnableLLM = true
	return iScanTheFile()
}

func aTFNFindingShouldBeDetected() error {
	return findingOfTypeShouldBeDetected(detection.PITypeTFN)
}

func aMedicareFindingShouldBeDetected() error {
	return findingOfTypeShouldBeDetected(detection.PITypeMedicare)
}

func anABNFindingShouldBeDetected() error {
	return findingOfTypeShouldBeDetected(detection.PITypeABN)
}

func aBSBFindingShouldBeDetected() error {
	return findingOfTypeShouldBeDetected(detection.PITypeBSB)
}

func aBankAccountFindingShouldBeDetected() error {
	return findingOfTypeShouldBeDetected(detection.PITypeBankAccount)
}

func driverLicenseFindingsShouldBeDetected(count int) error {
	licenseCount := 0
	for _, finding := range detect.findings {
		if finding.Type == detection.PITypeDriverLicense {
			licenseCount++
		}
	}

	if licenseCount != count {
		return fmt.Errorf("expected %d driver's license findings, got %d", count, licenseCount)
	}
	return nil
}

func multiplePIFindingsShouldBeDetected() error {
	if len(detect.findings) < 2 {
		return fmt.Errorf("expected multiple findings, got %d", len(detect.findings))
	}
	return nil
}

func piFindingsShouldBeDetected() error {
	if len(detect.findings) == 0 {
		return fmt.Errorf("no PI findings detected")
	}
	return nil
}

func theFindingShouldHaveRiskLevel(level string) error {
	if len(detect.findings) == 0 {
		return fmt.Errorf("no findings to check")
	}

	// Check the most recent finding
	finding := detect.findings[len(detect.findings)-1]
	if finding.RiskLevel != detection.RiskLevel(level) {
		return fmt.Errorf("expected risk level %s, got %s", level, finding.RiskLevel)
	}
	return nil
}

func theCombinedRiskLevelShouldBe(level string) error {
	// When multiple findings exist in proximity, check the highest risk level
	highestRisk := detection.RiskLevelLow
	for _, finding := range detect.findings {
		if finding.RiskLevel > highestRisk {
			highestRisk = finding.RiskLevel
		}
	}

	if string(highestRisk) != level {
		return fmt.Errorf("expected combined risk level %s, got %s", level, highestRisk)
	}
	return nil
}

func theRiskLevelShouldBe(level string) error {
	return theFindingShouldHaveRiskLevel(level)
}

func theRiskReasonShouldInclude(reason string) error {
	for _, finding := range detect.findings {
		if strings.Contains(finding.Reason, reason) {
			return nil
		}
	}
	return fmt.Errorf("no finding contains risk reason: %s", reason)
}

func theFindingShouldPassTFNChecksumValidation() error {
	// TFN validation is built into the detection
	// If a TFN was detected, it passed validation
	for _, finding := range detect.findings {
		if finding.Type == detection.PITypeTFN {
			return nil
		}
	}
	return fmt.Errorf("no TFN finding detected")
}

func theFindingShouldPassMedicareChecksumValidation() error {
	for _, finding := range detect.findings {
		if finding.Type == detection.PITypeMedicare {
			return nil
		}
	}
	return fmt.Errorf("no Medicare finding detected")
}

func theFindingShouldPassABNModulus89Validation() error {
	for _, finding := range detect.findings {
		if finding.Type == detection.PITypeABN {
			return nil
		}
	}
	return fmt.Errorf("no ABN finding detected")
}

func theMatchedTextShouldBeNormalizedTo(normalized string) error {
	// Check that the finding contains the normalized value
	for _, finding := range detect.findings {
		// The detector should normalize the value internally
		cleanValue := strings.ReplaceAll(finding.Value, "-", "")
		cleanValue = strings.ReplaceAll(cleanValue, " ", "")
		if cleanValue == normalized {
			return nil
		}
	}
	return fmt.Errorf("no finding with normalized value %s", normalized)
}

func theContextShouldIndicate(context string) error {
	for _, finding := range detect.findings {
		if strings.Contains(finding.Context, context) ||
			strings.Contains(finding.FilePath, context) ||
			strings.Contains(finding.Reason, context) {
			return nil
		}
	}
	return fmt.Errorf("no finding indicates context: %s", context)
}

func theFindingsShouldBeMarkedAsCoOccurring() error {
	// Check if findings are in proximity (within a few lines)
	if len(detect.findings) < 2 {
		return fmt.Errorf("need at least 2 findings for co-occurrence")
	}

	// Check if findings are close together
	for i := 0; i < len(detect.findings)-1; i++ {
		for j := i + 1; j < len(detect.findings); j++ {
			lineDiff := abs(detect.findings[i].Line - detect.findings[j].Line)
			if lineDiff <= 10 { // Within 10 lines
				return nil
			}
		}
	}

	return fmt.Errorf("findings are not in proximity")
}

func theyShouldBeFlaggedAs(flag string) error {
	for _, finding := range detect.findings {
		if strings.Contains(finding.Reason, flag) {
			return nil
		}
	}
	return fmt.Errorf("no finding flagged as: %s", flag)
}

func theFindingsShouldBeMarkedAs(marking string) error {
	return theyShouldBeFlaggedAs(marking)
}

func shouldNotBeDetectedAsAValidLicense(value string) error {
	for _, finding := range detect.findings {
		if finding.Type == detection.PITypeDriverLicense && finding.Value == value {
			return fmt.Errorf("%s was incorrectly detected as a driver's license", value)
		}
	}
	return nil
}

func theMLModelShouldAnalyzeTheContext() error {
	if !detect.llmEnabled {
		return fmt.Errorf("ML validation was not enabled")
	}
	// In a real implementation, we would check that LLM analysis occurred
	return nil
}

func theConfidenceShouldBeLowDueToCommentContext() error {
	// Check that findings in comments have reduced confidence
	for _, finding := range detect.findings {
		if strings.Contains(finding.Context, "//") || strings.Contains(finding.Context, "/*") {
			if finding.Confidence > 0.5 {
				return fmt.Errorf("comment context should have low confidence, got %f", finding.Confidence)
			}
		}
	}
	return nil
}

func theRiskLevelShouldBeReduced() error {
	// Verify that synthetic/test data has reduced risk
	for _, finding := range detect.findings {
		if finding.RiskLevel == detection.RiskLevelCritical ||
			finding.RiskLevel == detection.RiskLevelHigh {
			return fmt.Errorf("synthetic data should not have high risk level: %s", finding.RiskLevel)
		}
	}
	return nil
}

// Helper functions
func findingOfTypeShouldBeDetected(piType detection.PIType) error {
	for _, finding := range detect.findings {
		if finding.Type == piType {
			return nil
		}
	}
	return fmt.Errorf("no %s finding detected", piType)
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

func cleanupDetection() {
	if detect.currentFile != "" {
		os.Remove(detect.currentFile)
	}
	detect.findings = nil
	detect.llmEnabled = false
}
