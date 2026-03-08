package buffer

import (
	"testing"

	"dxcluster/spot"
)

func TestAddStoresIndependentSnapshot(t *testing.T) {
	rb := NewRingBuffer(4)
	src := spot.NewSpot("K1ABC", "W1XYZ", 14074.0, "FT8")
	src.Comment = "first"
	src.DEMetadata.Grid = "FN31"
	src.EnsureNormalized()

	rb.Add(src)

	src.Comment = "mutated"
	src.DEMetadata.Grid = "EM10"
	src.InvalidateMetadataCache()
	src.EnsureNormalized()

	recent := rb.GetRecent(1)
	if len(recent) != 1 {
		t.Fatalf("expected one recent spot, got %d", len(recent))
	}
	got := recent[0]
	if got == src {
		t.Fatalf("expected ring buffer to store a snapshot, got original pointer")
	}
	if got.ID == 0 || got.ID != src.ID {
		t.Fatalf("expected preserved monotonic ID, got recent=%d source=%d", got.ID, src.ID)
	}
	if got.Comment != "first" {
		t.Fatalf("expected buffered comment to remain stable, got %q", got.Comment)
	}
	if got.DEMetadata.Grid != "FN31" {
		t.Fatalf("expected buffered grid to remain stable, got %q", got.DEMetadata.Grid)
	}
	if got.DEGridNorm != "FN31" {
		t.Fatalf("expected buffered normalized grid to remain stable, got %q", got.DEGridNorm)
	}
}
