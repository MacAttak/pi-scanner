package datasets

import "github.com/MacAttak/pi-scanner/pkg/detection"

// RealWorldSample represents a realistic code sample for testing
type RealWorldSample struct {
	ID          string       `json:"id"`
	Description string       `json:"description"`
	Language    string       `json:"language"`
	Filename    string       `json:"filename"`
	Code        string       `json:"code"`
	ExpectedPIs []ExpectedPI `json:"expected_pis"`
	Context     string       `json:"context"`
	Complexity  string       `json:"complexity"` // simple, medium, complex
	Source      string       `json:"source"`     // github, enterprise, synthetic
}

// ExpectedPI defines what PI should be detected
type ExpectedPI struct {
	Type       detection.PIType `json:"type"`
	Value      string           `json:"value"`
	Line       int              `json:"line"`
	Column     int              `json:"column"`
	Confidence float64          `json:"confidence"`
}

// GetRealWorldSamples returns comprehensive real-world code samples
func GetRealWorldSamples() []RealWorldSample {
	return []RealWorldSample{
		// Government API Integration
		{
			ID:          "gov-api-001",
			Description: "Government service integration with TFN validation",
			Language:    "java",
			Filename:    "TaxFileNumberService.java",
			Code: `package au.gov.services.tax;

import org.springframework.stereotype.Service;

@Service
public class TaxFileNumberService {
    
    private static final String DEFAULT_TFN = "123456782"; // Test TFN
    
    public boolean validateTFN(String tfn) {
        // Remove spaces and validate format
        String cleanTFN = tfn.replaceAll("\\s", "");
        return cleanTFN.matches("\\d{9}") && isValidChecksum(cleanTFN);
    }
    
    private boolean isValidChecksum(String tfn) {
        // TFN checksum validation
        int[] weights = {1, 4, 3, 7, 5, 8, 6, 9, 10};
        int sum = 0;
        for (int i = 0; i < 9; i++) {
            sum += Character.getNumericValue(tfn.charAt(i)) * weights[i];
        }
        return sum % 11 == 0;
    }
}`,
			ExpectedPIs: []ExpectedPI{
				{Type: detection.PITypeTFN, Value: "123456782", Line: 7, Confidence: 0.95},
			},
			Context:    "production",
			Complexity: "medium",
			Source:     "synthetic",
		},

		// Banking Integration
		{
			ID:          "banking-001",
			Description: "Banking system with BSB and account numbers",
			Language:    "scala",
			Filename:    "BankingController.scala",
			Code: `package au.com.bank.controllers

import play.api.mvc._
import play.api.libs.json._

class BankingController @Inject()(cc: ControllerComponents) extends AbstractController(cc) {

  case class BankAccount(
    bsb: String,
    accountNumber: String,
    accountName: String
  )
  
  implicit val bankAccountWrites: Writes[BankAccount] = Json.writes[BankAccount]
  
  def createAccount = Action(parse.json) { request =>
    val defaultAccount = BankAccount(
      bsb = "062-001", // Commonwealth Bank BSB
      accountNumber = "12345678",
      accountName = "John Smith"
    )
    
    // Additional test accounts
    val testAccounts = List(
      BankAccount("013-006", "87654321", "Mary Johnson"), // ANZ
      BankAccount("083-004", "11223344", "David Wilson")  // NAB
    )
    
    Ok(Json.toJson(defaultAccount))
  }
}`,
			ExpectedPIs: []ExpectedPI{
				{Type: detection.PITypeBSB, Value: "062-001", Line: 16, Confidence: 0.95},
				{Type: detection.PITypeName, Value: "John Smith", Line: 18, Confidence: 0.85},
				{Type: detection.PITypeBSB, Value: "013-006", Line: 22, Confidence: 0.95},
				{Type: detection.PITypeName, Value: "Mary Johnson", Line: 22, Confidence: 0.85},
				{Type: detection.PITypeBSB, Value: "083-004", Line: 23, Confidence: 0.95},
				{Type: detection.PITypeName, Value: "David Wilson", Line: 23, Confidence: 0.85},
			},
			Context:    "production",
			Complexity: "complex",
			Source:     "synthetic",
		},

		// Healthcare System
		{
			ID:          "healthcare-001",
			Description: "Healthcare management system with Medicare numbers",
			Language:    "python",
			Filename:    "patient_service.py",
			Code: `from django.db import models
from django.core.validators import RegexValidator

class Patient(models.Model):
    """Patient model for healthcare management"""
    
    medicare_number = models.CharField(
        max_length=10,
        validators=[RegexValidator(r'^\d{10}$', 'Invalid Medicare number')],
        default="2428778132"  # Valid Medicare number for testing
    )
    
    first_name = models.CharField(max_length=50, default="Sarah")
    last_name = models.CharField(max_length=50, default="Connor")
    
    def get_medicare_display(self):
        """Format Medicare number for display"""
        medicare = self.medicare_number
        return f"{medicare[:4]} {medicare[4:9]} {medicare[9]}"
    
    @classmethod 
    def create_test_patient(cls):
        """Create a test patient for development"""
        return cls.objects.create(
            medicare_number="4562817392",  # Another test Medicare
            first_name="Test",
            last_name="Patient"
        )

def validate_medicare_checksum(medicare_str):
    """Validate Medicare number using official algorithm"""
    # Remove spaces
    clean = medicare_str.replace(" ", "")
    
    # Test data
    test_numbers = [
        "2428778132",  # Valid 
        "4562817392",  # Valid
        "1234567890"   # Invalid checksum
    ]
    
    return clean in test_numbers`,
			ExpectedPIs: []ExpectedPI{
				{Type: detection.PITypeMedicare, Value: "2428778132", Line: 10, Confidence: 0.95},
				{Type: detection.PITypeName, Value: "Sarah", Line: 13, Confidence: 0.80},
				{Type: detection.PITypeName, Value: "Connor", Line: 14, Confidence: 0.80},
				{Type: detection.PITypeMedicare, Value: "4562817392", Line: 24, Confidence: 0.95},
				{Type: detection.PITypeMedicare, Value: "2428778132", Line: 35, Confidence: 0.95},
				{Type: detection.PITypeMedicare, Value: "4562817392", Line: 36, Confidence: 0.95},
			},
			Context:    "production",
			Complexity: "complex",
			Source:     "synthetic",
		},

		// Configuration Files
		{
			ID:          "config-001",
			Description: "Configuration file with sensitive data",
			Language:    "yaml",
			Filename:    "application.yml",
			Code: `# Application Configuration
database:
  connection:
    host: localhost
    port: 5432
    username: admin
    password: secret123

# API Keys and Credentials  
external_services:
  tax_office:
    api_key: "to_api_key_12345"
    test_tfn: "123456782"  # Test TFN for development
  
  medicare:
    endpoint: "https://api.medicare.gov.au"
    test_number: "2428778132"  # Test Medicare number
    
# Employee Test Data
test_users:
  - name: "Jennifer Wilson"
    email: "j.wilson@company.com.au" 
    phone: "0412345678"
    tfn: "987654321"  # Invalid TFN for testing
  
  - name: "Michael Brown"
    email: "m.brown@company.com.au"
    phone: "0498765432" 
    medicare: "4562817392"

# Bank Details for Testing
test_bank_accounts:
  primary:
    bsb: "062-001"
    account: "12345678"
    name: "Test Account Holder"
  
  secondary:
    bsb: "013-006" 
    account: "87654321"
    name: "Secondary Test Account"`,
			ExpectedPIs: []ExpectedPI{
				{Type: detection.PITypeTFN, Value: "123456782", Line: 12, Confidence: 0.90},
				{Type: detection.PITypeMedicare, Value: "2428778132", Line: 16, Confidence: 0.90},
				{Type: detection.PITypeName, Value: "Jennifer Wilson", Line: 20, Confidence: 0.85},
				{Type: detection.PITypeEmail, Value: "j.wilson@company.com.au", Line: 21, Confidence: 0.90},
				{Type: detection.PITypePhone, Value: "0412345678", Line: 22, Confidence: 0.85},
				{Type: detection.PITypeName, Value: "Michael Brown", Line: 25, Confidence: 0.85},
				{Type: detection.PITypeEmail, Value: "m.brown@company.com.au", Line: 26, Confidence: 0.90},
				{Type: detection.PITypePhone, Value: "0498765432", Line: 27, Confidence: 0.85},
				{Type: detection.PITypeMedicare, Value: "4562817392", Line: 28, Confidence: 0.90},
				{Type: detection.PITypeBSB, Value: "062-001", Line: 32, Confidence: 0.95},
				{Type: detection.PITypeName, Value: "Test Account Holder", Line: 34, Confidence: 0.80},
				{Type: detection.PITypeBSB, Value: "013-006", Line: 37, Confidence: 0.95},
				{Type: detection.PITypeName, Value: "Secondary Test Account", Line: 39, Confidence: 0.80},
			},
			Context:    "production",
			Complexity: "complex",
			Source:     "enterprise",
		},

		// Unit Test File (should have lower detection)
		{
			ID:          "unittest-001",
			Description: "Unit test file with mock data",
			Language:    "java",
			Filename:    "TaxServiceTest.java",
			Code: `package au.gov.services.tax;

import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.BeforeEach;
import static org.mockito.Mockito.*;
import static org.junit.jupiter.api.Assertions.*;

class TaxServiceTest {
    
    private TaxService taxService;
    
    @BeforeEach
    void setUp() {
        taxService = new TaxService();
    }
    
    @Test
    void testValidTFNValidation() {
        // Test with valid TFN
        String validTFN = "123456782"; // Mock TFN for testing
        assertTrue(taxService.validateTFN(validTFN));
    }
    
    @Test 
    void testInvalidTFNValidation() {
        // Test with invalid TFN
        String invalidTFN = "000000000"; // Invalid mock TFN
        assertFalse(taxService.validateTFN(invalidTFN));
    }
    
    @Test
    void testTFNFormatting() {
        // Test TFN formatting
        String testTFN = "987654321";
        String formatted = taxService.formatTFN(testTFN);
        assertEquals("987 654 321", formatted);
    }
    
    @Test
    void testMedicareLookup() {
        // Mock Medicare service test
        String testMedicare = "2428778132"; // Test Medicare number
        Patient mockPatient = mock(Patient.class);
        when(mockPatient.getMedicareNumber()).thenReturn(testMedicare);
        
        assertEquals(testMedicare, mockPatient.getMedicareNumber());
    }
}`,
			ExpectedPIs: []ExpectedPI{
				// Test files should have reduced detection due to context filtering
			},
			Context:    "test",
			Complexity: "medium",
			Source:     "synthetic",
		},

		// Log Analysis Sample
		{
			ID:          "logs-001",
			Description: "Application log file with leaked PI",
			Language:    "text",
			Filename:    "application.log",
			Code: `2024-01-15 10:30:45 INFO  [PaymentService] Processing payment for TFN: 123456782
2024-01-15 10:30:46 DEBUG [ValidationService] Validating account BSB: 062-001, Account: 12345678
2024-01-15 10:30:47 ERROR [PatientService] Failed to lookup Medicare: 2428778132 for patient John Smith
2024-01-15 10:30:48 WARN  [SecurityService] Suspicious activity from IP: 192.168.1.100
2024-01-15 10:30:49 INFO  [EmailService] Sending notification to: patient@example.com.au
2024-01-15 10:30:50 DEBUG [UserService] User login: sarah.connor@company.com phone: 0412345678
2024-01-15 10:30:51 ERROR [CompanyService] ABN validation failed for: 51824753556
2024-01-15 10:30:52 INFO  [SystemHealth] Database connection established
2024-01-15 10:30:53 WARN  [AuditService] Access attempt with invalid TFN: 000000001
2024-01-15 10:30:54 DEBUG [BankingService] Processing BSB: 013-006 for transfer`,
			ExpectedPIs: []ExpectedPI{
				{Type: detection.PITypeTFN, Value: "123456782", Line: 1, Confidence: 0.95},
				{Type: detection.PITypeBSB, Value: "062-001", Line: 2, Confidence: 0.95},
				{Type: detection.PITypeMedicare, Value: "2428778132", Line: 3, Confidence: 0.95},
				{Type: detection.PITypeName, Value: "John Smith", Line: 3, Confidence: 0.85},
				{Type: detection.PITypeIP, Value: "192.168.1.100", Line: 4, Confidence: 0.80},
				{Type: detection.PITypeEmail, Value: "patient@example.com.au", Line: 5, Confidence: 0.90},
				{Type: detection.PITypeEmail, Value: "sarah.connor@company.com", Line: 6, Confidence: 0.90},
				{Type: detection.PITypePhone, Value: "0412345678", Line: 6, Confidence: 0.85},
				{Type: detection.PITypeABN, Value: "51824753556", Line: 7, Confidence: 0.95},
				{Type: detection.PITypeBSB, Value: "013-006", Line: 10, Confidence: 0.95},
			},
			Context:    "logging",
			Complexity: "medium",
			Source:     "enterprise",
		},
	}
}

// GetSamplesByComplexity returns samples filtered by complexity level
func GetSamplesByComplexity(complexity string) []RealWorldSample {
	samples := GetRealWorldSamples()
	var filtered []RealWorldSample

	for _, sample := range samples {
		if sample.Complexity == complexity {
			filtered = append(filtered, sample)
		}
	}

	return filtered
}

// GetSamplesByLanguage returns samples filtered by programming language
func GetSamplesByLanguage(language string) []RealWorldSample {
	samples := GetRealWorldSamples()
	var filtered []RealWorldSample

	for _, sample := range samples {
		if sample.Language == language {
			filtered = append(filtered, sample)
		}
	}

	return filtered
}

// GetSamplesByContext returns samples filtered by context
func GetSamplesByContext(context string) []RealWorldSample {
	samples := GetRealWorldSamples()
	var filtered []RealWorldSample

	for _, sample := range samples {
		if sample.Context == context {
			filtered = append(filtered, sample)
		}
	}

	return filtered
}
