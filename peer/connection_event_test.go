package peer

import "testing"

func TestManagerReportsConnectionEvent(t *testing.T) {
	m := &Manager{}
	var got ConnectionEvent
	m.SetConnectionReporter(func(ev ConnectionEvent) {
		got = ev
	})
	m.reportConnection(ConnectionEvent{
		Direction: "outbound",
		Action:    "dial_failed",
		Peer:      "N0PEER-1",
		Endpoint:  "peer.example:7300",
		Reason:    "connection_refused",
	})
	if got.Action != "dial_failed" || got.Peer != "N0PEER-1" || got.Endpoint != "peer.example:7300" {
		t.Fatalf("unexpected event: %+v", got)
	}
}

func TestDirectionLabel(t *testing.T) {
	if got := directionLabel(dirInbound); got != "inbound" {
		t.Fatalf("dirInbound label = %q", got)
	}
	if got := directionLabel(dirOutbound); got != "outbound" {
		t.Fatalf("dirOutbound label = %q", got)
	}
}
