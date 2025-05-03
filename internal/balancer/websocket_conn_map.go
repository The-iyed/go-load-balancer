package balancer

import (
	"crypto/rand"
	"encoding/hex"
	"sync"

	"github.com/gorilla/websocket"
)

type WebSocketConnection struct {
	ClientConn  *websocket.Conn
	BackendConn *websocket.Conn
}

type WebSocketConnectionMap struct {
	connections map[string]*WebSocketConnection
	mu          sync.RWMutex
}

func NewWebSocketConnectionMap() *WebSocketConnectionMap {
	return &WebSocketConnectionMap{
		connections: make(map[string]*WebSocketConnection),
	}
}

func (cm *WebSocketConnectionMap) Add(clientConn, backendConn *websocket.Conn) string {
	connID := generateConnID()

	cm.mu.Lock()
	defer cm.mu.Unlock()

	cm.connections[connID] = &WebSocketConnection{
		ClientConn:  clientConn,
		BackendConn: backendConn,
	}

	return connID
}

func (cm *WebSocketConnectionMap) Get(connID string) (*WebSocketConnection, bool) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	conn, exists := cm.connections[connID]
	return conn, exists
}

func (cm *WebSocketConnectionMap) Remove(connID string) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	delete(cm.connections, connID)
}

func (cm *WebSocketConnectionMap) Count() int {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	return len(cm.connections)
}

func generateConnID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}
