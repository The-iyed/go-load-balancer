#!/bin/bash

# Create necessary directories
mkdir -p conf/docker

# Make sure we have the latest configuration files
cp -f conf/docker/path_routing.conf conf/loadbalancer.conf

echo "Starting load balancer with Path-based Routing..."
docker-compose down
docker-compose up --build

# Additional commands for testing
# API requests should go to server1 or server2:
# curl -s http://localhost:8080/api | jq .id
# curl -s http://localhost:8080/api | jq .id
#
# Static content requests should always go to server3:
# curl -s http://localhost:8080/static | jq .id
# curl -s http://localhost:8080/images | jq .id 