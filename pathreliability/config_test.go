package pathreliability

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeTempConfig(t *testing.T, contents string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "path_reliability.yaml")
	if err := os.WriteFile(path, []byte(contents), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	return path
}

func TestLoadFileRejectsLegacyThresholdKeys(t *testing.T) {
	path := writeTempConfig(t, `
glyph_thresholds:
  excellent: -13
  good: -17
  marginal: -21
`)
	_, err := LoadFile(path)
	if err == nil {
		t.Fatalf("expected legacy threshold keys to fail")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "unsupported glyph threshold key") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadFileRejectsInvalidGlyphSymbols(t *testing.T) {
	path := writeTempConfig(t, `
glyph_symbols:
  high: "++"
`)
	_, err := LoadFile(path)
	if err == nil {
		t.Fatalf("expected invalid glyph symbol to fail")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "glyph_symbols.high") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDefaultNoiseOffsetsByBand(t *testing.T) {
	cfg := DefaultConfig()
	model := cfg.NoiseModel()
	cases := []struct {
		class   string
		band    string
		penalty float64
	}{
		{"QUIET", "160m", 0},
		{"RURAL", "160m", 6},
		{"RURAL", "6m", 0},
		{"SUBURBAN", "40m", 11},
		{"URBAN", "160m", 22},
		{"URBAN", "6m", 3},
		{"INDUSTRIAL", "160m", 28},
		{"INDUSTRIAL", "6m", 5},
	}
	for _, tc := range cases {
		if got := model.Penalty(tc.class, tc.band); got != tc.penalty {
			t.Fatalf("Penalty(%s, %s) = %v, want %v", tc.class, tc.band, got, tc.penalty)
		}
	}
}

func TestDefaultMaxPredictionAgeMultiplier(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.MaxPredictionAgeHalfLifeMultiplier != 1.25 {
		t.Fatalf("default max prediction age multiplier = %v, want 1.25", cfg.MaxPredictionAgeHalfLifeMultiplier)
	}
}

func TestLoadFileNegativeMaxPredictionAgeMultiplierDisablesGate(t *testing.T) {
	path := writeTempConfig(t, `
max_prediction_age_half_life_multiplier: -1
`)
	cfg, err := LoadFile(path)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.MaxPredictionAgeHalfLifeMultiplier != 0 {
		t.Fatalf("negative max prediction age multiplier normalized to %v, want 0", cfg.MaxPredictionAgeHalfLifeMultiplier)
	}
}

func TestLoadFileNoiseOffsetsByBandNormalizesAndFillsDefaults(t *testing.T) {
	path := writeTempConfig(t, `
noise_offsets_by_band:
  quiet:
    160M: 0
  rural:
    160M: -3
  suburban:
    20M: 8
  urban:
    20M: 12
  industrial:
    6M: 6
`)
	cfg, err := LoadFile(path)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	model := cfg.NoiseModel()
	if !model.HasClass("quiet") {
		t.Fatalf("expected quiet class to be valid")
	}
	if got := model.Penalty("RURAL", "160m"); got != 0 {
		t.Fatalf("expected negative rural override to clamp to 0, got %v", got)
	}
	if got := model.Penalty("SUBURBAN", "20m"); got != 8 {
		t.Fatalf("expected suburban 20m override, got %v", got)
	}
	if got := model.Penalty("URBAN", "6m"); got != 3 {
		t.Fatalf("expected missing urban 6m to fill from defaults, got %v", got)
	}
	if got := model.Penalty("INDUSTRIAL", "6m"); got != 6 {
		t.Fatalf("expected industrial 6m override, got %v", got)
	}
}

func TestLoadFileRejectsLegacyNoiseOffsets(t *testing.T) {
	path := writeTempConfig(t, `
noise_offsets:
  quiet: 0
`)
	_, err := LoadFile(path)
	if err == nil {
		t.Fatalf("expected legacy noise_offsets to fail")
	}
	if !strings.Contains(err.Error(), "noise_offsets is no longer supported") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadFileRejectsMissingNoiseClass(t *testing.T) {
	path := writeTempConfig(t, `
noise_offsets_by_band:
  quiet:
    20m: 0
  rural:
    20m: 3
  suburban:
    20m: 7
  urban:
    20m: 11
`)
	_, err := LoadFile(path)
	if err == nil {
		t.Fatalf("expected missing industrial class to fail")
	}
	if !strings.Contains(err.Error(), "missing required class INDUSTRIAL") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadFileRejectsMalformedNoiseOffsetsByBand(t *testing.T) {
	path := writeTempConfig(t, `
noise_offsets_by_band:
  quiet: 0
`)
	_, err := LoadFile(path)
	if err == nil {
		t.Fatalf("expected malformed noise_offsets_by_band to fail")
	}
	if !strings.Contains(err.Error(), "noise_offsets_by_band") {
		t.Fatalf("unexpected error: %v", err)
	}
}
