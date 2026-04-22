package config

import (
	"path/filepath"
	"testing"
)

func TestLoggingDropDedupeWindowDefault(t *testing.T) {
	dir := testConfigDir(t)
	writeRequiredFloodControlFile(t, dir)
	cfgText := `logging:
  enabled: true
`
	writeTestConfigOverlay(t, dir, "app.yaml", cfgText)

	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if cfg.Logging.DropDedupeWindowSeconds != 120 {
		t.Fatalf("expected default drop_dedupe_window_seconds=120, got %d", cfg.Logging.DropDedupeWindowSeconds)
	}
}

func TestLoggingDropDedupeWindowAllowsZero(t *testing.T) {
	dir := testConfigDir(t)
	writeRequiredFloodControlFile(t, dir)
	cfgText := `logging:
  drop_dedupe_window_seconds: 0
`
	writeTestConfigOverlay(t, dir, "app.yaml", cfgText)

	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if cfg.Logging.DropDedupeWindowSeconds != 0 {
		t.Fatalf("expected drop_dedupe_window_seconds=0, got %d", cfg.Logging.DropDedupeWindowSeconds)
	}
}

func TestLoggingDropDedupeWindowRejectsNegative(t *testing.T) {
	dir := testConfigDir(t)
	writeRequiredFloodControlFile(t, dir)
	cfgText := `logging:
  drop_dedupe_window_seconds: -1
`
	writeTestConfigOverlay(t, dir, "app.yaml", cfgText)

	if _, err := Load(dir); err == nil {
		t.Fatalf("expected Load() to fail for negative logging.drop_dedupe_window_seconds")
	}
}

func TestDroppedCallLoggingUsesShippedYAMLWithPerFileLogsEnabled(t *testing.T) {
	dir := testConfigDir(t)
	writeRequiredFloodControlFile(t, dir)
	cfgText := `logging:
  enabled: true
`
	writeTestConfigOverlay(t, dir, "app.yaml", cfgText)

	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if !cfg.Logging.DroppedCalls.Enabled {
		t.Fatalf("expected dropped-call logging enabled from shipped YAML")
	}
	if filepath.Clean(cfg.Logging.DroppedCalls.Dir) != filepath.Join("data", "logs", "dropped_calls") {
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
	dir := testConfigDir(t)
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
	writeTestConfigOverlay(t, dir, "app.yaml", cfgText)

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
			dir := testConfigDir(t)
			writeRequiredFloodControlFile(t, dir)
			writeTestConfigOverlay(t, dir, "app.yaml", tc.body)
			if _, err := Load(dir); err == nil {
				t.Fatalf("expected Load() to reject %s", tc.name)
			}
		})
	}
}
