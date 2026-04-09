package stats

import "testing"

func TestTrackerObserveFloodDecision(t *testing.T) {
	tr := NewTracker()
	tr.ObserveFloodDecision("observe", "flood_decall")
	tr.ObserveFloodDecision("suppress", "flood_decall")
	tr.ObserveFloodDecision("drop", "flood_dxcall")

	if tr.FloodObserved() != 1 {
		t.Fatalf("expected observed=1, got %d", tr.FloodObserved())
	}
	if tr.FloodSuppressed() != 1 {
		t.Fatalf("expected suppressed=1, got %d", tr.FloodSuppressed())
	}
	if tr.FloodDropped() != 1 {
		t.Fatalf("expected dropped=1, got %d", tr.FloodDropped())
	}
	reasons := tr.FloodDecisionReasons()
	if reasons["observe|flood_decall"] != 1 {
		t.Fatalf("unexpected flood decision reasons: %v", reasons)
	}
	if reasons["suppress|flood_decall"] != 1 {
		t.Fatalf("unexpected flood decision reasons: %v", reasons)
	}
	if reasons["drop|flood_dxcall"] != 1 {
		t.Fatalf("unexpected flood decision reasons: %v", reasons)
	}
}

func TestTrackerObserveFloodOverflow(t *testing.T) {
	tr := NewTracker()
	tr.ObserveFloodOverflow("flood_decall")
	tr.ObserveFloodOverflow("flood_decall")

	if tr.FloodOverflow() != 2 {
		t.Fatalf("expected overflow=2, got %d", tr.FloodOverflow())
	}
	reasons := tr.FloodOverflowReasons()
	if reasons["flood_decall"] != 2 {
		t.Fatalf("unexpected flood overflow reasons: %v", reasons)
	}
}
