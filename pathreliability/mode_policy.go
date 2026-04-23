package pathreliability

import (
	"strings"
	"sync/atomic"
)

// ModePolicy is the path-reliability subset derived from the spot taxonomy.
// It stays in this package to avoid an import cycle with spot/config.
type ModePolicy struct {
	Ingest     bool
	OffsetMode string
}

var currentModePolicies atomic.Value

func init() {
	ConfigureModePolicies(map[string]ModePolicy{
		"FT8":  {Ingest: true, OffsetMode: "FT8"},
		"FT4":  {Ingest: true, OffsetMode: "FT4"},
		"CW":   {Ingest: true, OffsetMode: "CW"},
		"RTTY": {Ingest: true, OffsetMode: "RTTY"},
		"PSK":  {Ingest: true, OffsetMode: "PSK"},
		"WSPR": {Ingest: true, OffsetMode: "WSPR"},
	})
}

func ConfigureModePolicies(policies map[string]ModePolicy) {
	clone := make(map[string]ModePolicy, len(policies))
	for mode, policy := range policies {
		key := strings.ToUpper(strings.TrimSpace(mode))
		if key == "" {
			continue
		}
		policy.OffsetMode = strings.ToUpper(strings.TrimSpace(policy.OffsetMode))
		clone[key] = policy
	}
	currentModePolicies.Store(clone)
}

func modePolicy(mode string) (ModePolicy, bool) {
	loaded := currentModePolicies.Load()
	policies, ok := loaded.(map[string]ModePolicy)
	if !ok {
		return ModePolicy{}, false
	}
	if len(policies) == 0 {
		return ModePolicy{}, false
	}
	key := normalizeMode(mode)
	policy, ok := policies[key]
	return policy, ok
}
