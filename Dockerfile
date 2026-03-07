# Build stage
FROM golang:1.23-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make gcc musl-dev

WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags="-w -s" -o gopherpaw ./cmd/gopherpaw/

# Final stage
FROM alpine:latest

# Install runtime dependencies
RUN apk --no-cache add ca-certificates tzdata python3 py3-pip nodejs npm bash

# Create non-root user
RUN addgroup -g 1000 gopherpaw && \
    adduser -D -u 1000 -G gopherpaw gopherpaw

WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/gopherpaw /usr/local/bin/gopherpaw

# Copy default config
COPY configs/config.yaml.example /app/configs/config.yaml.example

# Create necessary directories
RUN mkdir -p /app/data /app/logs /app/media && \
    chown -R gopherpaw:gopherpaw /app

# Switch to non-root user
USER gopherpaw

# Set environment variables
ENV GOPHERPAW_WORKING_DIR=/app/data \
    GOPHERPAW_LOG_LEVEL=info

# Expose default port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Entry point
ENTRYPOINT ["gopherpaw"]
CMD ["--help"]
