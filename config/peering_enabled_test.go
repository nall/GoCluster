package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPeeringPeerEnabledDefaultsFalseWhenOmitted(t *testing.T) {
	dir := t.TempDir()
	writeRequiredFloodControlFile(t, dir)
	cfgText := `peering:
  enabled: true
  peers:
    - host: "peer-omitted.example.net"
      port: 7300
`
	if err := os.WriteFile(filepath.Join(dir, "peering.yaml"), []byte(cfgText), 0o644); err != nil {
		t.Fatalf("write peering.yaml: %v", err)
	}

	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if len(cfg.Peering.Peers) != 1 {
		t.Fatalf("expected 1 peer, got %d", len(cfg.Peering.Peers))
	}
	if cfg.Peering.Peers[0].Enabled {
		t.Fatalf("expected omitted peering.peers[0].enabled to default false")
	}
}

func TestPeeringPeerEnabledHonorsExplicitFalse(t *testing.T) {
	dir := t.TempDir()
	writeRequiredFloodControlFile(t, dir)
	cfgText := `peering:
  enabled: true
  peers:
    - enabled: false
      host: "peer-disabled.example.net"
      port: 7300
`
	if err := os.WriteFile(filepath.Join(dir, "peering.yaml"), []byte(cfgText), 0o644); err != nil {
		t.Fatalf("write peering.yaml: %v", err)
	}

	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if len(cfg.Peering.Peers) != 1 {
		t.Fatalf("expected 1 peer, got %d", len(cfg.Peering.Peers))
	}
	if cfg.Peering.Peers[0].Enabled {
		t.Fatalf("expected explicit peering.peers[0].enabled=false to remain false")
	}
}

func TestPeeringPeerEnabledHonorsExplicitTrue(t *testing.T) {
	dir := t.TempDir()
	writeRequiredFloodControlFile(t, dir)
	cfgText := `peering:
  enabled: true
  peers:
    - enabled: true
      host: "peer-enabled.example.net"
      port: 7300
`
	if err := os.WriteFile(filepath.Join(dir, "peering.yaml"), []byte(cfgText), 0o644); err != nil {
		t.Fatalf("write peering.yaml: %v", err)
	}

	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if len(cfg.Peering.Peers) != 1 {
		t.Fatalf("expected 1 peer, got %d", len(cfg.Peering.Peers))
	}
	if !cfg.Peering.Peers[0].Enabled {
		t.Fatalf("expected explicit peering.peers[0].enabled=true to remain true")
	}
}
