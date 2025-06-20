package testing

import (
	"context"
	"testing"

	"github.com/MacAttak/pi-scanner/pkg/detection"
	"github.com/MacAttak/pi-scanner/pkg/testing/benchmark"
	"github.com/stretchr/testify/require"
)

func TestACNDebug(t *testing.T) {
	detector := detection.NewDetector()
	ctx := context.Background()
	generator := benchmark.NewTestDataGenerator()
	
	// Generate some ACNs
	acns := []string{
		generator.GenerateValidACN(),
		"123456789", // Simple 9 digit
		"123 456 789", // With spaces
	}
	
	for _, acn := range acns {
		t.Run(acn, func(t *testing.T) {
			code := `company.ACN = "` + acn + `"`
			t.Logf("Testing: %s", code)
			
			findings, err := detector.Detect(ctx, []byte(code), "test.go")
			require.NoError(t, err)
			
			t.Logf("Findings: %d", len(findings))
			for _, f := range findings {
				t.Logf("  -> Type: %s, Match: %s", f.Type, f.Match)
			}
			
			// Check if pattern matches at all
			hasACN := false
			hasTFN := false
			for _, f := range findings {
				if f.Type == detection.PITypeACN {
					hasACN = true
				}
				if f.Type == detection.PITypeTFN {
					hasTFN = true
				}
			}
			
			t.Logf("  Has ACN: %v, Has TFN: %v", hasACN, hasTFN)
		})
	}
}