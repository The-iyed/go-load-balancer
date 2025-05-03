# Load Balancing Algorithms

This document provides detailed information about the load balancing algorithms and session persistence methods implemented in this project.

## Load Balancing Algorithms

### Weighted Round Robin

The Weighted Round Robin algorithm distributes requests across backend servers proportionally based on their assigned weights.

#### Implementation Details

Our implementation uses a dynamic weight adjustment mechanism based on the highest effective weight approach:

```go
func (lb *WeightedRoundRobinBalancer) GetNextInstance(r *http.Request) *Process {
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

    for _, p := range lb.ProcessPack {
        if p.IsAlive() {
            p.Current += p.Weight
        }
    }

    selected.Current -= lb.TotalWeight
    return selected
}
```

#### Example Distribution

With backend servers weighted 5:3:2:

| Request | Selected Backend | Distribution |
|---------|------------------|--------------|
| 1-5     | Server 1         | 50%          |
| 6-8     | Server 2         | 30%          |
| 9-10    | Server 3         | 20%          |

#### Use Cases

- When backend servers have different processing capacities
- For controlled, predictable traffic distribution
- When you want to prioritize certain backends
- When requests have similar processing times across all servers

#### Configuration Example

```
upstream backend {
    method weighted_round_robin;
    server http://backend1:80 weight=5;
    server http://backend2:80 weight=3;
    server http://backend3:80 weight=2;
}
```

### Least Connections

The Least Connections algorithm routes each request to the backend server with the fewest active connections.

#### Implementation Details

Our implementation tracks active connections for each backend and selects the one with the lowest count:

```go
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

    return lb.ProcessPack[selectedIndex]
}
```

Connection tracking is handled by incrementing a counter when a request starts and decrementing it when the response is sent:

```go
func (p *Process) IncrementConnections() {
    atomic.AddInt32(&p.ActiveConnections, 1)
}

func (p *Process) DecrementConnections() {
    atomic.AddInt32(&p.ActiveConnections, -1)
}
```

#### Use Cases

- When request processing times vary significantly
- For handling long-lived connections
- When backends may become overloaded by too many concurrent connections
- For workloads with varying response times

#### Configuration Example

```
upstream backend {
    method least_conn;
    server http://backend1:80 weight=1;
    server http://backend2:80 weight=1;
    server http://backend3:80 weight=1;
}
```

## Session Persistence Methods

Session persistence ensures that requests from the same client are consistently routed to the same backend server.

### Cookie-Based Persistence

Cookie-based persistence uses HTTP cookies to track which backend server a client should use.

#### Implementation Details

```go
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
```

When a request is received, the load balancer:
1. Checks for a cookie that identifies a specific backend
2. If found and valid, routes to that backend (if alive)
3. Otherwise, selects a backend using the configured load balancing algorithm
4. Sets a cookie in the response to identify the selected backend

#### Use Cases

- For applications that store session state on the server
- When clients support cookies
- For standard web applications

#### Configuration Example

```
upstream backend {
    method weighted_round_robin;
    persistence cookie;
    server http://backend1:80 weight=3;
    server http://backend2:80 weight=2;
    server http://backend3:80 weight=1;
}
```

### IP Hash Persistence

IP hash persistence uses the client's IP address to determine which backend server to use.

#### Implementation Details

```go
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
```

The load balancer:
1. Extracts the client IP from the request (checking X-Forwarded-For header first)
2. Looks up the IP in a map to find the assigned backend
3. If found and the backend is alive, routes to that backend
4. Otherwise, selects a backend using the configured algorithm and stores the mapping

#### Use Cases

- When clients don't support cookies
- For applications accessed by clients behind shared proxies
- For API services

#### Configuration Example

```
upstream backend {
    method least_conn;
    persistence ip_hash;
    server http://backend1:80 weight=1;
    server http://backend2:80 weight=1;
    server http://backend3:80 weight=1;
}
```

### Consistent Hash Persistence

Consistent hashing creates a hash ring where backend servers are placed at different points, with request paths mapped to the closest server.

#### Implementation Details

```go
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
    
    return ch.ring[ch.sortedHashes[idx]]
}
```

Key features of the implementation:
1. Each backend server is assigned multiple virtual nodes on the hash ring
2. Higher-weight servers get proportionally more virtual nodes
3. Requests are routed to the closest server on the ring
4. When servers are added or removed, only a portion of the requests are redistributed

#### Use Cases

- For distributed caching systems
- When backend servers are frequently added or removed
- For large-scale deployments

#### Configuration Example

```
upstream backend {
    method weighted_round_robin;
    persistence consistent_hash;
    server http://backend1:80 weight=3;
    server http://backend2:80 weight=2;
    server http://backend3:80 weight=1;
}
```

## Selecting an Algorithm and Persistence Method

You can select the algorithm and persistence method to use in two ways:

1. **In the configuration file** (preferred):
   ```
   upstream backend {
       method least_conn;
       persistence cookie;
       server http://backend1:80 weight=1;
       ...
   }
   ```

2. **Via command line**:
   ```bash
   ./load-balancer --algorithm=least-connections --persistence=ip_hash
   ```

The command line options override the methods specified in the configuration file.

## Extending the Architecture

The load balancer uses a strategy pattern that makes it easy to add new algorithms and persistence methods. To implement a new algorithm or persistence method:

1. Create a new struct that implements the required interface
2. Implement the required methods
3. Add your algorithm/method to the factory function
4. Update the config parser to recognize your algorithm/method name

Example:

```go
// Add to interface.go
const (
    MyCustomPersistence PersistenceMethod = "my-custom"
)

// In config.go
case "my_custom":
    config.Persistence = MyCustomPersistence
``` 