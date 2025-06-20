package testing

import (
	"context"
	"testing"

	"github.com/MacAttak/pi-scanner/pkg/detection"
	"github.com/stretchr/testify/require"
)

func TestPhoneEdgeCases(t *testing.T) {
	detector := detection.NewDetector()
	ctx := context.Background()

	// Test specific failing cases
	testCases := []struct {
		name string
		code string
	}{
		{
			name: "International format",
			code: `phone := "+61412345678"`,
		},
		{
			name: "Landline format",
			code: `phone := "(02) 9999 9999"`,
		},
		{
			name: "ABN vs Phone confusion",
			code: `call("+61412345678")`,
		},
		{
			name: "Various formats",
			code: `phones = ["+61412345678", "(02) 9999 9999", "0412345678"]`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Logf("Testing: %s", tc.code)

			findings, err := detector.Detect(ctx, []byte(tc.code), "test.go")
			require.NoError(t, err)

			t.Logf("  Total findings: %d", len(findings))
			for _, f := range findings {
				t.Logf("    -> Type: %s, Match: %s, Context: %s", f.Type, f.Match, f.Context)
			}
		})
	}
}
