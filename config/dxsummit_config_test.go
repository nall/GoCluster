package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadDXSummitDefaultsAndOverrides(t *testing.T) {
	dir := t.TempDir()
	writeRequiredFloodControlFile(t, dir)
	cfgText := `dxsummit:
  enabled: true
  poll_interval_seconds: 30
  max_records_per_poll: 500
  request_timeout_ms: 10000
  lookback_seconds: 300
  include_bands: ["hf", "VHF", " UHF "]
`
	if err := os.WriteFile(filepath.Join(dir, "ingest.yaml"), []byte(cfgText), 0o644); err != nil {
		t.Fatalf("write ingest.yaml: %v", err)
	}

	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if !cfg.DXSummit.Enabled {
		t.Fatal("expected DXSummit enabled")
	}
	if cfg.DXSummit.Name != "DXSUMMIT" {
		t.Fatalf("unexpected name %q", cfg.DXSummit.Name)
	}
	if cfg.DXSummit.BaseURL != defaultDXSummitBaseURL {
		t.Fatalf("unexpected base URL %q", cfg.DXSummit.BaseURL)
	}
	if cfg.DXSummit.SpotChannelSize != defaultDXSummitSpotChannelSize {
		t.Fatalf("unexpected spot channel size %d", cfg.DXSummit.SpotChannelSize)
	}
	if cfg.DXSummit.MaxResponseBytes != defaultDXSummitMaxResponseBytes {
		t.Fatalf("unexpected max response bytes %d", cfg.DXSummit.MaxResponseBytes)
	}
	wantBands := []string{"HF", "VHF", "UHF"}
	if strings.Join(cfg.DXSummit.IncludeBands, ",") != strings.Join(wantBands, ",") {
		t.Fatalf("include bands = %#v, want %#v", cfg.DXSummit.IncludeBands, wantBands)
	}
}

func TestLoadRejectsInvalidDXSummitConfig(t *testing.T) {
	tests := []struct {
		name    string
		body    string
		wantErr string
	}{
		{
			name: "poll interval",
			body: `dxsummit:
  poll_interval_seconds: 0
`,
			wantErr: "poll_interval_seconds",
		},
		{
			name: "max records",
			body: `dxsummit:
  max_records_per_poll: 10001
`,
			wantErr: "max_records_per_poll",
		},
		{
			name: "timeout too high",
			body: `dxsummit:
  poll_interval_seconds: 30
  request_timeout_ms: 30000
`,
			wantErr: "request_timeout_ms",
		},
		{
			name: "lookback too short",
			body: `dxsummit:
  poll_interval_seconds: 30
  lookback_seconds: 10
`,
			wantErr: "lookback_seconds",
		},
		{
			name: "startup backfill",
			body: `dxsummit:
  lookback_seconds: 300
  startup_backfill_seconds: 301
`,
			wantErr: "startup_backfill_seconds",
		},
		{
			name: "bad band",
			body: `dxsummit:
  include_bands: ["HF", "SHF"]
`,
			wantErr: "include_bands",
		},
		{
			name: "bad url",
			body: `dxsummit:
  base_url: "ftp://www.dxsummit.fi/api/v1/spots"
`,
			wantErr: "base_url",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			writeRequiredFloodControlFile(t, dir)
			if err := os.WriteFile(filepath.Join(dir, "ingest.yaml"), []byte(tt.body), 0o644); err != nil {
				t.Fatalf("write ingest.yaml: %v", err)
			}
			_, err := Load(dir)
			if err == nil {
				t.Fatal("expected Load() error")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("expected error containing %q, got %v", tt.wantErr, err)
			}
		})
	}
}
