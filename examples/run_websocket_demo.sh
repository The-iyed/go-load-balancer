#!/bin/bash

# Set the directory to the script's location
cd "$(dirname "$0")"

# Build the load balancer
echo "Building the load balancer..."
cd ..
go build -o load-balancer ./cmd/server/main.go
cd examples

# Build the websocket server
echo "Building the WebSocket server..."
go build -o websocket_server ./websocket_server/main.go

# Start WebSocket servers
echo "Starting WebSocket servers..."
./websocket_server -addr=:8001 -id=server1 > server1.log 2>&1 &
SERVER1_PID=$!
./websocket_server -addr=:8002 -id=server2 > server2.log 2>&1 &
SERVER2_PID=$!
./websocket_server -addr=:8003 -id=server3 > server3.log 2>&1 &
SERVER3_PID=$!

echo "WebSocket servers started on ports 8001, 8002, and 8003"

# Start the load balancer
echo "Starting the load balancer on port 8080..."
cd ..
./load-balancer --config=examples/websocket_loadbalancer.conf > load_balancer.log 2>&1 &
LB_PID=$!

echo "Load balancer started with PID: $LB_PID"
echo "Web client available at: file://$(pwd)/examples/websocket_client/index.html"
echo ""
echo "To test the WebSocket functionality:"
echo "1. Open the file://$(pwd)/examples/websocket_client/index.html in your browser"
echo "2. Connect to ws://localhost:8080/ws"
echo "3. Send messages and see them echoed back through the load balancer"
echo ""
echo "Press Ctrl+C to stop all servers"

# Trap for cleanup
function cleanup {
    echo "Stopping servers..."
    kill -9 $SERVER1_PID $SERVER2_PID $SERVER3_PID $LB_PID 2>/dev/null
    exit 0
}

trap cleanup INT

# Wait for user to press Ctrl+C
while true; do
    sleep 1
done 