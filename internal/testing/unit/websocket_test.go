package unit

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/The-iyed/go-load-balancer/internal/balancer"
	"github.com/The-iyed/go-load-balancer/internal/logger"
	"github.com/gorilla/websocket"
)

func init() {
	logger.InitLogger()
}

func TestWebSocketDetection(t *testing.T) {
	tests := []struct {
		name     string
		headers  map[string]string
		method   string
		expected bool
	}{
		{
			name: "Valid WebSocket Request",
			headers: map[string]string{
				"Upgrade":    "websocket",
				"Connection": "Upgrade",
			},
			method:   "GET",
			expected: true,
		},
		{
			name: "Missing Upgrade Header",
			headers: map[string]string{
				"Connection": "Upgrade",
			},
			method:   "GET",
			expected: false,
		},
		{
			name: "Missing Connection Header",
			headers: map[string]string{
				"Upgrade": "websocket",
			},
			method:   "GET",
			expected: false,
		},
		{
			name: "Wrong Method",
			headers: map[string]string{
				"Upgrade":    "websocket",
				"Connection": "Upgrade",
			},
			method:   "POST",
			expected: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, "http://example.com/ws", nil)
			for k, v := range tc.headers {
				req.Header.Set(k, v)
			}

			if got := balancer.IsWebSocketRequest(req); got != tc.expected {
				t.Errorf("IsWebSocketRequest() = %v, want %v", got, tc.expected)
			}
		})
	}
}

func TestWebSocketConnectionMap(t *testing.T) {
	connMap := balancer.NewWebSocketConnectionMap()

	if count := connMap.Count(); count != 0 {
		t.Errorf("Initial count should be 0, got %d", count)
	}

	// Create mock connections
	clientConn := &websocket.Conn{}
	backendConn := &websocket.Conn{}

	// Add connection
	connID := connMap.Add(clientConn, backendConn)
	if connID == "" {
		t.Error("Generated connection ID should not be empty")
	}

	if count := connMap.Count(); count != 1 {
		t.Errorf("Count should be 1 after adding, got %d", count)
	}

	// Get connection
	conn, exists := connMap.Get(connID)
	if !exists {
		t.Error("Connection should exist after adding")
	}
	if conn.ClientConn != clientConn || conn.BackendConn != backendConn {
		t.Error("Retrieved connection doesn't match added connections")
	}

	// Remove connection
	connMap.Remove(connID)
	if count := connMap.Count(); count != 0 {
		t.Errorf("Count should be 0 after removal, got %d", count)
	}

	_, exists = connMap.Get(connID)
	if exists {
		t.Error("Connection should not exist after removal")
	}
}

func TestWebSocketProxy_Integration(t *testing.T) {
	t.Skip("Integration test requires a real WebSocket server")

	// Setup Echo WebSocket Server
	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin:     func(r *http.Request) bool { return true },
	}

	echoServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer c.Close()

		for {
			mt, message, err := c.ReadMessage()
			if err != nil {
				break
			}

			// Echo the message back
			err = c.WriteMessage(mt, message)
			if err != nil {
				break
			}
		}
	}))
	defer echoServer.Close()

	wsURL := "ws" + strings.TrimPrefix(echoServer.URL, "http")

	// Create process for the backend
	url, _ := balancer.ParseURL(wsURL)
	process := &balancer.Process{
		URL:   url,
		Alive: true,
	}

	// Create WebSocket proxy
	errorHandlerCalled := false
	proxy := balancer.NewWebSocketProxy(process, func(p *balancer.Process) {
		errorHandlerCalled = true
	})

	// Setup proxy server
	proxyServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if balancer.IsWebSocketRequest(r) {
			proxy.ProxyWebSocket(w, r)
			return
		}
		http.Error(w, "Not a WebSocket request", http.StatusBadRequest)
	}))
	defer proxyServer.Close()

	// Connect to proxy server
	wsProxyURL := "ws" + strings.TrimPrefix(proxyServer.URL, "http")
	clientConn, _, err := websocket.DefaultDialer.Dial(wsProxyURL, nil)
	if err != nil {
		t.Fatalf("Failed to connect to proxy: %v", err)
	}
	defer clientConn.Close()

	// Send a message through the proxy
	testMessage := "Hello WebSocket!"
	if err := clientConn.WriteMessage(websocket.TextMessage, []byte(testMessage)); err != nil {
		t.Fatalf("Failed to write message: %v", err)
	}

	// Wait for response
	clientConn.SetReadDeadline(time.Now().Add(time.Second))
	messageType, message, err := clientConn.ReadMessage()
	if err != nil {
		t.Fatalf("Failed to read message: %v", err)
	}

	if messageType != websocket.TextMessage {
		t.Errorf("Expected message type %d, got %d", websocket.TextMessage, messageType)
	}

	if string(message) != testMessage {
		t.Errorf("Expected message %q, got %q", testMessage, string(message))
	}

	// Verify that error handler wasn't called
	if errorHandlerCalled {
		t.Error("Error handler shouldn't have been called")
	}
}
