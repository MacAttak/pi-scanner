# Docker Deployment Guide

This guide covers deploying and using PI Scanner with Docker.

## Available Images

The PI Scanner Docker images are hosted on GitHub Container Registry (ghcr.io).

### Image Tags

- `latest` - Latest stable release
- `1.2.0`, `1.2`, `1` - Specific version tags
- `main` - Latest build from main branch
- `develop` - Latest build from develop branch

### Multi-Architecture Support

Images are built for multiple architectures:
- `linux/amd64` - Intel/AMD 64-bit
- `linux/arm64` - ARM 64-bit (Apple Silicon, AWS Graviton)

## Quick Start

### Pull the Image

```bash
# Latest stable version
docker pull ghcr.io/macattak/pi-scanner:latest

# Specific version
docker pull ghcr.io/macattak/pi-scanner:1.2.0
```

### Basic Usage

```bash
# Run with GitHub token
docker run --rm \
  -e GITHUB_TOKEN=$GITHUB_TOKEN \
  ghcr.io/macattak/pi-scanner:latest \
  https://github.com/example/repo
```

## Advanced Configuration

### Volume Mounts

```bash
# Mount output directory
docker run --rm \
  -e GITHUB_TOKEN=$GITHUB_TOKEN \
  -v $(pwd)/reports:/home/scanner/output \
  ghcr.io/macattak/pi-scanner:latest \
  https://github.com/example/repo

# Mount custom config
docker run --rm \
  -e GITHUB_TOKEN=$GITHUB_TOKEN \
  -v $(pwd)/config.yaml:/etc/pi-scanner/config/config.yaml:ro \
  ghcr.io/macattak/pi-scanner:latest \
  https://github.com/example/repo
```

### Environment Variables

```bash
# Disable color output
docker run --rm \
  -e GITHUB_TOKEN=$GITHUB_TOKEN \
  -e NO_COLOR=1 \
  ghcr.io/macattak/pi-scanner:latest \
  https://github.com/example/repo

# CI mode (non-interactive)
docker run --rm \
  -e GITHUB_TOKEN=$GITHUB_TOKEN \
  -e CI=true \
  ghcr.io/macattak/pi-scanner:latest \
  https://github.com/example/repo
```

## Docker Compose

### Basic Configuration

```yaml
version: '3.8'

services:
  pi-scanner:
    image: ghcr.io/macattak/pi-scanner:latest
    environment:
      - GITHUB_TOKEN=${GITHUB_TOKEN}
    volumes:
      - ./reports:/home/scanner/output
```

### With LLM Service

```yaml
version: '3.8'

services:
  pi-scanner:
    image: ghcr.io/macattak/pi-scanner:latest
    environment:
      - GITHUB_TOKEN=${GITHUB_TOKEN}
      - LLM_ENDPOINT=http://llm-service:8080/v1
    volumes:
      - ./reports:/home/scanner/output
    depends_on:
      - llm-service
    networks:
      - scanner-network

  llm-service:
    image: your-llm-service:latest
    ports:
      - "8080:8080"
    networks:
      - scanner-network

networks:
  scanner-network:
    driver: bridge
```

## Kubernetes Deployment

### ConfigMap for Configuration

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: pi-scanner-config
data:
  config.yaml: |
    detection:
      confidence_threshold: 0.8
    validation:
      llm_endpoint: http://llm-service:8080/v1
```

### Job Definition

```yaml
apiVersion: batch/v1
kind: Job
metadata:
  name: pi-scanner-job
spec:
  template:
    spec:
      containers:
      - name: pi-scanner
        image: ghcr.io/macattak/pi-scanner:latest
        env:
        - name: GITHUB_TOKEN
          valueFrom:
            secretKeyRef:
              name: github-credentials
              key: token
        args:
        - "https://github.com/example/repo"
        - "--no-input"
        - "--validate=high"
        volumeMounts:
        - name: config
          mountPath: /etc/pi-scanner/config
        - name: output
          mountPath: /home/scanner/output
      volumes:
      - name: config
        configMap:
          name: pi-scanner-config
      - name: output
        persistentVolumeClaim:
          claimName: scanner-output-pvc
      restartPolicy: Never
```

### CronJob for Scheduled Scans

```yaml
apiVersion: batch/v1
kind: CronJob
metadata:
  name: pi-scanner-cronjob
spec:
  schedule: "0 2 * * *"  # Daily at 2 AM
  jobTemplate:
    spec:
      template:
        spec:
          containers:
          - name: pi-scanner
            image: ghcr.io/macattak/pi-scanner:latest
            env:
            - name: GITHUB_TOKEN
              valueFrom:
                secretKeyRef:
                  name: github-credentials
                  key: token
            args:
            - "https://github.com/example/repo"
            - "--no-input"
            - "--validate=high"
            - "--masking=full"
          restartPolicy: OnFailure
```

## CI/CD Integration

### GitHub Actions

```yaml
name: PI Security Scan

on:
  push:
    branches: [main]
  pull_request:
    branches: [main]

jobs:
  scan:
    runs-on: ubuntu-latest

    steps:
    - name: Checkout code
      uses: actions/checkout@v4

    - name: Run PI Scanner
      run: |
        docker run --rm \
          -e GITHUB_TOKEN=${{ secrets.GITHUB_TOKEN }} \
          -v ${{ github.workspace }}/reports:/home/scanner/output \
          ghcr.io/macattak/pi-scanner:latest \
          ${{ github.event.repository.html_url }} \
          --no-input --validate=high --masking=full

    - name: Upload scan results
      uses: actions/upload-artifact@v4
      with:
        name: pi-scan-results
        path: reports/
```

### GitLab CI

```yaml
pi-security-scan:
  image: docker:latest
  services:
    - docker:dind
  script:
    - docker run --rm
        -e GITHUB_TOKEN=$GITHUB_TOKEN
        -v $CI_PROJECT_DIR/reports:/home/scanner/output
        ghcr.io/macattak/pi-scanner:latest
        $CI_PROJECT_URL
        --no-input --validate=high
  artifacts:
    paths:
      - reports/
    expire_in: 1 week
```

### Jenkins Pipeline

```groovy
pipeline {
    agent any

    environment {
        GITHUB_TOKEN = credentials('github-token')
    }

    stages {
        stage('PI Security Scan') {
            steps {
                sh '''
                    docker run --rm \
                        -e GITHUB_TOKEN=$GITHUB_TOKEN \
                        -v ${WORKSPACE}/reports:/home/scanner/output \
                        ghcr.io/macattak/pi-scanner:latest \
                        https://github.com/example/repo \
                        --no-input --validate=high
                '''
            }
        }

        stage('Archive Results') {
            steps {
                archiveArtifacts artifacts: 'reports/**/*',
                                 allowEmptyArchive: false
            }
        }
    }
}
```

## Security Considerations

### Image Signing

All release images are signed with cosign. To verify:

```bash
# Install cosign
brew install cosign

# Verify image signature
cosign verify ghcr.io/macattak/pi-scanner:1.2.0 \
  --certificate-identity-regexp "https://github.com/MacAttak/pi-scanner/*" \
  --certificate-oidc-issuer "https://token.actions.githubusercontent.com"
```

### SBOM (Software Bill of Materials)

Each image includes an SBOM that can be retrieved:

```bash
# Download SBOM
cosign download sbom ghcr.io/macattak/pi-scanner:1.2.0
```

### Vulnerability Scanning

Images are automatically scanned with Trivy. To scan manually:

```bash
docker run --rm \
  -v /var/run/docker.sock:/var/run/docker.sock \
  aquasec/trivy image ghcr.io/macattak/pi-scanner:latest
```

## Troubleshooting

### Permission Denied

If you encounter permission issues with output files:

```bash
# Run with current user's UID/GID
docker run --rm \
  --user $(id -u):$(id -g) \
  -e GITHUB_TOKEN=$GITHUB_TOKEN \
  -v $(pwd)/reports:/home/scanner/output \
  ghcr.io/macattak/pi-scanner:latest \
  https://github.com/example/repo
```

### Network Issues

For environments with proxy requirements:

```bash
docker run --rm \
  -e GITHUB_TOKEN=$GITHUB_TOKEN \
  -e HTTP_PROXY=$HTTP_PROXY \
  -e HTTPS_PROXY=$HTTPS_PROXY \
  -e NO_PROXY=$NO_PROXY \
  ghcr.io/macattak/pi-scanner:latest \
  https://github.com/example/repo
```

### Debug Mode

To troubleshoot issues:

```bash
# Run with verbose output
docker run --rm \
  -e GITHUB_TOKEN=$GITHUB_TOKEN \
  ghcr.io/macattak/pi-scanner:latest \
  https://github.com/example/repo --verbose

# Get shell access
docker run --rm -it \
  --entrypoint /bin/sh \
  ghcr.io/macattak/pi-scanner:latest
```

## Performance Optimization

### Resource Limits

```bash
# Limit CPU and memory
docker run --rm \
  --cpus="2.0" \
  --memory="4g" \
  -e GITHUB_TOKEN=$GITHUB_TOKEN \
  ghcr.io/macattak/pi-scanner:latest \
  https://github.com/example/repo
```

### Parallel Scanning

For scanning multiple repositories:

```bash
#!/bin/bash
repos=("repo1" "repo2" "repo3")

for repo in "${repos[@]}"; do
  docker run -d \
    --name "scan-$repo" \
    -e GITHUB_TOKEN=$GITHUB_TOKEN \
    -v $(pwd)/reports/$repo:/home/scanner/output \
    ghcr.io/macattak/pi-scanner:latest \
    "https://github.com/example/$repo" \
    --no-input --validate=high
done

# Wait for all containers to finish
for repo in "${repos[@]}"; do
  docker wait "scan-$repo"
  docker rm "scan-$repo"
done
```
