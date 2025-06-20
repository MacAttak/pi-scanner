package testing

import (
	"context"
	"fmt"
	"testing"

	"github.com/MacAttak/pi-scanner/pkg/detection"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPhoneDetection(t *testing.T) {
	detector := detection.NewDetector()
	ctx := context.Background()

	testCases := []struct {
		name     string
		code     string
		expected bool
	}{
		{
			name:     "Australian mobile",
			code:     `phone := "0412345678"`,
			expected: true,
		},
		{
			name:     "Australian mobile with spaces",
			code:     `phone := "0412 345 678"`,
			expected: true,
		},
		{
			name:     "Australian mobile international",
			code:     `phone := "+61412345678"`,
			expected: true,
		},
		{
			name:     "Australian landline",
			code:     `phone := "(02) 9999 9999"`,
			expected: true,
		},
		{
			name:     "Business 1300 number",
			code:     `support := "1300123456"`,
			expected: true,
		},
		{
			name:     "Business 1800 number",
			code:     `support := "1800123456"`,
			expected: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			findings, err := detector.Detect(ctx, []byte(tc.code), "test.go")
			require.NoError(t, err)

			found := false
			for _, f := range findings {
				if f.Type == detection.PITypePhone {
					found = true
					t.Logf("Found phone: %s", f.Match)
				}
			}

			if tc.expected {
				assert.True(t, found, "Should detect phone number")
			} else {
				assert.False(t, found, "Should not detect phone number")
			}
		})
	}
}

func TestPhonePatternDebug(t *testing.T) {
	// Test the actual regex pattern
	_ = `\b(?:\+?61|0)[2-9]\d{8}\b|\(\d{2}\)\s*\d{4}\s*\d{4}`

	testNumbers := []string{
		"0412345678",
		"0412 345 678",
		"+61412345678",
		"+61 412 345 678",
		"(02) 9999 9999",
		"1300123456",
		"1800123456",
	}

	for _, num := range testNumbers {
		code := fmt.Sprintf(`phone := "%s"`, num)
		t.Logf("Testing: %s", code)
	}
}
