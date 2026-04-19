package telnet

import (
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// BulletinDedupeSnapshot reports the bounded telnet bulletin dedupe state.
type BulletinDedupeSnapshot struct {
	Enabled    bool
	Window     time.Duration
	MaxEntries int
	Tracked    int
	Accepted   uint64
	Suppressed uint64
	Evicted    uint64
}

type bulletinDedupeCache struct {
	mu         sync.Mutex
	window     time.Duration
	maxEntries int
	items      map[string]time.Time
	order      []string
	accepted   atomic.Uint64
	suppressed atomic.Uint64
	evicted    atomic.Uint64
}

func newBulletinDedupeCache(window time.Duration, maxEntries int) *bulletinDedupeCache {
	if window <= 0 || maxEntries <= 0 {
		return nil
	}
	return &bulletinDedupeCache{
		window:     window,
		maxEntries: maxEntries,
		items:      make(map[string]time.Time),
		order:      make([]string, 0, maxEntries),
	}
}

func (c *bulletinDedupeCache) allow(kind, message string, now time.Time) bool {
	if c == nil {
		return true
	}
	key := bulletinDedupeKey(kind, message)
	if key == "" {
		return true
	}
	if now.IsZero() {
		now = time.Now().UTC()
	} else {
		now = now.UTC()
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	c.pruneLocked(now)
	if _, ok := c.items[key]; ok {
		c.suppressed.Add(1)
		return false
	}
	for len(c.items) >= c.maxEntries && len(c.order) > 0 {
		victim := c.order[0]
		c.order = c.order[1:]
		if _, ok := c.items[victim]; ok {
			delete(c.items, victim)
			c.evicted.Add(1)
		}
	}
	if len(c.items) >= c.maxEntries {
		c.items = make(map[string]time.Time)
		c.order = c.order[:0]
	}
	c.items[key] = now
	c.order = append(c.order, key)
	c.accepted.Add(1)
	return true
}

func (c *bulletinDedupeCache) snapshot() BulletinDedupeSnapshot {
	if c == nil {
		return BulletinDedupeSnapshot{}
	}
	c.mu.Lock()
	tracked := len(c.items)
	c.mu.Unlock()
	return BulletinDedupeSnapshot{
		Enabled:    true,
		Window:     c.window,
		MaxEntries: c.maxEntries,
		Tracked:    tracked,
		Accepted:   c.accepted.Load(),
		Suppressed: c.suppressed.Load(),
		Evicted:    c.evicted.Load(),
	}
}

func (c *bulletinDedupeCache) pruneLocked(now time.Time) {
	if c == nil || len(c.items) == 0 {
		return
	}
	cutoff := now.Add(-c.window)
	kept := c.order[:0]
	for _, key := range c.order {
		seenAt, ok := c.items[key]
		if !ok {
			continue
		}
		if seenAt.Before(cutoff) {
			delete(c.items, key)
			continue
		}
		kept = append(kept, key)
	}
	c.order = kept
}

func bulletinDedupeKey(kind, message string) string {
	kind = strings.ToUpper(strings.TrimSpace(kind))
	message = strings.TrimRight(message, "\r\n")
	if kind == "" || strings.TrimSpace(message) == "" {
		return ""
	}
	return kind + "\x00" + message
}

// BulletinDedupeSnapshot returns current telnet bulletin dedupe counters.
func (s *Server) BulletinDedupeSnapshot() BulletinDedupeSnapshot {
	if s == nil || s.bulletinDedupe == nil {
		return BulletinDedupeSnapshot{}
	}
	return s.bulletinDedupe.snapshot()
}
