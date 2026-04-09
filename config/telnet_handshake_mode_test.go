package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDefaultsTelnetSkipHandshakeToMinimal(t *testing.T) {
	dir := t.TempDir()
	writeRequiredFloodControlFile(t, dir)
	config := "telnet:\n  port: 8300\n"
	if err := os.WriteFile(filepath.Join(dir, "runtime.yaml"), []byte(config), 0o644); err != nil {
		t.Fatalf("write runtime.yaml: %v", err)
	}
	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got := cfg.Telnet.SkipHandshake; got != TelnetHandshakeMinimal {
		t.Fatalf("expected omitted skip_handshake to default to %q, got %q", TelnetHandshakeMinimal, got)
	}
}

func TestLoadAcceptsStringTelnetSkipHandshakeModes(t *testing.T) {
	cases := []struct {
		name string
		raw  string
		want TelnetHandshakeMode
	}{
		{name: "full", raw: "full", want: TelnetHandshakeFull},
		{name: "minimal", raw: "minimal", want: TelnetHandshakeMinimal},
		{name: "none", raw: "none", want: TelnetHandshakeNone},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			writeRequiredFloodControlFile(t, dir)
			config := "telnet:\n  port: 8300\n  skip_handshake: \"" + tc.raw + "\"\n"
			if err := os.WriteFile(filepath.Join(dir, "runtime.yaml"), []byte(config), 0o644); err != nil {
				t.Fatalf("write runtime.yaml: %v", err)
			}
			cfg, err := Load(dir)
			if err != nil {
				t.Fatalf("Load: %v", err)
			}
			if got := cfg.Telnet.SkipHandshake; got != tc.want {
				t.Fatalf("expected skip_handshake %q, got %q", tc.want, got)
			}
		})
	}
}

func TestLoadAcceptsLegacyBooleanTelnetSkipHandshake(t *testing.T) {
	cases := []struct {
		name string
		raw  string
		want TelnetHandshakeMode
	}{
		{name: "false maps to full", raw: "false", want: TelnetHandshakeFull},
		{name: "true maps to none", raw: "true", want: TelnetHandshakeNone},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			writeRequiredFloodControlFile(t, dir)
			config := "telnet:\n  port: 8300\n  skip_handshake: " + tc.raw + "\n"
			if err := os.WriteFile(filepath.Join(dir, "runtime.yaml"), []byte(config), 0o644); err != nil {
				t.Fatalf("write runtime.yaml: %v", err)
			}
			cfg, err := Load(dir)
			if err != nil {
				t.Fatalf("Load: %v", err)
			}
			if got := cfg.Telnet.SkipHandshake; got != tc.want {
				t.Fatalf("expected skip_handshake %q, got %q", tc.want, got)
			}
		})
	}
}

func TestLoadRejectsUnknownTelnetSkipHandshakeMode(t *testing.T) {
	dir := t.TempDir()
	writeRequiredFloodControlFile(t, dir)
	config := "telnet:\n  port: 8300\n  skip_handshake: \"weird\"\n"
	if err := os.WriteFile(filepath.Join(dir, "runtime.yaml"), []byte(config), 0o644); err != nil {
		t.Fatalf("write runtime.yaml: %v", err)
	}
	if _, err := Load(dir); err == nil {
		t.Fatal("expected error for unknown telnet.skip_handshake mode")
	}
}
