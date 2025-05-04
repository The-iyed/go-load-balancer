# Go Load Balancer

<p align="center">
  <img src="images/logo.png" alt="Go Load Balancer Logo" width="300">
</p>

A high-performance HTTP load balancer implemented in Go with support for multiple load balancing algorithms.

## Features

- Multiple load balancing algorithms:
  - Weighted Round Robin
  - Least Connections
- Session Persistence Methods:
  - Cookie-based persistence
  - IP hash persistence
  - Consistent hashing
- Path-Based Routing:
  - Route to backend pools based on URL paths
  - Regular expression pattern matching
  - Header-based routing
- WebSocket support with seamless proxying and connection management
- Nginx-style configuration syntax with algorithm selection
- Health checking with automatic backend recovery
- Docker support for easy deployment
- Command-line interface for customization

## Installation

### Prerequisites

- Go 1.16+
- Docker (optional, for containerized deployment)

### Building from Source

```bash
# Clone the repository
git clone https://github.com/The-iyed/go-load-balancer.git
cd go-load-balancer

# Build the binary
go build -o load-balancer ./cmd/server/main.go
```

### Using Docker

```bash
# Build the Docker image
docker build -t go-load-balancer .

# Run with Docker
docker run -p 8080:8080 go-load-balancer
```

### Using Docker Compose

```bash
docker-compose up --build
```

## Usage

### Running the Load Balancer

```bash
# Using default configuration
./load-balancer

# Specifying a config file
./load-balancer --config=conf/loadbalancer.conf

# Overriding the balancing algorithm from config
./load-balancer --algorithm=least-connections

# Overriding the persistence method from config
./load-balancer --persistence=cookie

# Explicitly enabling path-based routing
./load-balancer --path-routing
```

### Available Command-Line Options

| Option | Default | Description |
|--------|---------|-------------|
| `--config` | `conf/loadbalancer.conf` | Path to the configuration file |
| `--algorithm` | `` | Override the load balancing algorithm defined in config |
| `--persistence` | `` | Override the session persistence method defined in config |
| `--path-routing` | `false` | Explicitly enable path-based routing (auto-detected if routes exist) |

### Supported Algorithms

- `weighted-round-robin`: Distributes traffic based on server weights
- `round-robin`: Simple round-robin distribution (weights are ignored)
- `least-connections`: Routes to the server with the fewest active connections

### Supported Persistence Methods

- `none`: No session persistence (default)
- `cookie`: Uses cookies to track which backend served a client
- `ip_hash`: Maps client IPs to specific backend servers
- `consistent_hash`: Uses consistent hashing for even distribution with minimal redistribution

## Configuration

The load balancer is configured using a simple Nginx-inspired syntax.

### Configuration File Format

```
# Backend definitions
upstream <pool-name> {
    server <URL> weight=<WEIGHT>;
    server <URL> weight=<WEIGHT>;
    ...
}

# Load balancing algorithm for all backend pools
method <algorithm>;

# Session persistence method
persistence <method>;

# Path-based routing
route path <path-prefix> <backend-pool>
route regex <regex-pattern> <backend-pool>
route header <header-name> <value> <backend-pool>

# Default backend pool
default_backend <backend-pool>
```

Where:
- `<pool-name>` is the name of the backend pool
- `<URL>` is the URL of the backend server (e.g., `http://backend1:80`)
- `<WEIGHT>` is the weight of the server (default: 1)
- `<algorithm>` is the load balancing algorithm to use
- `<method>` is the session persistence method to use
- `<path-prefix>` is the URL path prefix to match for routing
- `<regex-pattern>` is a regular expression to match the full URL path
- `<backend-pool>` is the name of a defined upstream pool

### Available Methods

| Method Name | Description |
|-------------|-------------|
| `weighted_round_robin` | Distributes traffic based on server weights |
| `round_robin` | Simple round-robin distribution (weights are ignored) |
| `least_conn` | Routes to the server with the fewest active connections |

### Examples

See the [examples](examples/) directory for sample configuration files.

## Architecture

The load balancer follows a clean architecture with the following components:

### Core Components

- **Balancer Interface**: Defines the common interface for all load balancing algorithms
- **Backend Processes**: Represents and manages backend servers
- **Path-Based Router**: Routes requests to backend pools based on paths and patterns
- **Health Checking**: Monitors backend health and handles failure recovery
- **Request Proxying**: Proxies client requests to selected backends
- **Session Persistence**: Maintains client affinity to specific backends

### Directory Structure

```
go-load-balancer/
├── cmd/
│   └── server/           # Application entry point
├── conf/                 # Configuration files
├── docs/                 # Documentation
│   └── examples/         # Example configurations
├── internal/             # Internal packages
│   ├── balancer/         # Load balancing algorithms
│   └── logger/           # Logging utilities
├── backends/             # Example backend servers
├── Dockerfile            # Container definition
└── docker-compose.yml    # Docker Compose configuration
```

## Health Checking

The load balancer performs passive health checking:

1. When a request to a backend fails, its error count is incremented
2. After 3 consecutive failures, the backend is marked as unhealthy
3. The load balancer automatically attempts to revive the backend after 10 seconds
4. Unhealthy backends are excluded from load balancing until revived

## Load Balancing Algorithms

### Weighted Round Robin

The Weighted Round Robin algorithm distributes traffic proportionally based on server weights.

#### How It Works

1. Each server is assigned a weight value (default: 1)
2. Traffic distribution follows the ratio of weights
3. For example, with weights of 5:3:2, servers receive 50%, 30%, and 20% of traffic respectively

#### Configuration Example

```
upstream backend {
    server http://backend1:80 weight=5;
    server http://backend2:80 weight=3;
    server http://backend3:80 weight=2;
}

method weighted_round_robin;
```

#### When to Use

- When backends have different capacity/performance
- When you need predictable distribution of requests
- When requests have similar processing times

### Least Connections

The Least Connections algorithm routes traffic to the server with the fewest active connections.

#### How It Works

1. For each request, the load balancer selects the backend with the lowest number of active connections
2. If multiple backends have the same number of connections, weights are used as a tiebreaker
3. Connection counts are tracked in real-time

#### Configuration Example

```
upstream backend {
    server http://backend1:80 weight=1;
    server http://backend2:80 weight=1;
    server http://backend3:80 weight=1;
}

method least_connections;
```

#### When to Use

- When request processing times vary significantly
- When some requests take much longer than others
- When backends can become overloaded easily

## Session Persistence

Session persistence ensures that requests from the same client are routed to the same backend server.

### Cookie-Based Persistence

#### How It Works

1. On the first request, a backend is selected using the configured load balancing algorithm
2. A cookie is set in the response that identifies the selected backend
3. Subsequent requests from the same client include the cookie and are routed to the same backend
4. If the backend becomes unhealthy, a new backend is selected and the cookie is updated

#### Configuration Example

```
upstream backend {
    server http://backend1:80 weight=3;
    server http://backend2:80 weight=2;
    server http://backend3:80 weight=1;
}

method weighted_round_robin;
persistence cookie;
```

#### When to Use

- When the application relies on server-side session state
- For web applications with user logins or shopping carts
- When client state should be preserved between requests

## Path-Based Routing

Path-based routing allows you to route requests to different backend pools based on URL path, HTTP headers, or regex patterns.

### How It Works

1. The load balancer evaluates each request against defined routing rules
2. The first matching rule determines the backend pool to use
3. Each backend pool can use its own set of servers
4. If no rule matches, the default backend is used

### Configuration Example

```
# Define multiple backend pools
upstream api {
    server http://api1:8000 weight=5;
    server http://api2:8000 weight=5;
}

upstream static {
    server http://static1:8080 weight=10;
}

upstream admin {
    server http://admin:8071;
}

# Default pool
upstream backend {
    server http://default1:8000 weight=1;
    server http://default2:8000 weight=1;
}

# Define routing rules
route path /api/ api
route path /static/ static
route path /admin/ admin
route regex ^/api/v2/.* api
route header X-Internal true admin

# Set default backend pool
default_backend backend

# Global settings
method weighted_round_robin;
persistence cookie;
```

### Route Types

- **Path Routes**: Match URL path prefixes
- **Regex Routes**: Match URL paths using regular expressions
- **Header Routes**: Match HTTP header values

### When to Use

- In microservices architectures
- For content-type based routing (API vs Web)
- For multi-tenant applications
- For A/B testing or gradual rollouts
- For device-specific routing (mobile vs desktop)

For more detailed documentation on path-based routing, see [Path-Based Routing](path-based-routing.md).

## WebSocket Support

The load balancer includes built-in support for WebSocket connections.

### How It Works

1. WebSocket connections are detected based on the `Upgrade: websocket` header
2. The connection is proxied to the selected backend with the appropriate headers
3. The WebSocket connection is maintained until closed by either party

### Configuration

WebSocket support works automatically with all load balancing algorithms and persistence methods. No special configuration is needed beyond standard backend setup.

### When to Use

- For real-time web applications
- For chat applications
- For streaming data applications
- For interactive web applications

For more detailed documentation on WebSocket support, see [WebSockets](websockets.md).

## Performance Considerations

- The load balancer is designed to be lightweight and efficient
- Connection tracking is optimized for minimal overhead
- Health checking is passive to reduce network traffic
- Caching of route patterns improves routing performance

## Docker Support

The load balancer includes Docker support for easy deployment:

- Multi-stage build for smaller image size
- Configuration via environment variables
- Ready-to-use Docker Compose configuration with example backends

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Documentation

* [Configuration Reference](configuration.md)
* [Architecture Overview](architecture.md)
* [Load Balancing Algorithms](algorithms.md)
* [Session Persistence](persistence.md)
* [WebSocket Support](websockets.md)
* [API Reference](api.md)
* [Performance Benchmarks](benchmarks.md)
* [Contributing](contributing.md)
* [Path-Based Routing](path-based-routing.md) 