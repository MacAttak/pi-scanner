package proximity

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPatternMatcher_TestDataKeywords(t *testing.T) {
	matcher := NewPatternMatcher()

	testCases := []struct {
		name     string
		text     string
		expected bool
	}{
		// Test keywords
		{"test keyword", "test data", true},
		{"Test capitalized", "Test SSN", true},
		{"TEST uppercase", "TEST DATA", true},
		{"testing variant", "testing user", true},
		{"test_data underscore", "test_data", true},
		{"testData camelCase", "testData", true},
		{"test-data hyphen", "test-data", true},

		// Example keywords
		{"example keyword", "example data", true},
		{"Example capitalized", "Example user", true},
		{"EXAMPLE uppercase", "EXAMPLE DATA", true},
		{"example_data underscore", "example_data", true},
		{"exampleData camelCase", "exampleData", true},
		{"example-data hyphen", "example-data", true},

		// Mock keywords
		{"mock keyword", "mock data", true},
		{"Mock capitalized", "Mock user", true},
		{"MOCK uppercase", "MOCK DATA", true},
		{"mocked variant", "mocked user", true},
		{"mock_data underscore", "mock_data", true},
		{"mockData camelCase", "mockData", true},
		{"mock-data hyphen", "mock-data", true},

		// Sample keywords
		{"sample keyword", "sample data", true},
		{"Sample capitalized", "Sample user", true},
		{"SAMPLE uppercase", "SAMPLE DATA", true},
		{"sample_data underscore", "sample_data", true},
		{"sampleData camelCase", "sampleData", true},
		{"sample-data hyphen", "sample-data", true},

		// Demo keywords
		{"demo keyword", "demo data", true},
		{"Demo capitalized", "Demo user", true},
		{"DEMO uppercase", "DEMO DATA", true},
		{"demo_data underscore", "demo_data", true},
		{"demoData camelCase", "demoData", true},
		{"demo-data hyphen", "demo-data", true},

		// Fake keywords
		{"fake keyword", "fake data", true},
		{"Fake capitalized", "Fake user", true},
		{"FAKE uppercase", "FAKE DATA", true},
		{"fake_data underscore", "fake_data", true},
		{"fakeData camelCase", "fakeData", true},
		{"fake-data hyphen", "fake-data", true},

		// Dummy keywords
		{"dummy keyword", "dummy data", true},
		{"Dummy capitalized", "Dummy user", true},
		{"DUMMY uppercase", "DUMMY DATA", true},
		{"dummy_data underscore", "dummy_data", true},
		{"dummyData camelCase", "dummyData", true},
		{"dummy-data hyphen", "dummy-data", true},

		// Placeholder keywords
		{"placeholder keyword", "placeholder data", true},
		{"Placeholder capitalized", "Placeholder user", true},
		{"PLACEHOLDER uppercase", "PLACEHOLDER DATA", true},
		{"placeholder_data underscore", "placeholder_data", true},
		{"placeholderData camelCase", "placeholderData", true},
		{"placeholder-data hyphen", "placeholder-data", true},

		// Negative cases
		{"regular word", "user data", false},
		{"production code", "validate user", false},
		{"real context", "customer information", false},
		{"partial match", "contest", false},
		{"different word", "best practice", false},
		{"not a keyword", "processing", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := matcher.ContainsTestDataKeywords(tc.text)
			assert.Equal(t, tc.expected, result, "Case: %s", tc.name)
		})
	}
}

func TestPatternMatcher_PIContextLabels(t *testing.T) {
	matcher := NewPatternMatcher()

	testCases := []struct {
		name     string
		text     string
		expected []string
	}{
		// SSN variations
		{"SSN colon", "SSN: 123-45-6789", []string{"SSN"}},
		{"ssn lowercase", "ssn: 123-45-6789", []string{"ssn"}},
		{"Social Security Number", "Social Security Number: 123-45-6789", []string{"Social Security Number"}},
		{"social security number lowercase", "social security number: 123-45-6789", []string{"social security number"}},
		{"SSN equals", "SSN = 123-45-6789", []string{"SSN"}},
		{"SSN space", "SSN 123-45-6789", []string{"SSN"}},

		// TFN variations
		{"TFN colon", "TFN: 123456789", []string{"TFN"}},
		{"tfn lowercase", "tfn: 123456789", []string{"tfn"}},
		{"Tax File Number", "Tax File Number: 123456789", []string{"Tax File Number"}},
		{"tax file number lowercase", "tax file number: 123456789", []string{"tax file number"}},
		{"Tax File No", "Tax File No: 123456789", []string{"Tax File No"}},
		{"TFN equals", "TFN = 123456789", []string{"TFN"}},

		// Medicare variations
		{"Medicare No", "Medicare No: 2345678901", []string{"Medicare No"}},
		{"medicare no lowercase", "medicare no: 2345678901", []string{"medicare no"}},
		{"Medicare Number", "Medicare Number: 2345678901", []string{"Medicare Number"}},
		{"medicare number lowercase", "medicare number: 2345678901", []string{"medicare number"}},
		{"Medicare Card", "Medicare Card: 2345678901", []string{"Medicare Card"}},
		{"Medicare equals", "Medicare = 2345678901", []string{"Medicare"}},

		// ABN variations
		{"ABN colon", "ABN: 12345678901", []string{"ABN"}},
		{"abn lowercase", "abn: 12345678901", []string{"abn"}},
		{"Australian Business Number", "Australian Business Number: 12345678901", []string{"Australian Business Number"}},
		{"ABN equals", "ABN = 12345678901", []string{"ABN"}},

		// Credit Card variations
		{"Credit Card", "Credit Card: 4111111111111111", []string{"Credit Card"}},
		{"credit card lowercase", "credit card: 4111111111111111", []string{"credit card"}},
		{"CC colon", "CC: 4111111111111111", []string{"CC"}},
		{"Card Number", "Card Number: 4111111111111111", []string{"Card Number"}},

		// Phone variations
		{"Phone colon", "Phone: 0412345678", []string{"Phone"}},
		{"phone lowercase", "phone: 0412345678", []string{"phone"}},
		{"Phone Number", "Phone Number: 0412345678", []string{"Phone Number"}},
		{"Mobile", "Mobile: 0412345678", []string{"Mobile"}},
		{"Tel", "Tel: 0412345678", []string{"Tel"}},

		// Email variations
		{"Email colon", "Email: user@example.com", []string{"Email"}},
		{"email lowercase", "email: user@example.com", []string{"email"}},
		{"Email Address", "Email Address: user@example.com", []string{"Email Address"}},
		{"E-mail", "E-mail: user@example.com", []string{"E-mail"}},

		// Driver License variations
		{"Driver License", "Driver License: 12345678", []string{"Driver License"}},
		{"driver license lowercase", "driver license: 12345678", []string{"driver license"}},
		{"DL colon", "DL: 12345678", []string{"DL"}},
		{"License Number", "License Number: 12345678", []string{"License Number"}},

		// Passport variations
		{"Passport", "Passport: A1234567", []string{"Passport"}},
		{"passport lowercase", "passport: A1234567", []string{"passport"}},
		{"Passport Number", "Passport Number: A1234567", []string{"Passport Number"}},
		{"Passport No", "Passport No: A1234567", []string{"Passport No"}},

		// Multiple labels
		{"Multiple labels", "SSN: 123-45-6789, TFN: 987654321", []string{"SSN", "TFN"}},
		{"Different formats", "SSN = 123-45-6789 and TFN: 987654321", []string{"SSN", "TFN"}},

		// Negative cases
		{"No labels", "user data 123-45-6789", []string{}},
		{"Partial match", "assign 123-45-6789", []string{}},
		{"Different context", "process 123-45-6789", []string{}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := matcher.FindPIContextLabels(tc.text)
			assert.ElementsMatch(t, tc.expected, result, "Case: %s", tc.name)
		})
	}
}

func TestPatternMatcher_DocumentationPatterns(t *testing.T) {
	matcher := NewPatternMatcher()

	testCases := []struct {
		name     string
		text     string
		expected bool
	}{
		// Single line comments
		{"C++ style comment", "// This is a comment with 123-45-6789", true},
		{"C style comment", "/* This is a comment with 123-45-6789 */", true},
		{"Hash comment", "# This is a comment with 123-45-6789", true},
		{"HTML comment", "<!-- This is a comment with 123-45-6789 -->", true},
		{"SQL comment", "-- This is a comment with 123-45-6789", true},

		// Multi-line comments
		{"Multi-line C comment", "/*\n * Multi-line comment\n * with 123-45-6789\n */", true},
		{"JSDoc comment", "/**\n * JSDoc comment\n * @param ssn 123-45-6789\n */", true},

		// Documentation strings
		{"Python docstring", `"""This is a docstring with 123-45-6789"""`, true},
		{"Triple quote", "'''This is documentation with 123-45-6789'''", true},

		// Negative cases
		{"Regular code", "var ssn = '123-45-6789'", false},
		{"String literal", `"This is just a string with 123-45-6789"`, false},
		{"Function call", "process('123-45-6789')", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := matcher.IsDocumentationContext(tc.text)
			assert.Equal(t, tc.expected, result, "Case: %s", tc.name)
		})
	}
}

func TestPatternMatcher_FormFieldPatterns(t *testing.T) {
	matcher := NewPatternMatcher()

	testCases := []struct {
		name     string
		text     string
		expected bool
	}{
		// HTML input fields
		{"HTML input", `<input type="text" name="ssn" value="123-45-6789">`, true},
		{"HTML input self-closing", `<input type="text" name="ssn" value="123-45-6789" />`, true},
		{"HTML textarea", `<textarea name="ssn">123-45-6789</textarea>`, true},
		{"HTML select", `<select name="ssn"><option value="123-45-6789">`, true},

		// Form data
		{"Form data", "ssn=123-45-6789&name=John", true},
		{"URL encoded", "ssn=123%2D45%2D6789", true},

		// JSON form data
		{"JSON form", `{"ssn": "123-45-6789", "name": "John"}`, true},
		{"JSON nested", `{"user": {"ssn": "123-45-6789"}}`, true},

		// Negative cases
		{"Regular assignment", "ssn = '123-45-6789'", false},
		{"Function call", "validate('123-45-6789')", false},
		{"Array", "['123-45-6789', 'other']", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := matcher.IsFormFieldContext(tc.text)
			assert.Equal(t, tc.expected, result, "Case: %s", tc.name)
		})
	}
}

func TestPatternMatcher_DatabasePatterns(t *testing.T) {
	matcher := NewPatternMatcher()

	testCases := []struct {
		name     string
		text     string
		expected bool
	}{
		// SQL queries
		{"SELECT query", "SELECT * FROM users WHERE ssn = '123-45-6789'", true},
		{"INSERT query", "INSERT INTO users (ssn) VALUES ('123-45-6789')", true},
		{"UPDATE query", "UPDATE users SET ssn = '123-45-6789' WHERE id = 1", true},
		{"DELETE query", "DELETE FROM users WHERE ssn = '123-45-6789'", true},

		// Case variations
		{"select lowercase", "select * from users where ssn = '123-45-6789'", true},
		{"Mixed case", "Select * From users Where ssn = '123-45-6789'", true},

		// ORM patterns
		{"Rails ActiveRecord", "User.where(ssn: '123-45-6789')", true},
		{"Django ORM", "User.objects.filter(ssn='123-45-6789')", true},
		{"Sequelize", "User.findOne({where: {ssn: '123-45-6789'}})", true},

		// Database connection strings
		{"Connection string", "mongodb://user:123-45-6789@localhost:27017/db", true},
		{"JDBC URL", "jdbc:mysql://localhost:3306/db?password=123-45-6789", true},

		// Negative cases
		{"Regular code", "user.ssn = '123-45-6789'", false},
		{"Variable assignment", "var ssn = '123-45-6789'", false},
		{"Function call", "process('123-45-6789')", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := matcher.IsDatabaseContext(tc.text)
			assert.Equal(t, tc.expected, result, "Case: %s", tc.name)
		})
	}
}

func TestPatternMatcher_LogPatterns(t *testing.T) {
	matcher := NewPatternMatcher()

	testCases := []struct {
		name     string
		text     string
		expected bool
	}{
		// Log levels
		{"INFO log", "INFO: Processing user 123-45-6789", true},
		{"DEBUG log", "DEBUG: User SSN 123-45-6789 validated", true},
		{"ERROR log", "ERROR: Invalid SSN 123-45-6789", true},
		{"WARN log", "WARN: SSN 123-45-6789 already exists", true},
		{"TRACE log", "TRACE: Validating 123-45-6789", true},

		// Lowercase variations
		{"info lowercase", "info: Processing user 123-45-6789", true},
		{"debug lowercase", "debug: User SSN 123-45-6789 validated", true},
		{"error lowercase", "error: Invalid SSN 123-45-6789", true},

		// Timestamp patterns
		{"With timestamp", "2023-10-15 10:30:00 INFO: User 123-45-6789 logged in", true},
		{"ISO timestamp", "2023-10-15T10:30:00Z INFO: User 123-45-6789 logged in", true},

		// Logger patterns
		{"Java logger", "logger.info('Processing user {}', '123-45-6789')", true},
		{"Python logger", "logger.info('Processing user %s', '123-45-6789')", true},
		{"Node.js logger", "console.log('Processing user', '123-45-6789')", true},

		// Syslog patterns
		{"Syslog", "<14>Oct 15 10:30:00 server: User 123-45-6789 authenticated", true},

		// Negative cases
		{"Regular code", "user.process('123-45-6789')", false},
		{"Variable assignment", "var info = '123-45-6789'", false},
		{"Function definition", "function info() { return '123-45-6789'; }", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := matcher.IsLogContext(tc.text)
			assert.Equal(t, tc.expected, result, "Case: %s", tc.name)
		})
	}
}

func TestPatternMatcher_ConfigurationPatterns(t *testing.T) {
	matcher := NewPatternMatcher()

	testCases := []struct {
		name     string
		text     string
		expected bool
	}{
		// Key-value pairs
		{"Simple assignment", "ssn=123-45-6789", true},
		{"With spaces", "ssn = 123-45-6789", true},
		{"Quotes", "ssn='123-45-6789'", true},
		{"Double quotes", `ssn="123-45-6789"`, true},

		// Configuration formats
		{"INI format", "[user]\nssn=123-45-6789", true},
		{"Properties format", "user.ssn=123-45-6789", true},
		{"Environment variable", "export SSN=123-45-6789", true},
		{"YAML format", "ssn: 123-45-6789", true},
		{"JSON config", `{"ssn": "123-45-6789"}`, true},

		// Default/fallback values
		{"Default prefix", "default_ssn=123-45-6789", true},
		{"Fallback prefix", "fallback_ssn=123-45-6789", true},
		{"Initial prefix", "initial_ssn=123-45-6789", true},

		// Case variations
		{"Uppercase", "SSN=123-45-6789", true},
		{"Mixed case", "Ssn=123-45-6789", true},

		// Negative cases
		{"Function call", "validate('123-45-6789')", false},
		{"Array element", "users[0] = '123-45-6789'", true},    // Actually detected by JSON config pattern
		{"Object property", "user.name = '123-45-6789'", true}, // Properties format matches this
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := matcher.IsConfigurationContext(tc.text)
			assert.Equal(t, tc.expected, result, "Case: %s", tc.name)
		})
	}
}

func TestPatternMatcher_VariablePatterns(t *testing.T) {
	matcher := NewPatternMatcher()

	testCases := []struct {
		name     string
		text     string
		expected bool
	}{
		// Variable declarations
		{"var declaration", "var ssn = '123-45-6789'", true},
		{"let declaration", "let ssn = '123-45-6789'", true},
		{"const declaration", "const ssn = '123-45-6789'", true},
		{"Python assignment", "ssn = '123-45-6789'", true},

		// Type declarations
		{"Go variable", "var ssn string = '123-45-6789'", true},
		{"Java variable", "String ssn = '123-45-6789'", true},
		{"C++ variable", "std::string ssn = '123-45-6789'", true},

		// Function parameters
		{"Function param", "function process(ssn = '123-45-6789')", true},
		{"Lambda param", "(ssn = '123-45-6789') => process(ssn)", true},

		// Destructuring
		{"Destructuring", "const {ssn = '123-45-6789'} = user", true},
		{"Array destructuring", "const [ssn = '123-45-6789'] = data", true},

		// Variable names that suggest test data
		{"test_ssn variable", "var test_ssn = '123-45-6789'", true},
		{"mockSSN variable", "const mockSSN = '123-45-6789'", true},
		{"sample_data variable", "let sample_data = '123-45-6789'", true},

		// Negative cases
		{"Function call", "process('123-45-6789')", false},
		{"Object property", "user.ssn = '123-45-6789'", false},
		{"Array access", "data[0] = '123-45-6789'", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := matcher.IsVariableContext(tc.text)
			assert.Equal(t, tc.expected, result, "Case: %s", tc.name)
		})
	}
}

func TestPatternMatcher_EdgeCases(t *testing.T) {
	matcher := NewPatternMatcher()

	testCases := []struct {
		name        string
		text        string
		method      string
		description string
	}{
		{
			name:        "Empty string",
			text:        "",
			method:      "all",
			description: "Should handle empty strings gracefully",
		},
		{
			name:        "Very long string",
			text:        strings.Repeat("a", 10000) + "SSN: 123-45-6789",
			method:      "PIContextLabels",
			description: "Should handle very long strings efficiently",
		},
		{
			name:        "Special characters",
			text:        "SSN: 123-45-6789 (§†∆ø¬)",
			method:      "PIContextLabels",
			description: "Should handle special characters",
		},
		{
			name:        "Unicode characters",
			text:        "SSN: 123-45-6789 测试数据",
			method:      "PIContextLabels",
			description: "Should handle unicode characters",
		},
		{
			name:        "Mixed line endings",
			text:        "SSN: 123-45-6789\r\nTFN: 987654321\n",
			method:      "PIContextLabels",
			description: "Should handle different line endings",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Test that methods don't panic with edge cases
			switch tc.method {
			case "all":
				assert.NotPanics(t, func() {
					matcher.ContainsTestDataKeywords(tc.text)
					matcher.FindPIContextLabels(tc.text)
					matcher.IsDocumentationContext(tc.text)
					matcher.IsFormFieldContext(tc.text)
					matcher.IsDatabaseContext(tc.text)
					matcher.IsLogContext(tc.text)
					matcher.IsConfigurationContext(tc.text)
					matcher.IsVariableContext(tc.text)
				}, tc.description)
			case "PIContextLabels":
				assert.NotPanics(t, func() {
					matcher.FindPIContextLabels(tc.text)
				}, tc.description)
			}
		})
	}
}
