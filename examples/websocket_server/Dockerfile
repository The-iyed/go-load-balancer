FROM golang:1.18-alpine as builder

WORKDIR /app

# Copy the Go module files
COPY go.mod go.sum ./

# Download the Go module dependencies
RUN go mod download

# Copy the server code
COPY ./examples/websocket_server/main.go ./main.go

# Build the server
RUN CGO_ENABLED=0 GOOS=linux go build -o websocket-server .

# Final stage
FROM alpine:latest

WORKDIR /app

# Copy the binary from the builder stage
COPY --from=builder /app/websocket-server .

# Set environment variables with defaults
ENV ID=server1
ENV ADDR=:8001

# Expose the default port
EXPOSE 8001

# Run the server
CMD ["sh", "-c", "./websocket-server -id=${ID} -addr=${ADDR}"] 