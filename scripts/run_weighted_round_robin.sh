#!/bin/bash

# Create necessary directories
mkdir -p conf/docker

# Make sure we have the latest configuration files
cp -f conf/docker/weighted_round_robin.conf conf/loadbalancer.conf

echo "Starting load balancer with Weighted Round Robin algorithm..."
docker-compose down
docker-compose up --build

# Additional commands for testing
# for i in {1..10}; do curl -s http://localhost:8080 | jq .id; done
# This should show server1 more frequently than server2, and server2 more frequently than server3 