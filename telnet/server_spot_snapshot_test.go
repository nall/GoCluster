package telnet

import (
	"sync"
	"testing"

	"dxcluster/filter"
	"dxcluster/spot"
)

func TestBroadcastSpotSnapshotsPayload(t *testing.T) {
	srv := NewServer(ServerOptions{BroadcastQueue: 1}, nil)
	src := spot.NewSpot("K1ABC", "W1XYZ", 14074.0, "FT8")
	src.Comment = "first"
	src.DEMetadata.Grid = "FN31"

	srv.BroadcastSpot(src, true, true, true)

	select {
	case payload := <-srv.broadcast:
		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 128; i++ {
				_ = payload.spot.FormatDXCluster()
			}
		}()
		for i := 0; i < 128; i++ {
			src.Comment = "mutated"
			src.DEMetadata.Grid = "EM10"
			src.InvalidateMetadataCache()
			src.EnsureNormalized()
		}
		wg.Wait()

		if payload.spot == src {
			t.Fatalf("expected broadcast payload to own a snapshot, got original pointer")
		}
		if payload.spot.Comment != "first" {
			t.Fatalf("expected broadcast comment to remain stable, got %q", payload.spot.Comment)
		}
		if payload.spot.DEMetadata.Grid != "FN31" {
			t.Fatalf("expected broadcast grid to remain stable, got %q", payload.spot.DEMetadata.Grid)
		}
	default:
		t.Fatalf("expected broadcast payload")
	}
}

func TestDeliverSelfSpotSnapshotsPayload(t *testing.T) {
	srv := NewServer(ServerOptions{ClientBuffer: 1}, nil)
	f := filter.NewFilter()
	f.SetSelfEnabled(true)
	client := &Client{
		server:   srv,
		callsign: "K1ABC",
		filter:   f,
		spotChan: make(chan *spotEnvelope, 1),
		done:     make(chan struct{}),
	}
	srv.clients["K1ABC"] = client

	src := spot.NewSpot("K1ABC", "W1XYZ", 14074.0, "FT8")
	src.Comment = "first"
	src.DEMetadata.Grid = "FN31"

	srv.DeliverSelfSpot(src)

	select {
	case env := <-client.spotChan:
		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 128; i++ {
				_ = env.spot.FormatDXCluster()
			}
		}()
		for i := 0; i < 128; i++ {
			src.Comment = "mutated"
			src.DEMetadata.Grid = "EM10"
			src.InvalidateMetadataCache()
			src.EnsureNormalized()
		}
		wg.Wait()

		if env.spot == src {
			t.Fatalf("expected self-delivery payload to own a snapshot, got original pointer")
		}
		if env.spot.Comment != "first" {
			t.Fatalf("expected self-delivery comment to remain stable, got %q", env.spot.Comment)
		}
		if env.spot.DEMetadata.Grid != "FN31" {
			t.Fatalf("expected self-delivery grid to remain stable, got %q", env.spot.DEMetadata.Grid)
		}
	default:
		t.Fatalf("expected self-delivery payload")
	}
}
