package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"sync/atomic"
	"syscall"
	"time"
)

type ServerInfo struct {
	ID           string    `json:"id"`
	RequestCount int64     `json:"request_count"`
	Timestamp    time.Time `json:"timestamp"`
}

func main() {
	var addr string
	var id string
	flag.StringVar(&addr, "addr", ":8001", "HTTP service address")
	flag.StringVar(&id, "id", "server1", "Server ID")
	flag.Parse()

	var requestCount int64 = 0
	startTime := time.Now()

	// Handler for API requests
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt64(&requestCount, 1)

		// Set headers to identify the backend
		w.Header().Set("X-Backend-ID", id)
		w.Header().Set("Content-Type", "application/json")

		// Return server info
		info := ServerInfo{
			ID:           id,
			RequestCount: count,
			Timestamp:    time.Now(),
		}

		json.NewEncoder(w).Encode(info)
	})

	// Health check endpoint
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Metrics endpoint
	http.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		data := map[string]interface{}{
			"server_id":      id,
			"request_count":  atomic.LoadInt64(&requestCount),
			"uptime_seconds": time.Since(startTime).Seconds(),
		}
		json.NewEncoder(w).Encode(data)
	})

	// Delay endpoint for testing least connections load balancing
	http.HandleFunc("/delay", func(w http.ResponseWriter, r *http.Request) {
		// Count the request
		count := atomic.AddInt64(&requestCount, 1)

		// Parse delay duration
		seconds := 1
		if s := r.URL.Query().Get("seconds"); s != "" {
			if val, err := strconv.Atoi(s); err == nil && val > 0 && val <= 30 {
				seconds = val
			}
		}

		// Log that we're delaying
		log.Printf("Server %s: Delaying response for %d seconds (request %d)", id, seconds, count)

		// Sleep for the specified duration
		time.Sleep(time.Duration(seconds) * time.Second)

		// Set headers to identify the backend
		w.Header().Set("X-Backend-ID", id)
		w.Header().Set("Content-Type", "application/json")

		// Return server info
		info := ServerInfo{
			ID:           id,
			RequestCount: count,
			Timestamp:    time.Now(),
		}

		json.NewEncoder(w).Encode(info)
	})

	// API specific endpoint for path-based routing tests
	http.HandleFunc("/api", func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt64(&requestCount, 1)

		w.Header().Set("X-Backend-ID", id)
		w.Header().Set("Content-Type", "application/json")

		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":            id,
			"request_count": count,
			"service":       "api",
			"timestamp":     time.Now(),
		})
	})

	// Static content endpoint for path-based routing tests
	http.HandleFunc("/static", func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt64(&requestCount, 1)

		w.Header().Set("X-Backend-ID", id)
		w.Header().Set("Content-Type", "application/json")

		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":            id,
			"request_count": count,
			"service":       "static",
			"timestamp":     time.Now(),
		})
	})

	// Images endpoint for path-based routing tests
	http.HandleFunc("/images", func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt64(&requestCount, 1)

		w.Header().Set("X-Backend-ID", id)
		w.Header().Set("Content-Type", "application/json")

		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":            id,
			"request_count": count,
			"service":       "images",
			"timestamp":     time.Now(),
		})
	})

	// Start server
	log.Printf("Starting HTTP API server %s on %s", id, addr)

	// Setup graceful shutdown
	go func() {
		server := &http.Server{Addr: addr}

		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan

		log.Printf("Shutting down server %s...", id)
		server.Close()
		os.Exit(0)
	}()

	// Start HTTP server
	err := http.ListenAndServe(addr, nil)
	if err != nil {
		fmt.Printf("Error starting server: %v\n", err)
		os.Exit(1)
	}
}
