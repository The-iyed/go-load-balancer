#!/bin/bash

# Check if the load balancer is running
if ! curl -s http://localhost:8080 > /dev/null; then
  echo "Load balancer is not running. Please start it first with one of the run_*.sh scripts."
  exit 1
fi

# Number of requests to send
REQUESTS=${1:-100}
CONCURRENCY=${2:-10}

echo "Running load test with $REQUESTS requests and $CONCURRENCY concurrent connections..."

# Install hey if not already present
if ! command -v hey &> /dev/null; then
  echo "Installing hey load testing tool..."
  go install github.com/rakyll/hey@latest
fi

# Run the load test
hey -n $REQUESTS -c $CONCURRENCY http://localhost:8080

# Check backend statistics
echo -e "\nBackend Statistics:"
curl -s http://localhost:8081/api/stats | jq .

echo -e "\nTest complete!" 