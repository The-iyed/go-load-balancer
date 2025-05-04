#!/bin/bash

# Create necessary directories
mkdir -p conf/docker

# Make sure we have the latest configuration files
cp -f conf/docker/least_connections.conf conf/loadbalancer.conf

echo "Starting load balancer with Least Connections algorithm..."
docker-compose down
docker-compose up --build

# Additional commands for testing
# To simulate long-running connections:
# for i in {1..5}; do curl -s "http://localhost:8080/delay?seconds=10" & done
# This will send requests to the servers with the fewest active connections 