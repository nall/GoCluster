package reputation

import (
	"strings"
	"sync"
	"time"
)

// prefixLimiter is a sharded token bucket for source-network admission. It
// protects the cluster from prefix-level bursts before those requests fan into
// per-call reputation state.
type prefixLimiter struct {
	capacity     int
	refillPerSec int
	ttl          time.Duration
	maxEntries   int
	shards       []prefixShard
}

type prefixShard struct {
	mu    sync.Mutex
	items map[string]*prefixState
}

type prefixState struct {
	tokens     int
	lastRefill time.Time
	lastSeen   time.Time
}

func newPrefixLimiter(capacity, refillPerSec int, ttl time.Duration, maxEntries int) *prefixLimiter {
	if capacity <= 0 || refillPerSec <= 0 {
		return nil
	}
	if maxEntries <= 0 {
		maxEntries = 200000
	}
	shards := 16
	pl := &prefixLimiter{
		capacity:     capacity,
		refillPerSec: refillPerSec,
		ttl:          ttl,
		maxEntries:   maxEntries,
		shards:       make([]prefixShard, shards),
	}
	for i := range pl.shards {
		pl.shards[i].items = make(map[string]*prefixState)
	}
	return pl
}

// allow spends one token for the prefix. Empty or disabled prefixes pass because
// reputation should not punish missing IP evidence more than the explicit
// unknown-IP penalties already do.
func (p *prefixLimiter) allow(prefix string, now time.Time) bool {
	if p == nil {
		return true
	}
	prefix = strings.TrimSpace(prefix)
	if prefix == "" {
		return true
	}
	shard := &p.shards[p.shardIndex(prefix)]
	shard.mu.Lock()
	defer shard.mu.Unlock()

	state := shard.items[prefix]
	if state == nil || (p.ttl > 0 && now.Sub(state.lastSeen) > p.ttl) {
		state = &prefixState{
			tokens:     p.capacity,
			lastRefill: now,
			lastSeen:   now,
		}
		shard.items[prefix] = state
	}
	p.refill(state, now)
	state.lastSeen = now
	if state.tokens <= 0 {
		return false
	}
	state.tokens--
	return true
}

// refill advances whole-second token increments so refill math stays stable and
// bounded even if checks arrive at sub-second cadence.
func (p *prefixLimiter) refill(state *prefixState, now time.Time) {
	if state == nil {
		return
	}
	if now.Before(state.lastRefill) {
		state.lastRefill = now
		return
	}
	elapsed := now.Sub(state.lastRefill)
	if elapsed < time.Second {
		return
	}
	add := int(elapsed/time.Second) * p.refillPerSec
	if add <= 0 {
		return
	}
	state.tokens += add
	if state.tokens > p.capacity {
		state.tokens = p.capacity
	}
	state.lastRefill = state.lastRefill.Add(time.Duration(add/p.refillPerSec) * time.Second)
}

// sweep removes stale prefix states and applies the hard shard cap. The cap
// bounds reconnect-storm memory even when many prefixes appear only once.
func (p *prefixLimiter) sweep(now time.Time) {
	if p == nil || p.ttl <= 0 {
		return
	}
	for i := range p.shards {
		shard := &p.shards[i]
		shard.mu.Lock()
		for key, state := range shard.items {
			if state == nil || now.Sub(state.lastSeen) > p.ttl {
				delete(shard.items, key)
			}
		}
		if p.maxEntries > 0 && len(shard.items) > p.maxEntries {
			for key := range shard.items {
				delete(shard.items, key)
				if len(shard.items) <= p.maxEntries {
					break
				}
			}
		}
		shard.mu.Unlock()
	}
}

func (p *prefixLimiter) shardIndex(key string) int {
	hash := fnv32a(key)
	return int(hash % uint32(len(p.shards)))
}
