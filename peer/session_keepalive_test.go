package peer

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestKeepaliveLoopSendsPC51ForPC9x(t *testing.T) {
	s := &session{
		localCall:      "N0CALL",
		remoteCall:     "N0PEER",
		pc92Bitmap:     5,
		nodeVersion:    "5457",
		hopCount:       99,
		pc9x:           true,
		keepalive:      5 * time.Millisecond,
		writeCh:        make(chan string, 8),
		priorityLineCh: make(chan string, 8),
		tsGen:          &timestampGenerator{},
	}
	s.ctx, s.cancel = context.WithCancel(context.Background())
	defer s.cancel()

	go s.keepaliveLoop()

	waitForKeepalives(t, s.priorityLineCh, true)
}

func TestKeepaliveLoopSendsPC51ForLegacy(t *testing.T) {
	s := &session{
		localCall:      "N0CALL",
		remoteCall:     "N0PEER",
		pc9x:           false,
		keepalive:      5 * time.Millisecond,
		writeCh:        make(chan string, 4),
		priorityLineCh: make(chan string, 4),
		tsGen:          &timestampGenerator{},
	}
	s.ctx, s.cancel = context.WithCancel(context.Background())
	defer s.cancel()

	go s.keepaliveLoop()

	waitForKeepalives(t, s.priorityLineCh, false)
}

func TestKeepaliveBypassesNormalWriteBacklog(t *testing.T) {
	s := &session{
		localCall:      "N0CALL",
		remoteCall:     "N0PEER",
		pc92Bitmap:     5,
		nodeVersion:    "5457",
		hopCount:       99,
		pc9x:           true,
		keepalive:      5 * time.Millisecond,
		writeCh:        make(chan string, 1),
		priorityLineCh: make(chan string, 8),
		tsGen:          &timestampGenerator{},
	}
	s.writeCh <- "queued spot backlog"
	s.ctx, s.cancel = context.WithCancel(context.Background())
	defer s.cancel()

	go s.keepaliveLoop()

	waitForKeepalives(t, s.priorityLineCh, true)
}

func TestKeepaliveLoopSendsPC92ConfigOnPriorityQueue(t *testing.T) {
	s := &session{
		localCall:      "N0CALL",
		remoteCall:     "N0PEER",
		pc92Bitmap:     5,
		nodeVersion:    "5457",
		hopCount:       99,
		pc9x:           true,
		keepalive:      time.Hour,
		configEvery:    5 * time.Millisecond,
		writeCh:        make(chan string, 1),
		priorityLineCh: make(chan string, 8),
		tsGen:          &timestampGenerator{},
	}
	s.ctx, s.cancel = context.WithCancel(context.Background())
	defer s.cancel()

	go s.keepaliveLoop()

	timeout := time.NewTimer(500 * time.Millisecond)
	defer timeout.Stop()
	for {
		select {
		case line := <-s.priorityLineCh:
			if strings.HasPrefix(line, "PC92^") && strings.Contains(line, "^C^") {
				return
			}
		case <-timeout.C:
			t.Fatal("timeout waiting for PC92 config refresh on priority queue")
		}
	}
}

func TestPriorityLaneSaturationClosesSession(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	s := &session{
		localCall:      "N0CALL",
		remoteCall:     "N0PEER",
		pc92Bitmap:     5,
		nodeVersion:    "5457",
		hopCount:       99,
		pc9x:           true,
		keepalive:      5 * time.Millisecond,
		writeCh:        make(chan string, 1),
		priorityLineCh: make(chan string, 1),
		ctx:            ctx,
		cancel:         cancel,
		tsGen:          &timestampGenerator{},
	}
	s.priorityLineCh <- "occupied"

	done := make(chan struct{})
	go func() {
		s.keepaliveLoop()
		close(done)
	}()

	select {
	case <-ctx.Done():
	case <-time.After(500 * time.Millisecond):
		t.Fatal("expected keepalive loop to close the session when priority lane is full")
	}

	select {
	case <-done:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("expected keepalive loop to exit after closing the session")
	}
}

func waitForKeepalives(t *testing.T, ch <-chan string, wantPC92 bool) {
	t.Helper()
	timeout := time.NewTimer(500 * time.Millisecond)
	defer timeout.Stop()

	var gotPC51, gotPC92 bool
	for !gotPC51 || (wantPC92 && !gotPC92) {
		select {
		case line := <-ch:
			if strings.HasPrefix(line, "PC51^") {
				gotPC51 = true
			}
			if strings.HasPrefix(line, "PC92^") && strings.Contains(line, "^K^") {
				gotPC92 = true
			}
		case <-timeout.C:
			t.Fatalf("timeout waiting for keepalives (pc51=%v pc92=%v)", gotPC51, gotPC92)
		}
	}
}
