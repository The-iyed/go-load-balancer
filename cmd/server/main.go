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

	flag.StringVar(&configPath, "config", "conf/loadbalancer.conf", "accessing configuration file")
	flag.StringVar(&algorithm, "algorithm", "", "override load balancing algorithm: round-robin, weighted-round-robin, least-connections")
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

	if algorithm != "" {
		lb = balancer.CreateLoadBalancer(balancer.LoadBalancerAlgorithm(algorithm), config.Backends)
		logger.Log.Info("Using algorithm from command line", zap.String("algorithm", algorithm))
	} else {
		lb = balancer.CreateLoadBalancer(config.Method, config.Backends)
		logger.Log.Info("Using algorithm from config file", zap.String("algorithm", string(config.Method)))
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
