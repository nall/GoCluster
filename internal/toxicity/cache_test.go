package toxicity

import (
	"testing"
	"time"

	"dxcluster/spot"
)

func TestCacheHitTTLAndEviction(t *testing.T) {
	now := time.Unix(100, 0)
	cache := NewCache(1, time.Minute)
	cache.Put("cq test", Decision{Status: spot.ToxicitySafe}, now)
	if _, ok := cache.Get("cq test", now.Add(30*time.Second)); !ok {
		t.Fatalf("expected cache hit before TTL")
	}
	if _, ok := cache.Get("cq test", now.Add(2*time.Minute)); ok {
		t.Fatalf("expected cache miss after TTL")
	}

	cache.Put("one", Decision{Status: spot.ToxicitySafe}, now)
	cache.Put("two", Decision{Status: spot.ToxicityToxic}, now)
	if cache.Len() != 1 || cache.Evictions() == 0 {
		t.Fatalf("expected hard-cap eviction, len=%d evictions=%d", cache.Len(), cache.Evictions())
	}
	if _, ok := cache.Get("one", now); ok {
		t.Fatalf("expected oldest entry to be evicted")
	}
}
