package mocks

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"time"
)

// MockBackend represents a mock backend server for testing
type MockBackend struct {
	Server          *httptest.Server
	ID              int
	RequestCount    atomic.Int32
	ResponseDelay   time.Duration
	FailureRate     float64
	FailureCount    atomic.Int32
	SuccessCount    atomic.Int32
	LastRequestTime time.Time
}

// NewMockBackend creates a new mock backend server for testing
func NewMockBackend(id int, responseDelay time.Duration, failureRate float64) *MockBackend {
	mb := &MockBackend{
		ID:            id,
		ResponseDelay: responseDelay,
		FailureRate:   failureRate,
	}

	mb.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mb.RequestCount.Add(1)
		mb.LastRequestTime = time.Now()

		// Simulate processing delay
		if mb.ResponseDelay > 0 {
			time.Sleep(mb.ResponseDelay)
		}

		// Simulate failure based on failure rate
		if mb.FailureRate > 0 && float64(mb.FailureCount.Load())/float64(mb.RequestCount.Load()) < mb.FailureRate {
			mb.FailureCount.Add(1)
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, "Backend %d error", mb.ID)
			return
		}

		mb.SuccessCount.Add(1)
		w.Header().Set("X-Backend-ID", fmt.Sprintf("%d", mb.ID))
		w.Header().Set("X-Request-Count", fmt.Sprintf("%d", mb.RequestCount.Load()))
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "Response from backend %d", mb.ID)
	}))

	return mb
}

// URL returns the URL of the mock backend
func (mb *MockBackend) URL() string {
	return mb.Server.URL
}

// Close shuts down the mock backend server
func (mb *MockBackend) Close() {
	mb.Server.Close()
}

// ResetStats resets the statistics of the mock backend
func (mb *MockBackend) ResetStats() {
	mb.RequestCount.Store(0)
	mb.FailureCount.Store(0)
	mb.SuccessCount.Store(0)
}

// BackendCluster represents a group of mock backends
type BackendCluster struct {
	Backends []*MockBackend
}

// NewBackendCluster creates a new cluster of mock backends
func NewBackendCluster(count int, responseDelays []time.Duration, failureRates []float64) *BackendCluster {
	cluster := &BackendCluster{
		Backends: make([]*MockBackend, count),
	}

	for i := 0; i < count; i++ {
		delay := time.Duration(0)
		if i < len(responseDelays) {
			delay = responseDelays[i]
		}

		failRate := 0.0
		if i < len(failureRates) {
			failRate = failureRates[i]
		}

		cluster.Backends[i] = NewMockBackend(i+1, delay, failRate)
	}

	return cluster
}

// URLs returns the URLs of all backends in the cluster
func (bc *BackendCluster) URLs() []string {
	urls := make([]string, len(bc.Backends))
	for i, backend := range bc.Backends {
		urls[i] = backend.URL()
	}
	return urls
}

// Close shuts down all backends in the cluster
func (bc *BackendCluster) Close() {
	for _, backend := range bc.Backends {
		backend.Close()
	}
}

// ResetStats resets the statistics of all backends in the cluster
func (bc *BackendCluster) ResetStats() {
	for _, backend := range bc.Backends {
		backend.ResetStats()
	}
}

// TotalRequests returns the total number of requests processed by all backends
func (bc *BackendCluster) TotalRequests() int {
	total := 0
	for _, backend := range bc.Backends {
		total += int(backend.RequestCount.Load())
	}
	return total
}

// GetBackendRequestCounts returns the request counts for each backend
func (bc *BackendCluster) GetBackendRequestCounts() []int {
	counts := make([]int, len(bc.Backends))
	for i, backend := range bc.Backends {
		counts[i] = int(backend.RequestCount.Load())
	}
	return counts
}

// RequestDistribution returns the percentage distribution of requests across backends
func (bc *BackendCluster) RequestDistribution() []float64 {
	total := bc.TotalRequests()
	if total == 0 {
		return make([]float64, len(bc.Backends))
	}

	distribution := make([]float64, len(bc.Backends))
	for i, backend := range bc.Backends {
		distribution[i] = float64(backend.RequestCount.Load()) / float64(total) * 100
	}
	return distribution
}
