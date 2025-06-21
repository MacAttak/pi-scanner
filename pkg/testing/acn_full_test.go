//go:build !ci
// +build !ci

package testing

import (
	"context"
	"testing"

	"github.com/MacAttak/pi-scanner/pkg/detection"
	"github.com/stretchr/testify/require"
)

func TestACNFullDetection(t *testing.T) {
	detector := detection.NewDetector()
	ctx := context.Background()

	// Test with full ACN context
	code := `company.ACN = "123456789"`
	t.Logf("Testing: %s", code)

	findings, err := detector.Detect(ctx, []byte(code), "test.go")
	require.NoError(t, err)

	t.Logf("Total findings: %d", len(findings))
	for i, f := range findings {
		t.Logf("Finding %d:", i)
		t.Logf("  Type: %s", f.Type)
		t.Logf("  Match: %s", f.Match)
		t.Logf("  Line: %d, Column: %d", f.Line, f.Column)
		t.Logf("  Context: %s", f.Context)
		t.Logf("  Confidence: %.2f", f.Confidence)
	}

	// Check what types were found
	foundTypes := make(map[detection.PIType]bool)
	for _, f := range findings {
		foundTypes[f.Type] = true
	}

	t.Logf("\nFound PI Types: %v", foundTypes)
}
