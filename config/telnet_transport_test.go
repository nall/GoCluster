package config

import (
	"testing"
)

func TestLoadRejectsUnknownTelnetTransport(t *testing.T) {
	dir := testConfigDir(t)
	writeRequiredFloodControlFile(t, dir)
	config := `telnet:
  transport: "unsupported"
`
	writeTestConfigOverlay(t, dir, "runtime.yaml", config)
	if _, err := Load(dir); err == nil {
		t.Fatalf("expected error for unknown telnet transport")
	}
}
