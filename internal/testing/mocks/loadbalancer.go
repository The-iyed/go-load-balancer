package mocks

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/The-iyed/go-load-balancer/internal/balancer"
	"github.com/The-iyed/go-load-balancer/internal/testing/testutils"
)

// LoadBalancerTestClient provides a client for testing the load balancer
type LoadBalancerTestClient struct {
	client         *http.Client
	ServerURL      string
	ConfigFilePath string
	LB             balancer.LoadBalancerStrategy
	httpServer     *http.Server
	initialized    bool
	mu             sync.Mutex
}

// NewLoadBalancerTestClient creates a new test client for the load balancer
func NewLoadBalancerTestClient() *LoadBalancerTestClient {
	// Create an HTTP client with a shorter timeout
	client := &http.Client{
		Timeout: 5 * time.Second,
		Transport: &http.Transport{
			DialContext: (&net.Dialer{
				Timeout:   1 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 100,
			IdleConnTimeout:     90 * time.Second,
		},
	}

	return &LoadBalancerTestClient{
		client: client,
	}
}

// Initialize sets up the load balancer with the provided configuration
func (c *LoadBalancerTestClient) Initialize(config string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Clean up any existing resources
	if c.initialized {
		c.Close()
	}

	// Create a temporary config file
	configPath, err := testutils.CreateTempConfig(config)
	if err != nil {
		return fmt.Errorf("failed to create config file: %v", err)
	}
	c.ConfigFilePath = configPath

	// Parse the configuration
	cfg, err := balancer.ParseConfig(configPath)
	if err != nil {
		return fmt.Errorf("failed to parse config: %v", err)
	}

	// Create the load balancer
	c.LB = balancer.CreateLoadBalancer(cfg.Method, cfg.Backends, cfg.Persistence)

	// Set up HTTP server
	c.httpServer = &http.Server{
		Addr:    ":0", // Let the OS assign a free port
		Handler: http.HandlerFunc(c.LB.ProxyRequest),
	}

	// Start the server
	listener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		return fmt.Errorf("failed to create listener: %v", err)
	}

	c.ServerURL = fmt.Sprintf("http://%s", listener.Addr().String())

	go func() {
		if err := c.httpServer.Serve(listener); err != nil && err != http.ErrServerClosed {
			fmt.Printf("HTTP server error: %v\n", err)
		}
	}()

	c.initialized = true
	return nil
}

// InitializeWithBackends sets up the load balancer with mock backends
func (c *LoadBalancerTestClient) InitializeWithBackends(
	algorithm balancer.LoadBalancerAlgorithm,
	persistence balancer.PersistenceMethod,
	backends []string,
	weights []int) error {

	config, err := testutils.CreateLoadBalancerConfig(algorithm, persistence, backends, weights)
	if err != nil {
		return fmt.Errorf("failed to create config: %v", err)
	}

	return c.Initialize(config)
}

// Close cleans up resources used by the test client
func (c *LoadBalancerTestClient) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.httpServer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		c.httpServer.Shutdown(ctx)
	}

	if c.ConfigFilePath != "" {
		// Delete the temporary config file
		_ = os.Remove(c.ConfigFilePath)
	}

	c.initialized = false
}

// SendRequest sends an HTTP request to the load balancer
func (c *LoadBalancerTestClient) SendRequest(path string, cookies []*http.Cookie) (*http.Response, error) {
	if !c.initialized {
		return nil, fmt.Errorf("client not initialized")
	}

	url := c.ServerURL + path
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	// Add cookies if provided
	for _, cookie := range cookies {
		req.AddCookie(cookie)
	}

	return c.client.Do(req)
}

// SendRequests sends multiple HTTP requests to the load balancer
func (c *LoadBalancerTestClient) SendRequests(count int, path string, cookies []*http.Cookie) ([]*http.Response, error) {
	responses := make([]*http.Response, count)

	for i := 0; i < count; i++ {
		resp, err := c.SendRequest(path, cookies)
		if err != nil {
			return responses[:i], fmt.Errorf("request %d failed: %v", i+1, err)
		}
		responses[i] = resp
	}

	return responses, nil
}

// SendConcurrentRequests sends multiple HTTP requests concurrently
func (c *LoadBalancerTestClient) SendConcurrentRequests(count int, path string, cookies []*http.Cookie) ([]*http.Response, error) {
	responses := make([]*http.Response, count)
	var wg sync.WaitGroup
	errCh := make(chan error, count)

	for i := 0; i < count; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			resp, err := c.SendRequest(path, cookies)
			if err != nil {
				errCh <- fmt.Errorf("request %d failed: %v", idx+1, err)
				return
			}
			responses[idx] = resp
		}(i)
	}

	wg.Wait()
	close(errCh)

	if len(errCh) > 0 {
		var errs bytes.Buffer
		for err := range errCh {
			errs.WriteString(err.Error() + "\n")
		}
		return responses, fmt.Errorf("errors occurred during concurrent requests:\n%s", errs.String())
	}

	return responses, nil
}
