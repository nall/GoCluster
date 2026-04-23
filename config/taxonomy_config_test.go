package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadRequiresSpotTaxonomyFile(t *testing.T) {
	dir := testConfigDir(t)
	if err := os.Remove(filepath.Join(dir, "spot_taxonomy.yaml")); err != nil {
		t.Fatalf("remove spot_taxonomy.yaml: %v", err)
	}
	if _, err := Load(dir); err == nil || !strings.Contains(err.Error(), "spot_taxonomy.yaml") {
		t.Fatalf("expected missing spot_taxonomy.yaml error, got %v", err)
	}
}

func TestLoadRejectsLegacyPSKReporterModeKeys(t *testing.T) {
	dir := testConfigDir(t)
	writeTestConfigOverlay(t, dir, "ingest.yaml", `
pskreporter:
  modes: [FT8]
`)
	if _, err := Load(dir); err == nil || !strings.Contains(err.Error(), "removed YAML setting \"pskreporter.modes\"") {
		t.Fatalf("expected legacy pskreporter.modes error, got %v", err)
	}
}

func TestLoadRejectsLegacyPSKReporterPathOnlyModeKeys(t *testing.T) {
	dir := testConfigDir(t)
	writeTestConfigOverlay(t, dir, "ingest.yaml", `
pskreporter:
  path_only_modes: [WSPR]
`)
	if _, err := Load(dir); err == nil || !strings.Contains(err.Error(), "removed YAML setting \"pskreporter.path_only_modes\"") {
		t.Fatalf("expected legacy pskreporter.path_only_modes error, got %v", err)
	}
}
