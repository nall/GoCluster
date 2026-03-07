package spot

import "testing"

func TestResolveIARURegionUsesContinentDefaultsAndOverrides(t *testing.T) {
	tests := []struct {
		name      string
		adif      int
		continent string
		want      IARURegion
	}{
		{name: "eu default", adif: 54, continent: "EU", want: IARURegion1},
		{name: "na default", adif: 291, continent: "NA", want: IARURegion2},
		{name: "asia default", adif: 324, continent: "AS", want: IARURegion3},
		{name: "asiatic russia override", adif: 15, continent: "AS", want: IARURegion1},
		{name: "chagos override", adif: 33, continent: "AF", want: IARURegion3},
		{name: "palestine override", adif: 510, continent: "AS", want: IARURegion1},
		{name: "unknown when metadata missing", adif: 0, continent: "", want: IARURegionUnknown},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := ResolveIARURegion(tc.adif, tc.continent); got != tc.want {
				t.Fatalf("ResolveIARURegion(%d, %q) = %q, want %q", tc.adif, tc.continent, got, tc.want)
			}
		})
	}
}
