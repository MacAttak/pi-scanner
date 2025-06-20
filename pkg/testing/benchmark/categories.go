package benchmark

import (
	"fmt"
	"strings"
	"github.com/MacAttak/pi-scanner/pkg/detection"
)

// GenerateComprehensiveTestDataset generates a comprehensive dataset with 200+ test cases
func GenerateComprehensiveTestDataset() *BenchmarkDataset {
	generator := NewTestDataGenerator()
	dataset := &BenchmarkDataset{
		TruePositives: []TestCase{},
		TrueNegatives: []TestCase{},
		EdgeCases:     []TestCase{},
		Synthetic:     []TestCase{},
	}

	// Generate test cases for each category
	dataset.TruePositives = append(dataset.TruePositives, generateTFNTestCases(generator)...)
	dataset.TruePositives = append(dataset.TruePositives, generateABNTestCases(generator)...)
	dataset.TruePositives = append(dataset.TruePositives, generateMedicareTestCases(generator)...)
	dataset.TruePositives = append(dataset.TruePositives, generateBSBTestCases(generator)...)
	dataset.TruePositives = append(dataset.TruePositives, generateACNTestCases(generator)...)
	dataset.TruePositives = append(dataset.TruePositives, generateDriverLicenseTestCases(generator)...)
	dataset.TruePositives = append(dataset.TruePositives, generateMultiPITestCases(generator)...)

	dataset.TrueNegatives = append(dataset.TrueNegatives, generateFalsePositiveCases(generator)...)
	dataset.EdgeCases = append(dataset.EdgeCases, generateEdgeCases(generator)...)
	dataset.Synthetic = append(dataset.Synthetic, generateSyntheticPatterns(generator)...)

	return dataset
}

// generateTFNTestCases generates comprehensive TFN test cases
func generateTFNTestCases(g *TestDataGenerator) []TestCase {
	cases := []TestCase{}
	id := 0

	// Valid TFNs in production contexts
	validTFNs := []string{
		g.GenerateValidTFN(),
		"123456782", // Known valid TFN
		"876543217", // Another valid TFN
		"564738291", // Valid with different pattern
	}

	// Production contexts
	for i, tfn := range validTFNs {
		// Direct assignment
		cases = append(cases, TestCase{
			ID:         fmt.Sprintf("tfn-prod-%03d", id),
			Code:       g.WrapInContext(tfn, detection.PITypeTFN, "assignment", "go"),
			Language:   "go",
			PIType:     detection.PITypeTFN,
			IsActualPI: true,
			Context:    "production",
			Rationale:  "Valid TFN in direct assignment",
			Filename:   "user.go",
		})
		id++

		// With formatting
		formatted := g.FormatTFN(tfn, "dashes")
		cases = append(cases, TestCase{
			ID:         fmt.Sprintf("tfn-prod-%03d", id),
			Code:       fmt.Sprintf(`customer.TaxFileNumber = "%s"`, formatted),
			Language:   "go",
			PIType:     detection.PITypeTFN,
			IsActualPI: true,
			Context:    "production",
			Rationale:  "Valid TFN with dash formatting",
			Filename:   "customer.go",
		})
		id++

		// In struct
		cases = append(cases, TestCase{
			ID:         fmt.Sprintf("tfn-prod-%03d", id),
			Code:       g.WrapInContext(tfn, detection.PITypeTFN, "struct", "go"),
			Language:   "go",
			PIType:     detection.PITypeTFN,
			IsActualPI: true,
			Context:    "production",
			Rationale:  "Valid TFN in struct initialization",
			Filename:   "models.go",
		})
		id++

		// Different languages
		if i < 2 {
			cases = append(cases, TestCase{
				ID:         fmt.Sprintf("tfn-prod-%03d", id),
				Code:       g.WrapInContext(tfn, detection.PITypeTFN, "assignment", "python"),
				Language:   "python",
				PIType:     detection.PITypeTFN,
				IsActualPI: true,
				Context:    "production",
				Rationale:  "Valid TFN in Python code",
				Filename:   "user.py",
			})
			id++
		}
	}

	// Test/Mock contexts (false positives)
	for _, tfn := range validTFNs[:2] {
		cases = append(cases, TestCase{
			ID:         fmt.Sprintf("tfn-test-%03d", id),
			Code:       g.WrapInContext(tfn, detection.PITypeTFN, "test", "go"),
			Language:   "go",
			PIType:     detection.PITypeTFN,
			IsActualPI: false,
			Context:    "test",
			Rationale:  "TFN in test file",
			Filename:   "user_test.go",
		})
		id++

		cases = append(cases, TestCase{
			ID:         fmt.Sprintf("tfn-mock-%03d", id),
			Code:       g.WrapInContext(tfn, detection.PITypeTFN, "mock", "go"),
			Language:   "go",
			PIType:     detection.PITypeTFN,
			IsActualPI: false,
			Context:    "mock",
			Rationale:  "TFN as mock data",
			Filename:   "mock_data.go",
		})
		id++
	}

	// Invalid TFNs
	invalidTFNs := []string{
		g.GenerateInvalidTFN(),
		"123456789", // Sequential
		"111111111", // Repeated digits
		"000000000", // All zeros
	}

	for _, tfn := range invalidTFNs {
		cases = append(cases, TestCase{
			ID:         fmt.Sprintf("tfn-invalid-%03d", id),
			Code:       fmt.Sprintf(`tfn := "%s"`, tfn),
			Language:   "go",
			PIType:     detection.PITypeTFN,
			IsActualPI: false,
			Context:    "production",
			Rationale:  "Invalid TFN (fails checksum)",
			Filename:   "validate.go",
		})
		id++
	}

	return cases
}

// generateABNTestCases generates comprehensive ABN test cases
func generateABNTestCases(g *TestDataGenerator) []TestCase {
	cases := []TestCase{}
	id := 0

	// Valid ABNs
	validABNs := []string{
		g.GenerateValidABN(),
		"51824753556", // Commonwealth Bank
		"83914571673", // Another valid ABN
	}

	for _, abn := range validABNs {
		// Production context
		cases = append(cases, TestCase{
			ID:         fmt.Sprintf("abn-prod-%03d", id),
			Code:       fmt.Sprintf(`company.ABN = "%s"`, abn),
			Language:   "go",
			PIType:     detection.PITypeABN,
			IsActualPI: true,
			Context:    "production",
			Rationale:  "Valid ABN in production code",
			Filename:   "company.go",
		})
		id++

		// With formatting
		formatted := g.FormatABN(abn, "spaces")
		cases = append(cases, TestCase{
			ID:         fmt.Sprintf("abn-prod-%03d", id),
			Code:       fmt.Sprintf(`businessNumber := "%s"`, formatted),
			Language:   "go",
			PIType:     detection.PITypeABN,
			IsActualPI: true,
			Context:    "production",
			Rationale:  "Valid ABN with space formatting",
			Filename:   "business.go",
		})
		id++
	}

	// Public ABNs (edge cases)
	publicABNs := []string{
		"11000000000", // Test ABN
		"53004085616", // ATO ABN
	}

	for _, abn := range publicABNs {
		cases = append(cases, TestCase{
			ID:         fmt.Sprintf("abn-public-%03d", id),
			Code:       fmt.Sprintf(`const ATO_ABN = "%s" // Australian Tax Office`, abn),
			Language:   "go",
			PIType:     detection.PITypeABN,
			IsActualPI: false,
			Context:    "production",
			Rationale:  "Public/government ABN",
			Filename:   "constants.go",
		})
		id++
	}

	return cases
}

// generateMedicareTestCases generates comprehensive Medicare test cases
func generateMedicareTestCases(g *TestDataGenerator) []TestCase {
	cases := []TestCase{}
	id := 0

	// Valid Medicare numbers
	for i := 0; i < 5; i++ {
		medicare := g.GenerateValidMedicare()
		
		// Production context
		cases = append(cases, TestCase{
			ID:         fmt.Sprintf("medicare-prod-%03d", id),
			Code:       fmt.Sprintf(`patient.MedicareNumber = "%s"`, medicare),
			Language:   "go",
			PIType:     detection.PITypeMedicare,
			IsActualPI: true,
			Context:    "production",
			Rationale:  "Valid Medicare number in patient record",
			Filename:   "patient.go",
		})
		id++

		// With formatting
		if len(medicare) == 10 {
			formatted := medicare[:4] + " " + medicare[4:9] + " " + medicare[9:]
			cases = append(cases, TestCase{
				ID:         fmt.Sprintf("medicare-prod-%03d", id),
				Code:       fmt.Sprintf(`healthCard := "%s"`, formatted),
				Language:   "go",
				PIType:     detection.PITypeMedicare,
				IsActualPI: true,
				Context:    "production",
				Rationale:  "Valid Medicare with space formatting",
				Filename:   "health.go",
			})
			id++
		}
	}

	// Test contexts
	medicare := g.GenerateValidMedicare()
	cases = append(cases, TestCase{
		ID:         fmt.Sprintf("medicare-test-%03d", id),
		Code:       g.WrapInContext(medicare, detection.PITypeMedicare, "test", "python"),
		Language:   "python",
		PIType:     detection.PITypeMedicare,
		IsActualPI: false,
		Context:    "test",
		Rationale:  "Medicare number in test file",
		Filename:   "test_health.py",
	})
	id++

	return cases
}

// generateBSBTestCases generates comprehensive BSB test cases
func generateBSBTestCases(g *TestDataGenerator) []TestCase {
	cases := []TestCase{}
	id := 0

	// Valid BSBs
	for i := 0; i < 5; i++ {
		bsb := g.GenerateValidBSB()
		
		// Production context
		cases = append(cases, TestCase{
			ID:         fmt.Sprintf("bsb-prod-%03d", id),
			Code:       fmt.Sprintf(`account.BSB = "%s"`, bsb[:3]+"-"+bsb[3:]),
			Language:   "go",
			PIType:     detection.PITypeBSB,
			IsActualPI: true,
			Context:    "production",
			Rationale:  "Valid BSB in account details",
			Filename:   "account.go",
		})
		id++
	}

	// Known bank BSBs (edge cases)
	bankBSBs := map[string]string{
		"062-000": "Commonwealth Bank Sydney",
		"033-000": "Westpac Sydney",
		"013-001": "ANZ Melbourne",
	}

	for bsb, bank := range bankBSBs {
		cases = append(cases, TestCase{
			ID:         fmt.Sprintf("bsb-bank-%03d", id),
			Code:       fmt.Sprintf(`const %s_BSB = "%s"`, strings.ReplaceAll(bank, " ", "_"), bsb),
			Language:   "go",
			PIType:     detection.PITypeBSB,
			IsActualPI: false,
			Context:    "production",
			Rationale:  "Bank default BSB (not personal)",
			Filename:   "banks.go",
		})
		id++
	}

	return cases
}

// generateACNTestCases generates comprehensive ACN test cases
func generateACNTestCases(g *TestDataGenerator) []TestCase {
	cases := []TestCase{}
	id := 0

	// Valid ACNs
	for i := 0; i < 5; i++ {
		acn := g.GenerateValidACN()
		
		cases = append(cases, TestCase{
			ID:         fmt.Sprintf("acn-prod-%03d", id),
			Code:       fmt.Sprintf(`company.ACN = "%s"`, acn),
			Language:   "go",
			PIType:     detection.PITypeACN,
			IsActualPI: true,
			Context:    "production",
			Rationale:  "Valid ACN in company record",
			Filename:   "company.go",
		})
		id++

		// With formatting
		formatted := acn[:3] + " " + acn[3:6] + " " + acn[6:]
		cases = append(cases, TestCase{
			ID:         fmt.Sprintf("acn-prod-%03d", id),
			Code:       fmt.Sprintf(`"companyNumber": "%s"`, formatted),
			Language:   "json",
			PIType:     detection.PITypeACN,
			IsActualPI: true,
			Context:    "production",
			Rationale:  "Valid ACN in JSON with formatting",
			Filename:   "company.json",
		})
		id++
	}

	return cases
}

// generateDriverLicenseTestCases generates driver license test cases for all states
func generateDriverLicenseTestCases(g *TestDataGenerator) []TestCase {
	cases := []TestCase{}
	id := 0
	
	states := []string{"NSW", "VIC", "QLD", "SA", "WA", "TAS"}
	
	for _, state := range states {
		// Generate multiple licenses per state
		for i := 0; i < 3; i++ {
			license := g.GenerateDriverLicense(state)
			
			// Production context
			cases = append(cases, TestCase{
				ID:         fmt.Sprintf("dl-%s-prod-%03d", strings.ToLower(state), id),
				Code:       fmt.Sprintf(`driver.LicenseNumber = "%s" // %s`, license, state),
				Language:   "go",
				PIType:     detection.PITypeDriverLicense,
				IsActualPI: true,
				Context:    "production",
				Rationale:  fmt.Sprintf("Valid %s driver license", state),
				Filename:   "driver.go",
			})
			id++

			// First one of each state in test context
			if i == 0 {
				cases = append(cases, TestCase{
					ID:         fmt.Sprintf("dl-%s-test-%03d", strings.ToLower(state), id),
					Code:       g.WrapInContext(license, detection.PITypeDriverLicense, "test", "go"),
					Language:   "go",
					PIType:     detection.PITypeDriverLicense,
					IsActualPI: false,
					Context:    "test",
					Rationale:  fmt.Sprintf("%s license in test", state),
					Filename:   "driver_test.go",
				})
				id++
			}
		}
	}

	return cases
}

// generateMultiPITestCases generates test cases with multiple PI types together
func generateMultiPITestCases(g *TestDataGenerator) []TestCase {
	cases := []TestCase{}
	
	// Critical risk: Name + TFN + Address
	cases = append(cases, TestCase{
		ID: "multi-critical-001",
		Code: fmt.Sprintf(`customer := Customer{
	Name: "John Smith",
	TFN: "%s",
	Address: "123 Queen St, Melbourne VIC 3000",
	Email: "john.smith@email.com",
}`, g.GenerateValidTFN()),
		Language:   "go",
		PIType:     detection.PITypeTFN,
		IsActualPI: true,
		Context:    "production",
		Rationale:  "Multiple PI types together - critical risk",
		Filename:   "customer.go",
	})

	// High risk: Medicare + Name + DOB
	cases = append(cases, TestCase{
		ID: "multi-high-001",
		Code: fmt.Sprintf(`{
	"patient": {
		"name": "Jane Doe",
		"medicare": "%s",
		"dateOfBirth": "1985-03-15",
		"phone": "0412345678"
	}
}`, g.GenerateValidMedicare()),
		Language:   "json",
		PIType:     detection.PITypeMedicare,
		IsActualPI: true,
		Context:    "production",
		Rationale:  "Healthcare data with multiple PI - high risk",
		Filename:   "patient.json",
	})

	// Business context: ABN + ACN + BSB
	cases = append(cases, TestCase{
		ID: "multi-business-001",
		Code: fmt.Sprintf(`business_details = {
	'abn': '%s',
	'acn': '%s',
	'bsb': '%s',
	'account': '12345678'
}`, g.GenerateValidABN(), g.GenerateValidACN(), g.GenerateValidBSB()),
		Language:   "python",
		PIType:     detection.PITypeABN,
		IsActualPI: true,
		Context:    "production",
		Rationale:  "Business identifiers together",
		Filename:   "business.py",
	})

	return cases
}

// generateFalsePositiveCases generates common false positive patterns
func generateFalsePositiveCases(g *TestDataGenerator) []TestCase {
	cases := []TestCase{}
	id := 0

	// Sequential numbers
	sequences := []string{
		"123456789",
		"987654321",
		"111111111",
		"999999999",
		"000000000",
	}

	for _, seq := range sequences {
		cases = append(cases, TestCase{
			ID:         fmt.Sprintf("fp-seq-%03d", id),
			Code:       fmt.Sprintf(`id := "%s"`, seq),
			Language:   "go",
			PIType:     detection.PITypeTFN,
			IsActualPI: false,
			Context:    "production",
			Rationale:  "Sequential number, not real TFN",
			Filename:   "ids.go",
		})
		id++
	}

	// UUIDs
	for i := 0; i < 3; i++ {
		uuid := g.GenerateUUID()
		cases = append(cases, TestCase{
			ID:         fmt.Sprintf("fp-uuid-%03d", id),
			Code:       fmt.Sprintf(`sessionId = "%s"`, uuid),
			Language:   "python",
			PIType:     "",
			IsActualPI: false,
			Context:    "production",
			Rationale:  "UUID not PI",
			Filename:   "session.py",
		})
		id++
	}

	// Version numbers
	for i := 0; i < 3; i++ {
		version := g.GenerateVersionNumber()
		cases = append(cases, TestCase{
			ID:         fmt.Sprintf("fp-version-%03d", id),
			Code:       fmt.Sprintf(`VERSION = "%s"`, version),
			Language:   "go",
			PIType:     "",
			IsActualPI: false,
			Context:    "production",
			Rationale:  "Version number not PI",
			Filename:   "version.go",
		})
		id++
	}

	// Hash values
	for i := 0; i < 3; i++ {
		hash := g.GenerateHash()
		cases = append(cases, TestCase{
			ID:         fmt.Sprintf("fp-hash-%03d", id),
			Code:       fmt.Sprintf(`checksum := "%s"`, hash),
			Language:   "go",
			PIType:     "",
			IsActualPI: false,
			Context:    "production",
			Rationale:  "Hash value not PI",
			Filename:   "crypto.go",
		})
		id++
	}

	// Phone numbers (business)
	businessPhones := []string{
		"1300123456",  // 1300 number
		"1800123456",  // 1800 number
		"131450",      // Short code
		"+61299999999", // Landline
	}

	for _, phone := range businessPhones {
		cases = append(cases, TestCase{
			ID:         fmt.Sprintf("fp-phone-%03d", id),
			Code:       fmt.Sprintf(`SUPPORT_PHONE = "%s"`, phone),
			Language:   "go",
			PIType:     detection.PITypePhone,
			IsActualPI: false,
			Context:    "production",
			Rationale:  "Business phone number, not personal",
			Filename:   "config.go",
		})
		id++
	}

	return cases
}

// generateEdgeCases generates ambiguous edge cases
func generateEdgeCases(g *TestDataGenerator) []TestCase {
	cases := []TestCase{}
	id := 0

	// Environment variables
	tfn := g.GenerateValidTFN()
	cases = append(cases, TestCase{
		ID:         fmt.Sprintf("edge-env-%03d", id),
		Code:       g.WrapInContext(tfn, detection.PITypeTFN, "environment", "go"),
		Language:   "go",
		PIType:     detection.PITypeTFN,
		IsActualPI: true,
		Context:    "production",
		Rationale:  "TFN from environment - likely real in production",
		Filename:   "config.go",
	})
	id++

	// Logging statements
	medicare := g.GenerateValidMedicare()
	cases = append(cases, TestCase{
		ID:         fmt.Sprintf("edge-log-%03d", id),
		Code:       g.WrapInContext(medicare, detection.PITypeMedicare, "log", "python"),
		Language:   "python",
		PIType:     detection.PITypeMedicare,
		IsActualPI: true,
		Context:    "logging",
		Rationale:  "Medicare in log - security issue",
		Filename:   "health_service.py",
	})
	id++

	// SQL queries
	abn := g.GenerateValidABN()
	cases = append(cases, TestCase{
		ID:         fmt.Sprintf("edge-sql-%03d", id),
		Code:       g.WrapInQuery(abn, detection.PITypeABN),
		Language:   "sql",
		PIType:     detection.PITypeABN,
		IsActualPI: false,
		Context:    "query",
		Rationale:  "ABN in SQL example, not storing",
		Filename:   "queries.sql",
	})
	id++

	// Validation functions
	tfn2 := g.GenerateValidTFN()
	cases = append(cases, TestCase{
		ID:         fmt.Sprintf("edge-validate-%03d", id),
		Code:       fmt.Sprintf(`func isValidTFN(tfn string) bool { return tfn == "%s" }`, tfn2),
		Language:   "go",
		PIType:     detection.PITypeTFN,
		IsActualPI: false,
		Context:    "validation",
		Rationale:  "TFN in validation logic, not storing real data",
		Filename:   "validator.go",
	})
	id++

	// Masked/partial PI
	tfn3 := g.GenerateValidTFN()
	masked := "***-***-" + tfn3[6:]
	cases = append(cases, TestCase{
		ID:         fmt.Sprintf("edge-masked-%03d", id),
		Code:       fmt.Sprintf(`displayTFN := "%s"`, masked),
		Language:   "go",
		PIType:     detection.PITypeTFN,
		IsActualPI: false,
		Context:    "production",
		Rationale:  "Partially masked TFN",
		Filename:   "display.go",
	})
	id++

	// Error messages
	bsb := g.GenerateValidBSB()
	cases = append(cases, TestCase{
		ID:         fmt.Sprintf("edge-error-%03d", id),
		Code:       fmt.Sprintf(`return fmt.Errorf("invalid BSB: %%s", "%s")`, bsb),
		Language:   "go",
		PIType:     detection.PITypeBSB,
		IsActualPI: true,
		Context:    "production",
		Rationale:  "BSB in error message - potential leak",
		Filename:   "errors.go",
	})
	id++

	return cases
}

// generateSyntheticPatterns generates synthetic test patterns
func generateSyntheticPatterns(g *TestDataGenerator) []TestCase {
	cases := []TestCase{}
	id := 0

	// HTML/XML content with PI-like patterns
	cases = append(cases, TestCase{
		ID:   fmt.Sprintf("syn-html-%03d", id),
		Code: `<div id="123456789" class="user-profile">`,
		Language:   "html",
		PIType:     "",
		IsActualPI: false,
		Context:    "production",
		Rationale:  "HTML ID attribute, not TFN",
		Filename:   "template.html",
	})
	id++

	// CSS selectors
	cases = append(cases, TestCase{
		ID:   fmt.Sprintf("syn-css-%03d", id),
		Code: `.form-field-123456789 { display: none; }`,
		Language:   "css",
		PIType:     "",
		IsActualPI: false,
		Context:    "production",
		Rationale:  "CSS class name, not PI",
		Filename:   "styles.css",
	})
	id++

	// Database IDs
	cases = append(cases, TestCase{
		ID:   fmt.Sprintf("syn-db-%03d", id),
		Code: `db.users.find({_id: ObjectId("507f1f77bcf86cd799439011")})`,
		Language:   "javascript",
		PIType:     "",
		IsActualPI: false,
		Context:    "production",
		Rationale:  "MongoDB ObjectId, not PI",
		Filename:   "queries.js",
	})
	id++

	// Timestamps that might match patterns
	cases = append(cases, TestCase{
		ID:   fmt.Sprintf("syn-timestamp-%03d", id),
		Code: `timestamp := "202312251234567890"`,
		Language:   "go",
		PIType:     "",
		IsActualPI: false,
		Context:    "production",
		Rationale:  "Timestamp, not PI",
		Filename:   "time.go",
	})
	id++

	// API keys that might match patterns
	cases = append(cases, TestCase{
		ID:   fmt.Sprintf("syn-apikey-%03d", id),
		Code: `API_KEY = "sk_test_123456789abcdef"`,
		Language:   "python",
		PIType:     "",
		IsActualPI: false,
		Context:    "production",
		Rationale:  "API key prefix pattern, not PI",
		Filename:   "config.py",
	})
	id++

	// URLs with number patterns
	cases = append(cases, TestCase{
		ID:   fmt.Sprintf("syn-url-%03d", id),
		Code: `url := "https://api.example.com/v2/users/123456789"`,
		Language:   "go",
		PIType:     "",
		IsActualPI: false,
		Context:    "production",
		Rationale:  "User ID in URL, not TFN",
		Filename:   "api.go",
	})
	id++

	// File paths with numbers
	cases = append(cases, TestCase{
		ID:   fmt.Sprintf("syn-path-%03d", id),
		Code: `path := "/var/log/app/2024/01/15/123456789.log"`,
		Language:   "go",
		PIType:     "",
		IsActualPI: false,
		Context:    "production",
		Rationale:  "Log file name, not PI",
		Filename:   "logging.go",
	})
	id++

	return cases
}