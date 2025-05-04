# Path-Based Routing

Path-based routing is a powerful feature that allows routing requests to different backend pools based on the request path, HTTP headers, or regex patterns. This enables more complex load balancing scenarios such as microservices architectures, content-based routing, and more.

## Features

- **Path-based routing**: Route requests to different backend pools based on URL path prefixes
- **Regex pattern matching**: Use regular expressions for complex routing rules
- **Header-based routing**: Route based on HTTP header values
- **Multiple backend pools**: Define groups of backend servers for different services
- **Fallback to default backend**: Requests that don't match any route are sent to the default backend
- **Compatible with all load balancing algorithms**: Each backend pool can use any supported load balancing method
- **Compatible with session persistence**: Each backend pool can use any supported session persistence method

## Configuration

Path-based routing is configured through the standard configuration file. Here's the syntax:

```conf
# Define backend pools
upstream <pool-name> {
    server <server-url> [weight=n] [max_conn=n]
    server <server-url> [weight=n] [max_conn=n]
    ...
}

# Define routes
route path <path-prefix> <backend-pool>     # Path-based route
route regex <regex-pattern> <backend-pool>  # Regex-based route
route header <header-name> <value> <backend-pool> # Header-based route

# Set default backend pool
default_backend <backend-pool>

# Global settings
method <load-balancing-method>
persistence <persistence-method> [options...]
```

### Example Configuration

```conf
# API Backend Servers
upstream api {
    server http://localhost:8081 weight=5
    server http://localhost:8082 weight=5
}

# Static Content Servers
upstream static {
    server http://localhost:8091 weight=10
}

# Default Backend Servers
upstream backend {
    server http://localhost:8001 weight=10
    server http://localhost:8002 weight=5
}

# Route definitions
route path /api/ api
route path /static/ static
route regex ^/api/v2/.* api
route header User-Agent Mobile mobile-backend

# Default backend pool
default_backend backend

# Load balancing method
method weighted_round_robin

# Session persistence
persistence cookie
```

## Route Types

### Path-Based Routes

Path-based routes match against the start of the request path. If the path starts with the specified prefix, the request is routed to the corresponding backend pool.

```conf
route path /api/ api
route path /static/ static
route path /admin/ admin
```

### Regex-Based Routes

Regex-based routes use Go's regular expression syntax to match against the full request path. This allows for more complex routing patterns.

```conf
route regex ^/api/v[12]/.*$ api-backend
route regex ^/images/users/\d+/.*$ user-images
route regex ^/(en|fr|de)/.*$ localized-content
```

### Header-Based Routes

Header-based routes match against HTTP header values. This is useful for device detection, internal services, and more.

```conf
route header User-Agent Mobile mobile-backend
route header X-Internal true admin-backend
route header Accept application/json api-backend
```

## Command Line Usage

Path-based routing is automatically enabled when route directives are detected in the configuration file. You can also enable it explicitly using the `--path-routing` flag:

```bash
./load-balancer --config=my-config.conf --path-routing
```

## How It Works

1. When a request is received, the load balancer checks it against each defined route in the order they appear in the configuration.
2. The first matching route determines which backend pool will handle the request.
3. The appropriate load balancing algorithm (as configured) is used to select a specific backend server from that pool.
4. If no routes match, the request is sent to the default backend pool.

## Best Practices

- **Order matters**: Routes are evaluated in the order they appear in the configuration. More specific routes should be placed before more general ones.
- **Use descriptive pool names**: Choose meaningful names for your backend pools to make configuration more readable.
- **Be careful with regex patterns**: Complex regex patterns can impact performance. Test thoroughly.
- **Combine with session persistence**: For stateful applications, ensure consistent routing by configuring session persistence.
- **Monitor backend health**: Even with path-based routing, backend health monitoring remains critical.

## Example Use Cases

### Microservices Architecture

```conf
upstream auth-service {
    server http://auth-service:8000
    server http://auth-service-replica:8000
}

upstream product-service {
    server http://product-service:8001
    server http://product-service-replica:8001
}

upstream user-service {
    server http://user-service:8002
    server http://user-service-replica:8002
}

route path /api/auth/ auth-service
route path /api/products/ product-service
route path /api/users/ user-service
```

### Content-Type Routing

```conf
upstream api-servers {
    server http://api1:8000
    server http://api2:8000
}

upstream web-servers {
    server http://web1:8080
    server http://web2:8080
}

route header Accept application/json api-servers
route header Accept text/html web-servers
```

### Device-Based Routing

```conf
upstream mobile-site {
    server http://mobile1:8080
    server http://mobile2:8080
}

upstream desktop-site {
    server http://desktop1:8080
    server http://desktop2:8080
}

route header User-Agent Mobile mobile-site
route header User-Agent Android mobile-site
route header User-Agent iPhone mobile-site
```

## Debugging and Troubleshooting

If you experience issues with path-based routing:

1. Check the log output for routing decisions
2. Verify that your routes are defined in the correct order
3. Test regex patterns independently to ensure they match as expected
4. Check that the specified backend pools exist in the configuration
5. Ensure backend servers in each pool are healthy and responding 