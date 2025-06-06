# Multi-stage build for efficient container size
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git

# Set working directory
WORKDIR /app

# Copy go mod file
COPY go.mod ./

# Copy go.sum if it exists (conditional copy)
COPY go.su[m] ./

# Download dependencies (if any)
RUN go mod download

# Copy source code
COPY . .

# Build both applications
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o mcp-memory-server cmd/server/main.go
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o mcp-memory-reporter cmd/reporting/main.go

# Production stage
FROM alpine:latest

# Install ca-certificates for HTTPS requests
RUN apk --no-cache add ca-certificates tzdata

# Create non-root user
RUN addgroup -g 1000 mcpuser && \
    adduser -D -s /bin/sh -u 1000 -G mcpuser mcpuser

# Set working directory
WORKDIR /app

# Copy binaries from builder
COPY --from=builder /app/mcp-memory-server .
COPY --from=builder /app/mcp-memory-reporter .

# Create data directory
RUN mkdir -p /app/.mcp-memory && \
    chown -R mcpuser:mcpuser /app

# Switch to non-root user
USER mcpuser

# Default command (can be overridden in docker-compose)
CMD ["./mcp-memory-server"]