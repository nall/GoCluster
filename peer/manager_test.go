package peer

import (
	"context"
	"reflect"
	"strings"
	"testing"
	"time"

	"dxcluster/config"
	"dxcluster/spot"
)

// Ensure PC92 enqueue is non-blocking and drops when the queue is full.
func TestHandleFramePC92QueueDropsWhenFull(t *testing.T) {
	m := &Manager{
		topology: &topologyStore{},
		dedupe:   newDedupeCache(time.Minute),
		pc92Ch:   make(chan pc92Work, 1),
	}
	// Fill the queue so the next enqueue would block.
	m.pc92Ch <- pc92Work{}

	frame, err := ParseFrame("PC92^NODE1^123^A^^9CALL:ver^H2^")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	start := time.Now()
	m.HandleFrame(frame, &session{remoteCall: "TEST"})
	if time.Since(start) > time.Second {
		t.Fatalf("HandleFrame blocked with full queue")
	}
	if len(m.pc92Ch) != 1 {
		t.Fatalf("expected queue to remain full (drop), got len=%d", len(m.pc92Ch))
	}
}

func TestActiveSessionSSIDsSortedUnique(t *testing.T) {
	m := &Manager{
		sessions: map[string]*session{
			"a": {remoteCall: "n2wq-73"},
			"b": {remoteCall: " KM3T-44 "},
			"c": {remoteCall: "km3t-44"},
			"d": {remoteCall: "*"},
			"e": {remoteCall: ""},
			"f": nil,
		},
	}

	got := m.ActiveSessionSSIDs()
	want := []string{"KM3T-44", "N2WQ-73"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
}

func TestHandleFrameRelaysInboundPC11AndPC61ToOtherPeers(t *testing.T) {
	tests := []struct {
		name         string
		line         string
		targetPC9x   bool
		wantPrefix   string
		wantHopToken string
	}{
		{
			name:         "pc11 relays to legacy peer as pc11",
			line:         "PC11^14074.0^K1ABC^23-Dec-2025^2001Z^CQ TEST^W1XYZ^ORIGIN^H3^",
			targetPC9x:   false,
			wantPrefix:   "PC11^",
			wantHopToken: "^H2^",
		},
		{
			name:         "pc61 relays to pc9x peer as pc61",
			line:         "PC61^14074.0^K1ABC^23-Dec-2025^2001Z^CQ TEST^W1XYZ^ORIGIN^203.0.113.7^H3^",
			targetPC9x:   true,
			wantPrefix:   "PC61^",
			wantHopToken: "^H2^",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			src := &session{
				id:         "src",
				remoteCall: "SRC",
				ctx:        context.Background(),
				writeCh:    make(chan string, 1),
			}
			dst := &session{
				id:         "dst",
				remoteCall: "DST",
				pc9x:       tc.targetPC9x,
				ctx:        context.Background(),
				writeCh:    make(chan string, 1),
			}
			ingest := make(chan *spot.Spot, 1)
			m := &Manager{
				cfg:      config.PeeringConfig{ForwardSpots: true},
				dedupe:   newDedupeCache(time.Minute),
				ingest:   ingest,
				sessions: map[string]*session{"src": src, "dst": dst},
			}

			frame, err := ParseFrame(tc.line)
			if err != nil {
				t.Fatalf("ParseFrame: %v", err)
			}

			m.HandleFrame(frame, src)

			select {
			case got := <-ingest:
				if got == nil {
					t.Fatal("expected inbound spot to be ingested locally")
				}
			default:
				t.Fatal("expected inbound spot to be ingested locally")
			}

			select {
			case got := <-dst.writeCh:
				if !strings.HasPrefix(got, tc.wantPrefix) {
					t.Fatalf("expected relayed frame prefix %q, got %q", tc.wantPrefix, got)
				}
				if !strings.Contains(got, tc.wantHopToken) {
					t.Fatalf("expected relayed frame hop token %q, got %q", tc.wantHopToken, got)
				}
			default:
				t.Fatal("expected relay to destination peer")
			}

			select {
			case got := <-src.writeCh:
				t.Fatalf("expected source peer to be excluded from relay, got %q", got)
			default:
			}
		})
	}
}

func TestInboundSpotNotRelayedWhenLocalIngestQueueFull(t *testing.T) {
	src := &session{
		id:         "src",
		remoteCall: "SRC",
		ctx:        context.Background(),
		writeCh:    make(chan string, 1),
	}
	dst := &session{
		id:         "dst",
		remoteCall: "DST",
		pc9x:       true,
		ctx:        context.Background(),
		writeCh:    make(chan string, 1),
	}
	ingest := make(chan *spot.Spot, 1)
	ingest <- spot.NewSpot("BUSY1", "LOCAL", 14074.0, "FT8")
	m := &Manager{
		cfg:      config.PeeringConfig{ForwardSpots: true},
		dedupe:   newDedupeCache(time.Minute),
		ingest:   ingest,
		sessions: map[string]*session{"src": src, "dst": dst},
	}

	frame, err := ParseFrame("PC61^14074.0^K1ABC^23-Dec-2025^2001Z^CQ TEST^W1XYZ^ORIGIN^203.0.113.7^H3^")
	if err != nil {
		t.Fatalf("ParseFrame: %v", err)
	}

	m.HandleFrame(frame, src)

	if got := len(ingest); got != 1 {
		t.Fatalf("expected ingest queue to stay full after local drop, got len=%d", got)
	}
	select {
	case got := <-dst.writeCh:
		t.Fatalf("expected relay suppression when local ingest drops, got %q", got)
	default:
	}
}

func TestInboundSpotRelayedWhenLocallyAccepted(t *testing.T) {
	src := &session{
		id:         "src",
		remoteCall: "SRC",
		ctx:        context.Background(),
		writeCh:    make(chan string, 1),
	}
	dst := &session{
		id:         "dst",
		remoteCall: "DST",
		pc9x:       true,
		ctx:        context.Background(),
		writeCh:    make(chan string, 1),
	}
	ingest := make(chan *spot.Spot, 1)
	m := &Manager{
		cfg:      config.PeeringConfig{ForwardSpots: true},
		dedupe:   newDedupeCache(time.Minute),
		ingest:   ingest,
		sessions: map[string]*session{"src": src, "dst": dst},
	}

	frame, err := ParseFrame("PC61^14074.0^K1ABC^23-Dec-2025^2001Z^CQ TEST^W1XYZ^ORIGIN^203.0.113.7^H3^")
	if err != nil {
		t.Fatalf("ParseFrame: %v", err)
	}

	m.HandleFrame(frame, src)

	select {
	case got := <-ingest:
		if got == nil {
			t.Fatal("expected local ingest acceptance")
		}
	default:
		t.Fatal("expected local ingest acceptance")
	}

	select {
	case got := <-dst.writeCh:
		if !strings.HasPrefix(got, "PC61^") {
			t.Fatalf("expected relayed PC61 frame, got %q", got)
		}
	default:
		t.Fatal("expected relay after local acceptance")
	}
}

func TestHandleFramePC92DuplicateSuppressedBeforeTopologyEnqueue(t *testing.T) {
	src := &session{
		id:         "src",
		remoteCall: "SRC",
		ctx:        context.Background(),
		writeCh:    make(chan string, 1),
	}
	dst := &session{
		id:         "dst",
		remoteCall: "DST",
		pc9x:       true,
		ctx:        context.Background(),
		writeCh:    make(chan string, 2),
	}
	m := &Manager{
		topology: &topologyStore{},
		dedupe:   newDedupeCache(time.Minute),
		pc92Ch:   make(chan pc92Work, 4),
		sessions: map[string]*session{"src": src, "dst": dst},
	}

	first, err := ParseFrame("PC92^NODE1^123^A^^9CALL:ver:build:1.2.3.4^H95^")
	if err != nil {
		t.Fatalf("ParseFrame(first): %v", err)
	}
	second, err := ParseFrame("PC92^NODE1^123^A^^9CALL:ver:build:1.2.3.4^H94^")
	if err != nil {
		t.Fatalf("ParseFrame(second): %v", err)
	}

	m.HandleFrame(first, src)
	m.HandleFrame(second, src)

	if got := len(m.pc92Ch); got != 1 {
		t.Fatalf("expected 1 topology enqueue after duplicate suppression, got %d", got)
	}
	select {
	case <-dst.writeCh:
	default:
		t.Fatal("expected first frame to forward to destination peer")
	}
	select {
	case got := <-dst.writeCh:
		t.Fatalf("expected duplicate frame to be suppressed, got relayed %q", got)
	default:
	}
}

func TestPublishDXReceiveOnlyStillPublishesManualSpot(t *testing.T) {
	dst := &session{
		id:         "dst",
		remoteCall: "DST",
		pc9x:       true,
		ctx:        context.Background(),
		writeCh:    make(chan string, 1),
	}
	m := &Manager{
		cfg:       config.PeeringConfig{ForwardSpots: false, HopCount: 3},
		localCall: "LOCAL",
		sessions:  map[string]*session{"dst": dst},
	}

	sp := spot.NewSpot("K1ABC", "W1XYZ", 14074.0, "FT8")
	if !m.PublishDX(sp) {
		t.Fatal("expected manual DX spot to publish in receive-only mode")
	}

	select {
	case got := <-dst.writeCh:
		if !strings.HasPrefix(got, "PC61^") {
			t.Fatalf("expected PC61 publish to pc9x peer, got %q", got)
		}
		if !strings.Contains(got, "^H3^") {
			t.Fatalf("expected configured hop token in publish, got %q", got)
		}
	default:
		t.Fatal("expected DX publish to destination peer")
	}
}

func TestHandleFrameSpotRelaySuppressedWhenForwardSpotsDisabled(t *testing.T) {
	tests := []struct {
		name string
		line string
	}{
		{
			name: "pc11 ingest only",
			line: "PC11^14074.0^K1ABC^23-Dec-2025^2001Z^CQ TEST^W1XYZ^ORIGIN^H3^",
		},
		{
			name: "pc26 ingest only",
			line: "PC26^14074.0^K1ABC^23-Dec-2025^2001Z^CQ TEST^W1XYZ^ORIGIN^ ^H3^",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			src := &session{
				id:         "src",
				remoteCall: "SRC",
				ctx:        context.Background(),
				writeCh:    make(chan string, 1),
			}
			dst := &session{
				id:         "dst",
				remoteCall: "DST",
				pc9x:       true,
				ctx:        context.Background(),
				writeCh:    make(chan string, 1),
			}
			ingest := make(chan *spot.Spot, 1)
			m := &Manager{
				cfg:      config.PeeringConfig{ForwardSpots: false},
				dedupe:   newDedupeCache(time.Minute),
				ingest:   ingest,
				sessions: map[string]*session{"src": src, "dst": dst},
			}

			frame, err := ParseFrame(tc.line)
			if err != nil {
				t.Fatalf("ParseFrame: %v", err)
			}

			m.HandleFrame(frame, src)

			select {
			case got := <-ingest:
				if got == nil {
					t.Fatal("expected ingested spot, got nil")
				}
			default:
				t.Fatal("expected inbound spot to be ingested locally")
			}

			select {
			case got := <-dst.writeCh:
				t.Fatalf("expected relay suppression, got %q", got)
			default:
			}
		})
	}
}
