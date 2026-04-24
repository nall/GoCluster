package cluster

import (
	"fmt"
	"log"
	"runtime/debug"

	"dxcluster/config"
)

const bytesPerMiB = 1024 * 1024

func applyGoRuntimeTuning(cfg config.GoRuntimeConfig) {
	if cfg.MemoryLimitMiB > 0 {
		debug.SetMemoryLimit(int64(cfg.MemoryLimitMiB) * bytesPerMiB)
	}
	if cfg.GCPercent > 0 {
		debug.SetGCPercent(cfg.GCPercent)
	}
}

func logGoRuntimeTuning(cfg config.GoRuntimeConfig) {
	memoryLimit := "unchanged"
	if cfg.MemoryLimitMiB > 0 {
		memoryLimit = formatGoRuntimeMiB(cfg.MemoryLimitMiB)
	}
	gcPercent := "unchanged"
	if cfg.GCPercent > 0 {
		gcPercent = formatGoRuntimePercent(cfg.GCPercent)
	}
	log.Printf("Go runtime tuning: memory_limit=%s gc_percent=%s", memoryLimit, gcPercent)
}

func formatGoRuntimeMiB(v int) string {
	return fmt.Sprintf("%dMiB", v)
}

func formatGoRuntimePercent(v int) string {
	return fmt.Sprintf("%d", v)
}
