# CHCA Load Balancer

A **Congestion-Aware Consistent Hashing** load balancer built in Go. Requests with the same session key are consistently routed to the same backend. When a backend becomes overloaded, it is automatically skipped and the next healthy server on the hash ring is selected.

## Features

- **Consistent Hashing** — same `X-Session-ID` always routes to the same backend (via [stathat/consistent](https://github.com/stathat/consistent))
- **Congestion Avoidance** — overloaded backends are temporarily skipped, not permanently removed
- **Least-Loaded Fallback** — when all backends are overloaded, picks the one with fewest active connections
- **Health Checking** — background goroutine pings `/health` on each backend; unhealthy servers are removed from the ring
- **Load Monitoring** — polls backend `/metrics` for GPU/CPU utilization with configurable thresholds
- **Prometheus Metrics** — exposes request counts, latency histograms, congestion skips, and more on `/metrics`
- **Structured Logging** — JSON-formatted logs via [Logrus](https://github.com/sirupsen/logrus)
- **Graceful Shutdown** — handles `SIGINT`/`SIGTERM` cleanly

## Project Structure

```
chca-loadbalancer/
├── main.go                  # Entry point, HTTP server, graceful shutdown
├── config/config.go         # YAML configuration loading (Viper)
├── logger/logger.go         # Structured JSON logging (Logrus)
├── backend/
│   ├── backend.go           # Backend model with atomic load counters
│   └── monitor.go           # Background health checks & load polling
├── hashring/ring.go         # Thread-safe consistent hash ring wrapper
├── balancer/chca.go         # Core CHCA routing algorithm
├── metrics/metrics.go       # Prometheus metric definitions
├── cmd/backends/main.go     # Mock backend servers for testing
└── config.yaml              # Default configuration
```

## Quick Start

### Prerequisites

- Go 1.24+

### Install & Run

```bash
# Clone the repo
git clone https://github.com/YOUR_USERNAME/chca-loadbalancer.git
cd chca-loadbalancer

# Install dependencies
go mod tidy

# Start 3 mock backends (ports 8001, 8002, 8003)
go run cmd/backends/main.go &

# Start the load balancer
go run main.go
```

### Test It

```bash
# Health check
curl http://localhost:9090/healthz

# Route a request with a session key
curl -H "X-Session-ID: user-123" http://localhost:9090/

# Same session always hits the same backend
curl -H "X-Session-ID: user-123" http://localhost:9090/
curl -H "X-Session-ID: user-123" http://localhost:9090/

# View Prometheus metrics
curl http://localhost:9090/metrics
```

### Run Tests

```bash
go test ./... -v
```

## Configuration

Edit `config.yaml` to customize:

```yaml
server:
  port: 9090

backends:
  - url: "http://localhost:8001"
    weight: 1
  - url: "http://localhost:8002"
    weight: 1
  - url: "http://localhost:8003"
    weight: 1

ring:
  virtual_nodes: 150

congestion:
  max_connections: 100
  max_gpu_utilization: 90.0
  max_cpu_utilization: 80.0
  monitor_interval_seconds: 5

health:
  check_interval_seconds: 5
  timeout_seconds: 2
  endpoint: "/health"

logging:
  level: "info"
```

## How It Works

```
                    ┌─────────────────────┐
  Request ──────►   │   CHCA Load Balancer │
  X-Session-ID      │                     │
                    │  1. Hash session key │
                    │  2. Find primary node│
                    │  3. Check congestion │
                    │  4. Skip if overloaded│
                    │  5. Route to healthy │
                    └─────┬───┬───┬───────┘
                          │   │   │
                ┌─────────┘   │   └─────────┐
                ▼             ▼             ▼
          ┌──────────┐ ┌──────────┐ ┌──────────┐
          │ Backend 1│ │ Backend 2│ │ Backend 3│
          │  :8001   │ │  :8002   │ │  :8003   │
          └──────────┘ └──────────┘ └──────────┘
```

## Key Metrics (Prometheus)

| Metric | Type | Description |
|--------|------|-------------|
| `chca_requests_total` | Counter | Total requests per backend |
| `chca_active_connections` | Gauge | Live connections per backend |
| `chca_request_duration_seconds` | Histogram | Request latency per backend |
| `chca_congestion_skips_total` | Counter | Times a congested backend was skipped |
| `chca_healthy_backends` | Gauge | Number of healthy backends |
| `chca_fallback_events_total` | Counter | Requests that used least-loaded fallback |

## License

MIT
