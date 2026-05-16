package backend

import (
	"context"
	"net/http"
	"time"

	"chca-loadbalancer/config"
	"chca-loadbalancer/hashring"
	"chca-loadbalancer/logger"
	"chca-loadbalancer/metrics"
)

// Monitor continuously checks backend health and load in background goroutines.
type Monitor struct {
	Backends []*Backend
	Ring     *hashring.Ring
	Config   *config.Config
	client   *http.Client
}

// NewMonitor creates a Monitor for the given backends and ring.
func NewMonitor(backends []*Backend, ring *hashring.Ring, cfg *config.Config) *Monitor {
	return &Monitor{
		Backends: backends,
		Ring:     ring,
		Config:   cfg,
		client: &http.Client{
			Timeout: time.Duration(cfg.Health.TimeoutSec) * time.Second,
		},
	}
}

// Start launches background goroutines for load monitoring and health checking.
// Both goroutines stop when the context is cancelled.
func (m *Monitor) Start(ctx context.Context) {
	go m.monitorLoad(ctx)
	go m.checkHealth(ctx)
}

// monitorLoad periodically fetches utilization metrics from each backend.
func (m *Monitor) monitorLoad(ctx context.Context) {
	interval := time.Duration(m.Config.Congestion.MonitorIntervalSec) * time.Second
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	logger.Log.Info("load monitor started")
	for {
		select {
		case <-ctx.Done():
			logger.Log.Info("load monitor stopped")
			return
		case <-ticker.C:
			for _, b := range m.Backends {
				if b.IsHealthy() {
					b.FetchMetrics(m.client)
					metrics.ActiveConnections.WithLabelValues(b.Address()).
						Set(float64(b.ActiveConns.Load()))
				}
			}
		}
	}
}

// checkHealth periodically pings each backend's health endpoint and updates ring membership.
func (m *Monitor) checkHealth(ctx context.Context) {
	interval := time.Duration(m.Config.Health.CheckIntervalSec) * time.Second
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	logger.Log.Info("health checker started")
	for {
		select {
		case <-ctx.Done():
			logger.Log.Info("health checker stopped")
			return
		case <-ticker.C:
			healthyCount := 0
			for _, b := range m.Backends {
				healthy := m.ping(b)
				wasHealthy := b.IsHealthy()

				if healthy && !wasHealthy {
					b.Healthy.Store(true)
					m.Ring.Add(b.Address())
					logger.Log.WithField("backend", b.Address()).Info("backend became healthy, added to ring")
				} else if !healthy && wasHealthy {
					b.Healthy.Store(false)
					m.Ring.Remove(b.Address())
					logger.Log.WithField("backend", b.Address()).Warn("backend became unhealthy, removed from ring")
				}

				if healthy {
					healthyCount++
				}
			}
			metrics.HealthyBackends.Set(float64(healthyCount))
		}
	}
}

// ping sends a GET request to the backend's health endpoint.
func (m *Monitor) ping(b *Backend) bool {
	url := b.URL.String() + m.Config.Health.Endpoint
	resp, err := m.client.Get(url)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}
