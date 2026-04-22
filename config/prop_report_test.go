package config

import (
	"testing"
)

func TestPropReportRefreshDefault(t *testing.T) {
	dir := testConfigDir(t)
	writeRequiredFloodControlFile(t, dir)
	cfg := `prop_report:
  enabled: true
`
	writeTestConfigOverlay(t, dir, "prop_report.yaml", cfg)
	loaded, err := Load(dir)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if loaded.PropReport.RefreshUTC != "00:05" {
		t.Fatalf("expected default refresh_utc 00:05, got %q", loaded.PropReport.RefreshUTC)
	}
}

func TestPropReportRefreshInvalid(t *testing.T) {
	dir := testConfigDir(t)
	writeRequiredFloodControlFile(t, dir)
	cfg := `prop_report:
  enabled: true
  refresh_utc: "25:99"
`
	writeTestConfigOverlay(t, dir, "prop_report.yaml", cfg)
	if _, err := Load(dir); err == nil {
		t.Fatalf("expected Load() to fail for invalid prop_report.refresh_utc")
	}
}
