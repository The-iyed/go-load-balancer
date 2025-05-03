# Go Load Balancer

<p align="center">
  <img src="docs/images/logo.png" alt="Go Load Balancer Logo" width="300">
</p>

[![Go Report Card](https://goreportcard.com/badge/github.com/The-iyed/go-load-balancer)](https://goreportcard.com/report/github.com/The-iyed/go-load-balancer)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

A high-performance HTTP load balancer written in Go with support for multiple load balancing algorithms and Docker integration.


## Features

- **Multiple Load Balancing Algorithms**
  - Weighted Round Robin
  - Least Connections
- **Session Persistence Methods**
  - Cookie-based persistence
  - IP hash persistence
  - Consistent hashing
- **Health Checking**
  - Automatic detection of failed backends
  - Self-healing with automatic recovery
- **Production Ready**
  - Docker and Docker Compose support
  - Customizable via command-line flags
- **Real-time Metrics**
  - Track active connections and server status

## Quick Start

### Using Docker Compose

The easiest way to try the load balancer is with Docker Compose:

```bash
git clone https://github.com/your-username/go-load-balancer.git
cd go-load-balancer
docker-compose up
```

This starts:
- The load balancer on port 8080
- Three backend web servers with different weights

### Building From Source

```bash
git clone https://github.com/your-username/go-load-balancer.git
cd go-load-balancer
go build -o load-balancer ./cmd/server
./load-balancer --config=conf/loadbalancer.conf
```

## Command-Line Options

```
Usage of ./load-balancer:
  -algorithm string
        override load balancing algorithm: round-robin, weighted-round-robin, least-connections
  -config string
        accessing configuration file (default "conf/loadbalancer.conf")
  -persistence string
        override session persistence method: none, cookie, ip_hash, consistent_hash
```

## Configuration File Format

```
upstream backend {
    method weighted_round_robin;
    persistence cookie;
    server http://backend1:80 weight=3;
    server http://backend2:80 weight=2;
    server http://backend3:80 weight=1;
}
```

See [Configuration Guide](docs/configuration.md) for more details.

## Supported Algorithms

### Weighted Round Robin

Distributes requests to backend servers proportionally to their weights.

### Least Connections

Routes each request to the backend server with the fewest active connections, using weights to break ties.

## Supported Persistence Methods

### None (Default)

No session persistence is applied. Each request is treated independently according to the selected load balancing algorithm.

### Cookie

Cookie-based persistence tracks which backend server each client should use via HTTP cookies:

1. When a client makes their first request, the load balancer selects a backend using the configured algorithm
2. The load balancer sets a cookie (GOLB_SESSION) containing an encoded reference to the selected backend
3. On subsequent requests, the client sends this cookie and the load balancer routes to the same backend
4. If the original backend is down, the load balancer selects a new one and updates the cookie

### IP Hash

IP-based persistence uses the client's IP address to determine which backend to use:

1. The load balancer extracts the client's IP address from the request
2. The IP is hashed to consistently map to the same backend server
3. All requests from the same IP are sent to the same backend
4. This method works even when cookies are not supported

### Consistent Hash

Consistent hashing uses the request path to distribute requests:

1. Each backend server is assigned multiple points on a hash ring
2. The request path is hashed to determine its position on the ring
3. The request is routed to the nearest backend server on the ring
4. When servers are added or removed, only a fraction of requests are redistributed

## Architecture

The load balancer consists of the following core components:

1. **HTTP Proxy**: Routes requests to backend servers
2. **Backend Pool**: Manages the set of available backend servers
3. **Algorithm Strategies**: Implements different load balancing algorithms
4. **Session Persistence**: Maintains client sessions with the same backend
5. **Health Checker**: Periodically checks backends and removes unhealthy ones
6. **Configuration Parser**: Reads and validates the config file

## How Session Persistence Works

### Cookie-Based Persistence

This method adds a cookie to track which backend server a client should use:

```
+---------+  1. Request   +---------------+  2. Select    +----------+
| Client  |-------------->| Load Balancer |-------------->| Backend1 |
+---------+               +---------------+               +----------+
     |                           |
     |                           | 3. Set cookie: GOLB_SESSION=0:hash
     |                           |
     | 4. Next request with cookie
     | GOLB_SESSION=0:hash       |
     v                           v
+---------+  5. Use cookie +---------------+  6. Route to  +----------+
| Client  |--------------->| Load Balancer |--------------->| Backend1 |
+---------+               +---------------+                +----------+
```

The cookie contains:
- Backend server index
- MD5 hash for verification to prevent tampering
- Default 24-hour expiration (configurable)

### IP Hash Persistence

This method uses the client's IP address to determine backend server assignment:

```
+---------+  1. Request from   +---------------+
| Client  |-------------------->| Load Balancer |
| IP: x.x.x.x              +---------------+
+---------+                    |
                              | 2. Hash IP: hash(x.x.x.x) % backends
                              v
                         +----------+
                         | Backend2 |
                         +----------+
```

All subsequent requests from the same IP address will route to the same backend server.

### Consistent Hash Persistence

This method uses consistent hashing of the request path:

```
           Backend1
           /      \
          /        \
         /          \
 -------+------------+------- Hash Ring
        |            |
        |            |
        +            +
    Backend3     Backend2

Requests are routed to the nearest server on the hash ring.
```

Key features:
- Each backend has multiple virtual nodes on the ring based on its weight
- Only a portion of requests are reassigned when servers change
- Provides good distribution while maintaining consistency

## Performance

Benchmarks show the load balancer can handle:

- 10,000+ requests per second on modest hardware
- Low latency overhead (typically < 1ms)
- Graceful handling of backend server failures

## Documentation

For detailed documentation, please see the following resources:

- [User Guide](docs/README.md)
- [Load Balancing Algorithms](docs/algorithms.md)
- [Configuration Guide](docs/configuration.md)
- [Example Configurations](docs/examples)

## Project Structure

```
go-load-balancer/
├── cmd/                  # Application entry points
│   └── server/           # Load balancer server
├── conf/                 # Configuration files
├── internal/             # Internal packages
│   ├── balancer/         # Load balancing implementation
│   └── logger/           # Logging utilities
├── docs/                 # Documentation
├── backends/             # Example backend servers
├── Dockerfile            # Container definition
└── docker-compose.yml    # Multi-container setup
```

## Development

### Prerequisites

- Go 1.16+
- Docker (optional, for containerized development)

### Building from Source

```bash
go build -o load-balancer ./cmd/server/main.go
```

### Running Tests

```bash
go test ./...
```

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Acknowledgements

- The Go Team for the excellent standard library
- The Nginx project for inspiration on the configuration format 