package pathreliability

import (
	"math"
	"time"
)

// Predictor coordinates store access and presentation mapping.
type Predictor struct {
	combined *Store
	cfg      Config
}

// NewPredictor builds a predictor with normalized configuration.
func NewPredictor(cfg Config, bands []string) *Predictor {
	cfg.normalize()
	return &Predictor{
		combined: NewStore(cfg, bands),
		cfg:      cfg,
	}
}

// Config returns the normalized configuration in use.
func (p *Predictor) Config() Config {
	if p == nil {
		return Config{}
	}
	return p.cfg
}

// Update ingests a spot contribution (FT8-equiv) with optional beacon weight cap.
func (p *Predictor) Update(bucket BucketClass, receiverCell, senderCell CellID, receiverCoarse, senderCoarse CellID, band string, ft8dB float64, weight float64, now time.Time, isBeacon bool) {
	if p == nil || !p.cfg.Enabled {
		return
	}
	w := weight
	if isBeacon && p.cfg.BeaconWeightCap > 0 && w > p.cfg.BeaconWeightCap {
		w = p.cfg.BeaconWeightCap
	}
	power := p.cfg.powerFromDB(ft8dB)
	switch bucket {
	case BucketCombined:
		if p.combined != nil {
			p.combined.Update(receiverCell, senderCell, receiverCoarse, senderCoarse, band, power, w, now)
		}
	default:
		return
	}
}

// PredictionSource describes which bucket family produced the glyph.
type PredictionSource uint8

const (
	SourceInsufficient PredictionSource = iota
	SourceCombined
)

// InsufficientReason describes why a prediction returned the insufficient glyph.
type InsufficientReason uint8

const (
	InsufficientNone InsufficientReason = iota
	InsufficientNoSample
	InsufficientLowCount
	InsufficientLowWeight
	InsufficientStale
)

func (r InsufficientReason) String() string {
	switch r {
	case InsufficientNone:
		return "none"
	case InsufficientNoSample:
		return "no_sample"
	case InsufficientLowCount:
		return "low_count"
	case InsufficientLowWeight:
		return "low_weight"
	case InsufficientStale:
		return "stale"
	default:
		return "unknown"
	}
}

// Result carries merged glyph and diagnostics.
type Result struct {
	Glyph              string
	Value              float64 // power
	Weight             float64
	AgeSec             int64
	Count              uint32
	Source             PredictionSource
	InsufficientReason InsufficientReason
}

// Predict returns a single merged glyph for the path.
func (p *Predictor) Predict(userCell, dxCell CellID, userCoarse, dxCoarse CellID, band string, mode string, noisePenalty float64, now time.Time) Result {
	minObservationCount := 0
	if p != nil {
		minObservationCount = p.cfg.MinObservationCount
	}
	return p.PredictWithMinObservationCount(userCell, dxCell, userCoarse, dxCoarse, band, mode, noisePenalty, minObservationCount, now)
}

// PredictWithMinObservationCount returns a merged glyph using the supplied raw
// observation floor. Callers may pass a floor above the configured default when
// applying stricter per-session display or filter policy.
func (p *Predictor) PredictWithMinObservationCount(userCell, dxCell CellID, userCoarse, dxCoarse CellID, band string, mode string, noisePenalty float64, minObservationCount int, now time.Time) Result {
	insufficient := "?"
	if p != nil && p.cfg.GlyphSymbols.Insufficient != "" {
		insufficient = p.cfg.GlyphSymbols.Insufficient
	}
	if p == nil || !p.cfg.Enabled {
		return Result{Glyph: insufficient, Source: SourceInsufficient}
	}

	modeKey := normalizeMode(mode)
	makeResult := func(power, weight float64, ageSec int64, count uint32, source PredictionSource) Result {
		return Result{Glyph: GlyphForPower(power, modeKey, p.cfg), Value: power, Weight: weight, AgeSec: ageSec, Count: count, Source: source}
	}
	makeInsufficient := func(power, weight float64, ageSec int64, count uint32, reason InsufficientReason) Result {
		return Result{Glyph: insufficient, Value: power, Weight: weight, AgeSec: ageSec, Count: count, Source: SourceInsufficient, InsufficientReason: reason}
	}

	mergedPower, mergedWeight, mergedAge, mergedCount, reason, ok := p.mergeFromStore(p.combined, userCell, dxCell, userCoarse, dxCoarse, band, noisePenalty, now)
	if minObservationCount < p.cfg.MinObservationCount {
		minObservationCount = p.cfg.MinObservationCount
	}
	if ok && mergedWeight >= p.cfg.MinEffectiveWeight && countMeetsMinimum(mergedCount, minObservationCount) {
		return makeResult(mergedPower, mergedWeight, mergedAge, mergedCount, SourceCombined)
	}
	if ok {
		if reason == InsufficientNone {
			if !countMeetsMinimum(mergedCount, minObservationCount) {
				reason = InsufficientLowCount
			} else {
				reason = InsufficientLowWeight
			}
		}
		return makeInsufficient(mergedPower, mergedWeight, mergedAge, mergedCount, reason)
	}
	if reason == InsufficientNone {
		reason = InsufficientNoSample
	}
	return makeInsufficient(0, 0, 0, mergedCount, reason)
}

func countMeetsMinimum(count uint32, min int) bool {
	return min <= 0 || uint64(count) >= uint64(min)
}

func mergeSamples(receive Sample, transmit Sample, cfg Config, noisePenalty float64) (float64, float64, int64, uint32, bool) {
	hasReceive := receive.Weight > 0
	hasTransmit := transmit.Weight > 0
	if !hasReceive && !hasTransmit {
		return 0, 0, 0, 0, false
	}
	receivePower := receive.Value
	if hasReceive && noisePenalty > 0 {
		receivePower = ApplyNoisePower(receivePower, noisePenalty, cfg)
	}
	if hasReceive && hasTransmit {
		mergedPower := cfg.MergeReceiveWeight*receivePower + cfg.MergeTransmitWeight*transmit.Value
		mergedWeight := cfg.MergeReceiveWeight*receive.Weight + cfg.MergeTransmitWeight*transmit.Weight
		return mergedPower, mergedWeight, weightedSampleAge(receive, transmit), saturatingAddCounts(receive.Count, transmit.Count), true
	}
	if hasReceive {
		return receivePower, receive.Weight * cfg.ReverseHintDiscount, receive.AgeSec, receive.Count, true
	}
	return transmit.Value, transmit.Weight * cfg.ReverseHintDiscount, transmit.AgeSec, transmit.Count, true
}

func (p *Predictor) mergeFromStore(store *Store, userCell, dxCell CellID, userCoarse, dxCoarse CellID, band string, noisePenalty float64, now time.Time) (float64, float64, int64, uint32, InsufficientReason, bool) {
	if store == nil {
		return 0, 0, 0, 0, InsufficientNoSample, false
	}
	// Receive (DX->user): receiver=user, sender=dx.
	rFine, rCoarse := store.Lookup(userCell, dxCell, userCoarse, dxCoarse, band, now)
	receive := SelectSample(rFine, rCoarse, p.cfg.MinFineWeight, p.cfg.FineOnlyWeight)

	// Transmit (user->DX): receiver=dx, sender=user.
	tFine, tCoarse := store.Lookup(dxCell, userCell, dxCoarse, userCoarse, band, now)
	transmit := SelectSample(tFine, tCoarse, p.cfg.MinFineWeight, p.cfg.FineOnlyWeight)

	receive, transmit, staleDropped, staleCount := p.applyFreshnessGate(store, band, receive, transmit)
	power, weight, ageSec, count, ok := mergeSamples(receive, transmit, p.cfg, noisePenalty)
	reason := InsufficientNone
	if staleDropped {
		reason = InsufficientStale
	}
	if !ok {
		if reason == InsufficientNone {
			reason = InsufficientNoSample
		}
		return 0, 0, 0, staleCount, reason, false
	}
	return power, weight, ageSec, count, reason, true
}

func (p *Predictor) applyFreshnessGate(store *Store, band string, receive Sample, transmit Sample) (Sample, Sample, bool, uint32) {
	maxAge := p.maxPredictionAgeSeconds(store, band)
	if maxAge <= 0 {
		return receive, transmit, false, 0
	}
	staleDropped := false
	var staleCount uint32
	if receive.Weight > 0 && receive.AgeSec > maxAge {
		staleCount = saturatingAddCounts(staleCount, receive.Count)
		receive = Sample{}
		staleDropped = true
	}
	if transmit.Weight > 0 && transmit.AgeSec > maxAge {
		staleCount = saturatingAddCounts(staleCount, transmit.Count)
		transmit = Sample{}
		staleDropped = true
	}
	return receive, transmit, staleDropped, staleCount
}

func (p *Predictor) maxPredictionAgeSeconds(store *Store, band string) int64 {
	if p == nil || store == nil || p.cfg.MaxPredictionAgeHalfLifeMultiplier <= 0 {
		return 0
	}
	halfLife := store.bandIndex.HalfLifeSeconds(band, p.cfg)
	if halfLife <= 0 {
		return 0
	}
	seconds := math.Ceil(float64(halfLife) * p.cfg.MaxPredictionAgeHalfLifeMultiplier)
	if seconds < 1 {
		return 1
	}
	return int64(seconds)
}

// PurgeStale runs a stale purge and returns removed count.
func (p *Predictor) PurgeStale(now time.Time) int {
	if p == nil {
		return 0
	}
	removed := 0
	if p.combined != nil {
		removed += p.combined.PurgeStale(now)
	}
	return removed
}

// Compact rebuilds shard maps that have shrunk far below peak size.
func (p *Predictor) Compact(minPeak int, shrinkRatio float64) int {
	if p == nil || !p.cfg.Enabled || p.combined == nil {
		return 0
	}
	return p.combined.Compact(minPeak, shrinkRatio)
}

// TotalBuckets returns the total buckets across all stores.
func (p *Predictor) TotalBuckets() int {
	if p == nil || !p.cfg.Enabled || p.combined == nil {
		return 0
	}
	return p.combined.TotalBuckets()
}

// RefreshStatsSnapshot recomputes the cached bucket-count snapshot used by
// operator-facing stats displays.
func (p *Predictor) RefreshStatsSnapshot(now time.Time) {
	if p == nil || !p.cfg.Enabled || p.combined == nil {
		return
	}
	p.combined.RefreshStatsSnapshot(now)
}

// PredictorStats returns counts of active fine/coarse buckets (non-stale).
type PredictorStats struct {
	CombinedFine   int
	CombinedCoarse int
}

// BandBucketStats reports fine/coarse bucket counts per band.
type BandBucketStats struct {
	Band   string
	Fine   int
	Coarse int
}

// BandWeightHistogram reports per-band bucket weight distributions.
type BandWeightHistogram struct {
	Band  string
	Total int
	Bins  []int
}

// WeightHistogram carries histogram edges and per-band counts.
type WeightHistogram struct {
	Edges []float64
	Bands []BandWeightHistogram
}

// weightHistogramEdges defines decayed weight bins for log-only histograms.
var weightHistogramEdges = []float64{1, 2, 3, 5, 10}

// Stats returns counts of active fine/coarse buckets for the combined store.
func (p *Predictor) Stats(now time.Time) PredictorStats {
	if p == nil || !p.cfg.Enabled {
		return PredictorStats{}
	}
	stats := PredictorStats{}
	if p.combined != nil {
		stats.CombinedFine, stats.CombinedCoarse = p.combined.Stats(now)
	}
	return stats
}

// StatsByBand returns per-band bucket counts for the combined store.
func (p *Predictor) StatsByBand(now time.Time) []BandBucketStats {
	if p == nil || !p.cfg.Enabled {
		return nil
	}
	if p.combined == nil {
		return nil
	}
	bands := p.combined.bandIndex.Bands()
	if len(bands) == 0 {
		return nil
	}
	counts := p.combined.StatsByBand(now)
	stats := make([]BandBucketStats, len(bands))
	for i, band := range bands {
		entry := BandBucketStats{Band: band}
		if i < len(counts) {
			entry.Fine = counts[i].fine
			entry.Coarse = counts[i].coarse
		}
		stats[i] = entry
	}
	return stats
}

// WeightHistogramByBand returns per-band bucket weight histograms for the combined store.
func (p *Predictor) WeightHistogramByBand(now time.Time) WeightHistogram {
	if p == nil || !p.cfg.Enabled || p.combined == nil {
		return WeightHistogram{}
	}
	bands := p.combined.bandIndex.Bands()
	if len(bands) == 0 {
		return WeightHistogram{}
	}
	counts := p.combined.WeightHistogramByBand(now, weightHistogramEdges)
	if len(counts) == 0 {
		return WeightHistogram{}
	}
	entries := make([]BandWeightHistogram, len(bands))
	for i, band := range bands {
		entry := BandWeightHistogram{Band: band}
		if i < len(counts) {
			entry.Total = counts[i].total
			entry.Bins = append([]int(nil), counts[i].bins...)
		}
		entries[i] = entry
	}
	return WeightHistogram{
		Edges: append([]float64(nil), weightHistogramEdges...),
		Bands: entries,
	}
}
