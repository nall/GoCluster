package telnet

import (
	"testing"
	"time"
)

func TestNormalizeServerOptionsBroadcastBatchIntervalDefaultAndExplicitZero(t *testing.T) {
	defaulted := normalizeServerOptions(ServerOptions{})
	if defaulted.BroadcastBatchInterval != defaultBroadcastBatchInterval {
		t.Fatalf("default broadcast batch interval = %v, want %v", defaulted.BroadcastBatchInterval, defaultBroadcastBatchInterval)
	}

	disabled := normalizeServerOptions(ServerOptions{
		BroadcastBatchInterval:    0,
		BroadcastBatchIntervalSet: true,
	})
	if disabled.BroadcastBatchInterval != 0 {
		t.Fatalf("explicit zero broadcast batch interval = %v, want 0", disabled.BroadcastBatchInterval)
	}

	configured := normalizeServerOptions(ServerOptions{
		BroadcastBatchInterval:    time.Millisecond,
		BroadcastBatchIntervalSet: true,
	})
	if configured.BroadcastBatchInterval != time.Millisecond {
		t.Fatalf("configured broadcast batch interval = %v, want 1ms", configured.BroadcastBatchInterval)
	}
}
