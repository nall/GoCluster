package peer

import "testing"

func TestParseFrameStripsSingleHopSuffix(t *testing.T) {
	frame, err := ParseFrame("PC92^OH2J^76586.01^A^^1RV1CC:178.70.200.33^H95^")
	if err != nil {
		t.Fatalf("ParseFrame: %v", err)
	}
	if frame.Hop != 95 {
		t.Fatalf("expected hop=95, got %d", frame.Hop)
	}
	if got, want := len(frame.Fields), 5; got != want {
		t.Fatalf("expected %d payload fields, got %d (%v)", want, got, frame.Fields)
	}
	last := frame.Fields[len(frame.Fields)-1]
	if last != "1RV1CC:178.70.200.33" {
		t.Fatalf("expected last payload field to be entry, got %q", last)
	}
}

func TestParseFrameStripsStackedHopSuffixAndUsesRightmostHop(t *testing.T) {
	frame, err := ParseFrame("PC92^OH2J^76586.01^A^^1RV1CC:178.70.200.33^H95^H94^H93^")
	if err != nil {
		t.Fatalf("ParseFrame: %v", err)
	}
	if frame.Hop != 93 {
		t.Fatalf("expected hop=93 from rightmost token, got %d", frame.Hop)
	}
	if got, want := len(frame.Fields), 5; got != want {
		t.Fatalf("expected %d payload fields, got %d (%v)", want, got, frame.Fields)
	}
}

func TestEncodeCanonicalizesTrailingHopSuffix(t *testing.T) {
	frame, err := ParseFrame("PC92^OH2J^76586.01^A^^1RV1CC:178.70.200.33^H95^")
	if err != nil {
		t.Fatalf("ParseFrame: %v", err)
	}
	got := frame.Encode(frame.Hop - 1)
	want := "PC92^OH2J^76586.01^A^^1RV1CC:178.70.200.33^H94^"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestPayloadFieldsStripsTrailingHopSuffixRun(t *testing.T) {
	in := []string{"NODE", "123", "A", "", "9CALL:ver", "H95", "H94", "H93", ""}
	out := PayloadFields(in)
	if got, want := len(out), 5; got != want {
		t.Fatalf("expected %d payload fields, got %d (%v)", want, got, out)
	}
	if out[len(out)-1] != "9CALL:ver" {
		t.Fatalf("expected final payload entry, got %q", out[len(out)-1])
	}
}

func TestParseFrameMalformedTrailingHopLikeTokensAreStripped(t *testing.T) {
	frame, err := ParseFrame("PC92^NODE^123^A^^9CALL:ver^H99^H9x^")
	if err != nil {
		t.Fatalf("ParseFrame: %v", err)
	}
	if frame.Hop != 99 {
		t.Fatalf("expected hop from rightmost numeric token, got %d", frame.Hop)
	}
	if got, want := len(frame.Fields), 5; got != want {
		t.Fatalf("expected %d payload fields, got %d (%v)", want, got, frame.Fields)
	}
}
