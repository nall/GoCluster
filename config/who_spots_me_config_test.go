package config

import (
	"strings"
	"testing"
)

func TestLoadWhoSpotsMeWindowMinutesFromShippedRuntimeYAML(t *testing.T) {
	dir := testConfigDir(t)

	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if cfg.WhoSpotsMe.WindowMinutes != 10 {
		t.Fatalf("expected who_spots_me.window_minutes=10, got %d", cfg.WhoSpotsMe.WindowMinutes)
	}
}

func TestLoadRejectsInvalidWhoSpotsMeWindowMinutes(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{
			name: "zero",
			body: "who_spots_me:\n  window_minutes: 0\n",
		},
		{
			name: "negative",
			body: "who_spots_me:\n  window_minutes: -1\n",
		},
		{
			name: "too_large",
			body: "who_spots_me:\n  window_minutes: 61\n",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dir := testConfigDir(t)
			writeTestConfigOverlay(t, dir, "runtime.yaml", tc.body)
			_, err := Load(dir)
			if err == nil {
				t.Fatal("expected Load() error")
			}
			if !strings.Contains(err.Error(), "who_spots_me.window_minutes") {
				t.Fatalf("expected who_spots_me.window_minutes error, got %v", err)
			}
		})
	}
}
