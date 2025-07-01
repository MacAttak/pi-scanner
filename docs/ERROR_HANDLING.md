# Error Handling Guidelines

## Overview

This document describes the error handling patterns used in the GitHub PI Scanner project.

## General Principles

1. **Always wrap errors with context** using `fmt.Errorf` with the `%w` verb
2. **Never ignore errors** - always handle or propagate them appropriately
3. **Provide meaningful error messages** that help diagnose the issue
4. **Use sentinel errors sparingly** - only for errors that callers need to handle specifically

## Error Wrapping Pattern

```go
// Good - provides context while preserving the original error
if err != nil {
    return fmt.Errorf("failed to process file %s: %w", filename, err)
}

// Bad - loses the original error
if err != nil {
    return fmt.Errorf("failed to process file: %s", err)
}
```

## Error Types

### Validation Errors
For input validation failures:
```go
if len(input) == 0 {
    return fmt.Errorf("invalid input: cannot be empty")
}
```

### External Service Errors
When calling external services (GitHub API, LLM, etc.):
```go
resp, err := client.Call()
if err != nil {
    return fmt.Errorf("GitHub API call failed: %w", err)
}
```

### File System Errors
For file operations:
```go
content, err := os.ReadFile(path)
if err != nil {
    return fmt.Errorf("failed to read file %s: %w", path, err)
}
```

## Error Logging

- Log errors at the highest level where they can be handled meaningfully
- Include relevant context in log messages
- Don't log and return the same error (avoid duplicate logging)

```go
// In main or top-level handlers
if err := scanner.Scan(); err != nil {
    log.Printf("Scan failed: %v", err)
    return 1
}
```

## Testing Errors

Always test error paths:
```go
func TestProcessFile_InvalidPath(t *testing.T) {
    _, err := ProcessFile("/invalid/path")
    require.Error(t, err)
    assert.Contains(t, err.Error(), "failed to read file")
}
```

## Common Patterns

### Early Return
```go
func ProcessData(data []byte) error {
    if len(data) == 0 {
        return fmt.Errorf("data cannot be empty")
    }

    // Process data...
    return nil
}
```

### Error Aggregation
When processing multiple items:
```go
var errs []error
for _, item := range items {
    if err := process(item); err != nil {
        errs = append(errs, fmt.Errorf("failed to process %s: %w", item.Name, err))
    }
}
if len(errs) > 0 {
    return fmt.Errorf("processing failed with %d errors: %v", len(errs), errs)
}
```

### Context Cancellation
Always check context in long-running operations:
```go
select {
case <-ctx.Done():
    return ctx.Err()
default:
    // Continue processing
}
```
