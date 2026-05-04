package main

import (
	stabilitypkg "dxcluster/internal/stability"
	"dxcluster/spot"
)

const stabilityBucketMinutes = 60

type replayStabilityConfig struct {
	WindowMinutes   int     `yaml:"window_minutes" json:"window_minutes"`
	MinFollowOn     int     `yaml:"min_follow_on" json:"min_follow_on"`
	FreqToleranceHz float64 `yaml:"freq_tolerance_hz" json:"freq_tolerance_hz"`
}

// replayStabilitySummary answers whether applied corrections were followed by
// enough confirming raw spots inside the configured window.
type replayStabilitySummary struct {
	WindowMinutes   int     `json:"window_minutes"`
	MinFollowOn     int     `json:"min_follow_on"`
	FreqToleranceHz float64 `json:"freq_tolerance_hz"`

	TotalApplied  int     `json:"total_applied"`
	StableApplied int     `json:"stable_applied"`
	StablePct     float64 `json:"stable_pct"`
}

// replayStabilityCollector keeps raw spots and applied corrections separate so
// stability evaluation can ask "did later evidence support this decision?"
// without feeding predictions back into the evidence set.
type replayStabilityCollector struct {
	cfg         replayStabilityConfig
	spots       map[string][]stabilitypkg.Spot
	corrections []stabilitypkg.Correction
}

// normalizeReplayStabilityConfig makes replay artifacts self-describing even
// when the YAML leaves stability settings at defaults.
func normalizeReplayStabilityConfig(cfg replayStabilityConfig) replayStabilityConfig {
	if cfg.WindowMinutes <= 0 {
		cfg.WindowMinutes = 60
	}
	if cfg.MinFollowOn <= 0 {
		cfg.MinFollowOn = 2
	}
	if cfg.FreqToleranceHz <= 0 {
		cfg.FreqToleranceHz = 1000
	}
	return cfg
}

func newReplayStabilityCollector(cfg replayStabilityConfig) *replayStabilityCollector {
	cfg = normalizeReplayStabilityConfig(cfg)
	return &replayStabilityCollector{
		cfg:         cfg,
		spots:       make(map[string][]stabilitypkg.Spot, 1<<15),
		corrections: make([]stabilitypkg.Correction, 0, 1<<14),
	}
}

// ObserveRaw records candidate raw evidence before correction decisions mutate
// callsigns.
func (c *replayStabilityCollector) ObserveRaw(row rbnHistoryRow) {
	if c == nil {
		return
	}
	call := spot.NormalizeCallsign(row.DXCall)
	if call == "" || row.Time.IsZero() {
		return
	}
	if row.FreqKHz <= 0 {
		return
	}
	c.spots[call] = append(c.spots[call], stabilitypkg.Spot{
		Ts:   row.Time.UTC().Unix(),
		Freq: row.FreqKHz,
	})
}

// ObserveApplied records only corrections that would have changed output.
func (c *replayStabilityCollector) ObserveApplied(tsUnix int64, winner string, freqKHz float64, band string) {
	if c == nil {
		return
	}
	winner = spot.NormalizeCallsign(winner)
	if winner == "" || tsUnix <= 0 || freqKHz <= 0 {
		return
	}
	band = spot.NormalizeBand(band)
	if band == "" || band == "???" {
		band = spot.NormalizeBand(spot.FreqToBand(freqKHz))
	}
	if band == "" {
		band = "unknown"
	}
	c.corrections = append(c.corrections, stabilitypkg.Correction{
		Ts:     tsUnix,
		Winner: winner,
		Freq:   freqKHz,
		Band:   band,
	})
}

// Evaluate delegates the scoring mechanics to the shared stability package but
// keeps replay-specific defaults and JSON shape here.
func (c *replayStabilityCollector) Evaluate(minTs int64) replayStabilitySummary {
	cfg := replayStabilityConfig{}
	if c != nil {
		cfg = c.cfg
	}
	cfg = normalizeReplayStabilityConfig(cfg)
	summary := replayStabilitySummary{
		WindowMinutes:   cfg.WindowMinutes,
		MinFollowOn:     cfg.MinFollowOn,
		FreqToleranceHz: cfg.FreqToleranceHz,
	}
	if c == nil {
		return summary
	}
	stabilityCfg := stabilitypkg.NormalizeConfig(stabilitypkg.Config{
		BucketMinutes:   stabilityBucketMinutes,
		WindowMinutes:   cfg.WindowMinutes,
		MinFollowOn:     cfg.MinFollowOn,
		FreqToleranceHz: cfg.FreqToleranceHz,
	})
	result := stabilitypkg.EvaluateFollowOn(c.corrections, c.spots, minTs, stabilityCfg)
	summary.TotalApplied = result.TotalCount
	summary.StableApplied = result.StableCount
	if summary.TotalApplied > 0 {
		summary.StablePct = roundFloat((100.0*float64(summary.StableApplied))/float64(summary.TotalApplied), 3)
	}
	return summary
}
