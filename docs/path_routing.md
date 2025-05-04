# Path-Based Routing

Path-based routing allows directing traffic to different backend pools based on the URL path, regex patterns, or HTTP headers. This enables more complex routing scenarios where different parts of an application can be served by specialized backend servers.

## Features

- **Path-based routing**: Route requests to specific backend pools based on URL path prefixes
- **Regex pattern matching**: Route requests using regular expressions for more complex path matching
- **Header-based routing**: Route requests based on HTTP header values
- **Fallback to default backends**: Requests that don't match any routing rules are sent to the default backend pool

## Configuration

Path-based routing is configured in the load balancer configuration file. Here's a complete example:

```
# Path-based routing configuration sample

# Default backend pool
upstream backend {
    method weighted_round_robin
    persistence cookie
    server http://backend1:80 weight=1
    server http://backend2:80 weight=1
}

# API backend pool
upstream api_servers {
    method least_connections
    server http://api1:80 weight=1
    server http://api2:80 weight=1
    server http://api3:80 weight=1
}

# Static content backend pool
upstream static_servers {
    method round_robin
    server http://static1:80 weight=1
    server http://static2:80 weight=1
}

# WebSocket backend pool
upstream websocket_servers {
    method least_connections
    server http://ws1:80 weight=1
    server http://ws2:80 weight=1
}

# Routing rules
route path /api/ api_servers
route regex ^/v[0-9]+/api/.* api_servers
route path /static/ static_servers
route path /images/ static_servers
route path /ws/ websocket_servers
route header X-API-Version v2 api_servers

# Default backend pool to use if no routes match
default_backend backend
```

### Backend Pools

Backend pools are defined using the `upstream` directive and can have their own load balancing method and server configuration:

```
upstream <pool_name> {
    method <load_balancing_method>
    [persistence <persistence_method>]
    server <server_url> [weight=<weight>]
    ...
}
```

### Routing Rules

There are three types of routing rules:

1. **Path routing**:
   ```
   route path /path_prefix/ backend_pool_name
   ```
   
2. **Regex routing**:
   ```
   route regex ^/regex_pattern/.* backend_pool_name
   ```
   
3. **Header routing**:
   ```
   route header Header-Name header-value backend_pool_name
   ```

### Default Backend

The `default_backend` directive specifies which backend pool to use when no routing rules match:

```
default_backend <pool_name>
```

## Using Path-Based Routing

To enable path-based routing, pass the `-path-routing` flag when starting the load balancer:

```bash
go run cmd/server/main.go -config conf/path_routing.conf -path-routing
```

You can also specify the configuration file path using the `-config` flag.

## Routing Precedence

Routing rules are evaluated in the order they appear in the configuration file. The first matching rule is used to determine the backend pool. If no rules match, the default backend pool is used.

## Examples

### Web Application with API and Static Content

For a web application with separate API and static content servers:

```
upstream app {
    method round_robin
    server http://app1:80
    server http://app2:80
}

upstream api {
    method least_connections
    server http://api1:80
    server http://api2:80
}

upstream static {
    method round_robin
    server http://static1:80
}

route path /api/ api
route path /static/ static
default_backend app
```

### Multi-Version API Routing

For routing to different API versions based on URL path:

```
upstream api_v1 {
    method round_robin
    server http://api-v1-1:80
    server http://api-v1-2:80
}

upstream api_v2 {
    method round_robin
    server http://api-v2-1:80
    server http://api-v2-2:80
}

route path /v1/api/ api_v1
route path /v2/api/ api_v2
```

### Header-Based Routing

For routing based on HTTP headers, useful for A/B testing or gradual rollouts:

```
upstream production {
    method round_robin
    server http://prod1:80
    server http://prod2:80
}

upstream beta {
    method round_robin
    server http://beta1:80
    server http://beta2:80
}

route header X-Beta-User true beta
default_backend production
```

## Implementation Details

Path-based routing in this load balancer is implemented using the `PathRouter` struct, which:

1. Parses the routing rules from the configuration file
2. Initializes a separate load balancer for each backend pool
3. Routes incoming requests to the appropriate backend pool based on the routing rules
4. Proxies requests to the selected backend server

The actual backend selection is still handled by the configured load balancing algorithm for each pool. 