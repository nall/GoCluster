package peer

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"dxcluster/config"
	"dxcluster/internal/netutil"
	"dxcluster/spot"
	"dxcluster/strutil"
)

type Manager struct {
	cfg               config.PeeringConfig
	localCall         string
	ingest            chan<- *spot.Spot
	maxAgeSeconds     int
	topology          *topologyStore
	sessions          map[string]*session
	outboundPeers     []PeerEndpoint
	inboundPeers      map[string]PeerEndpoint
	mu                sync.RWMutex
	allowIPs          []*net.IPNet
	allowCalls        map[string]struct{}
	dedupe            *dedupeCache
	ctx               context.Context
	cancel            context.CancelFunc
	listener          net.Listener
	pc92Ch            chan pc92Work
	legacyCh          chan legacyWork
	rawBroadcast      func(string) // optional hook to emit raw lines (e.g., PC26) to telnet clients
	wwvBroadcast      func(kind, line string)
	announceBroadcast func(line string)
	directMessage     func(to, line string)
	reconnects        atomic.Uint64
	userCountFn       func() int
	dropReporter      func(line string)
}

// pc92Work wraps an inbound PC92 frame with the time it was observed so topology
// updates can be applied off the socket read goroutine.
type pc92Work struct {
	frame *Frame
	ts    time.Time
}

// legacyWork wraps legacy topology frames so disk I/O never blocks the read loop.
type legacyWork struct {
	frame *Frame
	ts    time.Time
}

const (
	defaultPC92Queue   = 64
	defaultLegacyQueue = 64
)

func buildPeerRegistry(peers []config.PeeringPeer) ([]PeerEndpoint, map[string]PeerEndpoint, error) {
	outbound := make([]PeerEndpoint, 0, len(peers))
	inbound := make(map[string]PeerEndpoint)
	for _, peerCfg := range peers {
		if !peerCfg.Enabled {
			continue
		}
		endpoint, err := newPeerEndpoint(peerCfg)
		if err != nil {
			return nil, nil, err
		}
		if peerCfg.AllowsOutbound() {
			outbound = append(outbound, endpoint)
		}
		if peerCfg.AllowsInbound() {
			inbound[endpoint.remoteCall] = endpoint
		}
	}
	return outbound, inbound, nil
}

func NewManager(cfg config.PeeringConfig, localCall string, ingest chan<- *spot.Spot, maxAgeSeconds int, dropReporter func(string)) (*Manager, error) {
	if strings.TrimSpace(localCall) == "" {
		return nil, fmt.Errorf("peering local callsign is empty")
	}
	retention := time.Duration(cfg.Topology.RetentionHours) * time.Hour
	if retention <= 0 {
		retention = 24 * time.Hour
	}
	var topo *topologyStore
	var err error
	if strings.TrimSpace(cfg.Topology.DBPath) != "" {
		topo, err = openTopologyStore(cfg.Topology.DBPath, retention)
		if err != nil {
			return nil, err
		}
	}
	allowIPs, err := parseIPACL(cfg.ACL.AllowIPs)
	if err != nil {
		return nil, err
	}
	outboundPeers, inboundPeers, err := buildPeerRegistry(cfg.Peers)
	if err != nil {
		return nil, err
	}
	allowCalls := make(map[string]struct{})
	for _, call := range cfg.ACL.AllowCallsigns {
		call = strutil.NormalizeUpper(call)
		if call == "" {
			continue
		}
		allowCalls[call] = struct{}{}
	}

	return &Manager{
		cfg:           cfg,
		localCall:     strutil.NormalizeUpper(localCall),
		ingest:        ingest,
		maxAgeSeconds: maxAgeSeconds,
		topology:      topo,
		sessions:      make(map[string]*session),
		outboundPeers: outboundPeers,
		inboundPeers:  inboundPeers,
		allowIPs:      allowIPs,
		allowCalls:    allowCalls,
		dedupe:        newDedupeCache(10 * time.Minute),
		dropReporter:  dropReporter,
	}, nil
}

func (m *Manager) Start(ctx context.Context) error {
	if m == nil {
		return fmt.Errorf("nil manager")
	}
	runCtx, cancel := context.WithCancel(ctx)
	m.ctx = runCtx
	m.cancel = cancel

	if m.cfg.ListenPort > 0 {
		addr := fmt.Sprintf(":%d", m.cfg.ListenPort)
		var lc net.ListenConfig
		ln, err := lc.Listen(ctx, "tcp", addr)
		if err != nil {
			return fmt.Errorf("peering listen: %w", err)
		}
		m.listener = ln
		go m.acceptLoop()
	}

	for _, peer := range m.outboundPeers {
		go m.runOutbound(peer)
	}

	// Always run maintenance to prune the peer dedupe cache even when topology
	// persistence is disabled.
	go m.maintenanceLoop(runCtx)

	// Topology updates are handled off the session read goroutine to prevent
	// large PC92 maps from stalling spot delivery. The channel is deliberately
	// bounded; oversize or overflow frames are dropped with a warning.
	if m.topology != nil {
		m.pc92Ch = make(chan pc92Work, defaultPC92Queue)
		go m.topologyWorker(runCtx)
		m.legacyCh = make(chan legacyWork, defaultLegacyQueue)
		go m.legacyWorker(runCtx)
	}
	return nil
}

func (m *Manager) Stop() {
	if m == nil {
		return
	}
	if m.cancel != nil {
		m.cancel()
	}
	if m.listener != nil {
		_ = m.listener.Close()
	}
	m.mu.Lock()
	for _, sess := range m.sessions {
		sess.close()
	}
	m.mu.Unlock()
	if m.topology != nil {
		_ = m.topology.Close()
	}
}

// PublishDX publishes a locally produced spot to peers when the shared
// forwarding policy allows it. Receive-only mode still permits DX command
// spots while suppressing transit relay.
func (m *Manager) PublishDX(s *spot.Spot) bool {
	if s == nil {
		return false
	}
	return m.PublishDXWithComment(s, s.Comment)
}

// PublishDXWithComment publishes a locally produced spot to peers using the
// provided comment text for PC11/PC61 formatting.
func (m *Manager) PublishDXWithComment(s *spot.Spot, comment string) bool {
	if m == nil || s == nil || !m.ShouldPublishLocalSpot(s) {
		return false
	}
	hop := m.cfg.HopCount
	if hop <= 0 {
		hop = defaultHopCount
	}
	m.broadcastSpot(s, comment, hop, m.localCall, nil)
	return true
}

func (m *Manager) PublishWWV(ev WWVEvent) {
	if m == nil {
		return
	}
	m.broadcastWWV(ev)
}

func (m *Manager) HandleFrame(frame *Frame, sess *session) {
	if frame == nil {
		return
	}
	now := time.Now().UTC()
	switch frame.Type {
	case "PC92":
		seen := true
		if frame.Hop > 1 {
			seen = m.dedupe.markSeen(pc92Key(frame), now)
		}
		if !seen {
			return
		}
		if m.topology != nil && m.pc92Ch != nil {
			if m.cfg.PC92MaxBytes > 0 && len(frame.Raw) > m.cfg.PC92MaxBytes {
				log.Printf("Peering: dropping PC92 (%d bytes) from %s: over size limit", len(frame.Raw), sessionLabel(sess))
			} else {
				select {
				case m.pc92Ch <- pc92Work{frame: frame, ts: now}:
				default:
					log.Printf("Peering: dropping PC92 from %s: topology queue full", sessionLabel(sess))
				}
			}
		}
		if frame.Hop > 1 {
			m.forwardFrame(frame, frame.Hop-1, sess, true)
		}
	case "PC19", "PC16", "PC17", "PC21":
		if m.topology != nil && m.legacyCh != nil {
			select {
			case m.legacyCh <- legacyWork{frame: frame, ts: now}:
			default:
				log.Printf("Peering: dropping legacy %s from %s: topology queue full", frame.Type, sessionLabel(sess))
			}
		}
	case "PC26", "PC11", "PC61":
		spotEntry, err := parseSpotFromFrame(frame, sess.remoteCall)
		if err != nil {
			if frame.Type == "PC61" {
				m.reportDrop(formatPC61DropLine(frame, sess, err))
				return
			}
			log.Printf("Peering: parse %s from %s failed: %v", frame.Type, sessionLabel(sess), err)
			return
		}
		accepted := m.ingestSpot(spotEntry)
		if accepted && frame.Hop > 1 && m.shouldRelayDataFrame(frame.Type) {
			key := dxKey(frame, spotEntry)
			if m.dedupe.markSeen(key, now) {
				if frame.Type == "PC26" {
					// Preserve merge semantics by forwarding PC26; pc9x peers only. Telnet clients
					// see the formatted spot via normal broadcast after ingest.
					m.forwardFrame(frame, frame.Hop-1, sess, true)
				} else {
					m.broadcastSpot(spotEntry, spotEntry.Comment, frame.Hop-1, spotEntry.SourceNode, sess)
				}
			}
		}
	case "PC23", "PC73":
		if ev, ok := parseWWV(frame); ok {
			if m.dedupe.markSeen(wwvKey(frame), now) {
				m.broadcastWWV(ev)
			}
		}
	case "PC93":
		if msg, ok := parsePC93(frame); ok {
			if m.dedupe.markSeen(pc93Key(frame), now) {
				m.routePC93(msg)
			}
		}
	}
}

func (m *Manager) ingestSpot(s *spot.Spot) bool {
	if m == nil || s == nil || m.ingest == nil {
		return false
	}
	if m.maxAgeSeconds > 0 {
		if age := time.Since(s.Time); age > time.Duration(m.maxAgeSeconds)*time.Second {
			// Drop stale spots before they enter the shared pipeline to avoid wasting dedupe/work.
			return false
		}
	}
	select {
	case m.ingest <- s:
		return true
	default:
		log.Printf("Peering: ingest queue full, dropping spot from %s", s.SourceNode)
		return false
	}
}

// Purpose: Route a drop line to the UI reporter or logs.
// Key aspects: Uses the optional dropReporter to avoid system log duplication.
// Upstream: PC61 parse failures.
// Downstream: dropReporter or log.Print.
func (m *Manager) reportDrop(line string) {
	if line == "" {
		return
	}
	if m != nil && m.dropReporter != nil {
		m.dropReporter(line)
		return
	}
	log.Print(line)
}

// Purpose: Format a standardized PC61 drop line for the dropped pane.
// Key aspects: Best-effort extraction of fields; reason is stable for parsing.
// Upstream: HandleFrame parse errors.
// Downstream: spot.FreqToBand and spot.NormalizeBand.
func formatPC61DropLine(frame *Frame, sess *session, err error) string {
	reason := pc61DropReason(err)
	dx := "unknown"
	de := "unknown"
	freq := 0.0
	band := "unknown"
	source := sessionLabel(sess)
	if frame != nil {
		fields := frame.payloadFields()
		if len(fields) > 0 {
			if parsed, parseErr := strconv.ParseFloat(strings.TrimSpace(fields[0]), 64); parseErr == nil {
				freq = parsed
				band = spot.NormalizeBand(spot.FreqToBand(parsed))
				if band == "" {
					band = "unknown"
				}
			}
		}
		if len(fields) > 1 {
			dx = strutil.NormalizeUpper(fields[1])
			if dx == "" {
				dx = "unknown"
			}
		}
		if len(fields) > 5 {
			de = strutil.NormalizeUpper(fields[5])
			if de == "" {
				de = "unknown"
			}
		}
		if len(fields) > 6 {
			origin := strings.TrimSpace(fields[6])
			if origin != "" {
				source = origin
			}
		}
	}
	if source == "" {
		source = "unknown"
	}
	return fmt.Sprintf("PC61 drop: reason=%s de=%s dx=%s band=%s freq=%.1f source=%s", reason, de, dx, band, freq, source)
}

func pc61DropReason(err error) string {
	if err == nil {
		return "unknown"
	}
	msg := strings.ToLower(err.Error())
	switch {
	case strings.Contains(msg, "insufficient fields"):
		return "insufficient_fields"
	case strings.Contains(msg, "freq parse"):
		return "freq_parse"
	case strings.Contains(msg, "invalid dx"):
		return "invalid_dx"
	case strings.Contains(msg, "invalid de"):
		return "invalid_de"
	default:
		return "parse_error"
	}
}

func (m *Manager) registerSession(s *session) error {
	if m == nil || s == nil {
		return nil
	}
	key := strings.TrimSpace(s.id)
	if key == "" {
		key = strings.TrimSpace(s.remoteCall)
	}
	if key == "" {
		return fmt.Errorf("peer session identity is empty")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if existing, ok := m.sessions[key]; ok && existing != s {
		return fmt.Errorf("duplicate peer session: %s", key)
	}
	s.id = key
	m.sessions[key] = s
	return nil
}

func (m *Manager) unregisterSession(s *session) {
	if m == nil || s == nil {
		return
	}
	m.mu.Lock()
	if existing, ok := m.sessions[s.id]; ok && existing == s {
		delete(m.sessions, s.id)
	}
	m.mu.Unlock()
}

// SetRawBroadcast installs a callback used to forward raw lines (e.g., PC26) to telnet clients.
// This is optional; when unset, PC26 is only forwarded to peers.
func (m *Manager) SetRawBroadcast(fn func(string)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.rawBroadcast = fn
}

// SetWWVBroadcast installs a callback used to forward WWV/WCY bulletins to telnet clients.
// When unset, PC23/PC73 frames are parsed but not delivered.
func (m *Manager) SetWWVBroadcast(fn func(kind, line string)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.wwvBroadcast = fn
}

// SetAnnouncementBroadcast installs a callback used for PC93 announcements ("To ALL").
func (m *Manager) SetAnnouncementBroadcast(fn func(line string)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.announceBroadcast = fn
}

// SetDirectMessage installs a callback for PC93 talk messages addressed to a specific callsign.
func (m *Manager) SetDirectMessage(fn func(to, line string)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.directMessage = fn
}

func (m *Manager) broadcastSpot(s *spot.Spot, comment string, hop int, origin string, exclude *session) {
	if m == nil || s == nil {
		return
	}
	if m.maxAgeSeconds > 0 {
		if age := time.Since(s.Time); age > time.Duration(m.maxAgeSeconds)*time.Second {
			// Belt-and-suspenders: never forward stale spots to peers.
			return
		}
	}
	if strings.TrimSpace(origin) == "" {
		origin = m.localCall
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, sess := range m.sessions {
		if exclude != nil && sess == exclude {
			continue
		}
		if sess.pc9x {
			line := formatPC61(s, comment, origin, hop)
			m.trySendLine(sess, line, "spot")
		} else {
			line := formatPC11(s, comment, origin, hop)
			m.trySendLine(sess, line, "spot")
		}
	}
}

func (m *Manager) broadcastWWV(ev WWVEvent) {
	if m == nil {
		return
	}
	kind, line := formatWWVLine(ev)
	if line == "" {
		return
	}
	m.mu.RLock()
	cb := m.wwvBroadcast
	m.mu.RUnlock()
	if cb != nil {
		cb(kind, line)
	}
}

func (m *Manager) routePC93(msg pc93Message) {
	if m == nil {
		return
	}
	line := formatPC93Line(msg)
	if line == "" {
		return
	}
	target, broadcast := pc93Target(msg)
	m.mu.RLock()
	announce := m.announceBroadcast
	direct := m.directMessage
	m.mu.RUnlock()
	if !broadcast && target != "" {
		if direct != nil {
			direct(target, line)
		}
		return
	}
	if announce != nil {
		announce(line)
	}
}

func (m *Manager) forwardFrame(frame *Frame, hop int, exclude *session, pc9xOnly bool) {
	if m == nil || frame == nil {
		return
	}
	line := frame.Encode(hop)
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, sess := range m.sessions {
		if exclude != nil && sess == exclude {
			continue
		}
		if pc9xOnly && !sess.pc9x {
			continue
		}
		m.trySendLine(sess, line, "frame")
	}
}

func (m *Manager) trySendLine(sess *session, line string, kind string) {
	if m == nil || sess == nil || strings.TrimSpace(line) == "" {
		return
	}
	if err := sess.sendLine(line); err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, errSessionWriteQueueFull) {
			return
		}
		log.Printf("Peering: failed to enqueue %s for %s: %v", kind, sessionLabel(sess), err)
	}
}

func remoteAddrIP(addr net.Addr) net.IP {
	if addr == nil {
		return nil
	}
	host, _, err := net.SplitHostPort(addr.String())
	if err != nil {
		host = addr.String()
	}
	return net.ParseIP(host)
}

func ipAllowed(blocks []*net.IPNet, addr net.Addr) bool {
	if len(blocks) == 0 {
		return true
	}
	ip := remoteAddrIP(addr)
	if ip == nil {
		return false
	}
	for _, block := range blocks {
		if block.Contains(ip) {
			return true
		}
	}
	return false
}

func (m *Manager) hasActiveSession(id string) bool {
	if m == nil || strings.TrimSpace(id) == "" {
		return false
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, ok := m.sessions[id]
	return ok
}

func (m *Manager) authorizeInbound(call string, addr net.Addr) (PeerEndpoint, error) {
	call = strutil.NormalizeUpper(call)
	if len(m.allowCalls) > 0 {
		if _, ok := m.allowCalls[call]; !ok {
			return PeerEndpoint{}, fmt.Errorf("unauthorized inbound peer: %s", call)
		}
	}
	if !ipAllowed(m.allowIPs, addr) {
		return PeerEndpoint{}, fmt.Errorf("unauthorized inbound peer ip: %s", addr.String())
	}
	peer, ok := m.inboundPeers[call]
	if !ok {
		return PeerEndpoint{}, fmt.Errorf("unauthorized inbound peer: %s", call)
	}
	if !ipAllowed(peer.allowIPs, addr) {
		return PeerEndpoint{}, fmt.Errorf("unauthorized inbound peer ip: %s", addr.String())
	}
	if m.hasActiveSession(peer.ID()) {
		return PeerEndpoint{}, fmt.Errorf("duplicate peer session: %s", call)
	}
	if strings.TrimSpace(peer.host) == "" {
		peer.host = addr.String()
	}
	return peer, nil
}

func (m *Manager) acceptLoop() {
	for {
		conn, err := m.listener.Accept()
		if err != nil {
			if m.ctx != nil && m.ctx.Err() != nil {
				return
			}
			log.Printf("Peering: accept failed: %v", err)
			continue
		}
		if isTCP, enableErr, periodErr := netutil.EnableTCPKeepAlive(conn, 2*time.Minute); isTCP {
			if enableErr != nil {
				log.Printf("Peering: failed to enable keepalive for %s: %v", conn.RemoteAddr(), enableErr)
			}
			if periodErr != nil {
				log.Printf("Peering: failed to set keepalive period for %s: %v", conn.RemoteAddr(), periodErr)
			}
		}
		peer := PeerEndpoint{host: conn.RemoteAddr().String(), port: 0}
		settings := m.sessionSettings(peer)
		sess := newSession(conn, dirInbound, m, peer, settings)
		sess.id = conn.RemoteAddr().String()
		go func() {
			if err := sess.Run(m.ctx); err != nil && m.ctx.Err() == nil {
				log.Printf("Peering: inbound session ended: %v", err)
			}
		}()
	}
}

func (m *Manager) runOutbound(peer PeerEndpoint) {
	backoff := newBackoff(time.Duration(m.cfg.Backoff.BaseMS)*time.Millisecond, time.Duration(m.cfg.Backoff.MaxMS)*time.Millisecond)
	dialer := &net.Dialer{
		Timeout:   10 * time.Second,
		KeepAlive: 2 * time.Minute, // OS-level keepalive for peer links
	}
	for {
		if m.ctx != nil && m.ctx.Err() != nil {
			return
		}
		if m.hasActiveSession(peer.ID()) {
			delay := time.Duration(m.cfg.Backoff.BaseMS) * time.Millisecond
			if delay <= 0 {
				delay = 2 * time.Second
			}
			time.Sleep(delay)
			continue
		}
		addr := net.JoinHostPort(peer.host, strconv.Itoa(peer.port))
		log.Printf("Peering: dialing %s as %s", addr, peer.loginCall)
		conn, err := dialer.Dial("tcp", addr)
		if err != nil {
			delay := backoff.Next()
			log.Printf("Peering: dial %s failed: %v (retry in %s)", addr, err, delay)
			time.Sleep(delay)
			continue
		}
		log.Printf("Peering: connected to %s", addr)
		backoff.Reset()
		settings := m.sessionSettings(peer)
		sess := newSession(conn, dirOutbound, m, peer, settings)
		sess.remoteCall = peer.remoteCall
		if strings.TrimSpace(sess.remoteCall) == "" {
			sess.remoteCall = "*"
		}
		if err := sess.Run(m.ctx); err != nil && m.ctx.Err() == nil {
			log.Printf("Peering: session to %s ended: %v", addr, err)
		}
		m.reconnects.Add(1)
		delay := backoff.Next()
		time.Sleep(delay)
	}
}

// ReconnectCount returns the number of outbound peer reconnect attempts.
func (m *Manager) ReconnectCount() uint64 {
	if m == nil {
		return 0
	}
	return m.reconnects.Load()
}

// SetUserCountProvider wires a live user count callback (e.g., from the telnet server)
// so PC92 K keepalives can advertise current users instead of a static config value.
func (m *Manager) SetUserCountProvider(fn func() int) {
	if m == nil {
		return
	}
	m.userCountFn = fn
}

// liveNodeCount returns 1 (self) plus the number of active peer sessions.
func (m *Manager) liveNodeCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return 1 + len(m.sessions)
}

// ActiveSessionCount returns the number of active peer sessions.
// Purpose: Provide liveness info for dashboards.
// Key aspects: Safe with nil manager; uses read lock.
// Upstream: main stats loop.
// Downstream: none.
func (m *Manager) ActiveSessionCount() int {
	if m == nil {
		return 0
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.sessions)
}

// ActiveSessionSSIDs returns active peer callsigns (including SSID suffixes) for dashboard display.
// Purpose: Expose operator-friendly peer identity instead of aggregate counts.
// Key aspects: Safe with nil manager, read-locked snapshot, sorted and de-duplicated.
// Upstream: main stats loop.
// Downstream: overview ingest sources box.
func (m *Manager) ActiveSessionSSIDs() []string {
	if m == nil {
		return nil
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	if len(m.sessions) == 0 {
		return nil
	}
	ssids := make([]string, 0, len(m.sessions))
	seen := make(map[string]struct{}, len(m.sessions))
	for _, sess := range m.sessions {
		if sess == nil {
			continue
		}
		label := strutil.NormalizeUpper(sess.remoteCall)
		if label == "" || label == "*" {
			continue
		}
		if _, exists := seen[label]; exists {
			continue
		}
		seen[label] = struct{}{}
		ssids = append(ssids, label)
	}
	sort.Strings(ssids)
	return ssids
}

// liveUserCount returns the current telnet client count when provided, otherwise falls back to config.
func (m *Manager) liveUserCount() int {
	if m.userCountFn != nil {
		if v := m.userCountFn(); v >= 0 {
			return v
		}
	}
	return m.cfg.UserCount
}

func (m *Manager) resolveLocalCall(peer PeerEndpoint) string {
	local := peer.loginCall
	if local == "" {
		local = m.cfg.LocalCallsign
	}
	if local == "" {
		local = m.localCall
	}
	return strutil.NormalizeUpper(local)
}

func (m *Manager) sessionSettings(peer PeerEndpoint) sessionSettings {
	return sessionSettings{
		localCall:       m.resolveLocalCall(peer),
		preferPC9x:      peer.preferPC9x,
		nodeVersion:     m.cfg.NodeVersion,
		nodeBuild:       m.cfg.NodeBuild,
		legacyVersion:   m.cfg.LegacyVersion,
		pc92Bitmap:      m.cfg.PC92Bitmap,
		nodeCount:       m.cfg.NodeCount,
		userCount:       m.cfg.UserCount,
		hopCount:        m.cfg.HopCount,
		telnetTransport: m.cfg.TelnetTransport,
		loginTimeout:    time.Duration(m.cfg.Timeouts.LoginSeconds) * time.Second,
		initTimeout:     time.Duration(m.cfg.Timeouts.InitSeconds) * time.Second,
		idleTimeout:     time.Duration(m.cfg.Timeouts.IdleSeconds) * time.Second,
		keepalive:       time.Duration(m.cfg.KeepaliveSeconds) * time.Second,
		configEvery:     time.Duration(m.cfg.ConfigSeconds) * time.Second,
		writeQueue:      m.cfg.WriteQueueSize,
		maxLine:         m.cfg.MaxLineLength,
		pc92MaxBytes:    m.cfg.PC92MaxBytes,
		password:        peer.password,
		logKeepalive:    m.cfg.LogKeepalive,
		logLineTooLong:  m.cfg.LogLineTooLong,
	}
}

func (m *Manager) maintenanceLoop(ctx context.Context) {
	interval := time.Duration(m.cfg.Topology.PersistIntervalSeconds) * time.Second
	if interval <= 0 {
		interval = 5 * time.Minute
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			now := time.Now().UTC()
			if m.topology != nil {
				m.topology.prune(ctx, now)
			}
			if m.dedupe != nil {
				m.dedupe.prune(now)
			}
		}
	}
}

// topologyWorker applies PC92 frames off the socket read goroutine so spot
// traffic never blocks behind topology I/O. Oversize/overflow drops happen at
// enqueue time; this worker best-effort applies what it receives.
func (m *Manager) topologyWorker(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case work := <-m.pc92Ch:
			if m.topology == nil || work.frame == nil {
				continue
			}
			start := time.Now().UTC()
			m.topology.applyPC92Frame(ctx, work.frame, work.ts)
			if dur := time.Since(start); dur > 2*time.Second {
				log.Printf("Peering: PC92 apply slow (%s) from %s", dur.Truncate(time.Millisecond), pc92Origin(work.frame))
			}
		}
	}
}

// legacyWorker applies legacy topology frames off the socket read goroutine so
// synchronous SQLite calls never delay keepalive handling.
func (m *Manager) legacyWorker(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case work := <-m.legacyCh:
			if m.topology == nil || work.frame == nil {
				continue
			}
			m.topology.applyLegacy(ctx, work.frame, work.ts)
		}
	}
}

func pc92Origin(f *Frame) string {
	if f == nil {
		return ""
	}
	fields := f.payloadFields()
	if len(fields) > 0 {
		return strings.TrimSpace(fields[0])
	}
	return ""
}

func sessionLabel(s *session) string {
	if s == nil {
		return ""
	}
	if strings.TrimSpace(s.remoteCall) != "" {
		return s.remoteCall
	}
	return s.id
}
