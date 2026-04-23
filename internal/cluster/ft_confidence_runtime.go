package cluster

import (
	"container/heap"
	"math"
	"strings"
	"time"

	"dxcluster/config"
	"dxcluster/spot"
	"dxcluster/stats"
	"dxcluster/strutil"
)

const (
	ftConfidenceMaxPendingGroups      = 12000
	ftConfidenceMaxPendingSpots       = 30000
	ftConfidenceRecentMaxEntries      = 200000
	ftConfidenceRecentMaxSpotters     = 16
	ftConfidenceRecentCleanupInterval = 10 * time.Minute
)

type ftConfidenceTiming struct {
	quietGap time.Duration
	hardCap  time.Duration
}

type ftConfidencePolicy struct {
	pMinUniqueSpotters int
	vMinUniqueSpotters int
	ft8Timing          ftConfidenceTiming
	ft4Timing          ftConfidenceTiming
	ft2Timing          ftConfidenceTiming
}

type ftConfidenceKey struct {
	call    string
	mode    string
	freqKey int64
}

type ftConfidencePendingGroup struct {
	key          ftConfidenceKey
	firstSeen    time.Time
	lastSeen     time.Time
	hardDeadline time.Time
	due          time.Time
	seq          uint64
	contexts     []*outputSpotContext
	spotters     map[string]struct{}
}

type ftConfidenceItem struct {
	key ftConfidenceKey
	due time.Time
	seq uint64
}

type ftConfidenceHeap []ftConfidenceItem

func (h ftConfidenceHeap) Len() int { return len(h) }

func (h ftConfidenceHeap) Less(i, j int) bool {
	if h[i].due.Equal(h[j].due) {
		return h[i].seq < h[j].seq
	}
	return h[i].due.Before(h[j].due)
}

func (h ftConfidenceHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
}

func (h *ftConfidenceHeap) Push(x any) {
	item, ok := x.(ftConfidenceItem)
	if !ok {
		return
	}
	*h = append(*h, item)
}

func (h *ftConfidenceHeap) Pop() any {
	old := *h
	n := len(old)
	if n == 0 {
		return nil
	}
	item := old[n-1]
	*h = old[:n-1]
	return item
}

func popFTConfidenceDue(h *ftConfidenceHeap, now time.Time, force bool, scratch []ftConfidenceItem) []ftConfidenceItem {
	if h == nil || h.Len() == 0 {
		return nil
	}
	out := scratch[:0]
	for h.Len() > 0 {
		head := (*h)[0]
		if !force && head.due.After(now) {
			break
		}
		itemAny := heap.Pop(h)
		if item, ok := itemAny.(ftConfidenceItem); ok {
			out = append(out, item)
		}
	}
	return out
}

// ftConfidenceController owns bounded FT corroboration state for the single
// output-pipeline goroutine. It groups related FT reports long enough to assign
// ?/P/V without involving resolver mutation, and it fails open when bounds are
// reached by releasing the current spot immediately.
type ftConfidenceController struct {
	enabled          bool
	maxPendingGroups int
	maxPendingSpots  int
	pendingSpots     int
	policy           ftConfidencePolicy
	tracker          *stats.Tracker
	pending          map[ftConfidenceKey]*ftConfidencePendingGroup
	queue            ftConfidenceHeap
	dueScratch       []ftConfidenceItem
	seq              uint64
}

func newFTConfidencePolicy(cfg config.CallCorrectionConfig) ftConfidencePolicy {
	pMin := cfg.PMinUniqueSpotters
	if pMin < 2 {
		pMin = config.DefaultCallCorrectionFTPMinUniqueSpotters
	}
	vMin := cfg.VMinUniqueSpotters
	if vMin <= pMin {
		vMin = max(config.DefaultCallCorrectionFTVMinUniqueSpotters, pMin+1)
	}
	ft8QuietGap := cfg.FT8QuietGapSeconds
	if ft8QuietGap <= 0 {
		ft8QuietGap = config.DefaultCallCorrectionFT8QuietGapSeconds
	}
	ft8HardCap := cfg.FT8HardCapSeconds
	if ft8HardCap < ft8QuietGap {
		ft8HardCap = max(config.DefaultCallCorrectionFT8HardCapSeconds, ft8QuietGap)
	}
	ft4QuietGap := cfg.FT4QuietGapSeconds
	if ft4QuietGap <= 0 {
		ft4QuietGap = config.DefaultCallCorrectionFT4QuietGapSeconds
	}
	ft4HardCap := cfg.FT4HardCapSeconds
	if ft4HardCap < ft4QuietGap {
		ft4HardCap = max(config.DefaultCallCorrectionFT4HardCapSeconds, ft4QuietGap)
	}
	ft2QuietGap := cfg.FT2QuietGapSeconds
	if ft2QuietGap <= 0 {
		ft2QuietGap = config.DefaultCallCorrectionFT2QuietGapSeconds
	}
	ft2HardCap := cfg.FT2HardCapSeconds
	if ft2HardCap < ft2QuietGap {
		ft2HardCap = max(config.DefaultCallCorrectionFT2HardCapSeconds, ft2QuietGap)
	}
	return ftConfidencePolicy{
		pMinUniqueSpotters: pMin,
		vMinUniqueSpotters: vMin,
		ft8Timing: ftConfidenceTiming{
			quietGap: time.Duration(ft8QuietGap) * time.Second,
			hardCap:  time.Duration(ft8HardCap) * time.Second,
		},
		ft4Timing: ftConfidenceTiming{
			quietGap: time.Duration(ft4QuietGap) * time.Second,
			hardCap:  time.Duration(ft4HardCap) * time.Second,
		},
		ft2Timing: ftConfidenceTiming{
			quietGap: time.Duration(ft2QuietGap) * time.Second,
			hardCap:  time.Duration(ft2HardCap) * time.Second,
		},
	}
}

func newFTConfidenceController(cfg config.CallCorrectionConfig, tracker *stats.Tracker) *ftConfidenceController {
	controller := &ftConfidenceController{
		enabled:          true,
		maxPendingGroups: ftConfidenceMaxPendingGroups,
		maxPendingSpots:  ftConfidenceMaxPendingSpots,
		policy:           newFTConfidencePolicy(cfg),
		tracker:          tracker,
		pending:          make(map[ftConfidenceKey]*ftConfidencePendingGroup),
	}
	heap.Init(&controller.queue)
	controller.reportActiveBursts()
	return controller
}

func (c *ftConfidenceController) Enabled() bool {
	return c != nil && c.enabled
}

func (c *ftConfidenceController) Observe(now time.Time, ctx *outputSpotContext) (bool, int) {
	if !c.Enabled() || ctx == nil || ctx.spot == nil {
		return false, 1
	}
	if now.IsZero() {
		now = time.Now().UTC()
	} else {
		now = now.UTC()
	}
	key, spotter, ok := buildFTConfidenceKey(ctx.spot)
	if !ok {
		return false, 1
	}
	timing, ok := ftConfidenceTimingForMode(key.mode, c.policy)
	if !ok {
		return false, 1
	}
	group := c.pending[key]
	if group != nil {
		uniqueCount := len(group.spotters)
		if spotter != "" {
			if _, exists := group.spotters[spotter]; !exists {
				uniqueCount++
			}
		}
		if c.maxPendingSpots > 0 && c.pendingSpots >= c.maxPendingSpots {
			c.reportOverflowRelease()
			return false, uniqueCount
		}
		group.contexts = append(group.contexts, ctx)
		if spotter != "" {
			if group.spotters == nil {
				group.spotters = make(map[string]struct{}, 2)
			}
			group.spotters[spotter] = struct{}{}
		}
		group.lastSeen = now
		c.advanceGroupDue(group, timing)
		c.pendingSpots++
		return true, uniqueCount
	}

	if c.maxPendingGroups > 0 && len(c.pending) >= c.maxPendingGroups {
		c.reportOverflowRelease()
		return false, 1
	}
	if c.maxPendingSpots > 0 && c.pendingSpots >= c.maxPendingSpots {
		c.reportOverflowRelease()
		return false, 1
	}

	due := ftConfidenceBurstDue(now, now.Add(timing.hardCap), timing)
	group = &ftConfidencePendingGroup{
		key:          key,
		firstSeen:    now,
		lastSeen:     now,
		hardDeadline: now.Add(timing.hardCap),
		due:          due,
		seq:          c.seq,
		contexts:     make([]*outputSpotContext, 1, 4),
	}
	group.contexts[0] = ctx
	if spotter != "" {
		group.spotters = make(map[string]struct{}, 2)
		group.spotters[spotter] = struct{}{}
	}
	c.pending[key] = group
	c.pushDueItem(group)
	c.seq++
	c.pendingSpots++
	c.reportActiveBursts()
	return true, 1
}

func (c *ftConfidenceController) Drain(now time.Time, force bool) []*ftConfidencePendingGroup {
	if !c.Enabled() {
		return nil
	}
	if now.IsZero() {
		now = time.Now().UTC()
	} else {
		now = now.UTC()
	}
	items := popFTConfidenceDue(&c.queue, now, force, c.dueScratch)
	c.dueScratch = items[:0]
	if len(items) == 0 {
		return nil
	}
	out := make([]*ftConfidencePendingGroup, 0, len(items))
	for _, item := range items {
		group := c.pending[item.key]
		if group == nil {
			continue
		}
		if item.seq != group.seq || !item.due.Equal(group.due) {
			continue
		}
		delete(c.pending, item.key)
		c.pendingSpots -= len(group.contexts)
		if c.pendingSpots < 0 {
			c.pendingSpots = 0
		}
		c.reportBurstRelease(group)
		out = append(out, group)
	}
	c.reportActiveBursts()
	return out
}

func (c *ftConfidenceController) NextDue() (time.Time, bool) {
	if !c.Enabled() {
		return time.Time{}, false
	}
	for c.queue.Len() > 0 {
		head := c.queue[0]
		group := c.pending[head.key]
		if group == nil || head.seq != group.seq || !head.due.Equal(group.due) {
			heap.Pop(&c.queue)
			continue
		}
		return head.due, true
	}
	return time.Time{}, false
}

func buildFTConfidenceKey(s *spot.Spot) (ftConfidenceKey, string, bool) {
	if s == nil {
		return ftConfidenceKey{}, "", false
	}
	mode := s.ModeNorm
	if mode == "" {
		mode = s.Mode
	}
	mode = strutil.NormalizeUpper(mode)
	if _, _, ok := spot.FTConfidenceTimingKeys(mode); !ok {
		return ftConfidenceKey{}, "", false
	}
	call := s.DXCallNorm
	if call == "" {
		call = s.DXCall
	}
	call = spot.NormalizeCallsign(call)
	if call == "" {
		return ftConfidenceKey{}, "", false
	}
	spotter := s.DECallNorm
	if spotter == "" {
		spotter = s.DECall
	}
	spotter = spot.NormalizeCallsign(spotter)
	return ftConfidenceKey{
		call:    call,
		mode:    mode,
		freqKey: ftConfidenceFrequencyKey(s.Frequency),
	}, spotter, true
}

func ftConfidenceTimingForMode(mode string, policy ftConfidencePolicy) (ftConfidenceTiming, bool) {
	quietKey, hardCapKey, ok := spot.FTConfidenceTimingKeys(mode)
	if !ok {
		return ftConfidenceTiming{}, false
	}
	switch {
	case quietKey == "ft8_quiet_gap_seconds" && hardCapKey == "ft8_hard_cap_seconds":
		return policy.ft8Timing, true
	case quietKey == "ft4_quiet_gap_seconds" && hardCapKey == "ft4_hard_cap_seconds":
		return policy.ft4Timing, true
	case quietKey == "ft2_quiet_gap_seconds" && hardCapKey == "ft2_hard_cap_seconds":
		return policy.ft2Timing, true
	default:
		return ftConfidenceTiming{}, false
	}
}

func ftConfidenceBurstDue(lastSeen, hardDeadline time.Time, timing ftConfidenceTiming) time.Time {
	due := lastSeen.Add(timing.quietGap)
	if due.After(hardDeadline) {
		return hardDeadline
	}
	return due
}

func (c *ftConfidenceController) pushDueItem(group *ftConfidencePendingGroup) {
	if c == nil || group == nil {
		return
	}
	heap.Push(&c.queue, ftConfidenceItem{
		key: group.key,
		due: group.due,
		seq: group.seq,
	})
}

func (c *ftConfidenceController) advanceGroupDue(group *ftConfidencePendingGroup, timing ftConfidenceTiming) {
	if c == nil || group == nil {
		return
	}
	newDue := ftConfidenceBurstDue(group.lastSeen, group.hardDeadline, timing)
	if group.due.Equal(newDue) {
		return
	}
	group.due = newDue
	group.seq = c.seq
	c.pushDueItem(group)
	c.seq++
}

func (c *ftConfidenceController) reportActiveBursts() {
	if c == nil || c.tracker == nil {
		return
	}
	c.tracker.SetFTBurstActive(int64(len(c.pending)))
}

func (c *ftConfidenceController) reportOverflowRelease() {
	if c == nil || c.tracker == nil {
		return
	}
	c.tracker.IncrementFTBurstOverflowRelease()
}

func (c *ftConfidenceController) reportBurstRelease(group *ftConfidencePendingGroup) {
	if c == nil || c.tracker == nil || group == nil {
		return
	}
	c.tracker.IncrementFTBurstReleased()
	c.tracker.ObserveFTBurstSpan(group.key.mode, group.lastSeen.Sub(group.firstSeen))
}

func ftConfidenceFrequencyKey(freqKHz float64) int64 {
	if freqKHz <= 0 {
		return 0
	}
	return int64(math.Round(freqKHz * 100))
}

func ftConfidenceGlyphForUniqueSpotters(uniqueSpotters int, policy ftConfidencePolicy) string {
	switch {
	case uniqueSpotters >= policy.vMinUniqueSpotters:
		return "V"
	case uniqueSpotters >= policy.pMinUniqueSpotters:
		return "P"
	default:
		return "?"
	}
}

func newFTRecentBandStore(cfg config.CallCorrectionConfig) *spot.RecentBandStore {
	if !cfg.RecentBandBonusEnabled {
		return nil
	}
	window := time.Duration(cfg.RecentBandWindowSeconds) * time.Second
	return spot.NewRecentBandStoreWithOptions(spot.RecentBandOptions{
		Window:             window,
		MaxEntries:         ftConfidenceRecentMaxEntries,
		CleanupInterval:    ftConfidenceRecentCleanupInterval,
		MaxSpottersPerCall: ftConfidenceRecentMaxSpotters,
	})
}

func (p *outputPipeline) applyFTConfidenceStage(ctx *outputSpotContext, now time.Time) bool {
	if p == nil || ctx == nil || ctx.spot == nil {
		return false
	}
	s := ctx.spot
	if s.IsBeacon || !modeSupportsFTConfidenceGlyph(ctx.modeUpper) || strings.TrimSpace(s.Confidence) != "" {
		return true
	}
	if p.ftConfidence == nil || !p.ftConfidence.Enabled() {
		p.assignFTConfidence(ctx, 1)
		return true
	}
	p.releaseDueFT(now, false)
	held, uniqueCount := p.ftConfidence.Observe(now, ctx)
	if held {
		return false
	}
	p.assignFTConfidence(ctx, uniqueCount)
	return true
}

func (p *outputPipeline) assignFTConfidence(ctx *outputSpotContext, uniqueSpotters int) {
	if p == nil || ctx == nil || ctx.spot == nil {
		return
	}
	s := ctx.spot
	policy := newFTConfidencePolicy(p.correctionCfg)
	if p.ftConfidence != nil {
		policy = p.ftConfidence.policy
	}
	s.Confidence = ftConfidenceGlyphForUniqueSpotters(uniqueSpotters, policy)
	if s.Confidence == "?" {
		applySupportFloor(s, p.recentBandStore, p.customSCPStore, p.ftRecentBandStore, p.correctionCfg)
	}
	ctx.dirty = true
}

func (p *outputPipeline) releaseDueFT(now time.Time, force bool) {
	if p == nil || p.ftConfidence == nil || !p.ftConfidence.Enabled() {
		return
	}
	releases := p.ftConfidence.Drain(now, force)
	for _, group := range releases {
		if group == nil {
			continue
		}
		uniqueSpotters := len(group.spotters)
		for _, ctx := range group.contexts {
			if ctx == nil || ctx.spot == nil {
				continue
			}
			p.assignFTConfidence(ctx, uniqueSpotters)
			if !p.finalizeSpotForMetrics(ctx) {
				continue
			}
			if !p.prepareFanoutSpot(ctx) {
				continue
			}
			p.deliverSpot(ctx)
		}
	}
}

func (p *outputPipeline) stopFTTimer() {
	if p == nil || p.ftTimer == nil {
		return
	}
	if !p.ftTimer.Stop() {
		select {
		case <-p.ftTimer.C:
		default:
		}
	}
	p.ftTimer = nil
	p.ftTimerCh = nil
}

func (p *outputPipeline) scheduleFTTimer(now time.Time) {
	if p == nil || p.ftConfidence == nil || !p.ftConfidence.Enabled() {
		p.stopFTTimer()
		return
	}
	nextDue, ok := p.ftConfidence.NextDue()
	if !ok {
		p.stopFTTimer()
		return
	}
	delay := nextDue.Sub(now)
	if delay < 0 {
		delay = 0
	}
	if p.ftTimer == nil {
		p.ftTimer = time.NewTimer(delay)
	} else {
		if !p.ftTimer.Stop() {
			select {
			case <-p.ftTimer.C:
			default:
			}
		}
		p.ftTimer.Reset(delay)
	}
	p.ftTimerCh = p.ftTimer.C
}

func (p *outputPipeline) recordFTRecentBandObservation(s *spot.Spot) {
	if p == nil || p.ftRecentBandStore == nil || s == nil || s.IsBeacon {
		return
	}
	mode := s.ModeNorm
	if mode == "" {
		mode = s.Mode
	}
	if !modeSupportsFTConfidenceGlyph(mode) {
		return
	}
	call := s.DXCallNorm
	if call == "" {
		call = s.DXCall
	}
	band := s.BandNorm
	if band == "" || band == "???" {
		band = spot.FreqToBand(s.Frequency)
	}
	spotter := s.DECallNorm
	if spotter == "" {
		spotter = s.DECall
	}
	seenAt := s.Time.UTC()
	if seenAt.IsZero() {
		seenAt = time.Now().UTC()
	}
	keys := spot.CorrectionFamilyKeys(call)
	if len(keys) == 0 {
		keys = []string{call}
	}
	for _, key := range keys {
		p.ftRecentBandStore.Record(key, band, mode, spotter, seenAt)
	}
}
