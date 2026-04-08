package spot

import (
	"math"
	"sort"
)

const (
	ftDialMaxAudioOffsetKHz = 4.5
	ftDialNegativeSlackKHz  = 0.15
)

type ftDialKey struct {
	band string
	mode string
}

// FTDialRegistry is an immutable lookup table of canonical FT dial
// frequencies. It is built from the same seed data used by mode inference so
// PSKReporter ingest and inference buckets share one frequency source of truth.
type FTDialRegistry struct {
	entries map[ftDialKey][]float64
}

// NewFTDialRegistry builds an immutable FT dial-frequency registry from the
// configured digital seeds. Only FT2/FT4/FT8 entries are retained.
func NewFTDialRegistry(seeds []ModeSeed) *FTDialRegistry {
	if len(seeds) == 0 {
		return nil
	}
	entries := make(map[ftDialKey][]float64)
	for _, seed := range seeds {
		mode := normalizeUpperASCIITrim(seed.Mode)
		if !supportsFTDialCanonicalization(mode) || seed.FrequencyKHz <= 0 {
			continue
		}
		freqKHz := roundFrequencyTo10Hz(float64(seed.FrequencyKHz))
		band := NormalizeBand(FreqToBand(freqKHz))
		if band == "" || band == "???" {
			continue
		}
		key := ftDialKey{band: band, mode: mode}
		entries[key] = append(entries[key], freqKHz)
	}
	if len(entries) == 0 {
		return nil
	}
	for key, dials := range entries {
		sort.Float64s(dials)
		dst := dials[:0]
		last := math.NaN()
		for _, dial := range dials {
			if len(dst) == 0 || dial != last {
				dst = append(dst, dial)
				last = dial
			}
		}
		entries[key] = dst
	}
	return &FTDialRegistry{entries: entries}
}

// Canonicalize maps an observed FT RF frequency to a canonical dial frequency.
// The mapping is mode-specific, band-scoped, deterministic, and fail-open: when
// no safe candidate exists, it returns ok=false and leaves the caller's
// operational frequency unchanged.
func (r *FTDialRegistry) Canonicalize(mode string, observedFreqKHz float64) (float64, bool) {
	if r == nil || observedFreqKHz <= 0 {
		return 0, false
	}
	mode = normalizeUpperASCIITrim(mode)
	if !supportsFTDialCanonicalization(mode) {
		return 0, false
	}
	band := NormalizeBand(FreqToBand(observedFreqKHz))
	if band == "" || band == "???" {
		return 0, false
	}
	dials := r.entries[ftDialKey{band: band, mode: mode}]
	if len(dials) == 0 {
		return 0, false
	}
	bestDial := 0.0
	bestAbsDelta := math.MaxFloat64
	bestDelta := math.MaxFloat64
	found := false
	for _, dial := range dials {
		delta := observedFreqKHz - dial
		if delta < -ftDialNegativeSlackKHz || delta > ftDialMaxAudioOffsetKHz {
			continue
		}
		absDelta := math.Abs(delta)
		if !found ||
			absDelta < bestAbsDelta ||
			(absDelta == bestAbsDelta && preferFTDialCandidate(delta, bestDelta)) {
			bestDial = dial
			bestAbsDelta = absDelta
			bestDelta = delta
			found = true
		}
	}
	if !found {
		return 0, false
	}
	return roundFrequencyTo10Hz(bestDial), true
}

func supportsFTDialCanonicalization(mode string) bool {
	switch normalizeUpperASCIITrim(mode) {
	case "FT2", "FT4", "FT8":
		return true
	default:
		return false
	}
}

func preferFTDialCandidate(delta, currentBest float64) bool {
	if delta >= 0 && currentBest < 0 {
		return true
	}
	if delta < 0 && currentBest >= 0 {
		return false
	}
	if delta == currentBest {
		return false
	}
	return delta < currentBest
}
