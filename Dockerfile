# Multi-stage build for GitHub PI Scanner
# Stage 1: Build tokenizers library
FROM rust:1.75-alpine AS tokenizers-builder

# Install build dependencies
RUN apk add --no-cache git gcc g++ musl-dev make cmake

# Build tokenizers
WORKDIR /build
RUN git clone https://github.com/daulet/tokenizers.git && \
    cd tokenizers && \
    git checkout v1.20.2 && \
    make build

# Stage 2: Build Go application
FROM golang:1.23-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git gcc musl-dev

# Set working directory
WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy tokenizers library from previous stage
COPY --from=tokenizers-builder /build/tokenizers/libtokenizers.a /usr/local/lib/
COPY --from=tokenizers-builder /build/tokenizers/tokenizers.h /usr/local/include/

# Copy source code
COPY . .

# Build the application with proper library paths
ENV CGO_ENABLED=1
ENV CGO_LDFLAGS="-L/usr/local/lib"
ENV CGO_CFLAGS="-I/usr/local/include"
RUN go build -ldflags="-s -w" -o pi-scanner ./cmd/pi-scanner

# Runtime stage
FROM ubuntu:22.04

# Install runtime dependencies
RUN apt-get update && \
    apt-get install -y --no-install-recommends \
    ca-certificates \
    git \
    wget \
    && rm -rf /var/lib/apt/lists/*

# Install ONNX Runtime (architecture-specific)
ARG ONNX_VERSION=1.22.0
ARG TARGETPLATFORM
RUN if [ "$TARGETPLATFORM" = "linux/arm64" ]; then \
        ONNX_ARCH="aarch64"; \
    else \
        ONNX_ARCH="x64"; \
    fi && \
    wget -q https://github.com/microsoft/onnxruntime/releases/download/v${ONNX_VERSION}/onnxruntime-linux-${ONNX_ARCH}-${ONNX_VERSION}.tgz && \
    tar -xzf onnxruntime-linux-${ONNX_ARCH}-${ONNX_VERSION}.tgz && \
    cp -r onnxruntime-linux-${ONNX_ARCH}-${ONNX_VERSION}/lib/*.so* /usr/local/lib/ && \
    rm -rf onnxruntime-linux-${ONNX_ARCH}-${ONNX_VERSION}* && \
    ldconfig

# Create non-root user
RUN useradd -m -u 1000 scanner

# Copy binary from builder
COPY --from=builder /build/pi-scanner /usr/local/bin/pi-scanner

# Create directories
RUN mkdir -p /home/scanner/.pi-scanner/models /home/scanner/output && \
    chown -R scanner:scanner /home/scanner

# Switch to non-root user
USER scanner
WORKDIR /home/scanner

# Set environment variables
ENV ONNX_RUNTIME_PATH="/usr/local/lib/libonnxruntime.so"
ENV HOME="/home/scanner"

# Default command
ENTRYPOINT ["pi-scanner"]
CMD ["--help"]