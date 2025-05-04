package balancer

import (
	"net/http"
	"net/url"
)

// LoadBalancerAlgorithm represents the load balancing algorithm
type LoadBalancerAlgorithm int

const (
	// RoundRobin distributes requests in a circular fashion
	RoundRobin LoadBalancerAlgorithm = iota
	// WeightedRoundRobin distributes requests proportionally to backend weights
	WeightedRoundRobin
	// LeastConnections distributes requests to backends with the least active connections
	LeastConnections
	// PathBasedRouting routes requests based on URL paths, headers, or patterns
	PathBasedRouting
)

// PersistenceMethod represents the session persistence method
type PersistenceMethod int

const (
	// NoPersistence disables session persistence
	NoPersistence PersistenceMethod = iota
	// CookiePersistence uses HTTP cookies for session persistence
	CookiePersistence
	// IPHashPersistence uses client IP address hashing for persistence
	IPHashPersistence
	// ConsistentHashPersistence uses a consistent hashing algorithm
	ConsistentHashPersistence
)

// LoadBalancerStrategy defines the interface for load balancing strategies
type LoadBalancerStrategy interface {
	// GetNextInstance selects the next backend instance for a request
	GetNextInstance(r *http.Request) (*url.URL, error)
	// ProxyRequest handles proxying the HTTP request to a backend
	ProxyRequest(w http.ResponseWriter, r *http.Request)
	// SupportsWebSockets returns true if the load balancer supports WebSocket connections
	SupportsWebSockets() bool
}

// CreateLoadBalancer creates a load balancer with the specified algorithm
func CreateLoadBalancer(
	algorithm LoadBalancerAlgorithm,
	backends []BackendConfig,
	persistenceMethod PersistenceMethod,
	persistenceAttrs map[string]string,
) (LoadBalancerStrategy, error) {
	var baseBalancer LoadBalancerStrategy
	var err error

	// Create the base load balancer according to the algorithm
	switch algorithm {
	case RoundRobin:
		baseBalancer = NewRoundRobin(backends)
	case WeightedRoundRobin:
		baseBalancer = NewWeightedRoundRobin(backends)
	case LeastConnections:
		baseBalancer = NewLeastConnections(backends)
	default:
		return nil, ErrInvalidConfig{Message: "unsupported load balancing algorithm"}
	}

	// Apply session persistence if enabled
	if persistenceMethod != NoPersistence {
		baseBalancer, err = NewSessionPersistence(baseBalancer, persistenceMethod, persistenceAttrs)
		if err != nil {
			return nil, err
		}
	}

	return baseBalancer, nil
}

// CreatePathRouter creates a path-based router with multiple backend pools
func CreatePathRouter(
	config *Config,
) (LoadBalancerStrategy, error) {
	// Create a load balancer for each backend pool
	backendPools := make(map[string]LoadBalancerStrategy)

	// First create the default backend pool
	defaultPool, exists := config.BackendPools[config.DefaultBackend]
	if !exists {
		return nil, ErrInvalidConfig{Message: "default backend pool not found: " + config.DefaultBackend}
	}

	defaultLB, err := CreateLoadBalancer(
		config.Method,
		defaultPool,
		config.PersistenceType,
		config.PersistenceAttrs,
	)
	if err != nil {
		return nil, err
	}
	backendPools[config.DefaultBackend] = defaultLB

	// Create load balancers for all other backend pools
	for name, pool := range config.BackendPools {
		if name == config.DefaultBackend {
			continue // Already created
		}

		lb, err := CreateLoadBalancer(
			config.Method,
			pool,
			config.PersistenceType,
			config.PersistenceAttrs,
		)
		if err != nil {
			return nil, err
		}
		backendPools[name] = lb
	}

	// Create the path router with all backend pools
	return NewPathRouter(config.Routes, backendPools, config.DefaultBackend)
}
