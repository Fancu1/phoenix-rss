# ============================================================================
# Stage 1: Frontend Builder
# ============================================================================
FROM node:20-alpine AS frontend-builder

WORKDIR /app/web
COPY web/package*.json ./
RUN npm ci --silent
COPY web/ .
RUN npm run build

# ============================================================================
# Stage 2: Go Builder
# ============================================================================
FROM golang:1.23-alpine AS builder

RUN apk add --no-cache git

WORKDIR /app

# Copy go mod files first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Copy built frontend assets from the frontend-builder stage
COPY --from=frontend-builder /app/web/build ./cmd/api-service/dist

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o /bin/api-service ./cmd/api-service

# ============================================================================
# Stage 3: Final Image
# ============================================================================
FROM golang:1.23-alpine

# Install ca-certificates for HTTPS and wget for health checks
RUN apk --no-cache add ca-certificates tzdata wget

# Create non-root user
RUN addgroup -g 1001 -S appgroup && \
    adduser -u 1001 -S appuser -G appgroup

WORKDIR /app

# Copy binary from builder stage
COPY --from=builder /bin/api-service /app/api-service

# Change ownership to appuser
RUN chown -R appuser:appgroup /app

# Switch to non-root user
USER appuser

EXPOSE 8080

# Health check using the existing /api/v1/health endpoint
HEALTHCHECK --interval=10s --timeout=5s --start-period=10s --retries=3 \
    CMD wget -qO- http://localhost:8080/api/v1/health || exit 1

ENTRYPOINT ["/app/api-service"]


