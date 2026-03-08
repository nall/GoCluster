package archive

import (
	"sync"
	"testing"

	"dxcluster/spot"
)

func TestEnqueueSnapshotsSpot(t *testing.T) {
	w := &Writer{
		queue: make(chan *spot.Spot, 1),
		stop:  make(chan struct{}),
	}
	src := spot.NewSpot("K1ABC", "W1XYZ", 14074.0, "FT8")
	src.Comment = "first"
	src.DEMetadata.Grid = "FN31"

	w.Enqueue(src)

	select {
	case queued := <-w.queue:
		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 128; i++ {
				queued.EnsureNormalized()
			}
		}()
		for i := 0; i < 128; i++ {
			src.Comment = "mutated"
			src.DEMetadata.Grid = "EM10"
			src.InvalidateMetadataCache()
			src.EnsureNormalized()
		}
		wg.Wait()

		if queued == src {
			t.Fatalf("expected archive queue to own a snapshot, got original pointer")
		}
		if queued.Comment != "first" {
			t.Fatalf("expected queued comment to remain stable, got %q", queued.Comment)
		}
		if queued.DEMetadata.Grid != "FN31" {
			t.Fatalf("expected queued grid to remain stable, got %q", queued.DEMetadata.Grid)
		}
		queued.EnsureNormalized()
		if queued.DEGridNorm != "FN31" {
			t.Fatalf("expected queued normalized grid to remain stable, got %q", queued.DEGridNorm)
		}
	default:
		t.Fatalf("expected queued spot")
	}
}
