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

type LoadBalancer struct {
	ProcessPack []*Process
	Current     uint64
}

func NewLoadBalancer(urls []string) *LoadBalancer {
	var processes []*Process
	for _, raw := range urls {
		parsed, err := url.Parse(raw)
		if err != nil {
			logger.Log.Warn("Invalid backend URL", zap.String("url", raw), zap.Error(err))
			continue
		}
		processes = append(processes, &Process{
			URL:        parsed,
			Alive:      true,
			ErrorCount: 0,
		})
	}
	return &LoadBalancer{ProcessPack: processes}
}

func (lb *LoadBalancer) GetNextInstance() *Process {
	total := len(lb.ProcessPack)
	for i := 0; i < total; i++ {
		idx := int(atomic.AddUint64(&lb.Current, 1)) % total
		if lb.ProcessPack[idx].IsAlive() {
			return lb.ProcessPack[idx]
		}
	}
	return nil
}

func (lb *LoadBalancer) ProxyRequest(w http.ResponseWriter, r *http.Request) {
	target := lb.GetNextInstance()
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

func (lb *LoadBalancer) reviveLater(p *Process) {
	time.Sleep(10 * time.Second)
	p.SetAlive(true)
	atomic.StoreInt32(&p.ErrorCount, 0)
	logger.Log.Info("Backend revived", zap.String("backend", p.URL.String()))
}
