package balancer

import (
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
)

// RouteType definitions are now in config.go

// PathRouter handles routing requests to different backend pools based on rules
type PathRouter struct {
	routes        []RouteConfig
	backendPools  map[string]LoadBalancerStrategy
	defaultPool   LoadBalancerStrategy
	defaultPoolID string
}

// ErrInvalidConfig represents a configuration error
type ErrInvalidConfig struct {
	Message string
}

func (e ErrInvalidConfig) Error() string {
	return fmt.Sprintf("invalid configuration: %s", e.Message)
}

// NewPathRouter creates a new path-based router
func NewPathRouter(
	routes []RouteConfig,
	backendPools map[string]LoadBalancerStrategy,
	defaultPool string,
) (*PathRouter, error) {
	// Validate that the default pool exists
	defaultLB, exists := backendPools[defaultPool]
	if !exists {
		return nil, ErrInvalidConfig{Message: "default backend pool not found"}
	}

	// Validate that all route backend pools exist
	for _, route := range routes {
		if _, exists := backendPools[route.BackendPool]; !exists {
			return nil, ErrInvalidConfig{Message: "route references non-existent backend pool: " + route.BackendPool}
		}
	}

	// Precompile regex patterns for regex routes
	for _, route := range routes {
		if route.Type == RegexRoute {
			_, err := regexp.Compile(route.Pattern)
			if err != nil {
				return nil, ErrInvalidConfig{Message: "invalid regex pattern: " + route.Pattern}
			}
		}
	}

	return &PathRouter{
		routes:        routes,
		backendPools:  backendPools,
		defaultPool:   defaultLB,
		defaultPoolID: defaultPool,
	}, nil
}

// Route determines which backend pool should handle the request
func (pr *PathRouter) Route(r *http.Request) LoadBalancerStrategy {
	// Check each route in order
	for _, route := range pr.routes {
		var matched bool

		switch route.Type {
		case PathRoute:
			// Simple path prefix matching
			matched = strings.HasPrefix(r.URL.Path, route.Pattern)

		case RegexRoute:
			// Regex matching for path
			re, _ := regexp.Compile(route.Pattern)
			matched = re.MatchString(r.URL.Path)

		case HeaderRoute:
			// Match based on HTTP header
			headerValue := r.Header.Get(route.HeaderName)
			matched = headerValue == route.HeaderValue
		}

		if matched {
			return pr.backendPools[route.BackendPool]
		}
	}

	// Default to the default backend pool
	return pr.defaultPool
}

// GetNextInstance selects the appropriate backend pool and gets the next instance
func (pr *PathRouter) GetNextInstance(r *http.Request) (*url.URL, error) {
	lb := pr.Route(r)
	return lb.GetNextInstance(r)
}

// ProxyRequest routes the request to the appropriate backend pool
func (pr *PathRouter) ProxyRequest(w http.ResponseWriter, r *http.Request) {
	lb := pr.Route(r)
	lb.ProxyRequest(w, r)
}

// SupportsWebSockets checks if the router supports WebSockets
func (pr *PathRouter) SupportsWebSockets() bool {
	// Check if all backend pools support WebSockets
	for _, pool := range pr.backendPools {
		if !pool.SupportsWebSockets() {
			return false
		}
	}
	return true
}
