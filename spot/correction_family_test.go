package spot

import "testing"

var (
	benchmarkCorrectionFamilyKeysSink []string
	benchmarkCorrectionFamilyVoteKey  string
	benchmarkCorrectionFamilyBaseKey  string
	benchmarkCorrectionFamilyOK       bool
)

func TestCorrectionFamilyKeyPair(t *testing.T) {
	cases := []struct {
		name    string
		call    string
		voteKey string
		baseKey string
		ok      bool
	}{
		{
			name: "empty",
			call: "",
			ok:   false,
		},
		{
			name:    "simple",
			call:    " w1aw ",
			voteKey: "W1AW",
			ok:      true,
		},
		{
			name:    "slash prefix",
			call:    "kh6/w1aw",
			voteKey: "W1AW/KH6",
			baseKey: "W1AW",
			ok:      true,
		},
		{
			name:    "slash suffix",
			call:    "w1aw/kh6",
			voteKey: "W1AW/KH6",
			baseKey: "W1AW",
			ok:      true,
		},
		{
			name:    "single non-empty slash part",
			call:    "/w1aw/",
			voteKey: "W1AW",
			ok:      true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			voteKey, baseKey, ok := CorrectionFamilyKeyPair(tc.call)
			if ok != tc.ok {
				t.Fatalf("expected ok=%v, got %v", tc.ok, ok)
			}
			if voteKey != tc.voteKey || baseKey != tc.baseKey {
				t.Fatalf("expected vote/base %q/%q, got %q/%q", tc.voteKey, tc.baseKey, voteKey, baseKey)
			}
			keys := CorrectionFamilyKeys(tc.call)
			switch {
			case !tc.ok && len(keys) != 0:
				t.Fatalf("expected no family keys, got %v", keys)
			case tc.ok && tc.baseKey == "" && (len(keys) != 1 || keys[0] != tc.voteKey):
				t.Fatalf("expected one family key %q, got %v", tc.voteKey, keys)
			case tc.ok && tc.baseKey != "" && (len(keys) != 2 || keys[0] != tc.voteKey || keys[1] != tc.baseKey):
				t.Fatalf("expected family keys %q/%q, got %v", tc.voteKey, tc.baseKey, keys)
			}
		})
	}
}

func BenchmarkCorrectionFamilyKeysSimple(b *testing.B) {
	for i := 0; i < b.N; i++ {
		benchmarkCorrectionFamilyKeysSink = CorrectionFamilyKeys("K1ABC")
	}
}

func BenchmarkCorrectionFamilyKeyPairSimple(b *testing.B) {
	for i := 0; i < b.N; i++ {
		benchmarkCorrectionFamilyVoteKey, benchmarkCorrectionFamilyBaseKey, benchmarkCorrectionFamilyOK = CorrectionFamilyKeyPair("K1ABC")
	}
}

func BenchmarkCorrectionFamilyKeysSlash(b *testing.B) {
	for i := 0; i < b.N; i++ {
		benchmarkCorrectionFamilyKeysSink = CorrectionFamilyKeys("KH6/W1AW")
	}
}

func BenchmarkCorrectionFamilyKeyPairSlash(b *testing.B) {
	for i := 0; i < b.N; i++ {
		benchmarkCorrectionFamilyVoteKey, benchmarkCorrectionFamilyBaseKey, benchmarkCorrectionFamilyOK = CorrectionFamilyKeyPair("KH6/W1AW")
	}
}
