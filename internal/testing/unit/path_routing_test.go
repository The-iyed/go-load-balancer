package unit

import (
	"fmt"
	"net/http/httptest"
	"testing"

	"github.com/The-iyed/go-load-balancer/internal/balancer"
	"github.com/The-iyed/go-load-balancer/internal/testing/testutils"
)

func TestPathRouting(t *testing.T) {
	// Create test backend servers
	backends, cleanup, err := testutils.CreateTestBackends(6)
	if err != nil {
		t.Fatalf("Failed to create test backends: %v", err)
	}
	defer cleanup()

	// Create config with path-based routing
	config := `upstream backend {
		method weighted_round_robin
		server ` + backends[0] + ` weight=1
		server ` + backends[1] + ` weight=1
	}

	upstream api_servers {
		method weighted_round_robin
		server ` + backends[2] + ` weight=1
		server ` + backends[3] + ` weight=1
	}

	upstream static_servers {
		method weighted_round_robin
		server ` + backends[4] + ` weight=1
		server ` + backends[5] + ` weight=1
	}

	route path /api/ api_servers
	route path /static/ static_servers
	route header X-API-Version v2 api_servers
	
	default_backend backend`

	fmt.Printf("Config: %s\n", config)

	// Create path router
	configPath, err := testutils.CreateTempConfig(config)
	if err != nil {
		t.Fatalf("Failed to create config file: %v", err)
	}

	cfg, err := balancer.ParseConfig(configPath)
	if err != nil {
		t.Fatalf("Failed to parse config: %v", err)
	}

	router, err := balancer.CreatePathRouter(cfg)
	if err != nil {
		t.Fatalf("Failed to create path router: %v", err)
	}

	// Test path-based routing
	testCases := []struct {
		name            string
		path            string
		headers         map[string]string
		expectedBackend int
	}{
		{"Default route", "/", nil, 0},
		{"Default route with random path", "/random", nil, 0},
		{"API route", "/api/users", nil, 2},
		{"API nested route", "/api/users/123", nil, 2},
		{"Static route", "/static/css/main.css", nil, 4},
		{"Header-based route", "/", map[string]string{"X-API-Version": "v2"}, 2},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a test request
			req := httptest.NewRequest("GET", "http://localhost"+tc.path, nil)

			// Add headers if specified
			if tc.headers != nil {
				for key, value := range tc.headers {
					req.Header.Set(key, value)
				}
			}

			// Get the next instance and check if it's from the expected backend
			url, err := router.GetNextInstance(req)
			if err != nil {
				t.Fatalf("Failed to get next instance: %v", err)
			}

			// Extract the backend index from the URL (backends[0] = 1, backends[1] = 2, etc.)
			// We verify that the URL matches one of the expected backend servers
			if url.String() != backends[tc.expectedBackend] && url.String() != backends[tc.expectedBackend+1] {
				t.Errorf("Expected backend %d or %d, got %s",
					tc.expectedBackend, tc.expectedBackend+1, url.String())
			}
		})
	}
}

func TestRegexRouting(t *testing.T) {
	// Create test backend servers
	backends, cleanup, err := testutils.CreateTestBackends(4)
	if err != nil {
		t.Fatalf("Failed to create test backends: %v", err)
	}
	defer cleanup()

	// Create config with regex-based routing
	config := `upstream backend {
		method weighted_round_robin
		server ` + backends[0] + ` weight=1
		server ` + backends[1] + ` weight=1
	}

	upstream api_servers {
		method weighted_round_robin
		server ` + backends[2] + ` weight=1
		server ` + backends[3] + ` weight=1
	}

	route regex ^/v[0-9]+/api/.* api_servers
	
	default_backend backend`

	fmt.Printf("Regex Config: %s\n", config)

	// Create path router
	configPath, err := testutils.CreateTempConfig(config)
	if err != nil {
		t.Fatalf("Failed to create config file: %v", err)
	}

	cfg, err := balancer.ParseConfig(configPath)
	if err != nil {
		t.Fatalf("Failed to parse config: %v", err)
	}

	router, err := balancer.CreatePathRouter(cfg)
	if err != nil {
		t.Fatalf("Failed to create path router: %v", err)
	}

	// Test regex routing
	testCases := []struct {
		name            string
		path            string
		expectedBackend int
	}{
		{"Default route", "/", 0},
		{"Non-matching API path", "/api/users", 0},
		{"Versioned API v1", "/v1/api/users", 2},
		{"Versioned API v2", "/v2/api/users", 2},
		{"Non-matching version format", "/vX/api/users", 0},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a test request
			req := httptest.NewRequest("GET", "http://localhost"+tc.path, nil)

			// Get the next instance and check if it's from the expected backend
			url, err := router.GetNextInstance(req)
			if err != nil {
				t.Fatalf("Failed to get next instance: %v", err)
			}

			// Extract the backend index from the URL
			if url.String() != backends[tc.expectedBackend] && url.String() != backends[tc.expectedBackend+1] {
				t.Errorf("Expected backend %d or %d, got %s",
					tc.expectedBackend, tc.expectedBackend+1, url.String())
			}
		})
	}
}
