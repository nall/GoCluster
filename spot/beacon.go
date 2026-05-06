// File role: Beacon classification helpers for canonical spot records.
// Crawler notes: Preserve source-class beacon state separately from comments so
// peer compatibility and display/archive fallbacks can make different choices.
package spot

import (
	"strings"
	"sync"
)

var beaconCommentKeywords = []string{"NCDXF B", "BEACON", "BCN"}

const (
	BeaconCommentDefault = "BEACON"
	BeaconCommentNCDXF   = "NCDXF BEACON"
)

// Purpose: Detect beacon markers in a comment string.
// Key aspects: Case-insensitive substring match against known keywords.
// Upstream: RefreshBeaconFlag.
// Downstream: strings.ToUpper and strings.Contains.
// commentContainsBeaconKeyword reports true when the comment mentions well-known beacon markers.
func commentContainsBeaconKeyword(comment string) bool {
	if comment == "" {
		return false
	}
	upper := strings.ToUpper(comment)
	for _, keyword := range beaconCommentKeywords {
		if strings.Contains(upper, keyword) {
			return true
		}
	}
	return false
}

// RefreshBeaconFlag refreshes the IsBeacon flag from DX call and comment.
// Key aspects: Checks /B callsign suffix and known comment keywords.
// Upstream: Spot normalization and output pipeline.
// Downstream: IsBeaconCall and commentContainsBeaconKeyword.
// RefreshBeaconFlag recalculates the IsBeacon flag using the DX call and latest comment text.
func (s *Spot) RefreshBeaconFlag() {
	if s == nil {
		return
	}
	if s.DXCallNorm != "" {
		s.IsBeacon = s.BeaconSourceClass || strings.HasSuffix(s.DXCallNorm, "/B") || commentContainsBeaconKeyword(s.Comment)
		return
	}
	s.IsBeacon = s.BeaconSourceClass || IsBeaconCall(s.DXCall) || commentContainsBeaconKeyword(s.Comment)
}

// EnsureBlankBeaconComment stores the display fallback for a caller-owned beacon spot.
// Callers should use this only before an archive/history snapshot or another
// owned handoff where synthetic comment text is intended to become durable.
func (s *Spot) EnsureBlankBeaconComment() bool {
	if s == nil || !s.IsBeacon || sanitizeDXClusterComment(s.Comment) != "" {
		return false
	}
	s.Comment = s.blankBeaconComment()
	s.formatted = ""
	s.formatOnce = sync.Once{}
	return true
}

func (s *Spot) blankBeaconComment() string {
	if s == nil || strings.TrimSpace(s.BeaconComment) == "" {
		return BeaconCommentDefault
	}
	comment := sanitizeDXClusterComment(s.BeaconComment)
	if comment == "" {
		return BeaconCommentDefault
	}
	return comment
}
