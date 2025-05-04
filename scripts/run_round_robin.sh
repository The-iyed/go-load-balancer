#!/bin/bash

# Create necessary directories
mkdir -p conf/docker

# Make sure we have the latest configuration files
cp -f conf/docker/round_robin.conf conf/loadbalancer.conf

echo "Starting load balancer with Round Robin algorithm..."
docker-compose down
docker-compose up --build

# Additional commands for testing
# curl -v http://localhost:8080 # Make a request to the load balancer
# curl -v http://localhost:8080 # Make another request - should go to a different backend
# curl -v http://localhost:8080 # Make a third request - should complete the round robin 