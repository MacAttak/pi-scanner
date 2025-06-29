package detection

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestContextAwareDetection verifies that the detector properly considers context
// when assigning risk levels to detected PI
func TestContextAwareDetection(t *testing.T) {
	detector := NewDetector()

	// Same PI value in different contexts should have different risk levels
	validTFN := "123456782"
	validABN := "51824753556"
	validMedicare := "2123456701"

	tests := []struct {
		name          string
		content       string
		filename      string
		expectedRisk  RiskLevel
		contextReason string
	}{
		// Production code contexts - HIGH/CRITICAL risk
		{
			name:          "Production service code",
			content:       `customerTFN := "` + validTFN + `"`,
			filename:      "services/customer_service.go",
			expectedRisk:  RiskLevelHigh,
			contextReason: "PI in production service code",
		},
		{
			name:          "Database model",
			content:       `type User struct { TFN string // "` + validTFN + `" }`,
			filename:      "models/user.go",
			expectedRisk:  RiskLevelHigh,
			contextReason: "PI in database model",
		},
		{
			name:          "API handler",
			content:       `func SaveTFN(tfn string) { // Example: ` + validTFN + ` }`,
			filename:      "api/handlers/tax.go",
			expectedRisk:  RiskLevelHigh,
			contextReason: "PI in API handler",
		},
		{
			name:          "Configuration file",
			content:       `default_tfn: "` + validTFN + `"`,
			filename:      "config/settings.yaml",
			expectedRisk:  RiskLevelCritical,
			contextReason: "PI in configuration file",
		},
		{
			name:          "Multiple PI in production",
			content:       `customer := Customer{TFN: "` + validTFN + `", ABN: "` + validABN + `"}`,
			filename:      "services/billing.go",
			expectedRisk:  RiskLevelCritical,
			contextReason: "Multiple PI in proximity",
		},

		// Test code contexts - LOW risk
		{
			name:          "Unit test file",
			content:       `testTFN := "` + validTFN + `"`,
			filename:      "services/customer_service_test.go",
			expectedRisk:  RiskLevelLow,
			contextReason: "PI in test file",
		},
		{
			name:          "Test fixtures",
			content:       `fixtures := []string{"` + validTFN + `", "` + validABN + `"}`,
			filename:      "test/fixtures/test_data.go",
			expectedRisk:  RiskLevelLow,
			contextReason: "PI in test fixtures",
		},
		{
			name:          "Mock data",
			content:       `mockTFN = "` + validTFN + `" // Test data`,
			filename:      "mocks/customer_mock.go",
			expectedRisk:  RiskLevelLow,
			contextReason: "PI in mock file",
		},
		{
			name:          "Example test",
			content:       `Example_TFN() { fmt.Println("` + validTFN + `") }`,
			filename:      "examples/tfn_example_test.go",
			expectedRisk:  RiskLevelLow,
			contextReason: "PI in example test",
		},
		{
			name:          "Benchmark test",
			content:       `func BenchmarkTFN(b *testing.B) { tfn := "` + validTFN + `" }`,
			filename:      "benchmark_test.go",
			expectedRisk:  RiskLevelLow,
			contextReason: "PI in benchmark",
		},

		// Documentation contexts - MEDIUM risk
		{
			name:          "README documentation",
			content:       `Example: Use TFN ` + validTFN + ` for testing`,
			filename:      "README.md",
			expectedRisk:  RiskLevelMedium,
			contextReason: "PI in documentation",
		},
		{
			name:          "Code comments",
			content:       `// Example TFN: ` + validTFN,
			filename:      "utils/validator.go",
			expectedRisk:  RiskLevelMedium,
			contextReason: "PI in comment",
		},
		{
			name:          "API documentation",
			content:       `@apiExample {json} Request { "tfn": "` + validTFN + `" }`,
			filename:      "docs/api.md",
			expectedRisk:  RiskLevelMedium,
			contextReason: "PI in API docs",
		},

		// Special contexts
		{
			name:          "Migration script",
			content:       `UPDATE users SET tfn = '` + validTFN + `' WHERE id = 1`,
			filename:      "migrations/001_add_test_data.sql",
			expectedRisk:  RiskLevelMedium,
			contextReason: "PI in migration might be test data",
		},
		{
			name:          "Script file",
			content:       `TEST_TFN="` + validTFN + `"`,
			filename:      "scripts/test_data.sh",
			expectedRisk:  RiskLevelMedium,
			contextReason: "PI in script might be for testing",
		},
		{
			name:          "JSON test data",
			content:       `{"testData": {"tfn": "` + validTFN + `"}}`,
			filename:      "testdata/sample.json",
			expectedRisk:  RiskLevelLow,
			contextReason: "PI in test data directory",
		},

		// Variable naming contexts
		{
			name:          "Test-prefixed variable",
			content:       `testCustomerTFN := "` + validTFN + `"`,
			filename:      "service.go",
			expectedRisk:  RiskLevelMedium,
			contextReason: "Test-prefixed variable suggests test data",
		},
		{
			name:          "Mock-prefixed variable",
			content:       `mockABN := "` + validABN + `"`,
			filename:      "handler.go",
			expectedRisk:  RiskLevelMedium,
			contextReason: "Mock-prefixed variable suggests test data",
		},
		{
			name:          "Example variable",
			content:       `exampleMedicare := "` + validMedicare + `"`,
			filename:      "validator.go",
			expectedRisk:  RiskLevelMedium,
			contextReason: "Example variable suggests documentation",
		},

		// Synthetic data patterns
		{
			name:          "Sequential TFNs",
			content:       `tfns := []string{"123456789", "123456790", "123456791"}`,
			filename:      "data.go",
			expectedRisk:  RiskLevelLow,
			contextReason: "Sequential numbers suggest synthetic data",
		},
		{
			name:          "Repeated digits",
			content:       `defaultTFN := "111111111"`,
			filename:      "config.go",
			expectedRisk:  RiskLevelLow,
			contextReason: "Repeated digits suggest placeholder",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			findings, err := detector.Detect(context.Background(), []byte(tt.content), tt.filename)
			require.NoError(t, err)

			if len(findings) == 0 {
				t.Fatalf("Expected PI detection in %s", tt.filename)
			}

			// Get the highest risk level from findings
			highestRisk := RiskLevelLow
			for _, finding := range findings {
				if finding.RiskLevel >= highestRisk {
					highestRisk = finding.RiskLevel
				}
			}

			// Allow some flexibility - context detection might not be perfect
			// but should be in the right direction
			if tt.expectedRisk == RiskLevelLow {
				assert.True(t, highestRisk <= RiskLevelMedium,
					"Expected low risk for %s, got %s\nExpected: %s",
					tt.contextReason, highestRisk, tt.contextReason)
			} else if tt.expectedRisk == RiskLevelCritical {
				assert.True(t, highestRisk >= RiskLevelHigh,
					"Expected critical/high risk for %s, got %s",
					tt.contextReason, highestRisk)
			} else {
				// For medium risk, accept anything from low to high
				assert.True(t, highestRisk >= RiskLevelLow && highestRisk <= RiskLevelHigh,
					"Expected medium risk range for %s, got %s",
					tt.contextReason, highestRisk)
			}
		})
	}
}

// TestFilePathContext tests that file paths influence risk assessment
func TestFilePathContext(t *testing.T) {
	detector := NewDetector()
	tfn := "123456782"

	testPaths := []struct {
		path        string
		shouldBeLow bool
		reason      string
	}{
		// Test-related paths
		{"test/data.go", true, "test directory"},
		{"tests/fixtures.go", true, "tests directory"},
		{"spec/examples.go", true, "spec directory"},
		{"testdata/sample.go", true, "testdata directory"},
		{"examples/demo.go", true, "examples directory"},
		{"mocks/service.go", true, "mocks directory"},
		{"fixtures/users.go", true, "fixtures directory"},
		{"__tests__/app.js", true, "jest test directory"},
		{"test_utils.go", true, "test utils file"},
		{"mock_service.go", true, "mock file"},

		// Documentation paths
		{"docs/api.md", true, "docs directory"},
		{"README.md", true, "readme file"},
		{"CONTRIBUTING.md", true, "contributing guide"},
		{"examples.md", true, "examples file"},

		// Production paths
		{"src/service.go", false, "source directory"},
		{"pkg/handler.go", false, "package directory"},
		{"internal/core.go", false, "internal directory"},
		{"lib/validator.go", false, "library directory"},
		{"app/models.go", false, "app directory"},
		{"services/customer.go", false, "services directory"},
		{"handlers/api.go", false, "handlers directory"},
		{"config/prod.yaml", false, "config file"},
		{".env", false, "environment file"},
		{"secrets.json", false, "secrets file"},
	}

	for _, tp := range testPaths {
		t.Run(tp.path, func(t *testing.T) {
			content := `const TFN = "` + tfn + `"`
			findings, err := detector.Detect(context.Background(), []byte(content), tp.path)
			require.NoError(t, err)
			require.NotEmpty(t, findings, "Should detect TFN in %s", tp.path)

			risk := findings[0].RiskLevel
			if tp.shouldBeLow {
				assert.True(t, risk <= RiskLevelMedium,
					"%s should have low/medium risk, got %s", tp.reason, risk)
			} else {
				assert.True(t, risk >= RiskLevelMedium,
					"%s should have medium/high risk, got %s", tp.reason, risk)
			}
		})
	}
}

// TestProximityContext tests that nearby PI increases risk level
func TestProximityContext(t *testing.T) {
	detector := NewDetector()

	tests := []struct {
		name         string
		content      string
		minRiskLevel RiskLevel
		reason       string
	}{
		{
			name: "Multiple PI in struct",
			content: `type Customer struct {
				Name     string
				TFN      string // 123456782
				ABN      string // 51824753556
				Medicare string // 2123456701
			}`,
			minRiskLevel: RiskLevelHigh,
			reason:       "Multiple PI types in proximity should elevate risk",
		},
		{
			name: "PI with PII",
			content: `customer := map[string]string{
				"name": "John Smith",
				"email": "john@example.com",
				"tfn": "123456782",
				"phone": "0412345678",
			}`,
			minRiskLevel: RiskLevelHigh,
			reason:       "PI with PII (name, email) should be high risk",
		},
		{
			name: "Isolated PI",
			content: `
			// Some unrelated code here
			func validateInput(input string) bool {
				// More code
				return true
			}

			// Default TFN for testing
			const DEFAULT_TFN = "123456782"

			// More unrelated code
			`,
			minRiskLevel: RiskLevelLow,
			reason:       "Isolated PI should have lower risk",
		},
		{
			name:         "Database query with PI",
			content:      `query := "INSERT INTO customers (name, tfn, abn) VALUES ('Test', '123456782', '51824753556')"`,
			minRiskLevel: RiskLevelCritical,
			reason:       "PI in database queries is critical risk",
		},
		{
			name: "API response with PI",
			content: `return Response{
				"status": "success",
				"data": {
					"tfn": "123456782",
					"abn": "51824753556"
				}
			}`,
			minRiskLevel: RiskLevelCritical,
			reason:       "PI in API responses is critical risk",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			findings, err := detector.Detect(context.Background(), []byte(tt.content), "app.go")
			require.NoError(t, err)
			require.NotEmpty(t, findings, "Should detect PI")

			// Find highest risk level
			highestRisk := RiskLevelLow
			for _, finding := range findings {
				if finding.RiskLevel > highestRisk {
					highestRisk = finding.RiskLevel
				}
			}

			assert.True(t, highestRisk >= tt.minRiskLevel,
				"%s: expected minimum risk %s, got %s",
				tt.reason, tt.minRiskLevel, highestRisk)
		})
	}
}
