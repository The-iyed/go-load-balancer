package balancer

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

type Config struct {
	Backends    []BackendConfig
	Method      LoadBalancerAlgorithm
	Persistence PersistenceMethod
}

type BackendConfig struct {
	URL    string
	Weight int
}

func ParseConfig(filePath string) (*Config, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("error opening config file: %v", err)
	}
	defer file.Close()

	config := &Config{
		Method:      WeightedRoundRobin,
		Persistence: NoPersistence,
	}

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		if strings.HasPrefix(line, "upstream") {
			continue
		} else if strings.HasPrefix(line, "}") {
			continue
		} else if strings.HasPrefix(line, "server") {
			parts := strings.Fields(line)
			if len(parts) < 2 {
				continue
			}

			url := parts[1]
			weight := 1

			for _, part := range parts[2:] {
				if strings.HasPrefix(part, "weight=") {
					fmt.Sscanf(part, "weight=%d", &weight)
				}
			}

			config.Backends = append(config.Backends, BackendConfig{
				URL:    url,
				Weight: weight,
			})
		} else if strings.HasPrefix(line, "method") {
			parts := strings.Fields(line)
			if len(parts) < 2 {
				continue
			}

			method := parts[1]
			if strings.HasSuffix(method, ";") {
				method = method[:len(method)-1]
			}

			switch method {
			case "least_conn":
				config.Method = LeastConnections
			case "round_robin":
				config.Method = RoundRobin
			case "weighted_round_robin":
				config.Method = WeightedRoundRobin
			default:
				config.Method = WeightedRoundRobin
			}
		} else if strings.HasPrefix(line, "persistence") {
			parts := strings.Fields(line)
			if len(parts) < 2 {
				continue
			}

			persistence := parts[1]
			if strings.HasSuffix(persistence, ";") {
				persistence = persistence[:len(persistence)-1]
			}

			switch persistence {
			case "cookie":
				config.Persistence = CookiePersistence
			case "ip_hash":
				config.Persistence = IPHashPersistence
			case "consistent_hash":
				config.Persistence = ConsistentHashPersistence
			case "none":
				config.Persistence = NoPersistence
			default:
				config.Persistence = NoPersistence
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading config file: %v", err)
	}

	return config, nil
}
