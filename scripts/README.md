# Docker Scripts for Go Load Balancer

This directory contains scripts to run the Go Load Balancer with different load balancing algorithms and features using Docker.

## Setup Instructions

1. Make sure Docker and Docker Compose are installed on your system.
2. All scripts have been made executable with `chmod +x scripts/*.sh`.

## Available Scripts

### Individual Load Balancing Modes

- `run_round_robin.sh`: Run load balancer with simple Round Robin algorithm
- `run_weighted_round_robin.sh`: Run load balancer with Weighted Round Robin algorithm
- `run_least_connections.sh`: Run load balancer with Least Connections algorithm
- `run_cookie_persistence.sh`: Run load balancer with Cookie-based Session Persistence
- `run_ip_hash.sh`: Run load balancer with IP Hash-based Session Persistence
- `run_path_routing.sh`: Run load balancer with Path-based Routing

### Testing Scripts

- `load_test.sh`: Run a load test against the currently running load balancer
  - Usage: `./load_test.sh [NUMBER_OF_REQUESTS] [CONCURRENT_CONNECTIONS]`
  - Default: 100 requests with 10 concurrent connections

### Automated Testing

- `run_all_modes.sh`: Run all load balancing modes in sequence with automatic testing

## Example Usage

1. Run a specific load balancing mode:
   ```bash
   ./scripts/run_round_robin.sh
   ```

2. In a separate terminal, run a load test:
   ```bash
   ./scripts/load_test.sh 1000 20
   ```

3. Monitor the load balancer UI:
   - Open a web browser and go to http://localhost:8081

4. Run all load balancing modes in sequence:
   ```bash
   ./scripts/run_all_modes.sh
   ```

## Configuration Files

All Docker-specific configuration files are stored in `conf/docker/` directory:

- `round_robin.conf`
- `weighted_round_robin.conf`
- `least_connections.conf`
- `cookie_persistence.conf`
- `ip_hash_persistence.conf`
- `path_routing.conf`

You can modify these files to adjust the load balancing behavior. 