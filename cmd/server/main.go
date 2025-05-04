package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/The-iyed/go-load-balancer/internal/balancer"
	"github.com/The-iyed/go-load-balancer/internal/logger"
	"go.uber.org/zap"
)

func main() {
	var configPath string
	var algorithm string
	var persistence string
	var enablePathRouting bool
	var port int
	var adminPort int

	flag.StringVar(&configPath, "config", "conf/loadbalancer.conf", "accessing configuration file")
	flag.StringVar(&algorithm, "algorithm", "", "override load balancing algorithm: round-robin, weighted-round-robin, least-connections")
	flag.StringVar(&persistence, "persistence", "", "override persistence method: none, cookie, ip_hash, consistent_hash")
	flag.BoolVar(&enablePathRouting, "path-routing", false, "enable path-based routing")
	flag.IntVar(&port, "port", 8080, "port to listen on")
	flag.IntVar(&adminPort, "admin-port", 8081, "port for admin UI API server")
	flag.Parse()

	logger.InitLogger()

	config, err := balancer.ParseConfig(configPath)
	if err != nil {
		logger.Log.Fatal("Failed to parse configuration", zap.Error(err))
	}

	var lb balancer.LoadBalancerStrategy

	if enablePathRouting || len(config.Routes) > 0 {
		// Path-based routing mode
		logger.Log.Info("Using path-based routing")
		lb, err = balancer.CreatePathRouter(config)
		if err != nil {
			logger.Log.Fatal("Failed to create path router", zap.Error(err))
		}
	} else {
		// Traditional load balancing mode
		if len(config.Backends) == 0 {
			logger.Log.Fatal("No backends configured")
		}

		persistenceAttrs := make(map[string]string)

		var method balancer.LoadBalancerAlgorithm
		var persistenceMethod balancer.PersistenceMethod

		// Handle command line overrides
		if algorithm != "" {
			switch algorithm {
			case "round_robin", "round-robin":
				method = balancer.RoundRobin
			case "weighted_round_robin", "weighted-round-robin":
				method = balancer.WeightedRoundRobin
			case "least_connections", "least-connections":
				method = balancer.LeastConnections
			default:
				logger.Log.Fatal("Unknown algorithm", zap.String("algorithm", algorithm))
			}
		} else {
			method = config.Method
		}

		if persistence != "" {
			switch persistence {
			case "none":
				persistenceMethod = balancer.NoPersistence
			case "cookie":
				persistenceMethod = balancer.CookiePersistence
			case "ip_hash":
				persistenceMethod = balancer.IPHashPersistence
			case "consistent_hash":
				persistenceMethod = balancer.ConsistentHashPersistence
			default:
				logger.Log.Fatal("Unknown persistence method", zap.String("persistence", persistence))
			}
		} else {
			persistenceMethod = config.PersistenceType
			persistenceAttrs = config.PersistenceAttrs
		}

		lb, err = balancer.CreateLoadBalancer(method, config.Backends, persistenceMethod, persistenceAttrs)
		if err != nil {
			logger.Log.Fatal("Failed to create load balancer", zap.Error(err))
		}

		logger.Log.Info("Load balancer configured",
			zap.String("algorithm", algorithm),
			zap.String("persistence", persistence),
			zap.Int("backends", len(config.Backends)))
	}

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: http.HandlerFunc(lb.ProxyRequest),
	}

	// Create a listener first if using dynamic port
	var listener net.Listener
	var actualPort int
	if port == 0 {
		listener, err = net.Listen("tcp", ":0")
		if err != nil {
			logger.Log.Fatal("Failed to create listener", zap.Error(err))
		}
		actualPort = listener.Addr().(*net.TCPAddr).Port
		logger.Log.Info("Load balancer listening", zap.Int("port", actualPort))
	}

	// Start the main proxy server
	go func() {
		logger.Log.Info("Starting load balancer", zap.Int("port", port))

		var err error
		if listener != nil {
			err = server.Serve(listener)
		} else {
			err = server.ListenAndServe()
		}

		if err != nil && err != http.ErrServerClosed {
			logger.Log.Fatal("Failed to start server", zap.Error(err))
		}
	}()

	// Create the admin API server for the web UI
	adminServer := &http.Server{
		Addr: fmt.Sprintf(":%d", adminPort),
	}

	// Define API routes
	adminMux := http.NewServeMux()
	adminMux.HandleFunc("/api/stats", balancer.APIHandler(lb))

	// Add static file serving for the web UI
	adminMux.Handle("/", http.FileServer(http.Dir("./web-ui/dist")))

	adminServer.Handler = adminMux

	// Start the admin API server
	go func() {
		logger.Log.Info("Starting admin API server", zap.Int("port", adminPort))
		if err := adminServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Log.Error("Failed to start admin server", zap.Error(err))
		}
	}()

	// If port is 0, find the actual port the server is listening on
	if port == 0 {
		// Override the port with the actual port for tests
		port = actualPort
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Log.Info("Shutting down servers...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Shut down both servers
	if err := server.Shutdown(ctx); err != nil {
		logger.Log.Fatal("Main server forced to shutdown", zap.Error(err))
	}

	if err := adminServer.Shutdown(ctx); err != nil {
		logger.Log.Error("Admin server forced to shutdown", zap.Error(err))
	}

	logger.Log.Info("Servers exiting")
}
