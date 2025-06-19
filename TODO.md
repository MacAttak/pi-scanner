# PI Scanner TODO List

## Critical Issues to Fix

### 1. CI/CD Pipeline Failures
- **Missing package**: `pkg/ml/models` is imported but doesn't exist
  - Need to either create this package or update imports
- **Tokenizers linking**: Tests fail with "cannot find -ltokenizers"
  - Need to configure CI to build/download tokenizers library
- **Test failures**:
  - `TestScanCommand/scan_with_invalid_repo_URL` - validation not working
  - `TestGitleaksDetector_ContextModifier` - gitleaks not detecting test secrets
  - Several proximity detection tests failing

### 2. Docker Build Issues
- **Tokenizers build failing**: Missing C++ compiler in Alpine image
  - Added `g++` to Dockerfile but needs testing
- **ONNX Runtime architecture**: Need to handle ARM64 vs x64 properly
  - Already updated Dockerfile to use TARGETPLATFORM
- **Build time**: Multi-stage builds taking very long
  - Consider pre-built base images or caching strategy

### 3. Release Process
- GitHub release created as draft at: https://github.com/MacAttak/pi-scanner/releases/tag/v1.0.0
- Need to fix CI before making release public
- Docker images not yet published to ghcr.io/MacAttak/pi-scanner

## Fixes Needed

### Immediate (Before Release)
1. Fix missing `pkg/ml/models` package
2. Update CI workflow to handle tokenizers library
3. Fix failing tests
4. Complete Docker build and publish

### Short Term
1. Add pre-commit hooks for linting and tests
2. Set up automated dependency updates
3. Add integration tests for Docker image
4. Create Homebrew formula

### Long Term
1. Move to organization account if needed
2. Add Windows and Linux package managers
3. Set up security scanning in CI
4. Add performance benchmarks to CI

## Notes
- All code has been updated to use `github.com/MacAttak/pi-scanner`
- Repository is public at https://github.com/MacAttak/pi-scanner
- Using GitHub Container Registry (ghcr.io) not Docker Hub