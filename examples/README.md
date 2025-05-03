# WebSocket Load Balancing Examples

This directory contains examples for testing WebSocket support in the Go Load Balancer.

## Contents

- `websocket_client/` - HTML/JS client for testing WebSocket connections
- `websocket_server/` - Simple Go WebSocket server implementation
- `websocket_loadbalancer.conf` - Configuration file for WebSocket load balancing
- `websocket_configurations.conf` - Various example configurations for different WebSocket scenarios
- `docker-compose-websocket.yml` - Docker Compose setup for testing WebSockets
- `run_websocket_demo.sh` - Script to run the WebSocket demo locally

## Running the Docker Demo

To run the WebSocket load balancing demo using Docker:

1. Make sure Docker and Docker Compose are installed
2. From the project root, run:
   ```
   docker-compose -f examples/docker-compose-websocket.yml up --build
   ```
3. Access the WebSocket client at: http://localhost:8090
4. Connect to the load balancer WebSocket endpoint: ws://localhost:8080

## Running Locally

To run the WebSocket demo on your local machine:

1. Make the demo script executable:
   ```
   chmod +x examples/run_websocket_demo.sh
   ```
2. Run the script:
   ```
   ./examples/run_websocket_demo.sh
   ```
3. Open the WebSocket client in your browser at the URL shown in the terminal
4. Connect to WebSocket endpoint: ws://localhost:8080

## Testing Session Persistence

To verify that session persistence is working:

1. Connect to the WebSocket endpoint multiple times
2. Observe that messages from the same client are always handled by the same backend server
3. If using IP-hash persistence, try connecting from different IP addresses or browsers
4. If using cookie persistence, try connecting in normal and private browsing sessions

## Troubleshooting

- If you have issues connecting, check that all servers are running properly
- Verify the load balancer configuration matches the backend server addresses
- Check the browser console for any WebSocket connection errors
- Inspect network traffic using browser developer tools 