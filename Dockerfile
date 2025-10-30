# Multi-stage Dockerfile for Invoice OCR Service
# Optimized for Railway deployment (< 512MB RAM)

# Stage 1: Builder
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git build-base pkgconfig imagemagick-dev tesseract-ocr-dev

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build with optimizations for smaller binary and lower memory usage
RUN CGO_ENABLED=1 GOOS=linux go build \
    -a \
    -installsuffix cgo \
    -ldflags="-s -w" \
    -o server \
    ./cmd/server

# Stage 2: Runtime
FROM alpine:latest

# Install runtime dependencies (minimal set for Railway)
RUN apk add --no-cache \
    tesseract-ocr \
    tesseract-ocr-data-eng \
    tesseract-ocr-data-spa \
    imagemagick \
    ca-certificates \
    tzdata \
    wget

# Create non-root user for security
RUN addgroup -g 1000 app && \
    adduser -D -u 1000 -G app app

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/server .

# Copy config file
COPY config.yaml .

# Change ownership to non-root user
RUN chown -R app:app /app

# Switch to non-root user
USER app

# Port is configurable via environment variable (Railway sets $PORT)
ENV PORT=8080
EXPOSE 8080

# Set Go runtime environment variables for memory optimization
ENV GOGC=50
ENV GOMEMLIMIT=450MiB

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=40s --retries=3 \
  CMD wget --quiet --tries=1 --spider http://localhost:${PORT}/health || exit 1

# Run
CMD ["./server"]
