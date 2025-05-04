package testutils

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/The-iyed/go-load-balancer/internal/balancer"
)

func GetProjectRoot() string {
	wd, _ := os.Getwd()
	for {
		if _, err := os.Stat(filepath.Join(wd, "go.mod")); err == nil {
			return wd
		}
		wd = filepath.Dir(wd)
	}
}

func CreateTempConfig(content string) (string, error) {
	tmpfile, err := ioutil.TempFile("", "loadbalancer.conf")
	if err != nil {
		return "", err
	}

	if _, err := tmpfile.Write([]byte(content)); err != nil {
		tmpfile.Close()
		os.Remove(tmpfile.Name())
		return "", err
	}
	tmpfile.Close()

	return tmpfile.Name(), nil
}

func CreateTestBackends(count int) ([]string, func(), error) {
	backends := make([]string, count)
	servers := make([]*httptest.Server, count)

	for i := 0; i < count; i++ {
		id := i + 1
		servers[i] = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Backend-ID", fmt.Sprintf("%d", id))
			fmt.Fprintf(w, "Response from backend %d", id)
		}))
		backends[i] = servers[i].URL
	}

	cleanup := func() {
		for _, server := range servers {
			server.Close()
		}
	}

	return backends, cleanup, nil
}

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

	// Convert algorithm enum to string
	var methodStr string
	switch algorithm {
	case balancer.RoundRobin:
		methodStr = "round_robin"
	case balancer.WeightedRoundRobin:
		methodStr = "weighted_round_robin"
	case balancer.LeastConnections:
		methodStr = "least_connections"
	default:
		methodStr = "weighted_round_robin"
	}
	sb.WriteString(fmt.Sprintf("    method %s;\n", methodStr))

	// Convert persistence enum to string
	var persistenceStr string
	switch persistence {
	case balancer.NoPersistence:
		persistenceStr = "none"
	case balancer.CookiePersistence:
		persistenceStr = "cookie"
	case balancer.IPHashPersistence:
		persistenceStr = "ip_hash"
	case balancer.ConsistentHashPersistence:
		persistenceStr = "consistent_hash"
	default:
		persistenceStr = "none"
	}
	sb.WriteString(fmt.Sprintf("    persistence %s;\n", persistenceStr))

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

func CookieFromResponse(resp *http.Response, name string) (*http.Cookie, bool) {
	for _, cookie := range resp.Cookies() {
		if cookie.Name == name {
			return cookie, true
		}
	}
	return nil, false
}

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
