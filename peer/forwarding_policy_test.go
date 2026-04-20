package peer

import (
	"testing"

	"dxcluster/config"
	"dxcluster/spot"
)

func TestShouldPublishLocalSpotForwardDisabledAllowsOnlyLocalDX(t *testing.T) {
	m := &Manager{cfg: config.PeeringConfig{ForwardSpots: false}}
	tests := []struct {
		name string
		src  spot.SourceType
		test bool
		want bool
	}{
		{name: "manual allowed", src: spot.SourceManual, want: true},
		{name: "manual test spotter blocked", src: spot.SourceManual, test: true, want: false},
		{name: "peer blocked", src: spot.SourcePeer, want: false},
		{name: "upstream blocked", src: spot.SourceUpstream, want: false},
		{name: "rbn blocked", src: spot.SourceRBN, want: false},
		{name: "ft8 blocked", src: spot.SourceFT8, want: false},
		{name: "pskreporter blocked", src: spot.SourcePSKReporter, want: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s := spot.NewSpot("K1ABC", "W1XYZ", 7074.0, "FT8")
			s.SourceType = tc.src
			s.IsTestSpotter = tc.test
			if got := m.ShouldPublishLocalSpot(s); got != tc.want {
				t.Fatalf("ShouldPublishLocalSpot(%s,test=%v)=%v want %v", tc.src, tc.test, got, tc.want)
			}
		})
	}
}

func TestShouldPublishLocalSpotForwardEnabledUsesPeerPublishPolicy(t *testing.T) {
	m := &Manager{cfg: config.PeeringConfig{ForwardSpots: true}}
	tests := []struct {
		name string
		src  spot.SourceType
		test bool
		want bool
	}{
		{name: "manual allowed", src: spot.SourceManual, want: true},
		{name: "manual test spotter blocked", src: spot.SourceManual, test: true, want: false},
		{name: "upstream blocked", src: spot.SourceUpstream, want: false},
		{name: "peer blocked", src: spot.SourcePeer, want: false},
		{name: "rbn blocked", src: spot.SourceRBN, want: false},
		{name: "ft8 blocked", src: spot.SourceFT8, want: false},
		{name: "ft4 blocked", src: spot.SourceFT4, want: false},
		{name: "pskreporter blocked", src: spot.SourcePSKReporter, want: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s := spot.NewSpot("K1ABC", "W1XYZ", 7074.0, "FT8")
			s.SourceType = tc.src
			s.IsTestSpotter = tc.test
			if got := m.ShouldPublishLocalSpot(s); got != tc.want {
				t.Fatalf("ShouldPublishLocalSpot(%s,test=%v)=%v want %v", tc.src, tc.test, got, tc.want)
			}
		})
	}
}

func TestShouldPublishLocalSelfSpotWhenForwardingDisabled(t *testing.T) {
	m := &Manager{cfg: config.PeeringConfig{ForwardSpots: false}}
	s := spot.NewSpot("K1SELF", "K1SELF", 28400.0, "SSB")
	if !m.ShouldPublishLocalSpot(s) {
		t.Fatalf("expected local self spot to remain peer-publish eligible")
	}
}

func TestShouldRelayDataFrame(t *testing.T) {
	tests := []struct {
		name    string
		forward bool
		frame   string
		want    bool
	}{
		{name: "pc11 disabled", frame: "PC11", want: false},
		{name: "pc61 disabled", frame: "PC61", want: false},
		{name: "pc26 disabled", frame: "PC26", want: false},
		{name: "pc11 enabled", forward: true, frame: "PC11", want: true},
		{name: "pc61 enabled", forward: true, frame: "PC61", want: true},
		{name: "pc26 enabled", forward: true, frame: "PC26", want: true},
		{name: "pc92 control not spot relay", forward: true, frame: "PC92", want: false},
		{name: "pc93 control not spot relay", forward: true, frame: "PC93", want: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := &Manager{cfg: config.PeeringConfig{ForwardSpots: tc.forward}}
			if got := m.shouldRelayDataFrame(tc.frame); got != tc.want {
				t.Fatalf("shouldRelayDataFrame(%q, forward=%v)=%v want %v", tc.frame, tc.forward, got, tc.want)
			}
		})
	}
}
