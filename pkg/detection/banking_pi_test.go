package detection

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBankingPIDetection tests banking-specific PI patterns
func TestBankingPIDetection(t *testing.T) {
	detector := NewDetector()
	ctx := context.Background()

	testCases := []struct {
		name     string
		category string
		tests    []struct {
			name        string
			content     string
			shouldFind  bool
			expectedPI  string
			description string
		}
	}{
		{
			name:     "BSB (Bank State Branch)",
			category: "Australian Banking",
			tests: []struct {
				name        string
				content     string
				shouldFind  bool
				expectedPI  string
				description string
			}{
				{
					name:        "Standard BSB format",
					content:     "Transfer to BSB: 012-345",
					shouldFind:  true,
					expectedPI:  "012-345",
					description: "Valid BSB with hyphen",
				},
				{
					name:        "BSB without hyphen",
					content:     "Bank BSB 012345",
					shouldFind:  true,
					expectedPI:  "012345",
					description: "Valid BSB without hyphen",
				},
				{
					name:        "BSB in transaction",
					content:     "Payment processed: BSB 062-000 Account 12345678",
					shouldFind:  true,
					expectedPI:  "062-000",
					description: "BSB in payment context",
				},
				{
					name:        "Invalid BSB - too short",
					content:     "Code: 01234",
					shouldFind:  false,
					expectedPI:  "",
					description: "5 digits is not a valid BSB",
				},
			},
		},
		{
			name:     "Bank Account Numbers",
			category: "Australian Banking",
			tests: []struct {
				name        string
				content     string
				shouldFind  bool
				expectedPI  string
				description string
			}{
				{
					name:        "Standard account with BSB",
					content:     "Account: 012-345 12345678",
					shouldFind:  true,
					expectedPI:  "012-345 12345678",
					description: "BSB and account together",
				},
				{
					name:        "Account number only",
					content:     "Account number: 12345678",
					shouldFind:  true,
					expectedPI:  "12345678",
					description: "8-digit account number",
				},
				{
					name:        "Short account number",
					content:     "Acc: 123456",
					shouldFind:  true,
					expectedPI:  "123456",
					description: "6-digit account number",
				},
			},
		},
		{
			name:     "Credit Card Numbers",
			category: "International Banking",
			tests: []struct {
				name        string
				content     string
				shouldFind  bool
				expectedPI  string
				description string
			}{
				{
					name:        "Visa card",
					content:     "Card: 4111111111111111",
					shouldFind:  true,
					expectedPI:  "4111111111111111",
					description: "Valid Visa test card",
				},
				{
					name:        "MasterCard",
					content:     "Payment card: 5105105105105100",
					shouldFind:  true,
					expectedPI:  "5105105105105100",
					description: "Valid MasterCard test card",
				},
				{
					name:        "Card with spaces",
					content:     "CC: 4111 1111 1111 1111",
					shouldFind:  true,
					expectedPI:  "4111 1111 1111 1111",
					description: "Card number with spaces",
				},
				{
					name:        "Card with dashes",
					content:     "Number: 4111-1111-1111-1111",
					shouldFind:  true,
					expectedPI:  "4111-1111-1111-1111",
					description: "Card number with dashes",
				},
				{
					name:        "Invalid card - bad checksum",
					content:     "Card: 4111111111111112",
					shouldFind:  false,
					expectedPI:  "",
					description: "Invalid Luhn checksum",
				},
			},
		},
		{
			name:     "SWIFT/BIC Codes",
			category: "International Banking",
			tests: []struct {
				name        string
				content     string
				shouldFind  bool
				expectedPI  string
				description string
			}{
				{
					name:        "Standard SWIFT code",
					content:     "SWIFT: ANZBNZ22",
					shouldFind:  false, // Changed to false since we removed SWIFT detection
					expectedPI:  "ANZBNZ22",
					description: "8-character SWIFT code (removed due to false positives)",
				},
				{
					name:        "SWIFT with branch",
					content:     "BIC: ANZBNZ22MEL",
					shouldFind:  false, // Changed to false since we removed SWIFT detection
					expectedPI:  "ANZBNZ22MEL",
					description: "11-character SWIFT code (removed due to false positives)",
				},
			},
		},
		{
			name:     "IBAN (International Bank Account Number)",
			category: "International Banking",
			tests: []struct {
				name        string
				content     string
				shouldFind  bool
				expectedPI  string
				description string
			}{
				{
					name:        "German IBAN",
					content:     "IBAN: DE89370400440532013000",
					shouldFind:  true,
					expectedPI:  "DE89370400440532013000",
					description: "Valid German IBAN",
				},
				{
					name:        "UK IBAN",
					content:     "Account: GB82WEST12345698765432",
					shouldFind:  true,
					expectedPI:  "GB82WEST12345698765432",
					description: "Valid UK IBAN",
				},
				{
					name:        "IBAN with spaces",
					content:     "IBAN: GB82 WEST 1234 5698 7654 32",
					shouldFind:  true,
					expectedPI:  "GB82 WEST 1234 5698 7654 32",
					description: "IBAN with standard spacing",
				},
			},
		},
		{
			name:     "Routing Numbers",
			category: "US Banking",
			tests: []struct {
				name        string
				content     string
				shouldFind  bool
				expectedPI  string
				description string
			}{
				{
					name:        "US routing number",
					content:     "Routing: 021000021",
					shouldFind:  true,
					expectedPI:  "021000021",
					description: "Valid US routing number",
				},
				{
					name:        "Routing with account",
					content:     "Bank: 021000021 Account: 123456789",
					shouldFind:  true,
					expectedPI:  "021000021",
					description: "Routing number in context",
				},
			},
		},
	}

	for _, category := range testCases {
		t.Run(category.name, func(t *testing.T) {
			for _, test := range category.tests {
				t.Run(test.name, func(t *testing.T) {
					findings, err := detector.Detect(ctx, []byte(test.content), "test.txt")
					require.NoError(t, err)

					if test.shouldFind {
						require.NotEmpty(t, findings, "Expected to find %s but got no findings", test.description)

						// Check if the expected PI was found
						found := false
						for _, finding := range findings {
							if finding.Match == test.expectedPI {
								found = true
								break
							}
						}
						assert.True(t, found, "Expected to find '%s' in findings but didn't. Got: %v", test.expectedPI, findings)
					} else {
						// For items that shouldn't be found, check they're not detected
						for _, finding := range findings {
							assert.NotEqual(t, test.expectedPI, finding.Match, "Should not have detected '%s' (%s)", test.expectedPI, test.description)
						}
					}
				})
			}
		})
	}
}

// TestBankingContextualDetection tests banking PI detection with context
//
// DEBUG FINDINGS:
// 1. BSB detection works but risk level is LOW (weight=50) instead of expected HIGH
// 2. Bank account numbers like "12345678" are NOT detected without context keywords
//   - Pattern requires "account", "acct", etc. before the number
//   - Standalone numbers in variables like TEST_ACCOUNT = "12345678" won't match
//
// 3. Context validation was filtering some findings when enabled
// 4. Risk weights: BSB=50 (LOW), BankAccount=70 (MEDIUM) per DefaultConfig
func TestBankingContextualDetection(t *testing.T) {
	// Create detector with custom config to help debug
	config := DefaultConfig()
	config.EnableContextValidation = false // Disable to see all raw findings
	detector := NewDetectorWithConfig(config)
	ctx := context.Background()

	testCases := []struct {
		name        string
		content     string
		filename    string
		expectHigh  []string // PI that should be high risk
		expectLow   []string // PI that should be low risk
		description string
	}{
		{
			name: "Production payment service",
			content: `
package com.bank.payment

public class PaymentService {
    private static final String DEFAULT_BSB = "012-345"; // BSB: 012-345
    private static final String TEST_ACCOUNT = "12345678";

    public void processPayment(String bsb, String account) {
        // Process real payments
        transfer(bsb, account);
    }
}`,
			filename:    "src/main/java/com/bank/payment/PaymentService.java",
			expectHigh:  []string{"012-345", "12345678"},
			expectLow:   []string{},
			description: "Production code should be high risk",
		},
		{
			name: "Test payment data",
			content: `
public class PaymentServiceTest {
    @Test
    public void testPaymentProcessing() {
        String testBsb = "012-345"; // BSB: 012-345
        String testAccount = "12345678";

        // Test with mock data
        service.processPayment(testBsb, testAccount);
    }
}`,
			filename:    "src/test/java/PaymentServiceTest.java",
			expectHigh:  []string{},
			expectLow:   []string{"012-345", "12345678"},
			description: "Test files should be low risk",
		},
		{
			name: "Configuration file",
			content: `
# Payment Gateway Configuration
gateway.url=https://api.paymentgateway.com
gateway.merchant.id=MERCHANT123
gateway.api.key=test_key_example_12345
# BSB: 062-000
default.bsb=062-000
test.account=87654321
`,
			filename:    "config/payment-gateway.properties",
			expectHigh:  []string{"062-000", "87654321"},
			expectLow:   []string{},
			description: "Config files with real-looking data should be high risk",
		},
		{
			name: "SQL migration with banking data",
			content: `
-- Create sample accounts for testing
INSERT INTO accounts (bsb, account_number, balance) VALUES
    ('012-345', '12345678', 1000.00), -- BSB: 012-345
    ('062-000', '87654321', 2500.00), -- BSB: 062-000
    ('123-456', '11111111', 0.00); -- BSB: 123-456
`,
			filename:    "migrations/V001__create_test_accounts.sql",
			expectHigh:  []string{"012-345", "062-000", "123-456"},
			expectLow:   []string{},
			description: "SQL files with PI should be high risk",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			findings, err := detector.Detect(ctx, []byte(tc.content), tc.filename)
			require.NoError(t, err)

			// Debug logging
			t.Logf("Test case: %s", tc.name)
			t.Logf("Filename: %s", tc.filename)
			t.Logf("Total findings: %d", len(findings))
			for i, finding := range findings {
				t.Logf("Finding %d: Type=%s, Match=%s, RiskLevel=%s, Confidence=%f, ContextModifier=%f",
					i+1, finding.Type, finding.Match, finding.RiskLevel, finding.Confidence, finding.ContextModifier)
			}

			// Group findings by risk level
			// Use maps to deduplicate matches (same value can appear multiple times)
			highRiskMap := make(map[string]bool)
			lowRiskMap := make(map[string]bool)

			for _, finding := range findings {
				if finding.RiskLevel == RiskLevelHigh || finding.RiskLevel == RiskLevelCritical {
					highRiskMap[finding.Match] = true
				} else if finding.RiskLevel == RiskLevelLow {
					lowRiskMap[finding.Match] = true
				}
			}

			// Convert maps to slices for comparison
			highRiskFindings := []string{}
			for match := range highRiskMap {
				highRiskFindings = append(highRiskFindings, match)
			}
			lowRiskFindings := []string{}
			for match := range lowRiskMap {
				lowRiskFindings = append(lowRiskFindings, match)
			}

			// Debug: Show what was actually categorized
			t.Logf("High risk findings: %v", highRiskFindings)
			t.Logf("Low risk findings: %v", lowRiskFindings)
			t.Logf("Expected high risk: %v", tc.expectHigh)
			t.Logf("Expected low risk: %v", tc.expectLow)

			// Check high risk expectations
			for _, expected := range tc.expectHigh {
				assert.Contains(t, highRiskFindings, expected,
					"%s: Expected '%s' to be high risk", tc.description, expected)
			}

			// Check low risk expectations
			for _, expected := range tc.expectLow {
				assert.Contains(t, lowRiskFindings, expected,
					"%s: Expected '%s' to be low risk", tc.description, expected)
			}
		})
	}
}

// TestBankingFalsePositives tests that common banking false positives are not detected
func TestBankingFalsePositives(t *testing.T) {
	detector := NewDetector()
	ctx := context.Background()

	falsePositives := []struct {
		name        string
		content     string
		description string
	}{
		{
			name:        "Transaction IDs",
			content:     "Transaction ID: 012345678901234567890",
			description: "Long transaction IDs should not be detected as cards",
		},
		{
			name:        "Order numbers",
			content:     "Order #4111111111111111 processed",
			description: "Order numbers matching card patterns",
		},
		{
			name:        "Version numbers",
			content:     "Version: 5.1.0.5105105100",
			description: "Version numbers with card-like patterns",
		},
		{
			name:        "Database IDs",
			content:     "account_id: 123456789012",
			description: "Database IDs should not be bank accounts",
		},
		{
			name:        "Timestamps",
			content:     "Created: 20240123456789",
			description: "Timestamps should not be detected",
		},
		{
			name:        "Hash values",
			content:     "checksum: 0123456789abcdef",
			description: "Hash values should not be detected",
		},
		{
			name:        "Sequential test data",
			content:     "Test accounts: 111111, 222222, 333333",
			description: "Obviously synthetic data",
		},
	}

	for _, fp := range falsePositives {
		t.Run(fp.name, func(t *testing.T) {
			findings, err := detector.Detect(ctx, []byte(fp.content), "test.txt")
			require.NoError(t, err)

			// Check for specific banking PI types
			for _, finding := range findings {
				switch finding.Type {
				case PITypeBankAccount, PITypeCreditCard, PITypeBSB:
					t.Errorf("%s: Incorrectly detected %s as %s",
						fp.description, finding.Match, finding.Type)
				}
			}
		})
	}
}
