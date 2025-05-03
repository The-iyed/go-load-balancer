package unit

import (
	"net/http"
	"testing"

	"github.com/The-iyed/go-load-balancer/internal/balancer"
	"github.com/The-iyed/go-load-balancer/internal/testing/mocks"
	"github.com/The-iyed/go-load-balancer/internal/testing/testutils"
)

func TestCookiePersistence(t *testing.T) {
	cluster := mocks.NewBackendCluster(3, nil, nil)
	defer cluster.Close()

	client := mocks.NewLoadBalancerTestClient()
	defer client.Close()

	err := client.InitializeWithBackends(
		balancer.WeightedRoundRobin,
		balancer.CookiePersistence,
		cluster.URLs(),
		[]int{1, 1, 1},
	)
	if err != nil {
		t.Fatalf("Failed to initialize load balancer: %v", err)
	}

	resp, err := client.SendRequest("/", nil)
	if err != nil {
		t.Fatalf("Failed to send initial request: %v", err)
	}

	initialBackendID, err := testutils.ParseBackendResponse(resp)
	if err != nil {
		t.Fatalf("Failed to parse backend ID: %v", err)
	}

	cookie, found := testutils.CookieFromResponse(resp, "GOLB_SESSION")
	if !found {
		t.Fatalf("Session cookie not found in response")
	}

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

	counts := cluster.GetBackendRequestCounts()
	totalRequests := requestCount + 1

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
	cluster := mocks.NewBackendCluster(3, nil, nil)
	defer cluster.Close()

	client := mocks.NewLoadBalancerTestClient()
	defer client.Close()

	err := client.InitializeWithBackends(
		balancer.WeightedRoundRobin,
		balancer.IPHashPersistence,
		cluster.URLs(),
		[]int{1, 1, 1},
	)
	if err != nil {
		t.Fatalf("Failed to initialize load balancer: %v", err)
	}

	resp, err := client.SendRequest("/", nil)
	if err != nil {
		t.Fatalf("Failed to send initial request: %v", err)
	}

	initialBackendID, err := testutils.ParseBackendResponse(resp)
	if err != nil {
		t.Fatalf("Failed to parse backend ID: %v", err)
	}

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

	counts := cluster.GetBackendRequestCounts()
	totalRequests := requestCount + 1

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
	cluster := mocks.NewBackendCluster(3, nil, nil)
	defer cluster.Close()

	client := mocks.NewLoadBalancerTestClient()
	defer client.Close()

	err := client.InitializeWithBackends(
		balancer.WeightedRoundRobin,
		balancer.ConsistentHashPersistence,
		cluster.URLs(),
		[]int{1, 1, 1},
	)
	if err != nil {
		t.Fatalf("Failed to initialize load balancer: %v", err)
	}

	paths := []string{
		"/products",
		"/users",
		"/orders",
		"/cart",
		"/checkout",
	}

	for _, path := range paths {
		cluster.ResetStats()

		resp, err := client.SendRequest(path, nil)
		if err != nil {
			t.Fatalf("Failed to send initial request to %s: %v", path, err)
		}

		initialBackendID, err := testutils.ParseBackendResponse(resp)
		if err != nil {
			t.Fatalf("Failed to parse backend ID for %s: %v", path, err)
		}

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

		counts := cluster.GetBackendRequestCounts()
		totalPathRequests := requestCount + 1

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

	cluster.ResetStats()

	for _, path := range paths {
		_, err := client.SendRequest(path, nil)
		if err != nil {
			t.Fatalf("Failed to send request to %s: %v", path, err)
		}
	}

	counts := cluster.GetBackendRequestCounts()
	backendsUsed := 0
	for _, count := range counts {
		if count > 0 {
			backendsUsed++
		}
	}

	if backendsUsed < 2 {
		t.Errorf("Expected at least 2 backends to be used for different paths, got %d", backendsUsed)
	}
}
