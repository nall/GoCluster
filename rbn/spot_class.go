package rbn

import "strings"

// SpotClass is the RBN spot-type field named "mode" in RBN history CSVs.
// It is separate from tx_mode, which is the RF transmission mode (CW, RTTY).
type SpotClass string

const (
	SpotClassUnknown SpotClass = ""
	SpotClassCQ      SpotClass = "CQ"
	SpotClassDX      SpotClass = "DX"
	SpotClassBeacon  SpotClass = "BEACON"
	SpotClassNCDXFB  SpotClass = "NCDXF B"
)

// NormalizeSpotClass normalizes an RBN spot class from CSV/header-style input.
func NormalizeSpotClass(raw string) (SpotClass, bool) {
	normalized := strings.ToUpper(strings.Join(strings.Fields(raw), " "))
	switch normalized {
	case string(SpotClassCQ):
		return SpotClassCQ, true
	case string(SpotClassDX):
		return SpotClassDX, true
	case string(SpotClassBeacon):
		return SpotClassBeacon, true
	case string(SpotClassNCDXFB):
		return SpotClassNCDXFB, true
	default:
		return SpotClassUnknown, false
	}
}

// Accepted reports whether the RBN class is admitted into gocluster.
func (c SpotClass) Accepted() bool {
	switch c {
	case SpotClassCQ, SpotClassBeacon, SpotClassNCDXFB:
		return true
	default:
		return false
	}
}

// IsBeacon reports whether the RBN class should be tagged as beacon traffic.
func (c SpotClass) IsBeacon() bool {
	switch c {
	case SpotClassBeacon, SpotClassNCDXFB:
		return true
	default:
		return false
	}
}
