package config

import (
	"testing"
)

func TestReputationIPInfoBooleansDefaultTrueWhenOmitted(t *testing.T) {
	dir := testConfigDir(t)
	writeRequiredFloodControlFile(t, dir)
	cfgText := `reputation:
  enabled: true
`
	writeTestConfigOverlay(t, dir, "reputation.yaml", cfgText)

	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if !cfg.Reputation.IPInfoPebbleLoadIPv4 {
		t.Fatalf("expected ipinfo_pebble_load_ipv4 to default true when omitted")
	}
	if !cfg.Reputation.IPInfoDeleteCSVAfterImport {
		t.Fatalf("expected ipinfo_delete_csv_after_import to default true when omitted")
	}
	if !cfg.Reputation.IPInfoKeepGzip {
		t.Fatalf("expected ipinfo_keep_gzip to default true when omitted")
	}
	if !cfg.Reputation.IPInfoPebbleCleanup {
		t.Fatalf("expected ipinfo_pebble_cleanup to default true when omitted")
	}
	if !cfg.Reputation.IPInfoPebbleCompact {
		t.Fatalf("expected ipinfo_pebble_compact to default true when omitted")
	}
}

func TestReputationIPInfoBooleansHonorExplicitFalse(t *testing.T) {
	dir := testConfigDir(t)
	writeRequiredFloodControlFile(t, dir)
	cfgText := `reputation:
  enabled: true
  ipinfo_pebble_load_ipv4: false
  ipinfo_delete_csv_after_import: false
  ipinfo_keep_gzip: false
  ipinfo_pebble_cleanup: false
  ipinfo_pebble_compact: false
`
	writeTestConfigOverlay(t, dir, "reputation.yaml", cfgText)

	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if cfg.Reputation.IPInfoPebbleLoadIPv4 {
		t.Fatalf("expected explicit ipinfo_pebble_load_ipv4=false to remain false")
	}
	if cfg.Reputation.IPInfoDeleteCSVAfterImport {
		t.Fatalf("expected explicit ipinfo_delete_csv_after_import=false to remain false")
	}
	if cfg.Reputation.IPInfoKeepGzip {
		t.Fatalf("expected explicit ipinfo_keep_gzip=false to remain false")
	}
	if cfg.Reputation.IPInfoPebbleCleanup {
		t.Fatalf("expected explicit ipinfo_pebble_cleanup=false to remain false")
	}
	if cfg.Reputation.IPInfoPebbleCompact {
		t.Fatalf("expected explicit ipinfo_pebble_compact=false to remain false")
	}
}
