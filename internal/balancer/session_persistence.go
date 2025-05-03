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
	BaseLB             LoadBalancerStrategy
	PersistenceMethod  PersistenceMethod
	ConsistentHashRing *ConsistentHashRing
	CookieName         string
	CookieTTL          time.Duration
	IPToBackendMap     sync.Map
	BackendToIndexMap  map[string]int
}

func NewSessionPersistenceBalancer(configs []BackendConfig, algorithm LoadBalancerAlgorithm, persistenceMethod PersistenceMethod) *SessionPersistenceBalancer {
	var baseLB LoadBalancerStrategy

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

func (lb *SessionPersistenceBalancer) GetNextInstance(r *http.Request) *Process {
	switch lb.PersistenceMethod {
	case CookiePersistence:
		return lb.getInstanceByCookie(r)
	case IPHashPersistence:
		return lb.getInstanceByIPHash(r)
	case ConsistentHashPersistence:
		return lb.getInstanceByConsistentHash(r)
	default:
		return lb.BaseLB.GetNextInstance(r)
	}
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

	return lb.BaseLB.GetNextInstance(r)
}

func (lb *SessionPersistenceBalancer) getInstanceByIPHash(r *http.Request) *Process {
	ip := getClientIP(r)
	if ip == "" {
		return lb.BaseLB.GetNextInstance(r)
	}

	if target, ok := lb.IPToBackendMap.Load(ip); ok {
		index := target.(int)
		if index >= 0 && index < len(lb.ProcessPack) && lb.ProcessPack[index].IsAlive() {
			return lb.ProcessPack[index]
		}
	}

	target := lb.BaseLB.GetNextInstance(r)
	if target != nil {
		lb.IPToBackendMap.Store(ip, lb.BackendToIndexMap[target.URL.String()])
	}

	return target
}

func (lb *SessionPersistenceBalancer) getInstanceByConsistentHash(r *http.Request) *Process {
	key := r.URL.Path

	if key == "" {
		return lb.BaseLB.GetNextInstance(r)
	}

	return lb.ConsistentHashRing.GetNode(key)
}

func (lb *SessionPersistenceBalancer) ProxyRequest(w http.ResponseWriter, r *http.Request) {
	target := lb.GetNextInstance(r)
	if target == nil {
		http.Error(w, "No healthy backends available", http.StatusServiceUnavailable)
		return
	}

	if IsWebSocketRequest(r) && lb.SupportsWebSockets() {
		wsProxy := NewWebSocketProxy(target, func(p *Process) {
			go lb.reviveLater(p)
		})
		wsProxy.ProxyWebSocket(w, r)
		return
	}

	if lb.PersistenceMethod == CookiePersistence {
		index := -1
		for i, backend := range lb.ProcessPack {
			if backend.URL.String() == target.URL.String() {
				index = i
				break
			}
		}

		if index >= 0 {
			hash := md5.Sum([]byte(target.URL.String()))
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
