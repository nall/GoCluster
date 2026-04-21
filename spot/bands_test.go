package spot

import "testing"

func TestFreqToBandVHFUHFAndDisjoint13cm(t *testing.T) {
	tests := []struct {
		name string
		freq float64
		want string
	}{
		{name: "6m", freq: 50313, want: "6m"},
		{name: "2m", freq: 144360, want: "2m"},
		{name: "70cm", freq: 432100, want: "70cm"},
		{name: "33cm", freq: 903000, want: "33cm"},
		{name: "23cm", freq: 1296192, want: "23cm"},
		{name: "13cm lower segment", freq: 2304000, want: "13cm"},
		{name: "13cm upper segment", freq: 2400040, want: "13cm"},
		{name: "13cm upper sample", freq: 2400919, want: "13cm"},
		{name: "13cm allocation gap", freq: 2350000, want: "???"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FreqToBand(tt.freq); got != tt.want {
				t.Fatalf("FreqToBand(%v) = %q, want %q", tt.freq, got, tt.want)
			}
		})
	}
}

func TestSupportedBandNamesRemainUnique(t *testing.T) {
	seen := make(map[string]struct{})
	for _, name := range SupportedBandNames() {
		if _, exists := seen[name]; exists {
			t.Fatalf("duplicate supported band name %q", name)
		}
		seen[name] = struct{}{}
	}
	if _, ok := seen["13cm"]; !ok {
		t.Fatal("expected 13cm to remain a supported band")
	}
}

func TestFrequencyBoundsIncludeDisjoint13cmSegment(t *testing.T) {
	min, max := FrequencyBounds()
	if min != 135.7 {
		t.Fatalf("min = %v, want 135.7", min)
	}
	if max != 2450000 {
		t.Fatalf("max = %v, want 2450000", max)
	}
}
