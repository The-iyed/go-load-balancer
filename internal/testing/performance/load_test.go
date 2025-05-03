package performance

import (
	"fmt"
	"math"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/The-iyed/go-load-balancer/internal/balancer"
	"github.com/The-iyed/go-load-balancer/internal/testing/mocks"
)

// BenchmarkLoadBalancer measures load balancer performance under various conditions
func BenchmarkLoadBalancer(b *testing.B) {
	tests := []struct {
		name          string
		algorithm     balancer.LoadBalancerAlgorithm
		persistence   balancer.PersistenceMethod
		numBackends   int
		backendDelay  time.Duration
		concurrency   int
		requestsPerGo int
	}{
		{
			name:          "WeightedRoundRobin-NoPersistence-NoDelay",
			algorithm:     balancer.WeightedRoundRobin,
			persistence:   balancer.NoPersistence,
			numBackends:   3,
			backendDelay:  0,
			concurrency:   10,
			requestsPerGo: 100,
		},
		{
			name:          "LeastConnections-NoPersistence-NoDelay",
			algorithm:     balancer.LeastConnections,
			persistence:   balancer.NoPersistence,
			numBackends:   3,
			backendDelay:  0,
			concurrency:   10,
			requestsPerGo: 100,
		},
		{
			name:          "WeightedRoundRobin-CookiePersistence-NoDelay",
			algorithm:     balancer.WeightedRoundRobin,
			persistence:   balancer.CookiePersistence,
			numBackends:   3,
			backendDelay:  0,
			concurrency:   10,
			requestsPerGo: 100,
		},
		{
			name:          "WeightedRoundRobin-IPHashPersistence-NoDelay",
			algorithm:     balancer.WeightedRoundRobin,
			persistence:   balancer.IPHashPersistence,
			numBackends:   3,
			backendDelay:  0,
			concurrency:   10,
			requestsPerGo: 100,
		},
		{
			name:          "WeightedRoundRobin-NoDelay-HighConcurrency",
			algorithm:     balancer.WeightedRoundRobin,
			persistence:   balancer.NoPersistence,
			numBackends:   3,
			backendDelay:  0,
			concurrency:   50,
			requestsPerGo: 100,
		},
		{
			name:          "WeightedRoundRobin-WithDelay-MediumConcurrency",
			algorithm:     balancer.WeightedRoundRobin,
			persistence:   balancer.NoPersistence,
			numBackends:   3,
			backendDelay:  20 * time.Millisecond,
			concurrency:   20,
			requestsPerGo: 50,
		},
	}

	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			// Reset the timer for setup
			b.StopTimer()

			// Create mock backends with the specified delay
			delays := make([]time.Duration, tt.numBackends)
			for i := range delays {
				delays[i] = tt.backendDelay
			}
			cluster := mocks.NewBackendCluster(tt.numBackends, delays, nil)
			defer cluster.Close()

			// Create a load balancer client
			client := mocks.NewLoadBalancerTestClient()
			defer client.Close()

			// Initialize with the algorithm and persistence method
			weights := make([]int, tt.numBackends)
			for i := range weights {
				weights[i] = 1
			}
			err := client.InitializeWithBackends(
				tt.algorithm,
				tt.persistence,
				cluster.URLs(),
				weights,
			)
			if err != nil {
				b.Fatalf("Failed to initialize load balancer: %v", err)
			}

			// Prepare benchmarking
			totalRequests := tt.concurrency * tt.requestsPerGo
			b.ResetTimer()

			// Start the timer for the benchmark
			b.StartTimer()

			// Use a WaitGroup to wait for all goroutines to finish
			var wg sync.WaitGroup
			wg.Add(tt.concurrency)

			// Track response times
			responseTimes := make([]time.Duration, totalRequests)
			var responseTimeMutex sync.Mutex

			// Launch concurrent goroutines to simulate load
			for i := 0; i < tt.concurrency; i++ {
				go func(routineID int) {
					defer wg.Done()

					// Get a cookie for persistence tests
					var cookie *http.Cookie
					if tt.persistence == balancer.CookiePersistence {
						resp, err := client.SendRequest("/", nil)
						if err != nil {
							b.Logf("Initial request failed: %v", err)
							return
						}
						for _, c := range resp.Cookies() {
							if c.Name == "GOLB_SESSION" {
								cookie = c
								break
							}
						}
					}

					var cookies []*http.Cookie
					if cookie != nil {
						cookies = []*http.Cookie{cookie}
					}

					// Send requests
					for j := 0; j < tt.requestsPerGo; j++ {
						requestID := routineID*tt.requestsPerGo + j
						start := time.Now()

						_, err := client.SendRequest("/", cookies)

						duration := time.Since(start)

						if err != nil {
							b.Logf("Request failed: %v", err)
							continue
						}

						// Record response time
						responseTimeMutex.Lock()
						responseTimes[requestID] = duration
						responseTimeMutex.Unlock()
					}
				}(i)
			}

			// Wait for all goroutines to finish
			wg.Wait()

			// Stop the timer to exclude result processing
			b.StopTimer()

			// Calculate statistics
			var totalTime time.Duration
			var minTime = time.Hour
			var maxTime time.Duration
			var validResponses int

			for _, duration := range responseTimes {
				if duration > 0 {
					totalTime += duration
					validResponses++
					if duration < minTime {
						minTime = duration
					}
					if duration > maxTime {
						maxTime = duration
					}
				}
			}

			if validResponses == 0 {
				b.Fatalf("No valid responses received")
			}

			avgTime := totalTime / time.Duration(validResponses)

			// Calculate standard deviation
			var variance float64
			for _, duration := range responseTimes {
				if duration > 0 {
					diff := float64(duration - avgTime)
					variance += diff * diff
				}
			}
			variance /= float64(validResponses)
			stdDev := time.Duration(math.Sqrt(variance))

			// Calculate requests per second
			rps := float64(validResponses) / totalTime.Seconds()

			// Report results
			b.ReportMetric(rps, "req/s")
			b.ReportMetric(float64(avgTime.Microseconds()), "avg_µs")
			b.ReportMetric(float64(stdDev.Microseconds()), "stddev_µs")
			b.ReportMetric(float64(minTime.Microseconds()), "min_µs")
			b.ReportMetric(float64(maxTime.Microseconds()), "max_µs")

			// Display backend request distribution
			counts := cluster.GetBackendRequestCounts()
			distribution := cluster.RequestDistribution()

			fmt.Printf("Backend distribution for %s:\n", tt.name)
			for i, count := range counts {
				fmt.Printf("  Backend %d: %d requests (%.1f%%)\n", i+1, count, distribution[i])
			}
		})
	}
}

// BenchmarkHighLoad tests the load balancer under very high concurrency
func BenchmarkHighLoad(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping high load test in short mode")
	}

	// Create mock backends
	numBackends := 5
	cluster := mocks.NewBackendCluster(numBackends, nil, nil)
	defer cluster.Close()

	// Create a load balancer client
	client := mocks.NewLoadBalancerTestClient()
	defer client.Close()

	// Initialize with weighted round robin
	weights := make([]int, numBackends)
	for i := range weights {
		weights[i] = 1
	}
	err := client.InitializeWithBackends(
		balancer.WeightedRoundRobin,
		balancer.NoPersistence,
		cluster.URLs(),
		weights,
	)
	if err != nil {
		b.Fatalf("Failed to initialize load balancer: %v", err)
	}

	// Benchmark with increasing concurrency levels
	concurrencyLevels := []int{10, 50, 100, 200, 500}
	requestsPerGoroutine := 20

	for _, concurrency := range concurrencyLevels {
		name := fmt.Sprintf("Concurrency-%d", concurrency)
		b.Run(name, func(b *testing.B) {
			// Reset the timer for setup
			b.StopTimer()
			cluster.ResetStats()

			// Prepare benchmarking
			totalRequests := concurrency * requestsPerGoroutine
			responseTimes := make([]time.Duration, totalRequests)
			var responseTimeMutex sync.Mutex

			b.ResetTimer()
			b.StartTimer()

			// Use a WaitGroup to wait for all goroutines to finish
			var wg sync.WaitGroup
			wg.Add(concurrency)

			// Launch concurrent goroutines to simulate load
			for i := 0; i < concurrency; i++ {
				go func(routineID int) {
					defer wg.Done()

					for j := 0; j < requestsPerGoroutine; j++ {
						requestID := routineID*requestsPerGoroutine + j
						start := time.Now()

						_, err := client.SendRequest("/", nil)

						duration := time.Since(start)

						if err != nil {
							b.Logf("Request failed: %v", err)
							continue
						}

						// Record response time
						responseTimeMutex.Lock()
						responseTimes[requestID] = duration
						responseTimeMutex.Unlock()
					}
				}(i)
			}

			// Wait for all goroutines to finish
			wg.Wait()

			// Stop the timer to exclude result processing
			b.StopTimer()

			// Calculate statistics
			var totalTime time.Duration
			var validResponses int

			for _, duration := range responseTimes {
				if duration > 0 {
					totalTime += duration
					validResponses++
				}
			}

			if validResponses == 0 {
				b.Fatalf("No valid responses received")
			}

			avgTime := totalTime / time.Duration(validResponses)
			rps := float64(validResponses) / totalTime.Seconds()

			// Report results
			b.ReportMetric(rps, "req/s")
			b.ReportMetric(float64(avgTime.Microseconds()), "avg_µs")

			// Display success rate
			successRate := float64(validResponses) / float64(totalRequests) * 100
			b.ReportMetric(successRate, "success_%")
		})
	}
}
