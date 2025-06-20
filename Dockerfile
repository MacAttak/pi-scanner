# Multi-stage build for GitHub PI Scanner
# Stage 1: Build Go application
FROM golang:1.23-alpine AS builder

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

# Build the application
RUN go build -ldflags="-s -w" -o pi-scanner ./cmd/pi-scanner

# Runtime stage
FROM alpine:3.19

# Install runtime dependencies
RUN apk add --no-cache \
    ca-certificates \
    git

# Create non-root user
RUN adduser -D -u 1000 scanner

# Copy binary from builder
COPY --from=builder /build/pi-scanner /usr/local/bin/pi-scanner

# Create directories
RUN mkdir -p /home/scanner/output && \
    chown -R scanner:scanner /home/scanner

# Switch to non-root user
USER scanner
WORKDIR /home/scanner

# Default command
ENTRYPOINT ["pi-scanner"]
CMD ["--help"]