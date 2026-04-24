package cluster

import (
	"runtime/debug"
	"testing"

	"dxcluster/config"
)

func TestApplyGoRuntimeTuningAppliesConfiguredValues(t *testing.T) {
	oldMem := debug.SetMemoryLimit(1 << 62)
	oldGC := debug.SetGCPercent(100)
	defer debug.SetMemoryLimit(oldMem)
	defer debug.SetGCPercent(oldGC)

	applyGoRuntimeTuning(config.GoRuntimeConfig{
		MemoryLimitMiB: 750,
		GCPercent:      50,
	})

	if got := debug.SetMemoryLimit(-1); got != 750*bytesPerMiB {
		t.Fatalf("memory limit = %d, want %d", got, 750*bytesPerMiB)
	}
	if got := debug.SetGCPercent(100); got != 50 {
		t.Fatalf("GC percent = %d, want 50", got)
	}
}

func TestApplyGoRuntimeTuningZeroLeavesExistingValues(t *testing.T) {
	oldMem := debug.SetMemoryLimit(1 << 62)
	oldGC := debug.SetGCPercent(100)
	defer debug.SetMemoryLimit(oldMem)
	defer debug.SetGCPercent(oldGC)

	debug.SetMemoryLimit(900 * bytesPerMiB)
	debug.SetGCPercent(123)

	applyGoRuntimeTuning(config.GoRuntimeConfig{})

	if got := debug.SetMemoryLimit(-1); got != 900*bytesPerMiB {
		t.Fatalf("memory limit = %d, want %d", got, 900*bytesPerMiB)
	}
	if got := debug.SetGCPercent(100); got != 123 {
		t.Fatalf("GC percent = %d, want 123", got)
	}
}
