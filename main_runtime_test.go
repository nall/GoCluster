package main

import (
	"testing"
	"time"

	"dxcluster/config"
	"dxcluster/dedup"
)

func TestSelectArchivePeerSecondaryPolicy(t *testing.T) {
	fastWindow := 2 * time.Minute
	medWindow := 5 * time.Minute
	slowWindow := 8 * time.Minute

	tests := []struct {
		name       string
		cfg        *config.Config
		fastWindow time.Duration
		medWindow  time.Duration
		slowWindow time.Duration
		want       archivePeerSecondaryPolicy
	}{
		{
			name: "prefers med window",
			cfg: &config.Config{
				CallCorrection: config.CallCorrectionConfig{Enabled: true, StabilizerEnabled: true},
				Dedup: config.DedupConfig{
					SecondaryFastPreferStrong: true,
					SecondaryMedPreferStrong:  false,
					SecondarySlowPreferStrong: true,
				},
			},
			fastWindow: fastWindow,
			medWindow:  medWindow,
			slowWindow: slowWindow,
			want: archivePeerSecondaryPolicy{
				window:       medWindow,
				preferStrong: false,
				label:        "med",
				keyMode:      dedup.SecondaryKeyGrid2,
			},
		},
		{
			name: "falls back to fast",
			cfg: &config.Config{
				CallCorrection: config.CallCorrectionConfig{Enabled: true, StabilizerEnabled: true},
				Dedup: config.DedupConfig{
					SecondaryFastPreferStrong: true,
				},
			},
			fastWindow: fastWindow,
			medWindow:  0,
			slowWindow: 0,
			want: archivePeerSecondaryPolicy{
				window:       fastWindow,
				preferStrong: true,
				label:        "fast-fallback",
				keyMode:      dedup.SecondaryKeyGrid2,
			},
		},
		{
			name: "falls back to slow cq zone",
			cfg: &config.Config{
				CallCorrection: config.CallCorrectionConfig{Enabled: true, StabilizerEnabled: true},
				Dedup: config.DedupConfig{
					SecondarySlowPreferStrong: true,
				},
			},
			fastWindow: 0,
			medWindow:  0,
			slowWindow: slowWindow,
			want: archivePeerSecondaryPolicy{
				window:       slowWindow,
				preferStrong: true,
				label:        "slow-fallback",
				keyMode:      dedup.SecondaryKeyCQZone,
			},
		},
		{
			name: "disabled without stabilizer",
			cfg: &config.Config{
				CallCorrection: config.CallCorrectionConfig{Enabled: true, StabilizerEnabled: false},
			},
			fastWindow: fastWindow,
			medWindow:  medWindow,
			slowWindow: slowWindow,
			want:       archivePeerSecondaryPolicy{},
		},
		{
			name: "disabled when all windows are zero",
			cfg: &config.Config{
				CallCorrection: config.CallCorrectionConfig{Enabled: true, StabilizerEnabled: true},
			},
			want: archivePeerSecondaryPolicy{},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got := selectArchivePeerSecondaryPolicy(tc.cfg, tc.fastWindow, tc.medWindow, tc.slowWindow)
			if got != tc.want {
				t.Fatalf("selectArchivePeerSecondaryPolicy() = %+v, want %+v", got, tc.want)
			}
		})
	}
}
