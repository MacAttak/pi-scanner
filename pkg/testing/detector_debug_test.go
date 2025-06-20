package testing

import (
	"context"
	"testing"

	"github.com/MacAttak/pi-scanner/pkg/detection"
	"github.com/stretchr/testify/require"
)

func TestDetectorDebug(t *testing.T) {
	detector := detection.NewDetector()
	ctx := context.Background()

	// Test a simple case
	code := `phone := "0412345678"`
	t.Logf("Testing code: %s", code)

	findings, err := detector.Detect(ctx, []byte(code), "test.go")
	require.NoError(t, err)

	t.Logf("Total findings: %d", len(findings))
	for i, f := range findings {
		t.Logf("Finding %d: Type=%s, Match=%s, Line=%d, Col=%d",
			i, f.Type, f.Match, f.Line, f.Column)
	}

	// Test with various patterns
	testCodes := []string{
		`phone := "0412345678"`,
		`const PHONE = "0412345678"`,
		`"phone": "0412345678"`,
		`call("+61412345678")`,
		`1300123456`,
	}

	for _, testCode := range testCodes {
		t.Logf("\nTesting: %s", testCode)
		findings, err := detector.Detect(ctx, []byte(testCode), "test.go")
		require.NoError(t, err)
		t.Logf("  Findings: %d", len(findings))
		for _, f := range findings {
			t.Logf("    -> %s: %s", f.Type, f.Match)
		}
	}
}
