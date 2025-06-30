package integration

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/MacAttak/pi-scanner/pkg/ast"
	"github.com/MacAttak/pi-scanner/pkg/detection"
	"github.com/MacAttak/pi-scanner/pkg/discovery"
	"github.com/MacAttak/pi-scanner/pkg/processing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestASTIntegration tests the full AST-enhanced detection flow
func TestASTIntegration(t *testing.T) {
	// Create test directory structure
	testDir := t.TempDir()

	// Create test files simulating a banking application
	createTestFiles(t, testDir)

	// Verify files were created
	for path := range getTestFiles() {
		fullPath := filepath.Join(testDir, path)
		info, err := os.Stat(fullPath)
		require.NoError(t, err, "File should exist: %s", fullPath)
		t.Logf("Created file: %s (size: %d)", fullPath, info.Size())
	}

	ctx := context.Background()

	// Step 1: File discovery
	discoveryConfig := discovery.DefaultConfig()
	fileDiscovery := discovery.NewFileDiscovery(discoveryConfig)
	files, err := fileDiscovery.DiscoverFiles(ctx, testDir)
	require.NoError(t, err)
	require.NotEmpty(t, files)
	t.Logf("Discovered %d files", len(files))
	for _, f := range files {
		t.Logf("  - %s (binary: %v)", f.Path, f.IsBinary)
	}

	// Step 2: AST analysis
	astAnalyzer := ast.NewAnalyzer(ast.DefaultBankingConfig())
	repoStructure, err := astAnalyzer.AnalyzeRepository(ctx, testDir, files)
	require.NoError(t, err)
	require.NotNil(t, repoStructure)

	// Verify AST analysis results
	assert.NotEmpty(t, repoStructure.Languages)
	assert.NotEmpty(t, repoStructure.FileContexts)

	// Step 3: Detection with AST context
	detectors := []detection.Detector{
		detection.NewDetector(),
	}

	processorConfig := processing.DefaultProcessorConfig()
	processorConfig.NumWorkers = 2
	fileProcessor := processing.NewFileProcessor(processorConfig, detectors)

	err = fileProcessor.Start(ctx)
	require.NoError(t, err)
	defer fileProcessor.Stop()

	// Process files with AST context
	var findings []detection.Finding
	jobsSubmitted := 0
	for _, file := range files {
		if file.IsBinary {
			t.Logf("Skipping binary file: %s", file.Path)
			continue
		}

		content, err := os.ReadFile(file.Path)
		if err != nil {
			t.Logf("Failed to read file %s: %v", file.Path, err)
			continue
		}

		job := processing.FileJob{
			FilePath:   file.Path,
			Content:    content,
			FileInfo:   file,
			ASTContext: repoStructure.GetFileContext(file.Path),
		}

		err = fileProcessor.Submit(job)
		require.NoError(t, err)
		jobsSubmitted++
		t.Logf("Submitted job for %s (content length: %d)", file.Path, len(content))
	}

	// Collect results
	resultsReceived := 0
	for resultsReceived < jobsSubmitted {
		select {
		case result := <-fileProcessor.Results():
			if result.Error == nil {
				findings = append(findings, result.Findings...)
				t.Logf("Processed %s: %d findings", result.FilePath, len(result.Findings))
			} else {
				t.Logf("Error processing %s: %v", result.FilePath, result.Error)
			}
			resultsReceived++
		case <-ctx.Done():
			t.Fatal("Context cancelled")
		}
	}

	// Verify findings have appropriate risk levels
	testFileFindings := 0
	prodFileFindings := 0

	for _, finding := range findings {
		fileCtx := repoStructure.GetFileContext(finding.File)
		if fileCtx != nil && fileCtx.IsTestFile {
			testFileFindings++
			// Test files should have lower risk
			assert.True(t, finding.RiskLevel <= detection.RiskLevelMedium,
				"Test file finding should have lower risk: %s", finding.File)
		} else {
			prodFileFindings++
		}
	}

	// Should have found some PI in both test and production files
	assert.Greater(t, len(findings), 0, "Should find some PI overall")
	t.Logf("Found %d findings total: %d in test files, %d in production files", len(findings), testFileFindings, prodFileFindings)
}

// TestASTRiskZoneMapping tests that AST correctly identifies risk zones
func TestASTRiskZoneMapping(t *testing.T) {
	testCases := []struct {
		name         string
		filePath     string
		expectedZone string
		expectedRisk ast.RiskLevel
	}{
		{
			name:         "Payment service",
			filePath:     "src/main/java/com/bank/payment/PaymentService.java",
			expectedZone: "financial_data",
			expectedRisk: ast.RiskLevelHigh, // Pattern matches */payment/* which is HIGH
		},
		{
			name:         "Customer model",
			filePath:     "src/main/java/com/bank/model/Customer.java",
			expectedZone: "customer_data",
			expectedRisk: ast.RiskLevelHigh, // Pattern matches */model/* which is HIGH
		},
		{
			name:         "Auth controller",
			filePath:     "src/main/java/com/bank/auth/LoginController.java",
			expectedZone: "authentication",
			expectedRisk: ast.RiskLevelMedium, // Pattern matches */controller/* which is MEDIUM
		},
		{
			name:         "Utility class",
			filePath:     "src/main/java/com/bank/util/StringHelper.java",
			expectedZone: "utilities",
			expectedRisk: ast.RiskLevelLow,
		},
		{
			name:         "Test file",
			filePath:     "src/test/java/com/bank/PaymentTest.java",
			expectedZone: "tests",
			expectedRisk: ast.RiskLevelIgnore,
		},
	}

	analyzer := ast.NewAnalyzer(ast.DefaultBankingConfig())

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fileCtx := &ast.FileContext{
				FilePath:  tc.filePath,
				RiskZone:  analyzer.DetermineRiskZone(tc.filePath),
				RiskLevel: analyzer.DetermineRiskLevel(tc.filePath),
			}

			assert.Equal(t, tc.expectedZone, fileCtx.RiskZone,
				"Risk zone mismatch for %s", tc.filePath)
			assert.Equal(t, tc.expectedRisk, fileCtx.RiskLevel,
				"Risk level mismatch for %s", tc.filePath)
		})
	}
}

func getTestFiles() map[string]string {
	return map[string]string{
		"src/main/java/com/bank/payment/PaymentService.java": `
package com.bank.payment;

public class PaymentService {
    private static final String DEFAULT_BSB = "012-345";
    private static final String TEST_ACCOUNT = "12345678";

    public void processPayment(String bsb, String account) {
        // Process payment with BSB: 062-000 and account: 87654321
        transfer(bsb, account);
    }
}`,
		"src/main/java/com/bank/model/Customer.java": `
package com.bank.model;

public class Customer {
    private String name;
    private String tfn; // Example: 123-456-782
    private String email;

    public Customer(String name, String tfn) {
        this.name = name;
        this.tfn = tfn; // TFN: 865-432-108
    }
}`,
		"src/test/java/com/bank/PaymentTest.java": `
package com.bank;

import org.junit.Test;

public class PaymentTest {
    @Test
    public void testPayment() {
        String testBsb = "012-345";
        String testAccount = "12345678";
        String testCard = "4111111111111111"; // Test Visa card

        // Test with mock data
        service.processPayment(testBsb, testAccount);
    }
}`,
		"src/main/resources/application.properties": `
# Payment gateway configuration
payment.gateway.url=https://api.paymentgateway.com
payment.default.bsb=062-000
payment.test.account=87654321
`,
	}
}

// Helper function to create test files
func createTestFiles(t *testing.T, testDir string) {
	testFiles := getTestFiles()

	for path, content := range testFiles {
		fullPath := filepath.Join(testDir, path)
		dir := filepath.Dir(fullPath)

		err := os.MkdirAll(dir, 0755)
		require.NoError(t, err)

		err = os.WriteFile(fullPath, []byte(content), 0644)
		require.NoError(t, err)
	}
}
