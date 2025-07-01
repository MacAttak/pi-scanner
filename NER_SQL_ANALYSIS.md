# Name Detection (NER) Analysis for SQL False Positives

## Current Implementation Analysis

### How Name Detection Currently Works

The current implementation in `/pkg/detection/detector.go` uses a simple regex-based approach for name detection:

```go
// Name matcher with context-aware filtering for code scanning
// Only detects names in appropriate contexts (comments, strings, documentation)
d.matchers = append(d.matchers, &regexMatcher{
    pattern: `\b[A-Z][a-z]{2,}\s+[A-Z][a-z]{2,}(?:\s+[A-Z][a-z]{2,})?\b`,
    piType:  PITypeName,
    d:       d,
    validator: func(match string) bool {
        // Context-aware validation for code scanning
        return d.isValidPersonName(match)
    },
})
```

### Pattern Explanation
- Matches 2-3 capitalized words (e.g., "John Smith", "Mary Jane Wilson")
- Each word must start with uppercase followed by at least 2 lowercase letters
- Words separated by spaces

### Current Validation Logic

The `isValidPersonName` function attempts to filter out programming terms but has significant gaps:

1. **Limited Programming Terms List**: While it includes many common terms, it misses SQL-specific patterns
2. **No SQL Context Awareness**: Doesn't recognize SQL keywords, table names, or column names
3. **Case-Sensitive Matching**: SQL identifiers can be in various cases

## Identified Issues with SQL False Positives

From our test, the following SQL constructs are incorrectly flagged as personal names:

### 1. Table Names
- `Everyday Banking` (CREATE TABLE)
- `Staging Table` (CREATE TABLE)
- `Transaction History` (INSERT INTO)
- `Customer Profile` (UPDATE)
- `Daily Reports` (CREATE VIEW)
- `System Reports` (FROM clause)

### 2. Column Names
- `Pending Product Description`
- `Contract Account`
- `Customer Name`
- `Account Balance`
- `Product Type`
- `Account Status`
- `Report Name`
- `Report Date`
- `Account Number`
- `Interest Rate`

### 3. Values and Identifiers
- `Savings Account`
- `Premium Banking`
- `Test User`
- `Customer Accounts`
- `Customer Type`
- `Transaction Date`

### 4. Comments
- `Production Database`
- `Customer Banking`
- `Service Manager`
- `Account Processing`

### 5. Function Names
- `Calculate Interest`

## Root Causes

1. **Overly Generic Pattern**: The regex matches any 2-3 capitalized words, which is common in SQL
2. **Insufficient Context Analysis**: Doesn't consider SQL syntax context
3. **Missing SQL-Specific Filters**: No recognition of SQL keywords or patterns
4. **Two-Phase Architecture Philosophy**: The current design intentionally casts a wide net for LLM validation

## Best Practices from Research

### 1. Context-Aware Detection
- Use syntax-aware parsing to understand code structure
- Differentiate between code identifiers and literal strings
- Consider surrounding tokens (CREATE, TABLE, SELECT, etc.)

### 2. Domain-Specific Training
- Train models specifically on SQL and programming language datasets
- Include negative examples of SQL identifiers
- Use code-specific entity types

### 3. False Positive Suppression Techniques
- Implement SQL keyword detection
- Use AST (Abstract Syntax Tree) parsing for better context
- Apply different rules for different file types

## Recommendations

### 1. Immediate Improvements to `isValidPersonName`

Add SQL-specific filters:

```go
// SQL-specific terms to filter out
sqlTerms := []string{
    // Table name patterns
    "staging table", "audit table", "temp table", "backup table",
    "transaction history", "system reports", "daily reports",
    "customer profile", "customer accounts", "user accounts",

    // Column name patterns
    "account balance", "account status", "account number",
    "customer name", "user name", "report name",
    "product type", "customer type", "account type",
    "transaction date", "report date", "created date",
    "pending product", "contract account", "savings account",
    "premium banking", "everyday banking", "interest rate",

    // SQL operations
    "calculate interest", "process payment", "validate account",

    // Database terms
    "production database", "staging database", "test database",
}
```

### 2. Enhanced Context Detection

Implement SQL context detection:

```go
func (d *detector) isInSQLContext(finding Finding, content string) bool {
    line := d.getLineContent(content, finding.Line)
    lineLower := strings.ToLower(line)

    // SQL DDL keywords
    sqlDDL := []string{"create table", "alter table", "drop table",
                       "create view", "create function", "create procedure"}

    // SQL DML keywords
    sqlDML := []string{"select", "insert into", "update", "delete from",
                       "from", "where", "join", "group by", "order by"}

    // Check for SQL keywords in the line
    for _, keyword := range append(sqlDDL, sqlDML...) {
        if strings.Contains(lineLower, keyword) {
            return true
        }
    }

    // Check if match is between backticks or square brackets (SQL identifiers)
    if strings.Contains(line, "`"+finding.Match+"`") ||
       strings.Contains(line, "["+finding.Match+"]") {
        return true
    }

    return false
}
```

### 3. File Type Awareness

Add SQL file detection:

```go
func (d *detector) isSQLFile(filename string) bool {
    ext := strings.ToLower(filepath.Ext(filename))
    return ext == ".sql" || ext == ".ddl" || ext == ".dml"
}
```

### 4. Modified Validation Logic

Update the validator to consider SQL context:

```go
validator: func(match string) bool {
    // Skip validation if in SQL context
    if d.isInSQLContext(finding, fileContent) {
        return false
    }

    // For SQL files, be more restrictive
    if d.isSQLFile(finding.File) {
        // Only match if it's in a string literal or comment
        line := d.getLineContent(fileContent, finding.Line)
        if !strings.Contains(line, "'"+match+"'") &&
           !strings.Contains(line, "\""+match+"\"") &&
           !strings.Contains(line, "--") &&
           !strings.Contains(line, "/*") {
            return false
        }
    }

    return d.isValidPersonName(match)
}
```

### 5. LLM Integration Enhancement

Since the system uses a two-phase architecture, enhance the LLM prompt to specifically handle SQL contexts:

```go
// In LLM validation prompt
"Context-specific guidelines:
- For SQL files, table names and column names are NOT personal names
- Common SQL patterns like 'Customer Name', 'Account Status' are identifiers
- Only flag actual personal data in SQL string literals or comments
- Consider CREATE TABLE, ALTER TABLE, SELECT statements as code context"
```

### 6. Alternative Approach: AST-Based Detection

For more sophisticated detection, integrate with SQL parsers:

```go
import "github.com/xwb1989/sqlparser"

func (d *detector) parseSQL(content string) (*sqlparser.Statement, error) {
    // Parse SQL to understand structure
    stmt, err := sqlparser.Parse(content)
    if err != nil {
        return nil, err
    }
    return &stmt, nil
}
```

## Implementation Priority

1. **High Priority**: Add SQL-specific terms to `isValidPersonName`
2. **High Priority**: Implement basic SQL context detection
3. **Medium Priority**: Add file type awareness
4. **Medium Priority**: Enhance LLM prompts for SQL contexts
5. **Low Priority**: Implement AST-based parsing (requires additional dependencies)

## Testing Strategy

1. Create comprehensive test suite with SQL examples
2. Include various SQL dialects and styles
3. Test with real-world SQL schemas
4. Measure false positive reduction
5. Ensure legitimate names in SQL comments/strings are still detected

## Conclusion

The current name detection implementation generates many false positives in SQL contexts because it lacks SQL-specific awareness. By implementing the recommended improvements, particularly SQL context detection and an expanded filter list, the false positive rate can be significantly reduced while maintaining the ability to detect legitimate personal names in SQL files (such as in comments or string literals).

The two-phase architecture provides a safety net through LLM validation, but reducing false positives at the pattern detection phase will improve efficiency and reduce the load on the LLM validation step.
