package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPeeringPeerFamilyAndDirectionDefault(t *testing.T) {
	dir := t.TempDir()
	writeRequiredFloodControlFile(t, dir)
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
	if got := cfg.Peering.Peers[0].Family; got != PeeringPeerFamilyDXSpider {
		t.Fatalf("expected default family %q, got %q", PeeringPeerFamilyDXSpider, got)
	}
	if got := cfg.Peering.Peers[0].Direction; got != PeeringPeerDirectionOutbound {
		t.Fatalf("expected default direction %q, got %q", PeeringPeerDirectionOutbound, got)
	}
}

func TestPeeringPeerRejectsInvalidFamily(t *testing.T) {
	assertPeeringLoadErrorContains(t, `peering:
  enabled: true
  peers:
    - enabled: true
      host: "peer.example.net"
      port: 7300
      family: "mystery"
`, "invalid peering.peers[0].family")
}

func TestPeeringPeerRejectsInvalidDirection(t *testing.T) {
	assertPeeringLoadErrorContains(t, `peering:
  enabled: true
  peers:
    - enabled: true
      host: "peer.example.net"
      port: 7300
      direction: "sideways"
`, "invalid peering.peers[0].direction")
}

func TestPeeringInboundPeerRequiresRemoteCallsign(t *testing.T) {
	assertPeeringLoadErrorContains(t, `peering:
  enabled: true
  peers:
    - enabled: true
      direction: "inbound"
`, "remote_callsign")
}

func TestPeeringOutboundPeerRequiresHostAndPort(t *testing.T) {
	assertPeeringLoadErrorContains(t, `peering:
  enabled: true
  peers:
    - enabled: true
      direction: "both"
      remote_callsign: "REMOTE"
      host: ""
`, "host")

	assertPeeringLoadErrorContains(t, `peering:
  enabled: true
  peers:
    - enabled: true
      direction: "outbound"
      host: "peer.example.net"
`, "port")
}

func TestPeeringPeerRejectsInvalidAllowIPs(t *testing.T) {
	assertPeeringLoadErrorContains(t, `peering:
  enabled: true
  peers:
    - enabled: true
      direction: "inbound"
      remote_callsign: "REMOTE"
      allow_ips:
        - "not-a-cidr"
`, "allow_ips")
}

func assertPeeringLoadErrorContains(t *testing.T, cfgText string, want string) {
	t.Helper()
	dir := t.TempDir()
	writeRequiredFloodControlFile(t, dir)
	if err := os.WriteFile(filepath.Join(dir, "peering.yaml"), []byte(cfgText), 0o644); err != nil {
		t.Fatalf("write peering.yaml: %v", err)
	}
	_, err := Load(dir)
	if err == nil {
		t.Fatalf("expected Load() error containing %q, got nil", want)
	}
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("expected Load() error containing %q, got %v", want, err)
	}
}
