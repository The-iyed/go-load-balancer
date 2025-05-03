FROM golang:1.18-alpine as builder

WORKDIR /app

# Copy the Go module files
COPY go.mod go.sum ./

# Download the Go module dependencies
RUN go mod download

# Copy the source code
COPY . .

# Build the load balancer
RUN CGO_ENABLED=0 GOOS=linux go build -o load-balancer ./cmd/server/main.go

# Final stage
FROM alpine:latest

WORKDIR /app

# Copy the binary from the builder stage
COPY --from=builder /app/load-balancer .

# Create a directory for config
RUN mkdir -p /app/conf

# Expose the default port
EXPOSE 8080

# Run the load balancer with the config file
CMD ["./load-balancer", "-conf", "/app/conf/loadbalancer.conf"] 