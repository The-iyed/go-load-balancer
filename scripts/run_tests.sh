#!/bin/bash

# Run all tests for the Go Load Balancer

set -e  # Exit on any error

# Validate environment
if ! command -v go &> /dev/null; then
    echo "Error: Go is not installed or not in the PATH"
    exit 1
fi

# Determine the project root directory
PROJECT_ROOT=$(dirname "$(dirname "$(readlink -f "$0")")")
cd "$PROJECT_ROOT"

# Print Go version
go version

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}======================================${NC}"
echo -e "${BLUE}   Go Load Balancer Test Suite       ${NC}"
echo -e "${BLUE}======================================${NC}"

# Function to run a test and report results
run_test() {
    local test_type=$1
    local test_path=$2
    local test_flags=$3
    
    echo -e "\n${YELLOW}Running $test_type tests...${NC}"
    
    if go test $test_flags $test_path; then
        echo -e "${GREEN}✓ $test_type tests passed${NC}"
        return 0
    else
        echo -e "${RED}✗ $test_type tests failed${NC}"
        return 1
    fi
}

# Initialize error counter
ERRORS=0

# Run unit tests
if ! run_test "Unit" "./internal/testing/unit/..." "-v"; then
    ERRORS=$((ERRORS+1))
fi

# Run integration tests (skip in short mode)
if [[ "$1" != "--short" ]]; then
    if ! run_test "Integration" "./internal/testing/integration/..." "-v"; then
        ERRORS=$((ERRORS+1))
    fi
else
    echo -e "\n${YELLOW}Skipping integration tests in short mode${NC}"
fi

# Run performance benchmarks
if [[ "$1" != "--short" ]]; then
    echo -e "\n${YELLOW}Running performance benchmarks...${NC}"
    go test -bench=. -benchmem ./internal/testing/performance/...
else
    echo -e "\n${YELLOW}Skipping performance tests in short mode${NC}"
fi

# Run code coverage
echo -e "\n${YELLOW}Generating code coverage report...${NC}"
go test -coverprofile=coverage.out ./internal/balancer/...
go tool cover -func=coverage.out

# Generate HTML coverage report if not in short mode
if [[ "$1" != "--short" ]]; then
    echo -e "\n${YELLOW}Generating HTML coverage report...${NC}"
    go tool cover -html=coverage.out -o coverage.html
    echo -e "${GREEN}Coverage report generated at:${NC} coverage.html"
fi

# Run linting
echo -e "\n${YELLOW}Running linting...${NC}"
if ! command -v golangci-lint &> /dev/null; then
    echo -e "${RED}golangci-lint not found, skipping linting${NC}"
else
    if golangci-lint run ./...; then
        echo -e "${GREEN}✓ Linting passed${NC}"
    else
        echo -e "${RED}✗ Linting failed${NC}"
        ERRORS=$((ERRORS+1))
    fi
fi

# Report final status
echo -e "\n${BLUE}======================================${NC}"
if [ $ERRORS -eq 0 ]; then
    echo -e "${GREEN}All tests passed successfully!${NC}"
    exit 0
else
    echo -e "${RED}$ERRORS test categories failed${NC}"
    exit 1
fi 