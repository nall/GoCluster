package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoggingDropDedupeWindowDefault(t *testing.T) {
	dir := t.TempDir()
	writeRequiredFloodControlFile(t, dir)
	cfgText := `logging:
  enabled: true
`
	if err := os.WriteFile(filepath.Join(dir, "app.yaml"), []byte(cfgText), 0o644); err != nil {
		t.Fatalf("write app.yaml: %v", err)
	}

	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if cfg.Logging.DropDedupeWindowSeconds != 120 {
		t.Fatalf("expected default drop_dedupe_window_seconds=120, got %d", cfg.Logging.DropDedupeWindowSeconds)
	}
}

func TestLoggingDropDedupeWindowAllowsZero(t *testing.T) {
	dir := t.TempDir()
	writeRequiredFloodControlFile(t, dir)
	cfgText := `logging:
  drop_dedupe_window_seconds: 0
`
	if err := os.WriteFile(filepath.Join(dir, "app.yaml"), []byte(cfgText), 0o644); err != nil {
		t.Fatalf("write app.yaml: %v", err)
	}

	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if cfg.Logging.DropDedupeWindowSeconds != 0 {
		t.Fatalf("expected drop_dedupe_window_seconds=0, got %d", cfg.Logging.DropDedupeWindowSeconds)
	}
}

func TestLoggingDropDedupeWindowRejectsNegative(t *testing.T) {
	dir := t.TempDir()
	writeRequiredFloodControlFile(t, dir)
	cfgText := `logging:
  drop_dedupe_window_seconds: -1
`
	if err := os.WriteFile(filepath.Join(dir, "app.yaml"), []byte(cfgText), 0o644); err != nil {
		t.Fatalf("write app.yaml: %v", err)
	}

	if _, err := Load(dir); err == nil {
		t.Fatalf("expected Load() to fail for negative logging.drop_dedupe_window_seconds")
	}
}

func TestDroppedCallLoggingDefaultsDisabledWithPerFileLogsEnabled(t *testing.T) {
	dir := t.TempDir()
	writeRequiredFloodControlFile(t, dir)
	cfgText := `logging:
  enabled: true
`
	if err := os.WriteFile(filepath.Join(dir, "app.yaml"), []byte(cfgText), 0o644); err != nil {
		t.Fatalf("write app.yaml: %v", err)
	}

	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if cfg.Logging.DroppedCalls.Enabled {
		t.Fatalf("expected dropped-call logging disabled by default")
	}
	if cfg.Logging.DroppedCalls.Dir != filepath.Join("data", "logs", "dropped_calls") {
		t.Fatalf("unexpected dropped-call dir %q", cfg.Logging.DroppedCalls.Dir)
	}
	if cfg.Logging.DroppedCalls.RetentionDays != 7 {
		t.Fatalf("expected retention_days=7, got %d", cfg.Logging.DroppedCalls.RetentionDays)
	}
	if cfg.Logging.DroppedCalls.DedupeWindowSeconds != 120 {
		t.Fatalf("expected dedupe_window_seconds=120, got %d", cfg.Logging.DroppedCalls.DedupeWindowSeconds)
	}
	if !cfg.Logging.DroppedCalls.BadDEDX || !cfg.Logging.DroppedCalls.NoLicense || !cfg.Logging.DroppedCalls.Harmonics {
		t.Fatalf("expected all dropped-call file toggles enabled by default: %+v", cfg.Logging.DroppedCalls)
	}
}

func TestDroppedCallLoggingHonorsExplicitToggles(t *testing.T) {
	dir := t.TempDir()
	writeRequiredFloodControlFile(t, dir)
	cfgText := `logging:
  dropped_calls:
    enabled: true
    dir: "custom/drop"
    retention_days: 3
    dedupe_window_seconds: 0
    bad_de_dx: false
    no_license: false
    harmonics: false
`
	if err := os.WriteFile(filepath.Join(dir, "app.yaml"), []byte(cfgText), 0o644); err != nil {
		t.Fatalf("write app.yaml: %v", err)
	}

	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if !cfg.Logging.DroppedCalls.Enabled {
		t.Fatalf("expected dropped-call logging enabled")
	}
	if cfg.Logging.DroppedCalls.Dir != "custom/drop" {
		t.Fatalf("unexpected dropped-call dir %q", cfg.Logging.DroppedCalls.Dir)
	}
	if cfg.Logging.DroppedCalls.RetentionDays != 3 {
		t.Fatalf("expected retention_days=3, got %d", cfg.Logging.DroppedCalls.RetentionDays)
	}
	if cfg.Logging.DroppedCalls.DedupeWindowSeconds != 0 {
		t.Fatalf("expected dedupe_window_seconds=0, got %d", cfg.Logging.DroppedCalls.DedupeWindowSeconds)
	}
	if cfg.Logging.DroppedCalls.BadDEDX || cfg.Logging.DroppedCalls.NoLicense || cfg.Logging.DroppedCalls.Harmonics {
		t.Fatalf("expected all dropped-call file toggles disabled: %+v", cfg.Logging.DroppedCalls)
	}
}

func TestDroppedCallLoggingRejectsNegativeBounds(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{
			name: "retention",
			body: `logging:
  dropped_calls:
    retention_days: -1
`,
		},
		{
			name: "dedupe",
			body: `logging:
  dropped_calls:
    dedupe_window_seconds: -1
`,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			writeRequiredFloodControlFile(t, dir)
			if err := os.WriteFile(filepath.Join(dir, "app.yaml"), []byte(tc.body), 0o644); err != nil {
				t.Fatalf("write app.yaml: %v", err)
			}
			if _, err := Load(dir); err == nil {
				t.Fatalf("expected Load() to reject %s", tc.name)
			}
		})
	}
}
