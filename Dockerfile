# Stage 1: Build the application
FROM golang:1.21-alpine AS builder

# Set working directory
WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git

# Copy go.mod and go.sum files first and download dependencies
COPY go.mod go.sum* ./
RUN go mod download

# Copy the source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o loadbalancer cmd/server/main.go

# Stage 2: Build the final image
FROM alpine:latest

WORKDIR /app

# Install runtime dependencies
RUN apk --no-cache add ca-certificates tzdata

# Copy the binary from the builder stage
COPY --from=builder /app/loadbalancer .

# Copy configuration files
COPY conf/ /app/conf/

# Create a non-root user and switch to it
RUN addgroup -S appgroup && adduser -S appuser -G appgroup
RUN chown -R appuser:appgroup /app
USER appuser

# Expose ports
EXPOSE 8080 8081

# Set the entry point
ENTRYPOINT ["/app/loadbalancer"]
CMD ["--config", "/app/conf/docker.conf"] 