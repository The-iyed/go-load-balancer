package balancer

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

func ParseConfigFile() ([]string, error) {
	path := filepath.Join("config", "load_balancer.conf")
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var servers []string
	scanner := bufio.NewScanner(file)
	inUpstream := false

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if strings.HasPrefix(line, "upstream ") {
			inUpstream = true
			continue
		}

		if inUpstream {
			if strings.Contains(line, "server ") {
				line = strings.TrimSuffix(line, ";")
				parts := strings.Fields(line)
				if len(parts) == 2 {
					servers = append(servers, parts[1])
				}
			} else if line == "}" {
				break
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return servers, nil
}
