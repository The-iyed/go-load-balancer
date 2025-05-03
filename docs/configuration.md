# Configuration Guide

<p align="center">
  <img src="images/logo.png" alt="Go Load Balancer Logo" width="250">
</p>

This document explains how to configure the Go Load Balancer for various use cases.

## Configuration File Format

The load balancer uses a simple configuration file format to define backend servers and their properties.

### Basic Syntax

```
upstream backend {
    method <METHOD>;
    persistence <PERSISTENCE>;
    server <URL> weight=<WEIGHT>;
    server <URL> weight=<WEIGHT>;
    ...
}
```

Where:
- `<METHOD>` is the load balancing algorithm to use (weighted_round_robin, round_robin, least_conn)
- `<PERSISTENCE>` is the session persistence method to use (none, cookie, ip_hash, consistent_hash)
- `<URL>` is the URL of the backend server (e.g., `http://backend1:80`)
- `<WEIGHT>` is the weight of the server (default: 1)

### Parameters

| Parameter | Default | Description |
|-----------|---------|-------------|
| `method` | `weighted_round_robin` | The load balancing algorithm to use |
| `persistence` | `none` | The session persistence method to use |
| `weight` | 1 | The relative weight of the server for weighted algorithms |

### Available Methods

| Method Name | Description |
|-------------|-------------|
| `weighted_round_robin` | Distributes traffic based on server weights |
| `round_robin` | Simple round-robin distribution (weights are ignored) |
| `least_conn` | Routes to the server with the fewest active connections |

### Available Persistence Methods

| Persistence Method | Description |
|-------------|-------------|
| `none` | No session persistence (default) |
| `cookie` | Uses cookies to maintain client sessions with the same backend |
| `ip_hash` | Uses client IP address to determine the backend server |
| `consistent_hash` | Uses consistent hashing on request path for even distribution |

## Example Configurations

### Basic Configuration

A simple configuration with three equal backends using weighted round robin:

```
upstream backend {
    method weighted_round_robin;
    server http://backend1:80;
    server http://backend2:80;
    server http://backend3:80;
}
```

### Weighted Round Robin Configuration

A configuration for Weighted Round Robin load balancing with different server capacities:

```
upstream backend {
    method weighted_round_robin;
    server http://backend1:80 weight=5;  # 50% of traffic
    server http://backend2:80 weight=3;  # 30% of traffic
    server http://backend3:80 weight=2;  # 20% of traffic
}
```

### Least Connections Configuration

A configuration for Least Connections load balancing:

```
upstream backend {
    method least_conn;
    server http://backend1:80 weight=1;
    server http://backend2:80 weight=1;
    server http://backend3:80 weight=1;
}
```

### Cookie-Based Session Persistence

A configuration using cookies for session persistence:

```
upstream backend {
    method weighted_round_robin;
    persistence cookie;
    server http://backend1:80 weight=3;
    server http://backend2:80 weight=2;
    server http://backend3:80 weight=1;
}
```

### IP Hash Persistence

A configuration using client IP hashing for persistence:

```
upstream backend {
    method least_conn;
    persistence ip_hash;
    server http://backend1:80 weight=1;
    server http://backend2:80 weight=1;
    server http://backend3:80 weight=1;
}
```

### Consistent Hashing Persistence

A configuration using consistent hashing for persistence:

```
upstream backend {
    method weighted_round_robin;
    persistence consistent_hash;
    server http://backend1:80 weight=3;
    server http://backend2:80 weight=2;
    server http://backend3:80 weight=1;
}
```

## Docker Environment

When running in Docker, the configuration typically uses the Docker service names instead of localhost:

```
upstream backend {
    method weighted_round_robin;
    persistence cookie;
    server http://backend1:80 weight=3;
    server http://backend2:80 weight=2;
    server http://backend3:80 weight=1;
}
```

## Advanced Configuration

### Multiple Backend Groups

The current implementation supports a single backend group. To implement multiple backend groups, you would need to extend the configuration parser to support multiple upstream blocks and implement routing logic.

Future version example:

```
upstream api {
    method least_conn;
    persistence ip_hash;
    server http://api1:80 weight=1;
    server http://api2:80 weight=1;
}

upstream static {
    method weighted_round_robin;
    persistence consistent_hash;
    server http://static1:80 weight=1;
    server http://static2:80 weight=1;
}

routes {
    path /api/* upstream api;
    path /* upstream static;
}
```

### SSL/TLS Termination

The current implementation does not directly support SSL/TLS termination. For production environments, consider using a reverse proxy like Nginx in front of the load balancer or extending the code to support TLS.

## Running with Custom Configuration

To use a custom configuration file:

```bash
./load-balancer --config=path/to/your/config.conf
```

To override the method specified in the config file:

```bash
./load-balancer --algorithm=least-connections
```

To override the persistence method:

```bash
./load-balancer --persistence=cookie
```

## Configuration Best Practices

1. **Balance Weight Distribution**: Assign weights that reflect the true capacity ratio of your servers
2. **Consider Resource Usage**: For Least Connections, make sure your weights align with your server capacity
3. **Use Health Checks**: The load balancer has passive health checking built-in
4. **Choose Appropriate Persistence**: Select the right persistence method for your application:
   - `cookie` for standard web applications
   - `ip_hash` when cookies cannot be used
   - `consistent_hash` for distributed systems
5. **Monitor and Adjust**: Review the distribution and adjust weights or algorithms as needed

## Troubleshooting

### Common Issues

1. **Backend Not Receiving Traffic**: Check if the backend URL is correct and the server is running
2. **Uneven Distribution**: For Weighted Round Robin, check if the weights are set correctly
3. **Connection Refused Errors**: Ensure the backend servers are accepting connections on the specified ports
4. **All Backends Unhealthy**: Check if at least one backend is operational
5. **Session Persistence Issues**: Verify the client supports the chosen persistence method

### Logs

The load balancer logs important events, including:

- Backend server failures
- Backends marked as unhealthy
- Backend recovery events
- Session persistence decisions

To view these logs, check the standard output of the load balancer process.

package balancer

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

type Config struct {
	Backends    []BackendConfig
	Method      LoadBalancerAlgorithm
	Persistence PersistenceMethod
}

type BackendConfig struct {
	URL    string
	Weight int
} 