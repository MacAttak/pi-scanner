package integration

import (
	"context"
	"testing"

	"github.com/MacAttak/pi-scanner/pkg/detection"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSimpleDetection(t *testing.T) {
	detector := detection.NewDetector()
	ctx := context.Background()

	// Test Java file content
	javaContent := `
package com.bank.model;

public class Customer {
    private String name;
    private String tfn; // Example: 123-456-782
    private String email;

    public Customer(String name, String tfn) {
        this.name = name;
        this.tfn = tfn; // TFN: 865-432-108
    }
}`

	findings, err := detector.Detect(ctx, []byte(javaContent), "Customer.java")
	require.NoError(t, err)
	assert.NotEmpty(t, findings, "Should detect TFN in Java file")

	for _, f := range findings {
		t.Logf("Found: %s (%s) at line %d", f.Match, f.Type, f.Line)
	}
}
