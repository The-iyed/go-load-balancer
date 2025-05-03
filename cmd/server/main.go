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

	flag.StringVar(&configPath, "config", "conf/loadbalancer.conf", "accessing configuration file")
	flag.StringVar(&algorithm, "algorithm", "", "override load balancing algorithm: round-robin, weighted-round-robin, least-connections")
	flag.StringVar(&persistence, "persistence", "", "override persistence method: none, cookie, ip_hash, consistent_hash")
	flag.Parse()

	logger.InitLogger()

	config, err := balancer.ParseConfig(configPath)
	if err != nil {
		logger.Log.Fatal("Failed to parse configuration", zap.Error(err))
	}

	if len(config.Backends) == 0 {
		logger.Log.Fatal("No backends configured")
	}

	var lb balancer.LoadBalancerStrategy

	algoOverride := balancer.LoadBalancerAlgorithm(algorithm)
	persistenceOverride := balancer.PersistenceMethod(persistence)

	if algorithm != "" {
		if persistence != "" {
			lb = balancer.CreateLoadBalancer(algoOverride, config.Backends, persistenceOverride)
			logger.Log.Info("Using algorithm and persistence from command line", zap.String("algorithm", algorithm), zap.String("persistence", persistence))
		} else {
			lb = balancer.CreateLoadBalancer(algoOverride, config.Backends, config.Persistence)
			logger.Log.Info("Using algorithm from command line, persistence from config", zap.String("algorithm", algorithm), zap.String("persistence", string(config.Persistence)))
		}
	} else if persistence != "" {
		lb = balancer.CreateLoadBalancer(config.Method, config.Backends, persistenceOverride)
		logger.Log.Info("Using algorithm from config, persistence from command line", zap.String("algorithm", string(config.Method)), zap.String("persistence", persistence))
	} else {
		lb = balancer.CreateLoadBalancer(config.Method, config.Backends, config.Persistence)
		logger.Log.Info("Using algorithm and persistence from config", zap.String("algorithm", string(config.Method)), zap.String("persistence", string(config.Persistence)))
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
