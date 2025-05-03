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

// PersistenceMethod represents different session persistence methods
type PersistenceMethod string

const (
	// NoPersistence indicates no session persistence
	NoPersistence PersistenceMethod = "none"

	// CookiePersistence uses cookies to maintain session
	CookiePersistence PersistenceMethod = "cookie"

	// IPHashPersistence uses IP hashing to maintain session
	IPHashPersistence PersistenceMethod = "ip_hash"

	// ConsistentHashPersistence uses consistent hashing to maintain session
	ConsistentHashPersistence PersistenceMethod = "consistent_hash"
)

// LoadBalancerStrategy interface for different load balancing algorithms
type LoadBalancerStrategy interface {
	GetNextInstance(r *http.Request) *Process
	ProxyRequest(w http.ResponseWriter, r *http.Request)
}

// CreateLoadBalancer factory function to create a load balancer based on the algorithm
func CreateLoadBalancer(algorithm LoadBalancerAlgorithm, configs []BackendConfig, persistenceMethod PersistenceMethod) LoadBalancerStrategy {
	switch algorithm {
	case LeastConnections:
		return NewLeastConnectionsBalancer(configs)
	case WeightedRoundRobin, RoundRobin:
		if persistenceMethod != NoPersistence {
			return NewSessionPersistenceBalancer(configs, algorithm, persistenceMethod)
		}
		return NewLoadBalancer(configs)
	default:
		if persistenceMethod != NoPersistence {
			return NewSessionPersistenceBalancer(configs, WeightedRoundRobin, persistenceMethod)
		}
		return NewLoadBalancer(configs)
	}
}
