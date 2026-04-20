package peer

import (
	"testing"
	"time"

	"dxcluster/config"
)

func inboundPeer(family string) config.PeeringPeer {
	return config.PeeringPeer{
		Enabled:        true,
		Direction:      config.PeeringPeerDirectionInbound,
		Family:         family,
		RemoteCallsign: "REMOTE",
	}
}

func TestInboundHandshakeConfiguredPeerBehavior(t *testing.T) {
	t.Run("unknown inbound peer is rejected before pc18", func(t *testing.T) {
		runInboundScenario(t, inboundScenario{
			name:             "unknown inbound peer is rejected before pc18",
			globalAllowCalls: []string{"REMOTE"},
			wantRegistered:   false,
			wantRemoteCall:   "",
			wantPC9x:         true,
			steps: []handshakeStep{
				{kind: handshakeExpectTx, matcher: exactLine("login:")},
				{kind: handshakeSendRx, line: "REMOTE"},
				{kind: handshakeAwaitResult, timeout: 500 * time.Millisecond, errCheck: errContains("unauthorized inbound peer")},
			},
		})
	})

	t.Run("disabled inbound peer is rejected before pc18", func(t *testing.T) {
		runInboundScenario(t, inboundScenario{
			name: "disabled inbound peer is rejected before pc18",
			peers: []config.PeeringPeer{{
				Enabled:        false,
				Direction:      config.PeeringPeerDirectionInbound,
				Family:         config.PeeringPeerFamilyDXSpider,
				RemoteCallsign: "REMOTE",
			}},
			wantRegistered: false,
			wantRemoteCall: "",
			wantPC9x:       true,
			steps: []handshakeStep{
				{kind: handshakeExpectTx, matcher: exactLine("login:")},
				{kind: handshakeSendRx, line: "REMOTE"},
				{kind: handshakeAwaitResult, timeout: 500 * time.Millisecond, errCheck: errContains("unauthorized inbound peer")},
			},
		})
	})

	t.Run("dxspider banner without pc20 still times out", func(t *testing.T) {
		runInboundScenario(t, inboundScenario{
			name:           "dxspider banner without pc20 still times out",
			peers:          []config.PeeringPeer{inboundPeer(config.PeeringPeerFamilyDXSpider)},
			wantRegistered: false,
			wantRemoteCall: "REMOTE",
			wantPC9x:       true,
			steps: []handshakeStep{
				{kind: handshakeExpectTx, matcher: exactLine("login:")},
				{kind: handshakeSendRx, line: "REMOTE"},
				{kind: handshakeExpectTx, matcher: exactLine("PC18^DXSpider Version: 5457 gocluster pc9x^5457^")},
				{kind: handshakeSendRx, line: "PC18^DXSpider Version: 1.57 [pc9x 91]^1.57^"},
				{kind: handshakeAwaitResult, timeout: 500 * time.Millisecond, errCheck: errIsTimeout},
			},
		})
	})

	t.Run("dxspider pc92 without pc20 still times out", func(t *testing.T) {
		runInboundScenario(t, inboundScenario{
			name:           "dxspider pc92 without pc20 still times out",
			peers:          []config.PeeringPeer{inboundPeer(config.PeeringPeerFamilyDXSpider)},
			wantRegistered: false,
			wantRemoteCall: "REMOTE",
			wantPC9x:       true,
			wantPC92Queued: 1,
			steps: []handshakeStep{
				{kind: handshakeExpectTx, matcher: exactLine("login:")},
				{kind: handshakeSendRx, line: "REMOTE"},
				{kind: handshakeExpectTx, matcher: exactLine("PC18^DXSpider Version: 5457 gocluster pc9x^5457^")},
				{kind: handshakeSendRx, line: "PC92^REMOTE^123^A^^5REMOTE:6.0^H99^"},
				{kind: handshakeAwaitResult, timeout: 500 * time.Millisecond, errCheck: errIsTimeout},
			},
		})
	})

	t.Run("ccluster banner establishes without pc20", func(t *testing.T) {
		runInboundScenario(t, inboundScenario{
			name:           "ccluster banner establishes without pc20",
			peers:          []config.PeeringPeer{inboundPeer(config.PeeringPeerFamilyCCluster)},
			wantRegistered: true,
			wantRemoteCall: "REMOTE",
			wantPC9x:       true,
			steps: []handshakeStep{
				{kind: handshakeExpectTx, matcher: exactLine("login:")},
				{kind: handshakeSendRx, line: "REMOTE"},
				{kind: handshakeExpectTx, matcher: exactLine("PC18^DXSpider Version: 5457 gocluster pc9x^5457^")},
				{kind: handshakeSendRx, line: "PC18^CC Cluster Version: 6.0^6.0^"},
				{kind: handshakeExpectTx, matcher: pc92TypeLine("A")},
				{kind: handshakeExpectTx, matcher: pc92TypeLine("K")},
				{kind: handshakeExpectTx, matcher: exactLine("PC20^")},
				{kind: handshakeAwaitRegistered, timeout: 250 * time.Millisecond},
				{kind: handshakeCloseRemote},
				{kind: handshakeAwaitResult, timeout: 500 * time.Millisecond, errCheck: errIsEOFOrClosedPipe},
			},
		})
	})

	t.Run("ccluster pc92 establishes without prior banner and uses peer login callsign", func(t *testing.T) {
		peer := inboundPeer(config.PeeringPeerFamilyCCluster)
		peer.LoginCallsign = "LOCAL-9"
		runInboundScenario(t, inboundScenario{
			name:           "ccluster pc92 establishes without prior banner and uses peer login callsign",
			peers:          []config.PeeringPeer{peer},
			wantRegistered: true,
			wantRemoteCall: "REMOTE",
			wantPC9x:       true,
			wantPC92Queued: 1,
			steps: []handshakeStep{
				{kind: handshakeExpectTx, matcher: exactLine("login:")},
				{kind: handshakeSendRx, line: "REMOTE"},
				{kind: handshakeExpectTx, matcher: exactLine("PC18^DXSpider Version: 5457 gocluster pc9x^5457^")},
				{kind: handshakeSendRx, line: "PC92^REMOTE^123^A^^5REMOTE:6.0^H99^"},
				{kind: handshakeExpectTx, matcher: pc92CallTypeLine("LOCAL-9", "A")},
				{kind: handshakeExpectTx, matcher: pc92CallTypeLine("LOCAL-9", "K")},
				{kind: handshakeExpectTx, matcher: exactLine("PC20^")},
				{kind: handshakeAwaitRegistered, timeout: 250 * time.Millisecond},
				{kind: handshakeCloseRemote},
				{kind: handshakeAwaitResult, timeout: 500 * time.Millisecond, errCheck: errIsEOFOrClosedPipe},
			},
		})
	})

	t.Run("ccluster banner plus pc92 establishes and processes topology", func(t *testing.T) {
		runInboundScenario(t, inboundScenario{
			name:           "ccluster banner plus pc92 establishes and processes topology",
			peers:          []config.PeeringPeer{inboundPeer(config.PeeringPeerFamilyCCluster)},
			wantRegistered: true,
			wantRemoteCall: "REMOTE",
			wantPC9x:       true,
			wantPC92Queued: 1,
			steps: []handshakeStep{
				{kind: handshakeExpectTx, matcher: exactLine("login:")},
				{kind: handshakeSendRx, line: "REMOTE"},
				{kind: handshakeExpectTx, matcher: exactLine("PC18^DXSpider Version: 5457 gocluster pc9x^5457^")},
				{kind: handshakeSendRx, line: "PC18^CC Cluster Version: 6.0 [pc9x 91]^6.0^"},
				{kind: handshakeExpectTx, matcher: pc92TypeLine("A")},
				{kind: handshakeExpectTx, matcher: pc92TypeLine("K")},
				{kind: handshakeExpectTx, matcher: exactLine("PC20^")},
				{kind: handshakeAwaitRegistered, timeout: 250 * time.Millisecond},
				{kind: handshakeSendRx, line: "PC92^REMOTE^123^A^^5REMOTE:6.0^H99^"},
				{kind: handshakeCloseRemote},
				{kind: handshakeAwaitResult, timeout: 500 * time.Millisecond, errCheck: errIsEOFOrClosedPipe},
			},
		})
	})

	t.Run("late pc20 after ccluster pc92 establish gets pc22", func(t *testing.T) {
		runInboundScenario(t, inboundScenario{
			name:           "late pc20 after ccluster pc92 establish gets pc22",
			peers:          []config.PeeringPeer{inboundPeer(config.PeeringPeerFamilyCCluster)},
			wantRegistered: true,
			wantRemoteCall: "REMOTE",
			wantPC9x:       true,
			wantPC92Queued: 1,
			steps: []handshakeStep{
				{kind: handshakeExpectTx, matcher: exactLine("login:")},
				{kind: handshakeSendRx, line: "REMOTE"},
				{kind: handshakeExpectTx, matcher: exactLine("PC18^DXSpider Version: 5457 gocluster pc9x^5457^")},
				{kind: handshakeSendRx, line: "PC92^REMOTE^123^A^^5REMOTE:6.0^H99^"},
				{kind: handshakeExpectTx, matcher: pc92TypeLine("A")},
				{kind: handshakeExpectTx, matcher: pc92TypeLine("K")},
				{kind: handshakeExpectTx, matcher: exactLine("PC20^")},
				{kind: handshakeAwaitRegistered, timeout: 250 * time.Millisecond},
				{kind: handshakeSendRx, line: "PC20^"},
				{kind: handshakeExpectTx, matcher: exactLine("PC22^")},
				{kind: handshakeCloseRemote},
				{kind: handshakeAwaitResult, timeout: 500 * time.Millisecond, errCheck: errIsEOFOrClosedPipe},
			},
		})
	})

	t.Run("late pc20 after ccluster establish gets pc22", func(t *testing.T) {
		runInboundScenario(t, inboundScenario{
			name:           "late pc20 after ccluster establish gets pc22",
			peers:          []config.PeeringPeer{inboundPeer(config.PeeringPeerFamilyCCluster)},
			wantRegistered: true,
			wantRemoteCall: "REMOTE",
			wantPC9x:       true,
			steps: []handshakeStep{
				{kind: handshakeExpectTx, matcher: exactLine("login:")},
				{kind: handshakeSendRx, line: "REMOTE"},
				{kind: handshakeExpectTx, matcher: exactLine("PC18^DXSpider Version: 5457 gocluster pc9x^5457^")},
				{kind: handshakeSendRx, line: "PC18^CC Cluster Version: 6.0 [pc9x 91]^6.0^"},
				{kind: handshakeExpectTx, matcher: pc92TypeLine("A")},
				{kind: handshakeExpectTx, matcher: pc92TypeLine("K")},
				{kind: handshakeExpectTx, matcher: exactLine("PC20^")},
				{kind: handshakeAwaitRegistered, timeout: 250 * time.Millisecond},
				{kind: handshakeSendRx, line: "PC20^"},
				{kind: handshakeExpectTx, matcher: exactLine("PC22^")},
				{kind: handshakeCloseRemote},
				{kind: handshakeAwaitResult, timeout: 500 * time.Millisecond, errCheck: errIsEOFOrClosedPipe},
			},
		})
	})

	t.Run("dxspider configured peer rejects cc banner", func(t *testing.T) {
		runInboundScenario(t, inboundScenario{
			name:           "dxspider configured peer rejects cc banner",
			peers:          []config.PeeringPeer{inboundPeer(config.PeeringPeerFamilyDXSpider)},
			wantRegistered: false,
			wantRemoteCall: "REMOTE",
			wantPC9x:       true,
			steps: []handshakeStep{
				{kind: handshakeExpectTx, matcher: exactLine("login:")},
				{kind: handshakeSendRx, line: "REMOTE"},
				{kind: handshakeExpectTx, matcher: exactLine("PC18^DXSpider Version: 5457 gocluster pc9x^5457^")},
				{kind: handshakeSendRx, line: "PC18^CC Cluster Version: 6.0 [pc9x 91]^6.0^"},
				{kind: handshakeAwaitResult, timeout: 500 * time.Millisecond, errCheck: errContains("family mismatch")},
			},
		})
	})

	t.Run("ccluster configured peer rejects dxspider banner", func(t *testing.T) {
		runInboundScenario(t, inboundScenario{
			name:           "ccluster configured peer rejects dxspider banner",
			peers:          []config.PeeringPeer{inboundPeer(config.PeeringPeerFamilyCCluster)},
			wantRegistered: false,
			wantRemoteCall: "REMOTE",
			wantPC9x:       true,
			steps: []handshakeStep{
				{kind: handshakeExpectTx, matcher: exactLine("login:")},
				{kind: handshakeSendRx, line: "REMOTE"},
				{kind: handshakeExpectTx, matcher: exactLine("PC18^DXSpider Version: 5457 gocluster pc9x^5457^")},
				{kind: handshakeSendRx, line: "PC18^DXSpider Version: 1.57 [pc9x 91]^1.57^"},
				{kind: handshakeAwaitResult, timeout: 500 * time.Millisecond, errCheck: errContains("family mismatch")},
			},
		})
	})

	t.Run("legacy startup flips to non-pc9x before completion", func(t *testing.T) {
		runInboundScenario(t, inboundScenario{
			name:             "legacy startup flips to non-pc9x before completion",
			peers:            []config.PeeringPeer{inboundPeer(config.PeeringPeerFamilyDXSpider)},
			wantRegistered:   true,
			wantRemoteCall:   "REMOTE",
			wantPC9x:         false,
			wantLegacyQueued: 1,
			steps: []handshakeStep{
				{kind: handshakeExpectTx, matcher: exactLine("login:")},
				{kind: handshakeSendRx, line: "REMOTE"},
				{kind: handshakeExpectTx, matcher: exactLine("PC18^DXSpider Version: 5457 gocluster pc9x^5457^")},
				{kind: handshakeSendRx, line: "PC19^1^REMOTE^0^1.57^H99^"},
				{kind: handshakeSendRx, line: "PC20^"},
				{kind: handshakeExpectTx, matcher: prefixLine("PC19^1^N0CALL^0^1.57^H99^")},
				{kind: handshakeExpectTx, matcher: exactLine("PC20^")},
				{kind: handshakeExpectTx, matcher: exactLine("PC22^")},
				{kind: handshakeAwaitRegistered, timeout: 250 * time.Millisecond},
				{kind: handshakeCloseRemote},
				{kind: handshakeAwaitResult, timeout: 500 * time.Millisecond, errCheck: errIsEOFOrClosedPipe},
			},
		})
	})
}
