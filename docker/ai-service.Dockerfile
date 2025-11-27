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
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o /bin/ai-service ./cmd/ai-service

# ============================================================================
# Stage 2: Final Image
# ============================================================================
FROM alpine:3.20

# Install ca-certificates for HTTPS (needed for LLM API calls)
RUN apk --no-cache add ca-certificates tzdata

# Create non-root user
RUN addgroup -g 1001 -S appgroup && \
    adduser -u 1001 -S appuser -G appgroup

WORKDIR /app

# Copy binary from builder stage
COPY --from=builder /bin/ai-service /app/ai-service

# Change ownership to appuser
RUN chown -R appuser:appgroup /app

# Switch to non-root user
USER appuser

# AI service is a Kafka consumer, no port exposed
# No health check needed as it's a message consumer

ENTRYPOINT ["/app/ai-service"]


