# Multi-stage build for GitHub PI Scanner
# Stage 1: Build Go application
FROM golang:1.27-alpine AS builder

# Build arguments
ARG VERSION="dev"
ARG COMMIT="unknown"
ARG BUILD_DATE=""

# Install build dependencies
RUN apk add --no-cache git gcc musl-dev

# Set working directory
WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application with version information
RUN go build -ldflags="-s -w \
    -X 'main.version=${VERSION}' \
    -X 'main.commit=${COMMIT}' \
    -X 'main.buildDate=${BUILD_DATE}'" \
    -o pi-scanner ./cmd/pi-scanner

# Runtime stage
FROM alpine:3.22

# Build arguments for labels
ARG VERSION="dev"
ARG COMMIT="unknown"
ARG BUILD_DATE=""

# Labels
LABEL org.opencontainers.image.title="GitHub PI Scanner"
LABEL org.opencontainers.image.description="Scanner for detecting personally identifiable information (PI) in GitHub repositories"
LABEL org.opencontainers.image.version="${VERSION}"
LABEL org.opencontainers.image.revision="${COMMIT}"
LABEL org.opencontainers.image.created="${BUILD_DATE}"
LABEL org.opencontainers.image.source="https://github.com/MacAttak/pi-scanner"
LABEL org.opencontainers.image.licenses="MIT"

# Install runtime dependencies
RUN apk add --no-cache \
    ca-certificates \
    git

# Create non-root user
RUN adduser -D -u 1000 scanner

# Copy binary from builder
COPY --from=builder /build/pi-scanner /usr/local/bin/pi-scanner

# Copy config files
COPY --from=builder /build/config /etc/pi-scanner/config

# Create directories
RUN mkdir -p /home/scanner/output && \
    chown -R scanner:scanner /home/scanner

# Switch to non-root user
USER scanner
WORKDIR /home/scanner

# Default command
ENTRYPOINT ["pi-scanner"]
CMD ["--help"]
