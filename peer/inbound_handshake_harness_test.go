package peer

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"testing"
	"time"

	"dxcluster/config"
	"dxcluster/spot"
)

type handshakeStepKind int

const (
	handshakeExpectTx handshakeStepKind = iota
	handshakeSendRx
	handshakeExpectQuiet
	handshakeAwaitRegistered
	handshakeCloseRemote
	handshakeAwaitResult
)

type lineMatcher struct {
	name  string
	match func(string) bool
}

type handshakeStep struct {
	kind     handshakeStepKind
	matcher  lineMatcher
	line     string
	timeout  time.Duration
	errCheck func(error) string
}

type inboundScenario struct {
	name              string
	peers             []config.PeeringPeer
	globalAllowCalls  []string
	globalAllowIPs    []string
	loginTimeout      time.Duration
	initTimeout       time.Duration
	idleTimeout       time.Duration
	steps             []handshakeStep
	wantRegistered    bool
	wantRemoteCall    string
	wantPC9x          bool
	wantPC92Queued    int
	wantLegacyQueued  int
	wantIngestedSpots int
}

type transcriptEvent struct {
	dir  string
	line string
}

type inboundScenarioResult struct {
	transcript         []transcriptEvent
	runErr             error
	registeredObserved bool
	finalRemoteCall    string
	finalPC9x          bool
	pc92Queued         int
	legacyQueued       int
	ingestedSpots      int
}

func exactLine(line string) lineMatcher {
	return lineMatcher{
		name: fmt.Sprintf("exact %q", line),
		match: func(got string) bool {
			return got == line
		},
	}
}

func prefixLine(prefix string) lineMatcher {
	return lineMatcher{
		name: fmt.Sprintf("prefix %q", prefix),
		match: func(got string) bool {
			return strings.HasPrefix(got, prefix)
		},
	}
}

func pc92TypeLine(recordType string) lineMatcher {
	return lineMatcher{
		name: fmt.Sprintf("PC92 %s", recordType),
		match: func(got string) bool {
			frame, err := ParseFrame(got)
			if err != nil || frame.Type != "PC92" {
				return false
			}
			fields := frame.payloadFields()
			if len(fields) < 3 {
				return false
			}
			return strings.EqualFold(strings.TrimSpace(fields[2]), recordType)
		},
	}
}

func pc92CallTypeLine(call, recordType string) lineMatcher {
	return lineMatcher{
		name: fmt.Sprintf("PC92 %s from %s", recordType, call),
		match: func(got string) bool {
			frame, err := ParseFrame(got)
			if err != nil || frame.Type != "PC92" {
				return false
			}
			fields := frame.payloadFields()
			if len(fields) < 3 {
				return false
			}
			return strings.EqualFold(strings.TrimSpace(fields[0]), call) &&
				strings.EqualFold(strings.TrimSpace(fields[2]), recordType)
		},
	}
}

func errContains(sub string) func(error) string {
	return func(err error) string {
		if err == nil {
			return fmt.Sprintf("expected error containing %q, got nil", sub)
		}
		if !strings.Contains(err.Error(), sub) {
			return fmt.Sprintf("expected error containing %q, got %v", sub, err)
		}
		return ""
	}
}

func errIsTimeout(err error) string {
	if err == nil {
		return "expected timeout error, got nil"
	}
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return ""
	}
	if strings.Contains(strings.ToLower(err.Error()), "timeout") {
		return ""
	}
	return fmt.Sprintf("expected timeout error, got %v", err)
}

func errIsEOFOrClosedPipe(err error) string {
	if err == nil {
		return "expected EOF/closed pipe after remote close, got nil"
	}
	if errors.Is(err, io.EOF) {
		return ""
	}
	if strings.Contains(strings.ToLower(err.Error()), "closed pipe") {
		return ""
	}
	return fmt.Sprintf("expected EOF/closed pipe after remote close, got %v", err)
}

func newInboundHarnessManager(t *testing.T, scenario inboundScenario) (*Manager, chan *spot.Spot) {
	t.Helper()

	ingest := make(chan *spot.Spot, 8)
	cfg := config.PeeringConfig{
		NodeVersion:    "5457",
		LegacyVersion:  "1.57",
		PC92Bitmap:     5,
		HopCount:       99,
		WriteQueueSize: 8,
		MaxLineLength:  4096,
		PC92MaxBytes:   4096,
		Backoff: config.PeeringBackoff{
			BaseMS: 25,
			MaxMS:  100,
		},
		Timeouts: config.PeeringTimeouts{
			LoginSeconds: 1,
			InitSeconds:  1,
			IdleSeconds:  1,
		},
		ACL: config.PeeringACL{
			AllowCallsigns: append([]string(nil), scenario.globalAllowCalls...),
			AllowIPs:       append([]string(nil), scenario.globalAllowIPs...),
		},
		Peers: scenario.peers,
	}
	manager, err := NewManager(cfg, "N0CALL", ingest, 0, nil)
	if err != nil {
		t.Fatalf("%s: NewManager() error: %v", scenario.name, err)
	}
	manager.topology = &topologyStore{}
	manager.pc92Ch = make(chan pc92Work, 8)
	manager.legacyCh = make(chan legacyWork, 8)
	return manager, ingest
}

// runInboundScenario drives session.Run over net.Pipe so tests exercise the real
// inbound handshake and post-handshake registration path without OS listener
// noise. Each script step is deadline-bound to keep failure modes deterministic.
func runInboundScenario(t *testing.T, scenario inboundScenario) inboundScenarioResult {
	t.Helper()

	loginTimeout := scenario.loginTimeout
	if loginTimeout <= 0 {
		loginTimeout = 80 * time.Millisecond
	}
	initTimeout := scenario.initTimeout
	if initTimeout <= 0 {
		initTimeout = 120 * time.Millisecond
	}
	idleTimeout := scenario.idleTimeout
	if idleTimeout <= 0 {
		idleTimeout = 250 * time.Millisecond
	}

	server, client := net.Pipe()
	t.Cleanup(func() {
		_ = server.Close()
		_ = client.Close()
	})

	manager, ingest := newInboundHarnessManager(t, scenario)
	settings := sessionSettings{
		localCall:     "N0CALL",
		nodeVersion:   "5457",
		legacyVersion: "1.57",
		pc92Bitmap:    5,
		nodeCount:     1,
		userCount:     0,
		hopCount:      99,
		loginTimeout:  loginTimeout,
		initTimeout:   initTimeout,
		idleTimeout:   idleTimeout,
		writeQueue:    8,
		maxLine:       4096,
		pc92MaxBytes:  4096,
	}
	sess := newSession(server, dirInbound, manager, PeerEndpoint{host: "pipe"}, settings)
	sess.id = "inbound-harness"

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	runErrCh := make(chan error, 1)
	go func() {
		runErrCh <- sess.Run(ctx)
	}()

	reader := NewLineReader(client, 4096, 4096, nil)
	writer := bufio.NewWriter(client)

	var result inboundScenarioResult

	sendLine := func(line string) {
		t.Helper()
		if !strings.HasSuffix(line, "\n") {
			line += "\r\n"
		}
		if err := client.SetWriteDeadline(time.Now().UTC().Add(250 * time.Millisecond)); err != nil {
			t.Fatalf("%s: set write deadline: %v", scenario.name, err)
		}
		if _, err := writer.WriteString(line); err != nil {
			t.Fatalf("%s: write remote line: %v", scenario.name, err)
		}
		if err := writer.Flush(); err != nil {
			t.Fatalf("%s: flush remote line: %v", scenario.name, err)
		}
		result.transcript = append(result.transcript, transcriptEvent{dir: "rx", line: strings.TrimRight(line, "\r\n")})
	}

	readLine := func(timeout time.Duration) (string, error) {
		t.Helper()
		if timeout <= 0 {
			timeout = 250 * time.Millisecond
		}
		line, err := reader.ReadLine(time.Now().UTC().Add(timeout))
		if err == nil {
			result.transcript = append(result.transcript, transcriptEvent{dir: "tx", line: line})
		}
		return line, err
	}

	awaitRegistered := func(timeout time.Duration) {
		t.Helper()
		deadline := time.Now().UTC().Add(timeout)
		for time.Now().UTC().Before(deadline) {
			manager.mu.RLock()
			_, ok := manager.sessions[sess.id]
			manager.mu.RUnlock()
			if ok {
				result.registeredObserved = true
				return
			}
			time.Sleep(5 * time.Millisecond)
		}
		t.Fatalf("%s: timed out waiting for session registration", scenario.name)
	}

	awaitResult := func(timeout time.Duration, check func(error) string) {
		t.Helper()
		if timeout <= 0 {
			timeout = time.Second
		}
		select {
		case err := <-runErrCh:
			result.runErr = err
			if check != nil {
				if msg := check(err); msg != "" {
					t.Fatalf("%s: %s\ntranscript:\n%s", scenario.name, msg, formatTranscript(result.transcript))
				}
			}
		case <-time.After(timeout):
			t.Fatalf("%s: timed out waiting for session result\ntranscript:\n%s", scenario.name, formatTranscript(result.transcript))
		}
	}

	for _, step := range scenario.steps {
		switch step.kind {
		case handshakeExpectTx:
			line, err := readLine(step.timeout)
			if err != nil {
				t.Fatalf("%s: expected outbound line %q, got error %v\ntranscript:\n%s", scenario.name, step.matcher.name, err, formatTranscript(result.transcript))
			}
			if !step.matcher.match(line) {
				t.Fatalf("%s: expected outbound line %q, got %q\ntranscript:\n%s", scenario.name, step.matcher.name, line, formatTranscript(result.transcript))
			}
		case handshakeSendRx:
			sendLine(step.line)
		case handshakeExpectQuiet:
			line, err := readLine(step.timeout)
			if err == nil {
				t.Fatalf("%s: expected quiet window, got %q\ntranscript:\n%s", scenario.name, line, formatTranscript(result.transcript))
			}
			var netErr net.Error
			if !errors.As(err, &netErr) || !netErr.Timeout() {
				t.Fatalf("%s: expected timeout during quiet window, got %v\ntranscript:\n%s", scenario.name, err, formatTranscript(result.transcript))
			}
		case handshakeAwaitRegistered:
			awaitRegistered(step.timeout)
		case handshakeCloseRemote:
			if err := client.Close(); err != nil {
				t.Fatalf("%s: close remote: %v", scenario.name, err)
			}
		case handshakeAwaitResult:
			awaitResult(step.timeout, step.errCheck)
		default:
			t.Fatalf("%s: unknown step kind %d", scenario.name, step.kind)
		}
	}

	manager.mu.RLock()
	result.finalRemoteCall = sess.remoteCall
	result.finalPC9x = sess.pc9x
	manager.mu.RUnlock()
	result.pc92Queued = len(manager.pc92Ch)
	result.legacyQueued = len(manager.legacyCh)
	result.ingestedSpots = len(ingest)

	if result.registeredObserved != scenario.wantRegistered {
		t.Fatalf("%s: expected registered=%v, got %v\ntranscript:\n%s", scenario.name, scenario.wantRegistered, result.registeredObserved, formatTranscript(result.transcript))
	}
	if result.finalRemoteCall != scenario.wantRemoteCall {
		t.Fatalf("%s: expected remoteCall=%q, got %q\ntranscript:\n%s", scenario.name, scenario.wantRemoteCall, result.finalRemoteCall, formatTranscript(result.transcript))
	}
	if result.finalPC9x != scenario.wantPC9x {
		t.Fatalf("%s: expected pc9x=%v, got %v\ntranscript:\n%s", scenario.name, scenario.wantPC9x, result.finalPC9x, formatTranscript(result.transcript))
	}
	if result.pc92Queued != scenario.wantPC92Queued {
		t.Fatalf("%s: expected PC92 queue len=%d, got %d\ntranscript:\n%s", scenario.name, scenario.wantPC92Queued, result.pc92Queued, formatTranscript(result.transcript))
	}
	if result.legacyQueued != scenario.wantLegacyQueued {
		t.Fatalf("%s: expected legacy queue len=%d, got %d\ntranscript:\n%s", scenario.name, scenario.wantLegacyQueued, result.legacyQueued, formatTranscript(result.transcript))
	}
	if result.ingestedSpots != scenario.wantIngestedSpots {
		t.Fatalf("%s: expected ingested spots=%d, got %d\ntranscript:\n%s", scenario.name, scenario.wantIngestedSpots, result.ingestedSpots, formatTranscript(result.transcript))
	}

	return result
}

func formatTranscript(events []transcriptEvent) string {
	if len(events) == 0 {
		return "(empty)"
	}
	var b strings.Builder
	for _, event := range events {
		b.WriteString(event.dir)
		b.WriteString(" ")
		b.WriteString(event.line)
		b.WriteByte('\n')
	}
	return strings.TrimRight(b.String(), "\n")
}
