package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadRequiresFloodControlFile(t *testing.T) {
	dir := t.TempDir()
	cfgText := `dedup:
  cluster_window_seconds: 120
`
	if err := os.WriteFile(filepath.Join(dir, "dedupe.yaml"), []byte(cfgText), 0o644); err != nil {
		t.Fatalf("write dedupe.yaml: %v", err)
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
	dir := t.TempDir()
	cfgText := `flood_control:
  enabled: true
`
	if err := os.WriteFile(filepath.Join(dir, "floodcontrol.yaml"), []byte(cfgText), 0o644); err != nil {
		t.Fatalf("write floodcontrol.yaml: %v", err)
	}

	_, err := Load(dir)
	if err == nil {
		t.Fatalf("expected incomplete flood control config to fail")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "flood_control") {
		t.Fatalf("expected flood_control validation error, got %v", err)
	}
}

func TestLoadAcceptsRequiredFloodControlConfig(t *testing.T) {
	dir := t.TempDir()
	writeRequiredFloodControlFile(t, dir)
	cfgText := `dedup:
  cluster_window_seconds: 120
`
	if err := os.WriteFile(filepath.Join(dir, "dedupe.yaml"), []byte(cfgText), 0o644); err != nil {
		t.Fatalf("write dedupe.yaml: %v", err)
	}

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
