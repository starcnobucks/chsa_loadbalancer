package hashring

import (
	"sync"

	"github.com/stathat/consistent"
)

// Ring wraps the stathat/consistent hash ring with thread-safe operations.
type Ring struct {
	mu   sync.RWMutex
	ring *consistent.Consistent
}

// NewRing creates a new consistent hash ring with the given number of virtual nodes.
func NewRing(virtualNodes int) *Ring {
	c := consistent.New()
	c.NumberOfReplicas = virtualNodes
	return &Ring{ring: c}
}

// Add adds a member (backend address) to the ring.
func (r *Ring) Add(member string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.ring.Add(member)
}

// Remove removes a member from the ring.
func (r *Ring) Remove(member string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.ring.Remove(member)
}

// GetNode returns the single closest member for the given key.
func (r *Ring) GetNode(key string) (string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.ring.Get(key)
}

// GetNodes returns up to n closest members for the given key, walking the ring clockwise.
// This is used for congestion-aware fallback traversal.
func (r *Ring) GetNodes(key string, n int) ([]string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.ring.GetN(key, n)
}

// Members returns all current members in the ring.
func (r *Ring) Members() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.ring.Members()
}
