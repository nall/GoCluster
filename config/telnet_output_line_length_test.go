package config

import (
	"strings"
	"testing"
)

func TestLoadRejectsShortTelnetOutputLineLength(t *testing.T) {
	dir := testConfigDir(t)
	writeRequiredFloodControlFile(t, dir)
	config := `telnet:
  output_line_length: 64
`
	writeTestConfigOverlay(t, dir, "runtime.yaml", config)
	_, err := Load(dir)
	if err == nil {
		t.Fatalf("expected error for short telnet output line length")
	}
	if !strings.Contains(err.Error(), "telnet.output_line_length") {
		t.Fatalf("expected telnet.output_line_length error, got %v", err)
	}
}
