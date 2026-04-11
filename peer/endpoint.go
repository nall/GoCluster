package peer

import (
	"dxcluster/config"
	"dxcluster/strutil"
	"net"
	"strconv"
)

// PeerEndpoint wraps a configured peer.
//
//nolint:revive // Retained for package API clarity.
type PeerEndpoint struct {
	host       string
	port       int
	loginCall  string
	remoteCall string
	password   string
	preferPC9x bool
	family     string
	allowIPs   []*net.IPNet
}

// Purpose: Build a peer endpoint from configuration.
// Key aspects: Copies connection credentials and preferences.
// Upstream: Peer manager initialization.
// Downstream: PeerEndpoint.ID and session creation.
func newPeerEndpoint(p config.PeeringPeer) (PeerEndpoint, error) {
	if p.Family == "" {
		p.Family = config.PeeringPeerFamilyDXSpider
	}
	allowIPs, err := parseIPACL(p.AllowIPs)
	if err != nil {
		return PeerEndpoint{}, err
	}
	return PeerEndpoint{
		host:       p.Host,
		port:       p.Port,
		loginCall:  strutil.NormalizeUpper(p.LoginCallsign),
		remoteCall: strutil.NormalizeUpper(p.RemoteCallsign),
		password:   p.Password,
		preferPC9x: p.PreferPC9x,
		family:     p.Family,
		allowIPs:   allowIPs,
	}, nil
}

// ID returns a stable identifier for this peer.
// Key aspects: Prefers remote callsign; falls back to host:port for peers
// without an explicit remote identity so different services on the same host
// remain distinct.
// Upstream: Peer manager maps and logs.
// Downstream: None.
func (p PeerEndpoint) ID() string {
	if p.remoteCall != "" {
		return p.remoteCall
	}
	if p.host != "" && p.port > 0 {
		return net.JoinHostPort(p.host, strconv.Itoa(p.port))
	}
	return p.host
}
