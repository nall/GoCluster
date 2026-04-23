package main

import (
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"dxcluster/config"
	"dxcluster/spot"
	"dxcluster/strutil"
)

const (
	defaultDroppedCallLogDedupeMaxKeys = 512

	droppedCallRoleDE = "DE"
	droppedCallRoleDX = "DX"
)

type droppedCallLogEvent struct {
	Source string
	Role   string
	Reason string
	Call   string
	DE     string
	DX     string
	Mode   string
	Detail string
}

type droppedCallLogger struct {
	badDEDX   lineSink
	noLicense lineSink
	harmonics lineSink

	badDEDXDedupe   *droppedCallLogDeduper
	noLicenseDedupe *droppedCallLogDeduper
	harmonicsDedupe *droppedCallLogDeduper
}

func newDroppedCallLogger(cfg config.DroppedCallLoggingConfig) (*droppedCallLogger, error) {
	if !cfg.Enabled {
		//nolint:nilnil // Disabled logging intentionally means no logger and no setup error.
		return nil, nil
	}
	logger := &droppedCallLogger{}
	var setupErrs []string
	setup := func(enabled bool, subdir string) lineSink {
		if !enabled {
			return nil
		}
		sink, err := newDailyLogSink(filepath.Join(cfg.Dir, subdir), cfg.RetentionDays)
		if err != nil {
			setupErrs = append(setupErrs, fmt.Sprintf("%s: %v", subdir, err))
			return nil
		}
		return sink
	}
	logger.badDEDX = setup(cfg.BadDEDX, "bad_de_dx")
	logger.noLicense = setup(cfg.NoLicense, "no_license")
	logger.harmonics = setup(cfg.Harmonics, "harmonics")

	window := time.Duration(cfg.DedupeWindowSeconds) * time.Second
	logger.badDEDXDedupe = newDroppedCallLogDeduper(window, defaultDroppedCallLogDedupeMaxKeys)
	logger.noLicenseDedupe = newDroppedCallLogDeduper(window, defaultDroppedCallLogDedupeMaxKeys)
	logger.harmonicsDedupe = newDroppedCallLogDeduper(window, defaultDroppedCallLogDedupeMaxKeys)

	if len(setupErrs) > 0 {
		return logger, fmt.Errorf("dropped-call logging setup: %s", strings.Join(setupErrs, "; "))
	}
	return logger, nil
}

func (l *droppedCallLogger) Close() error {
	if l == nil {
		return nil
	}
	var firstErr error
	for _, sink := range []lineSink{l.badDEDX, l.noLicense, l.harmonics} {
		if sink == nil {
			continue
		}
		if err := sink.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

func (l *droppedCallLogger) LogBadCall(source, role, reason, call, deCall, dxCall, mode, detail string) {
	l.write(l.badDEDX, l.badDEDXDedupe, droppedCallLogEvent{
		Source: source,
		Role:   role,
		Reason: reason,
		Call:   call,
		DE:     deCall,
		DX:     dxCall,
		Mode:   mode,
		Detail: detail,
	})
}

func (l *droppedCallLogger) LogNoLicense(source, role, call, deCall, dxCall, mode, detail string) {
	l.write(l.noLicense, l.noLicenseDedupe, droppedCallLogEvent{
		Source: source,
		Role:   role,
		Reason: "unlicensed_us",
		Call:   call,
		DE:     deCall,
		DX:     dxCall,
		Mode:   mode,
		Detail: detail,
	})
}

func (l *droppedCallLogger) LogHarmonic(source, call, deCall, dxCall, mode, detail string) {
	l.write(l.harmonics, l.harmonicsDedupe, droppedCallLogEvent{
		Source: source,
		Role:   droppedCallRoleDX,
		Reason: "harmonic",
		Call:   call,
		DE:     deCall,
		DX:     dxCall,
		Mode:   mode,
		Detail: detail,
	})
}

func (l *droppedCallLogger) write(sink lineSink, deduper *droppedCallLogDeduper, event droppedCallLogEvent) {
	if l == nil || sink == nil {
		return
	}
	event = normalizeDroppedCallLogEvent(event)
	if deduper != nil {
		var ok bool
		event, ok = deduper.Process(event)
		if !ok {
			return
		}
	}
	sink.WriteLine(event.Line(), time.Now().UTC())
}

func normalizeDroppedCallLogEvent(event droppedCallLogEvent) droppedCallLogEvent {
	event.Source = droppedLogValue(event.Source, "unknown")
	event.Role = strutil.NormalizeUpper(event.Role)
	if event.Role != droppedCallRoleDE && event.Role != droppedCallRoleDX {
		event.Role = "unknown"
	}
	event.Reason = droppedLogValue(event.Reason, "unknown")
	event.Call = droppedCallValue(event.Call)
	event.DE = droppedCallValue(event.DE)
	event.DX = droppedCallValue(event.DX)
	event.Mode = droppedLogValue(event.Mode, "unknown")
	event.Detail = droppedLogValue(event.Detail, "none")
	return event
}

func (e droppedCallLogEvent) Line() string {
	return "source=" + e.Source +
		" role=" + e.Role +
		" reason=" + e.Reason +
		" call=" + e.Call +
		" de=" + e.DE +
		" dx=" + e.DX +
		" mode=" + e.Mode +
		" detail=" + e.Detail
}

func droppedCallValue(value string) string {
	return droppedLogValue(value, "unknown")
}

func droppedCallSourceFromSpot(s *spot.Spot) string {
	if s == nil {
		return ""
	}
	if source := strings.TrimSpace(s.SourceNode); source != "" {
		return source
	}
	if s.SourceType != "" {
		return string(s.SourceType)
	}
	return ""
}

func droppedCallDEFromSpot(s *spot.Spot) string {
	if s == nil {
		return ""
	}
	if call := strings.TrimSpace(s.DECallNorm); call != "" {
		return call
	}
	return s.DECall
}

func droppedCallDXFromSpot(s *spot.Spot) string {
	if s == nil {
		return ""
	}
	if call := strings.TrimSpace(s.DXCallNorm); call != "" {
		return call
	}
	return s.DXCall
}

func droppedCallModeFromSpot(s *spot.Spot) string {
	if s == nil {
		return ""
	}
	if mode := strings.TrimSpace(s.ModeNorm); mode != "" {
		return mode
	}
	return s.Mode
}

func droppedLogValue(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		value = fallback
	}
	value = strings.ReplaceAll(value, "\r", "_")
	value = strings.ReplaceAll(value, "\n", "_")
	value = strings.Join(strings.Fields(value), "_")
	if value == "" {
		return fallback
	}
	return value
}

type droppedCallLogDeduper struct {
	mu      sync.Mutex
	window  time.Duration
	maxKeys int
	now     func() time.Time
	entries map[string]droppedCallLogDedupeEntry
}

type droppedCallLogDedupeEntry struct {
	nextEmit   time.Time
	lastSeen   time.Time
	suppressed uint64
}

func newDroppedCallLogDeduper(window time.Duration, maxKeys int) *droppedCallLogDeduper {
	if window <= 0 || maxKeys <= 0 {
		return nil
	}
	return &droppedCallLogDeduper{
		window:  window,
		maxKeys: maxKeys,
		now:     func() time.Time { return time.Now().UTC() },
		entries: make(map[string]droppedCallLogDedupeEntry, maxKeys),
	}
}

func (d *droppedCallLogDeduper) Process(event droppedCallLogEvent) (droppedCallLogEvent, bool) {
	if d == nil {
		return event, true
	}
	key := droppedCallLogDedupeKey(event)
	now := d.now()
	d.mu.Lock()
	defer d.mu.Unlock()

	entry, found := d.entries[key]
	if !found {
		d.evictOneIfNeededLocked()
		d.entries[key] = droppedCallLogDedupeEntry{
			nextEmit: now.Add(d.window),
			lastSeen: now,
		}
		return event, true
	}
	entry.lastSeen = now
	if now.Before(entry.nextEmit) {
		entry.suppressed++
		d.entries[key] = entry
		return droppedCallLogEvent{}, false
	}
	suppressed := entry.suppressed
	entry.suppressed = 0
	entry.nextEmit = now.Add(d.window)
	d.entries[key] = entry
	if suppressed > 0 {
		if event.Detail == "" || event.Detail == "none" {
			event.Detail = fmt.Sprintf("suppressed=%d_window=%s", suppressed, d.window)
		} else {
			event.Detail = fmt.Sprintf("%s,suppressed=%d_window=%s", event.Detail, suppressed, d.window)
		}
	}
	return event, true
}

func droppedCallLogDedupeKey(event droppedCallLogEvent) string {
	return event.Source + "|" + event.Role + "|" + event.Reason + "|" + event.Call
}

func (d *droppedCallLogDeduper) evictOneIfNeededLocked() {
	if d == nil || d.maxKeys <= 0 {
		return
	}
	if len(d.entries) < d.maxKeys {
		return
	}
	var oldestKey string
	var oldestSeen time.Time
	haveOldest := false
	for key, entry := range d.entries {
		if !haveOldest || entry.lastSeen.Before(oldestSeen) {
			oldestKey = key
			oldestSeen = entry.lastSeen
			haveOldest = true
		}
	}
	if haveOldest {
		delete(d.entries, oldestKey)
	}
}
