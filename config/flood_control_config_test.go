package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadRequiresFloodControlFile(t *testing.T) {
	dir := testConfigDir(t)
	if err := os.Remove(filepath.Join(dir, "floodcontrol.yaml")); err != nil {
		t.Fatalf("remove floodcontrol.yaml: %v", err)
	}

	_, err := Load(dir)
	if err == nil {
		t.Fatalf("expected missing floodcontrol.yaml to fail")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "floodcontrol.yaml") {
		t.Fatalf("expected missing floodcontrol.yaml error, got %v", err)
	}
}

func TestLoadRejectsIncompleteFloodControlConfig(t *testing.T) {
	dir := testConfigDir(t)
	cfgText := `flood_control:
  enabled: true
`
	replaceTestConfigFile(t, dir, "floodcontrol.yaml", cfgText)

	_, err := Load(dir)
	if err == nil {
		t.Fatalf("expected incomplete flood control config to fail")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "flood_control") {
		t.Fatalf("expected flood_control validation error, got %v", err)
	}
}

func TestLoadAcceptsRequiredFloodControlConfig(t *testing.T) {
	dir := testConfigDir(t)
	writeRequiredFloodControlFile(t, dir)
	cfgText := `dedup:
  cluster_window_seconds: 120
`
	writeTestConfigOverlay(t, dir, "dedupe.yaml", cfgText)

	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if !cfg.FloodControl.Enabled {
		t.Fatalf("expected flood control enabled")
	}
	if cfg.FloodControl.PartitionMode != FloodControlPartitionExactSourceType {
		t.Fatalf("expected partition mode %q, got %q", FloodControlPartitionExactSourceType, cfg.FloodControl.PartitionMode)
	}
	if cfg.FloodControl.Rails.DXCall.ActiveMode != floodModeConservative {
		t.Fatalf("expected dxcall active_mode=%q, got %q", floodModeConservative, cfg.FloodControl.Rails.DXCall.ActiveMode)
	}
}
