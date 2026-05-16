package backend

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"
	"sync/atomic"
	"time"

	"chca-loadbalancer/config"
	"chca-loadbalancer/logger"
)

// Backend represents a single upstream server with live load metrics.
type Backend struct {
	URL            *url.URL
	Weight         int
	Proxy          *httputil.ReverseProxy
	ActiveConns    atomic.Int64
	GPUUtilization atomic.Value // stores float64
	CPUUtilization atomic.Value // stores float64
	Healthy        atomic.Bool
	mu             sync.RWMutex
}

// NewBackend creates a Backend from a config entry.
func NewBackend(bc config.BackendConfig) (*Backend, error) {
	u, err := url.Parse(bc.URL)
	if err != nil {
		return nil, fmt.Errorf("invalid backend URL %q: %w", bc.URL, err)
	}

	proxy := httputil.NewSingleHostReverseProxy(u)
	proxy.Transport = &http.Transport{
		MaxIdleConns:        256,
		MaxIdleConnsPerHost: 64,
		MaxConnsPerHost:     0, // unlimited
		IdleConnTimeout:     90 * time.Second,
	}

	b := &Backend{
		URL:    u,
		Weight: bc.Weight,
		Proxy:  proxy,
	}
	b.Healthy.Store(true)
	b.GPUUtilization.Store(float64(0))
	b.CPUUtilization.Store(float64(0))

	return b, nil
}

// Address returns the host string used as the hash ring key.
func (b *Backend) Address() string {
	return b.URL.String()
}

// IncrementConn atomically increments the active connection counter.
func (b *Backend) IncrementConn() {
	b.ActiveConns.Add(1)
}

// DecrementConn atomically decrements the active connection counter.
func (b *Backend) DecrementConn() {
	b.ActiveConns.Add(-1)
}

// IsHealthy returns the current health status.
func (b *Backend) IsHealthy() bool {
	return b.Healthy.Load()
}

// IsOverloaded returns true if the backend exceeds any congestion threshold.
func (b *Backend) IsOverloaded(cfg config.CongestionConfig) bool {
	if int(b.ActiveConns.Load()) >= cfg.MaxConnections {
		return true
	}
	if b.GPUUtilization.Load().(float64) >= cfg.MaxGPUUtilization {
		return true
	}
	if b.CPUUtilization.Load().(float64) >= cfg.MaxCPUUtilization {
		return true
	}
	return false
}

// GetLoad returns the current active connection count (used for least-loaded fallback).
func (b *Backend) GetLoad() int64 {
	return b.ActiveConns.Load()
}

// MetricsResponse represents the JSON payload from a backend's /metrics endpoint.
type MetricsResponse struct {
	GPUUtilization float64 `json:"gpu_utilization"`
	CPUUtilization float64 `json:"cpu_utilization"`
}

// FetchMetrics polls the backend's /metrics endpoint and updates utilization values.
func (b *Backend) FetchMetrics(client *http.Client) {
	metricsURL := fmt.Sprintf("%s/metrics", b.URL.String())
	resp, err := client.Get(metricsURL)
	if err != nil {
		logger.Log.WithField("backend", b.Address()).
			WithError(err).Warn("failed to fetch metrics")
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return
	}

	var mr MetricsResponse
	if err := json.NewDecoder(resp.Body).Decode(&mr); err != nil {
		logger.Log.WithField("backend", b.Address()).
			WithError(err).Warn("failed to decode metrics")
		return
	}

	b.GPUUtilization.Store(mr.GPUUtilization)
	b.CPUUtilization.Store(mr.CPUUtilization)
}
