package integration

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/The-iyed/go-load-balancer/internal/testing/mocks"
	"github.com/The-iyed/go-load-balancer/internal/testing/testutils"
)

// TestLoadBalancerWithRealBinary tests the load balancer by running the actual binary
func TestLoadBalancerWithRealBinary(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create mock backend servers
	cluster := mocks.NewBackendCluster(3, nil, nil)
	defer cluster.Close()

	// Create a temporary config file
	config := fmt.Sprintf(`upstream backend {
    method weighted_round_robin;
    persistence cookie;
    server %s weight=3;
    server %s weight=2;
    server %s weight=1;
}`, cluster.Backends[0].URL(), cluster.Backends[1].URL(), cluster.Backends[2].URL())

	configFile, err := testutils.CreateTempConfig(config)
	if err != nil {
		t.Fatalf("Failed to create temp config: %v", err)
	}
	defer os.Remove(configFile)

	// Get project root
	projectRoot := testutils.GetProjectRoot()

	// Build the load balancer binary
	binaryPath := filepath.Join(projectRoot, "loadbalancer_test")
	buildCmd := exec.Command("go", "build", "-o", binaryPath, "./cmd/server")
	buildCmd.Dir = projectRoot
	if output, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to build load balancer: %v\n%s", err, output)
	}
	defer os.Remove(binaryPath)

	// Start the load balancer process
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Use a free port for the load balancer
	cmd := exec.CommandContext(ctx, binaryPath, "--config", configFile)
	cmd.Env = append(os.Environ(), "PORT=0") // Let the OS assign a port

	// Redirect output to test logs for debugging
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		t.Fatalf("Failed to start load balancer: %v", err)
	}

	// Wait for load balancer to start
	time.Sleep(2 * time.Second)

	// Create HTTP client
	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	// Make requests to the load balancer
	const numRequests = 100
	lbURL := "http://localhost:8080"

	var sessionCookie *http.Cookie

	for i := 0; i < numRequests; i++ {
		req, err := http.NewRequest("GET", lbURL, nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		if sessionCookie != nil {
			req.AddCookie(sessionCookie)
		}

		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		// Read and discard response body
		_, err = io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			t.Fatalf("Failed to read response: %v", err)
		}

		// Check if we got a session cookie and save it
		if i == 0 {
			for _, cookie := range resp.Cookies() {
				if cookie.Name == "GOLB_SESSION" {
					sessionCookie = cookie
					break
				}
			}
			if sessionCookie == nil {
				t.Fatalf("No session cookie found in response")
			}
		}
	}

	// Terminate the load balancer
	if err := cmd.Process.Signal(os.Interrupt); err != nil {
		t.Fatalf("Failed to stop load balancer: %v", err)
	}

	// Wait for process to exit
	if err := cmd.Wait(); err != nil {
		// Ignore error since we're sending an interrupt signal
	}

	// Verify the distribution of requests according to weights
	counts := cluster.GetBackendRequestCounts()
	total := cluster.TotalRequests()

	if total != numRequests {
		t.Errorf("Expected %d total requests, got %d", numRequests, total)
	}

	// With cookie persistence, all requests should go to the same backend
	nonZeroBackends := 0
	for _, count := range counts {
		if count > 0 {
			nonZeroBackends++
		}
	}

	if nonZeroBackends != 1 {
		t.Errorf("Expected all requests to go to 1 backend with cookie persistence, but %d backends received requests", nonZeroBackends)
		t.Logf("Request counts: %v", counts)
	}
}

// TestLoadBalancerFailover tests the load balancer's ability to handle backend failures
func TestLoadBalancerFailover(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create mock backend servers, with the primary one having a high failure rate
	// We want the first backend to fail completely after a few requests
	cluster := mocks.NewBackendCluster(3, nil, []float64{1.0, 0, 0})
	defer cluster.Close()

	// Create a temporary config file
	config := fmt.Sprintf(`upstream backend {
    method weighted_round_robin;
    persistence cookie;
    server %s weight=1;
    server %s weight=1;
    server %s weight=1;
}`, cluster.Backends[0].URL(), cluster.Backends[1].URL(), cluster.Backends[2].URL())

	configFile, err := testutils.CreateTempConfig(config)
	if err != nil {
		t.Fatalf("Failed to create temp config: %v", err)
	}
	defer os.Remove(configFile)

	// Get project root
	projectRoot := testutils.GetProjectRoot()

	// Build the load balancer binary
	binaryPath := filepath.Join(projectRoot, "loadbalancer_failover_test")
	buildCmd := exec.Command("go", "build", "-o", binaryPath, "./cmd/server")
	buildCmd.Dir = projectRoot
	if output, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to build load balancer: %v\n%s", err, output)
	}
	defer os.Remove(binaryPath)

	// Start the load balancer process
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cmd := exec.CommandContext(ctx, binaryPath, "--config", configFile)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		t.Fatalf("Failed to start load balancer: %v", err)
	}

	// Wait for load balancer to start
	time.Sleep(2 * time.Second)

	// Create HTTP client
	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	// Make initial request to get a cookie
	lbURL := "http://localhost:8080"
	req, err := http.NewRequest("GET", lbURL, nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		// Initial request might fail if it goes to the failing backend
		// That's actually expected
		t.Logf("Initial request failed (expected if routed to failing backend): %v", err)
	} else {
		io.ReadAll(resp.Body)
		resp.Body.Close()
	}

	// Wait a bit for health check to mark the failing server as dead
	time.Sleep(5 * time.Second)

	// Reset stats to track only the failover requests
	cluster.ResetStats()

	// Send a batch of requests
	const numRequests = 20
	var sessionCookie *http.Cookie

	for i := 0; i < numRequests; i++ {
		req, err := http.NewRequest("GET", lbURL, nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		if sessionCookie != nil {
			req.AddCookie(sessionCookie)
		}

		resp, err := client.Do(req)
		if err != nil {
			t.Logf("Request %d failed: %v", i, err)
			continue
		}

		// Read and discard response body
		_, err = io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			t.Fatalf("Failed to read response: %v", err)
		}

		// Check if we got a session cookie and save it
		if sessionCookie == nil {
			for _, cookie := range resp.Cookies() {
				if cookie.Name == "GOLB_SESSION" {
					sessionCookie = cookie
					break
				}
			}
		}
	}

	// Terminate the load balancer
	if err := cmd.Process.Signal(os.Interrupt); err != nil {
		t.Fatalf("Failed to stop load balancer: %v", err)
	}

	// Wait for process to exit
	if err := cmd.Wait(); err != nil {
		// Ignore error since we're sending an interrupt signal
	}

	// Verify the distribution of requests after failover
	counts := cluster.GetBackendRequestCounts()

	// First backend should have received zero or very few requests since it's failing
	if counts[0] > 3 {
		t.Errorf("Expected failing backend to receive few or no requests, got %d", counts[0])
	}

	// The other backends should have received requests
	if counts[1] == 0 && counts[2] == 0 {
		t.Errorf("Expected healthy backends to receive requests, got %v", counts)
	}

	t.Logf("Request distribution after failover: %v", counts)
}
