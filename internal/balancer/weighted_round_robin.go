package balancer

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync/atomic"
	"time"

	"github.com/The-iyed/go-load-balancer/internal/logger"
	"go.uber.org/zap"
)

type WeightedRoundRobinBalancer struct {
	ProcessPack []*Process
	Current     uint64
	TotalWeight int
}

func NewLoadBalancer(configs []BackendConfig) *WeightedRoundRobinBalancer {
	var processes []*Process
	totalWeight := 0

	for _, config := range configs {
		parsed, err := url.Parse(config.URL)
		if err != nil {
			logger.Log.Warn("Invalid backend URL", zap.String("url", config.URL), zap.Error(err))
			continue
		}

		weight := config.Weight
		if weight <= 0 {
			weight = 1
		}

		process := &Process{
			URL:        parsed,
			Alive:      true,
			ErrorCount: 0,
			Weight:     weight,
		}
		process.ResetCurrentWeight()

		processes = append(processes, process)
		totalWeight += weight
	}

	return &WeightedRoundRobinBalancer{
		ProcessPack: processes,
		TotalWeight: totalWeight,
	}
}

func (lb *WeightedRoundRobinBalancer) GetNextInstance(r *http.Request) *Process {
	if len(lb.ProcessPack) == 0 {
		return nil
	}

	var selected *Process
	maxCurrent := 0

	for _, p := range lb.ProcessPack {
		if !p.IsAlive() {
			continue
		}

		if p.Current > maxCurrent {
			maxCurrent = p.Current
			selected = p
		}
	}

	if selected == nil {
		return nil
	}

	for _, p := range lb.ProcessPack {
		if p.IsAlive() {
			p.Current += p.Weight
		}
	}

	selected.Current -= lb.TotalWeight
	return selected
}

func (lb *WeightedRoundRobinBalancer) ProxyRequest(w http.ResponseWriter, r *http.Request) {
	target := lb.GetNextInstance(r)
	if target == nil {
		http.Error(w, "No healthy backends available", http.StatusServiceUnavailable)
		return
	}

	proxy := httputil.NewSingleHostReverseProxy(target.URL)
	proxy.ErrorHandler = func(w http.ResponseWriter, req *http.Request, err error) {
		logger.Log.Error("Request failed",
			zap.String("backend", target.URL.String()),
			zap.Error(err),
		)

		atomic.AddInt32(&target.ErrorCount, 1)
		if atomic.LoadInt32(&target.ErrorCount) >= 3 {
			target.SetAlive(false)
			logger.Log.Warn("Backend marked dead", zap.String("backend", target.URL.String()))
			go lb.reviveLater(target)
		}

		lb.ProxyRequest(w, r)
	}

	proxy.ServeHTTP(w, r)
}

func (lb *WeightedRoundRobinBalancer) reviveLater(p *Process) {
	time.Sleep(10 * time.Second)
	p.SetAlive(true)
	atomic.StoreInt32(&p.ErrorCount, 0)
	logger.Log.Info("Backend revived", zap.String("backend", p.URL.String()))
}
