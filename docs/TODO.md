# PI Scanner TODO List

## Critical Issues to Fix

### 1. CI/CD Pipeline Fixes
- **Test failures**:
  - `TestScanCommand/scan_with_invalid_repo_URL` - validation not working
  - `TestGitleaksDetector_ContextModifier` - gitleaks not detecting test secrets
  - Several proximity detection tests failing
  - File processor queue test has race condition

### 2. Docker Build Optimization
- **Build time**: Consider caching strategies for faster builds
- **Image size**: Optimize Alpine-based image for minimal size
- **Multi-arch support**: Ensure ARM64 and x64 builds work correctly

### 3. Release Process
- GitHub release created as draft at: https://github.com/MacAttak/pi-scanner/releases/tag/v1.0.0
- Need to fix CI before making release public
- Docker images not yet published to ghcr.io/MacAttak/pi-scanner

## Fixes Needed

### Immediate (Before Release)
1. Fix failing tests
2. Complete Docker build and publish
3. Update documentation to reflect context validation approach
4. Clean up repository structure

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
- Switched from ML validation to context validation for better performance