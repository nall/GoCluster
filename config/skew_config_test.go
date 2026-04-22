package config

import "testing"

// Purpose: Verify skew min_abs_skew defaults to 1 when omitted.
// Key aspects: Ensures selection threshold is active by default.
// Upstream: go test.
// Downstream: Load.
func TestSkewMinAbsSkewDefault(t *testing.T) {
	dir := testConfigDir(t)
	writeRequiredFloodControlFile(t, dir)
	cfgText := "skew:\n  enabled: true\n"
	writeTestConfigOverlay(t, dir, "data.yaml", cfgText)

	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if cfg.Skew.MinAbsSkew != 1 {
		t.Fatalf("expected skew.min_abs_skew=1, got %v", cfg.Skew.MinAbsSkew)
	}
}

// Purpose: Verify explicit skew min_abs_skew values are preserved.
// Key aspects: Confirms no normalization override for positive values.
// Upstream: go test.
// Downstream: Load.
func TestSkewMinAbsSkewOverride(t *testing.T) {
	dir := testConfigDir(t)
	writeRequiredFloodControlFile(t, dir)
	cfgText := "skew:\n  enabled: true\n  min_abs_skew: 1.25\n"
	writeTestConfigOverlay(t, dir, "data.yaml", cfgText)

	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if cfg.Skew.MinAbsSkew != 1.25 {
		t.Fatalf("expected skew.min_abs_skew=1.25, got %v", cfg.Skew.MinAbsSkew)
	}
}
