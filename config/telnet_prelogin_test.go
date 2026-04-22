package config

import (
	"strings"
	"testing"
)

func TestLoadRejectsInvalidTelnetPreloginSettings(t *testing.T) {
	dir := testConfigDir(t)
	writeRequiredFloodControlFile(t, dir)
	config := `telnet:
  max_prelogin_sessions: 0
`
	writeTestConfigOverlay(t, dir, "runtime.yaml", config)

	_, err := Load(dir)
	if err == nil {
		t.Fatalf("expected Load() error for invalid max_prelogin_sessions")
	}
	if !strings.Contains(err.Error(), "telnet.max_prelogin_sessions") {
		t.Fatalf("expected max_prelogin_sessions error, got %v", err)
	}
}

func TestLoadClampsPreloginConcurrencyToGlobalCap(t *testing.T) {
	dir := testConfigDir(t)
	writeRequiredFloodControlFile(t, dir)
	config := `telnet:
  max_prelogin_sessions: 2
  prelogin_concurrency_per_ip: 10
`
	writeTestConfigOverlay(t, dir, "runtime.yaml", config)
	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got := cfg.Telnet.PreloginConcurrencyPerIP; got != 2 {
		t.Fatalf("expected prelogin_concurrency_per_ip clamped to 2, got %d", got)
	}
}

func TestLoadRejectsAdmissionSampleRateOutOfRange(t *testing.T) {
	dir := testConfigDir(t)
	writeRequiredFloodControlFile(t, dir)
	config := `telnet:
  admission_log_sample_rate: -1
`
	writeTestConfigOverlay(t, dir, "runtime.yaml", config)
	_, err := Load(dir)
	if err == nil {
		t.Fatalf("expected Load() error for invalid admission_log_sample_rate")
	}
	if !strings.Contains(err.Error(), "telnet.admission_log_sample_rate") {
		t.Fatalf("expected admission_log_sample_rate error, got %v", err)
	}
}
