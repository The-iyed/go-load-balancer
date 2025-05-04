#!/bin/bash

# Make scripts executable
chmod +x scripts/*.sh

# Function to run a specific load balancing mode and perform testing
run_mode() {
  local script=$1
  local name=$2
  local duration=${3:-60}  # Default duration: 60 seconds

  echo "==============================================="
  echo "Running $name mode..."
  echo "==============================================="
  
  # Start the load balancer in the background
  bash $script &
  LB_PID=$!
  
  # Wait for load balancer to start
  echo "Waiting for load balancer to start..."
  sleep 15
  
  # Test if load balancer is running
  if curl -s http://localhost:8080 > /dev/null; then
    echo "Load balancer is running. Starting tests..."
    
    # Run a quick load test
    bash scripts/load_test.sh 50 5
    
    # Wait for specified duration
    echo "Running for $duration seconds..."
    sleep $duration
    
    # Get final stats
    echo "Final backend statistics:"
    curl -s http://localhost:8081/api/stats | jq .
  else
    echo "Load balancer failed to start!"
  fi
  
  # Stop the load balancer
  echo "Stopping load balancer..."
  kill $LB_PID 2>/dev/null
  docker-compose down
  
  # Wait before starting the next mode
  echo "Waiting before starting next mode..."
  sleep 5
}

# Run each load balancing mode
run_mode "scripts/run_round_robin.sh" "Round Robin" 30
run_mode "scripts/run_weighted_round_robin.sh" "Weighted Round Robin" 30
run_mode "scripts/run_least_connections.sh" "Least Connections" 30
run_mode "scripts/run_cookie_persistence.sh" "Cookie Persistence" 30
run_mode "scripts/run_ip_hash.sh" "IP Hash Persistence" 30
run_mode "scripts/run_path_routing.sh" "Path-based Routing" 30

echo "All load balancing modes have been tested!" 