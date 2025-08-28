# Multi-stage build for Framework LED Matrix Daemon
FROM golang:1.24.5-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Set working directory
WORKDIR /src

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download && go mod verify

# Copy source code
COPY . .

# Build the application
ARG VERSION=unknown
ARG BUILD_TIME=unknown
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-w -s -X main.version=${VERSION} -X main.buildTime=${BUILD_TIME}" \
    -trimpath \
    -o /bin/framework-led-daemon \
    ./cmd/daemon

# Final stage
FROM scratch

# Copy CA certificates and timezone data
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

# Copy binary
COPY --from=builder /bin/framework-led-daemon /bin/framework-led-daemon

# Copy default configuration to user-accessible directory
COPY --chown=65534:65534 configs/config.yaml /app/config.yaml

# Create user
USER 65534:65534

# Expose health check port (if implemented)
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD ["/bin/framework-led-daemon", "status"]

# Set entrypoint
ENTRYPOINT ["/bin/framework-led-daemon"]
CMD ["-config", "/app/config.yaml", "run"]