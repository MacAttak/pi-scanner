# Secure Output Framework Implementation

## Overview

The Secure Output Framework ensures that PI data is properly masked in all scanner outputs while maintaining the ability for users to locate and verify findings. It provides configurable masking levels, audit logging, and validation to prevent accidental PI exposure.

## Key Components

### 1. Enhanced Masking System (`pkg/output/masking.go`)

The masking system provides three levels of data protection:

```go
type MaskingLevel string

const (
    MaskingLevelFull    = "FULL"    // Complete redaction
    MaskingLevelPartial = "PARTIAL" // Shows some characters
    MaskingLevelNone    = "NONE"    // No masking (requires explicit flag)
)
```

**Features:**
- Type-specific masking patterns for each PI type
- Context-aware masking that handles variations (spaces, dashes)
- Preserves data utility while protecting sensitive information

**Example Masking Patterns:**
- **TFN**: `123456789` → `123****89`
- **Email**: `john.doe@example.com` → `jo***@example.com`
- **Credit Card**: `1234567812345678` → `************5678`
- **Phone**: `0412345678` → `0412****78`

### 2. Secure Output Manager (`pkg/output/manager.go`)

Centralizes all output handling with automatic security controls:

```go
type Manager struct {
    masker       *Masker
    config       *Config
    logger       *slog.Logger
    auditLogger  *AuditLogger
    sanitizer    *LogSanitizer
}
```

**Key Features:**
- Automatic masking based on configuration
- Output validation to detect unmasked PI
- Audit logging of all output operations
- Log sanitization to prevent PI in application logs
- Format restrictions and security warnings

### 3. Report Security Layer (`pkg/report/security.go`)

Wraps all report generators with security controls:

```go
type SecurityLayer struct {
    outputManager *output.Manager
    config        *SecurityConfig
    logger        *slog.Logger
}
```

**Security Controls:**
- Enforces masking policy before output
- Validates generated reports for PI leakage
- Logs all report generation for audit trail
- Prevents bypassing security controls

## Configuration

### Default Configuration (Secure by Default)

```go
config := output.DefaultConfig()
// Returns:
// - MaskingLevel: PARTIAL
// - RequireExplicitUnmasked: true
// - EnableAuditLogging: true
// - SanitizeLogs: true
```

### Custom Configuration

```go
config := &output.Config{
    MaskingLevel:            output.MaskingLevelFull,
    RequireExplicitUnmasked: true,
    EnableAuditLogging:      true,
    AuditLogPath:           "/secure/audit.log",
    SanitizeLogs:           true,
    AllowedOutputFormats:   []string{"json", "csv"},
    WarnOnInsecureConfig:   true,
}
```

## Usage Examples

### Basic Report Generation

```go
// Create output manager
outputConfig := output.DefaultConfig()
outputManager, err := output.NewManager(outputConfig, logger)
if err != nil {
    log.Fatal(err)
}
defer outputManager.Close()

// Create report factory
reportFactory := report.NewReportFactory(outputManager, nil)

// Generate CSV report with automatic masking
csvExporter := reportFactory.CreateCSVExporter(
    report.WithMaskedValues(),
    report.WithContext(),
)

err = csvExporter.ExportFindings(writer, findings, metadata)
```

### JSON Output with Validation

```go
// Write JSON with automatic masking and validation
err = outputManager.WriteJSON(writer, scanResult)
if err != nil {
    // Error if output contains unmasked PI
    log.Error("Failed to generate secure output", err)
}
```

### Custom Masking Patterns

```go
masker := output.NewMasker(output.MaskingLevelPartial)

// Set custom pattern for specific PI type
masker.SetPattern(detection.PITypeTFN, output.MaskPattern{
    ShowPrefix: 2,
    ShowSuffix: 3,
    MaskChar:   "#",
})

// Result: 123456789 → 12####789
```

## Security Features

### 1. Output Validation

Before any output is written, the framework validates that no unmasked PI appears:

```go
func (m *Manager) ValidateOutput(output []byte, findings []detection.Finding) error
```

This prevents accidental exposure even if masking fails.

### 2. Audit Logging

All output operations are logged for compliance:

```json
{
  "time": "2024-01-20T10:30:00Z",
  "level": "INFO",
  "msg": "output_generated",
  "format": "csv",
  "finding_count": 42,
  "masking_level": "PARTIAL"
}
```

### 3. Log Sanitization

Application logs are automatically sanitized to prevent PI exposure:

```go
logger := outputManager.GetSafeLogger()
logger.Info("Found TFN 123456789") // Logged as: "Found TFN [TFN_REDACTED]"
```

### 4. Configuration Warnings

The framework warns about insecure configurations:

```
WARN: Output masking is disabled - PI data will be exposed
WARN: Audit logging is disabled - recommendation: Enable for compliance
```

## Integration with LLM Validation

The framework ensures that:
1. **LLM receives full data**: The local LLM gets unmasked PI for accurate validation
2. **Output is masked**: All reports show masked PI values
3. **Location preserved**: File paths, line numbers, and columns are always included

```go
// LLM validation gets full data
llmResult := llmValidator.ValidateFinding(ctx, finding) // Unmasked

// Output shows masked data
report := outputManager.PrepareFindings([]Finding{finding}) // Masked
```

## Report Formats

All report formats support secure output:

### CSV Export
```csv
File Path,Line,PI Type,Masked Value
/src/config.go,42,TFN,123****89
/src/user.go,100,EMAIL,jo***@example.com
```

### JSON Export
```json
{
  "findings": [{
    "file": "/src/config.go",
    "line": 42,
    "type": "TFN",
    "match": "123****89",
    "risk_level": "HIGH"
  }]
}
```

### HTML Report
- Visual representation with color-coded risk levels
- Masked values displayed prominently
- Full location information for verification

## Migration Guide

### For Existing Code

1. **Replace direct masking calls**:
```go
// Old
masked := maskSensitiveData(value, piType)

// New
masker := output.NewMasker(output.MaskingLevelPartial)
masked := masker.Mask(value, piType)
```

2. **Use secure report generators**:
```go
// Old
exporter := report.NewCSVExporter()

// New
factory := report.NewReportFactory(outputManager, nil)
exporter := factory.CreateCSVExporter()
```

3. **Update configuration**:
```yaml
# Add to config file
output:
  masking_level: PARTIAL
  enable_audit: true
  sanitize_logs: true
```

## Testing

The framework includes comprehensive tests:

1. **Unit Tests**: All masking patterns and edge cases
2. **Integration Tests**: Report generation with validation
3. **Security Tests**: Validation of PI detection in outputs
4. **Performance Tests**: Large report generation

## Best Practices

1. **Always use PARTIAL masking** as the default
2. **Enable audit logging** for compliance tracking
3. **Validate outputs** before writing to disk
4. **Use the safe logger** for all application logging
5. **Review audit logs** regularly for security incidents

## Conclusion

The Secure Output Framework successfully addresses the PI exposure gap by:
- ✅ Implementing configurable masking levels
- ✅ Ensuring LLM validation accuracy with full data
- ✅ Providing detailed location information
- ✅ Adding comprehensive audit logging
- ✅ Preventing PI leakage in logs and outputs

This maintains the scanner's security-by-design principle while providing the flexibility needed for different use cases.
