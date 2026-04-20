package telnet

import (
	"errors"
	"io"
	"net"
	"strings"
	"testing"
	"time"

	"dxcluster/filter"
)

const hybridLoginGreeting = "" +
	"Hello <CALL>, this is <CLUSTER>\n" +
	"Cluster: Users <USER_COUNT>  Uptime <UPTIME>\n" +
	"Last login: <LAST_LOGIN> from <LAST_IP>\n" +
	"Current commands: <DIALECT> (<DIALECT_SOURCE>).\n" +
	"Use DIALECT LIST or DIALECT <DIALECT_DEFAULT> to change.\n" +
	"Grid: <GRID> (SET GRID <grid>)\n" +
	"Noise: <NOISE> (SET NOISE QUIET|RURAL|SUBURBAN|URBAN|INDUSTRIAL)\n" +
	"Dedupe policy: <DEDUPE> (SET DEDUPE FAST|MED|SLOW)\n" +
	"Type HELP for all commands.\n\n" +
	"<CALL> de <CLUSTER> <DATETIME>>"

func TestHandleClientPromptOnlyPreloginTranscript(t *testing.T) {
	s := newHandshakeTranscriptServer(t)
	serverConn, clientConn, done := startHandshakeTranscriptSession(t, s)
	defer closeHandshakeTranscriptSession(t, clientConn, done)
	defer serverConn.Close()

	got := readUntilContains(t, clientConn, "login: ", 2*time.Second)
	if got != "login: " {
		t.Fatalf("expected only login prompt, got %q", got)
	}
}

func TestHandleClientSuccessfulLoginTranscript(t *testing.T) {
	s := newHandshakeTranscriptServer(t)
	serverConn, clientConn, done := startHandshakeTranscriptSession(t, s)
	defer closeHandshakeTranscriptSession(t, clientConn, done)
	defer serverConn.Close()

	initial := readUntilContains(t, clientConn, "login: ", 2*time.Second)
	if initial != "login: " {
		t.Fatalf("expected only login prompt, got %q", initial)
	}
	if _, err := io.WriteString(clientConn, "n0call\r\n"); err != nil {
		t.Fatalf("write callsign: %v", err)
	}

	got := readUntilContains(t, clientConn, "UTC>", 2*time.Second)
	for _, want := range []string{
		"Hello N0CALL, this is N2WQ-2",
		"Cluster: Users 1  Uptime ",
		"Last login: (first login) from (unknown)",
		"Current commands: GO (default).",
		"Grid: unset (SET GRID <grid>)",
		"Noise: QUIET (SET NOISE QUIET|RURAL|SUBURBAN|URBAN|INDUSTRIAL)",
		"Dedupe policy: FAST (SET DEDUPE FAST|MED|SLOW)",
		"Type HELP for all commands.",
		"N0CALL de N2WQ-2 ",
		"UTC>",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("login transcript missing %q:\n%s", want, got)
		}
	}
}

func TestHandleClientRePromptsAfterEmptyAndInvalidLogin(t *testing.T) {
	s := newHandshakeTranscriptServer(t)
	serverConn, clientConn, done := startHandshakeTranscriptSession(t, s)
	defer closeHandshakeTranscriptSession(t, clientConn, done)
	defer serverConn.Close()

	initial := readUntilContains(t, clientConn, "login: ", 2*time.Second)
	if initial != "login: " {
		t.Fatalf("expected only login prompt, got %q", initial)
	}
	if _, err := io.WriteString(clientConn, "\r\n"); err != nil {
		t.Fatalf("write blank login: %v", err)
	}
	emptyResp := readUntilContains(t, clientConn, "login: ", 2*time.Second)
	if !strings.Contains(emptyResp, normalizeOutboundLine("Callsign cannot be empty. Please try again.\n")) || !strings.HasSuffix(emptyResp, "login: ") {
		t.Fatalf("unexpected empty-login response: %q", emptyResp)
	}

	if _, err := io.WriteString(clientConn, "AA\r\n"); err != nil {
		t.Fatalf("write invalid login: %v", err)
	}
	invalidResp := readUntilContains(t, clientConn, "login: ", 2*time.Second)
	if !strings.Contains(invalidResp, normalizeOutboundLine("Invalid call. Please try again.\n")) || !strings.HasSuffix(invalidResp, "login: ") {
		t.Fatalf("unexpected invalid-login response: %q", invalidResp)
	}
}

func newHandshakeTranscriptServer(t *testing.T) *Server {
	t.Helper()
	tmp := t.TempDir()
	orig := filter.UserDataDir
	filter.UserDataDir = tmp
	t.Cleanup(func() { filter.UserDataDir = orig })

	return NewServer(ServerOptions{
		HandshakeMode:          telnetHandshakeNone,
		WelcomeMessage:         "",
		LoginPrompt:            "login: ",
		LoginGreeting:          hybridLoginGreeting,
		LoginEmptyMessage:      "Callsign cannot be empty. Please try again.\n",
		LoginInvalidMessage:    "Invalid call. Please try again.\n",
		DialectSourceDefault:   "default",
		DialectSourcePersisted: "persisted",
		ClusterCall:            "N2WQ-2",
		LoginLineLimit:         32,
		CommandLineLimit:       128,
	}, nil)
}

func startHandshakeTranscriptSession(t *testing.T, s *Server) (net.Conn, net.Conn, <-chan struct{}) {
	t.Helper()
	serverConn, clientConn := net.Pipe()
	done := make(chan struct{})
	go func() {
		s.handleClient(serverConn, nil)
		close(done)
	}()
	return serverConn, clientConn, done
}

func closeHandshakeTranscriptSession(t *testing.T, clientConn net.Conn, done <-chan struct{}) {
	t.Helper()
	if clientConn != nil {
		_ = clientConn.Close()
	}
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("handleClient did not exit after test connection close")
	}
}

//nolint:unparam // Timeout stays explicit because transcript tests are deadline-sensitive.
func readUntilContains(t *testing.T, conn net.Conn, want string, timeout time.Duration) string {
	t.Helper()
	if timeout <= 0 {
		timeout = 2 * time.Second
	}
	var out strings.Builder
	buf := make([]byte, 256)
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if err := conn.SetReadDeadline(time.Now().Add(100 * time.Millisecond)); err != nil {
			t.Fatalf("set read deadline: %v", err)
		}
		n, err := conn.Read(buf)
		if n > 0 {
			out.Write(buf[:n])
			if strings.Contains(out.String(), want) {
				return out.String()
			}
		}
		if err == nil {
			continue
		}
		var ne net.Error
		if errors.As(err, &ne) && ne.Timeout() {
			continue
		}
		if errors.Is(err, io.EOF) {
			break
		}
		t.Fatalf("read transcript: %v", err)
	}
	t.Fatalf("timed out waiting for %q in transcript %q", want, out.String())
	return ""
}
