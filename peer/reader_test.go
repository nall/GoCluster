package peer

import (
	"errors"
	"net"
	"strings"
	"testing"
	"time"
)

func TestLineReaderDropsOversizePC92(t *testing.T) {
	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()

	maxLine := 1024
	pc92Max := 64
	reader := newLineReaderWithTransport(server, maxLine, pc92Max, server.Read, nil, nil)

	pc92Payload := strings.Repeat("A", pc92Max+10)
	pc92Line := "PC92^" + pc92Payload + "^H99^~"
	pc51Line := "PC51^N2WQ-77^N2WQ-73^0^~"

	go func() {
		_, _ = client.Write([]byte(pc92Line + pc51Line))
	}()

	deadline := time.Now().Add(2 * time.Second)
	_, err := reader.ReadLine(deadline)
	if err == nil {
		t.Fatal("expected oversize PC92 to return an error")
	}
	var tooLong ErrLineTooLong
	if !errors.As(err, &tooLong) {
		t.Fatalf("expected ErrLineTooLong, got %v", err)
	}
	if tooLong.Reason != overlongReasonPC92MaxBytes {
		t.Fatalf("expected reason=%q, got %q", overlongReasonPC92MaxBytes, tooLong.Reason)
	}
	if tooLong.Limit != pc92Max {
		t.Fatalf("expected limit=%d, got %d", pc92Max, tooLong.Limit)
	}

	line, err := reader.ReadLine(deadline)
	if err != nil {
		t.Fatalf("expected next line, got error: %v", err)
	}
	if line != "PC51^N2WQ-77^N2WQ-73^0^" {
		t.Fatalf("unexpected line after drop: %q", line)
	}
}

func TestLineReaderFlagsMaxLineReason(t *testing.T) {
	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()

	maxLine := 64
	reader := newLineReaderWithTransport(server, maxLine, 0, server.Read, nil, nil)

	go func() {
		_, _ = client.Write([]byte(strings.Repeat("X", maxLine+16)))
		time.Sleep(200 * time.Millisecond)
	}()

	deadline := time.Now().Add(2 * time.Second)
	_, err := reader.ReadLine(deadline)
	if err == nil {
		t.Fatal("expected max-line overflow to return an error")
	}
	var tooLong ErrLineTooLong
	if !errors.As(err, &tooLong) {
		t.Fatalf("expected ErrLineTooLong, got %v", err)
	}
	if tooLong.Reason != overlongReasonMaxLine {
		t.Fatalf("expected reason=%q, got %q", overlongReasonMaxLine, tooLong.Reason)
	}
	if tooLong.Limit != maxLine {
		t.Fatalf("expected limit=%d, got %d", maxLine, tooLong.Limit)
	}
}
