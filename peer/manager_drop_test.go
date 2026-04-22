package peer

import (
	"errors"
	"strings"
	"testing"
)

func TestFormatPC61DropLine(t *testing.T) {
	frame := &Frame{
		Type: "PC61",
		Fields: []string{
			"14025.0",
			"DX1AAA",
			"01-Jan-2025",
			"1200Z",
			"comment",
			"DE1BBB",
			"ORIGIN",
			"1.2.3.4",
		},
	}
	line := formatPC61DropLine(frame, nil, errors.New("pc61: invalid DX callsign"))
	wantParts := []string{
		"PC61 drop: reason=invalid_dx",
		"de=DE1BBB",
		"dx=DX1AAA",
		"band=20m",
		"freq=14025.0",
		"source=ORIGIN",
	}
	for _, part := range wantParts {
		if !strings.Contains(line, part) {
			t.Fatalf("expected %q in drop line, got %q", part, line)
		}
	}
}

func TestHandleFrameReportsBadCallParseDrop(t *testing.T) {
	m := &Manager{}
	var got struct {
		source string
		role   string
		reason string
		call   string
		de     string
		dx     string
		mode   string
		detail string
	}
	m.SetBadCallReporter(func(source, role, reason, call, deCall, dxCall, mode, detail string) {
		got.source = source
		got.role = role
		got.reason = reason
		got.call = call
		got.de = deCall
		got.dx = dxCall
		got.mode = mode
		got.detail = detail
	})
	frame, err := ParseFrame("PC61^14074.0^BAD!^23-Dec-2025^2001Z^FT8 CQ^W1XYZ^ORIGIN^203.0.113.7^H3^")
	if err != nil {
		t.Fatalf("ParseFrame() error: %v", err)
	}

	m.HandleFrame(frame, &session{remoteCall: "SRC"})

	if got.source != "peer:ORIGIN" || got.role != "DX" || got.reason != "invalid_callsign" || got.call != "BAD!" || got.de != "W1XYZ" || got.dx != "BAD!" || got.mode != "FT8" || got.detail != "peer_parse" {
		t.Fatalf("unexpected bad-call report: %+v", got)
	}
}
