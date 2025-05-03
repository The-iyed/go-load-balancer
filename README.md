# Go Load Balancer

[![Go Report Card](https://goreportcard.com/badge/github.com/The-iyed/go-load-balancer)](https://goreportcard.com/report/github.com/The-iyed/go-load-balancer)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

A high-performance HTTP load balancer written in Go with support for multiple load balancing algorithms and Docker integration.

![Load Balancer Diagram](docs/images/load-balancer-diagram.png)

## Features

- **Multiple Load Balancing Algorithms**
  - Weighted Round Robin
  - Least Connections
- **Nginx-style Configuration**
  - Simple, familiar syntax
  - Algorithm selection in config file
- **Health Checking**
  - Automatic detection of failed backends
  - Self-healing with automatic recovery
- **Production Ready**
  - Docker and Docker Compose support
  - Customizable via command-line flags

## Quick Start

```bash
# Clone the repository
git clone https://github.com/The-iyed/go-load-balancer.git
cd go-load-balancer

# Run with Docker Compose
docker-compose up --build

# Or build and run natively
go build -o load-balancer ./cmd/server/main.go
./load-balancer
```

Then access the load balancer at http://localhost:8080

## Usage

### Command Line Options

```bash
# Basic usage
./load-balancer

# With custom configuration
./load-balancer --config=conf/loadbalancer.conf

# Override the method in config file
./load-balancer --algorithm=least-connections
```

### Configuration Example

```
upstream backend {
    method weighted_round_robin;  # Load balancing algorithm
    server http://backend1:80 weight=3;
    server http://backend2:80 weight=2;
    server http://backend3:80 weight=1;
}
```

Available methods:
- `weighted_round_robin` - Distributes traffic based on weights
- `round_robin` - Simple round-robin (ignores weights)
- `least_conn` - Routes to server with fewest active connections

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