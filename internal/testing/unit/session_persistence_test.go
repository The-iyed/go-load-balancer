package unit

import (
	"net/http"
	"testing"

	"github.com/The-iyed/go-load-balancer/internal/balancer"
	"github.com/The-iyed/go-load-balancer/internal/testing/mocks"
	"github.com/The-iyed/go-load-balancer/internal/testing/testutils"
)

func TestCookiePersistence(t *testing.T) {
	// Create mock backend cluster
	cluster := mocks.NewBackendCluster(3, nil, nil)
	defer cluster.Close()

	// Create load balancer client
	client := mocks.NewLoadBalancerTestClient()
	defer client.Close()

	// Initialize with cookie-based persistence
	err := client.InitializeWithBackends(
		balancer.WeightedRoundRobin,
		balancer.CookiePersistence,
		cluster.URLs(),
		[]int{1, 1, 1},
	)
	if err != nil {
		t.Fatalf("Failed to initialize load balancer: %v", err)
	}

	// Send initial request to get a cookie
	resp, err := client.SendRequest("/", nil)
	if err != nil {
		t.Fatalf("Failed to send initial request: %v", err)
	}

	// Extract backend ID from the response
	initialBackendID, err := testutils.ParseBackendResponse(resp)
	if err != nil {
		t.Fatalf("Failed to parse backend ID: %v", err)
	}

	// Extract cookie from the response
	cookie, found := testutils.CookieFromResponse(resp, "GOLB_SESSION")
	if !found {
		t.Fatalf("Session cookie not found in response")
	}

	// Send multiple requests with the cookie and verify they all go to the same backend
	requestCount := 10
	for i := 0; i < requestCount; i++ {
		resp, err := client.SendRequest("/", []*http.Cookie{cookie})
		if err != nil {
			t.Fatalf("Failed to send request %d: %v", i+1, err)
		}

		backendID, err := testutils.ParseBackendResponse(resp)
		if err != nil {
			t.Fatalf("Failed to parse backend ID in request %d: %v", i+1, err)
		}

		if backendID != initialBackendID {
			t.Errorf("Request %d: Expected backend %d, got backend %d",
				i+1, initialBackendID, backendID)
		}
	}

	// Verify the distribution of requests
	counts := cluster.GetBackendRequestCounts()
	totalRequests := requestCount + 1 // Initial request + subsequent requests

	// The backend selected in the initial request should have received all requests
	for i, count := range counts {
		backendID := i + 1
		if backendID == initialBackendID {
			if count != totalRequests {
				t.Errorf("Backend %d: Expected %d requests, got %d",
					backendID, totalRequests, count)
			}
		} else {
			if count != 0 {
				t.Errorf("Backend %d: Expected 0 requests, got %d",
					backendID, count)
			}
		}
	}
}

func TestIPHashPersistence(t *testing.T) {
	// Create mock backend cluster
	cluster := mocks.NewBackendCluster(3, nil, nil)
	defer cluster.Close()

	// Create load balancer client
	client := mocks.NewLoadBalancerTestClient()
	defer client.Close()

	// Initialize with IP hash persistence
	err := client.InitializeWithBackends(
		balancer.WeightedRoundRobin,
		balancer.IPHashPersistence,
		cluster.URLs(),
		[]int{1, 1, 1},
	)
	if err != nil {
		t.Fatalf("Failed to initialize load balancer: %v", err)
	}

	// Send initial request
	resp, err := client.SendRequest("/", nil)
	if err != nil {
		t.Fatalf("Failed to send initial request: %v", err)
	}

	// Extract backend ID from the response
	initialBackendID, err := testutils.ParseBackendResponse(resp)
	if err != nil {
		t.Fatalf("Failed to parse backend ID: %v", err)
	}

	// Send multiple requests and verify they all go to the same backend
	requestCount := 10
	for i := 0; i < requestCount; i++ {
		resp, err := client.SendRequest("/", nil)
		if err != nil {
			t.Fatalf("Failed to send request %d: %v", i+1, err)
		}

		backendID, err := testutils.ParseBackendResponse(resp)
		if err != nil {
			t.Fatalf("Failed to parse backend ID in request %d: %v", i+1, err)
		}

		if backendID != initialBackendID {
			t.Errorf("Request %d: Expected backend %d, got backend %d",
				i+1, initialBackendID, backendID)
		}
	}

	// Verify the distribution of requests
	counts := cluster.GetBackendRequestCounts()
	totalRequests := requestCount + 1 // Initial request + subsequent requests

	// The backend selected by IP hash should have received all requests
	for i, count := range counts {
		backendID := i + 1
		if backendID == initialBackendID {
			if count != totalRequests {
				t.Errorf("Backend %d: Expected %d requests, got %d",
					backendID, totalRequests, count)
			}
		} else {
			if count != 0 {
				t.Errorf("Backend %d: Expected 0 requests, got %d",
					backendID, count)
			}
		}
	}
}

func TestConsistentHashPersistence(t *testing.T) {
	// Create mock backend cluster
	cluster := mocks.NewBackendCluster(3, nil, nil)
	defer cluster.Close()

	// Create load balancer client
	client := mocks.NewLoadBalancerTestClient()
	defer client.Close()

	// Initialize with consistent hash persistence
	err := client.InitializeWithBackends(
		balancer.WeightedRoundRobin,
		balancer.ConsistentHashPersistence,
		cluster.URLs(),
		[]int{1, 1, 1},
	)
	if err != nil {
		t.Fatalf("Failed to initialize load balancer: %v", err)
	}

	// Test multiple paths and verify consistent routing
	paths := []string{
		"/products",
		"/users",
		"/orders",
		"/cart",
		"/checkout",
	}

	// For each path, send multiple requests and verify consistency
	for _, path := range paths {
		// Reset stats for this path test
		cluster.ResetStats()

		// Send initial request for this path
		resp, err := client.SendRequest(path, nil)
		if err != nil {
			t.Fatalf("Failed to send initial request to %s: %v", path, err)
		}

		initialBackendID, err := testutils.ParseBackendResponse(resp)
		if err != nil {
			t.Fatalf("Failed to parse backend ID for %s: %v", path, err)
		}

		// Send more requests to the same path
		requestCount := 5
		for i := 0; i < requestCount; i++ {
			resp, err := client.SendRequest(path, nil)
			if err != nil {
				t.Fatalf("Failed to send request %d to %s: %v", i+1, path, err)
			}

			backendID, err := testutils.ParseBackendResponse(resp)
			if err != nil {
				t.Fatalf("Failed to parse backend ID in request %d to %s: %v", i+1, path, err)
			}

			if backendID != initialBackendID {
				t.Errorf("Path %s request %d: Expected backend %d, got backend %d",
					path, i+1, initialBackendID, backendID)
			}
		}

		// Verify all requests for this path went to the same backend
		counts := cluster.GetBackendRequestCounts()
		totalPathRequests := requestCount + 1 // Initial request + subsequent requests

		for i, count := range counts {
			backendID := i + 1
			if backendID == initialBackendID {
				if count != totalPathRequests {
					t.Errorf("Path %s Backend %d: Expected %d requests, got %d",
						path, backendID, totalPathRequests, count)
				}
			} else {
				if count != 0 {
					t.Errorf("Path %s Backend %d: Expected 0 requests, got %d",
						path, backendID, count)
				}
			}
		}
	}

	// Verify that different paths can go to different backends
	// by checking if at least 2 backends received requests across all tests
	cluster.ResetStats()

	// Send one request to each path
	for _, path := range paths {
		_, err := client.SendRequest(path, nil)
		if err != nil {
			t.Fatalf("Failed to send request to %s: %v", path, err)
		}
	}

	// Count how many backends received requests
	counts := cluster.GetBackendRequestCounts()
	backendsUsed := 0
	for _, count := range counts {
		if count > 0 {
			backendsUsed++
		}
	}

	// Verify that at least 2 different backends were used
	// (though all 3 could be used depending on the hash function)
	if backendsUsed < 2 {
		t.Errorf("Expected at least 2 backends to be used for different paths, got %d", backendsUsed)
	}
}
