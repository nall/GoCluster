package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadAppliesTelnetPreloginDefaults(t *testing.T) {
	dir := t.TempDir()
	writeRequiredFloodControlFile(t, dir)
	config := `telnet:
  max_prelogin_sessions: 0
  prelogin_timeout_seconds: 0
  accept_rate_per_ip: 0
  accept_burst_per_ip: 0
  accept_rate_per_subnet: 0
  accept_burst_per_subnet: 0
  accept_rate_global: 0
  accept_burst_global: 0
  accept_rate_per_asn: 0
  accept_burst_per_asn: 0
  accept_rate_per_country: 0
  accept_burst_per_country: 0
  prelogin_concurrency_per_ip: 0
  admission_log_interval_seconds: 0
  admission_log_sample_rate: 2.5
  admission_log_max_reason_lines_per_interval: 0
  reject_workers: 0
  reject_queue_size: 0
  reject_write_deadline_ms: 0
  writer_batch_max_bytes: 0
  writer_batch_wait_ms: 0
`
	if err := os.WriteFile(filepath.Join(dir, "runtime.yaml"), []byte(config), 0o644); err != nil {
		t.Fatalf("write runtime.yaml: %v", err)
	}
	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got := cfg.Telnet.MaxPreloginSessions; got != 256 {
		t.Fatalf("expected max_prelogin_sessions default 256, got %d", got)
	}
	if got := cfg.Telnet.PreloginTimeoutSeconds; got != 15 {
		t.Fatalf("expected prelogin_timeout_seconds default 15, got %d", got)
	}
	if got := cfg.Telnet.AcceptRatePerIP; got != 3 {
		t.Fatalf("expected accept_rate_per_ip default 3, got %v", got)
	}
	if got := cfg.Telnet.AcceptBurstPerIP; got != 6 {
		t.Fatalf("expected accept_burst_per_ip default 6, got %d", got)
	}
	if got := cfg.Telnet.AcceptRatePerSubnet; got != 24 {
		t.Fatalf("expected accept_rate_per_subnet default 24, got %v", got)
	}
	if got := cfg.Telnet.AcceptBurstPerSubnet; got != 48 {
		t.Fatalf("expected accept_burst_per_subnet default 48, got %d", got)
	}
	if got := cfg.Telnet.AcceptRateGlobal; got != 300 {
		t.Fatalf("expected accept_rate_global default 300, got %v", got)
	}
	if got := cfg.Telnet.AcceptBurstGlobal; got != 600 {
		t.Fatalf("expected accept_burst_global default 600, got %d", got)
	}
	if got := cfg.Telnet.AcceptRatePerASN; got != 40 {
		t.Fatalf("expected accept_rate_per_asn default 40, got %v", got)
	}
	if got := cfg.Telnet.AcceptBurstPerASN; got != 80 {
		t.Fatalf("expected accept_burst_per_asn default 80, got %d", got)
	}
	if got := cfg.Telnet.AcceptRatePerCountry; got != 120 {
		t.Fatalf("expected accept_rate_per_country default 120, got %v", got)
	}
	if got := cfg.Telnet.AcceptBurstPerCountry; got != 240 {
		t.Fatalf("expected accept_burst_per_country default 240, got %d", got)
	}
	if got := cfg.Telnet.PreloginConcurrencyPerIP; got != 3 {
		t.Fatalf("expected prelogin_concurrency_per_ip default 3, got %d", got)
	}
	if got := cfg.Telnet.AdmissionLogIntervalSeconds; got != 10 {
		t.Fatalf("expected admission_log_interval_seconds default 10, got %d", got)
	}
	if got := cfg.Telnet.AdmissionLogSampleRate; got != 1 {
		t.Fatalf("expected admission_log_sample_rate clamped to 1, got %v", got)
	}
	if got := cfg.Telnet.AdmissionLogMaxReasonLinesPerInterval; got != 20 {
		t.Fatalf("expected admission_log_max_reason_lines_per_interval default 20, got %d", got)
	}
	if got := cfg.Telnet.RejectWorkers; got != 2 {
		t.Fatalf("expected reject_workers default 2, got %d", got)
	}
	if got := cfg.Telnet.RejectQueueSize; got != 1024 {
		t.Fatalf("expected reject_queue_size default 1024, got %d", got)
	}
	if got := cfg.Telnet.RejectWriteDeadlineMS; got != 500 {
		t.Fatalf("expected reject_write_deadline_ms default 500, got %d", got)
	}
	if got := cfg.Telnet.WriterBatchMaxBytes; got != 16384 {
		t.Fatalf("expected writer_batch_max_bytes default 16384, got %d", got)
	}
	if got := cfg.Telnet.WriterBatchWaitMS; got != 5 {
		t.Fatalf("expected writer_batch_wait_ms default 5, got %d", got)
	}
}

func TestLoadClampsPreloginConcurrencyToGlobalCap(t *testing.T) {
	dir := t.TempDir()
	writeRequiredFloodControlFile(t, dir)
	config := `telnet:
  max_prelogin_sessions: 2
  prelogin_concurrency_per_ip: 10
`
	if err := os.WriteFile(filepath.Join(dir, "runtime.yaml"), []byte(config), 0o644); err != nil {
		t.Fatalf("write runtime.yaml: %v", err)
	}
	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got := cfg.Telnet.PreloginConcurrencyPerIP; got != 2 {
		t.Fatalf("expected prelogin_concurrency_per_ip clamped to 2, got %d", got)
	}
}

func TestLoadClampsAdmissionSampleRateLowerBound(t *testing.T) {
	dir := t.TempDir()
	writeRequiredFloodControlFile(t, dir)
	config := `telnet:
  admission_log_sample_rate: -1
`
	if err := os.WriteFile(filepath.Join(dir, "runtime.yaml"), []byte(config), 0o644); err != nil {
		t.Fatalf("write runtime.yaml: %v", err)
	}
	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got := cfg.Telnet.AdmissionLogSampleRate; got != 0 {
		t.Fatalf("expected admission_log_sample_rate clamped to 0, got %v", got)
	}
}
