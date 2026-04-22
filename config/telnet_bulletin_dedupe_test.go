package config

import (
	"strings"
	"testing"
)

func TestLoadAppliesTelnetBulletinDedupeDefaults(t *testing.T) {
	dir := testConfigDir(t)
	writeRequiredFloodControlFile(t, dir)

	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if got := cfg.Telnet.BulletinDedupeWindowSeconds; got != 600 {
		t.Fatalf("expected bulletin dedupe window default 600, got %d", got)
	}
	if got := cfg.Telnet.BulletinDedupeMaxEntries; got != 4096 {
		t.Fatalf("expected bulletin dedupe max default 4096, got %d", got)
	}
}

func TestLoadAllowsDisabledTelnetBulletinDedupe(t *testing.T) {
	dir := testConfigDir(t)
	writeRequiredFloodControlFile(t, dir)
	cfgText := `telnet:
  bulletin_dedupe_window_seconds: 0
`
	writeTestConfigOverlay(t, dir, "runtime.yaml", cfgText)

	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if got := cfg.Telnet.BulletinDedupeWindowSeconds; got != 0 {
		t.Fatalf("expected disabled bulletin dedupe window 0, got %d", got)
	}
	if got := cfg.Telnet.BulletinDedupeMaxEntries; got != 4096 {
		t.Fatalf("expected configured max entries to remain 4096 when window disables dedupe, got %d", got)
	}
}

func TestLoadRejectsNegativeTelnetBulletinDedupeWindow(t *testing.T) {
	dir := testConfigDir(t)
	writeRequiredFloodControlFile(t, dir)
	cfgText := `telnet:
  bulletin_dedupe_window_seconds: -1
`
	writeTestConfigOverlay(t, dir, "runtime.yaml", cfgText)

	_, err := Load(dir)
	if err == nil {
		t.Fatalf("expected error for negative bulletin dedupe window")
	}
	if !strings.Contains(err.Error(), "telnet.bulletin_dedupe_window_seconds") {
		t.Fatalf("expected bulletin dedupe window error, got %v", err)
	}
}

func TestLoadRejectsInvalidTelnetBulletinDedupeMaxEntriesWhenEnabled(t *testing.T) {
	dir := testConfigDir(t)
	writeRequiredFloodControlFile(t, dir)
	cfgText := `telnet:
  bulletin_dedupe_window_seconds: 60
  bulletin_dedupe_max_entries: 0
`
	writeTestConfigOverlay(t, dir, "runtime.yaml", cfgText)

	_, err := Load(dir)
	if err == nil {
		t.Fatalf("expected error for zero max entries when bulletin dedupe is enabled")
	}
	if !strings.Contains(err.Error(), "telnet.bulletin_dedupe_max_entries") {
		t.Fatalf("expected bulletin dedupe max error, got %v", err)
	}
}

func TestLoadAcceptsExplicitTelnetBulletinDedupeSettings(t *testing.T) {
	dir := testConfigDir(t)
	writeRequiredFloodControlFile(t, dir)
	cfgText := `telnet:
  bulletin_dedupe_window_seconds: 120
  bulletin_dedupe_max_entries: 64
`
	writeTestConfigOverlay(t, dir, "runtime.yaml", cfgText)

	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if got := cfg.Telnet.BulletinDedupeWindowSeconds; got != 120 {
		t.Fatalf("expected explicit bulletin dedupe window 120, got %d", got)
	}
	if got := cfg.Telnet.BulletinDedupeMaxEntries; got != 64 {
		t.Fatalf("expected explicit bulletin dedupe max 64, got %d", got)
	}
}
