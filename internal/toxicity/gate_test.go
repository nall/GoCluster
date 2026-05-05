package toxicity

import "testing"

func testGate(t *testing.T) *SafeGate {
	t.Helper()
	gate, err := NewSafeGate(safeGateConfig{
		MaxTokens:     8,
		SafeTokens:    []string{"CQ", "POTA", "SOTA", "TNX", "73", "TU", "LOTW", "TEST"},
		EventPrefixes: []string{"POTA", "SOTA"},
	})
	if err != nil {
		t.Fatal(err)
	}
	return gate
}

func TestSafeGateAllowsRoutineHamComments(t *testing.T) {
	gate := testGate(t)
	for _, comment := range []string{"CQ", "POTA 73", "POTA-1234 TU", "FN20 +10DB", "25WPM TEST"} {
		if !gate.IsSafe(comment) {
			t.Fatalf("expected routine comment %q to be safe", comment)
		}
	}
}

func TestSafeGateRejectsHamWordInjection(t *testing.T) {
	gate := testGate(t)
	for _, comment := range []string{"POTA idiot", "CQ go away", "SOTA insulto"} {
		if gate.IsSafe(comment) {
			t.Fatalf("expected mixed comment %q to route to AI", comment)
		}
	}
}

func TestNormalizeCommentPreservesWesternAccents(t *testing.T) {
	got := NormalizeComment("  grâce\tà vous\r\nPOTA  ")
	if got != "grâce à vous POTA" {
		t.Fatalf("NormalizeComment() = %q", got)
	}
}
