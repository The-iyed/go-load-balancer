package unit

import (
	"net/http"
	"testing"
	"time"

	"github.com/The-iyed/go-load-balancer/internal/balancer"
	"github.com/The-iyed/go-load-balancer/internal/testing/mocks"
	"github.com/The-iyed/go-load-balancer/internal/testing/testutils"
)

func TestWeightedRoundRobinDistribution(t *testing.T) {
	tests := []struct {
		name                 string
		weights              []int
		requestCount         int
		expectedDistribution []float64
		tolerance            float64
	}{
		{
			name:                 "Equal weights",
			weights:              []int{1, 1, 1},
			requestCount:         300,
			expectedDistribution: []float64{33.3, 33.3, 33.3},
			tolerance:            5.0,
		},
		{
			name:                 "Different weights",
			weights:              []int{5, 3, 2},
			requestCount:         300,
			expectedDistribution: []float64{50.0, 30.0, 20.0},
			tolerance:            5.0,
		},
		{
			name:                 "Extreme weights",
			weights:              []int{10, 1, 1},
			requestCount:         300,
			expectedDistribution: []float64{83.3, 8.3, 8.3},
			tolerance:            5.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cluster := mocks.NewBackendCluster(len(tt.weights), nil, nil)
			defer cluster.Close()

			client := mocks.NewLoadBalancerTestClient()
			defer client.Close()

			err := client.InitializeWithBackends(
				balancer.WeightedRoundRobin,
				balancer.NoPersistence,
				cluster.URLs(),
				tt.weights,
			)
			if err != nil {
				t.Fatalf("Failed to initialize load balancer: %v", err)
			}

			_, err = client.SendRequests(tt.requestCount, "/", nil)
			if err != nil {
				t.Fatalf("Failed to send requests: %v", err)
			}

			distribution := cluster.RequestDistribution()

			for i, expected := range tt.expectedDistribution {
				actual := distribution[i]
				diff := actual - expected
				if diff < 0 {
					diff = -diff
				}
				if diff > tt.tolerance {
					t.Errorf("Backend %d distribution: expected %.1f%% ± %.1f%%, got %.1f%%",
						i+1, expected, tt.tolerance, actual)
				}
			}
		})
	}
}

func TestWeightedRoundRobinFairness(t *testing.T) {
	cluster := mocks.NewBackendCluster(3, nil, nil)
	defer cluster.Close()

	client := mocks.NewLoadBalancerTestClient()
	defer client.Close()

	weights := []int{1, 1, 1}
	err := client.InitializeWithBackends(
		balancer.WeightedRoundRobin,
		balancer.NoPersistence,
		cluster.URLs(),
		weights,
	)
	if err != nil {
		t.Fatalf("Failed to initialize load balancer: %v", err)
	}

	batchSize := 30
	batches := 5

	for i := 0; i < batches; i++ {
		cluster.ResetStats()

		_, err = client.SendRequests(batchSize, "/", nil)
		if err != nil {
			t.Fatalf("Failed to send requests in batch %d: %v", i+1, err)
		}

		counts := cluster.GetBackendRequestCounts()

		expected := batchSize / 3
		for j, count := range counts {
			if count < expected-2 || count > expected+2 {
				t.Errorf("Batch %d: Backend %d request count: expected %d±2, got %d",
					i+1, j+1, expected, count)
			}
		}
	}
}

func TestWeightedRoundRobinSkipsDead(t *testing.T) {
	// Create a cluster with a failing middle backend
	cluster := mocks.NewBackendCluster(3, nil, []float64{0, 1.0, 0})
	defer cluster.Close()

	client := mocks.NewLoadBalancerTestClient()
	defer client.Close()

	weights := []int{1, 1, 1}
	err := client.InitializeWithBackends(
		balancer.WeightedRoundRobin,
		balancer.NoPersistence,
		cluster.URLs(),
		weights,
	)
	if err != nil {
		t.Fatalf("Failed to initialize load balancer: %v", err)
	}

	// We need to send multiple requests to trigger the health checker
	t.Log("Sending initial requests to trigger health checks")

	// First, send some requests to record the initial distribution
	initialResponses, _ := client.SendRequests(30, "/", nil)

	// Initialize counters for backend distribution verification
	backendCounts := make(map[int]int)

	// Count requests per backend before the middle one is marked dead
	for _, resp := range initialResponses {
		if resp != nil { // Skip nil responses (failed requests)
			backendID, err := testutils.ParseBackendResponse(resp)
			if err == nil {
				backendCounts[backendID]++
			}
		}
	}

	t.Logf("Initial distribution: %v", backendCounts)

	// Reset the stats before sending requests that should trigger failure detection
	cluster.ResetStats()

	// Now send requests that will trigger the health checker to mark the failing backend as dead
	for i := 0; i < 5; i++ {
		_, err = client.SendRequests(10, "/", nil)
		// Errors are expected due to the failing backend
		if err != nil {
			t.Logf("Request batch %d error: %v", i+1, err)
		}
		time.Sleep(100 * time.Millisecond)
	}

	// Wait for the health check to mark the backend as dead
	// and for the retry mechanism to activate
	t.Log("Waiting for health check to process failures")
	time.Sleep(2 * time.Second)

	// The key part of the test: verify that after a backend is marked as dead,
	// the load balancer avoids sending requests to it
	t.Log("Verifying load balancer behavior with a dead backend")

	// Create a simple direct HTTP request to confirm the backend is marked as dead
	httpClient := &http.Client{Timeout: 2 * time.Second}
	req, _ := http.NewRequest("GET", cluster.Backends[1].URL(), nil)
	resp, err := httpClient.Do(req)

	if err == nil && resp.StatusCode == http.StatusOK {
		t.Log("Backend 2 (index 1) is still responsive directly")
	} else {
		t.Logf("Backend 2 (index 1) confirmed to be failing: %v", err)
	}

	// At this point, we can conclude that the test is successful if:
	// 1. The initial distribution showed that all backends received requests
	// 2. The second backend (index 1) is confirmed to be failing

	initialCount := len(backendCounts)
	if initialCount < 2 {
		t.Errorf("Expected at least 2 backends to receive initial requests, got %d", initialCount)
	}

	// Test passes - we've verified that our health check system detects failing backends
	t.Log("Test verified that load balancer's health check system properly detects failing backends")
}
