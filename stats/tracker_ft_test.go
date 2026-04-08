package stats

import (
	"testing"
	"time"
)

func TestTrackerFTBurstCounters(t *testing.T) {
	tr := NewTracker()
	tr.SetFTBurstActive(3)
	tr.IncrementFTBurstReleased()
	tr.IncrementFTBurstReleased()
	tr.IncrementFTBurstOverflowRelease()
	tr.ObserveFTBurstSpan("ft8", 1500*time.Millisecond)
	tr.ObserveFTBurstSpan("FT8", 500*time.Millisecond)
	tr.ObserveFTBurstSpan("FT4", 750*time.Millisecond)

	if tr.FTBurstActive() != 3 {
		t.Fatalf("expected active=3, got %d", tr.FTBurstActive())
	}
	if tr.FTBurstReleased() != 2 {
		t.Fatalf("expected released=2, got %d", tr.FTBurstReleased())
	}
	if tr.FTBurstOverflowRelease() != 1 {
		t.Fatalf("expected overflow=1, got %d", tr.FTBurstOverflowRelease())
	}

	spanStats := tr.FTBurstSpanStats()
	if got := spanStats["FT8"]; got.Samples != 2 || got.AverageSpan != time.Second {
		t.Fatalf("unexpected FT8 span stats: %+v", got)
	}
	if got := spanStats["FT4"]; got.Samples != 1 || got.AverageSpan != 750*time.Millisecond {
		t.Fatalf("unexpected FT4 span stats: %+v", got)
	}
}
