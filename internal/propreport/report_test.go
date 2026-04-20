package propreport

import (
	"testing"

	"dxcluster/pathreliability"
)

func TestParsePredictionTotalsWithAndWithoutStale(t *testing.T) {
	withStale := "2026/04/20 12:00:00 Path predictions (5m): total=1,200 derived=5 combined=700 insufficient=500 no_sample=300 low_weight=150 stale=50 override_r=0 override_g=0"
	got, ok := parsePredictionTotals(withStale)
	if !ok {
		t.Fatalf("expected prediction totals to parse")
	}
	if got.Total != 1200 || got.Combined != 700 || got.Insufficient != 500 || got.NoSample != 300 || got.LowWeight != 150 || got.Stale != 50 {
		t.Fatalf("unexpected parsed totals with stale: %+v", got)
	}

	withoutStale := "2026/04/20 12:00:00 Path predictions (5m): total=100 derived=2 combined=60 insufficient=40 no_sample=30 low_weight=10 override_r=0 override_g=0"
	got, ok = parsePredictionTotals(withoutStale)
	if !ok {
		t.Fatalf("expected legacy prediction totals to parse")
	}
	if got.Total != 100 || got.Combined != 60 || got.Insufficient != 40 || got.NoSample != 30 || got.LowWeight != 10 || got.Stale != 0 {
		t.Fatalf("unexpected parsed legacy totals: %+v", got)
	}
}

func TestBuildModelContextIncludesPredictionAgeGate(t *testing.T) {
	cfg := pathreliability.DefaultConfig()
	cfg.DefaultHalfLifeSec = 240
	cfg.BandHalfLifeSec = map[string]int{"20m": 360, "10m": 240}
	cfg.MaxPredictionAgeHalfLifeMultiplier = 1.25

	ctx := buildModelContext(cfg, []string{"20m", "10m"})
	if ctx.MaxPredictionAgeHalfLifeMultiplier != 1.25 {
		t.Fatalf("expected max prediction age multiplier 1.25, got %v", ctx.MaxPredictionAgeHalfLifeMultiplier)
	}
	if got := ctx.MaxPredictionAgeByBand["20m"]; got != 450 {
		t.Fatalf("expected 20m max age 450, got %d", got)
	}
	if got := ctx.MaxPredictionAgeByBand["10m"]; got != 300 {
		t.Fatalf("expected 10m max age 300, got %d", got)
	}

	cfg.MaxPredictionAgeHalfLifeMultiplier = 0
	ctx = buildModelContext(cfg, []string{"20m"})
	if got := ctx.MaxPredictionAgeByBand["20m"]; got != 0 {
		t.Fatalf("expected disabled max age 0, got %d", got)
	}
}
