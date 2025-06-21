//go:build !ci
// +build !ci

package testing

import (
	"regexp"
	"testing"
)

func TestPhoneRegexPatterns(t *testing.T) {
	pattern := `\b(?:(?:\+?61|0)[\s.-]?[2-9](?:[\s.-]?\d){8}|\(\d{2}\)\s*\d{4}\s*\d{4}|1[38]00[\s.-]?\d{3}[\s.-]?\d{3})\b`
	re := regexp.MustCompile(pattern)

	testCases := []struct {
		input    string
		expected bool
	}{
		{"0412345678", true},
		{"0412 345 678", true},
		{"+61412345678", true},
		{"+61 412 345 678", true},
		{"(02) 9999 9999", true},
		{"1300123456", true},
		{"1800123456", true},
		{"1300 123 456", true},
		{"61412345678", true}, // without +
	}

	for _, tc := range testCases {
		matches := re.FindAllString(tc.input, -1)
		found := len(matches) > 0

		t.Logf("Input: %s - Expected: %v - Found: %v - Matches: %v",
			tc.input, tc.expected, found, matches)

		if found != tc.expected {
			t.Errorf("Pattern mismatch for %s", tc.input)
		}
	}
}
