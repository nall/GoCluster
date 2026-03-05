package peer

import (
	"dxcluster/strutil"
	"fmt"
	"strconv"
	"strings"
)

const (
	telnetIAC  = 255
	telnetDONT = 254
	telnetDO   = 253
	telnetWONT = 252
	telnetWILL = 251
	telnetSB   = 250
	telnetSE   = 240
)

// telnetParser strips telnet IAC sequences from input and returns clean payload bytes plus replies.
// Replies perform a minimal refuse-all negotiation to keep the link in character mode.
type telnetParser struct{}

// Feed strips telnet IAC sequences and emits minimal refusal replies.
// Key aspects: Filters subnegotiation payloads and replies with WONT/DONT.
// Upstream: Peer reader for native telnet mode.
// Downstream: None.
func (p *telnetParser) Feed(input []byte) (output []byte, replies [][]byte) {
	var out []byte
	var inIAC, inSB bool
	for i := 0; i < len(input); i++ {
		b := input[i]
		if inIAC {
			switch b {
			case telnetSB:
				inSB = true
			case telnetSE:
				inSB = false
			case telnetDO:
				if i+1 < len(input) {
					replies = append(replies, []byte{telnetIAC, telnetWONT, input[i+1]})
					i++
				}
			case telnetWILL:
				if i+1 < len(input) {
					replies = append(replies, []byte{telnetIAC, telnetDONT, input[i+1]})
					i++
				}
			case telnetIAC:
				out = append(out, telnetIAC)
			}
			inIAC = false
			continue
		}
		if b == telnetIAC {
			inIAC = true
			continue
		}
		if inSB {
			continue
		}
		out = append(out, b)
	}
	return out, replies
}

// Frame represents a parsed PC protocol sentence.
type Frame struct {
	Type   string
	Fields []string
	Hop    int
	Raw    string
}

// ParseFrame parses a caret-delimited PC frame line into a Frame.
// Key aspects: Trims trailing "~" and extracts hop suffix.
// Upstream: Peer reader.
// Downstream: Frame payload handling.
func ParseFrame(line string) (*Frame, error) {
	raw := strings.TrimSpace(line)
	if raw == "" {
		return nil, fmt.Errorf("empty line")
	}
	raw = strings.TrimSuffix(raw, "~")
	parts := strings.Split(raw, "^")
	if len(parts) == 0 {
		return nil, fmt.Errorf("no parts")
	}
	f := &Frame{Raw: line}
	f.Type = strutil.NormalizeUpper(parts[0])
	payload, hop := stripTrailingHopSuffix(parts[1:])
	f.Fields = payload
	f.Hop = hop
	return f, nil
}

// Encode encodes a Frame back to wire format with optional hop override.
// Key aspects: Preserves fields and appends Hn when hop>0.
// Upstream: Peer writer.
// Downstream: fmt.Sprintf.
func (f *Frame) Encode(hop int) string {
	if f == nil {
		return ""
	}
	// Defensive canonicalization ensures we never emit stacked hop suffixes even
	// when a caller passes legacy fields that still include trailing H tokens.
	fields, _ := stripTrailingHopSuffix(f.Fields)
	out := f.Type + "^" + strings.Join(fields, "^")
	if hop > 0 {
		if !strings.HasSuffix(out, "^") {
			out += "^"
		}
		out += fmt.Sprintf("H%d^", hop)
	}
	return out
}

// Purpose: Return payload fields excluding hop marker.
// Key aspects: Preserves trailing empty fields for protocol fidelity.
// Upstream: parseSpotFromFrame and PC parsers.
// Downstream: None.
func (f *Frame) payloadFields() []string {
	if f == nil {
		return nil
	}
	return PayloadFields(f.Fields)
}

// PayloadFields returns non-hop payload fields with trailing empties preserved.
func PayloadFields(fields []string) []string {
	if len(fields) == 0 {
		return fields
	}
	out, _ := stripTrailingHopSuffix(fields)
	return out
}

// stripTrailingHopSuffix removes a trailing hop suffix sequence (e.g., H95 or
// H95,H94,H93) and returns the payload fields plus effective hop. The effective
// hop is the rightmost numeric hop token in the trailing suffix.
func stripTrailingHopSuffix(fields []string) ([]string, int) {
	if len(fields) == 0 {
		return fields, 0
	}
	out := make([]string, len(fields))
	copy(out, fields)

	i := len(out) - 1
	for i >= 0 && strings.TrimSpace(out[i]) == "" {
		i--
	}
	if i < 0 {
		return out, 0
	}

	hop := 0
	haveSuffix := false
	for i >= 0 {
		trimmed := strings.TrimSpace(out[i])
		v, isHopLike, ok := parseHopToken(trimmed)
		if !isHopLike {
			break
		}
		haveSuffix = true
		if ok && hop == 0 {
			hop = v
		}
		i--
		for i >= 0 && strings.TrimSpace(out[i]) == "" {
			i--
		}
	}
	if !haveSuffix {
		return out, 0
	}
	return out[:i+1], hop
}

// parseHopToken classifies hop-like tokens (H...) and, when numeric, returns
// their integer value.
func parseHopToken(token string) (value int, isHopLike bool, ok bool) {
	token = strings.TrimSpace(token)
	if len(token) < 2 {
		return 0, false, false
	}
	if token[0] != 'H' && token[0] != 'h' {
		return 0, false, false
	}
	if token[1] < '0' || token[1] > '9' {
		return 0, false, false
	}
	v, err := strconv.Atoi(token[1:])
	if err != nil {
		return 0, true, false
	}
	return v, true, true
}
