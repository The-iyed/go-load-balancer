package unit

import (
	"testing"
	"time"

	"github.com/The-iyed/go-load-balancer/internal/balancer"
	"github.com/The-iyed/go-load-balancer/internal/testing/mocks"
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
			// Create mock backend cluster
			cluster := mocks.NewBackendCluster(len(tt.weights), nil, nil)
			defer cluster.Close()

			// Create load balancer client
			client := mocks.NewLoadBalancerTestClient()
			defer client.Close()

			// Initialize with weighted round robin algorithm
			err := client.InitializeWithBackends(
				balancer.WeightedRoundRobin,
				balancer.NoPersistence,
				cluster.URLs(),
				tt.weights,
			)
			if err != nil {
				t.Fatalf("Failed to initialize load balancer: %v", err)
			}

			// Send requests
			_, err = client.SendRequests(tt.requestCount, "/", nil)
			if err != nil {
				t.Fatalf("Failed to send requests: %v", err)
			}

			// Get actual distribution
			distribution := cluster.RequestDistribution()

			// Verify distribution is within tolerance
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
	// Create mock backend cluster with 3 servers
	cluster := mocks.NewBackendCluster(3, nil, nil)
	defer cluster.Close()

	// Create load balancer client
	client := mocks.NewLoadBalancerTestClient()
	defer client.Close()

	// Initialize with weighted round robin and equal weights
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

	// Send small batches of requests and check if distribution is fair
	// for each small batch
	batchSize := 30 // 10 requests per backend expected
	batches := 5

	for i := 0; i < batches; i++ {
		// Reset stats between batches
		cluster.ResetStats()

		// Send a batch of requests
		_, err = client.SendRequests(batchSize, "/", nil)
		if err != nil {
			t.Fatalf("Failed to send requests in batch %d: %v", i+1, err)
		}

		// Get request counts
		counts := cluster.GetBackendRequestCounts()

		// Check if each backend received approximately batchSize/3 requests
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
	// Create mock backend cluster with 3 servers
	// The second server will fail 100% of the time
	cluster := mocks.NewBackendCluster(3, nil, []float64{0, 1.0, 0})
	defer cluster.Close()

	// Create load balancer client
	client := mocks.NewLoadBalancerTestClient()
	defer client.Close()

	// Initialize with weighted round robin and equal weights
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

	// Send requests
	requestCount := 60
	_, err = client.SendRequests(requestCount, "/", nil)
	if err != nil {
		t.Fatalf("Failed to send requests: %v", err)
	}

	// Wait a bit for health check to mark the failing server as dead
	time.Sleep(2 * time.Second)

	// Reset stats
	cluster.ResetStats()

	// Send more requests
	_, err = client.SendRequests(requestCount, "/", nil)
	if err != nil {
		t.Fatalf("Failed to send second batch of requests: %v", err)
	}

	// Get request counts
	counts := cluster.GetBackendRequestCounts()

	// Check that the second backend received no requests (it's marked as dead)
	if counts[1] > 0 {
		t.Errorf("Expected dead backend to receive 0 requests, got %d", counts[1])
	}

	// Check that the other backends shared the load roughly equally
	expected := requestCount / 2
	for i, count := range []int{counts[0], counts[2]} {
		backendID := i*2 + 1 // 1 or 3
		if count < expected-5 || count > expected+5 {
			t.Errorf("Backend %d request count: expected %d±5, got %d",
				backendID, expected, count)
		}
	}
}
