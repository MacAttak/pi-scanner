package benchmark

import (
	"github.com/MacAttak/pi-scanner/pkg/detection"
)

// GenerateAustralianPITestCases creates a comprehensive set of test cases for Australian PI detection
func GenerateAustralianPITestCases() *BenchmarkDataset {
	dataset := &BenchmarkDataset{}

	// TRUE POSITIVES - Actual PI in production code
	dataset.TruePositives = []TestCase{
		{
			ID:         "au-tfn-001",
			Code:       `const DEFAULT_TFN = "123456782"`, // Valid TFN
			Language:   "go",
			PIType:     detection.PITypeTFN,
			IsActualPI: true,
			Context:    "production",
			Rationale:  "Hardcoded valid TFN in production constant",
			Filename:   "config.go",
		},
		{
			ID:         "au-tfn-002",
			Code:       `user.TaxFileNumber = "876543217"`, // Valid TFN
			Language:   "go",
			PIType:     detection.PITypeTFN,
			IsActualPI: true,
			Context:    "production",
			Rationale:  "Valid TFN assigned to user object",
			Filename:   "user.go",
		},
		{
			ID:         "au-abn-001",
			Code:       `company := &Company{ABN: "51824753556"}`, // Valid ABN
			Language:   "go",
			PIType:     detection.PITypeABN,
			IsActualPI: true,
			Context:    "production",
			Rationale:  "Valid ABN assigned to company",
			Filename:   "company.go",
		},
		{
			ID:         "au-medicare-001",
			Code:       `patient.MedicareNumber = "2428778132"`, // Valid Medicare
			Language:   "go",
			PIType:     detection.PITypeMedicare,
			IsActualPI: true,
			Context:    "production",
			Rationale:  "Valid Medicare number assigned to patient",
			Filename:   "patient.go",
		},
		{
			ID:         "au-email-001",
			Code:       `adminEmail := "admin@example.com"`,
			Language:   "go",
			PIType:     detection.PITypeEmail,
			IsActualPI: true,
			Context:    "production",
			Rationale:  "Email address hardcoded in production",
			Filename:   "config.go",
		},
	}

	// FALSE POSITIVES - Look like PI but aren't
	dataset.TrueNegatives = []TestCase{
		{
			ID:         "au-tfn-003",
			Code:       `// Example TFN: 123456782`,
			Language:   "go",
			PIType:     detection.PITypeTFN,
			IsActualPI: false,
			Context:    "comment",
			Rationale:  "TFN in comment for documentation",
			Filename:   "README.go",
		},
		{
			ID:         "au-tfn-004",
			Code:       `func TestTFNValidation(t *testing.T) { tfn := "123456782" }`,
			Language:   "go",
			PIType:     detection.PITypeTFN,
			IsActualPI: false,
			Context:    "test",
			Rationale:  "Valid TFN but in test file",
			Filename:   "user_test.go",
		},
		{
			ID:         "au-tfn-005",
			Code:       `// Replace with your TFN: 123456782`,
			Language:   "go",
			PIType:     detection.PITypeTFN,
			IsActualPI: false,
			Context:    "comment",
			Rationale:  "TFN in comment as placeholder",
			Filename:   "example.go",
		},
		{
			ID:         "au-medicare-002",
			Code:       `func validateMedicare(num string) bool { return num == "2428778132" }`, // Valid Medicare in validation
			Language:   "go",
			PIType:     detection.PITypeMedicare,
			IsActualPI: false,
			Context:    "validation",
			Rationale:  "Medicare number in validation function - not storing real data",
			Filename:   "validator.go",
		},
		{
			ID:         "au-abn-002",
			Code:       `const MOCK_ABN = "51824753556" // For testing only`,
			Language:   "go",
			PIType:     detection.PITypeABN,
			IsActualPI: false,
			Context:    "mock",
			Rationale:  "ABN clearly marked as mock data",
			Filename:   "mock_data.go",
		},
		{
			ID:         "au-email-002",
			Code:       `testEmail := "test@example.com" // For unit tests`,
			Language:   "go",
			PIType:     detection.PITypeEmail,
			IsActualPI: false,
			Context:    "test",
			Rationale:  "Test email address, not real PI",
			Filename:   "email_test.go",
		},
	}

	// EDGE CASES - Ambiguous situations
	dataset.EdgeCases = []TestCase{
		{
			ID:         "au-edge-001",
			Code:       `input := os.Getenv("USER_TFN") // "123456782"`,
			Language:   "go",
			PIType:     detection.PITypeTFN,
			IsActualPI: true, // Environment variable could contain real PI
			Context:    "production",
			Rationale:  "TFN from environment - likely real in production",
			Filename:   "config.go",
		},
		{
			ID:         "au-edge-002",
			Code:       `log.Printf("Processing TFN: %s", tfn) // tfn = "123456782"`,
			Language:   "go",
			PIType:     detection.PITypeTFN,
			IsActualPI: true, // Logging real PI is a security issue
			Context:    "logging",
			Rationale:  "TFN being logged - security issue even if not stored",
			Filename:   "processor.go",
		},
		{
			ID:         "au-edge-003",
			Code:       `SELECT * FROM users WHERE medicare_number = '2428778132'`,
			Language:   "sql",
			PIType:     detection.PITypeMedicare,
			IsActualPI: false, // Query example, not storing
			Context:    "query",
			Rationale:  "SQL query example - not storing real data",
			Filename:   "schema.sql",
		},
	}

	// CRITICAL RISK - Multiple PI types together
	dataset.TruePositives = append(dataset.TruePositives, TestCase{
		ID:   "au-multi-001",
		Code: `
customer := Customer{
	Name: "John Smith",
	TFN: "123456782",
	Medicare: "2428778132",
	Address: "123 Queen St, Melbourne",
	Email: "john.smith@email.com",
}`,
		Language:   "go",
		PIType:     detection.PITypeTFN, // Multiple types, using TFN as primary
		IsActualPI: true,
		Context:    "production",
		Rationale:  "Multiple PI types together = critical risk",
		Filename:   "customer.go",
	})

	// SYNTHETIC - Generated patterns
	dataset.Synthetic = []TestCase{
		{
			ID:         "syn-sequential-001",
			Code:       `id := "123456789" // Sequential numbers`,
			Language:   "go",
			PIType:     detection.PITypeTFN,
			IsActualPI: false,
			Context:    "production",
			Rationale:  "Sequential numbers, not real TFN",
			Filename:   "generator.go",
		},
		{
			ID:         "syn-uuid-001",
			Code:       `uuid := "123e4567-e89b-12d3-a456-426614174000"`,
			Language:   "go",
			PIType:     detection.PITypeEmail, // UUID might match email pattern
			IsActualPI: false,
			Context:    "production",
			Rationale:  "UUID, not email address",
			Filename:   "uuid.go",
		},
		{
			ID:         "syn-hash-001",
			Code:       `hash := "5f4dcc3b5aa765d61d8327deb882cf99" // password hash`,
			Language:   "go",
			PIType:     detection.PITypeTFN, // Hash might match TFN pattern
			IsActualPI: false,
			Context:    "production",
			Rationale:  "Hash value, not TFN",
			Filename:   "auth.go",
		},
	}

	return dataset
}

// GenerateExtendedTestCases creates additional test cases for comprehensive evaluation
func GenerateExtendedTestCases() []TestCase {
	return []TestCase{
		// Names in different contexts
		{
			ID:         "ext-name-001",
			Code:       `author := "John Smith" // Code author`,
			Language:   "go",
			PIType:     detection.PITypeName,
			IsActualPI: false,
			Context:    "comment",
			Rationale:  "Author name in comment, not personal data",
			Filename:   "main.go",
		},
		{
			ID:         "ext-name-002",
			Code:       `customer.FullName = firstName + " " + lastName`,
			Language:   "go",
			PIType:     detection.PITypeName,
			IsActualPI: true,
			Context:    "production",
			Rationale:  "Actual customer name being processed",
			Filename:   "customer.go",
		},
		
		// Phone numbers
		{
			ID:         "ext-phone-001",
			Code:       `supportPhone := "+61 2 8123 4567" // Customer service`,
			Language:   "go",
			PIType:     detection.PITypePhone,
			IsActualPI: false,
			Context:    "production",
			Rationale:  "Business phone number, not personal",
			Filename:   "config.go",
		},
		{
			ID:         "ext-phone-002",
			Code:       `user.MobileNumber = "0412345678"`,
			Language:   "go",
			PIType:     detection.PITypePhone,
			IsActualPI: true,
			Context:    "production",
			Rationale:  "Personal mobile number",
			Filename:   "user.go",
		},

		// BSB codes
		{
			ID:         "ext-bsb-001",
			Code:       `defaultBSB := "063000" // Commonwealth Bank`,
			Language:   "go",
			PIType:     detection.PITypeBSB,
			IsActualPI: false,
			Context:    "production",
			Rationale:  "Default BSB for bank, not customer specific",
			Filename:   "banking.go",
		},
		{
			ID:         "ext-bsb-002",
			Code:       `account.BSB = userInput.BSB // "062-000"`,
			Language:   "go",
			PIType:     detection.PITypeBSB,
			IsActualPI: true,
			Context:    "production",
			Rationale:  "User's actual BSB code",
			Filename:   "account.go",
		},
	}
}