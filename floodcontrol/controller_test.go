package floodcontrol

import (
	"testing"
	"time"

	"dxcluster/config"
	"dxcluster/dedup"
	"dxcluster/spot"
)

func TestControllerObserveStillForwards(t *testing.T) {
	downstream := make(chan *spot.Spot, 8)
	controller := New(testFloodConfig(config.FloodActionObserve, 4), downstream, nil, nil)
	controller.Start()
	defer controller.Stop()

	controller.Input() <- testSpot("W1AAA", spot.SourceManual)
	controller.Input() <- testSpot("W1AAA", spot.SourceManual)

	mustReceiveSpot(t, downstream)
	mustReceiveSpot(t, downstream)

	_, observed, suppressed, dropped, overflow, _ := controller.GetStats()
	if observed != 1 {
		t.Fatalf("expected observed=1, got %d", observed)
	}
	if suppressed != 0 || dropped != 0 || overflow != 0 {
		t.Fatalf("unexpected counters suppress=%d drop=%d overflow=%d", suppressed, dropped, overflow)
	}
}

func TestControllerPartitionsBySourceType(t *testing.T) {
	downstream := make(chan *spot.Spot, 8)
	controller := New(testFloodConfig(config.FloodActionSuppress, 4), downstream, nil, nil)
	controller.Start()
	defer controller.Stop()

	controller.Input() <- testSpot("W1AAA", spot.SourceManual)
	controller.Input() <- testSpot("W1AAA", spot.SourceRBN)
	controller.Input() <- testSpot("W1AAA", spot.SourceManual)

	mustReceiveSpot(t, downstream)
	mustReceiveSpot(t, downstream)
	mustNotReceiveSpot(t, downstream)

	_, _, suppressed, _, _, _ := controller.GetStats()
	if suppressed != 1 {
		t.Fatalf("expected suppressed=1, got %d", suppressed)
	}
}

func TestControllerOverflowFailsOpenForUnseenActors(t *testing.T) {
	downstream := make(chan *spot.Spot, 8)
	controller := New(testFloodConfig(config.FloodActionSuppress, 1), downstream, nil, nil)
	controller.Start()
	defer controller.Stop()

	controller.Input() <- testSpot("W1AAA", spot.SourceManual)
	controller.Input() <- testSpot("W1BBB", spot.SourceManual)
	controller.Input() <- testSpot("W1BBB", spot.SourceManual)

	mustReceiveSpot(t, downstream)
	mustReceiveSpot(t, downstream)
	mustReceiveSpot(t, downstream)

	_, _, suppressed, _, overflow, _ := controller.GetStats()
	if suppressed != 0 {
		t.Fatalf("expected suppressed=0 with fail-open overflow, got %d", suppressed)
	}
	if overflow == 0 {
		t.Fatalf("expected overflow > 0")
	}
}

func TestControllerSuppressesBeforePrimaryDedupe(t *testing.T) {
	d := dedup.NewDeduplicator(time.Minute, false, 8)
	d.Start()
	defer d.Stop()

	controller := New(testFloodConfig(config.FloodActionSuppress, 4), d.GetInputChannel(), nil, nil)
	controller.Start()
	defer controller.Stop()

	controller.Input() <- testSpot("W1AAA", spot.SourceManual)
	mustReceiveSpot(t, d.GetOutputChannel())

	controller.Input() <- testSpot("W1AAA", spot.SourceManual)
	mustNotReceiveSpot(t, d.GetOutputChannel())

	processed, duplicates, _ := d.GetStats()
	if processed != 1 {
		t.Fatalf("expected primary dedupe to process only the forwarded spot, got %d", processed)
	}
	if duplicates != 0 {
		t.Fatalf("expected primary dedupe duplicates=0, got %d", duplicates)
	}
}

func testFloodConfig(action string, maxEntries int) config.FloodControlConfig {
	return config.FloodControlConfig{
		Enabled:            true,
		LogIntervalSeconds: 30,
		PartitionMode:      config.FloodControlPartitionExactSourceType,
		Rails: config.FloodControlRailsConfig{
			DECall: config.FloodRailConfig{
				Enabled:                true,
				Action:                 action,
				WindowSeconds:          60,
				MaxEntriesPerPartition: maxEntries,
				ThresholdsPerSourceType: map[string]int{
					"MANUAL":      1,
					"UPSTREAM":    0,
					"PEER":        0,
					"RBN":         1,
					"FT8":         0,
					"FT4":         0,
					"PSKREPORTER": 0,
				},
			},
			SourceNode: disabledRail(),
			SpotterIP:  disabledRail(),
			DXCall: config.FloodRailConfig{
				Enabled:                false,
				Action:                 action,
				WindowSeconds:          60,
				MaxEntriesPerPartition: maxEntries,
				ActiveMode:             "conservative",
				ThresholdsByMode: config.FloodRailThresholdsByMode{
					Conservative: disabledThresholds(),
					Moderate:     disabledThresholds(),
					Aggressive:   disabledThresholds(),
				},
			},
		},
	}
}

func disabledRail() config.FloodRailConfig {
	return config.FloodRailConfig{
		Enabled:                 false,
		Action:                  config.FloodActionObserve,
		WindowSeconds:           60,
		MaxEntriesPerPartition:  4,
		ThresholdsPerSourceType: disabledThresholds(),
	}
}

func disabledThresholds() map[string]int {
	return map[string]int{
		"MANUAL":      0,
		"UPSTREAM":    0,
		"PEER":        0,
		"RBN":         0,
		"FT8":         0,
		"FT4":         0,
		"PSKREPORTER": 0,
	}
}

func testSpot(deCall string, sourceType spot.SourceType) *spot.Spot {
	now := time.Now().UTC()
	return &spot.Spot{
		DXCall:     "K1ABC",
		DXCallNorm: "K1ABC",
		DECall:     deCall,
		DECallNorm: deCall,
		SourceType: sourceType,
		Time:       now,
		Frequency:  7010.0,
	}
}

func mustReceiveSpot(t *testing.T, ch <-chan *spot.Spot) *spot.Spot {
	t.Helper()
	select {
	case s := <-ch:
		if s == nil {
			t.Fatalf("received nil spot")
		}
		return s
	case <-time.After(2 * time.Second):
		t.Fatalf("timed out waiting for spot")
		return nil
	}
}

func mustNotReceiveSpot(t *testing.T, ch <-chan *spot.Spot) {
	t.Helper()
	select {
	case s := <-ch:
		t.Fatalf("unexpected spot received: %+v", s)
	case <-time.After(150 * time.Millisecond):
	}
}
