package toxicity

import (
	"container/list"
	"sync"
	"time"

	"dxcluster/spot"

	"github.com/zeebo/xxh3"
)

// Decision is the normalized classifier output cached and stamped onto spots.
type Decision struct {
	Status     spot.ToxicityStatus
	Categories []string
	Model      string
}

type cacheEntry struct {
	hash      uint64
	key       string
	decision  Decision
	expiresAt time.Time
}

// Cache is a hard-capped LRU with TTL expiry. It stores the normalized comment
// beside the hash so a hash collision cannot reuse another comment's decision.
type Cache struct {
	mu      sync.Mutex
	max     int
	ttl     time.Duration
	items   map[uint64][]*list.Element
	lru     *list.List
	evicted uint64
}

func NewCache(max int, ttl time.Duration) *Cache {
	if max <= 0 || ttl <= 0 {
		return nil
	}
	return &Cache{
		max:   max,
		ttl:   ttl,
		items: make(map[uint64][]*list.Element, max),
		lru:   list.New(),
	}
}

func (c *Cache) Get(key string, now time.Time) (Decision, bool) {
	if c == nil || key == "" {
		return Decision{}, false
	}
	hash := xxh3.HashString(key)
	c.mu.Lock()
	defer c.mu.Unlock()
	bucket := c.items[hash]
	for _, elem := range bucket {
		entry, ok := cacheEntryFromElement(elem)
		if !ok {
			continue
		}
		if entry.key != key {
			continue
		}
		if !entry.expiresAt.After(now) {
			c.removeElement(elem)
			return Decision{}, false
		}
		c.lru.MoveToFront(elem)
		return cloneDecision(entry.decision), true
	}
	return Decision{}, false
}

func (c *Cache) Put(key string, decision Decision, now time.Time) {
	if c == nil || key == "" {
		return
	}
	hash := xxh3.HashString(key)
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, elem := range c.items[hash] {
		entry, ok := cacheEntryFromElement(elem)
		if !ok {
			continue
		}
		if entry.key == key {
			entry.decision = cloneDecision(decision)
			entry.expiresAt = now.Add(c.ttl)
			c.lru.MoveToFront(elem)
			return
		}
	}
	entry := &cacheEntry{
		hash:      hash,
		key:       key,
		decision:  cloneDecision(decision),
		expiresAt: now.Add(c.ttl),
	}
	elem := c.lru.PushFront(entry)
	c.items[hash] = append(c.items[hash], elem)
	for c.lru.Len() > c.max {
		c.removeElement(c.lru.Back())
		c.evicted++
	}
}

func (c *Cache) Len() int {
	if c == nil {
		return 0
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.lru.Len()
}

func (c *Cache) Evictions() uint64 {
	if c == nil {
		return 0
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.evicted
}

func (c *Cache) removeElement(elem *list.Element) {
	if elem == nil {
		return
	}
	entry, ok := cacheEntryFromElement(elem)
	if !ok {
		c.lru.Remove(elem)
		return
	}
	bucket := c.items[entry.hash]
	for i, candidate := range bucket {
		if candidate == elem {
			bucket = append(bucket[:i], bucket[i+1:]...)
			break
		}
	}
	if len(bucket) == 0 {
		delete(c.items, entry.hash)
	} else {
		c.items[entry.hash] = bucket
	}
	c.lru.Remove(elem)
}

func cacheEntryFromElement(elem *list.Element) (*cacheEntry, bool) {
	if elem == nil {
		return nil, false
	}
	entry, ok := elem.Value.(*cacheEntry)
	return entry, ok
}

func cloneDecision(decision Decision) Decision {
	decision.Status = spot.NormalizeToxicityStatus(string(decision.Status))
	decision.Categories = append([]string(nil), decision.Categories...)
	return decision
}
