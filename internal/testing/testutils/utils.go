package testutils

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/The-iyed/go-load-balancer/internal/balancer"
)

// GetProjectRoot returns the absolute path to the project root directory
func GetProjectRoot() string {
	_, filename, _, _ := runtime.Caller(0)
	dir := filepath.Dir(filename)
	return filepath.Join(dir, "..", "..", "..")
}

// CreateTempConfig creates a temporary config file with the given content
// and returns the path to the file. The caller is responsible for removing the file.
func CreateTempConfig(content string) (string, error) {
	tmpFile, err := os.CreateTemp("", "loadbalancer-test-*.conf")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %v", err)
	}
	defer tmpFile.Close()

	if _, err := tmpFile.Write([]byte(content)); err != nil {
		return "", fmt.Errorf("failed to write to temp file: %v", err)
	}

	return tmpFile.Name(), nil
}

// CreateTestBackends creates a specified number of test HTTP servers
// and returns their URLs and a cleanup function.
func CreateTestBackends(count int) ([]string, func(), error) {
	var backends []string
	var servers []*httptest.Server

	cleanup := func() {
		for _, server := range servers {
			server.Close()
		}
	}

	for i := 0; i < count; i++ {
		backendID := i + 1
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Backend-ID", fmt.Sprintf("%d", backendID))
			fmt.Fprintf(w, "Response from backend %d", backendID)
		}))
		backends = append(backends, srv.URL)
		servers = append(servers, srv)
	}

	return backends, cleanup, nil
}

// CreateLoadBalancerConfig creates a load balancer configuration with the specified
// algorithm, persistence method, and backends.
func CreateLoadBalancerConfig(algorithm balancer.LoadBalancerAlgorithm,
	persistence balancer.PersistenceMethod, backends []string, weights []int) (string, error) {

	if len(backends) == 0 {
		return "", fmt.Errorf("at least one backend is required")
	}

	if weights != nil && len(weights) != len(backends) {
		return "", fmt.Errorf("if weights are provided, they must match the number of backends")
	}

	var sb strings.Builder

	sb.WriteString("upstream backend {\n")
	sb.WriteString(fmt.Sprintf("    method %s;\n", algorithm))
	sb.WriteString(fmt.Sprintf("    persistence %s;\n", persistence))

	for i, backend := range backends {
		weight := 1
		if weights != nil {
			weight = weights[i]
		}
		sb.WriteString(fmt.Sprintf("    server %s weight=%d;\n", backend, weight))
	}

	sb.WriteString("}\n")

	return sb.String(), nil
}

// AssertEventually repeatedly checks the given condition until it returns true or the timeout is reached.
func AssertEventually(t *testing.T, condition func() bool, timeout time.Duration, message string) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}

	t.Fatalf("Condition not met within %v: %s", timeout, message)
}

// CookieFromResponse extracts a cookie with the given name from an HTTP response.
func CookieFromResponse(resp *http.Response, name string) (*http.Cookie, bool) {
	for _, cookie := range resp.Cookies() {
		if cookie.Name == name {
			return cookie, true
		}
	}
	return nil, false
}

// ParseBackendResponse parses the backend ID from a response.
func ParseBackendResponse(resp *http.Response) (int, error) {
	backendID := resp.Header.Get("X-Backend-ID")
	if backendID == "" {
		return 0, fmt.Errorf("X-Backend-ID header not found in response")
	}

	var id int
	_, err := fmt.Sscanf(backendID, "%d", &id)
	if err != nil {
		return 0, fmt.Errorf("failed to parse backend ID: %v", err)
	}

	return id, nil
}
