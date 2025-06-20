package benchmark

import (
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/MacAttak/pi-scanner/pkg/detection"
)

// TestDataGenerator generates high-quality test cases for PI detection
type TestDataGenerator struct {
	rand *rand.Rand
}

// NewTestDataGenerator creates a new test data generator
func NewTestDataGenerator() *TestDataGenerator {
	return &TestDataGenerator{
		rand: rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// GenerateValidTFN generates a valid TFN using the correct mod 11 algorithm
// Weights: [1, 4, 3, 7, 5, 8, 6, 9, 10] - official ATO algorithm
func (g *TestDataGenerator) GenerateValidTFN() string {
	weights := []int{1, 4, 3, 7, 5, 8, 6, 9, 10}
	
	// Generate first 8 digits randomly (first digit cannot be 0)
	digits := make([]int, 9)
	digits[0] = 1 + g.rand.Intn(9) // First digit 1-9
	for i := 1; i < 8; i++ {
		digits[i] = g.rand.Intn(10)
	}
	
	// Calculate weighted sum for first 8 digits
	sum := 0
	for i := 0; i < 8; i++ {
		sum += digits[i] * weights[i]
	}
	
	// Find the 9th digit that makes sum divisible by 11
	for d := 0; d <= 9; d++ {
		if (sum + d*weights[8]) % 11 == 0 {
			digits[8] = d
			break
		}
	}
	
	// Convert to string
	tfn := ""
	for _, d := range digits {
		tfn += fmt.Sprintf("%d", d)
	}
	
	return tfn
}

// GenerateInvalidTFN generates an invalid TFN (fails checksum)
func (g *TestDataGenerator) GenerateInvalidTFN() string {
	tfn := g.GenerateValidTFN()
	// Corrupt the last digit
	lastDigit := tfn[8] - '0'
	newDigit := (lastDigit + 1) % 10
	return tfn[:8] + fmt.Sprintf("%d", newDigit)
}

// GenerateValidABN generates a valid ABN using the correct mod 89 algorithm
// Weights: [10, 1, 3, 5, 7, 9, 11, 13, 15, 17, 19] - official ABR algorithm
func (g *TestDataGenerator) GenerateValidABN() string {
	weights := []int{10, 1, 3, 5, 7, 9, 11, 13, 15, 17, 19}
	
	// Generate 10 random digits for positions 2-11
	digits := make([]int, 11)
	for i := 1; i < 11; i++ {
		digits[i] = g.rand.Intn(10)
	}
	
	// Calculate sum for positions 2-11 (digits[1] to digits[10])
	sum := 0
	for i := 1; i < 11; i++ {
		sum += digits[i] * weights[i]
	}
	
	// Find first digit: subtract 1 from first digit, multiply by weight, add to sum
	// Result must be divisible by 89
	for d := 1; d <= 9; d++ {
		if ((d-1)*weights[0] + sum) % 89 == 0 {
			digits[0] = d
			break
		}
	}
	
	// Convert to string
	abn := ""
	for _, d := range digits {
		abn += fmt.Sprintf("%d", d)
	}
	
	return abn
}

// GenerateValidMedicare generates a valid Medicare number using correct algorithm
// Weights: [1, 3, 7, 9, 1, 3, 7, 9] - official Medicare algorithm
// Format: 8 digits + 1 check digit + 1 issue number (optional)
func (g *TestDataGenerator) GenerateValidMedicare() string {
	weights := []int{1, 3, 7, 9, 1, 3, 7, 9}
	
	// First digit must be 2-6 (Medicare requirement)
	digits := make([]int, 10)
	digits[0] = 2 + g.rand.Intn(5) // 2, 3, 4, 5, or 6
	
	// Generate next 7 digits (positions 1-7)
	for i := 1; i < 8; i++ {
		digits[i] = g.rand.Intn(10)
	}
	
	// Calculate check digit (position 8)
	sum := 0
	for i := 0; i < 8; i++ {
		sum += digits[i] * weights[i]
	}
	digits[8] = sum % 10
	
	// Issue number (position 9) - typically 1-9
	digits[9] = 1 + g.rand.Intn(9)
	
	// Convert to string (return 10 digits)
	medicare := ""
	for i := 0; i < 10; i++ {
		medicare += fmt.Sprintf("%d", digits[i])
	}
	
	return medicare
}

// GenerateValidBSB generates a valid BSB code using correct format
// Format: XXY-ZZZ where XX=bank, Y=state, ZZZ=branch
func (g *TestDataGenerator) GenerateValidBSB() string {
	// Major Australian bank prefixes (first two digits)
	bankPrefixes := []string{
		"01", // ANZ
		"03", // Westpac
		"06", // Commonwealth Bank
		"08", // NAB
		"11", // St George
		"12", // Bank of Queensland
		"14", // Bendigo Bank
		"48", // Macquarie Bank
		"63", // Building societies
		"70", // Credit unions
		"73", // Westpac savings
		"76", // Various institutions
		"80", // Cuscal administered
	}
	
	// Pick random bank prefix
	prefix := bankPrefixes[g.rand.Intn(len(bankPrefixes))]
	
	// State digit (third digit) - 0-9 representing different states/territories
	stateDigit := g.rand.Intn(10)
	
	// Random branch code (last 3 digits)
	branch := fmt.Sprintf("%03d", g.rand.Intn(1000))
	
	return prefix + fmt.Sprintf("%d", stateDigit) + branch
}

// GenerateValidACN generates a valid ACN using the correct ASIC algorithm
// Weights: [8, 7, 6, 5, 4, 3, 2, 1] - official ASIC modified modulus 10 algorithm
func (g *TestDataGenerator) GenerateValidACN() string {
	weights := []int{8, 7, 6, 5, 4, 3, 2, 1}
	
	// Generate first 8 digits
	digits := make([]int, 9)
	for i := 0; i < 8; i++ {
		digits[i] = g.rand.Intn(10)
	}
	
	// Calculate weighted sum for first 8 digits
	sum := 0
	for i := 0; i < 8; i++ {
		sum += digits[i] * weights[i]
	}
	
	// Calculate check digit using modified modulus 10
	remainder := sum % 10
	checkDigit := (10 - remainder) % 10
	// If complement equals 10, set it to 0
	if checkDigit == 10 {
		checkDigit = 0
	}
	digits[8] = checkDigit
	
	// Convert to string
	acn := ""
	for _, d := range digits {
		acn += fmt.Sprintf("%d", d)
	}
	
	return acn
}

// GenerateDriverLicense generates driver license numbers for different states
func (g *TestDataGenerator) GenerateDriverLicense(state string) string {
	switch strings.ToUpper(state) {
	case "NSW":
		// NSW: 8 digits
		return fmt.Sprintf("%08d", g.rand.Intn(100000000))
	case "VIC":
		// VIC: 8-10 digits
		length := 8 + g.rand.Intn(3)
		format := fmt.Sprintf("%%0%dd", length)
		return fmt.Sprintf(format, g.rand.Intn(1000000000))
	case "QLD":
		// QLD: 8 digits
		return fmt.Sprintf("%08d", g.rand.Intn(100000000))
	case "SA":
		// SA: Letter followed by 6 digits
		letter := string(rune('A' + g.rand.Intn(26)))
		return letter + fmt.Sprintf("%06d", g.rand.Intn(1000000))
	case "WA":
		// WA: 7 digits
		return fmt.Sprintf("%07d", g.rand.Intn(10000000))
	case "TAS":
		// TAS: 7 digits or 2 letters + 5 digits
		if g.rand.Intn(2) == 0 {
			return fmt.Sprintf("%07d", g.rand.Intn(10000000))
		}
		letters := string(rune('A' + g.rand.Intn(26))) + string(rune('A' + g.rand.Intn(26)))
		return letters + fmt.Sprintf("%05d", g.rand.Intn(100000))
	default:
		// Default: 8 digits
		return fmt.Sprintf("%08d", g.rand.Intn(100000000))
	}
}

// FormatTFN formats a TFN with different delimiters
func (g *TestDataGenerator) FormatTFN(tfn string, style string) string {
	if len(tfn) != 9 {
		return tfn
	}
	
	switch style {
	case "dashes":
		return tfn[:3] + "-" + tfn[3:6] + "-" + tfn[6:]
	case "spaces":
		return tfn[:3] + " " + tfn[3:6] + " " + tfn[6:]
	case "dots":
		return tfn[:3] + "." + tfn[3:6] + "." + tfn[6:]
	default:
		return tfn
	}
}

// FormatABN formats an ABN with different delimiters
func (g *TestDataGenerator) FormatABN(abn string, style string) string {
	if len(abn) != 11 {
		return abn
	}
	
	switch style {
	case "spaces":
		return abn[:2] + " " + abn[2:5] + " " + abn[5:8] + " " + abn[8:]
	case "dashes":
		return abn[:2] + "-" + abn[2:5] + "-" + abn[5:8] + "-" + abn[8:]
	default:
		return abn
	}
}

// GenerateSequentialNumber generates numbers that look like PI but aren't
func (g *TestDataGenerator) GenerateSequentialNumber() string {
	start := g.rand.Intn(900000000) + 100000000
	return fmt.Sprintf("%d", start)
}

// GenerateUUID generates a UUID that might match PI patterns
func (g *TestDataGenerator) GenerateUUID() string {
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		g.rand.Uint32(),
		g.rand.Uint32()&0xffff,
		g.rand.Uint32()&0xffff,
		g.rand.Uint32()&0xffff,
		g.rand.Uint64()&0xffffffffffff)
}

// GenerateHash generates a hash that might match PI patterns
func (g *TestDataGenerator) GenerateHash() string {
	// Generate MD5-like hash (32 hex chars)
	hash := ""
	for i := 0; i < 32; i++ {
		hash += fmt.Sprintf("%x", g.rand.Intn(16))
	}
	return hash
}

// GenerateVersionNumber generates version numbers that might match patterns
func (g *TestDataGenerator) GenerateVersionNumber() string {
	major := g.rand.Intn(10)
	minor := g.rand.Intn(100)
	patch := g.rand.Intn(1000)
	build := g.rand.Intn(10000)
	
	return fmt.Sprintf("%d.%d.%d.%d", major, minor, patch, build)
}

// WrapInContext wraps PI data in various code contexts
func (g *TestDataGenerator) WrapInContext(value string, piType detection.PIType, context string, language string) string {
	switch context {
	case "assignment":
		return g.wrapInAssignment(value, piType, language)
	case "function":
		return g.wrapInFunction(value, piType, language)
	case "struct":
		return g.wrapInStruct(value, piType, language)
	case "comment":
		return g.wrapInComment(value, piType, language)
	case "test":
		return g.wrapInTest(value, piType, language)
	case "mock":
		return g.wrapInMock(value, piType, language)
	case "log":
		return g.wrapInLog(value, piType, language)
	case "config":
		return g.wrapInConfig(value, piType, language)
	case "environment":
		return g.wrapInEnvironment(value, piType, language)
	case "query":
		return g.WrapInQuery(value, piType)
	default:
		return value
	}
}

func (g *TestDataGenerator) wrapInAssignment(value string, piType detection.PIType, language string) string {
	varName := g.getVarName(piType)
	
	switch language {
	case "go":
		return fmt.Sprintf(`%s := "%s"`, varName, value)
	case "python":
		return fmt.Sprintf(`%s = "%s"`, varName, value)
	case "javascript":
		return fmt.Sprintf(`const %s = "%s";`, varName, value)
	case "java":
		return fmt.Sprintf(`String %s = "%s";`, varName, value)
	default:
		return fmt.Sprintf(`%s = "%s"`, varName, value)
	}
}

func (g *TestDataGenerator) wrapInFunction(value string, piType detection.PIType, language string) string {
	funcName := g.getFuncName(piType)
	
	switch language {
	case "go":
		return fmt.Sprintf(`func %s() string { return "%s" }`, funcName, value)
	case "python":
		return fmt.Sprintf(`def %s():\n    return "%s"`, funcName, value)
	case "javascript":
		return fmt.Sprintf(`function %s() { return "%s"; }`, funcName, value)
	default:
		return fmt.Sprintf(`%s() { return "%s"; }`, funcName, value)
	}
}

func (g *TestDataGenerator) wrapInStruct(value string, piType detection.PIType, language string) string {
	fieldName := g.getFieldName(piType)
	
	switch language {
	case "go":
		return fmt.Sprintf(`user := User{\n    %s: "%s",\n}`, fieldName, value)
	case "python":
		return fmt.Sprintf(`user = {\n    "%s": "%s"\n}`, strings.ToLower(fieldName), value)
	case "javascript":
		return fmt.Sprintf(`const user = {\n    %s: "%s"\n};`, strings.ToLower(fieldName), value)
	default:
		return fmt.Sprintf(`{ %s: "%s" }`, fieldName, value)
	}
}

func (g *TestDataGenerator) wrapInComment(value string, piType detection.PIType, language string) string {
	switch language {
	case "python":
		return fmt.Sprintf(`# Example %s: %s`, piType, value)
	case "javascript", "go", "java":
		return fmt.Sprintf(`// Example %s: %s`, piType, value)
	default:
		return fmt.Sprintf(`// %s: %s`, piType, value)
	}
}

func (g *TestDataGenerator) wrapInTest(value string, piType detection.PIType, language string) string {
	switch language {
	case "go":
		return fmt.Sprintf(`func Test%sValidation(t *testing.T) {\n    %s := "%s"\n}`, piType, strings.ToLower(string(piType)), value)
	case "python":
		return fmt.Sprintf(`def test_%s_validation():\n    %s = "%s"`, strings.ToLower(string(piType)), strings.ToLower(string(piType)), value)
	default:
		return fmt.Sprintf(`test("%s validation", () => {\n    const %s = "%s";\n});`, piType, strings.ToLower(string(piType)), value)
	}
}

func (g *TestDataGenerator) wrapInMock(value string, piType detection.PIType, language string) string {
	switch language {
	case "go":
		return fmt.Sprintf(`const MOCK_%s = "%s" // For testing only`, strings.ToUpper(string(piType)), value)
	default:
		return fmt.Sprintf(`MOCK_%s = "%s"`, strings.ToUpper(string(piType)), value)
	}
}

func (g *TestDataGenerator) wrapInLog(value string, piType detection.PIType, language string) string {
	switch language {
	case "go":
		return fmt.Sprintf(`log.Printf("Processing %s: %%s", "%s")`, piType, value)
	case "python":
		return fmt.Sprintf(`logger.info("Processing %s: %s", "%s")`, piType, "%s", value)
	default:
		return fmt.Sprintf(`console.log("Processing %s:", "%s");`, piType, value)
	}
}

func (g *TestDataGenerator) wrapInConfig(value string, piType detection.PIType, language string) string {
	key := strings.ToLower(string(piType))
	
	switch language {
	case "yaml":
		return fmt.Sprintf(`%s: "%s"`, key, value)
	case "json":
		return fmt.Sprintf(`  "%s": "%s"`, key, value)
	default:
		return fmt.Sprintf(`%s = %s`, key, value)
	}
}

func (g *TestDataGenerator) wrapInEnvironment(value string, piType detection.PIType, language string) string {
	envVar := fmt.Sprintf("USER_%s", strings.ToUpper(string(piType)))
	
	switch language {
	case "go":
		return fmt.Sprintf(`os.Getenv("%s") // "%s"`, envVar, value)
	case "python":
		return fmt.Sprintf(`os.environ.get("%s") # "%s"`, envVar, value)
	default:
		return fmt.Sprintf(`process.env.%s // "%s"`, envVar, value)
	}
}

// WrapInQuery wraps PI data in a SQL query context
func (g *TestDataGenerator) WrapInQuery(value string, piType detection.PIType) string {
	table := "users"
	column := strings.ToLower(string(piType))
	
	return fmt.Sprintf(`SELECT * FROM %s WHERE %s = '%s'`, table, column, value)
}

func (g *TestDataGenerator) getVarName(piType detection.PIType) string {
	varNames := map[detection.PIType][]string{
		detection.PITypeTFN:      {"tfn", "taxFileNumber", "userTFN"},
		detection.PITypeABN:      {"abn", "businessNumber", "companyABN"},
		detection.PITypeMedicare: {"medicare", "medicareNumber", "patientMedicare"},
		detection.PITypeBSB:      {"bsb", "bankCode", "branchCode"},
		detection.PITypeACN:      {"acn", "companyNumber", "businessACN"},
	}
	
	if names, ok := varNames[piType]; ok {
		return names[g.rand.Intn(len(names))]
	}
	return strings.ToLower(string(piType))
}

func (g *TestDataGenerator) getFieldName(piType detection.PIType) string {
	fieldNames := map[detection.PIType][]string{
		detection.PITypeTFN:      {"TFN", "TaxFileNumber", "TaxNumber"},
		detection.PITypeABN:      {"ABN", "BusinessNumber", "AustralianBusinessNumber"},
		detection.PITypeMedicare: {"Medicare", "MedicareNumber", "HealthCard"},
		detection.PITypeBSB:      {"BSB", "BankCode", "BranchCode"},
		detection.PITypeACN:      {"ACN", "CompanyNumber", "AustralianCompanyNumber"},
	}
	
	if names, ok := fieldNames[piType]; ok {
		return names[g.rand.Intn(len(names))]
	}
	return string(piType)
}

func (g *TestDataGenerator) getFuncName(piType detection.PIType) string {
	return fmt.Sprintf("get%s", string(piType))
}