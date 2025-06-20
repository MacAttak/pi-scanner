package testing

import (
	"regexp"
	"testing"
)

func TestACNRegex(t *testing.T) {
	pattern := `(?i)(?:(?:\.)?acn|company\s*number|australian\s*company\s*number)\s*[=:\s"']*\s*\d{3}[\s]?\d{3}[\s]?\d{3}\b`
	re := regexp.MustCompile(pattern)
	
	testCases := []struct {
		input    string
		expected bool
	}{
		{`company.ACN = "123456789"`, true},
		{`ACN: 123456789`, true},
		{`acn="123456789"`, true},
		{`Australian Company Number: 123 456 789`, true},
		{`"acn": "123456789"`, true},
		{`ACN 123456789`, true},
		{`.ACN = "123456789"`, true},
	}
	
	for _, tc := range testCases {
		matches := re.FindAllString(tc.input, -1)
		found := len(matches) > 0
		
		t.Logf("Input: %s", tc.input)
		t.Logf("  Expected: %v, Found: %v", tc.expected, found)
		if found {
			t.Logf("  Matches: %v", matches)
		}
		
		if found != tc.expected {
			t.Errorf("Pattern mismatch for %s", tc.input)
		}
	}
}