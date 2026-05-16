package balancer

import (
	"net/http"
	"time"

	"github.com/google/uuid"

	"chca-loadbalancer/backend"
	"chca-loadbalancer/config"
	"chca-loadbalancer/hashring"
	"chca-loadbalancer/logger"
	"chca-loadbalancer/metrics"
)

// CHCABalancer implements the Congestion-Aware Consistent Hashing algorithm
// and satisfies http.Handler.
type CHCABalancer struct {
	Ring     *hashring.Ring
	Backends map[string]*backend.Backend // keyed by address
	Config   *config.Config
}

// NewCHCABalancer creates a new load balancer from backends, ring, and config.
func NewCHCABalancer(backends []*backend.Backend, ring *hashring.Ring, cfg *config.Config) *CHCABalancer {
	bmap := make(map[string]*backend.Backend, len(backends))
	for _, b := range backends {
		bmap[b.Address()] = b
	}
	return &CHCABalancer{
		Ring:     ring,
		Backends: bmap,
		Config:   cfg,
	}
}

// ServeHTTP routes each request using the CHCA algorithm:
//  1. Extract the session key from X-Session-ID (or generate a UUID).
//  2. Get ordered candidates from the hash ring.
//  3. Pick the first healthy, non-overloaded candidate.
//  4. If all are overloaded, fall back to the least-loaded backend.
func (c *CHCABalancer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	// 1. Extract key
	key := r.Header.Get("X-Session-ID")
	if key == "" {
		key = uuid.New().String()
	}

	// 2. Get ordered candidates from ring (clockwise walk)
	candidates, err := c.Ring.GetNodes(key, len(c.Backends))
	if err != nil || len(candidates) == 0 {
		logger.Log.WithError(err).Error("no backends available in ring")
		http.Error(w, "no backends available", http.StatusServiceUnavailable)
		return
	}

	// 3. Find the first healthy, non-overloaded backend
	var selected *backend.Backend
	for _, addr := range candidates {
		b, ok := c.Backends[addr]
		if !ok {
			continue
		}
		if !b.IsHealthy() {
			continue
		}
		if b.IsOverloaded(c.Config.Congestion) {
			metrics.CongestionSkips.WithLabelValues(addr).Inc()
			logger.Log.WithField("backend", addr).
				WithField("session", key).
				Debug("skipping congested backend")
			continue
		}
		selected = b
		break
	}

	// 4. Fallback — all overloaded: pick the least-loaded healthy backend
	if selected == nil {
		metrics.FallbackEvents.Inc()
		var minLoad int64 = -1
		for _, addr := range candidates {
			b, ok := c.Backends[addr]
			if !ok || !b.IsHealthy() {
				continue
			}
			load := b.GetLoad()
			if minLoad < 0 || load < minLoad {
				minLoad = load
				selected = b
			}
		}
	}

	if selected == nil {
		logger.Log.Error("all backends unavailable")
		http.Error(w, "all backends unavailable", http.StatusServiceUnavailable)
		return
	}

	// 5. Route the request
	addr := selected.Address()
	selected.IncrementConn()
	metrics.ActiveConnections.WithLabelValues(addr).Inc()
	metrics.RequestsTotal.WithLabelValues(addr).Inc()

	logger.Log.WithField("session", key).
		WithField("backend", addr).
		Debug("routing request")

	selected.Proxy.ServeHTTP(w, r)

	selected.DecrementConn()
	metrics.ActiveConnections.WithLabelValues(addr).Dec()

	duration := time.Since(start).Seconds()
	metrics.RequestDuration.WithLabelValues(addr).Observe(duration)
}
