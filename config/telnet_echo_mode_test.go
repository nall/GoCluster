package config

import (
	"testing"
)

func TestLoadRejectsUnknownTelnetEchoMode(t *testing.T) {
	dir := testConfigDir(t)
	writeRequiredFloodControlFile(t, dir)
	config := `telnet:
  echo_mode: "invalid"
`
	writeTestConfigOverlay(t, dir, "runtime.yaml", config)
	if _, err := Load(dir); err == nil {
		t.Fatalf("expected error for unknown telnet echo mode")
	}
}
