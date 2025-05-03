# Changelog

All notable changes to the Go Load Balancer project will be documented in this file.

## [1.1.0] - 2023-10-15

### Added
- Session persistence functionality with multiple methods:
  - Cookie-based persistence (`persistence cookie`)
  - IP-hash persistence (`persistence ip_hash`)
  - Consistent hash persistence (`persistence consistent_hash`)
- Configuration options for persistence methods in `loadbalancer.conf`
- Command-line flag `--persistence` to override the configuration file
- Documentation for session persistence in README and configuration docs
- New example configuration files for different persistence methods

### Changed
- Updated algorithm implementation to support session persistence
- Enhanced proxy logic to maintain client sessions with the same backend
- Improved documentation with detailed explanations of persistence methods
- Refactored load balancer code to separate configuration parsing from implementation

### Fixed
- Edge case with IP-based persistence behind multiple proxies
- Cookie validation to prevent tampering
- Backend selection when a preferred backend is down

## [1.0.0] - 2023-09-01

### Added
- Initial release of Go Load Balancer
- Support for multiple load balancing algorithms:
  - Round Robin
  - Weighted Round Robin
  - Least Connections
- Health checking of backend servers
- Simple configuration file format
- Command-line interface
- Logging with zap
- Docker support 