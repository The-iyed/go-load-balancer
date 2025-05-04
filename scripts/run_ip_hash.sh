#!/bin/bash

# Create necessary directories
mkdir -p conf/docker

# Make sure we have the latest configuration files
cp -f conf/docker/ip_hash_persistence.conf conf/loadbalancer.conf

echo "Starting load balancer with IP Hash Session Persistence..."
docker-compose down
docker-compose up --build

# Additional commands for testing
# Multiple requests from the same IP should go to the same backend:
# for i in {1..5}; do curl -s http://localhost:8080 | jq .id; done 