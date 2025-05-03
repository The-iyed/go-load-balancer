# Load Balancing Algorithms

This document provides detailed information about the load balancing algorithms implemented in this project.

## Weighted Round Robin

The Weighted Round Robin algorithm distributes requests across backend servers proportionally based on their assigned weights.

### Implementation Details

Our implementation uses a dynamic weight adjustment mechanism based on the highest effective weight approach:

```go
func (lb *WeightedRoundRobinBalancer) GetNextInstance() *Process {
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

### Example Distribution

With backend servers weighted 5:3:2:

| Request | Selected Backend | Distribution |
|---------|------------------|--------------|
| 1-5     | Server 1         | 50%          |
| 6-8     | Server 2         | 30%          |
| 9-10    | Server 3         | 20%          |

### Use Cases

- When backend servers have different processing capacities
- For controlled, predictable traffic distribution
- When you want to prioritize certain backends
- When requests have similar processing times across all servers

### Configuration Example

```
upstream backend {
    method weighted_round_robin;
    server http://backend1:80 weight=5;
    server http://backend2:80 weight=3;
    server http://backend3:80 weight=2;
}
```

## Least Connections

The Least Connections algorithm routes each request to the backend server with the fewest active connections.

### Implementation Details

Our implementation tracks active connections for each backend and selects the one with the lowest count:

```go
func (lb *LeastConnectionsBalancer) GetNextInstance() *Process {
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

### Use Cases

- When request processing times vary significantly
- For handling long-lived connections
- When backends may become overloaded by too many concurrent connections
- For workloads with varying response times

### Configuration Example

```
upstream backend {
    method least_conn;
    server http://backend1:80 weight=1;
    server http://backend2:80 weight=1;
    server http://backend3:80 weight=1;
}
```

### Using Weights with Least Connections

The Least Connections algorithm can also consider weights as a tie-breaker when multiple backends have the same number of connections:

```
upstream backend {
    method least_conn;
    server http://backend1:80 weight=4;
    server http://backend2:80 weight=2;
    server http://backend3:80 weight=1;
}
```

In this configuration, if multiple servers have the same number of active connections, the one with the higher weight will be selected.

## Comparison

| Factor | Weighted Round Robin | Least Connections |
|--------|----------------------|-------------------|
| Distribution | Based on static weights | Dynamic based on current load |
| Adaptability | Doesn't adapt to backend load | Adapts based on connection count |
| Complexity | Simple implementation | Requires connection tracking |
| Overhead | Lower | Slightly higher |
| Best for | Predictable, similar-duration requests | Varying request durations |
| Backend differences | Handles different capacities via weights | Auto-balances based on load |

## Selecting an Algorithm

You can select the algorithm to use in two ways:

1. **In the configuration file** (preferred):
   ```
   upstream backend {
       method least_conn;  # or weighted_round_robin, round_robin
       server http://backend1:80 weight=1;
       ...
   }
   ```

2. **Via command line**:
   ```bash
   ./load-balancer --algorithm=least-connections
   ```

The command line option overrides the method specified in the configuration file.

## Extending the Architecture

The load balancer uses a strategy pattern that makes it easy to add new algorithms. To implement a new algorithm:

1. Create a new struct that implements the `LoadBalancerStrategy` interface
2. Implement the required methods: `GetNextInstance()` and `ProxyRequest()`
3. Add your algorithm to the factory function in `interface.go`
4. Update the config parser to recognize your algorithm's name

Example:

```go
// In your new algorithm file
type MyCustomBalancer struct {
    ProcessPack []*Process
}

func NewMyCustomBalancer(configs []BackendConfig) *MyCustomBalancer {
    // Initialize your balancer
}

func (lb *MyCustomBalancer) GetNextInstance() *Process {
    // Your selection logic
}

func (lb *MyCustomBalancer) ProxyRequest(w http.ResponseWriter, r *http.Request) {
    // Your proxying logic
}

// In interface.go
const (
    // Add your algorithm constant
    MyCustomAlgorithm LoadBalancerAlgorithm = "my-custom"
)

// Update the factory function
func CreateLoadBalancer(algorithm LoadBalancerAlgorithm, configs []BackendConfig) LoadBalancerStrategy {
    switch algorithm {
    case MyCustomAlgorithm:
        return NewMyCustomBalancer(configs)
    // ... other cases
    }
}

// In config.go
case "my_custom":
    config.Method = MyCustomAlgorithm
``` 