package peer

import (
	"fmt"
	"hash/fnv"
	"strconv"
	"strings"

	"dxcluster/spot"
	"dxcluster/strutil"
)

// Purpose: Build dedupe keys for peer frames and spots.
// Key aspects: Encodes frame type and relevant identifiers for uniqueness.
// Upstream: Peer dedupe caches.
// Downstream: fmt.Sprintf.
func pc92Key(f *Frame) string {
	if f == nil {
		return "pc92:nil"
	}
	fields := f.payloadFields()
	if len(fields) < 3 {
		return fmt.Sprintf("pc92:%s:short:%s", f.Type, strings.TrimSpace(f.Raw))
	}
	origin := strutil.NormalizeUpper(fields[0])
	ts := strings.TrimSpace(fields[1])
	recordType := strutil.NormalizeUpper(fields[2])

	h := fnv.New64a()
	entryCount := 0
	for _, entry := range fields[3:] {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		_, isHopLike, _ := parseHopToken(entry)
		if isHopLike {
			continue
		}
		_, _ = h.Write([]byte(entry))
		_, _ = h.Write([]byte{0x1f})
		entryCount++
	}
	return "pc92:" + origin + ":" + ts + ":" + recordType + ":" + strconv.Itoa(entryCount) + ":" + strconv.FormatUint(h.Sum64(), 16)
}

// Purpose: Build a dedupe key for a DX spot frame.
// Key aspects: Uses DX/DE/frequency/time to preserve ordering.
// Upstream: Peer dedupe caches for DX frames.
// Downstream: fmt.Sprintf.
func dxKey(f *Frame, s *spot.Spot) string {
	return fmt.Sprintf("dx:%s:%s:%s:%.1f:%d", f.Type, s.DXCall, s.DECall, s.Frequency, s.Time.Unix())
}

// Purpose: Build a dedupe key for WWV/WCY frames.
// Key aspects: Uses canonical payload content so hop-only route variants collapse.
// Upstream: Peer dedupe caches.
// Downstream: fmt.Sprintf.
func wwvKey(f *Frame) string {
	return fmt.Sprintf("wwv:%s:%s", f.Type, canonicalPayloadKey(f))
}

// Purpose: Build a dedupe key for PC93 announcement frames.
// Key aspects: Uses parsed message content so hop-only route variants collapse.
// Upstream: Peer dedupe caches.
// Downstream: fmt.Sprintf.
func pc93Key(f *Frame) string {
	if msg, ok := parsePC93(f); ok {
		return "pc93:" + f.Type + ":" + canonicalPC93MessageKey(msg)
	}
	return fmt.Sprintf("pc93:%s:%s", f.Type, canonicalPayloadKey(f))
}

func canonicalPC93MessageKey(msg pc93Message) string {
	fields := []string{
		msg.NodeCall,
		msg.Timestamp,
		msg.To,
		msg.From,
		msg.Via,
		msg.Text,
		msg.Onode,
		msg.IP,
	}
	for i, field := range fields {
		fields[i] = strings.TrimSpace(field)
	}
	return strings.Join(fields, "\x1f")
}

func canonicalPayloadKey(f *Frame) string {
	if f == nil {
		return "nil"
	}
	fields := f.payloadFields()
	if len(fields) == 0 {
		return ""
	}
	parts := make([]string, 0, len(fields))
	for _, field := range fields {
		parts = append(parts, strings.TrimSpace(field))
	}
	return strings.Join(parts, "\x1f")
}
