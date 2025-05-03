package main

import (
	"flag"
	"log"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

var (
	addr     = flag.String("addr", ":8001", "HTTP service address")
	id       = flag.String("id", "server1", "Server ID")
	upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin:     func(r *http.Request) bool { return true },
	}
	messageCounter int64
)

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Upgrade error:", err)
		return
	}
	defer c.Close()

	// Set read deadline and pong handler
	c.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.SetPongHandler(func(string) error {
		c.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	// Send initial message
	if err := c.WriteMessage(websocket.TextMessage, []byte("Welcome from "+*id)); err != nil {
		log.Println("Write error:", err)
		return
	}

	// Start ping ticker
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	go func() {
		for range ticker.C {
			if err := c.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}()

	for {
		messageType, message, err := c.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("Error: %v", err)
			}
			break
		}

		count := atomic.AddInt64(&messageCounter, 1)
		log.Printf("Received message [%d]: %s from %s", count, message, r.RemoteAddr)

		// Echo the message back with server ID
		response := []byte(*id + " received: " + string(message))
		err = c.WriteMessage(messageType, response)
		if err != nil {
			log.Println("Write error:", err)
			break
		}
	}
}

func statusHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte("WebSocket Server: " + *id + "\n"))
	w.Write([]byte("Messages Received: " + string(messageCounter) + "\n"))
}

func main() {
	flag.Parse()
	log.SetFlags(0)

	http.HandleFunc("/status", statusHandler)
	http.HandleFunc("/ws", handleWebSocket)

	log.Printf("Starting WebSocket server %s on %s", *id, *addr)
	log.Fatal(http.ListenAndServe(*addr, nil))
}
