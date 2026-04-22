package spot

import (
	"strings"
)

// EventMask is a fixed bitset of supported portable/activation event families.
// It stays on Spot rather than a slice/map so fan-out filtering and archive
// snapshots do not add per-spot allocations.
type EventMask uint16

const (
	EventLLOTA EventMask = 1 << iota
	EventIOTA
	EventPOTA
	EventSOTA
	EventWWFF
)

// SupportedEvents lists user-facing EVENT filter tokens in stable display order.
var SupportedEvents = []string{"LLOTA", "IOTA", "POTA", "SOTA", "WWFF"}

var eventByName = map[string]EventMask{
	"LLOTA": EventLLOTA,
	"IOTA":  EventIOTA,
	"POTA":  EventPOTA,
	"SOTA":  EventSOTA,
	"WWFF":  EventWWFF,
}

// NormalizeEvent returns a canonical EVENT family token or empty if unsupported.
func NormalizeEvent(event string) string {
	event = strings.ToUpper(strings.TrimSpace(event))
	if eventByName[event] == 0 {
		return ""
	}
	return event
}

// EventMaskForName returns the bit corresponding to a canonical EVENT token.
func EventMaskForName(event string) EventMask {
	return eventByName[NormalizeEvent(event)]
}

// EventNames returns canonical EVENT names present in mask, in stable order.
func EventNames(mask EventMask) []string {
	if mask == 0 {
		return nil
	}
	out := make([]string, 0, len(SupportedEvents))
	for _, name := range SupportedEvents {
		if bit := eventByName[name]; bit != 0 && mask&bit != 0 {
			out = append(out, name)
		}
	}
	return out
}

// EventString returns a canonical comma-separated EVENT list for archive storage.
func EventString(mask EventMask) string {
	return strings.Join(EventNames(mask), ",")
}

// ParseEventString parses a canonical or whitespace/comma-separated EVENT list.
func ParseEventString(value string) EventMask {
	fields := strings.FieldsFunc(value, func(r rune) bool {
		return r == ',' || r == ' ' || r == '\t'
	})
	var mask EventMask
	for _, field := range fields {
		mask |= EventMaskForName(field)
	}
	return mask
}

func eventFromCommentToken(upper string) EventMask {
	if bit := eventByName[upper]; bit != 0 {
		return bit
	}
	for _, name := range SupportedEvents {
		prefix := name + "-"
		if strings.HasPrefix(upper, prefix) && validEventReferenceSuffix(upper[len(prefix):]) {
			return eventByName[name]
		}
	}
	return 0
}

func validEventReferenceSuffix(s string) bool {
	if s == "" {
		return false
	}
	hasAlnum := false
	for i := 0; i < len(s); i++ {
		ch := s[i]
		switch {
		case ch >= 'A' && ch <= 'Z':
			hasAlnum = true
		case ch >= '0' && ch <= '9':
			hasAlnum = true
		case ch == '-':
		default:
			return false
		}
	}
	return hasAlnum
}
