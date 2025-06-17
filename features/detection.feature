Feature: Australian PI Detection
  As a security engineer
  I want to detect Australian personally identifiable information
  So that I can ensure compliance with AU banking regulations

  Background:
    Given the scanner is configured for Australian PI detection

  Scenario: Detect valid Tax File Number
    Given a file "test.go" containing:
      """
      const customerTFN = "123456789"
      """
    When I scan the file
    Then a TFN finding should be detected
    And the finding should have risk level "HIGH"
    And the finding should pass TFN checksum validation

  Scenario: Detect Tax File Number with formatting
    Given a file "config.yaml" containing:
      """
      default_tfn: "123-456-789"
      """
    When I scan the file
    Then a TFN finding should be detected
    And the matched text should be normalized to "123456789"

  Scenario: Detect Medicare number
    Given a file "data.json" containing:
      """
      {
        "medicare": "2123456701"
      }
      """
    When I scan the file
    Then a Medicare finding should be detected
    And the finding should have risk level "HIGH"
    And the finding should pass Medicare checksum validation

  Scenario: Detect Australian Business Number
    Given a file "vendor.txt" containing:
      """
      Company ABN: 51824753556
      """
    When I scan the file
    Then an ABN finding should be detected
    And the finding should have risk level "MEDIUM"
    And the finding should pass ABN modulus 89 validation

  Scenario: Detect Bank State Branch code
    Given a file "payment.go" containing:
      """
      bsb := "062-001"
      account := "12345678"
      """
    When I scan the file
    Then a BSB finding should be detected
    And a bank account finding should be detected
    And the combined risk level should be "HIGH"

  Scenario: Critical risk for multiple PI in proximity
    Given a file "customer.go" containing:
      """
      type Customer struct {
        Name     string // "John Smith"
        TFN      string // "123456789"
        Address  string // "123 Main St, Sydney"
        Account  string // "12345678"
      }
      """
    When I scan the file
    Then multiple PI findings should be detected
    And the findings should be marked as co-occurring
    And the risk level should be "CRITICAL"
    And the risk reason should include "Multiple high-risk PI in proximity"

  Scenario: Suppress test data in test files
    Given a file "customer_test.go" containing:
      """
      func TestCustomer() {
        testTFN := "123456789"
        mockAddress := "123 Test Street"
      }
      """
    When I scan the file
    Then PI findings should be detected
    But the risk level should be "LOW"
    And the context should indicate "test file"

  Scenario: Detect synthetic data patterns
    Given a file "mock_data.go" containing:
      """
      customers := []Customer{
        {Name: "Test User 1", TFN: "111111111"},
        {Name: "Test User 2", TFN: "222222222"},
        {Name: "Test User 3", TFN: "333333333"},
      }
      """
    When I scan the file
    Then PI findings should be detected
    But they should be flagged as "likely synthetic"
    And the risk level should be reduced

  Scenario: ML validation reduces false positives
    Given a file "comments.go" containing:
      """
      // TFN format is 9 digits like 123456789
      // Example: medicare number 2123456701
      """
    When I scan the file with ML validation enabled
    Then the ML model should analyze the context
    And the confidence should be low due to comment context
    And the findings should be marked as "documentation"

  Scenario: Validate driver's license formats
    Given a file "license.txt" containing:
      """
      NSW License: 12345678
      VIC License: 123456789
      Invalid: ABC123
      """
    When I scan the file
    Then 2 driver's license findings should be detected
    And "ABC123" should not be detected as a valid license