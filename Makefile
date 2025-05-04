.PHONY: build run test clean docker-build docker-run all help

BINARY_NAME=loadbalancer
BUILD_DIR=bin
MAIN_FILE=cmd/server/main.go
CONFIG_FILE=conf/loadbalancer.conf
DOCKER_IMAGE_NAME=go-load-balancer
VERSION=0.1.0

help: ## Display this help message
	@echo "Usage:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

all: clean build ## Clean and build the application

build: ## Build the load balancer binary
	go build -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_FILE)

run: ## Run the load balancer from source
	go run $(MAIN_FILE) --config $(CONFIG_FILE)

test: ## Run tests
	go test -v ./...

test-race: ## Run tests with race detector
	go test -race -v ./...

test-coverage: ## Run tests with coverage report
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

lint: ## Run linters
	golangci-lint run

clean: ## Remove build artifacts
	rm -rf $(BUILD_DIR)
	rm -f coverage.out coverage.html

docker-build: ## Build Docker image
	docker build -t $(DOCKER_IMAGE_NAME):$(VERSION) .

docker-run: ## Run Docker container
	docker run -p 8080:8080 -p 8081:8081 $(DOCKER_IMAGE_NAME):$(VERSION)

docker-compose-up: ## Start all services with Docker Compose
	docker-compose up -d

docker-compose-down: ## Stop all services with Docker Compose
	docker-compose down

go-mod-tidy: ## Tidy and verify Go dependencies
	go mod tidy
	go mod verify 