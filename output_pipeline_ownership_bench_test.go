package main

import (
	"runtime"
	"testing"

	"dxcluster/spot"
)

func BenchmarkOutputPipelineEmitSpotOwnership(b *testing.B) {
	src := spot.NewSpot("K1ABC", "W1XYZ-1", 14074.0, "FT8")
	src.Comment = "CQ TEST"
	src.SourceType = spot.SourceManual
	src.DXMetadata.Grid = "EN61"
	src.DEMetadata.Grid = "FN31"
	src.DECallStripped = "W1XYZ"
	src.DECallNormStripped = "W1XYZ"
	src.EnsureNormalized()
	_ = src.FormatDXCluster()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ringSnapshot := src.Clone()
		ringSnapshot.ID = uint64(i + 1)
		archiveSnapshot := src.Clone()
		telnetSnapshot := src.Clone()
		peerSnapshot := cloneSpotForPeerPublish(src)
		runtime.KeepAlive(ringSnapshot)
		runtime.KeepAlive(archiveSnapshot)
		runtime.KeepAlive(telnetSnapshot)
		runtime.KeepAlive(peerSnapshot)
	}
}
