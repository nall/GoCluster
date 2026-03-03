package peer

import (
	"reflect"
	"testing"
	"time"
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
