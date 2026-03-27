package spot

// IsLocalSelfSpot reports whether a spot matches the current local self-spot
// contract: a non-test local manual spot whose normalized DX and DE callsigns
// are identical. The runtime currently creates such spots only from the local
// DX command path; if new local manual producers are added later, this helper
// must be revisited explicitly rather than silently widening the bypass.
func IsLocalSelfSpot(s *Spot) bool {
	if s == nil || s.IsTestSpotter || s.SourceType != SourceManual {
		return false
	}
	dxCall := s.DXCallNorm
	if dxCall == "" {
		dxCall = NormalizeCallsign(s.DXCall)
	}
	if dxCall == "" {
		return false
	}
	deCall := s.DECallNorm
	if deCall == "" {
		deCall = NormalizeCallsign(s.DECall)
	}
	return deCall != "" && dxCall == deCall
}
