package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPeeringForwardSpotsDefaultsFalseWhenOmitted(t *testing.T) {
	dir := t.TempDir()
	cfgText := `peering:
  enabled: true
  peers:
    - enabled: true
      host: "peer.example.net"
      port: 7300
`
	if err := os.WriteFile(filepath.Join(dir, "peering.yaml"), []byte(cfgText), 0o644); err != nil {
		t.Fatalf("write peering.yaml: %v", err)
	}

	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if cfg.Peering.ForwardSpots {
		t.Fatalf("expected omitted peering.forward_spots to default false")
	}
}

func TestPeeringForwardSpotsHonorsExplicitFalse(t *testing.T) {
	dir := t.TempDir()
	cfgText := `peering:
  enabled: true
  forward_spots: false
  peers:
    - enabled: true
      host: "peer.example.net"
      port: 7300
`
	if err := os.WriteFile(filepath.Join(dir, "peering.yaml"), []byte(cfgText), 0o644); err != nil {
		t.Fatalf("write peering.yaml: %v", err)
	}

	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if cfg.Peering.ForwardSpots {
		t.Fatalf("expected explicit peering.forward_spots=false to remain false")
	}
}

func TestPeeringForwardSpotsHonorsExplicitTrue(t *testing.T) {
	dir := t.TempDir()
	cfgText := `peering:
  enabled: true
  forward_spots: true
  peers:
    - enabled: true
      host: "peer.example.net"
      port: 7300
`
	if err := os.WriteFile(filepath.Join(dir, "peering.yaml"), []byte(cfgText), 0o644); err != nil {
		t.Fatalf("write peering.yaml: %v", err)
	}

	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if !cfg.Peering.ForwardSpots {
		t.Fatalf("expected explicit peering.forward_spots=true to remain true")
	}
}
