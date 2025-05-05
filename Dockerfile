# Build stage
FROM golang:1.22-alpine AS builder

# Install required build tools
RUN apk add --no-cache gcc musl-dev

# Set working directory
WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /app/build/server cmd/server/main.go

# Final stage
FROM alpine:3.19

# Install dependencies for production
RUN apk add --no-cache ca-certificates tzdata

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/build/server .

# Copy migrations
COPY migrations/ ./migrations/

# Copy docs
COPY doc/ ./doc/


# Create non-root user
RUN adduser -D -g '' appuser && \
    chown -R appuser:appuser /app

# Switch to non-root user
USER appuser

# Expose ports
EXPOSE 8080 50051

# Set entrypoint
ENTRYPOINT ["./server"]