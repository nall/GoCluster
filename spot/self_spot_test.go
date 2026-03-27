package spot

import "testing"

func TestIsLocalSelfSpot(t *testing.T) {
	tests := []struct {
		name string
		spot *Spot
		want bool
	}{
		{
			name: "manual self spot",
			spot: NewSpot("K1SELF", "K1SELF", 7010.0, "CW"),
			want: true,
		},
		{
			name: "manual non self spot",
			spot: NewSpot("K1DX", "K1SELF", 7010.0, "CW"),
			want: false,
		},
		{
			name: "peer self looking blocked",
			spot: func() *Spot {
				s := NewSpot("K1SELF", "K1SELF", 7010.0, "CW")
				s.SourceType = SourcePeer
				return s
			}(),
			want: false,
		},
		{
			name: "test spotter blocked",
			spot: func() *Spot {
				s := NewSpot("K1SELF", "K1SELF", 7010.0, "CW")
				s.IsTestSpotter = true
				return s
			}(),
			want: false,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			if got := IsLocalSelfSpot(tc.spot); got != tc.want {
				t.Fatalf("IsLocalSelfSpot()=%v want %v", got, tc.want)
			}
		})
	}
}
