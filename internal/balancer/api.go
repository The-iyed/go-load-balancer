package balancer

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/The-iyed/go-load-balancer/internal/logger"
	"go.uber.org/zap"
)

// Stats holds the statistics for the load balancer
type Stats struct {
	Backends        []BackendStats    `json:"backends"`
	Method          string            `json:"method"`
	TotalRequests   int64             `json:"totalRequests"`
	PersistenceType string            `json:"persistenceType"`
	RouteStats      map[string]string `json:"routeStats,omitempty"`
	StartTime       time.Time         `json:"startTime"`
	Uptime          string            `json:"uptime"`
}

// BackendStats holds the statistics for a backend server
type BackendStats struct {
	URL             string  `json:"url"`
	Alive           bool    `json:"alive"`
	Weight          int     `json:"weight"`
	RequestCount    int64   `json:"requestCount"`
	ErrorCount      int32   `json:"errorCount"`
	LoadPercentage  float64 `json:"loadPercentage"`
	ResponseTimeAvg int64   `json:"responseTimeAvg"`
}

var (
	// Global stats instance
	globalStats     Stats
	globalStatsMu   sync.RWMutex
	startTime       = time.Now()
	totalRequests   int64
	requestCountsMu sync.RWMutex
)

// GetStats returns the current statistics
func GetStats(lb LoadBalancerStrategy) Stats {
	globalStatsMu.Lock()
	defer globalStatsMu.Unlock()

	// Update the stats
	UpdateStats(lb)

	// Calculate uptime
	globalStats.Uptime = time.Since(startTime).String()

	return globalStats
}

// UpdateStats updates the global statistics
func UpdateStats(lb LoadBalancerStrategy) {
	// Update total requests
	requestCountsMu.RLock()
	globalStats.TotalRequests = totalRequests
	requestCountsMu.RUnlock()

	// Update start time
	globalStats.StartTime = startTime

	// Handle different types of load balancers
	switch typedLB := lb.(type) {
	case *SessionPersistenceBalancer:
		updateSessionPersistenceStats(typedLB)
	case *PathRouter:
		updatePathRouterStats(typedLB)
	case *LegacyLoadBalancerAdapter:
		updateLegacyAdapterStats(typedLB)
	default:
		logger.Log.Warn("Unknown load balancer type for statistics")
	}
}

// updateSessionPersistenceStats updates statistics for session persistence balancers
func updateSessionPersistenceStats(lb *SessionPersistenceBalancer) {
	globalStats.Method = getMethodName(lb.BaseLB)
	globalStats.PersistenceType = getPersistenceMethodName(lb.PersistenceMethod)

	// Get backends stats
	totalRequests := int64(0)
	backends := make([]BackendStats, 0, len(lb.ProcessPack))

	for _, process := range lb.ProcessPack {
		reqCount := process.GetRequestCount()
		totalRequests += reqCount

		backends = append(backends, BackendStats{
			URL:             process.URL.String(),
			Alive:           process.IsAlive(),
			Weight:          process.Weight,
			RequestCount:    reqCount,
			ErrorCount:      process.ErrorCount,
			ResponseTimeAvg: 0, // We don't track this yet
		})
	}

	// Calculate load percentages
	if totalRequests > 0 {
		for i := range backends {
			backends[i].LoadPercentage = float64(backends[i].RequestCount) / float64(totalRequests) * 100
		}
	}

	globalStats.Backends = backends
}

// updatePathRouterStats updates statistics for path router
func updatePathRouterStats(lb *PathRouter) {
	globalStats.Method = "Path Router"
	globalStats.PersistenceType = "N/A"

	// Collect route stats
	routeStats := make(map[string]string)
	for i, route := range lb.routes {
		routeStats[fmt.Sprintf("route_%d", i)] = route.Pattern
	}
	globalStats.RouteStats = routeStats

	// We don't have direct access to backend stats in path router
	// This would need to be implemented in the PathRouter to track properly
	globalStats.Backends = []BackendStats{}
}

// updateLegacyAdapterStats updates statistics for legacy adapter
func updateLegacyAdapterStats(lb *LegacyLoadBalancerAdapter) {
	// Use method mapping from adapter.go
	switch lb.wrappedBalancer.(type) {
	case *WeightedRoundRobinBalancer:
		globalStats.Method = "Weighted Round Robin"
	case *LeastConnectionsBalancer:
		globalStats.Method = "Least Connections"
	case *SessionPersistenceBalancer:
		spb := lb.wrappedBalancer.(*SessionPersistenceBalancer)
		globalStats.Method = getMethodName(spb.BaseLB)
		globalStats.PersistenceType = getPersistenceMethodName(spb.PersistenceMethod)
		return
	default:
		globalStats.Method = "Round Robin"
	}

	globalStats.PersistenceType = "None"

	// This is just a stub - we'd need to implement request tracking in the adapter
	globalStats.Backends = []BackendStats{}
}

// getMethodName returns the name of the load balancing method
func getMethodName(lb interface{}) string {
	switch lb.(type) {
	case *WeightedRoundRobinBalancer:
		return "Weighted Round Robin"
	case *LeastConnectionsBalancer:
		return "Least Connections"
	default:
		return "Round Robin"
	}
}

// getPersistenceMethodName returns the name of the persistence method
func getPersistenceMethodName(method PersistenceMethod) string {
	switch method {
	case CookiePersistence:
		return "Cookie"
	case IPHashPersistence:
		return "IP Hash"
	case ConsistentHashPersistence:
		return "Consistent Hash"
	case NoPersistence:
		return "None"
	default:
		return "Unknown"
	}
}

// IncrementRequestCount increments the total request count
func IncrementRequestCount() {
	requestCountsMu.Lock()
	defer requestCountsMu.Unlock()
	totalRequests++
}

// APIHandler handles API requests for stats
func APIHandler(lb LoadBalancerStrategy) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Set CORS headers for the preflight request
		if r.Method == http.MethodOptions {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			w.WriteHeader(http.StatusOK)
			return
		}

		// Set CORS headers for the main request
		w.Header().Set("Access-Control-Allow-Origin", "*")

		// Only allow GET requests
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Get stats and return as JSON
		stats := GetStats(lb)

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(stats); err != nil {
			logger.Log.Error("Failed to encode stats", zap.Error(err))
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
	}
}

// Add a method to Process to get request count
func (p *Process) GetRequestCount() int64 {
	// We'll need to add a proper request counter in the Process struct later
	// For now, return 0
	return 0
}
