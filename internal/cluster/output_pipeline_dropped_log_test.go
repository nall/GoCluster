package cluster

import (
	"os"
	"strings"
	"testing"
	"time"

	"dxcluster/config"
	"dxcluster/spot"
)

func TestOutputPipelineLogsHarmonicDrop(t *testing.T) {
	logger, err := newDroppedCallLogger(config.DroppedCallLoggingConfig{
		Enabled:             true,
		Dir:                 t.TempDir(),
		RetentionDays:       1,
		DedupeWindowSeconds: 0,
		Harmonics:           true,
	})
	if err != nil {
		t.Fatalf("newDroppedCallLogger() error: %v", err)
	}

	detector := spot.NewHarmonicDetector(spot.HarmonicSettings{
		Enabled:              true,
		RecencyWindow:        time.Hour,
		MaxHarmonicMultiple:  4,
		FrequencyToleranceHz: 100,
		MinReportDelta:       10,
	})
	now := time.Now().UTC()
	fundamental := spot.NewSpotNormalized("K1ABC", "W2BBB", 7011.0, "CW")
	fundamental.Report = 30
	fundamental.HasReport = true
	fundamental.Time = now
	if drop, _, _, _ := detector.ShouldDrop(fundamental, now); drop {
		t.Fatal("fundamental should seed detector, not drop")
	}

	pipeline := &outputPipeline{
		harmonicDetector:  detector,
		harmonicCfg:       config.HarmonicConfig{Enabled: true},
		droppedCallLogger: logger,
	}
	harmonic := spot.NewSpotNormalized("K1ABC", "W1XYZ", 14022.0, "CW")
	harmonic.SourceNode = "RBN"
	harmonic.Report = 10
	harmonic.HasReport = true
	harmonic.Time = now.Add(time.Second)

	if pipeline.applyPostResolverAdjustments(&outputSpotContext{spot: harmonic}) {
		t.Fatal("expected harmonic spot to be suppressed")
	}
	if err := logger.Close(); err != nil {
		t.Fatalf("Close() error: %v", err)
	}

	path := droppedLogPath(logger.harmonics.(*dailyFileSink).dir, "")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	line := strings.TrimSpace(string(data))
	want := "source=RBN role=DX reason=harmonic call=K1ABC de=W1XYZ dx=K1ABC mode=CW detail=corroborators=1_delta_db=20"
	if !strings.Contains(line, want) {
		t.Fatalf("unexpected harmonic log line:\ngot  %q\nwant %q", line, want)
	}
}
