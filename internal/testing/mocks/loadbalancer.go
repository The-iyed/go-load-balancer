package mocks

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/The-iyed/go-load-balancer/internal/balancer"
	"github.com/The-iyed/go-load-balancer/internal/testing/testutils"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

type WSMockBackend struct {
	Server   *httptest.Server
	Requests int32
	Path     string
	ID       string
}

func (m *WSMockBackend) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	atomic.AddInt32(&m.Requests, 1)
	if websocket.IsWebSocketUpgrade(r) {
		m.handleWebSocket(w, r)
		return
	}
	fmt.Fprintf(w, "Backend %s: %s", m.ID, r.URL.Path)
}

func (m *WSMockBackend) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer c.Close()

	// Send greeting
	err = c.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("Hello from backend %s", m.ID)))
	if err != nil {
		return
	}

	// Echo back messages
	for {
		mt, message, err := c.ReadMessage()
		if err != nil {
			break
		}
		message = append([]byte(fmt.Sprintf("Backend %s: ", m.ID)), message...)
		err = c.WriteMessage(mt, message)
		if err != nil {
			break
		}
	}
}

func (m *WSMockBackend) Requests2() int {
	return int(atomic.LoadInt32(&m.Requests))
}

func (m *WSMockBackend) URL() string {
	return m.Server.URL
}

func CreateTestLoadBalancer(numBackends int, algorithm balancer.LoadBalancerAlgorithm, cfg *balancer.Config) *TestLoadBalancer {
	backends := make([]*MockBackend, numBackends)
	backendConfigs := make([]balancer.BackendConfig, numBackends)

	for i := 0; i < numBackends; i++ {
		backends[i] = NewMockBackend(i+1, 0, 0)
		backendConfigs[i] = balancer.BackendConfig{
			URL:    backends[i].URL(),
			Weight: 1,
		}
	}

	// Add PersistenceType if missing in cfg
	persistenceType := balancer.NoPersistence
	if cfg != nil && cfg.PersistenceType != 0 {
		persistenceType = cfg.PersistenceType
	}

	// Create load balancer with the updated signature
	lb, err := balancer.CreateLoadBalancer(algorithm, backendConfigs, persistenceType, cfg.PersistenceAttrs)
	if err != nil {
		panic(fmt.Sprintf("Failed to create load balancer: %v", err))
	}

	return &TestLoadBalancer{
		LoadBalancer: lb,
		Backends:     backends,
		mutex:        &sync.Mutex{},
	}
}

type TestLoadBalancer struct {
	LoadBalancer balancer.LoadBalancerStrategy
	Backends     []*MockBackend
	mutex        *sync.Mutex
}

func (t *TestLoadBalancer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	t.LoadBalancer.ProxyRequest(w, r)
}

func (t *TestLoadBalancer) GetRequestCount(backendIndex int) int {
	if backendIndex < 0 || backendIndex >= len(t.Backends) {
		return 0
	}
	return int(t.Backends[backendIndex].RequestCount.Load())
}

func (t *TestLoadBalancer) GetTotalRequests() int {
	total := 0
	for _, backend := range t.Backends {
		total += int(backend.RequestCount.Load())
	}
	return total
}

func (t *TestLoadBalancer) SendRequest(path string) *http.Response {
	r := httptest.NewRequest("GET", "http://localhost"+path, nil)
	w := httptest.NewRecorder()
	t.LoadBalancer.ProxyRequest(w, r)
	return w.Result()
}

func (t *TestLoadBalancer) GetBackendByPath(path string) *MockBackend {
	return nil // Path-based lookup not supported with real MockBackend
}

func (t *TestLoadBalancer) Shutdown() {
	for _, backend := range t.Backends {
		backend.Close()
	}
	time.Sleep(100 * time.Millisecond) // Give servers time to shut down
}

type LoadBalancerTestClient struct {
	client         *http.Client
	ServerURL      string
	ConfigFilePath string
	LB             balancer.LoadBalancerStrategy
	httpServer     *http.Server
	initialized    bool
	mu             sync.Mutex
}

func NewLoadBalancerTestClient() *LoadBalancerTestClient {
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

func (c *LoadBalancerTestClient) Initialize(config string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.initialized {
		c.Close()
	}

	configPath, err := testutils.CreateTempConfig(config)
	if err != nil {
		return fmt.Errorf("failed to create config file: %v", err)
	}
	c.ConfigFilePath = configPath

	cfg, err := balancer.ParseConfig(configPath)
	if err != nil {
		return fmt.Errorf("failed to parse config: %v", err)
	}

	c.LB, err = balancer.CreateLoadBalancer(cfg.Method, cfg.Backends, cfg.PersistenceType, cfg.PersistenceAttrs)
	if err != nil {
		return fmt.Errorf("failed to create load balancer: %v", err)
	}

	c.httpServer = &http.Server{
		Addr:    ":0",
		Handler: http.HandlerFunc(c.LB.ProxyRequest),
	}

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

func (c *LoadBalancerTestClient) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.httpServer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		c.httpServer.Shutdown(ctx)
	}

	if c.ConfigFilePath != "" {
		_ = os.Remove(c.ConfigFilePath)
	}

	c.initialized = false
}

func (c *LoadBalancerTestClient) SendRequest(path string, cookies []*http.Cookie) (*http.Response, error) {
	if !c.initialized {
		return nil, fmt.Errorf("client not initialized")
	}

	req, err := http.NewRequest("GET", c.ServerURL+path, nil)
	if err != nil {
		return nil, err
	}

	for _, cookie := range cookies {
		req.AddCookie(cookie)
	}

	return c.client.Do(req)
}

func (c *LoadBalancerTestClient) SendRequests(count int, path string, cookies []*http.Cookie) ([]*http.Response, error) {
	responses := make([]*http.Response, 0, count)

	for i := 0; i < count; i++ {
		resp, err := c.SendRequest(path, cookies)
		if err != nil {
			return responses, err
		}
		responses = append(responses, resp)
	}

	return responses, nil
}

func (c *LoadBalancerTestClient) SendConcurrentRequests(count int, path string, cookies []*http.Cookie) ([]*http.Response, error) {
	responses := make([]*http.Response, count)
	errors := make([]error, count)
	wg := sync.WaitGroup{}
	wg.Add(count)

	for i := 0; i < count; i++ {
		go func(idx int) {
			defer wg.Done()
			resp, err := c.SendRequest(path, cookies)
			responses[idx] = resp
			errors[idx] = err
		}(i)
	}

	wg.Wait()

	for _, err := range errors {
		if err != nil {
			return responses, err
		}
	}

	return responses, nil
}
