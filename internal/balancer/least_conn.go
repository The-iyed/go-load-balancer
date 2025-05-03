package balancer

import (
	"bufio"
	"fmt"
	"math"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync/atomic"
	"time"

	"github.com/The-iyed/go-load-balancer/internal/logger"
	"go.uber.org/zap"
)

type LeastConnectionsBalancer struct {
	ProcessPack []*Process
}

func NewLeastConnectionsBalancer(configs []BackendConfig) *LeastConnectionsBalancer {
	var processes []*Process

	for _, config := range configs {
		parsed, err := url.Parse(config.URL)
		if err != nil {
			logger.Log.Warn("Invalid backend URL", zap.String("url", config.URL), zap.Error(err))
			continue
		}

		process := &Process{
			URL:               parsed,
			Alive:             true,
			ErrorCount:        0,
			Weight:            config.Weight,
			ActiveConnections: 0,
		}

		processes = append(processes, process)
	}

	return &LeastConnectionsBalancer{
		ProcessPack: processes,
	}
}

func (lb *LeastConnectionsBalancer) GetNextInstance(r *http.Request) *Process {
	var minConnections int32 = math.MaxInt32
	var selectedIndex = -1

	for i, p := range lb.ProcessPack {
		if !p.IsAlive() {
			continue
		}

		connections := p.GetActiveConnections()

		if connections == minConnections && selectedIndex >= 0 {
			if p.Weight > lb.ProcessPack[selectedIndex].Weight {
				selectedIndex = i
			}
		} else if connections < minConnections {
			minConnections = connections
			selectedIndex = i
		}
	}

	if selectedIndex == -1 {
		return nil
	}

	return lb.ProcessPack[selectedIndex]
}

func (lb *LeastConnectionsBalancer) ProxyRequest(w http.ResponseWriter, r *http.Request) {
	target := lb.GetNextInstance(r)
	if target == nil {
		http.Error(w, "No healthy backends available", http.StatusServiceUnavailable)
		return
	}

	target.IncrementConnections()

	proxy := httputil.NewSingleHostReverseProxy(target.URL)

	rwWriter := &responseWriterInterceptor{
		ResponseWriter: w,
		process:        target,
	}

	proxy.ErrorHandler = func(w http.ResponseWriter, req *http.Request, err error) {
		logger.Log.Error("Request failed",
			zap.String("backend", target.URL.String()),
			zap.Error(err),
		)

		target.DecrementConnections()

		atomic.AddInt32(&target.ErrorCount, 1)
		if atomic.LoadInt32(&target.ErrorCount) >= 3 {
			target.SetAlive(false)
			logger.Log.Warn("Backend marked dead", zap.String("backend", target.URL.String()))
			go lb.reviveLater(target)
		}

		lb.ProxyRequest(w, r)
	}

	proxy.ServeHTTP(rwWriter, r)
}

func (lb *LeastConnectionsBalancer) reviveLater(p *Process) {
	time.Sleep(10 * time.Second)
	p.SetAlive(true)
	atomic.StoreInt32(&p.ErrorCount, 0)
	logger.Log.Info("Backend revived", zap.String("backend", p.URL.String()))
}

type responseWriterInterceptor struct {
	http.ResponseWriter
	process *Process
}

func (w *responseWriterInterceptor) WriteHeader(statusCode int) {
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *responseWriterInterceptor) Write(b []byte) (int, error) {
	n, err := w.ResponseWriter.Write(b)
	if err == nil {
		w.process.DecrementConnections()
	}
	return n, err
}

func (w *responseWriterInterceptor) Flush() {
	if flusher, ok := w.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

func (w *responseWriterInterceptor) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if hijacker, ok := w.ResponseWriter.(http.Hijacker); ok {
		return hijacker.Hijack()
	}
	return nil, nil, fmt.Errorf("response writer does not implement http.Hijacker")
}
