# Go Load Balancer Testing Framework

<p align="center">
  <img src="../../docs/images/logo.png" alt="Go Load Balancer Logo" width="250">
</p>

This package provides a comprehensive testing framework for the Go Load Balancer, including unit tests, integration tests, performance benchmarks, and mocks.

## Testing Structure

The testing package is organized into the following directories:

- **testutils**: Common utilities and helper functions used across all test types
- **mocks**: Mock implementations of backends and other components
- **unit**: Unit tests for individual components
- **integration**: Integration tests that test multiple components together
- **performance**: Benchmarks and performance tests

## Running Tests

Use the provided test script to run all tests:

```bash
# Run all tests
./scripts/run_tests.sh

# Run only quick tests (skips integration and performance tests)
./scripts/run_tests.sh --short
```

Or run specific test categories:

```bash
# Run only unit tests
go test -v ./internal/testing/unit/...

# Run integration tests
go test -v ./internal/testing/integration/...

# Run performance benchmarks
go test -bench=. -benchmem ./internal/testing/performance/...
```

## Test Utilities

The `testutils` package provides common utilities used across tests:

- `CreateTempConfig`: Creates temporary config files for testing
- `CreateTestBackends`: Sets up mock backend servers
- `AssertEventually`: Polls a condition until it becomes true or times out
- `CookieFromResponse`: Extracts cookies from HTTP responses
- `ParseBackendResponse`: Parses backend ID from test responses

## Mock Components

The `mocks` package provides mock implementations:

- `MockBackend`: A mock backend server implementation
- `BackendCluster`: A cluster of mock backends with statistics tracking
- `LoadBalancerTestClient`: A client for testing the load balancer

## Unit Tests

Unit tests focus on testing individual components in isolation:

- Algorithm implementations (Weighted Round Robin, Least Connections)
- Session persistence mechanisms (Cookie, IP Hash, Consistent Hash)
- Configuration parsing

## Integration Tests

Integration tests validate that components work together correctly:

- End-to-end testing with the actual load balancer binary
- Backend failure handling and health checking
- Session persistence across multiple requests

## Performance Tests

Performance benchmarks measure the load balancer's performance:

- Various algorithms and persistence methods
- Different concurrency levels
- Backend server response time effects
- Request throughput and latency measurements

## Coverage Reports

Test coverage reports are generated when running the test script. An HTML report is created at `coverage.html` in the project root.

## Writing New Tests

When adding new tests:

1. Decide which test category is appropriate (unit, integration, performance)
2. Use the existing mock components and test utilities
3. Follow the existing patterns and conventions
4. Make sure your tests are deterministic and don't rely on external services
5. Add appropriate logging and error messages

For performance tests, make sure to:
- Use Go's benchmarking framework properly
- Measure both throughput (requests/second) and latency
- Test with various concurrency levels
- Reset timers appropriately during setup/teardown 