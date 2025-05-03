# WebSocket Support

The Go Load Balancer includes comprehensive support for WebSocket connections, enabling real-time communication applications to work seamlessly with the load balancer.

## Features

- Transparent proxying of WebSocket connections
- Persistence of WebSocket connections to backend servers
- Automatic protocol upgrade handling
- Connection health monitoring with ping/pong mechanism
- Support for both ws:// and wss:// (secure WebSocket) protocols
- Graceful error handling and backend failover

## How WebSocket Load Balancing Works

When a client attempts to establish a WebSocket connection through the load balancer:

1. The load balancer detects the WebSocket upgrade request based on HTTP headers
2. A backend server is selected using the configured load balancing algorithm
3. The load balancer upgrades the client connection to WebSocket
4. The load balancer establishes a WebSocket connection to the selected backend
5. Messages are bidirectionally proxied between the client and backend
6. Periodic ping frames monitor the health of both connections
7. If a backend fails, connections are cleaned up and future requests route to healthy backends

## Configuration

WebSocket support is automatically enabled in all load balancing algorithms. No additional configuration is required to enable WebSocket support.

When using session persistence with WebSockets:

- Cookie-based persistence uses cookies in the initial HTTP handshake
- IP-based persistence keeps connections from the same client IP routed to the same backend
- Consistent hashing ensures similar WebSocket paths route to the same backend

## Scaling Considerations

The load balancer is designed to efficiently handle WebSocket connections at scale:

- Connection tracking uses minimal memory per connection
- Goroutines that handle message proxying are lightweight
- Periodic health checks prevent resource leaks from disconnected clients
- Connection idle timeout prevents holding resources indefinitely

## Example WebSocket Client

Here's a simple example of a JavaScript WebSocket client connecting through the load balancer:

```javascript
// Connect to load balancer
const socket = new WebSocket('ws://loadbalancer:8080/socket');

// Set up event handlers
socket.onopen = function(e) {
  console.log('Connection established');
  socket.send('Hello from client!');
};

socket.onmessage = function(event) {
  console.log(`Data received: ${event.data}`);
};

socket.onclose = function(event) {
  if (event.wasClean) {
    console.log(`Connection closed cleanly, code=${event.code} reason=${event.reason}`);
  } else {
    console.log('Connection died');
  }
};

socket.onerror = function(error) {
  console.log(`Error: ${error.message}`);
};
```

## Example Backend WebSocket Server (Go)

```go
package main

import (
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins in this example
	},
}

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Upgrade error:", err)
		return
	}
	defer conn.Close()

	for {
		messageType, p, err := conn.ReadMessage()
		if err != nil {
			log.Println("Read error:", err)
			return
		}
		log.Printf("Received message: %s", p)

		// Echo the message back
		if err := conn.WriteMessage(messageType, p); err != nil {
			log.Println("Write error:", err)
			return
		}
	}
}

func main() {
	http.HandleFunc("/socket", handleWebSocket)
	log.Fatal(http.ListenAndServe(":8000", nil))
}
```

## Health Checking for WebSocket Backends

The load balancer monitors WebSocket connection health through:

1. Connection establishment failures
2. WebSocket protocol errors
3. Ping/pong frame timeouts
4. Unexpected connection closures

When a backend fails health checks, it is marked as unhealthy and connections are temporarily routed to other backends until the failed backend recovers.

## Performance Considerations

- The load balancer can handle thousands of concurrent WebSocket connections
- Memory usage scales linearly with the number of active connections
- For very high connection counts (10,000+), consider horizontal scaling with multiple load balancer instances
- Configure appropriate timeouts to prevent resource exhaustion from idle connections 