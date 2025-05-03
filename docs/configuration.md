# Configuration Guide

This document explains how to configure the Go Load Balancer for various use cases.

## Configuration File Format

The load balancer uses a simple configuration file format to define backend servers and their properties.

### Basic Syntax

```
upstream backend {
    method <METHOD>;
    server <URL> weight=<WEIGHT>;
    server <URL> weight=<WEIGHT>;
    ...
}
```

Where:
- `<METHOD>` is the load balancing algorithm to use (weighted_round_robin, round_robin, least_conn)
- `<URL>` is the URL of the backend server (e.g., `http://backend1:80`)
- `<WEIGHT>` is the weight of the server (default: 1)

### Parameters

| Parameter | Default | Description |
|-----------|---------|-------------|
| `method` | `weighted_round_robin` | The load balancing algorithm to use |
| `weight` | 1 | The relative weight of the server for weighted algorithms |

### Available Methods

| Method Name | Description |
|-------------|-------------|
| `weighted_round_robin` | Distributes traffic based on server weights |
| `round_robin` | Simple round-robin distribution (weights are ignored) |
| `least_conn` | Routes to the server with the fewest active connections |

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

### Weighted Least Connections Configuration

A configuration for Least Connections that also considers server capacity:

```
upstream backend {
    method least_conn;
    server http://backend1:80 weight=4;  # High-capacity server
    server http://backend2:80 weight=2;  # Medium-capacity server
    server http://backend3:80 weight=1;  # Low-capacity server
}
```

## Docker Environment

When running in Docker, the configuration typically uses the Docker service names instead of localhost:

```
upstream backend {
    method weighted_round_robin;
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
    server http://api1:80 weight=1;
    server http://api2:80 weight=1;
}

upstream static {
    method weighted_round_robin;
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

## Configuration Best Practices

1. **Balance Weight Distribution**: Assign weights that reflect the true capacity ratio of your servers
2. **Consider Resource Usage**: For Least Connections, make sure your weights align with your server capacity
3. **Use Health Checks**: The load balancer has passive health checking built-in
4. **Monitor and Adjust**: Review the distribution and adjust weights or algorithms as needed

## Troubleshooting

### Common Issues

1. **Backend Not Receiving Traffic**: Check if the backend URL is correct and the server is running
2. **Uneven Distribution**: For Weighted Round Robin, check if the weights are set correctly
3. **Connection Refused Errors**: Ensure the backend servers are accepting connections on the specified ports
4. **All Backends Unhealthy**: Check if at least one backend is operational

### Logs

The load balancer logs important events, including:

- Backend server failures
- Backends marked as unhealthy
- Backend recovery events

To view these logs, check the standard output of the load balancer process. 