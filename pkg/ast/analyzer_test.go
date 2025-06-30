package ast

import (
	"context"
	"strings"
	"testing"
)

func TestAnalyzer_DetectLanguage(t *testing.T) {
	analyzer := NewAnalyzer(nil)

	tests := []struct {
		filePath string
		expected Language
	}{
		{"Customer.java", LanguageJava},
		{"Account.scala", LanguageScala},
		{"payment_processor.py", LanguagePython},
		{"main.go", LanguageGo},
		{"unknown.txt", Language("")},
	}

	for _, test := range tests {
		result := analyzer.detectLanguage(test.filePath)
		if result != test.expected {
			t.Errorf("detectLanguage(%s) = %s, expected %s", test.filePath, result, test.expected)
		}
	}
}

func TestAnalyzer_DetermineRiskLevel(t *testing.T) {
	analyzer := NewAnalyzer(DefaultConfig())

	tests := []struct {
		filePath string
		expected RiskLevel
	}{
		// High risk patterns
		{"src/main/java/model/Customer.java", RiskLevelHigh},
		{"src/entity/Account.scala", RiskLevelHigh},
		{"payment/Transaction.py", RiskLevelHigh},

		// Medium risk patterns
		{"src/service/PaymentService.java", RiskLevelMedium},
		{"controller/AccountController.scala", RiskLevelMedium},
		{"api/customer_api.py", RiskLevelMedium},

		// Low risk patterns
		{"src/util/StringUtils.java", RiskLevelLow},
		{"helper/DateHelper.scala", RiskLevelLow},
		{"config/app_config.py", RiskLevelLow},

		// Excluded patterns
		{"src/test/java/CustomerTest.java", RiskLevelIgnore},
		{"test/AccountSpec.scala", RiskLevelIgnore},
		{"tests/test_payment.py", RiskLevelIgnore},

		// Unknown patterns default to medium
		{"src/unknown/SomeFile.java", RiskLevelMedium},
	}

	for _, test := range tests {
		result := analyzer.DetermineRiskLevel(test.filePath)
		if result != test.expected {
			t.Errorf("determineRiskLevel(%s) = %s, expected %s", test.filePath, result, test.expected)
		}
	}
}

func TestAnalyzer_DetermineRiskZone(t *testing.T) {
	analyzer := NewAnalyzer(DefaultConfig())

	tests := []struct {
		filePath string
		expected string
	}{
		{"src/customer/Customer.java", "customer_data"},
		{"payment/PaymentProcessor.scala", "financial_data"},
		{"auth/AuthService.py", "authentication"},
		{"service/BusinessLogic.java", "business_logic"},
		{"test/CustomerTest.java", "tests"},
		{"util/Helper.scala", "utilities"},
		{"src/Unknown.py", "general"},
	}

	for _, test := range tests {
		result := analyzer.DetermineRiskZone(test.filePath)
		if result != test.expected {
			t.Errorf("determineRiskZone(%s) = %s, expected %s", test.filePath, result, test.expected)
		}
	}
}

func TestAnalyzer_JavaAnalysis(t *testing.T) {
	analyzer := NewAnalyzer(DefaultConfig())

	javaCode := `
package com.bank.customer;

import java.util.List;
import javax.persistence.Entity;

@Entity
@Table(name = "customers")
public class Customer {
    private String customerId;
    private String firstName;
    private String lastName;
    private String tfn; // Tax File Number

    public Customer() {}

    @Override
    public String toString() {
        return "Customer{id=" + customerId + "}";
    }

    public void setPassword(String password) {
        this.password = "hardcoded123"; // Security issue
    }
}
`

	result, err := analyzer.AnalyzeFile(context.Background(), "Customer.java", []byte(javaCode))
	if err != nil {
		t.Fatalf("AnalyzeFile failed: %v", err)
	}

	// Check basic properties
	if result.Language != LanguageJava {
		t.Errorf("Expected language %s, got %s", LanguageJava, result.Language)
	}

	if result.RiskLevel != RiskLevelMedium {
		t.Errorf("Expected risk level %s, got %s", RiskLevelMedium, result.RiskLevel)
	}

	// Check parsed structure
	if len(result.CodeStructure.Classes) != 1 {
		t.Errorf("Expected 1 class, got %d", len(result.CodeStructure.Classes))
	}

	if result.CodeStructure.Classes[0].Name != "Customer" {
		t.Errorf("Expected class name 'Customer', got '%s'", result.CodeStructure.Classes[0].Name)
	}

	// Check imports
	if len(result.CodeStructure.Imports) < 1 {
		t.Errorf("Expected at least 1 import, got %d", len(result.CodeStructure.Imports))
	}

	// Check security hints
	if len(result.SecurityHints) < 1 {
		t.Errorf("Expected at least 1 security hint for hardcoded password, got %d", len(result.SecurityHints))
	}
}

func TestAnalyzer_ScalaAnalysis(t *testing.T) {
	analyzer := NewAnalyzer(DefaultConfig())

	scalaCode := `
package com.bank.payment

import scala.concurrent.Future
import javax.persistence.Entity

@Entity
case class Payment(
  id: String,
  amount: BigDecimal,
  accountNumber: String
) {
  val apiKey = "sk_test_12345" // Hardcoded API key

  def processPayment(amount: BigDecimal): Future[Boolean] = {
    // Payment processing logic
    Future.successful(true)
  }
}

object PaymentService {
  def validatePayment(payment: Payment): Boolean = {
    payment.amount > 0
  }
}
`

	result, err := analyzer.AnalyzeFile(context.Background(), "Payment.scala", []byte(scalaCode))
	if err != nil {
		t.Fatalf("AnalyzeFile failed: %v", err)
	}

	// Check basic properties
	if result.Language != LanguageScala {
		t.Errorf("Expected language %s, got %s", LanguageScala, result.Language)
	}

	// Check parsed structure
	if len(result.CodeStructure.Classes) < 1 {
		t.Errorf("Expected at least 1 class/object, got %d", len(result.CodeStructure.Classes))
	}

	// Check security hints for hardcoded API key
	found := false
	for _, hint := range result.SecurityHints {
		if strings.Contains(hint.Description, "API key") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected security hint for hardcoded API key")
	}
}

func TestAnalyzer_PythonAnalysis(t *testing.T) {
	analyzer := NewAnalyzer(DefaultConfig())

	pythonCode := `
import os
from typing import Optional
from flask import Flask, request

class CustomerService:
    API_KEY = "secret_key_123"  # Hardcoded secret

    def __init__(self):
        self.db_password = "admin123"  # Another hardcoded secret

    def authenticate_user(self, username: str, password: str) -> bool:
        # Dangerous eval usage
        query = f"SELECT * FROM users WHERE username='{username}'"
        return eval(f"check_user('{username}')")

    @property
    def connection_string(self) -> str:
        return f"postgresql://user:{self.db_password}@localhost/db"

def process_payment(amount):
    os.system(f"payment_processor --amount {amount}")  # Command injection risk
    return True
`

	result, err := analyzer.AnalyzeFile(context.Background(), "customer_service.py", []byte(pythonCode))
	if err != nil {
		t.Fatalf("AnalyzeFile failed: %v", err)
	}

	// Check basic properties
	if result.Language != LanguagePython {
		t.Errorf("Expected language %s, got %s", LanguagePython, result.Language)
	}

	// Check parsed structure
	if len(result.CodeStructure.Classes) < 1 {
		t.Errorf("Expected at least 1 class, got %d", len(result.CodeStructure.Classes))
	}

	// Check security hints - should find multiple issues
	t.Logf("Found %d security hints", len(result.SecurityHints))
	for i, hint := range result.SecurityHints {
		t.Logf("Security hint %d: %s (line %d, severity %s)", i+1, hint.Description, hint.LineNumber, hint.Severity)
	}

	// Check for expected security findings
	expectedFindings := map[string]bool{
		"secret_or_api": false, // Should find either secret or API key (API_KEY contains secret)
		"password":      false,
		"eval":          false,
		"system":        false,
	}

	for _, hint := range result.SecurityHints {
		desc := strings.ToLower(hint.Description)
		if strings.Contains(desc, "password") {
			expectedFindings["password"] = true
		}
		if strings.Contains(desc, "eval") {
			expectedFindings["eval"] = true
		}
		if strings.Contains(desc, "system") {
			expectedFindings["system"] = true
		}
		if strings.Contains(desc, "secret") || strings.Contains(desc, "api") {
			expectedFindings["secret_or_api"] = true
		}
	}

	for finding, found := range expectedFindings {
		if !found {
			t.Errorf("Expected security finding for '%s' not found. Available hints: %v", finding, result.SecurityHints)
		}
	}
}

func TestBankingDomainConfig(t *testing.T) {
	config := DefaultBankingDomainConfig()

	// Test that banking domain patterns are present
	if len(config.HighRiskPatterns) == 0 {
		t.Error("Expected high risk patterns to be configured")
	}

	if len(config.RiskZones) == 0 {
		t.Error("Expected risk zones to be configured")
	}

	// Check specific banking patterns
	expectedRiskZones := []string{"customer_data", "financial_data", "payment_processing", "authentication"}
	for _, zone := range expectedRiskZones {
		if _, exists := config.RiskZones[zone]; !exists {
			t.Errorf("Expected risk zone '%s' to be configured", zone)
		}
	}
}

func TestAnalyzer_LanguageSupport(t *testing.T) {
	config := DefaultConfig()
	analyzer := NewAnalyzer(config)

	// Test enabled languages
	enabledLanguages := []Language{LanguageJava, LanguageScala, LanguagePython}
	for _, lang := range enabledLanguages {
		if !analyzer.isLanguageEnabled(lang) {
			t.Errorf("Language %s should be enabled by default", lang)
		}
	}

	// Test that Go is not enabled by default for banking domain
	if analyzer.isLanguageEnabled(LanguageGo) {
		t.Error("Go language should not be enabled by default for banking domain")
	}
}

func TestAnalyzer_SecurityHintSeverity(t *testing.T) {
	analyzer := NewAnalyzer(DefaultConfig())

	// Test Java security hints
	javaCodeWithSecurityIssues := `
public class SecurityTest {
    private String password = "admin123";
    private String apiKey = "sk_12345";

    public void executeQuery(String input) {
        Statement stmt = connection.createStatement();
        stmt.execute("SELECT * FROM users WHERE id = " + input);
    }
}
`

	result, err := analyzer.AnalyzeFile(context.Background(), "SecurityService.java", []byte(javaCodeWithSecurityIssues))
	if err != nil {
		t.Fatalf("AnalyzeFile failed: %v", err)
	}

	// Debug output
	t.Logf("Found %d security hints", len(result.SecurityHints))
	for i, hint := range result.SecurityHints {
		t.Logf("Security hint %d: %s (line %d, severity %s)", i+1, hint.Description, hint.LineNumber, hint.Severity)
	}

	// Check that high severity issues are properly identified
	highSeverityCount := 0
	for _, hint := range result.SecurityHints {
		if hint.Severity == "HIGH" {
			highSeverityCount++
		}
	}

	if highSeverityCount == 0 {
		t.Error("Expected at least one HIGH severity security hint")
	}
}
