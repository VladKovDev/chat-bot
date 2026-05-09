# Build stage
FROM golang:1.24-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git

# Set working directory
WORKDIR /build

# Copy go mod files
COPY backend/go.mod backend/go.sum ./

# Download dependencies
RUN go mod download

# Copy backend source code
COPY backend/ .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o web-adapter ./cmd/app

# Runtime stage
FROM alpine:latest

# Install ca-certificates for HTTPS
RUN apk --no-cache add ca-certificates

# Create non-root user
RUN addgroup -g 1000 appuser && \
    adduser -D -u 1000 -G appuser appuser

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/web-adapter .

# Copy config files
COPY backend/configs ./configs

# Copy frontend files
COPY frontend ./frontend

# Change ownership to appuser
RUN chown -R appuser:appuser /app

# Switch to non-root user
USER appuser

# Expose WebSocket port
EXPOSE 8081

# Set default environment variables
ENV CONFIG_PATH=./configs
ENV APP_ENV=local
ENV DECISION_ENGINE_URL=http://localhost:8080

# Run the application
CMD ["./web-adapter"]
