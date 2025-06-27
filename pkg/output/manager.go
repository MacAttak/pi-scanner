package output

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/MacAttak/pi-scanner/pkg/detection"
)

// Manager handles secure output generation with automatic masking
type Manager struct {
	masker      *Masker
	config      *Config
	logger      *slog.Logger
	auditLogger *AuditLogger
	sanitizer   *LogSanitizer
	mu          sync.RWMutex
}

// Config configures the output manager
type Config struct {
	// MaskingLevel controls how PI is masked in outputs
	MaskingLevel MaskingLevel

	// RequireExplicitUnmasked requires a flag to output unmasked data
	RequireExplicitUnmasked bool

	// EnableAuditLogging logs all output operations
	EnableAuditLogging bool

	// AuditLogPath is the path for audit logs
	AuditLogPath string

	// SanitizeLogs ensures PI doesn't appear in application logs
	SanitizeLogs bool

	// AllowedOutputFormats restricts which formats can be used
	AllowedOutputFormats []string

	// WarnOnInsecureConfig warns when using insecure settings
	WarnOnInsecureConfig bool
}

// DefaultConfig returns secure default configuration
func DefaultConfig() *Config {
	return &Config{
		MaskingLevel:            MaskingLevelPartial,
		RequireExplicitUnmasked: true,
		EnableAuditLogging:      true,
		AuditLogPath:            "pi-scanner-audit.log",
		SanitizeLogs:            true,
		AllowedOutputFormats:    []string{"json", "csv", "html", "sarif"},
		WarnOnInsecureConfig:    true,
	}
}

// NewManager creates a new output manager
func NewManager(config *Config, logger *slog.Logger) (*Manager, error) {
	if config == nil {
		config = DefaultConfig()
	}

	if logger == nil {
		logger = slog.Default()
	}

	// Validate configuration
	if err := validateConfig(config); err != nil {
		return nil, fmt.Errorf("invalid output configuration: %w", err)
	}

	// Warn about insecure configurations
	if config.WarnOnInsecureConfig {
		warnInsecureConfig(config, logger)
	}

	manager := &Manager{
		masker:    NewMasker(config.MaskingLevel),
		config:    config,
		logger:    logger,
		sanitizer: NewLogSanitizer(),
	}

	// Initialize audit logging if enabled
	if config.EnableAuditLogging {
		auditLogger, err := NewAuditLogger(config.AuditLogPath)
		if err != nil {
			return nil, fmt.Errorf("failed to create audit logger: %w", err)
		}
		manager.auditLogger = auditLogger
	}

	return manager, nil
}

// SetMaskingLevel changes the masking level (with audit logging)
func (m *Manager) SetMaskingLevel(level MaskingLevel) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	oldLevel := m.config.MaskingLevel
	m.config.MaskingLevel = level
	m.masker.SetLevel(level)

	// Audit the change
	if m.auditLogger != nil {
		m.auditLogger.LogConfigChange("masking_level", string(oldLevel), string(level))
	}

	// Warn if changing to insecure level
	if level == MaskingLevelNone && m.config.WarnOnInsecureConfig {
		m.logger.Warn("Masking disabled - PI data will be exposed in outputs",
			"level", level,
			"risk", "HIGH")
	}

	return nil
}

// PrepareFindings prepares findings for output with appropriate masking
func (m *Manager) PrepareFindings(findings []detection.Finding) []detection.Finding {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Create masked copies
	masked := make([]detection.Finding, len(findings))
	for i, finding := range findings {
		masked[i] = m.masker.MaskFinding(&finding)
	}

	// Audit the operation
	if m.auditLogger != nil {
		m.auditLogger.LogOutputOperation("prepare_findings", len(findings), m.config.MaskingLevel)
	}

	return masked
}

// WriteJSON writes findings as JSON with automatic masking
func (m *Manager) WriteJSON(w io.Writer, result *detection.ScanResult) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Check if format is allowed
	if !m.isFormatAllowed("json") {
		return fmt.Errorf("JSON output format is not allowed")
	}

	// Create a copy with masked findings
	maskedResult := *result
	maskedResult.Findings = m.PrepareFindings(result.Findings)

	// Encode to JSON
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")

	if err := encoder.Encode(maskedResult); err != nil {
		return fmt.Errorf("failed to encode JSON: %w", err)
	}

	// Audit the operation
	if m.auditLogger != nil {
		m.auditLogger.LogOutputGeneration("json", len(result.Findings), m.config.MaskingLevel)
	}

	return nil
}

// ValidateOutput checks that output doesn't contain unmasked PI
func (m *Manager) ValidateOutput(output []byte, findings []detection.Finding) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Skip validation if masking is disabled
	if m.config.MaskingLevel == MaskingLevelNone {
		return nil
	}

	outputStr := string(output)

	// Check each finding to ensure the raw value doesn't appear
	for _, finding := range findings {
		if finding.Match != "" && strings.Contains(outputStr, finding.Match) {
			// Check if it's properly masked
			masked := m.masker.Mask(finding.Match, finding.Type)
			if !strings.Contains(outputStr, masked) {
				return fmt.Errorf("unmasked PI detected in output: %s type at line %d",
					finding.Type, finding.Line)
			}
		}
	}

	return nil
}

// GetSafeLogger returns a logger that sanitizes PI from logs
func (m *Manager) GetSafeLogger() *slog.Logger {
	if !m.config.SanitizeLogs {
		return m.logger
	}

	// Wrap the logger with sanitization
	return slog.New(&sanitizingHandler{
		handler:   m.logger.Handler(),
		sanitizer: m.sanitizer,
	})
}

// Close closes the output manager and its resources
func (m *Manager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.auditLogger != nil {
		return m.auditLogger.Close()
	}

	return nil
}

// isFormatAllowed checks if an output format is allowed
func (m *Manager) isFormatAllowed(format string) bool {
	if len(m.config.AllowedOutputFormats) == 0 {
		return true // No restrictions
	}

	for _, allowed := range m.config.AllowedOutputFormats {
		if strings.EqualFold(allowed, format) {
			return true
		}
	}

	return false
}

// validateConfig validates the output configuration
func validateConfig(config *Config) error {
	// Validate masking level
	if _, err := ValidateMaskingLevel(string(config.MaskingLevel)); err != nil {
		return fmt.Errorf("invalid masking level: %w", err)
	}

	// Validate audit log path if enabled
	if config.EnableAuditLogging && config.AuditLogPath == "" {
		return fmt.Errorf("audit log path required when audit logging is enabled")
	}

	return nil
}

// warnInsecureConfig warns about insecure configuration options
func warnInsecureConfig(config *Config, logger *slog.Logger) {
	if config.MaskingLevel == MaskingLevelNone {
		logger.Warn("Output masking is disabled - PI data will be exposed",
			"recommendation", "Use PARTIAL or FULL masking")
	}

	if !config.RequireExplicitUnmasked && config.MaskingLevel == MaskingLevelNone {
		logger.Warn("Unmasked output allowed without explicit flag",
			"recommendation", "Enable RequireExplicitUnmasked")
	}

	if !config.EnableAuditLogging {
		logger.Warn("Audit logging is disabled",
			"recommendation", "Enable audit logging for compliance")
	}

	if !config.SanitizeLogs {
		logger.Warn("Log sanitization is disabled - PI may appear in logs",
			"recommendation", "Enable SanitizeLogs")
	}
}

// AuditLogger handles audit logging for output operations
type AuditLogger struct {
	file   *os.File
	logger *slog.Logger
	mu     sync.Mutex
}

// NewAuditLogger creates a new audit logger
func NewAuditLogger(path string) (*AuditLogger, error) {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return nil, fmt.Errorf("failed to open audit log: %w", err)
	}

	logger := slog.New(slog.NewJSONHandler(file, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	return &AuditLogger{
		file:   file,
		logger: logger,
	}, nil
}

// LogOutputOperation logs an output operation
func (a *AuditLogger) LogOutputOperation(operation string, findingCount int, maskingLevel MaskingLevel) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.logger.Info("output_operation",
		"operation", operation,
		"finding_count", findingCount,
		"masking_level", maskingLevel,
		"timestamp", slog.TimeValue(time.Now()),
	)
}

// LogOutputGeneration logs output file generation
func (a *AuditLogger) LogOutputGeneration(format string, findingCount int, maskingLevel MaskingLevel) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.logger.Info("output_generated",
		"format", format,
		"finding_count", findingCount,
		"masking_level", maskingLevel,
		"timestamp", slog.TimeValue(time.Now()),
	)
}

// LogConfigChange logs configuration changes
func (a *AuditLogger) LogConfigChange(setting, oldValue, newValue string) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.logger.Info("config_changed",
		"setting", setting,
		"old_value", oldValue,
		"new_value", newValue,
		"timestamp", slog.TimeValue(time.Now()),
	)
}

// Close closes the audit logger
func (a *AuditLogger) Close() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	return a.file.Close()
}

// LogSanitizer removes PI from log messages
type LogSanitizer struct {
	patterns map[string]*regexp.Regexp
	mu       sync.RWMutex
}

// NewLogSanitizer creates a new log sanitizer
func NewLogSanitizer() *LogSanitizer {
	return &LogSanitizer{
		patterns: make(map[string]*regexp.Regexp),
	}
}

// AddPattern adds a pattern to sanitize
func (s *LogSanitizer) AddPattern(name string, pattern *regexp.Regexp) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.patterns[name] = pattern
}

// Sanitize removes sensitive patterns from a string
func (s *LogSanitizer) Sanitize(input string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := input
	for name, pattern := range s.patterns {
		result = pattern.ReplaceAllString(result, fmt.Sprintf("[%s_REDACTED]", name))
	}

	return result
}

// sanitizingHandler wraps a slog handler to sanitize logs
type sanitizingHandler struct {
	handler   slog.Handler
	sanitizer *LogSanitizer
}

func (h *sanitizingHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.handler.Enabled(ctx, level)
}

func (h *sanitizingHandler) Handle(ctx context.Context, record slog.Record) error {
	// Sanitize the message
	record.Message = h.sanitizer.Sanitize(record.Message)

	// Sanitize attributes
	record.Attrs(func(attr slog.Attr) bool {
		if attr.Value.Kind() == slog.KindString {
			attr.Value = slog.StringValue(h.sanitizer.Sanitize(attr.Value.String()))
		}
		return true
	})

	return h.handler.Handle(ctx, record)
}

func (h *sanitizingHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &sanitizingHandler{
		handler:   h.handler.WithAttrs(attrs),
		sanitizer: h.sanitizer,
	}
}

func (h *sanitizingHandler) WithGroup(name string) slog.Handler {
	return &sanitizingHandler{
		handler:   h.handler.WithGroup(name),
		sanitizer: h.sanitizer,
	}
}
