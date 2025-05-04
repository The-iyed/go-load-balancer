#!/bin/bash

# Create necessary directories
mkdir -p conf/docker

# Make sure we have the latest configuration files
cp -f conf/docker/cookie_persistence.conf conf/loadbalancer.conf

echo "Starting load balancer with Cookie-based Session Persistence..."
docker-compose down
docker-compose up --build

# Additional commands for testing
# First request will get a cookie:
# curl -v http://localhost:8080
# Save the cookie and use it in subsequent requests:
# curl -v -b "lb_session_id=VALUE_FROM_PREVIOUS_RESPONSE" http://localhost:8080
# The requests should always go to the same backend server 