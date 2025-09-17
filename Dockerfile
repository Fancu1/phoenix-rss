# Stage 1: Frontend Builder
FROM node:20-alpine AS frontend-builder
WORKDIR /app/web
COPY web/package*.json ./
RUN npm install
COPY web/ .
RUN npm run build

# Stage 2: Backend Builder
FROM golang:1.23-alpine AS backend-builder

# Install git (might be needed for some Go modules)
RUN apk add --no-cache git

# Set working directory
WORKDIR /app

# Copy go mod files first for better caching
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Copy built frontend assets from the frontend-builder stage
COPY --from=frontend-builder /app/web/build ./web/build

# Copy frontend build to the location expected by embed
RUN rm -rf cmd/api-service/dist && mkdir -p cmd/api-service/dist && cp -a web/build/. cmd/api-service/dist/

# Build all services
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o bin/scheduler-service ./cmd/scheduler-service
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o bin/ai-service ./cmd/ai-service
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o bin/api-service ./cmd/api-service
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o bin/feed-service ./cmd/feed-service
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o bin/user-service ./cmd/user-service

# Scheduler Service
FROM alpine:latest AS scheduler-service

# Install ca-certificates for HTTPS requests
RUN apk --no-cache add ca-certificates tzdata

# Create non-root user
RUN addgroup -g 1001 -S appgroup && \
    adduser -u 1001 -S appuser -G appgroup

WORKDIR /app

# Copy binary from builder stage
COPY --from=backend-builder /app/bin/scheduler-service /app/scheduler-service

# Copy config files
COPY --from=backend-builder /app/configs /app/configs

# Change ownership to appuser
RUN chown -R appuser:appgroup /app

# Switch to non-root user
USER appuser

# Expose port (adjust if needed)
EXPOSE 50054

# Run the binary
ENTRYPOINT ["/app/scheduler-service"]

# AI Service
FROM alpine:latest AS ai-service

# Install ca-certificates for HTTPS requests
RUN apk --no-cache add ca-certificates tzdata

# Create non-root user
RUN addgroup -g 1001 -S appgroup && \
    adduser -u 1001 -S appuser -G appgroup

WORKDIR /app

# Copy binary from builder stage
COPY --from=backend-builder /app/bin/ai-service /app/ai-service

# Copy config files
COPY --from=backend-builder /app/configs /app/configs

# Change ownership to appuser
RUN chown -R appuser:appgroup /app

# Switch to non-root user
USER appuser

# AI service doesn't expose HTTP ports (Kafka consumer only)
# But we can expose a health check port if needed
# EXPOSE 8080

# Run the binary
ENTRYPOINT ["/app/ai-service"]

# API Service (for future use)
FROM alpine:latest AS api-service

RUN apk --no-cache add ca-certificates tzdata

RUN addgroup -g 1001 -S appgroup && \
    adduser -u 1001 -S appuser -G appgroup

WORKDIR /app

COPY --from=backend-builder /app/bin/api-service /app/api-service
COPY --from=backend-builder /app/configs /app/configs

RUN chown -R appuser:appgroup /app

USER appuser

EXPOSE 8080

ENTRYPOINT ["/app/api-service"]

# Feed Service (for future use)
FROM alpine:latest AS feed-service

RUN apk --no-cache add ca-certificates tzdata

RUN addgroup -g 1001 -S appgroup && \
    adduser -u 1001 -S appuser -G appgroup

WORKDIR /app

COPY --from=backend-builder /app/bin/feed-service /app/feed-service
COPY --from=backend-builder /app/configs /app/configs

RUN chown -R appuser:appgroup /app

USER appuser

EXPOSE 50053

ENTRYPOINT ["/app/feed-service"]

# User Service (for future use)
FROM alpine:latest AS user-service

RUN apk --no-cache add ca-certificates tzdata

RUN addgroup -g 1001 -S appgroup && \
    adduser -u 1001 -S appuser -G appgroup

WORKDIR /app

COPY --from=backend-builder /app/bin/user-service /app/user-service
COPY --from=backend-builder /app/configs /app/configs

RUN chown -R appuser:appgroup /app

USER appuser

EXPOSE 50051

ENTRYPOINT ["/app/user-service"]
