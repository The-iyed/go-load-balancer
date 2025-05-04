package balancer

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// RouteType defines the type of routing rule
type RouteType int

const (
	// PathRoute matches against URL path
	PathRoute RouteType = iota
	// RegexRoute uses regex pattern matching against URL path
	RegexRoute
	// HeaderRoute matches based on HTTP headers
	HeaderRoute
)

type BackendConfig struct {
	URL      string
	Weight   int
	MaxConns int
}

type RouteConfig struct {
	Type        RouteType
	Pattern     string
	HeaderName  string
	HeaderValue string
	BackendPool string
}

type Config struct {
	Backends         []BackendConfig
	BackendPools     map[string][]BackendConfig
	Routes           []RouteConfig
	DefaultBackend   string
	Method           LoadBalancerAlgorithm
	PersistenceType  PersistenceMethod
	PersistenceAttrs map[string]string
}

func ParseConfig(filename string) (*Config, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	cfg := &Config{
		Backends:         []BackendConfig{},
		BackendPools:     make(map[string][]BackendConfig),
		Routes:           []RouteConfig{},
		DefaultBackend:   "",
		Method:           RoundRobin,
		PersistenceType:  NoPersistence,
		PersistenceAttrs: make(map[string]string),
	}

	scanner := bufio.NewScanner(file)
	var currentUpstream string
	isInsideUpstream := false

	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.Fields(line)
		directive := parts[0]

		switch directive {
		case "upstream":
			if len(parts) < 2 {
				return nil, fmt.Errorf("line %d: upstream directive requires a name", lineNum)
			}
			currentUpstream = parts[1]
			isInsideUpstream = true
			if _, exists := cfg.BackendPools[currentUpstream]; !exists {
				cfg.BackendPools[currentUpstream] = []BackendConfig{}
			}

		case "server":
			if !isInsideUpstream {
				return nil, fmt.Errorf("line %d: server directive must be inside an upstream block", lineNum)
			}
			if len(parts) < 2 {
				return nil, fmt.Errorf("line %d: server directive requires an URL", lineNum)
			}

			backend := BackendConfig{URL: parts[1], Weight: 1, MaxConns: 0}

			for i := 2; i < len(parts); i++ {
				if strings.HasPrefix(parts[i], "weight=") {
					weightStr := strings.TrimPrefix(parts[i], "weight=")
					weight, err := strconv.Atoi(weightStr)
					if err != nil {
						return nil, fmt.Errorf("line %d: invalid weight: %s", lineNum, weightStr)
					}
					backend.Weight = weight
				} else if strings.HasPrefix(parts[i], "max_conn=") {
					maxConnStr := strings.TrimPrefix(parts[i], "max_conn=")
					maxConn, err := strconv.Atoi(maxConnStr)
					if err != nil {
						return nil, fmt.Errorf("line %d: invalid max_conn: %s", lineNum, maxConnStr)
					}
					backend.MaxConns = maxConn
				}
			}

			// If this is the default backend pool, add to both
			if currentUpstream == "backend" {
				cfg.Backends = append(cfg.Backends, backend)
			}
			// Add to the named backend pool
			cfg.BackendPools[currentUpstream] = append(cfg.BackendPools[currentUpstream], backend)

		case "}":
			isInsideUpstream = false

		case "method":
			if len(parts) < 2 {
				return nil, fmt.Errorf("line %d: method directive requires a value", lineNum)
			}

			method := strings.ToLower(parts[1])
			switch method {
			case "round_robin":
				cfg.Method = RoundRobin
			case "weighted_round_robin", "weighted":
				cfg.Method = WeightedRoundRobin
			case "least_connections", "least_conn":
				cfg.Method = LeastConnections
			default:
				return nil, fmt.Errorf("line %d: unknown load balancing method: %s", lineNum, method)
			}

		case "persistence":
			if len(parts) < 2 {
				return nil, fmt.Errorf("line %d: persistence directive requires a method", lineNum)
			}

			method := strings.ToLower(parts[1])
			switch method {
			case "none":
				cfg.PersistenceType = NoPersistence
			case "cookie":
				cfg.PersistenceType = CookiePersistence
				if len(parts) > 2 {
					for i := 2; i < len(parts); i++ {
						if strings.HasPrefix(parts[i], "name=") {
							cfg.PersistenceAttrs["cookie_name"] = strings.TrimPrefix(parts[i], "name=")
						} else if strings.HasPrefix(parts[i], "ttl=") {
							cfg.PersistenceAttrs["cookie_ttl"] = strings.TrimPrefix(parts[i], "ttl=")
						}
					}
				}
			case "ip_hash":
				cfg.PersistenceType = IPHashPersistence
			case "consistent_hash":
				cfg.PersistenceType = ConsistentHashPersistence
			default:
				return nil, fmt.Errorf("line %d: unknown persistence method: %s", lineNum, method)
			}

		case "route":
			if len(parts) < 4 {
				return nil, fmt.Errorf("line %d: route directive requires type, pattern, and backend", lineNum)
			}

			routeType := strings.ToLower(parts[1])
			pattern := parts[2]
			backendPool := parts[3]

			var routeConfig RouteConfig

			switch routeType {
			case "path":
				routeConfig = RouteConfig{
					Type:        PathRoute,
					Pattern:     pattern,
					BackendPool: backendPool,
				}
			case "regex":
				routeConfig = RouteConfig{
					Type:        RegexRoute,
					Pattern:     pattern,
					BackendPool: backendPool,
				}
			case "header":
				if len(parts) < 5 {
					return nil, fmt.Errorf("line %d: header route requires name, value, and backend", lineNum)
				}
				routeConfig = RouteConfig{
					Type:        HeaderRoute,
					Pattern:     "", // Not used for header routing
					HeaderName:  parts[2],
					HeaderValue: parts[3],
					BackendPool: parts[4],
				}
			default:
				return nil, fmt.Errorf("line %d: unknown route type: %s", lineNum, routeType)
			}

			cfg.Routes = append(cfg.Routes, routeConfig)

		case "default_backend":
			if len(parts) < 2 {
				return nil, fmt.Errorf("line %d: default_backend directive requires a backend pool name", lineNum)
			}
			cfg.DefaultBackend = parts[1]

		default:
			return nil, fmt.Errorf("line %d: unknown directive: %s", lineNum, directive)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	// If no default backend specified, use "backend" if available
	if cfg.DefaultBackend == "" {
		if _, ok := cfg.BackendPools["backend"]; ok {
			cfg.DefaultBackend = "backend"
		} else if len(cfg.BackendPools) > 0 {
			// Otherwise, use the first available backend pool
			for name := range cfg.BackendPools {
				cfg.DefaultBackend = name
				break
			}
		} else {
			return nil, fmt.Errorf("no backend pools defined in configuration")
		}
	}

	return cfg, nil
}
