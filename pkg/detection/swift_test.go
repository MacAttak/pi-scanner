package detection

import (
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestSWIFTDetection(t *testing.T) {
	detector := NewDetector()
	ctx := context.Background()

	tests := []struct {
		name        string
		content     string
		filename    string
		shouldFind  bool
		expectedVal string
		description string
	}{
		// Valid SWIFT codes
		{
			name:        "Valid 8-char SWIFT",
			content:     "Send payment to SWIFT: ANZBNZ22",
			filename:    "payment.txt",
			shouldFind:  true,
			expectedVal: "ANZBNZ22",
			description: "Valid ANZ Bank SWIFT code",
		},
		{
			name:        "Valid 11-char SWIFT",
			content:     "BIC: ANZBNZ22MEL for Melbourne branch",
			filename:    "payment.txt",
			shouldFind:  true,
			expectedVal: "ANZBNZ22MEL",
			description: "Valid SWIFT with branch code",
		},
		{
			name:        "Valid SWIFT in SQL comment",
			content:     "-- Send to ANZBNZ22 bank",
			filename:    "query.sql",
			shouldFind:  true,
			expectedVal: "ANZBNZ22",
			description: "SWIFT in SQL comment should be detected",
		},
		// SQL keywords that shouldn't be detected
		{
			name:        "SQL DISTINCT keyword",
			content:     "SELECT DISTINCT customer_id FROM accounts",
			filename:    "query.sql",
			shouldFind:  false,
			expectedVal: "DISTINCT",
			description: "SQL keyword should not be detected as SWIFT",
		},
		{
			name:        "SQL TRUNCATE keyword",
			content:     "TRUNCATE TABLE customers",
			filename:    "maintenance.sql",
			shouldFind:  false,
			expectedVal: "TRUNCATE",
			description: "SQL keyword should not be detected as SWIFT",
		},
		{
			name:        "SQL COALESCE function",
			content:     "COALESCE(name, 'Unknown') AS customer_name",
			filename:    "query.sql",
			shouldFind:  false,
			expectedVal: "COALESCE",
			description: "SQL function should not be detected as SWIFT",
		},
		{
			name:        "SQL INPUTFORMAT in Hive",
			content:     "STORED AS INPUTFORMAT 'org.apache.hadoop.mapred.TextInputFormat'",
			filename:    "create_table.sql",
			shouldFind:  false,
			expectedVal: "INPUTFORMAT",
			description: "Hive SQL keyword should not be detected as SWIFT",
		},
		// Invalid country codes
		{
			name:        "Invalid country code",
			content:     "Invalid SWIFT: ABCDXX12",
			filename:    "test.txt",
			shouldFind:  false,
			expectedVal: "ABCDXX12",
			description: "XX is not a valid country code",
		},
		{
			name:        "Invalid country code ZZ",
			content:     "Bad code: ABCDZZ12",
			filename:    "test.txt",
			shouldFind:  false,
			expectedVal: "ABCDZZ12",
			description: "ZZ is reserved and not valid",
		},
		// Kosovo special case
		{
			name:        "Kosovo SWIFT code",
			content:     "Kosovo bank: RBKOXKPR",
			filename:    "banks.txt",
			shouldFind:  true,
			expectedVal: "RBKOXKPR",
			description: "XK is valid for Kosovo in SWIFT",
		},
		// Edge cases
		{
			name:        "SWIFT with underscore",
			content:     "BANK_CODE_US12",
			filename:    "config.txt",
			shouldFind:  false,
			expectedVal: "BANK_CODE_US12",
			description: "Underscores indicate constants, not SWIFT",
		},
		{
			name:        "Too short",
			content:     "CODE: ABCD12",
			filename:    "test.txt",
			shouldFind:  false,
			expectedVal: "ABCD12",
			description: "6 characters is too short for SWIFT",
		},
		{
			name:        "Invalid structure",
			content:     "Bad: 1234US12",
			filename:    "test.txt",
			shouldFind:  false,
			expectedVal: "1234US12",
			description: "Bank code must be letters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			findings, err := detector.Detect(ctx, []byte(tt.content), tt.filename)
			require.NoError(t, err)

			found := false
			for _, f := range findings {
				if f.Type == PITypeSWIFT && f.Match == tt.expectedVal {
					found = true
					break
				}
			}

			if tt.shouldFind {
				assert.True(t, found, "Expected to find SWIFT code '%s' but didn't. Description: %s", tt.expectedVal, tt.description)
			} else {
				assert.False(t, found, "Should not have detected '%s' as SWIFT. Description: %s", tt.expectedVal, tt.description)
			}
		})
	}
}

func TestSWIFTSQLKeywordFiltering(t *testing.T) {
	detector := NewDetector()
	ctx := context.Background()

	// Test that SQL keywords are filtered even in non-SQL files
	sqlKeywords := []string{
		"DISTINCT", "TRUNCATE", "COALESCE", "INTERVAL", "ROLLBACK",
		"FUNCTION", "EXTERNAL", "POSITION", "ABSOLUTE", "RELATIVE",
		"CONSTRAINTS", "INPUTFORMAT", "OUTPUTFORMAT", "STATISTICS",
	}

	for _, keyword := range sqlKeywords {
		t.Run(keyword, func(t *testing.T) {
			// Test in regular text file
			content := "The " + keyword + " operation"
			findings, err := detector.Detect(ctx, []byte(content), "doc.txt")
			require.NoError(t, err)

			for _, f := range findings {
				if f.Type == PITypeSWIFT {
					assert.NotEqual(t, keyword, f.Match, "SQL keyword '%s' should not be detected as SWIFT even in non-SQL files", keyword)
				}
			}
		})
	}
}
