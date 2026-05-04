// File role: Centralizes CTY prefilter admission checks for runtime cluster
// paths before CTY metadata lookup.
package cluster

import (
	"strings"

	"dxcluster/spot"
	"dxcluster/uls"
)

func shouldRejectCTYCall(call string) bool {
	normalized := spot.NormalizeCallsign(call)
	base := strings.TrimSpace(uls.NormalizeForLicense(normalized))
	if uls.AllowlistMatchAny(base) {
		return false
	}
	if base != "" {
		return !spot.IsValidNormalizedCallsign(base)
	}
	return normalized != "" && !spot.IsValidNormalizedCallsign(normalized)
}
