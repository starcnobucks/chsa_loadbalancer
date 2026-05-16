package hashring

import (
	"fmt"
	"math"
	"testing"
)

func TestRingDistribution(t *testing.T) {
	ring := NewRing(150)

	backends := []string{
		"http://localhost:8001",
		"http://localhost:8002",
		"http://localhost:8003",
	}
	for _, b := range backends {
		ring.Add(b)
	}

	// Distribute 3000 keys and check roughly even distribution (±30%)
	counts := make(map[string]int)
	numKeys := 3000
	for i := 0; i < numKeys; i++ {
		key := fmt.Sprintf("session-%d", i)
		node, err := ring.GetNode(key)
		if err != nil {
			t.Fatalf("GetNode(%q) error: %v", key, err)
		}
		counts[node]++
	}

	expected := float64(numKeys) / float64(len(backends))
	tolerance := 0.30
	for _, b := range backends {
		c := float64(counts[b])
		deviation := math.Abs(c-expected) / expected
		if deviation > tolerance {
			t.Errorf("backend %s got %d keys (expected ~%.0f, deviation %.1f%%)",
				b, counts[b], expected, deviation*100)
		}
	}
	t.Logf("Distribution: %v", counts)
}

func TestRingConsistency(t *testing.T) {
	ring := NewRing(150)
	backends := []string{
		"http://localhost:8001",
		"http://localhost:8002",
		"http://localhost:8003",
	}
	for _, b := range backends {
		ring.Add(b)
	}

	// Record initial mapping
	key := "stable-session-key"
	initial, _ := ring.GetNode(key)

	// Same key should always map to same node
	for i := 0; i < 100; i++ {
		node, _ := ring.GetNode(key)
		if node != initial {
			t.Fatalf("key %q mapped to %s, expected %s", key, node, initial)
		}
	}
}

func TestRingRemoveStability(t *testing.T) {
	ring := NewRing(150)
	backends := []string{
		"http://localhost:8001",
		"http://localhost:8002",
		"http://localhost:8003",
	}
	for _, b := range backends {
		ring.Add(b)
	}

	// Record mappings before removal
	numKeys := 1000
	before := make(map[string]string)
	for i := 0; i < numKeys; i++ {
		key := fmt.Sprintf("key-%d", i)
		node, _ := ring.GetNode(key)
		before[key] = node
	}

	// Remove one backend
	removed := "http://localhost:8002"
	ring.Remove(removed)

	// Keys that were on remaining backends should stay on the same backend
	movedCount := 0
	for i := 0; i < numKeys; i++ {
		key := fmt.Sprintf("key-%d", i)
		node, _ := ring.GetNode(key)
		if before[key] != removed && node != before[key] {
			movedCount++
		}
	}

	// At most a small fraction of non-removed keys should move
	maxAllowedMoves := numKeys / 10
	if movedCount > maxAllowedMoves {
		t.Errorf("too many keys moved after removal: %d (max allowed %d)", movedCount, maxAllowedMoves)
	}
	t.Logf("Keys moved from surviving backends: %d/%d", movedCount, numKeys)
}

func TestRingGetNodes(t *testing.T) {
	ring := NewRing(150)
	backends := []string{
		"http://localhost:8001",
		"http://localhost:8002",
		"http://localhost:8003",
	}
	for _, b := range backends {
		ring.Add(b)
	}

	nodes, err := ring.GetNodes("test-key", 3)
	if err != nil {
		t.Fatalf("GetNodes error: %v", err)
	}
	if len(nodes) != 3 {
		t.Fatalf("expected 3 nodes, got %d", len(nodes))
	}

	// All three backends should appear exactly once
	seen := make(map[string]bool)
	for _, n := range nodes {
		if seen[n] {
			t.Errorf("duplicate node in GetNodes result: %s", n)
		}
		seen[n] = true
	}
}
