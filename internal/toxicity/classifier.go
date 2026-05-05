package toxicity

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"dxcluster/spot"
)

type Stats struct {
	SafeBypass    uint64
	CacheHits     uint64
	AICalls       uint64
	Toxic         uint64
	Unavailable   uint64
	Timeouts      uint64
	Malformed     uint64
	QueueFull     uint64
	Evictions     uint64
	Pending       int64
	ResultBacklog int
	CacheEntries  int
}

type Result struct {
	Spot     *spot.Spot
	Decision Decision
}

type classifierClient interface {
	Classify(context.Context, string) (Decision, error)
}

type job struct {
	spot    *spot.Spot
	comment string
}

// Classifier owns bounded AI work for human comments. Callers either receive a
// synchronous local/cache decision or enqueue one job and let the output loop
// continue processing unrelated spots.
type Classifier struct {
	cfg      Config
	gate     *SafeGate
	cache    *Cache
	client   classifierClient
	jobs     chan job
	results  chan Result
	stop     chan struct{}
	stopOnce sync.Once
	wg       sync.WaitGroup

	pending     atomic.Int64
	safeBypass  atomic.Uint64
	cacheHits   atomic.Uint64
	aiCalls     atomic.Uint64
	toxic       atomic.Uint64
	unavailable atomic.Uint64
	timeouts    atomic.Uint64
	malformed   atomic.Uint64
	queueFull   atomic.Uint64
}

func NewClassifier(cfg Config, gate *SafeGate, client classifierClient) *Classifier {
	if !cfg.Enabled || cfg.Workers <= 0 || cfg.QueueSize <= 0 || client == nil {
		return nil
	}
	return &Classifier{
		cfg:     cfg,
		gate:    gate,
		cache:   NewCache(cfg.CacheMaxEntries, time.Duration(cfg.CacheTTLSeconds)*time.Second),
		client:  client,
		jobs:    make(chan job, cfg.QueueSize),
		results: make(chan Result, cfg.QueueSize),
		stop:    make(chan struct{}),
	}
}

func (c *Classifier) Start() {
	if c == nil {
		return
	}
	for i := 0; i < c.cfg.Workers; i++ {
		c.wg.Add(1)
		go c.worker()
	}
}

func (c *Classifier) Stop() {
	if c == nil {
		return
	}
	c.stopOnce.Do(func() {
		close(c.stop)
	})
	c.wg.Wait()
}

func (c *Classifier) Results() <-chan Result {
	if c == nil {
		return nil
	}
	return c.results
}

// ClassifyOrEnqueue returns true when the caller can continue processing the
// spot immediately. It returns false only after a bounded job enqueue succeeds.
func (c *Classifier) ClassifyOrEnqueue(s *spot.Spot, now time.Time) bool {
	if s == nil || !s.IsHuman || spot.NormalizeToxicityStatus(string(s.ToxicityStatus)) != spot.ToxicityUnknown {
		return true
	}
	clean := clampCommentBytes(NormalizeComment(s.Comment), c.cfg.MaxCommentBytes)
	if clean == "" || c.gate.IsSafe(clean) {
		applyDecision(s, Decision{Status: spot.ToxicitySafeLocal})
		c.safeBypass.Add(1)
		return true
	}
	if decision, ok := c.cache.Get(clean, now); ok {
		applyDecision(s, decision)
		c.cacheHits.Add(1)
		if decision.Status == spot.ToxicityToxic {
			c.toxic.Add(1)
		}
		return true
	}
	c.pending.Add(1)
	select {
	case c.jobs <- job{spot: s, comment: clean}:
		return false
	default:
		c.pending.Add(-1)
		applyDecision(s, Decision{Status: spot.ToxicityUnavailable})
		c.queueFull.Add(1)
		c.unavailable.Add(1)
		return true
	}
}

func (c *Classifier) worker() {
	defer c.wg.Done()
	for {
		select {
		case <-c.stop:
			return
		case j := <-c.jobs:
			c.handleJob(j)
		}
	}
}

func (c *Classifier) handleJob(j job) {
	defer c.pending.Add(-1)
	ctx, cancel := context.WithTimeout(context.Background(), c.cfg.timeout())
	defer cancel()
	c.aiCalls.Add(1)
	decision, err := c.client.Classify(ctx, j.comment)
	if err != nil {
		if ctx.Err() != nil {
			c.timeouts.Add(1)
		} else {
			c.malformed.Add(1)
		}
		decision = Decision{Status: spot.ToxicityUnavailable}
		c.unavailable.Add(1)
	} else if decision.Status == spot.ToxicityToxic {
		c.toxic.Add(1)
	}
	if shouldCacheDecision(decision) {
		c.cache.Put(j.comment, decision, time.Now().UTC())
	}
	select {
	case c.results <- Result{Spot: j.spot, Decision: decision}:
	case <-c.stop:
	}
}

func (c *Classifier) Pending() int64 {
	if c == nil {
		return 0
	}
	return c.pending.Load()
}

func (c *Classifier) ResultBacklog() int {
	if c == nil {
		return 0
	}
	return len(c.results)
}

func (c *Classifier) DrainTimeout() time.Duration {
	if c == nil {
		return 0
	}
	timeout := 2*c.cfg.timeout() + 250*time.Millisecond
	if timeout < 500*time.Millisecond {
		return 500 * time.Millisecond
	}
	if timeout > 5*time.Second {
		return 5 * time.Second
	}
	return timeout
}

func (c *Classifier) Snapshot() Stats {
	if c == nil {
		return Stats{}
	}
	return Stats{
		SafeBypass:    c.safeBypass.Load(),
		CacheHits:     c.cacheHits.Load(),
		AICalls:       c.aiCalls.Load(),
		Toxic:         c.toxic.Load(),
		Unavailable:   c.unavailable.Load(),
		Timeouts:      c.timeouts.Load(),
		Malformed:     c.malformed.Load(),
		QueueFull:     c.queueFull.Load(),
		Evictions:     c.cache.Evictions(),
		Pending:       c.Pending(),
		ResultBacklog: c.ResultBacklog(),
		CacheEntries:  c.cache.Len(),
	}
}

func shouldCacheDecision(decision Decision) bool {
	return spot.NormalizeToxicityStatus(string(decision.Status)) != spot.ToxicityUnavailable
}

func applyDecision(s *spot.Spot, decision Decision) {
	if s == nil {
		return
	}
	s.ToxicityStatus = spot.NormalizeToxicityStatus(string(decision.Status))
	s.ToxicityCategories = append([]string(nil), decision.Categories...)
	s.ToxicityModel = decision.Model
}

func ApplyResult(result Result) {
	applyDecision(result.Spot, result.Decision)
}
