package main

import (
	"context"
	"flag"
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

	flag.StringVar(&configPath, "config", "conf/loadbalancer.conf", "accessing configuration file")
	flag.StringVar(&algorithm, "algorithm", "", "override load balancing algorithm: round-robin, weighted-round-robin, least-connections")
	flag.StringVar(&persistence, "persistence", "", "override persistence method: none, cookie, ip_hash, consistent_hash")
	flag.BoolVar(&enablePathRouting, "path-routing", false, "enable path-based routing")
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
		Addr:    ":8080",
		Handler: http.HandlerFunc(lb.ProxyRequest),
	}

	go func() {
		logger.Log.Info("Starting load balancer", zap.Int("port", 8080))
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Log.Fatal("Failed to start server", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Log.Info("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Log.Fatal("Server forced to shutdown", zap.Error(err))
	}

	logger.Log.Info("Server exiting")
}
