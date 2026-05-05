package toxicity

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"dxcluster/internal/yamlconfig"
	"dxcluster/strutil"
)

var (
	gridTokenRE  = regexp.MustCompile(`^[A-R]{2}[0-9]{2}([A-X]{2})?$`)
	dbTokenRE    = regexp.MustCompile(`^[+-]?[0-9]{1,2}(DB)?$`)
	speedTokenRE = regexp.MustCompile(`^[0-9]{1,4}(WPM|BPS|BAUD)$`)
	rstTokenRE   = regexp.MustCompile(`^[1-5][1-9][1-9]?$`)
)

// SafeGate is a conservative whole-comment grammar for routine radio comments.
// It bypasses AI only when every token is explainable as routine ham shorthand.
type SafeGate struct {
	tokens        map[string]struct{}
	eventPrefixes map[string]struct{}
	maxTokens     int
}

type safeGateConfig struct {
	MaxTokens     int      `yaml:"max_tokens"`
	SafeTokens    []string `yaml:"safe_tokens"`
	EventPrefixes []string `yaml:"event_prefixes"`
}

var requiredSafeGatePaths = []yamlconfig.Path{
	{"max_tokens"},
	{"safe_tokens"},
	{"event_prefixes"},
}

func LoadSafeGate(path string) (*SafeGate, error) {
	var cfg safeGateConfig
	if err := yamlconfig.DecodeFile(path, &cfg, requiredSafeGatePaths); err != nil {
		return nil, err
	}
	return NewSafeGate(cfg)
}

func NewSafeGate(cfg safeGateConfig) (*SafeGate, error) {
	if cfg.MaxTokens <= 0 {
		return nil, fmt.Errorf("toxicity_safe_gate.yaml max_tokens must be > 0")
	}
	g := &SafeGate{
		tokens:        make(map[string]struct{}, len(cfg.SafeTokens)),
		eventPrefixes: make(map[string]struct{}, len(cfg.EventPrefixes)),
		maxTokens:     cfg.MaxTokens,
	}
	for _, token := range cfg.SafeTokens {
		token = strutil.NormalizeUpper(token)
		if token == "" {
			return nil, fmt.Errorf("toxicity_safe_gate.yaml safe_tokens must not contain blanks")
		}
		g.tokens[token] = struct{}{}
	}
	for _, prefix := range cfg.EventPrefixes {
		prefix = strutil.NormalizeUpper(prefix)
		if prefix == "" {
			return nil, fmt.Errorf("toxicity_safe_gate.yaml event_prefixes must not contain blanks")
		}
		g.eventPrefixes[prefix] = struct{}{}
	}
	return g, nil
}

func NewSafeGateFromLists(tokens, eventPrefixes []string, maxTokens int) (*SafeGate, error) {
	return NewSafeGate(safeGateConfig{
		MaxTokens:     maxTokens,
		SafeTokens:    tokens,
		EventPrefixes: eventPrefixes,
	})
}

func (g *SafeGate) IsSafe(comment string) bool {
	clean := NormalizeComment(comment)
	if clean == "" {
		return true
	}
	fields := strings.Fields(clean)
	if len(fields) == 0 {
		return true
	}
	if g == nil || len(fields) > g.maxTokens {
		return false
	}
	for _, field := range fields {
		if !g.safeToken(field) {
			return false
		}
	}
	return true
}

func (g *SafeGate) safeToken(token string) bool {
	token = strings.Trim(token, ".,;:!?()[]{}\"'")
	if token == "" {
		return true
	}
	upper := strutil.NormalizeUpper(token)
	if _, ok := g.tokens[upper]; ok {
		return true
	}
	if gridTokenRE.MatchString(upper) || dbTokenRE.MatchString(upper) || speedTokenRE.MatchString(upper) || rstTokenRE.MatchString(upper) {
		return true
	}
	if i := strings.IndexByte(upper, '-'); i > 0 {
		prefix := upper[:i]
		if _, ok := g.eventPrefixes[prefix]; ok && len(upper[i+1:]) > 0 {
			return true
		}
	}
	return false
}

// NormalizeComment preserves language content while removing line/control
// characters that should not affect cache keys or Worker payloads.
func NormalizeComment(comment string) string {
	if comment == "" {
		return ""
	}
	var b strings.Builder
	b.Grow(len(comment))
	space := true
	for _, r := range comment {
		if r == utf8.RuneError || unicode.IsControl(r) || unicode.IsSpace(r) {
			if !space {
				b.WriteByte(' ')
				space = true
			}
			continue
		}
		b.WriteRune(r)
		space = false
	}
	return strings.TrimSpace(b.String())
}

func clampCommentBytes(comment string, maxBytes int) string {
	if maxBytes <= 0 || len(comment) <= maxBytes {
		return comment
	}
	for maxBytes > 0 && !utf8.ValidString(comment[:maxBytes]) {
		maxBytes--
	}
	return strings.TrimSpace(comment[:maxBytes])
}
