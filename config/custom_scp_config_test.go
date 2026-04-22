package config

import (
	"testing"
)

func TestCustomSCPHistoryAndStaticHorizonDefaults(t *testing.T) {
	dir := testConfigDir(t)
	writeRequiredFloodControlFile(t, dir)
	pipeline := "call_correction:\n  custom_scp:\n    enabled: true\n"
	writeTestConfigOverlay(t, dir, "pipeline.yaml", pipeline)

	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if got := cfg.CallCorrection.CustomSCP.HistoryHorizonDays; got != 60 {
		t.Fatalf("expected history_horizon_days default 60, got %d", got)
	}
	if got := cfg.CallCorrection.CustomSCP.StaticHorizonDays; got != 395 {
		t.Fatalf("expected static_horizon_days default 395, got %d", got)
	}
}
