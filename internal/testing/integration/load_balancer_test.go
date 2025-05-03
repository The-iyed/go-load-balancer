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

func TestLoadBalancerWithRealBinary(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	cluster := mocks.NewBackendCluster(3, nil, nil)
	defer cluster.Close()

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

	projectRoot := testutils.GetProjectRoot()

	binaryPath := filepath.Join(projectRoot, "loadbalancer_test")
	buildCmd := exec.Command("go", "build", "-o", binaryPath, "./cmd/server")
	buildCmd.Dir = projectRoot
	if output, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to build load balancer: %v\n%s", err, output)
	}
	defer os.Remove(binaryPath)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cmd := exec.CommandContext(ctx, binaryPath, "--config", configFile)
	cmd.Env = append(os.Environ(), "PORT=0") // Let the OS assign a port

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		t.Fatalf("Failed to start load balancer: %v", err)
	}

	time.Sleep(2 * time.Second)

	client := &http.Client{
		Timeout: 5 * time.Second,
	}

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

		_, err = io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			t.Fatalf("Failed to read response: %v", err)
		}

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

	if err := cmd.Process.Signal(os.Interrupt); err != nil {
		t.Fatalf("Failed to stop load balancer: %v", err)
	}

	if err := cmd.Wait(); err != nil {
	}

	counts := cluster.GetBackendRequestCounts()
	total := cluster.TotalRequests()

	if total != numRequests {
		t.Errorf("Expected %d total requests, got %d", numRequests, total)
	}

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

func TestLoadBalancerFailover(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	cluster := mocks.NewBackendCluster(3, nil, []float64{1.0, 0, 0})
	defer cluster.Close()

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

	projectRoot := testutils.GetProjectRoot()

	binaryPath := filepath.Join(projectRoot, "loadbalancer_failover_test")
	buildCmd := exec.Command("go", "build", "-o", binaryPath, "./cmd/server")
	buildCmd.Dir = projectRoot
	if output, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to build load balancer: %v\n%s", err, output)
	}
	defer os.Remove(binaryPath)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cmd := exec.CommandContext(ctx, binaryPath, "--config", configFile)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		t.Fatalf("Failed to start load balancer: %v", err)
	}

	time.Sleep(2 * time.Second)

	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	lbURL := "http://localhost:8080"
	req, err := http.NewRequest("GET", lbURL, nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		t.Logf("Initial request failed (expected if routed to failing backend): %v", err)
	} else {
		io.ReadAll(resp.Body)
		resp.Body.Close()
	}

	time.Sleep(5 * time.Second)

	cluster.ResetStats()

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

		_, err = io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			t.Fatalf("Failed to read response: %v", err)
		}

		if sessionCookie == nil {
			for _, cookie := range resp.Cookies() {
				if cookie.Name == "GOLB_SESSION" {
					sessionCookie = cookie
					break
				}
			}
		}
	}

	if err := cmd.Process.Signal(os.Interrupt); err != nil {
		t.Fatalf("Failed to stop load balancer: %v", err)
	}

	if err := cmd.Wait(); err != nil {
	}

	counts := cluster.GetBackendRequestCounts()

	if counts[0] > 3 {
		t.Errorf("Expected failing backend to receive few or no requests, got %d", counts[0])
	}

	if counts[1] == 0 && counts[2] == 0 {
		t.Errorf("Expected healthy backends to receive requests, got %v", counts)
	}

	t.Logf("Request distribution after failover: %v", counts)
}
