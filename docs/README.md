# Go Load Balancer

A high-performance HTTP load balancer implemented in Go with support for multiple load balancing algorithms.

## Features

- Multiple load balancing algorithms:
  - Weighted Round Robin
  - Least Connections
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
```

### Available Command-Line Options

| Option | Default | Description |
|--------|---------|-------------|
| `--config` | `conf/loadbalancer.conf` | Path to the configuration file |
| `--algorithm` | `` | Override the load balancing algorithm defined in config |

### Supported Algorithms

- `weighted-round-robin`: Distributes traffic based on server weights
- `round-robin`: Simple round-robin distribution (weights are ignored)
- `least-connections`: Routes to the server with the fewest active connections

## Configuration

The load balancer is configured using a simple Nginx-inspired syntax.

### Configuration File Format

```
upstream backend {
    method weighted_round_robin;  # Load balancing algorithm
    server <URL> weight=<WEIGHT>;
    server <URL> weight=<WEIGHT>;
    ...
}
```

Where:
- `method` specifies the load balancing algorithm to use
- `<URL>` is the URL of the backend server (e.g., `http://backend1:80`)
- `<WEIGHT>` is the weight of the server (default: 1)

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
- **Health Checking**: Monitors backend health and handles failure recovery
- **Request Proxying**: Proxies client requests to selected backends

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
    method weighted_round_robin;
    server http://backend1:80 weight=5;
    server http://backend2:80 weight=3;
    server http://backend3:80 weight=2;
}
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
    method least_conn;
    server http://backend1:80 weight=1;
    server http://backend2:80 weight=1;
    server http://backend3:80 weight=1;
}
```

#### When to Use

- When request processing times vary significantly
- When some requests take much longer than others
- When backends can become overloaded easily

## Performance Considerations

- The load balancer uses Go's concurrency primitives for high performance
- Connection tracking uses atomic operations to avoid locks
- The proxy implementation is based on Go's standard library reverse proxy

## Docker Support

The load balancer includes Docker support for easy deployment:

- Multi-stage build for smaller image size
- Configuration via environment variables
- Ready-to-use Docker Compose configuration with example backends

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the MIT License - see the LICENSE file for details. 