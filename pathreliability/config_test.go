package pathreliability

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
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

func writeTempConfigOverlay(t *testing.T, contents string) string {
	t.Helper()
	base, err := os.ReadFile(filepath.Join("..", "data", "config", "path_reliability.yaml"))
	if err != nil {
		t.Fatalf("read shipped path reliability config: %v", err)
	}
	var merged map[string]any
	if err := yaml.Unmarshal(base, &merged); err != nil {
		t.Fatalf("parse shipped path reliability config: %v", err)
	}
	var override map[string]any
	if err := yaml.Unmarshal([]byte(contents), &override); err != nil {
		t.Fatalf("parse override path reliability config: %v", err)
	}
	merged = mergeTestYAMLMaps(merged, override)
	data, err := yaml.Marshal(merged)
	if err != nil {
		t.Fatalf("marshal override path reliability config: %v", err)
	}
	return writeTempConfig(t, string(data))
}

func writeTempConfigWithoutKey(t *testing.T, path ...string) string {
	t.Helper()
	base, err := os.ReadFile(filepath.Join("..", "data", "config", "path_reliability.yaml"))
	if err != nil {
		t.Fatalf("read shipped path reliability config: %v", err)
	}
	var doc map[string]any
	if err := yaml.Unmarshal(base, &doc); err != nil {
		t.Fatalf("parse shipped path reliability config: %v", err)
	}
	current := doc
	for _, key := range path[:len(path)-1] {
		next, ok := current[key].(map[string]any)
		if !ok {
			t.Fatalf("test path %s missing before final key", strings.Join(path, "."))
		}
		current = next
	}
	delete(current, path[len(path)-1])
	data, err := yaml.Marshal(doc)
	if err != nil {
		t.Fatalf("marshal config without %s: %v", strings.Join(path, "."), err)
	}
	return writeTempConfig(t, string(data))
}

func mergeTestYAMLMaps(dst, src map[string]any) map[string]any {
	if dst == nil {
		dst = make(map[string]any)
	}
	for key, val := range src {
		if existing, ok := dst[key]; ok {
			existingMap, okExisting := existing.(map[string]any)
			incomingMap, okIncoming := val.(map[string]any)
			if okExisting && okIncoming {
				dst[key] = mergeTestYAMLMaps(existingMap, incomingMap)
				continue
			}
		}
		dst[key] = val
	}
	return dst
}

func TestLoadFileRejectsLegacyThresholdKeys(t *testing.T) {
	path := writeTempConfigOverlay(t, `
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
	path := writeTempConfigOverlay(t, `
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

func TestDefaultMinObservationCount(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.MinObservationCount != 19 {
		t.Fatalf("default min observation count = %v, want 19", cfg.MinObservationCount)
	}
}

func TestLoadFileRejectsNegativeMaxPredictionAgeMultiplier(t *testing.T) {
	path := writeTempConfigOverlay(t, `
max_prediction_age_half_life_multiplier: -1
`)
	_, err := LoadFile(path)
	if err == nil {
		t.Fatalf("expected negative max prediction age multiplier to fail")
	}
	if !strings.Contains(err.Error(), "max_prediction_age_half_life_multiplier") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadFileRejectsNonPositiveMinObservationCount(t *testing.T) {
	path := writeTempConfigOverlay(t, `
min_observation_count: 0
`)
	_, err := LoadFile(path)
	if err == nil {
		t.Fatalf("expected non-positive min observation count to fail")
	}
	if !strings.Contains(err.Error(), "min_observation_count") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadFileRejectsNegativeNoisePenalty(t *testing.T) {
	path := writeTempConfigOverlay(t, `
noise_offsets_by_band:
  rural:
    160M: -3
`)
	_, err := LoadFile(path)
	if err == nil {
		t.Fatalf("expected negative noise penalty to fail")
	}
	if !strings.Contains(err.Error(), "noise_offsets_by_band.RURAL.160m") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadFilePreservesExplicitFT4Zero(t *testing.T) {
	cfg, err := LoadFile(filepath.Join("..", "data", "config", "path_reliability.yaml"))
	if err != nil {
		t.Fatalf("load shipped config: %v", err)
	}
	if cfg.ModeOffsets.FT4 != 0 {
		t.Fatalf("expected explicit mode_offsets.ft4=0 to survive load, got %v", cfg.ModeOffsets.FT4)
	}
}

func TestLoadFileRejectsMissingRequiredYAMLSettings(t *testing.T) {
	cases := []struct {
		name string
		path []string
		want string
	}{
		{name: "enabled", path: []string{"enabled"}, want: "enabled"},
		{name: "display enabled", path: []string{"display_enabled"}, want: "display_enabled"},
		{name: "min observation count", path: []string{"min_observation_count"}, want: "min_observation_count"},
		{name: "ft4 offset", path: []string{"mode_offsets", "ft4"}, want: "mode_offsets.ft4"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := LoadFile(writeTempConfigWithoutKey(t, tc.path...))
			if err == nil {
				t.Fatalf("expected missing %s to fail", tc.want)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("expected error to mention %s, got %v", tc.want, err)
			}
		})
	}
}

func TestLoadFileRejectsNullRequiredYAMLSetting(t *testing.T) {
	path := writeTempConfigOverlay(t, `
mode_offsets:
  ft4:
`)
	_, err := LoadFile(path)
	if err == nil {
		t.Fatalf("expected null mode_offsets.ft4 to fail")
	}
	if !strings.Contains(err.Error(), "mode_offsets.ft4") {
		t.Fatalf("expected error to mention mode_offsets.ft4, got %v", err)
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
