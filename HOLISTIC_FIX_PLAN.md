# PI Scanner Holistic Fix Plan

## Executive Summary

The PI Scanner project has three interconnected issues preventing release:
1. **CI/CD failures** due to missing native libraries (tokenizers, ONNX Runtime)
2. **Docker build failures** due to compilation complexity
3. **Test failures** from library dependencies and validation logic

All issues stem from the hybrid Go/C architecture required for ML features.

## Root Cause Analysis

### 1. Architecture Complexity
```
┌─────────────────┐     ┌──────────────────┐     ┌─────────────────┐
│   Go Code       │────▶│ CGO Bindings     │────▶│ Native Libraries│
│ (PI Detection)  │     │ (tokenizers.go)  │     │ (libtokenizers) │
└─────────────────┘     └──────────────────┘     └─────────────────┘
                                │
                                ▼
                        ┌──────────────────┐
                        │ ONNX Runtime     │
                        │ (libonnxruntime) │
                        └──────────────────┘
```

### 2. Dependency Chain
- **tokenizers Go package** → requires **libtokenizers.a** (Rust-compiled C library)
- **ONNX Runtime Go bindings** → requires **libonnxruntime.so**
- **CGO compilation** → requires platform-specific toolchains
- **Cross-platform builds** → impossible with CGO enabled

### 3. Current State Issues
- CI environment lacks native libraries
- Docker builds attempt to compile from source (slow, error-prone)
- Tests assume libraries are present
- No graceful degradation without ML features

## Comprehensive Solution Plan

### Phase 1: Immediate CI/CD Fix (Day 1)

#### 1.1 Replace CI Workflow
```bash
# Replace the broken workflow
mv .github/workflows/ci.yml .github/workflows/ci.yml.old
mv .github/workflows/ci-fixed.yml .github/workflows/ci.yml
```

**Key improvements in ci-fixed.yml:**
- Downloads pre-built tokenizers library
- Installs ONNX Runtime
- Proper CGO environment setup
- Fallback tests without ML
- Separate builds with/without ML support

#### 1.2 Add Build Tags for Optional ML
Create `pkg/ml/build_tags.go`:
```go
//go:build ml
// +build ml

package ml

const MLEnabled = true
```

Create `pkg/ml/build_tags_no_ml.go`:
```go
//go:build !ml
// +build !ml

package ml

const MLEnabled = false

// Stub implementations for non-ML builds
```

#### 1.3 Fix Test Failures
1. **Invalid repo URL test**: Update validation logic
2. **Gitleaks context tests**: Fix test data or expectations
3. **Proximity detection tests**: Update expected PI labels

### Phase 2: Docker Optimization (Day 1-2)

#### 2.1 Create Multi-Stage Dockerfile
```dockerfile
# Stage 1: Download pre-built libraries
FROM alpine:3.19 AS libs
RUN apk add --no-cache wget
WORKDIR /libs

# Download tokenizers
ARG TARGETPLATFORM
RUN if [ "$TARGETPLATFORM" = "linux/arm64" ]; then \
        wget -q https://github.com/daulet/tokenizers/releases/download/v1.20.2/libtokenizers.linux-aarch64.tar.gz && \
        tar -xzf libtokenizers.linux-aarch64.tar.gz; \
    else \
        wget -q https://github.com/daulet/tokenizers/releases/download/v1.20.2/libtokenizers.linux-x86_64.tar.gz && \
        tar -xzf libtokenizers.linux-x86_64.tar.gz; \
    fi

# Stage 2: Build Go binary
FROM golang:1.23-alpine AS builder
RUN apk add --no-cache git gcc musl-dev
WORKDIR /app

# Copy libraries
COPY --from=libs /libs/libtokenizers.a /app/lib/

# Copy source
COPY . .

# Build with proper flags
ENV CGO_ENABLED=1
ENV CGO_LDFLAGS="-L/app/lib"
ENV CGO_CFLAGS="-I/app/lib"
RUN go build -ldflags="-s -w" -o pi-scanner ./cmd/pi-scanner

# Stage 3: Runtime
FROM ubuntu:22.04
# ... (install ONNX Runtime as before)
COPY --from=builder /app/pi-scanner /usr/local/bin/
```

#### 2.2 Build Cache Strategy
- Use GitHub Actions cache for Docker layers
- Pre-build base images with libraries
- Push to ghcr.io/macattak/pi-scanner-base

### Phase 3: Architectural Improvements (Day 2-3)

#### 3.1 Create Library Manager
`pkg/ml/libs/manager.go`:
```go
package libs

import (
    "fmt"
    "os"
    "path/filepath"
    "runtime"
)

type LibraryManager struct {
    libDir string
}

func (lm *LibraryManager) EnsureLibraries() error {
    // Check if libraries exist
    // Download if missing
    // Set environment variables
}

func (lm *LibraryManager) GetTokenizersPath() string {
    return filepath.Join(lm.libDir, "libtokenizers.a")
}
```

#### 3.2 Implement Feature Flags
`pkg/config/features.go`:
```go
type Features struct {
    EnableML        bool `yaml:"enable_ml"`
    EnableGitleaks  bool `yaml:"enable_gitleaks"`
    EnableValidation bool `yaml:"enable_validation"`
}

func (f *Features) Available() []string {
    available := []string{"pattern_matching", "validation"}
    if f.EnableML && ml.MLEnabled {
        available = append(available, "ml_detection")
    }
    return available
}
```

#### 3.3 Add Graceful Degradation
```go
// In detector.go
func (d *Detector) Detect(ctx context.Context, content []byte) ([]Finding, error) {
    findings := []Finding{}
    
    // Always run pattern matching
    findings = append(findings, d.runPatternMatching(content)...)
    
    // Conditionally run ML
    if d.config.Features.EnableML && ml.MLEnabled {
        mlFindings, err := d.runMLDetection(content)
        if err != nil {
            log.Warnf("ML detection failed, continuing without: %v", err)
        } else {
            findings = append(findings, mlFindings...)
        }
    }
    
    return findings, nil
}
```

### Phase 4: Testing Strategy (Day 3)

#### 4.1 Separate Test Suites
```bash
# Create test scripts
scripts/test-core.sh      # Tests without ML
scripts/test-ml.sh        # Tests requiring ML
scripts/test-all.sh       # Runs both with appropriate setup
```

#### 4.2 Mock ML Components
```go
// pkg/ml/mock_detector.go
type MockMLDetector struct {
    mock.Mock
}

func (m *MockMLDetector) Detect(content []byte) ([]Detection, error) {
    args := m.Called(content)
    return args.Get(0).([]Detection), args.Error(1)
}
```

#### 4.3 Integration Test Setup
```yaml
# .github/workflows/integration-tests.yml
- name: Setup test environment
  run: |
    # Download test models
    ./scripts/download-test-models.sh
    # Setup test data
    ./scripts/setup-test-data.sh
```

### Phase 5: Release Preparation (Day 4)

#### 5.1 Version Matrix
| Platform | Architecture | ML Support | Distribution |
|----------|-------------|------------|--------------|
| Linux    | amd64       | ✅ Yes     | Binary + Docker |
| Linux    | arm64       | ✅ Yes     | Binary + Docker |
| macOS    | amd64       | ❌ No      | Binary only |
| macOS    | arm64       | ✅ Yes*    | Binary only |
| Windows  | amd64       | ❌ No      | Binary only |

*Requires manual library installation

#### 5.2 Documentation Updates
1. **INSTALL.md**: Platform-specific instructions
2. **DOCKER.md**: Docker usage guide
3. **ML_SETUP.md**: ML feature setup guide
4. **TROUBLESHOOTING.md**: Common issues

#### 5.3 Release Checklist
- [ ] CI/CD passing on main branch
- [ ] Docker builds for linux/amd64 and linux/arm64
- [ ] All tests passing (with appropriate feature flags)
- [ ] Documentation updated
- [ ] Release notes include limitations
- [ ] Binaries tested on each platform

## Implementation Timeline

### Day 1 (Immediate)
- [ ] Apply CI workflow fix
- [ ] Fix failing tests
- [ ] Implement build tags
- [ ] Test CI pipeline

### Day 2
- [ ] Optimize Dockerfile
- [ ] Build and test Docker images
- [ ] Push test images to registry

### Day 3
- [ ] Implement feature flags
- [ ] Add graceful degradation
- [ ] Create comprehensive test suite

### Day 4
- [ ] Final testing
- [ ] Documentation updates
- [ ] Release v1.0.0

## Risk Mitigation

1. **Library Availability**: Mirror tokenizers releases in project
2. **Platform Support**: Clear documentation on ML availability
3. **Performance**: Benchmark with/without ML features
4. **Maintenance**: Automated dependency updates

## Success Criteria

1. ✅ CI/CD pipeline passes consistently
2. ✅ Docker images build for both architectures
3. ✅ All tests pass (with appropriate flags)
4. ✅ Clear documentation on feature availability
5. ✅ Successful deployment on all target platforms

## Alternative Approach (If Time-Constrained)

### Quick Release Strategy
1. **v1.0.0**: Pattern matching + validation only (no ML)
   - Remove ML dependencies temporarily
   - All platforms supported equally
   - Quick release timeline

2. **v1.1.0**: Add ML features
   - Selective platform support
   - Docker-first for ML features
   - Comprehensive testing

This approach ensures users get value immediately while we perfect the ML integration.

## Conclusion

The holistic fix addresses:
- **Technical debt**: Proper dependency management
- **User experience**: Graceful feature degradation
- **Maintainability**: Clear separation of concerns
- **Release quality**: Comprehensive testing

Following this plan ensures a robust, production-ready release that handles the complexity of hybrid Go/C architecture while providing value across all platforms.