package spot

import (
	"testing"
	"time"
)

func TestModeAssignerDXCacheHit(t *testing.T) {
	now := time.Date(2025, 1, 2, 3, 4, 5, 0, time.UTC)
	assigner := newModeAssigner(ModeInferenceSettings{
		DXFreqCacheTTL:        5 * time.Minute,
		DXFreqCacheSize:       10,
		DigitalWindow:         5 * time.Minute,
		DigitalMinCorroborate: 2,
		DigitalSeedTTL:        5 * time.Minute,
		DigitalCacheSize:      10,
	}, func() time.Time { return now }, func(*Spot) RegionalModeResult {
		return RegionalModeResult{Mode: "LSB", Provenance: ModeProvenanceRegionalVoiceDefault}
	})

	explicit := NewSpot("K1ABC", "W1AAA", 14074.0, "FT8")
	explicit.IsHuman = false
	explicit.SourceType = SourcePSKReporter
	assigner.Assign(explicit, true)

	missing := NewSpot("K1ABC", "W1BBB", 14074.0, "")
	missing.IsHuman = false
	missing.SourceType = SourcePSKReporter
	mode := assigner.Assign(missing, false)
	if mode != "FT8" {
		t.Fatalf("expected DX+freq cache to return FT8, got %q", mode)
	}
}

func TestModeAssignerDigitalSeedExpires(t *testing.T) {
	now := time.Date(2025, 1, 2, 3, 4, 5, 0, time.UTC)
	assigner := newModeAssigner(ModeInferenceSettings{
		DXFreqCacheTTL:        1 * time.Minute,
		DXFreqCacheSize:       10,
		DigitalWindow:         5 * time.Minute,
		DigitalMinCorroborate: 10,
		DigitalSeedTTL:        5 * time.Second,
		DigitalCacheSize:      10,
		DigitalSeeds: []ModeSeed{
			{FrequencyKHz: 7074, Mode: "FT8"},
		},
	}, func() time.Time { return now }, func(*Spot) RegionalModeResult {
		return RegionalModeResult{Mode: "LSB", Provenance: ModeProvenanceRegionalVoiceDefault}
	})

	seeded := NewSpot("K1ABC", "W1AAA", 7074.0, "")
	mode := assigner.Assign(seeded, false)
	if mode != "FT8" {
		t.Fatalf("expected seed to infer FT8, got %q", mode)
	}

	now = now.Add(10 * time.Second)
	after := NewSpot("K2ABC", "W1CCC", 7074.0, "")
	mode = assigner.Assign(after, false)
	if mode != "LSB" {
		t.Fatalf("expected expired seed to fall back to regional label, got %q", mode)
	}
}

func TestModeAssignerDigitalCorroborators(t *testing.T) {
	now := time.Date(2025, 1, 2, 3, 4, 5, 0, time.UTC)
	assigner := newModeAssigner(ModeInferenceSettings{
		DXFreqCacheTTL:        1 * time.Minute,
		DXFreqCacheSize:       10,
		DigitalWindow:         5 * time.Minute,
		DigitalMinCorroborate: 2,
		DigitalSeedTTL:        10 * time.Minute,
		DigitalCacheSize:      10,
	}, func() time.Time { return now }, func(*Spot) RegionalModeResult {
		return RegionalModeResult{Mode: "LSB", Provenance: ModeProvenanceRegionalVoiceDefault}
	})

	first := NewSpot("K1ABC", "W1AAA", 14074.0, "FT8")
	first.IsHuman = false
	first.SourceType = SourcePSKReporter
	assigner.Assign(first, true)

	second := NewSpot("K1ABC", "W1BBB", 14074.0, "FT8")
	second.IsHuman = false
	second.SourceType = SourcePSKReporter
	assigner.Assign(second, true)

	missing := NewSpot("K2ABC", "W1CCC", 14074.0, "")
	missing.IsHuman = false
	missing.SourceType = SourcePSKReporter
	mode := assigner.Assign(missing, false)
	if mode != "FT8" {
		t.Fatalf("expected corroborators to infer FT8, got %q", mode)
	}
}

func TestModeAssignerIgnoresHumanForDigitalMap(t *testing.T) {
	now := time.Date(2025, 1, 2, 3, 4, 5, 0, time.UTC)
	assigner := newModeAssigner(ModeInferenceSettings{
		DXFreqCacheTTL:        1 * time.Minute,
		DXFreqCacheSize:       10,
		DigitalWindow:         5 * time.Minute,
		DigitalMinCorroborate: 1,
		DigitalSeedTTL:        10 * time.Minute,
		DigitalCacheSize:      10,
	}, func() time.Time { return now }, func(*Spot) RegionalModeResult {
		return RegionalModeResult{Mode: "LSB", Provenance: ModeProvenanceRegionalVoiceDefault}
	})

	human := NewSpot("K1ABC", "W1AAA", 14074.0, "FT8")
	human.IsHuman = true
	human.SourceType = SourceManual
	assigner.Assign(human, true)

	missing := NewSpot("K2ABC", "W1BBB", 14074.0, "")
	missing.IsHuman = false
	missing.SourceType = SourcePSKReporter
	mode := assigner.Assign(missing, false)
	if mode != "LSB" {
		t.Fatalf("expected human evidence to be ignored, got %q", mode)
	}
}

func TestModeAssignerRegionalDefaultDoesNotSeedDXCache(t *testing.T) {
	now := time.Date(2025, 1, 2, 3, 4, 5, 0, time.UTC)
	defaultMode := "LSB"
	assigner := newModeAssigner(ModeInferenceSettings{
		DXFreqCacheTTL:        5 * time.Minute,
		DXFreqCacheSize:       10,
		DigitalWindow:         5 * time.Minute,
		DigitalMinCorroborate: 2,
		DigitalSeedTTL:        5 * time.Minute,
		DigitalCacheSize:      10,
	}, func() time.Time { return now }, func(*Spot) RegionalModeResult {
		return RegionalModeResult{Mode: defaultMode, Provenance: ModeProvenanceRegionalVoiceDefault}
	})

	first := NewSpot("K1ABC", "W1AAA", 1843.0, "")
	first.SourceType = SourceManual
	mode := assigner.Assign(first, false)
	if mode != "LSB" {
		t.Fatalf("expected first regional default LSB, got %q", mode)
	}
	if first.ModeProvenance != ModeProvenanceRegionalVoiceDefault {
		t.Fatalf("expected regional voice provenance, got %q", first.ModeProvenance)
	}

	defaultMode = "USB"
	second := NewSpot("K1ABC", "W1BBB", 1843.0, "")
	second.SourceType = SourceManual
	mode = assigner.Assign(second, false)
	if mode != "USB" {
		t.Fatalf("expected second lookup to re-run fallback, got %q", mode)
	}
	if second.ModeProvenance != ModeProvenanceRegionalVoiceDefault {
		t.Fatalf("expected second regional voice provenance, got %q", second.ModeProvenance)
	}
}

func TestModeAssignerCacheHitMarksRecentEvidence(t *testing.T) {
	now := time.Date(2025, 1, 2, 3, 4, 5, 0, time.UTC)
	assigner := newModeAssigner(ModeInferenceSettings{
		DXFreqCacheTTL:        5 * time.Minute,
		DXFreqCacheSize:       10,
		DigitalWindow:         5 * time.Minute,
		DigitalMinCorroborate: 2,
		DigitalSeedTTL:        5 * time.Minute,
		DigitalCacheSize:      10,
	}, func() time.Time { return now }, func(*Spot) RegionalModeResult {
		return RegionalModeResult{Mode: "LSB", Provenance: ModeProvenanceRegionalVoiceDefault}
	})

	explicit := NewSpot("K1ABC", "W1AAA", 14074.0, "FT8")
	explicit.SourceType = SourcePSKReporter
	assigner.Assign(explicit, true)

	missing := NewSpot("K1ABC", "W1BBB", 14074.0, "")
	missing.SourceType = SourceManual
	mode := assigner.Assign(missing, false)
	if mode != "FT8" {
		t.Fatalf("expected cached FT8, got %q", mode)
	}
	if missing.ModeProvenance != ModeProvenanceRecentEvidence {
		t.Fatalf("expected recent-evidence provenance, got %q", missing.ModeProvenance)
	}
}
