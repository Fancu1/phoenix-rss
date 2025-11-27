# ============================================================================
# Stage 1: Go Builder
# ============================================================================
FROM golang:1.23-alpine AS builder

RUN apk add --no-cache git

WORKDIR /app

# Copy go mod files first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o /bin/feed-service ./cmd/feed-service

# Download grpc-health-probe for health checking (with retries)
RUN GRPC_HEALTH_PROBE_VERSION=v0.4.28 && \
    for i in 1 2 3 4 5; do \
      wget --timeout=30 -qO/bin/grpc_health_probe \
        https://github.com/grpc-ecosystem/grpc-health-probe/releases/download/${GRPC_HEALTH_PROBE_VERSION}/grpc_health_probe-linux-amd64 && break || sleep 5; \
    done && \
    chmod +x /bin/grpc_health_probe

# ============================================================================
# Stage 2: Final Image
# ============================================================================
FROM alpine:3.20

# Install ca-certificates for HTTPS
RUN apk --no-cache add ca-certificates tzdata

# Create non-root user
RUN addgroup -g 1001 -S appgroup && \
    adduser -u 1001 -S appuser -G appgroup

WORKDIR /app

# Copy binary and health probe from builder stage
COPY --from=builder /bin/feed-service /app/feed-service
COPY --from=builder /bin/grpc_health_probe /bin/grpc_health_probe

# Change ownership to appuser
RUN chown -R appuser:appgroup /app

# Switch to non-root user
USER appuser

EXPOSE 50053

# Health check using grpc-health-probe
HEALTHCHECK --interval=10s --timeout=5s --start-period=10s --retries=3 \
    CMD /bin/grpc_health_probe -addr=:50053 || exit 1

ENTRYPOINT ["/app/feed-service"]


