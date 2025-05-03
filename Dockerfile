FROM golang:1.21-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o load-balancer ./cmd/server/main.go

FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/load-balancer .
COPY conf/loadbalancer.conf /app/conf/loadbalancer.conf

EXPOSE 8080

CMD ["./load-balancer", "-config", "/app/conf/loadbalancer.conf"] 