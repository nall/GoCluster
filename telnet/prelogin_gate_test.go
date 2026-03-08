package telnet

import (
	"errors"
	"net"
	"testing"
	"time"
)

func newPreloginTestServer(now *time.Time) *Server {
	s := &Server{
		maxPreloginSessions:  8,
		acceptRatePerIP:      10,
		acceptBurstPerIP:     10,
		preloginConcPerIP:    4,
		preloginByIP:         make(map[string]preloginIPState),
		preloginTrackedMax:   64,
		preloginStateIdleTTL: time.Minute,
	}
	if now != nil {
		s.nowFn = func() time.Time { return *now }
	}
	return s
}

func tcpAddr(ip string, port int) net.Addr {
	return &net.TCPAddr{IP: net.ParseIP(ip), Port: port}
}

func TestTryAcquirePreloginHonorsGlobalCap(t *testing.T) {
	now := time.Unix(100, 0).UTC()
	s := newPreloginTestServer(&now)
	s.maxPreloginSessions = 1

	t1, reason := s.tryAcquirePrelogin(tcpAddr("203.0.113.10", 1000))
	if t1 == nil || reason != "" {
		t.Fatalf("expected first admission success, got ticket=%v reason=%q", t1, reason)
	}
	t2, reason := s.tryAcquirePrelogin(tcpAddr("203.0.113.11", 1001))
	if t2 != nil || reason != preloginRejectGlobalCap {
		t.Fatalf("expected global cap reject, got ticket=%v reason=%q", t2, reason)
	}
	t1.Release()
	t3, reason := s.tryAcquirePrelogin(tcpAddr("203.0.113.11", 1002))
	if t3 == nil || reason != "" {
		t.Fatalf("expected admission after release, got ticket=%v reason=%q", t3, reason)
	}
	t3.Release()
}

func TestTryAcquirePreloginHonorsPerIPConcurrency(t *testing.T) {
	now := time.Unix(100, 0).UTC()
	s := newPreloginTestServer(&now)
	s.preloginConcPerIP = 2

	t1, reason := s.tryAcquirePrelogin(tcpAddr("203.0.113.20", 2000))
	if t1 == nil || reason != "" {
		t.Fatalf("expected first admission success, got ticket=%v reason=%q", t1, reason)
	}
	t2, reason := s.tryAcquirePrelogin(tcpAddr("203.0.113.20", 2001))
	if t2 == nil || reason != "" {
		t.Fatalf("expected second admission success, got ticket=%v reason=%q", t2, reason)
	}
	t3, reason := s.tryAcquirePrelogin(tcpAddr("203.0.113.20", 2002))
	if t3 != nil || reason != preloginRejectIPConcurrency {
		t.Fatalf("expected per-ip concurrency reject, got ticket=%v reason=%q", t3, reason)
	}
	t1.Release()
	t4, reason := s.tryAcquirePrelogin(tcpAddr("203.0.113.20", 2003))
	if t4 == nil || reason != "" {
		t.Fatalf("expected admission after concurrency release, got ticket=%v reason=%q", t4, reason)
	}
	t2.Release()
	t4.Release()
}

func TestTryAcquirePreloginHonorsPerIPRate(t *testing.T) {
	now := time.Unix(100, 0).UTC()
	s := newPreloginTestServer(&now)
	s.acceptRatePerIP = 1
	s.acceptBurstPerIP = 1
	s.preloginConcPerIP = 8

	t1, reason := s.tryAcquirePrelogin(tcpAddr("203.0.113.30", 3000))
	if t1 == nil || reason != "" {
		t.Fatalf("expected first admission success, got ticket=%v reason=%q", t1, reason)
	}
	t1.Release()

	t2, reason := s.tryAcquirePrelogin(tcpAddr("203.0.113.30", 3001))
	if t2 != nil || reason != preloginRejectIPRate {
		t.Fatalf("expected rate reject, got ticket=%v reason=%q", t2, reason)
	}

	now = now.Add(1100 * time.Millisecond)
	t3, reason := s.tryAcquirePrelogin(tcpAddr("203.0.113.30", 3002))
	if t3 == nil || reason != "" {
		t.Fatalf("expected admission after refill, got ticket=%v reason=%q", t3, reason)
	}
	t3.Release()
}

func TestTryAcquirePreloginEvictsIdleStateWhenTrackerFull(t *testing.T) {
	now := time.Unix(100, 0).UTC()
	s := newPreloginTestServer(&now)
	s.preloginTrackedMax = 2
	s.preloginStateIdleTTL = 24 * time.Hour

	t1, reason := s.tryAcquirePrelogin(tcpAddr("203.0.113.40", 4000))
	if t1 == nil || reason != "" {
		t.Fatalf("expected first admission success, got ticket=%v reason=%q", t1, reason)
	}
	t1.Release()

	now = now.Add(10 * time.Millisecond)
	t2, reason := s.tryAcquirePrelogin(tcpAddr("203.0.113.41", 4001))
	if t2 == nil || reason != "" {
		t.Fatalf("expected second admission success, got ticket=%v reason=%q", t2, reason)
	}
	t2.Release()

	now = now.Add(10 * time.Millisecond)
	t3, reason := s.tryAcquirePrelogin(tcpAddr("203.0.113.42", 4002))
	if t3 == nil || reason != "" {
		t.Fatalf("expected third admission success with eviction, got ticket=%v reason=%q", t3, reason)
	}
	t3.Release()

	if got := len(s.preloginByIP); got > 2 {
		t.Fatalf("expected tracked prelogin states <=2, got %d", got)
	}
	_, _, _, _, _, evictions, _ := s.PreloginMetricSnapshot()
	if evictions == 0 {
		t.Fatal("expected at least one prelogin state eviction")
	}
}

func TestHandleClientPreloginTimeoutReleasesTicket(t *testing.T) {
	s := NewServer(ServerOptions{
		SkipHandshake:            true,
		WelcomeMessage:           "",
		LoginPrompt:              "",
		PreloginTimeout:          20 * time.Millisecond,
		MaxPreloginSessions:      32,
		AcceptRatePerIP:          10,
		AcceptBurstPerIP:         10,
		PreloginConcurrencyPerIP: 8,
		LoginLineLimit:           32,
		CommandLineLimit:         128,
	}, nil)

	ticket, reason := s.tryAcquirePrelogin(tcpAddr("203.0.113.50", 5000))
	if ticket == nil || reason != "" {
		t.Fatalf("expected prelogin ticket acquisition, got ticket=%v reason=%q", ticket, reason)
	}

	serverConn, clientConn := net.Pipe()
	defer clientConn.Close()

	done := make(chan struct{})
	go func() {
		s.handleClient(serverConn, ticket)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("handleClient did not exit after prelogin timeout")
	}

	active, _, _, _, timeouts, _, _ := s.PreloginMetricSnapshot()
	if active != 0 {
		t.Fatalf("expected prelogin active gauge 0 after timeout, got %d", active)
	}
	if timeouts == 0 {
		t.Fatal("expected prelogin timeout counter increment")
	}
}

func TestPreloginTicketReleasedOnTransportWrapFailure(t *testing.T) {
	s := newPreloginTestServer(nil)
	s.useZiutek = true
	s.wrapConnFn = func(conn net.Conn) (net.Conn, net.Conn, error) {
		return nil, nil, errors.New("wrap failed")
	}

	ticket, reason := s.tryAcquirePrelogin(tcpAddr("203.0.113.60", 6000))
	if ticket == nil || reason != "" {
		t.Fatalf("expected prelogin ticket acquisition, got ticket=%v reason=%q", ticket, reason)
	}

	serverConn, clientConn := net.Pipe()
	defer clientConn.Close()

	done := make(chan struct{})
	go func() {
		s.handleClient(serverConn, ticket)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("handleClient did not exit after wrap failure")
	}

	active, _, _, _, _, _, _ := s.PreloginMetricSnapshot()
	if active != 0 {
		t.Fatalf("expected prelogin active gauge 0 after wrap failure, got %d", active)
	}
}

func TestPreloginActiveGaugeRecoversAfterEarlySetupFailure(t *testing.T) {
	s := newPreloginTestServer(nil)
	s.maxPreloginSessions = 1
	s.useZiutek = true
	s.wrapConnFn = func(conn net.Conn) (net.Conn, net.Conn, error) {
		return nil, nil, errors.New("wrap failed")
	}

	ticket, reason := s.tryAcquirePrelogin(tcpAddr("203.0.113.61", 6100))
	if ticket == nil || reason != "" {
		t.Fatalf("expected first prelogin ticket acquisition, got ticket=%v reason=%q", ticket, reason)
	}

	serverConn, clientConn := net.Pipe()
	defer clientConn.Close()
	s.handleClient(serverConn, ticket)

	nextTicket, nextReason := s.tryAcquirePrelogin(tcpAddr("203.0.113.62", 6200))
	if nextTicket == nil || nextReason != "" {
		t.Fatalf("expected capacity recovery after setup failure, got ticket=%v reason=%q", nextTicket, nextReason)
	}
	nextTicket.Release()
}

func TestTryAcquirePreloginHonorsGlobalRate(t *testing.T) {
	now := time.Unix(100, 0).UTC()
	s := newPreloginTestServer(&now)
	s.acceptRateGlobal = 1
	s.acceptBurstGlobal = 1
	s.acceptRatePerIP = 100
	s.acceptBurstPerIP = 100
	s.acceptRatePerSubnet = 100
	s.acceptBurstPerSubnet = 100

	t1, reason := s.tryAcquirePrelogin(tcpAddr("203.0.113.70", 7000))
	if t1 == nil || reason != "" {
		t.Fatalf("expected first admission success, got ticket=%v reason=%q", t1, reason)
	}
	t1.Release()

	t2, reason := s.tryAcquirePrelogin(tcpAddr("203.0.113.71", 7001))
	if t2 != nil || reason != preloginRejectGlobalRate {
		t.Fatalf("expected global rate reject, got ticket=%v reason=%q", t2, reason)
	}
}

func TestTryAcquirePreloginHonorsSubnetRate(t *testing.T) {
	now := time.Unix(100, 0).UTC()
	s := newPreloginTestServer(&now)
	s.acceptRatePerIP = 100
	s.acceptBurstPerIP = 100
	s.acceptRateGlobal = 100
	s.acceptBurstGlobal = 100
	s.acceptRatePerSubnet = 1
	s.acceptBurstPerSubnet = 1

	t1, reason := s.tryAcquirePrelogin(tcpAddr("203.0.113.80", 8000))
	if t1 == nil || reason != "" {
		t.Fatalf("expected first admission success, got ticket=%v reason=%q", t1, reason)
	}
	t1.Release()

	t2, reason := s.tryAcquirePrelogin(tcpAddr("203.0.113.81", 8001))
	if t2 != nil || reason != preloginRejectSubnetRate {
		t.Fatalf("expected subnet rate reject, got ticket=%v reason=%q", t2, reason)
	}

	now = now.Add(1100 * time.Millisecond)
	t3, reason := s.tryAcquirePrelogin(tcpAddr("203.0.113.82", 8002))
	if t3 == nil || reason != "" {
		t.Fatalf("expected subnet refill success, got ticket=%v reason=%q", t3, reason)
	}
	t3.Release()
}

func TestTryAcquirePreloginHonorsASNAndCountryRate(t *testing.T) {
	now := time.Unix(100, 0).UTC()
	s := newPreloginTestServer(&now)
	s.acceptRatePerIP = 100
	s.acceptBurstPerIP = 100
	s.acceptRateGlobal = 100
	s.acceptBurstGlobal = 100
	s.acceptRatePerSubnet = 100
	s.acceptBurstPerSubnet = 100
	s.acceptRatePerASN = 1
	s.acceptBurstPerASN = 1
	s.acceptRatePerCountry = 1
	s.acceptBurstPerCountry = 1
	s.admissionGeoLookupFn = func(ip string, now time.Time) (string, string) {
		if ip == "203.0.114.10" || ip == "203.0.114.11" {
			return "AS64500", "US"
		}
		if ip == "198.51.100.10" {
			return "AS64501", "CA"
		}
		if ip == "198.51.100.11" {
			return "AS64502", "CA"
		}
		return "", ""
	}

	t1, reason := s.tryAcquirePrelogin(tcpAddr("203.0.114.10", 9000))
	if t1 == nil || reason != "" {
		t.Fatalf("expected first ASN admission success, got ticket=%v reason=%q", t1, reason)
	}
	t1.Release()

	t2, reason := s.tryAcquirePrelogin(tcpAddr("203.0.114.11", 9001))
	if t2 != nil || reason != preloginRejectASNRate {
		t.Fatalf("expected ASN rate reject, got ticket=%v reason=%q", t2, reason)
	}

	now = now.Add(1100 * time.Millisecond)
	t3, reason := s.tryAcquirePrelogin(tcpAddr("198.51.100.10", 9100))
	if t3 == nil || reason != "" {
		t.Fatalf("expected first country admission success, got ticket=%v reason=%q", t3, reason)
	}
	t3.Release()

	t4, reason := s.tryAcquirePrelogin(tcpAddr("198.51.100.11", 9101))
	if t4 != nil || reason != preloginRejectCountryRate {
		t.Fatalf("expected country rate reject, got ticket=%v reason=%q", t4, reason)
	}
}

func TestLogAdmissionRejectAppliesSamplingAndIntervalLimits(t *testing.T) {
	now := time.Unix(100, 0).UTC()
	s := newPreloginTestServer(&now)
	s.admissionLogInterval = time.Second
	s.admissionLogSample = 1.0
	s.admissionLogMaxLines = 2

	s.logAdmissionReject(preloginRejectIPRate, "203.0.113.90")
	s.logAdmissionReject(preloginRejectIPRate, "203.0.113.90")
	s.logAdmissionReject(preloginRejectIPRate, "203.0.113.90")

	s.preloginMu.Lock()
	if got := s.admissionLogLines; got != 2 {
		s.preloginMu.Unlock()
		t.Fatalf("expected admission log sample lines capped at 2, got %d", got)
	}
	if got := s.admissionLogCounts[string(preloginRejectIPRate)]; got != 3 {
		s.preloginMu.Unlock()
		t.Fatalf("expected 3 counted rejects, got %d", got)
	}
	s.preloginMu.Unlock()

	now = now.Add(2 * time.Second)
	s.logAdmissionReject(preloginRejectIPRate, "203.0.113.91")

	s.preloginMu.Lock()
	defer s.preloginMu.Unlock()
	if got := s.admissionLogCounts[string(preloginRejectIPRate)]; got != 1 {
		t.Fatalf("expected interval rollover to reset counts to current event, got %d", got)
	}
}
