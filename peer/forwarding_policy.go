package peer

import (
	"strings"

	"dxcluster/config"
	"dxcluster/spot"
)

// ShouldPublishLocalSpot reports whether a spot from the local output pipeline
// may be published to peers. The current runtime only produces SourceManual
// spots from the DX command path; if additional local manual producers are
// added later, this policy must be revisited explicitly.
func (m *Manager) ShouldPublishLocalSpot(s *spot.Spot) bool {
	if m == nil {
		return false
	}
	return shouldPublishLocalSpot(m.cfg, s)
}

func shouldPublishLocalSpot(cfg config.PeeringConfig, s *spot.Spot) bool {
	if s == nil || s.IsTestSpotter {
		return false
	}
	if !cfg.ForwardSpots {
		return isLocalDXCommandSpot(s)
	}
	return isPeerPublishEligibleSpot(s)
}

func isLocalDXCommandSpot(s *spot.Spot) bool {
	return s != nil && s.SourceType == spot.SourceManual && !s.IsTestSpotter
}

func isPeerPublishEligibleSpot(s *spot.Spot) bool {
	if s == nil || s.IsTestSpotter {
		return false
	}
	if spot.IsSkimmerSource(s.SourceType) {
		return false
	}
	switch s.SourceType {
	case spot.SourceUpstream, spot.SourcePeer:
		return false
	default:
		return true
	}
}

// shouldRelayDataFrame reports whether an inbound peer frame belongs to the
// spot data-plane relay contract. Maintenance/control traffic is intentionally
// excluded so receive-only peering keeps sessions alive when forwarding is off.
func (m *Manager) shouldRelayDataFrame(frameType string) bool {
	if m == nil {
		return false
	}
	return shouldRelayDataFrame(m.cfg, frameType)
}

func shouldRelayDataFrame(cfg config.PeeringConfig, frameType string) bool {
	if !cfg.ForwardSpots {
		return false
	}
	switch strings.TrimSpace(frameType) {
	case "PC11", "PC61", "PC26":
		return true
	default:
		return false
	}
}
