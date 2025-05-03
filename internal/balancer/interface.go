package balancer

import (
	"net/http"
)

// LoadBalancerAlgorithm represents different load balancing algorithms
type LoadBalancerAlgorithm string

const (
	// RoundRobin is a simple round-robin load balancing algorithm
	RoundRobin LoadBalancerAlgorithm = "round-robin"

	// WeightedRoundRobin distributes traffic based on server weights
	WeightedRoundRobin LoadBalancerAlgorithm = "weighted-round-robin"

	// LeastConnections routes to the server with fewest active connections
	LeastConnections LoadBalancerAlgorithm = "least-connections"
)

// LoadBalancerStrategy interface for different load balancing algorithms
type LoadBalancerStrategy interface {
	GetNextInstance() *Process
	ProxyRequest(w http.ResponseWriter, r *http.Request)
}

// CreateLoadBalancer factory function to create a load balancer based on the algorithm
func CreateLoadBalancer(algorithm LoadBalancerAlgorithm, configs []BackendConfig) LoadBalancerStrategy {
	switch algorithm {
	case LeastConnections:
		return NewLeastConnectionsBalancer(configs)
	case WeightedRoundRobin, RoundRobin:
		// Default to weighted round robin which handles the standard round robin as well
		return NewLoadBalancer(configs)
	default:
		// Default to weighted round robin
		return NewLoadBalancer(configs)
	}
}
