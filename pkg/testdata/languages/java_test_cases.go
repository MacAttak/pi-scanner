package languages

import "github.com/MacAttak/pi-scanner/pkg/detection"

// JavaTestCases returns comprehensive test cases for Java code scanning
func JavaTestCases() []MultiLanguageTestCase {
	return []MultiLanguageTestCase{
		// TRUE POSITIVES - Real PI in Java code
		{
			ID:         "java-tfn-001",
			Language:   "java",
			Filename:   "UserService.java",
			Code:       `public class UserService {\n    private String DEFAULT_TFN = "123456782"; // Valid TFN\n}`,
			ExpectedPI: true,
			PIType:     detection.PITypeTFN,
			Context:    "production",
			Rationale:  "Hardcoded valid TFN in Java production code",
		},
		{
			ID:         "java-medicare-001",
			Language:   "java",
			Filename:   "PatientController.java",
			Code:       `@RestController\npublic class PatientController {\n    @PostMapping("/patient")\n    public Patient createPatient(@RequestBody PatientDto dto) {\n        patient.setMedicareNumber("2428778132");\n        return patient;\n    }\n}`,
			ExpectedPI: true,
			PIType:     detection.PITypeMedicare,
			Context:    "production",
			Rationale:  "Valid Medicare number in Spring Boot controller",
		},
		{
			ID:         "java-abn-001",
			Language:   "java",
			Filename:   "CompanyEntity.java",
			Code:       `@Entity\n@Table(name = "companies")\npublic class Company {\n    @Column(name = "abn")\n    private String abn = "51824753556"; // Commonwealth Bank ABN\n}`,
			ExpectedPI: true,
			PIType:     detection.PITypeABN,
			Context:    "production",
			Rationale:  "Valid ABN in JPA entity",
		},
		{
			ID:         "java-bsb-001",
			Language:   "java",
			Filename:   "BankingService.java",
			Code:       `public class BankingService {\n    private String CBA_BSB = "062-001"; // Commonwealth Bank BSB\n    private String ANZ_BSB = "013-006"; // ANZ Bank BSB\n}`,
			ExpectedPI: true,
			PIType:     detection.PITypeBSB,
			Context:    "production",
			Rationale:  "Valid BSB numbers in banking service",
		},
		{
			ID:         "java-acn-001",
			Language:   "java",
			Filename:   "CompanyController.java",
			Code:       `@RestController\npublic class CompanyController {\n    @GetMapping("/company")\n    public Company getCompany() {\n        // ACN: 123456780\n        return companyService.findByACN("123456780");\n    }\n}`,
			ExpectedPI: true,
			PIType:     detection.PITypeACN,
			Context:    "production",
			Rationale:  "Valid ACN in Spring controller with context",
		},

		// FALSE POSITIVES - Code constructs that look like names
		{
			ID:         "java-false-name-001",
			Language:   "java",
			Filename:   "UserService.java",
			Code:       `public class UserService {\n    private DataProcessor processor;\n    private HttpClient client;\n}`,
			ExpectedPI: false,
			PIType:     detection.PITypeName,
			Context:    "production",
			Rationale:  "Class and variable names, not person names",
		},
		{
			ID:         "java-false-name-002",
			Language:   "java",
			Filename:   "SecurityConfig.java",
			Code:       `@Configuration\npublic class SecurityConfig {\n    public AuthenticationManager authManager() {\n        return new CustomAuthManager();\n    }\n}`,
			ExpectedPI: false,
			PIType:     detection.PITypeName,
			Context:    "production",
			Rationale:  "Spring configuration class and method names",
		},
		{
			ID:         "java-false-name-003",
			Language:   "java",
			Filename:   "PaymentService.java",
			Code:       `public class PaymentService {\n    private RestTemplate restTemplate;\n    private JsonParser jsonParser;\n    private ErrorHandler errorHandler;\n}`,
			ExpectedPI: false,
			PIType:     detection.PITypeName,
			Context:    "production",
			Rationale:  "Technical component names, not person names",
		},

		// TEST CONTEXTS - Valid PI but in test files (should be filtered)
		{
			ID:         "java-test-tfn-001",
			Language:   "java",
			Filename:   "UserServiceTest.java",
			Code:       `@Test\npublic void testValidateTFN() {\n    String testTFN = "123456782";\n    assertTrue(validator.isValid(testTFN));\n}`,
			ExpectedPI: false,
			PIType:     detection.PITypeTFN,
			Context:    "test",
			Rationale:  "Valid TFN but in test file context",
		},
		{
			ID:         "java-mock-medicare-001",
			Language:   "java",
			Filename:   "TestDataFactory.java",
			Code:       `public class TestDataFactory {\n    public static final String MOCK_MEDICARE = "2428778132";\n    public static Patient createTestPatient() {\n        return new Patient(MOCK_MEDICARE);\n    }\n}`,
			ExpectedPI: false,
			PIType:     detection.PITypeMedicare,
			Context:    "test",
			Rationale:  "Mock data for testing, not production PI",
		},

		// EDGE CASES
		{
			ID:         "java-annotation-pi-001",
			Language:   "java",
			Filename:   "ValidationTest.java",
			Code:       `@ParameterizedTest\n@ValueSource(strings = {"123456782", "876543217"})\npublic void testTFNValidation(String tfn) {\n    assertTrue(TFNValidator.isValid(tfn));\n}`,
			ExpectedPI: false,
			PIType:     detection.PITypeTFN,
			Context:    "test",
			Rationale:  "Test data in annotations, not production usage",
		},
		{
			ID:         "java-comment-pi-001",
			Language:   "java",
			Filename:   "UserService.java",
			Code:       `public class UserService {\n    // Example TFN format: 123456782\n    public boolean validateTFN(String tfn) {\n        return tfn.matches("\\\\d{9}");\n    }\n}`,
			ExpectedPI: false,
			PIType:     detection.PITypeTFN,
			Context:    "documentation",
			Rationale:  "TFN in comment for documentation purposes",
		},

		// LOGGING CONCERNS
		{
			ID:         "java-logging-pi-001",
			Language:   "java",
			Filename:   "AuditService.java",
			Code:       `public class AuditService {\n    private static final Logger logger = LoggerFactory.getLogger(AuditService.class);\n    \n    public void logUserAction(String userId, String tfn) {\n        logger.info("User {} accessed TFN: {}", userId, tfn); // TFN: 123456782\n    }\n}`,
			ExpectedPI: true,
			PIType:     detection.PITypeTFN,
			Context:    "logging",
			Rationale:  "TFN being logged - security risk even if example",
		},

		// MULTI-PI SCENARIOS
		{
			ID:         "java-multi-pi-001",
			Language:   "java",
			Filename:   "CustomerDto.java",
			Code:       `public class CustomerDto {\n    private String fullName = "John Smith";\n    private String tfn = "123456782";\n    private String medicare = "2428778132";\n    private String email = "john.smith@example.com";\n    private String address = "123 Collins St, Melbourne VIC 3000";\n}`,
			ExpectedPI: true,
			PIType:     detection.PITypeTFN, // Multiple types, use primary
			Context:    "production",
			Rationale:  "Multiple PI types in DTO - critical risk",
		},
	}
}

// MultiLanguageTestCase represents a test case for multi-language support
type MultiLanguageTestCase struct {
	ID         string           `json:"id"`
	Language   string           `json:"language"`
	Filename   string           `json:"filename"`
	Code       string           `json:"code"`
	ExpectedPI bool             `json:"expected_pi"`
	PIType     detection.PIType `json:"pi_type"`
	Context    string           `json:"context"`
	Rationale  string           `json:"rationale"`
}
