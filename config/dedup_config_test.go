package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDedupSecondaryPreferStrongDefaultsFromPrimaryWhenOmitted(t *testing.T) {
	dir := t.TempDir()
	writeRequiredFloodControlFile(t, dir)
	cfgText := `dedup:
  prefer_stronger_snr: true
`
	if err := os.WriteFile(filepath.Join(dir, "dedup.yaml"), []byte(cfgText), 0o644); err != nil {
		t.Fatalf("write dedup.yaml: %v", err)
	}

	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if !cfg.Dedup.SecondaryFastPreferStrong {
		t.Fatalf("expected secondary_fast_prefer_stronger_snr to follow prefer_stronger_snr=true when omitted")
	}
	if !cfg.Dedup.SecondaryMedPreferStrong {
		t.Fatalf("expected secondary_med_prefer_stronger_snr to follow prefer_stronger_snr=true when omitted")
	}
	if !cfg.Dedup.SecondarySlowPreferStrong {
		t.Fatalf("expected secondary_slow_prefer_stronger_snr to follow prefer_stronger_snr=true when omitted")
	}
}

func TestDedupLegacySecondaryKeysRemainIgnored(t *testing.T) {
	dir := t.TempDir()
	writeRequiredFloodControlFile(t, dir)
	cfgText := `dedup:
  secondary_window_seconds: 999
  secondary_prefer_stronger_snr: true
`
	if err := os.WriteFile(filepath.Join(dir, "dedup.yaml"), []byte(cfgText), 0o644); err != nil {
		t.Fatalf("write dedup.yaml: %v", err)
	}

	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if cfg.Dedup.SecondaryFastWindowSeconds != 120 {
		t.Fatalf("expected legacy secondary_window_seconds to be ignored; got secondary_fast_window_seconds=%d", cfg.Dedup.SecondaryFastWindowSeconds)
	}
	if cfg.Dedup.SecondaryMedWindowSeconds != 300 {
		t.Fatalf("expected legacy secondary_window_seconds to be ignored; got secondary_med_window_seconds=%d", cfg.Dedup.SecondaryMedWindowSeconds)
	}
	if cfg.Dedup.SecondarySlowWindowSeconds != 480 {
		t.Fatalf("expected legacy secondary_window_seconds to be ignored; got secondary_slow_window_seconds=%d", cfg.Dedup.SecondarySlowWindowSeconds)
	}
	if cfg.Dedup.SecondaryFastPreferStrong {
		t.Fatalf("expected legacy secondary_prefer_stronger_snr to be ignored for secondary_fast_prefer_stronger_snr")
	}
	if cfg.Dedup.SecondaryMedPreferStrong {
		t.Fatalf("expected legacy secondary_prefer_stronger_snr to be ignored for secondary_med_prefer_stronger_snr")
	}
	if cfg.Dedup.SecondarySlowPreferStrong {
		t.Fatalf("expected legacy secondary_prefer_stronger_snr to be ignored for secondary_slow_prefer_stronger_snr")
	}
}
