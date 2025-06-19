# CI/CD Failure Analysis for PI Scanner

## Executive Summary

The PI Scanner project has multiple interconnected CI/CD issues that prevent successful builds and tests in the GitHub Actions workflow. The primary issues are:

1. **Tokenizers library linking failure** - The C++ tokenizers library is not available in CI
2. **ONNX Runtime missing** - ML tests require ONNX runtime which isn't installed in CI
3. **Import path issues** - Some test failures indicate potential module import problems
4. **Docker build complexity** - Multi-stage builds with Rust compilation are failing

## Detailed Analysis

### 1. Tokenizers Library Linking Issue

**Root Cause**: The project depends on `github.com/daulet/tokenizers` v1.20.2, which requires a C library (`libtokenizers.a`) to be built from Rust source code.

**Current State**:
- Local development has `libtokenizers.a` in `/lib` directory
- CI environment doesn't have this library
- Tests fail with: `ld: library 'tokenizers' not found`

**Evidence**:
```bash
# From test output:
ld: library 'tokenizers' not found
clang: error: linker command failed with exit code 1
```

**Dependencies**:
- Requires Rust toolchain to build
- Needs CGO_LDFLAGS and CGO_CFLAGS properly set
- Library must be in the linker path

### 2. ONNX Runtime Missing

**Root Cause**: The ML components use ONNX Runtime for model inference, but it's not installed in the CI environment.

**Current State**:
- Tests fail with: `Error loading ONNX shared library "onnxruntime.so"`
- The code checks multiple paths but can't find the library
- Dockerfile attempts to install ONNX Runtime but CI doesn't use Docker for tests

**Evidence**:
```
failed to initialize ONNX runtime: Platform-specific initialization failed: 
Error loading ONNX shared library "onnxruntime.so": dlopen(onnxruntime.so, 0x0001)
```

**Dependencies**:
- ONNX Runtime must be downloaded and installed for the target platform
- Different paths for Linux/macOS/Windows
- Version compatibility with go bindings

### 3. Module Structure Issues

**Root Cause**: The TODO.md mentions "pkg/ml/models is imported but doesn't exist", but investigation shows the package does exist.

**Current State**:
- The package exists at `/pkg/ml/models/`
- Contains `deberta.go`, `deberta_test.go`, `downloader.go`, `downloader_test.go`
- The issue might be import path mismatches or test isolation

**Evidence**:
- Directory structure confirms package exists
- Tests fail due to library dependencies, not missing packages

### 4. Docker Build Issues

**Root Cause**: Multi-stage Docker build attempts to compile Rust code for tokenizers library.

**Current State**:
- Uses Alpine Linux which has limited Rust/C++ toolchain support
- Long build times due to Rust compilation
- Architecture-specific issues (ARM64 vs x64)

**Evidence from Dockerfile**:
```dockerfile
# Stage 1: Build tokenizers library
FROM rust:1.75-alpine AS tokenizers-builder
# Attempts to build from source
```

## CI Workflow Analysis

The GitHub Actions workflow (`ci.yml`) has these jobs:

1. **test** - Runs Go tests without setting up required libraries
2. **security** - Runs security scanners (likely passing)
3. **build** - Attempts cross-platform builds (fails due to CGO dependencies)
4. **benchmark** - Runs performance tests
5. **codeql** - Static analysis

**Key Issues in Workflow**:
- No step to install/build tokenizers library
- No step to install ONNX Runtime
- CGO_ENABLED=1 but no C dependencies setup
- Cross-platform builds with CGO are complex

## Dependency Chain

```
PI Scanner Tests
    ├── ML Package Tests
    │   ├── Tokenizers (C library)
    │   │   └── Requires Rust build or pre-built binary
    │   └── ONNX Runtime
    │       └── Platform-specific shared library
    └── Other Package Tests (passing)
```

## Impact Analysis

1. **Development**: Local builds work with manual library setup
2. **CI/CD**: All builds fail, blocking PRs and releases
3. **Docker**: Builds are slow and may fail on Rust compilation
4. **Release**: v1.0.0 draft release blocked by CI failures

## Recommendations

### Immediate Fixes

1. **Add Library Setup to CI**:
   - Download pre-built tokenizers library
   - Install ONNX Runtime in CI environment
   - Set proper environment variables

2. **Create CI-specific Test Tags**:
   - Tag ML tests to skip in CI if needed
   - Run basic tests without ML dependencies

3. **Fix Docker Build**:
   - Use pre-built tokenizers library instead of compiling
   - Create base image with dependencies

### Long-term Solutions

1. **Vendoring Dependencies**:
   - Include pre-built libraries for common platforms
   - Use GitHub releases to store binary artifacts

2. **Conditional Compilation**:
   - Make ML features optional with build tags
   - Allow building without tokenizers/ONNX

3. **CI Optimization**:
   - Cache built dependencies
   - Use matrix builds for different feature sets

## Test Failure Summary

**Passing Tests**:
- Config package
- Detection/proximity package  
- Validation package
- Other non-ML packages

**Failing Tests**:
- ML/models package (tokenizers linking)
- ML/inference package (ONNX runtime)
- Any package importing ML components

## Next Steps

1. **Priority 1**: Fix tokenizers library in CI
   - Add build step or download pre-built library
   - Update CI workflow with proper paths

2. **Priority 2**: Install ONNX Runtime in CI
   - Add installation step for ubuntu-latest
   - Set library paths correctly

3. **Priority 3**: Optimize Docker builds
   - Create base image with dependencies
   - Use build cache effectively

4. **Priority 4**: Add integration tests
   - Test full binary with all features
   - Verify Docker image functionality