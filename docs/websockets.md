# WebSocket Support in Go Load Balancer

This document covers how to configure and use WebSocket support in the Go Load Balancer.

## Overview

The Go Load Balancer provides full WebSocket support, allowing you to distribute WebSocket connections across multiple backend servers while maintaining session persistence. WebSocket connections can be load balanced using the same methods available for HTTP traffic, with additional configuration options specific to WebSocket needs.

## Configuration

### Basic WebSocket Configuration

To configure the load balancer for WebSocket traffic, use the following configuration format:

```conf
upstream backend {
    method weighted_round_robin;  # Choose your preferred balancing method
    persistence ip_hash;          # Session persistence is important for WebSockets
    
    server http://backend1:8001 weight=1;
    server http://backend2:8001 weight=1;
    server http://backend3:8001 weight=1;
}

server {
    listen 8080;
    server_name localhost;
    
    location / {
        proxy_pass backend;
        
        # Required WebSocket headers
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
}
```

### WebSocket Headers

For WebSocket connections to work properly, the load balancer automatically adds the following headers when it detects a WebSocket upgrade request:

- `Upgrade: websocket`
- `Connection: upgrade`
- `Sec-WebSocket-Version: 13`
- `Sec-WebSocket-Key: [base64-encoded key]`

These headers facilitate the WebSocket protocol handshake between the client and backend servers.

## Session Persistence for WebSockets

WebSocket connections benefit significantly from session persistence. The load balancer supports several persistence methods that are particularly useful for WebSocket applications:

### IP Hash Persistence

```conf
persistence ip_hash;
```

IP-based persistence ensures that connections from the same client IP address are consistently routed to the same backend server. This is ideal for WebSockets as it maintains connection state across multiple WebSocket connections from the same client.

### Cookie-based Persistence

```conf
persistence cookie;
```

Cookie-based persistence uses HTTP cookies to track which backend server should handle each client. This works well for browser-based WebSocket clients and provides stickiness even when a client's IP address changes.

### Consistent Hash Persistence

```conf
persistence consistent_hash;
```

Consistent hashing uses information from the request path or headers to determine the backend server. This is useful for scaling WebSocket services with path-based routing.

## Load Balancing Methods for WebSockets

While all load balancing methods work with WebSockets, some are more suitable depending on your WebSocket application's characteristics:

### Least Connections

```conf
method least_connections;
```

The least connections method is ideal for WebSocket applications where connections remain open for long periods. It ensures that new connections are routed to the backend server with the fewest active connections, helping to distribute load evenly.

### Weighted Round Robin

```conf
method weighted_round_robin;
```

Weighted round robin is suitable for WebSocket applications where all connections have similar resource requirements. It distributes connections among backend servers according to their assigned weights.

## WebSocket Timeouts and Keepalive

WebSocket connections typically remain open longer than regular HTTP connections. You may need to adjust timeouts to prevent the load balancer from closing idle WebSocket connections:

```conf
server {
    # ... other configuration ...
    
    # Extend timeout for WebSocket connections
    proxy_read_timeout 3600s;  # 1 hour
    proxy_send_timeout 3600s;  # 1 hour
    
    # Enable WebSocket ping/pong keepalive
    proxy_websocket_keepalive on;
}
```

## Handling WebSocket Disconnections

When a backend server disconnects or becomes unavailable, the load balancer will close the corresponding WebSocket connections. Clients should implement reconnection logic with exponential backoff to handle these situations gracefully.

## Scaling WebSocket Applications

For high-volume WebSocket applications:

1. Use multiple backend servers and distribute them across different physical machines
2. Consider using the least connections balancing method to evenly distribute the connection load
3. Implement proper session persistence to maintain connection state
4. Consider containerization with Docker for easy scaling of backend WebSocket servers

## Example Configurations

### Chat Application

```conf
upstream chat_backend {
    method least_connections;
    persistence ip_hash;
    
    server http://chat1:8001 weight=1;
    server http://chat2:8001 weight=1;
    server http://chat3:8001 weight=1;
}

server {
    listen 8080;
    server_name chat.example.com;
    
    location /ws {
        proxy_pass chat_backend;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
    }
}
```

### Game Server

```conf
upstream game_backend {
    method weighted_round_robin;
    persistence cookie;
    
    server http://game1:8001 weight=3;  # High-capacity server
    server http://game2:8001 weight=1;  # Low-capacity server
}

server {
    listen 8080;
    server_name game.example.com;
    
    location /game/ws {
        proxy_pass game_backend;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
    }
}
```

## Testing WebSocket Support

To test WebSocket support, you can use the provided example in the `examples/` directory:

```bash
# Run the WebSocket demo
./examples/run_websocket_demo.sh

# Or use Docker Compose
docker-compose -f examples/docker-compose-websocket.yml up --build
```

The example includes a simple WebSocket client that connects through the load balancer to multiple backend WebSocket servers, demonstrating both load balancing and session persistence.

## Troubleshooting

### Common Issues

1. **Connection Refused**: Ensure backend servers are running and accessible.
2. **WebSocket Handshake Failed**: Check that the required WebSocket headers are being forwarded.
3. **Connections Not Persistent**: Verify that session persistence is configured correctly.
4. **Timeouts**: Adjust proxy timeout settings for long-lived WebSocket connections.

### Debugging

To debug WebSocket issues, enable verbose logging in the load balancer:

```bash
./load-balancer -conf loadbalancer.conf -v
```

This will show detailed information about WebSocket upgrade requests and connection handling. 