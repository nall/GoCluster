package cluster

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

const (
	defaultDropLogDedupeMaxKeys = 512
)

// dropLogDeduper protects the operator console/system log from repeated
// validation drops while preserving periodic evidence that a drop class is
// still happening. It is intentionally bounded by maxKeys because bad calls,
// IPs, or feed bugs can create unbounded distinct log text.
type dropLogDeduper struct {
	mu      sync.Mutex
	window  time.Duration
	maxKeys int
	now     func() time.Time
	entries map[string]dropLogDedupeEntry
}

type dropLogDedupeEntry struct {
	nextEmit   time.Time
	lastSeen   time.Time
	suppressed uint64
}

// newDropLogDeduper returns nil when dedupe is disabled so the caller's logging
// path stays simple and explicit.
func newDropLogDeduper(window time.Duration, maxKeys int) *dropLogDeduper {
	if window <= 0 || maxKeys <= 0 {
		return nil
	}
	return &dropLogDeduper{
		window:  window,
		maxKeys: maxKeys,
		now:     func() time.Time { return time.Now().UTC() },
		entries: make(map[string]dropLogDedupeEntry, maxKeys),
	}
}

// Process decides whether a validation log line should be emitted now. Only
// known high-chatter drop formats are deduped; unfamiliar lines pass through so
// new troubleshooting evidence is not hidden by an overbroad filter.
func (d *dropLogDeduper) Process(line string) (string, bool) {
	line = strings.TrimSpace(line)
	if line == "" {
		return "", false
	}
	if d == nil {
		return line, true
	}
	key, ok := dropLogDedupeKey(line)
	if !ok {
		return line, true
	}
	now := d.now()
	d.mu.Lock()
	defer d.mu.Unlock()

	entry, found := d.entries[key]
	if !found {
		d.evictOneIfNeededLocked()
		d.entries[key] = dropLogDedupeEntry{
			nextEmit: now.Add(d.window),
			lastSeen: now,
		}
		return line, true
	}
	entry.lastSeen = now
	if now.Before(entry.nextEmit) {
		entry.suppressed++
		d.entries[key] = entry
		return "", false
	}
	suppressed := entry.suppressed
	entry.suppressed = 0
	entry.nextEmit = now.Add(d.window)
	d.entries[key] = entry
	if suppressed > 0 {
		line = fmt.Sprintf("%s (suppressed=%d over %s)", line, suppressed, d.window)
	}
	return line, true
}

// evictOneIfNeededLocked keeps the dedupe table bounded by removing the
// oldest-seen key. Callers hold d.mu.
func (d *dropLogDeduper) evictOneIfNeededLocked() {
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

// dropLogDedupeKey recognizes the validation-drop families that can repeat at
// high volume. The key keeps role and call because those are the support fields
// operators need when investigating CTY or license-data problems.
func dropLogDedupeKey(line string) (string, bool) {
	fields := strings.Fields(strings.TrimSpace(line))
	if len(fields) < 5 {
		return "", false
	}
	if fields[0] == "CTY" && fields[1] == "drop:" {
		kind := strings.ToLower(cleanLogToken(fields[2]))
		if kind != "unknown" && kind != "invalid" {
			return "", false
		}
		role := strings.ToUpper(cleanLogToken(fields[3]))
		call := strings.ToUpper(cleanLogToken(fields[4]))
		if role == "" || call == "" {
			return "", false
		}
		return "cty:" + kind + ":" + role + ":" + call, true
	}
	if fields[0] == "Unlicensed" && strings.EqualFold(fields[1], "US") {
		role := strings.ToUpper(cleanLogToken(fields[2]))
		call := strings.ToUpper(cleanLogToken(fields[3]))
		if role == "" || call == "" {
			return "", false
		}
		return "unlicensed:" + role + ":" + call, true
	}
	return "", false
}

// cleanLogToken strips cosmetic tags/punctuation before keying log lines so
// formatting changes do not defeat dedupe for the same operational event.
func cleanLogToken(token string) string {
	if token == "" {
		return ""
	}
	trimmed := strings.TrimSpace(token)
	if trimmed == "" {
		return ""
	}
	var b strings.Builder
	b.Grow(len(trimmed))
	inTag := false
	for i := 0; i < len(trimmed); i++ {
		ch := trimmed[i]
		switch ch {
		case '[':
			inTag = true
		case ']':
			if inTag {
				inTag = false
			} else {
				b.WriteByte(ch)
			}
		default:
			if !inTag {
				b.WriteByte(ch)
			}
		}
	}
	return strings.TrimSpace(strings.Trim(b.String(), "():,;"))
}
