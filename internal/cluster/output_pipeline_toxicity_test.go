package cluster

import (
	"context"
	"strings"
	"testing"
	"time"

	"dxcluster/buffer"
	"dxcluster/cty"
	"dxcluster/internal/toxicity"
	"dxcluster/spot"
)

type clusterToxicityClient struct{}

func (clusterToxicityClient) Classify(context.Context, string) (toxicity.Decision, error) {
	return toxicity.Decision{Status: spot.ToxicityToxic, Categories: []string{"S10"}, Model: "mock"}, nil
}

type blockingClusterToxicityClient struct {
	release chan struct{}
}

func (b blockingClusterToxicityClient) Classify(ctx context.Context, _ string) (toxicity.Decision, error) {
	select {
	case <-b.release:
		return toxicity.Decision{Status: spot.ToxicityToxic, Categories: []string{"S10"}, Model: "mock"}, nil
	case <-ctx.Done():
		return toxicity.Decision{}, ctx.Err()
	}
}

func TestOutputPipelineToxicityStageQueuesHumanOnly(t *testing.T) {
	gate, err := toxicity.NewSafeGateFromLists([]string{"CQ", "73"}, []string{"POTA"}, 8)
	if err != nil {
		t.Fatal(err)
	}
	classifier := toxicity.NewClassifier(toxicity.Config{
		Enabled:         true,
		Workers:         1,
		QueueSize:       2,
		CacheMaxEntries: 8,
		CacheTTLSeconds: 60,
		MaxCommentBytes: 512,
		TimeoutMS:       100,
	}, gate, clusterToxicityClient{})
	classifier.Start()
	defer classifier.Stop()
	pipeline := &outputPipeline{toxicityClassifier: classifier}

	skimmer := spot.NewSpot("K1ABC", "W1XYZ", 14074, "FT8")
	skimmer.SourceType = spot.SourceRBN
	skimmer.Comment = "non routine"
	if !pipeline.applyToxicityStage(skimmer) {
		t.Fatalf("skimmer spot should bypass toxicity queue")
	}

	human := spot.NewSpot("K1ABC", "W1XYZ", 14074, "CW")
	human.Comment = "non routine"
	if pipeline.applyToxicityStage(human) {
		t.Fatalf("non-routine human comment should queue for AI")
	}
	select {
	case result := <-classifier.Results():
		toxicity.ApplyResult(result)
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for toxicity result")
	}
	if human.ToxicityStatus != spot.ToxicityToxic {
		t.Fatalf("expected toxic status, got %q", human.ToxicityStatus)
	}
	if !pipeline.applyToxicityStage(human) {
		t.Fatalf("classified spot should not requeue")
	}
}

func TestOutputPipelineDrainsToxicityOnInputClose(t *testing.T) {
	gate, err := toxicity.NewSafeGateFromLists([]string{"CQ", "73"}, []string{"POTA"}, 8)
	if err != nil {
		t.Fatal(err)
	}
	release := make(chan struct{})
	classifier := toxicity.NewClassifier(toxicity.Config{
		Enabled:         true,
		Workers:         1,
		QueueSize:       2,
		CacheMaxEntries: 8,
		CacheTTLSeconds: 60,
		MaxCommentBytes: 512,
		TimeoutMS:       500,
	}, gate, blockingClusterToxicityClient{release: release})
	classifier.Start()

	output := make(chan *spot.Spot, 1)
	recent := buffer.NewRingBuffer(4)
	pipeline := &outputPipeline{
		outputChan:         output,
		toxicityClassifier: classifier,
		buf:                recent,
		ctyLookup:          func() *cty.CTYDatabase { return nil },
	}

	done := make(chan struct{})
	go func() {
		pipeline.run()
		close(done)
	}()

	human := spot.NewSpot("K1ABC", "W1XYZ", 14074, "CW")
	human.Comment = "non routine"
	output <- human

	deadline := time.After(time.Second)
	for classifier.Pending() == 0 {
		select {
		case <-deadline:
			t.Fatal("timed out waiting for pending classifier job")
		default:
			time.Sleep(time.Millisecond)
		}
	}
	close(output)
	close(release)

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("pipeline did not stop")
	}

	recentSpots := recent.GetRecent(1)
	if len(recentSpots) != 1 {
		t.Fatalf("expected drained spot in recent buffer, got %d", len(recentSpots))
	}
	if recentSpots[0].ToxicityStatus != spot.ToxicityToxic {
		t.Fatalf("expected drained toxicity status TOXIC, got %q", recentSpots[0].ToxicityStatus)
	}
}

func TestFormatToxicitySummaryExposesBoundedCounters(t *testing.T) {
	gate, err := toxicity.NewSafeGateFromLists([]string{"CQ", "73"}, []string{"POTA"}, 8)
	if err != nil {
		t.Fatal(err)
	}
	classifier := toxicity.NewClassifier(toxicity.Config{
		Enabled:         true,
		Workers:         1,
		QueueSize:       1,
		CacheMaxEntries: 8,
		CacheTTLSeconds: 60,
		MaxCommentBytes: 512,
		TimeoutMS:       100,
	}, gate, clusterToxicityClient{})
	classifier.Start()
	defer classifier.Stop()

	routine := spot.NewSpot("K1ABC", "W1XYZ", 14074, "CW")
	routine.IsHuman = true
	routine.Comment = "CQ 73"
	if !classifier.ClassifyOrEnqueue(routine, time.Now().UTC()) {
		t.Fatal("routine comment should not enqueue")
	}

	human := spot.NewSpot("K1ABC", "W1XYZ", 14074, "CW")
	human.IsHuman = true
	human.Comment = "non routine"
	if classifier.ClassifyOrEnqueue(human, time.Now().UTC()) {
		t.Fatal("non-routine comment should enqueue")
	}
	select {
	case result := <-classifier.Results():
		toxicity.ApplyResult(result)
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for classifier result")
	}

	line := formatToxicitySummary(classifier)
	for _, want := range []string{"Toxicity:", "(AI)", "(L)", "(H)", "(T)", "(U)", "q=0", "cache=", "err"} {
		if !strings.Contains(line, want) {
			t.Fatalf("expected toxicity summary to contain %q, got %q", want, line)
		}
	}
	if strings.Contains(line, human.Comment) || strings.Contains(line, "mock") || strings.Contains(line, "S10") {
		t.Fatalf("toxicity summary leaked classifier payload detail: %q", line)
	}
}
