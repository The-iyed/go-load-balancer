package balancer

import (
	"net/http"
	"net/url"
)

// LegacyLoadBalancerAdapter adapts existing load balancers to the new interface
type LegacyLoadBalancerAdapter struct {
	wrappedBalancer interface{}
}

// NewRoundRobin creates a round robin load balancer
func NewRoundRobin(backends []BackendConfig) LoadBalancerStrategy {
	return &LegacyLoadBalancerAdapter{
		wrappedBalancer: NewLoadBalancer(backends),
	}
}

// NewWeightedRoundRobin creates a weighted round robin load balancer
func NewWeightedRoundRobin(backends []BackendConfig) LoadBalancerStrategy {
	return &LegacyLoadBalancerAdapter{
		wrappedBalancer: NewLoadBalancer(backends),
	}
}

// NewLeastConnections creates a least connections load balancer
func NewLeastConnections(backends []BackendConfig) LoadBalancerStrategy {
	return &LegacyLoadBalancerAdapter{
		wrappedBalancer: NewLeastConnectionsBalancer(backends),
	}
}

// NewSessionPersistence creates a session persistence wrapper
func NewSessionPersistence(strategy LoadBalancerStrategy, method PersistenceMethod, attrs map[string]string) (LoadBalancerStrategy, error) {
	// Since we're wrapping a strategy that is already using the new interface,
	// we need to get the backends from the underlying implementation
	// For simplicity, we'll use a fixed array for now
	configs := []BackendConfig{}

	if adapter, ok := strategy.(*LegacyLoadBalancerAdapter); ok {
		if lb, ok := adapter.wrappedBalancer.(*WeightedRoundRobinBalancer); ok {
			for _, process := range lb.ProcessPack {
				configs = append(configs, BackendConfig{
					URL:    process.URL.String(),
					Weight: process.Weight,
				})
			}
		}
	}

	var algorithm LoadBalancerAlgorithm
	if adapter, ok := strategy.(*LegacyLoadBalancerAdapter); ok {
		if _, ok := adapter.wrappedBalancer.(*WeightedRoundRobinBalancer); ok {
			algorithm = WeightedRoundRobin
		} else if _, ok := adapter.wrappedBalancer.(*LeastConnectionsBalancer); ok {
			algorithm = LeastConnections
		} else {
			algorithm = RoundRobin
		}
	}

	return &LegacyLoadBalancerAdapter{
		wrappedBalancer: NewSessionPersistenceBalancer(configs, algorithm, method),
	}, nil
}

// GetNextInstance implements the LoadBalancerStrategy interface
func (l *LegacyLoadBalancerAdapter) GetNextInstance(r *http.Request) (*url.URL, error) {
	var process *Process

	switch lb := l.wrappedBalancer.(type) {
	case *WeightedRoundRobinBalancer:
		process = lb.GetNextInstance(r)
	case *LeastConnectionsBalancer:
		process = lb.GetNextInstance(r)
	case *SessionPersistenceBalancer:
		url, err := lb.GetNextInstance(r)
		if err != nil {
			return nil, err
		}
		return url, nil
	}

	if process == nil {
		return nil, nil
	}

	return process.URL, nil
}

// ProxyRequest implements the LoadBalancerStrategy interface
func (l *LegacyLoadBalancerAdapter) ProxyRequest(w http.ResponseWriter, r *http.Request) {
	switch lb := l.wrappedBalancer.(type) {
	case *WeightedRoundRobinBalancer:
		lb.ProxyRequest(w, r)
	case *LeastConnectionsBalancer:
		lb.ProxyRequest(w, r)
	case *SessionPersistenceBalancer:
		lb.ProxyRequest(w, r)
	}
}

// SupportsWebSockets implements the LoadBalancerStrategy interface
func (l *LegacyLoadBalancerAdapter) SupportsWebSockets() bool {
	switch lb := l.wrappedBalancer.(type) {
	case *WeightedRoundRobinBalancer:
		return lb.SupportsWebSockets()
	case *LeastConnectionsBalancer:
		return lb.SupportsWebSockets()
	case *SessionPersistenceBalancer:
		return lb.SupportsWebSockets()
	}
	return false
}
