# PI Scanner API Design

## Overview

This document defines the REST API and SDK interfaces for the PI Scanner, enabling programmatic integration with enterprise systems.

## API Architecture

### Design Principles

1. **RESTful Design**: Resource-based URLs with standard HTTP methods
2. **Versioning**: URL-based versioning for backward compatibility
3. **Security First**: Authentication, authorization, and encryption required
4. **Rate Limiting**: Protect against abuse and ensure fair usage
5. **Async Operations**: Long-running scans handled asynchronously

### API Structure

```
https://api.pi-scanner.local/v1/
├── /auth
│   ├── POST   /token        # Get access token
│   └── POST   /refresh      # Refresh token
├── /scans
│   ├── GET    /             # List scans
│   ├── POST   /             # Create scan
│   ├── GET    /{id}         # Get scan details
│   ├── DELETE /{id}         # Cancel scan
│   └── GET    /{id}/results # Get scan results
├── /repositories
│   ├── GET    /             # List repositories
│   ├── POST   /validate     # Validate repository access
│   └── GET    /{id}/stats   # Repository statistics
├── /patterns
│   ├── GET    /             # List detection patterns
│   ├── POST   /             # Add custom pattern
│   ├── PUT    /{id}         # Update pattern
│   └── DELETE /{id}         # Delete pattern
├── /reports
│   ├── POST   /generate     # Generate report
│   ├── GET    /{id}         # Download report
│   └── GET    /templates    # List report templates
└── /admin
    ├── GET    /health       # Health check
    ├── GET    /metrics      # Prometheus metrics
    └── GET    /config       # Current configuration
```

## Authentication & Authorization

### JWT Authentication

```go
// pkg/api/auth/jwt.go
package auth

import (
    "time"
    "github.com/golang-jwt/jwt/v5"
)

type TokenClaims struct {
    UserID      string   `json:"user_id"`
    Email       string   `json:"email"`
    Roles       []string `json:"roles"`
    Permissions []string `json:"permissions"`
    jwt.RegisteredClaims
}

type TokenService struct {
    signingKey []byte
    issuer     string
    audience   string
    expiry     time.Duration
}

func (s *TokenService) GenerateToken(user User) (string, error) {
    claims := TokenClaims{
        UserID:      user.ID,
        Email:       user.Email,
        Roles:       user.Roles,
        Permissions: user.GetPermissions(),
        RegisteredClaims: jwt.RegisteredClaims{
            Issuer:    s.issuer,
            Subject:   user.ID,
            Audience:  jwt.ClaimStrings{s.audience},
            ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.expiry)),
            NotBefore: jwt.NewNumericDate(time.Now()),
            IssuedAt:  jwt.NewNumericDate(time.Now()),
            ID:        generateTokenID(),
        },
    }

    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    return token.SignedString(s.signingKey)
}

func (s *TokenService) ValidateToken(tokenString string) (*TokenClaims, error) {
    token, err := jwt.ParseWithClaims(tokenString, &TokenClaims{}, func(token *jwt.Token) (interface{}, error) {
        if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
            return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
        }
        return s.signingKey, nil
    })

    if err != nil {
        return nil, err
    }

    if claims, ok := token.Claims.(*TokenClaims); ok && token.Valid {
        return claims, nil
    }

    return nil, fmt.Errorf("invalid token")
}
```

### API Key Authentication

```go
// pkg/api/auth/apikey.go
type APIKey struct {
    ID          string    `json:"id"`
    Key         string    `json:"-"`
    HashedKey   string    `json:"hashed_key"`
    Name        string    `json:"name"`
    Permissions []string  `json:"permissions"`
    RateLimit   int       `json:"rate_limit"`
    ExpiresAt   time.Time `json:"expires_at"`
    CreatedAt   time.Time `json:"created_at"`
}

type APIKeyService struct {
    store KeyStore
    hasher Hasher
}

func (s *APIKeyService) CreateAPIKey(name string, permissions []string) (*APIKey, string, error) {
    // Generate secure random key
    rawKey := make([]byte, 32)
    if _, err := rand.Read(rawKey); err != nil {
        return nil, "", err
    }

    key := base64.URLEncoding.EncodeToString(rawKey)
    hashedKey := s.hasher.Hash(key)

    apiKey := &APIKey{
        ID:          generateID(),
        HashedKey:   hashedKey,
        Name:        name,
        Permissions: permissions,
        RateLimit:   1000, // requests per hour
        ExpiresAt:   time.Now().Add(365 * 24 * time.Hour),
        CreatedAt:   time.Now(),
    }

    if err := s.store.Save(apiKey); err != nil {
        return nil, "", err
    }

    return apiKey, key, nil
}
```

## API Endpoints

### Scan Management

```go
// pkg/api/handlers/scans.go
package handlers

import (
    "net/http"
    "github.com/gin-gonic/gin"
)

type ScanHandler struct {
    scanner     Scanner
    validator   Validator
    authorizer  Authorizer
    auditLogger AuditLogger
}

// CreateScan - POST /api/v1/scans
func (h *ScanHandler) CreateScan(c *gin.Context) {
    var req CreateScanRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, ErrorResponse{
            Error: "Invalid request body",
            Details: err.Error(),
        })
        return
    }

    // Validate request
    if err := h.validator.ValidateScanRequest(req); err != nil {
        c.JSON(http.StatusBadRequest, ErrorResponse{
            Error: "Validation failed",
            Details: err.Error(),
        })
        return
    }

    // Check authorization
    user := GetUser(c)
    if !h.authorizer.CanCreateScan(user, req) {
        h.auditLogger.LogUnauthorized(user.ID, "create_scan", req.Repository)
        c.JSON(http.StatusForbidden, ErrorResponse{
            Error: "Insufficient permissions",
        })
        return
    }

    // Create scan job
    scan, err := h.scanner.CreateScan(c.Request.Context(), ScanConfig{
        Repository:      req.Repository,
        Branch:          req.Branch,
        IncludePatterns: req.IncludePatterns,
        ExcludePatterns: req.ExcludePatterns,
        EnableLLM:       req.EnableLLM,
        RedactionLevel:  req.RedactionLevel,
    })

    if err != nil {
        c.JSON(http.StatusInternalServerError, ErrorResponse{
            Error: "Failed to create scan",
            Details: err.Error(),
        })
        return
    }

    // Log scan creation
    h.auditLogger.LogScanCreated(user.ID, scan.ID, req.Repository)

    c.JSON(http.StatusCreated, ScanResponse{
        ID:        scan.ID,
        Status:    scan.Status,
        CreatedAt: scan.CreatedAt,
        Links: Links{
            Self:    fmt.Sprintf("/api/v1/scans/%s", scan.ID),
            Results: fmt.Sprintf("/api/v1/scans/%s/results", scan.ID),
        },
    })
}

// GetScanResults - GET /api/v1/scans/{id}/results
func (h *ScanHandler) GetScanResults(c *gin.Context) {
    scanID := c.Param("id")

    // Get scan
    scan, err := h.scanner.GetScan(c.Request.Context(), scanID)
    if err != nil {
        c.JSON(http.StatusNotFound, ErrorResponse{
            Error: "Scan not found",
        })
        return
    }

    // Check authorization
    user := GetUser(c)
    if !h.authorizer.CanViewScan(user, scan) {
        c.JSON(http.StatusForbidden, ErrorResponse{
            Error: "Insufficient permissions",
        })
        return
    }

    // Get pagination parameters
    page := c.DefaultQuery("page", "1")
    limit := c.DefaultQuery("limit", "100")

    // Get results
    results, total, err := h.scanner.GetScanResults(c.Request.Context(), scanID, PaginationParams{
        Page:  parseInt(page),
        Limit: parseInt(limit),
    })

    if err != nil {
        c.JSON(http.StatusInternalServerError, ErrorResponse{
            Error: "Failed to get results",
        })
        return
    }

    // Apply redaction based on user permissions
    redactedResults := h.applyRedaction(user, results, scan.RedactionLevel)

    c.JSON(http.StatusOK, ResultsResponse{
        Results: redactedResults,
        Pagination: Pagination{
            Page:  parseInt(page),
            Limit: parseInt(limit),
            Total: total,
        },
    })
}
```

### Pattern Management

```go
// pkg/api/handlers/patterns.go
type PatternHandler struct {
    patternRegistry PatternRegistry
    validator       Validator
    authorizer      Authorizer
}

// CreatePattern - POST /api/v1/patterns
func (h *PatternHandler) CreatePattern(c *gin.Context) {
    var req CreatePatternRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, ErrorResponse{
            Error: "Invalid request body",
        })
        return
    }

    // Validate pattern
    if err := h.validator.ValidatePattern(req); err != nil {
        c.JSON(http.StatusBadRequest, ErrorResponse{
            Error: "Invalid pattern",
            Details: err.Error(),
        })
        return
    }

    // Check authorization
    user := GetUser(c)
    if !h.authorizer.HasPermission(user, "patterns:write") {
        c.JSON(http.StatusForbidden, ErrorResponse{
            Error: "Insufficient permissions",
        })
        return
    }

    // Create pattern
    pattern := Pattern{
        ID:              generateID(),
        Name:            req.Name,
        Type:            req.Type,
        Pattern:         req.Pattern,
        ContextKeywords: req.ContextKeywords,
        MinConfidence:   req.MinConfidence,
        CreatedBy:       user.ID,
        CreatedAt:       time.Now(),
    }

    if err := h.patternRegistry.Register(pattern); err != nil {
        c.JSON(http.StatusInternalServerError, ErrorResponse{
            Error: "Failed to create pattern",
        })
        return
    }

    c.JSON(http.StatusCreated, pattern)
}
```

### Report Generation

```go
// pkg/api/handlers/reports.go
type ReportHandler struct {
    reportGenerator ReportGenerator
    storage         Storage
    encryptor       Encryptor
}

// GenerateReport - POST /api/v1/reports/generate
func (h *ReportHandler) GenerateReport(c *gin.Context) {
    var req GenerateReportRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, ErrorResponse{
            Error: "Invalid request body",
        })
        return
    }

    // Get scan results
    results, err := h.getScanResults(req.ScanID)
    if err != nil {
        c.JSON(http.StatusNotFound, ErrorResponse{
            Error: "Scan not found",
        })
        return
    }

    // Generate report
    report, err := h.reportGenerator.Generate(GenerateConfig{
        Format:         req.Format,
        Template:       req.Template,
        RedactionLevel: req.RedactionLevel,
        IncludeContext: req.IncludeContext,
        Findings:       results.Findings,
    })

    if err != nil {
        c.JSON(http.StatusInternalServerError, ErrorResponse{
            Error: "Failed to generate report",
        })
        return
    }

    // Encrypt if requested
    if req.Encrypt {
        encrypted, err := h.encryptor.Encrypt(report.Data, req.EncryptionKey)
        if err != nil {
            c.JSON(http.StatusInternalServerError, ErrorResponse{
                Error: "Failed to encrypt report",
            })
            return
        }
        report.Data = encrypted
        report.Encrypted = true
    }

    // Store report
    reportID := generateID()
    if err := h.storage.Store(reportID, report); err != nil {
        c.JSON(http.StatusInternalServerError, ErrorResponse{
            Error: "Failed to store report",
        })
        return
    }

    c.JSON(http.StatusCreated, ReportResponse{
        ID:        reportID,
        Format:    report.Format,
        Size:      len(report.Data),
        Encrypted: report.Encrypted,
        ExpiresAt: time.Now().Add(24 * time.Hour),
        Links: Links{
            Download: fmt.Sprintf("/api/v1/reports/%s", reportID),
        },
    })
}
```

## API Models

### Request/Response Models

```go
// pkg/api/models/requests.go
package models

// CreateScanRequest represents a scan creation request
type CreateScanRequest struct {
    Repository      string            `json:"repository" binding:"required"`
    Branch          string            `json:"branch,omitempty"`
    IncludePatterns []string          `json:"include_patterns,omitempty"`
    ExcludePatterns []string          `json:"exclude_patterns,omitempty"`
    EnableLLM       bool              `json:"enable_llm"`
    LLMConfig       *LLMConfig        `json:"llm_config,omitempty"`
    RedactionLevel  RedactionLevel    `json:"redaction_level"`
    NotifyWebhook   string            `json:"notify_webhook,omitempty"`
    Tags            map[string]string `json:"tags,omitempty"`
}

// ScanResponse represents a scan creation response
type ScanResponse struct {
    ID        string    `json:"id"`
    Status    string    `json:"status"`
    CreatedAt time.Time `json:"created_at"`
    Links     Links     `json:"_links"`
}

// Finding represents a detected PI instance
type Finding struct {
    ID             string         `json:"id"`
    Type           string         `json:"type"`
    Match          string         `json:"match"`
    FilePath       string         `json:"file_path"`
    LineNumber     int            `json:"line_number"`
    Column         int            `json:"column"`
    Context        string         `json:"context,omitempty"`
    Confidence     float32        `json:"confidence"`
    RiskLevel      string         `json:"risk_level"`
    ValidationInfo ValidationInfo `json:"validation_info"`
}

// ResultsResponse represents paginated scan results
type ResultsResponse struct {
    Results    []Finding   `json:"results"`
    Summary    Summary     `json:"summary"`
    Pagination Pagination  `json:"pagination"`
    Links      Links       `json:"_links"`
}

// Summary provides aggregated statistics
type Summary struct {
    TotalFindings   int                 `json:"total_findings"`
    ByType          map[string]int      `json:"by_type"`
    ByRiskLevel     map[string]int      `json:"by_risk_level"`
    FilesScanned    int                 `json:"files_scanned"`
    FilesWithIssues int                 `json:"files_with_issues"`
    ScanDuration    string              `json:"scan_duration"`
}
```

### Error Response Format

```go
// pkg/api/models/errors.go
type ErrorResponse struct {
    Error   string      `json:"error"`
    Code    string      `json:"code,omitempty"`
    Details interface{} `json:"details,omitempty"`
    TraceID string      `json:"trace_id"`
}

// Standard error codes
const (
    ErrCodeValidation       = "VALIDATION_ERROR"
    ErrCodeAuthentication   = "AUTHENTICATION_ERROR"
    ErrCodeAuthorization    = "AUTHORIZATION_ERROR"
    ErrCodeNotFound         = "NOT_FOUND"
    ErrCodeRateLimit        = "RATE_LIMIT_EXCEEDED"
    ErrCodeInternalError    = "INTERNAL_ERROR"
)
```

## SDK Design

### Go SDK

```go
// sdk/go/client.go
package piscannerapi

import (
    "context"
    "net/http"
    "time"
)

// Client is the PI Scanner API client
type Client struct {
    baseURL    string
    apiKey     string
    httpClient *http.Client

    // Services
    Scans      *ScansService
    Patterns   *PatternsService
    Reports    *ReportsService
    Repositories *RepositoriesService
}

// NewClient creates a new API client
func NewClient(apiKey string, opts ...ClientOption) *Client {
    c := &Client{
        baseURL: "https://api.pi-scanner.local/v1",
        apiKey:  apiKey,
        httpClient: &http.Client{
            Timeout: 30 * time.Second,
        },
    }

    // Apply options
    for _, opt := range opts {
        opt(c)
    }

    // Initialize services
    c.Scans = &ScansService{client: c}
    c.Patterns = &PatternsService{client: c}
    c.Reports = &ReportsService{client: c}
    c.Repositories = &RepositoriesService{client: c}

    return c
}

// ScansService handles scan operations
type ScansService struct {
    client *Client
}

// Create initiates a new scan
func (s *ScansService) Create(ctx context.Context, req CreateScanRequest) (*Scan, error) {
    var scan Scan
    err := s.client.post(ctx, "/scans", req, &scan)
    return &scan, err
}

// Get retrieves scan details
func (s *ScansService) Get(ctx context.Context, scanID string) (*Scan, error) {
    var scan Scan
    err := s.client.get(ctx, fmt.Sprintf("/scans/%s", scanID), &scan)
    return &scan, err
}

// WaitForCompletion waits for a scan to complete
func (s *ScansService) WaitForCompletion(ctx context.Context, scanID string, opts ...WaitOption) (*Scan, error) {
    config := waitConfig{
        pollInterval: 5 * time.Second,
        timeout:      30 * time.Minute,
    }

    for _, opt := range opts {
        opt(&config)
    }

    ticker := time.NewTicker(config.pollInterval)
    defer ticker.Stop()

    timeout := time.After(config.timeout)

    for {
        select {
        case <-ctx.Done():
            return nil, ctx.Err()
        case <-timeout:
            return nil, fmt.Errorf("timeout waiting for scan completion")
        case <-ticker.C:
            scan, err := s.Get(ctx, scanID)
            if err != nil {
                return nil, err
            }

            switch scan.Status {
            case "completed":
                return scan, nil
            case "failed", "cancelled":
                return scan, fmt.Errorf("scan %s: %s", scan.Status, scan.Error)
            }
        }
    }
}

// Results retrieves scan results with pagination
func (s *ScansService) Results(ctx context.Context, scanID string, opts ...ResultOption) (*ResultsPage, error) {
    config := resultConfig{
        page:  1,
        limit: 100,
    }

    for _, opt := range opts {
        opt(&config)
    }

    var results ResultsPage
    err := s.client.get(ctx, fmt.Sprintf("/scans/%s/results?page=%d&limit=%d",
        scanID, config.page, config.limit), &results)
    return &results, err
}
```

### Python SDK

```python
# sdk/python/pi_scanner_api/client.py
import time
from typing import Optional, List, Dict, Any
import requests
from dataclasses import dataclass

@dataclass
class Scan:
    id: str
    status: str
    created_at: str
    repository: str
    branch: Optional[str] = None
    error: Optional[str] = None

@dataclass
class Finding:
    id: str
    type: str
    match: str
    file_path: str
    line_number: int
    confidence: float
    risk_level: str

class PIScannerclient:
    """PI Scanner API client for Python."""

    def __init__(self, api_key: str, base_url: str = "https://api.pi-scanner.local/v1"):
        self.api_key = api_key
        self.base_url = base_url
        self.session = requests.Session()
        self.session.headers.update({
            "Authorization": f"Bearer {api_key}",
            "Content-Type": "application/json"
        })

        # Initialize services
        self.scans = ScansService(self)
        self.patterns = PatternsService(self)
        self.reports = ReportsService(self)

    def _request(self, method: str, path: str, **kwargs) -> Dict[str, Any]:
        """Make an API request."""
        url = f"{self.base_url}{path}"
        response = self.session.request(method, url, **kwargs)
        response.raise_for_status()
        return response.json()

class ScansService:
    """Service for scan operations."""

    def __init__(self, client: PIScannerclient):
        self.client = client

    def create(self, repository: str, **kwargs) -> Scan:
        """Create a new scan."""
        data = {
            "repository": repository,
            **kwargs
        }

        result = self.client._request("POST", "/scans", json=data)
        return Scan(**result)

    def get(self, scan_id: str) -> Scan:
        """Get scan details."""
        result = self.client._request("GET", f"/scans/{scan_id}")
        return Scan(**result)

    def wait_for_completion(self, scan_id: str, timeout: int = 1800,
                          poll_interval: int = 5) -> Scan:
        """Wait for a scan to complete."""
        start_time = time.time()

        while time.time() - start_time < timeout:
            scan = self.get(scan_id)

            if scan.status in ["completed", "failed", "cancelled"]:
                return scan

            time.sleep(poll_interval)

        raise TimeoutError(f"Scan {scan_id} did not complete within {timeout} seconds")

    def get_results(self, scan_id: str, page: int = 1, limit: int = 100) -> List[Finding]:
        """Get scan results."""
        result = self.client._request(
            "GET",
            f"/scans/{scan_id}/results",
            params={"page": page, "limit": limit}
        )

        return [Finding(**f) for f in result["results"]]

# Example usage
if __name__ == "__main__":
    # Initialize client
    client = PIScannerclient(api_key="your-api-key")

    # Create a scan
    scan = client.scans.create(
        repository="https://github.com/example/repo",
        branch="main",
        enable_llm=True,
        redaction_level="partial"
    )

    print(f"Scan created: {scan.id}")

    # Wait for completion
    completed_scan = client.scans.wait_for_completion(scan.id)
    print(f"Scan completed with status: {completed_scan.status}")

    # Get results
    results = client.scans.get_results(scan.id)
    for finding in results:
        print(f"Found {finding.type} in {finding.file_path}:{finding.line_number}")
```

## API Security

### Rate Limiting

```go
// pkg/api/middleware/ratelimit.go
package middleware

import (
    "fmt"
    "net/http"
    "time"

    "github.com/gin-gonic/gin"
    "github.com/ulule/limiter/v3"
    "github.com/ulule/limiter/v3/drivers/store/redis"
)

func RateLimit(store limiter.Store) gin.HandlerFunc {
    // Define rate limit rules
    rate := limiter.Rate{
        Period: time.Hour,
        Limit:  1000, // Default limit
    }

    instance := limiter.New(store, rate)

    return func(c *gin.Context) {
        // Get identifier (API key or user ID)
        identifier := getIdentifier(c)

        // Apply custom limits based on user tier
        customRate := getCustomRate(identifier)
        if customRate != nil {
            instance = limiter.New(store, *customRate)
        }

        // Check rate limit
        context, err := instance.Get(c.Request.Context(), identifier)
        if err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{
                "error": "Internal server error",
            })
            c.Abort()
            return
        }

        // Add rate limit headers
        c.Header("X-RateLimit-Limit", fmt.Sprintf("%d", context.Limit))
        c.Header("X-RateLimit-Remaining", fmt.Sprintf("%d", context.Remaining))
        c.Header("X-RateLimit-Reset", fmt.Sprintf("%d", context.Reset))

        if context.Reached {
            c.JSON(http.StatusTooManyRequests, gin.H{
                "error": "Rate limit exceeded",
                "retry_after": context.Reset,
            })
            c.Abort()
            return
        }

        c.Next()
    }
}
```

### Request Validation

```go
// pkg/api/middleware/validation.go
func ValidateRequest() gin.HandlerFunc {
    return func(c *gin.Context) {
        // Validate content type
        if c.Request.Method != http.MethodGet &&
           c.GetHeader("Content-Type") != "application/json" {
            c.JSON(http.StatusBadRequest, gin.H{
                "error": "Content-Type must be application/json",
            })
            c.Abort()
            return
        }

        // Validate request size
        if c.Request.ContentLength > 10*1024*1024 { // 10MB limit
            c.JSON(http.StatusRequestEntityTooLarge, gin.H{
                "error": "Request body too large",
            })
            c.Abort()
            return
        }

        c.Next()
    }
}
```

## API Documentation

### OpenAPI Specification

```yaml
# api/openapi.yaml
openapi: 3.0.3
info:
  title: PI Scanner API
  description: API for scanning repositories for Australian Personally Identifiable Information
  version: 1.0.0
  contact:
    name: API Support
    email: support@pi-scanner.local

servers:
  - url: https://api.pi-scanner.local/v1
    description: Production server
  - url: http://localhost:8080/v1
    description: Development server

security:
  - BearerAuth: []
  - ApiKeyAuth: []

components:
  securitySchemes:
    BearerAuth:
      type: http
      scheme: bearer
      bearerFormat: JWT
    ApiKeyAuth:
      type: apiKey
      in: header
      name: X-API-Key

  schemas:
    CreateScanRequest:
      type: object
      required:
        - repository
      properties:
        repository:
          type: string
          format: uri
          example: "https://github.com/example/repo"
        branch:
          type: string
          example: "main"
        enable_llm:
          type: boolean
          default: false
        redaction_level:
          type: string
          enum: [none, partial, full]
          default: full

    Scan:
      type: object
      properties:
        id:
          type: string
          format: uuid
        status:
          type: string
          enum: [pending, running, completed, failed, cancelled]
        created_at:
          type: string
          format: date-time
        completed_at:
          type: string
          format: date-time
        repository:
          type: string
        statistics:
          $ref: '#/components/schemas/ScanStatistics'

    Finding:
      type: object
      properties:
        id:
          type: string
          format: uuid
        type:
          type: string
          enum: [TFN, ABN, MEDICARE, BSB, ACN, PHONE, EMAIL, NAME, ADDRESS, PASSPORT, DRIVERS_LICENSE]
        match:
          type: string
          description: The detected PI value (may be redacted)
        file_path:
          type: string
        line_number:
          type: integer
        confidence:
          type: number
          format: float
          minimum: 0
          maximum: 1
        risk_level:
          type: string
          enum: [CRITICAL, HIGH, MEDIUM, LOW]

paths:
  /scans:
    post:
      summary: Create a new scan
      operationId: createScan
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/CreateScanRequest'
      responses:
        '201':
          description: Scan created successfully
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Scan'
        '400':
          description: Invalid request
        '401':
          description: Unauthorized
        '403':
          description: Forbidden
        '429':
          description: Rate limit exceeded

  /scans/{scanId}:
    get:
      summary: Get scan details
      operationId: getScan
      parameters:
        - name: scanId
          in: path
          required: true
          schema:
            type: string
            format: uuid
      responses:
        '200':
          description: Scan details
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Scan'
        '404':
          description: Scan not found

  /scans/{scanId}/results:
    get:
      summary: Get scan results
      operationId: getScanResults
      parameters:
        - name: scanId
          in: path
          required: true
          schema:
            type: string
            format: uuid
        - name: page
          in: query
          schema:
            type: integer
            default: 1
        - name: limit
          in: query
          schema:
            type: integer
            default: 100
            maximum: 1000
      responses:
        '200':
          description: Scan results
          content:
            application/json:
              schema:
                type: object
                properties:
                  results:
                    type: array
                    items:
                      $ref: '#/components/schemas/Finding'
                  pagination:
                    type: object
                    properties:
                      page:
                        type: integer
                      limit:
                        type: integer
                      total:
                        type: integer
```

## Webhooks

### Webhook Configuration

```go
// pkg/api/webhooks/webhook.go
package webhooks

import (
    "bytes"
    "crypto/hmac"
    "crypto/sha256"
    "encoding/hex"
    "encoding/json"
    "net/http"
    "time"
)

type WebhookEvent struct {
    ID        string                 `json:"id"`
    Type      string                 `json:"type"`
    CreatedAt time.Time              `json:"created_at"`
    Data      map[string]interface{} `json:"data"`
}

type WebhookSender struct {
    client    *http.Client
    secret    string
    maxRetries int
}

func (w *WebhookSender) Send(url string, event WebhookEvent) error {
    // Marshal event
    payload, err := json.Marshal(event)
    if err != nil {
        return err
    }

    // Create signature
    signature := w.createSignature(payload)

    // Create request
    req, err := http.NewRequest("POST", url, bytes.NewBuffer(payload))
    if err != nil {
        return err
    }

    // Add headers
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("X-Webhook-Signature", signature)
    req.Header.Set("X-Webhook-ID", event.ID)
    req.Header.Set("X-Webhook-Timestamp", fmt.Sprintf("%d", time.Now().Unix()))

    // Send with retries
    return w.sendWithRetry(req)
}

func (w *WebhookSender) createSignature(payload []byte) string {
    h := hmac.New(sha256.New, []byte(w.secret))
    h.Write(payload)
    return hex.EncodeToString(h.Sum(nil))
}

// Webhook event types
const (
    EventScanCreated   = "scan.created"
    EventScanCompleted = "scan.completed"
    EventScanFailed    = "scan.failed"
    EventFindingHigh   = "finding.high_risk"
    EventReportReady   = "report.ready"
)
```

## CLI Integration

### CLI API Client

```go
// cmd/pi-scanner/api_client.go
package main

import (
    "fmt"
    "os"
    "time"

    "github.com/spf13/cobra"
    piscannerapi "github.com/pi-scanner/sdk/go"
)

var apiCmd = &cobra.Command{
    Use:   "api",
    Short: "Interact with PI Scanner API",
}

var apiScanCmd = &cobra.Command{
    Use:   "scan [repository]",
    Short: "Create a scan via API",
    Args:  cobra.ExactArgs(1),
    RunE: func(cmd *cobra.Command, args []string) error {
        // Get API key
        apiKey := os.Getenv("PI_SCANNER_API_KEY")
        if apiKey == "" {
            return fmt.Errorf("PI_SCANNER_API_KEY environment variable not set")
        }

        // Create client
        client := piscannerapi.NewClient(apiKey)

        // Create scan
        scan, err := client.Scans.Create(cmd.Context(), piscannerapi.CreateScanRequest{
            Repository:     args[0],
            Branch:         flagBranch,
            EnableLLM:      flagEnableLLM,
            RedactionLevel: flagRedactionLevel,
        })

        if err != nil {
            return fmt.Errorf("create scan: %w", err)
        }

        fmt.Printf("Scan created: %s\n", scan.ID)

        // Wait for completion if requested
        if flagWait {
            fmt.Println("Waiting for scan to complete...")

            completed, err := client.Scans.WaitForCompletion(cmd.Context(), scan.ID)
            if err != nil {
                return fmt.Errorf("wait for scan: %w", err)
            }

            fmt.Printf("Scan completed with status: %s\n", completed.Status)

            // Get results
            results, err := client.Scans.Results(cmd.Context(), scan.ID)
            if err != nil {
                return fmt.Errorf("get results: %w", err)
            }

            fmt.Printf("Found %d issues\n", results.Total)
        }

        return nil
    },
}
```

## Monitoring & Metrics

### Prometheus Metrics

```go
// pkg/api/metrics/metrics.go
package metrics

import (
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promauto"
)

var (
    // Request metrics
    RequestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
        Name: "pi_scanner_api_request_duration_seconds",
        Help: "API request duration in seconds",
        Buckets: prometheus.DefBuckets,
    }, []string{"method", "endpoint", "status"})

    // Scan metrics
    ScansCreated = promauto.NewCounter(prometheus.CounterOpts{
        Name: "pi_scanner_scans_created_total",
        Help: "Total number of scans created",
    })

    ScansActive = promauto.NewGauge(prometheus.GaugeOpts{
        Name: "pi_scanner_scans_active",
        Help: "Number of currently active scans",
    })

    // Finding metrics
    FindingsDetected = promauto.NewCounterVec(prometheus.CounterOpts{
        Name: "pi_scanner_findings_detected_total",
        Help: "Total number of findings detected",
    }, []string{"type", "risk_level"})

    // Rate limit metrics
    RateLimitHits = promauto.NewCounterVec(prometheus.CounterOpts{
        Name: "pi_scanner_rate_limit_hits_total",
        Help: "Total number of rate limit hits",
    }, []string{"identifier"})
)
```

## API Testing

### Integration Tests

```go
// tests/api/integration_test.go
package api_test

import (
    "testing"
    "net/http/httptest"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestAPI_ScanLifecycle(t *testing.T) {
    // Setup test server
    server := setupTestServer(t)
    defer server.Close()

    // Create client
    client := piscannerapi.NewClient("test-key",
        piscannerapi.WithBaseURL(server.URL))

    // Test scan creation
    t.Run("CreateScan", func(t *testing.T) {
        scan, err := client.Scans.Create(context.Background(),
            piscannerapi.CreateScanRequest{
                Repository: "https://github.com/test/repo",
                Branch:     "main",
            })

        require.NoError(t, err)
        assert.NotEmpty(t, scan.ID)
        assert.Equal(t, "pending", scan.Status)
    })

    // Test results retrieval
    t.Run("GetResults", func(t *testing.T) {
        // Wait for scan to complete
        scan, err := client.Scans.WaitForCompletion(context.Background(),
            scan.ID, piscannerapi.WithTimeout(1*time.Minute))

        require.NoError(t, err)
        assert.Equal(t, "completed", scan.Status)

        // Get results
        results, err := client.Scans.Results(context.Background(), scan.ID)
        require.NoError(t, err)
        assert.Greater(t, len(results.Results), 0)
    })
}

func TestAPI_RateLimiting(t *testing.T) {
    server := setupTestServer(t)
    defer server.Close()

    client := piscannerapi.NewClient("test-key",
        piscannerapi.WithBaseURL(server.URL))

    // Make requests up to limit
    for i := 0; i < 100; i++ {
        _, err := client.Scans.List(context.Background())
        require.NoError(t, err)
    }

    // Next request should be rate limited
    _, err := client.Scans.List(context.Background())
    require.Error(t, err)
    assert.Contains(t, err.Error(), "rate limit exceeded")
}
```

## Conclusion

This API design provides:

1. **RESTful Interface**: Standard HTTP methods and resource-based URLs
2. **Security**: JWT/API key auth, rate limiting, input validation
3. **SDKs**: Go and Python clients for easy integration
4. **Webhooks**: Real-time event notifications
5. **Documentation**: OpenAPI spec for automatic client generation
6. **Monitoring**: Prometheus metrics for observability

The API enables enterprise integration while maintaining security and performance standards.
