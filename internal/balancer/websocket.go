package balancer

import (
	"net/http"
	"sync/atomic"
	"time"

	"github.com/The-iyed/go-load-balancer/internal/logger"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

type WebSocketProxy struct {
	backend        *Process
	upgrader       websocket.Upgrader
	dialer         *websocket.Dialer
	connMap        *WebSocketConnectionMap
	errorHandler   func(backend *Process)
	connectionTTL  time.Duration
	pingInterval   time.Duration
	pongWait       time.Duration
	writeWait      time.Duration
	maxMessageSize int64
}

func NewWebSocketProxy(backend *Process, errorHandler func(backend *Process)) *WebSocketProxy {
	return &WebSocketProxy{
		backend: backend,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin:     func(r *http.Request) bool { return true },
		},
		dialer: &websocket.Dialer{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			Proxy:           http.ProxyFromEnvironment,
		},
		connMap:        NewWebSocketConnectionMap(),
		errorHandler:   errorHandler,
		connectionTTL:  3 * time.Hour,
		pingInterval:   30 * time.Second,
		pongWait:       60 * time.Second,
		writeWait:      10 * time.Second,
		maxMessageSize: 1024 * 1024, // 1MB
	}
}

func (wp *WebSocketProxy) ProxyWebSocket(w http.ResponseWriter, r *http.Request) {
	clientConn, err := wp.upgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.Log.Error("Failed to upgrade client connection", zap.Error(err))
		return
	}

	clientConn.SetReadLimit(wp.maxMessageSize)
	clientConn.SetPongHandler(func(string) error {
		clientConn.SetReadDeadline(time.Now().Add(wp.pongWait))
		return nil
	})

	backendURL := *wp.backend.URL
	if backendURL.Scheme == "http" {
		backendURL.Scheme = "ws"
	} else if backendURL.Scheme == "https" {
		backendURL.Scheme = "wss"
	}

	backendURL.Path = r.URL.Path
	backendURL.RawQuery = r.URL.RawQuery

	requestHeader := http.Header{}
	for k, vs := range r.Header {
		for _, v := range vs {
			requestHeader.Add(k, v)
		}
	}

	backendConn, resp, err := wp.dialer.Dial(backendURL.String(), requestHeader)
	if err != nil {
		logger.Log.Error("Failed to connect to backend",
			zap.String("backend", backendURL.String()),
			zap.Error(err))
		clientConn.Close()

		atomic.AddInt32(&wp.backend.ErrorCount, 1)
		if atomic.LoadInt32(&wp.backend.ErrorCount) >= 3 {
			wp.backend.SetAlive(false)
			wp.errorHandler(wp.backend)
		}

		return
	}

	if resp != nil && resp.Body != nil {
		resp.Body.Close()
	}

	connID := wp.connMap.Add(clientConn, backendConn)
	logger.Log.Info("WebSocket connection established",
		zap.String("connID", connID),
		zap.String("backend", backendURL.String()))

	backendConn.SetReadLimit(wp.maxMessageSize)
	backendConn.SetPongHandler(func(string) error {
		backendConn.SetReadDeadline(time.Now().Add(wp.pongWait))
		return nil
	})

	go wp.pumpToClient(clientConn, backendConn, connID)
	go wp.pumpToBackend(clientConn, backendConn, connID)
	go wp.pingConnection(clientConn, backendConn, connID)
}

func (wp *WebSocketProxy) pumpToClient(clientConn, backendConn *websocket.Conn, connID string) {
	defer func() {
		clientConn.Close()
		backendConn.Close()
		wp.connMap.Remove(connID)
		logger.Log.Info("WebSocket connection closed", zap.String("connID", connID))
	}()

	for {
		messageType, message, err := backendConn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				logger.Log.Error("Backend WebSocket error", zap.Error(err))
			}
			break
		}

		clientConn.SetWriteDeadline(time.Now().Add(wp.writeWait))
		if err := clientConn.WriteMessage(messageType, message); err != nil {
			break
		}
	}
}

func (wp *WebSocketProxy) pumpToBackend(clientConn, backendConn *websocket.Conn, connID string) {
	defer func() {
		clientConn.Close()
		backendConn.Close()
		wp.connMap.Remove(connID)
	}()

	for {
		messageType, message, err := clientConn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				logger.Log.Error("Client WebSocket error", zap.Error(err))
			}
			break
		}

		backendConn.SetWriteDeadline(time.Now().Add(wp.writeWait))
		if err := backendConn.WriteMessage(messageType, message); err != nil {
			break
		}
	}
}

func (wp *WebSocketProxy) pingConnection(clientConn, backendConn *websocket.Conn, connID string) {
	ticker := time.NewTicker(wp.pingInterval)
	defer func() {
		ticker.Stop()
		clientConn.Close()
		backendConn.Close()
		wp.connMap.Remove(connID)
	}()

	for {
		select {
		case <-ticker.C:
			clientConn.SetWriteDeadline(time.Now().Add(wp.writeWait))
			if err := clientConn.WriteMessage(websocket.PingMessage, []byte{}); err != nil {
				return
			}

			backendConn.SetWriteDeadline(time.Now().Add(wp.writeWait))
			if err := backendConn.WriteMessage(websocket.PingMessage, []byte{}); err != nil {
				return
			}
		}
	}
}

func IsWebSocketRequest(r *http.Request) bool {
	contains := func(key, val string) bool {
		values := r.Header.Values(key)
		for _, v := range values {
			if val == v {
				return true
			}
		}
		return false
	}

	return contains("Connection", "Upgrade") &&
		contains("Upgrade", "websocket") &&
		(r.Method == "GET")
}
