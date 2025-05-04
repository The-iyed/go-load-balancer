package balancer

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"hash/crc32"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/The-iyed/go-load-balancer/internal/logger"
	"go.uber.org/zap"
)

type SessionPersistenceBalancer struct {
	ProcessPack        []*Process
	BaseLB             interface{}
	PersistenceMethod  PersistenceMethod
	ConsistentHashRing *ConsistentHashRing
	CookieName         string
	CookieTTL          time.Duration
	IPToBackendMap     sync.Map
	BackendToIndexMap  map[string]int
}

func NewSessionPersistenceBalancer(configs []BackendConfig, algorithm LoadBalancerAlgorithm, persistenceMethod PersistenceMethod) *SessionPersistenceBalancer {
	var baseLB interface{}

	switch algorithm {
	case LeastConnections:
		baseLB = NewLeastConnectionsBalancer(configs)
	case WeightedRoundRobin, RoundRobin:
		baseLB = NewLoadBalancer(configs)
	default:
		baseLB = NewLoadBalancer(configs)
	}

	var processes []*Process
	backendToIndexMap := make(map[string]int)

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

		processes = append(processes, process)
		backendToIndexMap[parsed.String()] = len(processes) - 1
	}

	consistentHashRing := NewConsistentHashRing(configs)

	return &SessionPersistenceBalancer{
		ProcessPack:        processes,
		BaseLB:             baseLB,
		PersistenceMethod:  persistenceMethod,
		ConsistentHashRing: consistentHashRing,
		CookieName:         "GOLB_SESSION",
		CookieTTL:          24 * time.Hour,
		BackendToIndexMap:  backendToIndexMap,
	}
}

func (lb *SessionPersistenceBalancer) GetNextInstance(r *http.Request) (*url.URL, error) {
	var process *Process

	switch lb.PersistenceMethod {
	case CookiePersistence:
		process = lb.getInstanceByCookie(r)
	case IPHashPersistence:
		process = lb.getInstanceByIPHash(r)
	case ConsistentHashPersistence:
		process = lb.getInstanceByConsistentHash(r)
	default:
		if adapter, ok := lb.BaseLB.(*LegacyLoadBalancerAdapter); ok {
			return adapter.GetNextInstance(r)
		}

		// Get from the underlying implementation directly
		switch base := lb.BaseLB.(type) {
		case *WeightedRoundRobinBalancer:
			process = base.GetNextInstance(r)
		case *LeastConnectionsBalancer:
			process = base.GetNextInstance(r)
		}
	}

	if process == nil {
		return nil, fmt.Errorf("no available backends")
	}

	return process.URL, nil
}

func (lb *SessionPersistenceBalancer) getInstanceByCookie(r *http.Request) *Process {
	cookie, err := r.Cookie(lb.CookieName)

	if err == nil && cookie.Value != "" {
		parts := strings.Split(cookie.Value, ":")
		if len(parts) == 2 {
			index, err := strconv.Atoi(parts[0])
			if err == nil && index >= 0 && index < len(lb.ProcessPack) {
				backend := lb.ProcessPack[index]
				if backend.IsAlive() {
					return backend
				}
			}
		}
	}

	// Get from the underlying implementation
	var process *Process
	switch base := lb.BaseLB.(type) {
	case *WeightedRoundRobinBalancer:
		process = base.GetNextInstance(r)
	case *LeastConnectionsBalancer:
		process = base.GetNextInstance(r)
	}
	return process
}

func (lb *SessionPersistenceBalancer) getInstanceByIPHash(r *http.Request) *Process {
	ip := getClientIP(r)
	if ip == "" {
		// Get from the underlying implementation
		var process *Process
		switch base := lb.BaseLB.(type) {
		case *WeightedRoundRobinBalancer:
			process = base.GetNextInstance(r)
		case *LeastConnectionsBalancer:
			process = base.GetNextInstance(r)
		}
		return process
	}

	if target, ok := lb.IPToBackendMap.Load(ip); ok {
		index := target.(int)
		if index >= 0 && index < len(lb.ProcessPack) && lb.ProcessPack[index].IsAlive() {
			return lb.ProcessPack[index]
		}
	}

	// Get from the underlying implementation
	var target *Process
	switch base := lb.BaseLB.(type) {
	case *WeightedRoundRobinBalancer:
		target = base.GetNextInstance(r)
	case *LeastConnectionsBalancer:
		target = base.GetNextInstance(r)
	}

	if target != nil {
		lb.IPToBackendMap.Store(ip, lb.BackendToIndexMap[target.URL.String()])
	}

	return target
}

func (lb *SessionPersistenceBalancer) getInstanceByConsistentHash(r *http.Request) *Process {
	key := r.URL.Path

	if key == "" {
		// Get from the underlying implementation
		var process *Process
		switch base := lb.BaseLB.(type) {
		case *WeightedRoundRobinBalancer:
			process = base.GetNextInstance(r)
		case *LeastConnectionsBalancer:
			process = base.GetNextInstance(r)
		}
		return process
	}

	return lb.ConsistentHashRing.GetNode(key)
}

func (lb *SessionPersistenceBalancer) ProxyRequest(w http.ResponseWriter, r *http.Request) {
	target, err := lb.GetNextInstance(r)
	if err != nil || target == nil {
		http.Error(w, "No healthy backends available", http.StatusServiceUnavailable)
		return
	}

	var process *Process
	for _, p := range lb.ProcessPack {
		if p.URL.String() == target.String() {
			process = p
			break
		}
	}

	if process == nil {
		http.Error(w, "Backend not found", http.StatusInternalServerError)
		return
	}

	if IsWebSocketRequest(r) && lb.SupportsWebSockets() {
		wsProxy := NewWebSocketProxy(process, func(p *Process) {
			go lb.reviveLater(p)
		})
		wsProxy.ProxyWebSocket(w, r)
		return
	}

	if lb.PersistenceMethod == CookiePersistence {
		index := -1
		for i, backend := range lb.ProcessPack {
			if backend.URL.String() == target.String() {
				index = i
				break
			}
		}

		if index >= 0 {
			hash := md5.Sum([]byte(target.String()))
			cookie := &http.Cookie{
				Name:     lb.CookieName,
				Value:    fmt.Sprintf("%d:%s", index, hex.EncodeToString(hash[:])),
				Path:     "/",
				HttpOnly: true,
				Secure:   r.TLS != nil,
				MaxAge:   int(lb.CookieTTL.Seconds()),
			}
			http.SetCookie(w, cookie)
		}
	}

	proxy := httputil.NewSingleHostReverseProxy(target)
	proxy.ErrorHandler = func(w http.ResponseWriter, req *http.Request, err error) {
		logger.Log.Error("Request failed",
			zap.String("backend", target.String()),
			zap.Error(err),
		)

		if process != nil {
			atomic.AddInt32(&process.ErrorCount, 1)
			if atomic.LoadInt32(&process.ErrorCount) >= 3 {
				process.SetAlive(false)
				logger.Log.Warn("Backend marked dead", zap.String("backend", target.String()))
				go lb.reviveLater(process)
			}
		}

		lb.ProxyRequest(w, r)
	}

	proxy.ServeHTTP(w, r)
}

func (lb *SessionPersistenceBalancer) reviveLater(p *Process) {
	time.Sleep(10 * time.Second)
	p.SetAlive(true)
	atomic.StoreInt32(&p.ErrorCount, 0)
	logger.Log.Info("Backend revived", zap.String("backend", p.URL.String()))
}

func (lb *SessionPersistenceBalancer) SupportsWebSockets() bool {
	return true
}

type ConsistentHashRing struct {
	ring         map[uint32]*Process
	sortedHashes []uint32
	replicaCount int
	processes    []*Process
}

func NewConsistentHashRing(configs []BackendConfig) *ConsistentHashRing {
	ch := &ConsistentHashRing{
		ring:         make(map[uint32]*Process),
		replicaCount: 100,
	}

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

		ch.processes = append(ch.processes, process)

		for i := 0; i < ch.replicaCount*weight; i++ {
			key := fmt.Sprintf("%s:%d", parsed.String(), i)
			hash := crc32.ChecksumIEEE([]byte(key))
			ch.ring[hash] = process
			ch.sortedHashes = append(ch.sortedHashes, hash)
		}
	}

	sort.Slice(ch.sortedHashes, func(i, j int) bool {
		return ch.sortedHashes[i] < ch.sortedHashes[j]
	})

	return ch
}

func (ch *ConsistentHashRing) GetNode(key string) *Process {
	if len(ch.ring) == 0 {
		return nil
	}

	hash := crc32.ChecksumIEEE([]byte(key))

	idx := sort.Search(len(ch.sortedHashes), func(i int) bool {
		return ch.sortedHashes[i] >= hash
	})

	if idx == len(ch.sortedHashes) {
		idx = 0
	}

	process := ch.ring[ch.sortedHashes[idx]]

	if !process.IsAlive() {
		for i := 0; i < len(ch.processes); i++ {
			nextIdx := (idx + i) % len(ch.sortedHashes)
			process = ch.ring[ch.sortedHashes[nextIdx]]
			if process.IsAlive() {
				return process
			}
		}
		return nil
	}

	return process
}

func getClientIP(r *http.Request) string {
	xForwardedFor := r.Header.Get("X-Forwarded-For")
	if xForwardedFor != "" {
		ips := strings.Split(xForwardedFor, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}

	if r.RemoteAddr != "" {
		ip, _, err := net.SplitHostPort(r.RemoteAddr)
		if err == nil {
			return ip
		}
	}

	return ""
}
