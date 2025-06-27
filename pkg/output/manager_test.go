package output

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"os"
	"regexp"
	"testing"
	"time"

	"github.com/MacAttak/pi-scanner/pkg/detection"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestManager_NewManager(t *testing.T) {
	tests := []struct {
		name      string
		config    *Config
		expectErr bool
	}{
		{
			name:      "default config",
			config:    nil,
			expectErr: false,
		},
		{
			name:      "valid config",
			config:    DefaultConfig(),
			expectErr: false,
		},
		{
			name: "invalid masking level",
			config: &Config{
				MaskingLevel: "INVALID",
			},
			expectErr: true,
		},
		{
			name: "missing audit path",
			config: &Config{
				MaskingLevel:       MaskingLevelPartial,
				EnableAuditLogging: true,
				AuditLogPath:       "",
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Use temp file for audit log
			if tt.config != nil && tt.config.EnableAuditLogging && tt.config.AuditLogPath != "" {
				tmpFile, err := os.CreateTemp("", "audit-*.log")
				require.NoError(t, err)
				tmpFile.Close()
				defer os.Remove(tmpFile.Name())
				tt.config.AuditLogPath = tmpFile.Name()
			}

			manager, err := NewManager(tt.config, nil)
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, manager)
				if manager != nil {
					manager.Close()
				}
			}
		})
	}
}

func TestManager_PrepareFindings(t *testing.T) {
	config := &Config{
		MaskingLevel:       MaskingLevelPartial,
		EnableAuditLogging: false,
	}

	manager, err := NewManager(config, nil)
	require.NoError(t, err)
	defer manager.Close()

	findings := []detection.Finding{
		{
			Type:    detection.PITypeTFN,
			Match:   "123456789",
			File:    "test.go",
			Line:    10,
			Context: "TFN: 123456789",
		},
		{
			Type:  detection.PITypeEmail,
			Match: "john@example.com",
			File:  "config.yaml",
			Line:  20,
		},
	}

	masked := manager.PrepareFindings(findings)

	// Check that findings are masked
	assert.Len(t, masked, 2)
	assert.Equal(t, "123****89", masked[0].Match)
	assert.Equal(t, "jo***@example.com", masked[1].Match)
	assert.Contains(t, masked[0].Context, "123****89")

	// Original findings should not be modified
	assert.Equal(t, "123456789", findings[0].Match)
}

func TestManager_SetMaskingLevel(t *testing.T) {
	// Create temp audit log
	tmpFile, err := os.CreateTemp("", "audit-*.log")
	require.NoError(t, err)
	tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	config := &Config{
		MaskingLevel:       MaskingLevelPartial,
		EnableAuditLogging: true,
		AuditLogPath:       tmpFile.Name(),
	}

	manager, err := NewManager(config, slog.Default())
	require.NoError(t, err)
	defer manager.Close()

	// Change masking level
	err = manager.SetMaskingLevel(MaskingLevelFull)
	assert.NoError(t, err)

	// Verify the change
	findings := []detection.Finding{
		{
			Type:  detection.PITypeTFN,
			Match: "123456789",
		},
	}

	masked := manager.PrepareFindings(findings)
	assert.Equal(t, "*********", masked[0].Match)

	// Check audit log was written
	data, err := os.ReadFile(tmpFile.Name())
	assert.NoError(t, err)
	assert.Contains(t, string(data), "config_changed")
	assert.Contains(t, string(data), "PARTIAL")
	assert.Contains(t, string(data), "FULL")
}

func TestManager_WriteJSON(t *testing.T) {
	config := &Config{
		MaskingLevel:         MaskingLevelPartial,
		EnableAuditLogging:   false,
		AllowedOutputFormats: []string{"json", "csv"},
	}

	manager, err := NewManager(config, nil)
	require.NoError(t, err)
	defer manager.Close()

	scanResult := &detection.ScanResult{
		Repository: "test-repo",
		StartTime:  time.Now(),
		EndTime:    time.Now(),
		Findings: []detection.Finding{
			{
				Type:  detection.PITypeTFN,
				Match: "123456789",
				File:  "test.go",
				Line:  10,
			},
		},
	}

	// Write to buffer
	var buf bytes.Buffer
	err = manager.WriteJSON(&buf, scanResult)
	assert.NoError(t, err)

	// Parse the output
	var result detection.ScanResult
	err = json.Unmarshal(buf.Bytes(), &result)
	assert.NoError(t, err)

	// Check that findings are masked
	assert.Len(t, result.Findings, 1)
	assert.Equal(t, "123****89", result.Findings[0].Match)
}

func TestManager_ValidateOutput(t *testing.T) {
	config := &Config{
		MaskingLevel: MaskingLevelPartial,
	}

	manager, err := NewManager(config, nil)
	require.NoError(t, err)
	defer manager.Close()

	findings := []detection.Finding{
		{
			Type:  detection.PITypeTFN,
			Match: "123456789",
			Line:  10,
		},
	}

	tests := []struct {
		name      string
		output    string
		expectErr bool
	}{
		{
			name:      "properly masked output",
			output:    `{"match": "123****89"}`,
			expectErr: false,
		},
		{
			name:      "unmasked PI in output",
			output:    `{"match": "123456789"}`,
			expectErr: true,
		},
		{
			name:      "no PI in output",
			output:    `{"status": "completed"}`,
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := manager.ValidateOutput([]byte(tt.output), findings)
			if tt.expectErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "unmasked PI detected")
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestManager_ValidateOutput_NoMasking(t *testing.T) {
	config := &Config{
		MaskingLevel: MaskingLevelNone,
	}

	manager, err := NewManager(config, nil)
	require.NoError(t, err)
	defer manager.Close()

	findings := []detection.Finding{
		{
			Type:  detection.PITypeTFN,
			Match: "123456789",
		},
	}

	// Should not validate when masking is disabled
	output := `{"match": "123456789"}`
	err = manager.ValidateOutput([]byte(output), findings)
	assert.NoError(t, err)
}

func TestManager_AllowedFormats(t *testing.T) {
	config := &Config{
		MaskingLevel:         MaskingLevelPartial,
		AllowedOutputFormats: []string{"json"},
	}

	manager, err := NewManager(config, nil)
	require.NoError(t, err)
	defer manager.Close()

	// JSON should be allowed
	assert.True(t, manager.isFormatAllowed("json"))
	assert.True(t, manager.isFormatAllowed("JSON")) // Case insensitive

	// CSV should not be allowed
	assert.False(t, manager.isFormatAllowed("csv"))
}

func TestAuditLogger(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "audit-test-*.log")
	require.NoError(t, err)
	tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	logger, err := NewAuditLogger(tmpFile.Name())
	require.NoError(t, err)
	defer logger.Close()

	// Log operations
	logger.LogOutputOperation("test_op", 10, MaskingLevelPartial)
	logger.LogOutputGeneration("json", 5, MaskingLevelFull)
	logger.LogConfigChange("masking_level", "PARTIAL", "FULL")

	// Close to flush
	logger.Close()

	// Read and verify
	data, err := os.ReadFile(tmpFile.Name())
	assert.NoError(t, err)

	logs := string(data)
	assert.Contains(t, logs, "output_operation")
	assert.Contains(t, logs, "output_generated")
	assert.Contains(t, logs, "config_changed")
}

func TestLogSanitizer(t *testing.T) {
	sanitizer := NewLogSanitizer()

	// Add patterns for common PI
	tfnPattern := regexp.MustCompile(`\b\d{3}[\s\-]?\d{3}[\s\-]?\d{3}\b`)
	emailPattern := regexp.MustCompile(`\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Z|a-z]{2,}\b`)

	sanitizer.AddPattern("TFN", tfnPattern)
	sanitizer.AddPattern("EMAIL", emailPattern)

	tests := []struct {
		input    string
		expected string
	}{
		{
			input:    "Found TFN 123-456-789 in file",
			expected: "Found TFN [TFN_REDACTED] in file",
		},
		{
			input:    "User email is john@example.com",
			expected: "User email is [EMAIL_REDACTED]",
		},
		{
			input:    "No PI in this message",
			expected: "No PI in this message",
		},
		{
			input:    "Multiple: 123456789 and test@example.com",
			expected: "Multiple: [TFN_REDACTED] and [EMAIL_REDACTED]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := sanitizer.Sanitize(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestWarnInsecureConfig(t *testing.T) {
	// Create a buffer to capture log output
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))

	tests := []struct {
		name          string
		config        *Config
		expectedWarns []string
	}{
		{
			name: "no masking warning",
			config: &Config{
				MaskingLevel:         MaskingLevelNone,
				WarnOnInsecureConfig: true,
			},
			expectedWarns: []string{"Output masking is disabled"},
		},
		{
			name: "multiple warnings",
			config: &Config{
				MaskingLevel:            MaskingLevelNone,
				RequireExplicitUnmasked: false,
				EnableAuditLogging:      false,
				SanitizeLogs:            false,
				WarnOnInsecureConfig:    true,
			},
			expectedWarns: []string{
				"Output masking is disabled",
				"Unmasked output allowed",
				"Audit logging is disabled",
				"Log sanitization is disabled",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf.Reset()
			warnInsecureConfig(tt.config, logger)

			logs := buf.String()
			for _, warning := range tt.expectedWarns {
				assert.Contains(t, logs, warning)
			}
		})
	}
}
