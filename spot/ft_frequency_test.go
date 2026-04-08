package spot

import "testing"

func TestFTDialRegistryCanonicalizeNearestDial(t *testing.T) {
	registry := NewFTDialRegistry([]ModeSeed{
		{FrequencyKHz: 14074, Mode: "FT8"},
		{FrequencyKHz: 14090, Mode: "FT8"},
		{FrequencyKHz: 14080, Mode: "FT4"},
		{FrequencyKHz: 14074, Mode: "FT8"},
	})
	if registry == nil {
		t.Fatal("expected registry")
	}

	canonical, ok := registry.Canonicalize("FT8", 14076.11)
	if !ok {
		t.Fatal("expected FT8 frequency to canonicalize")
	}
	if canonical != 14074.0 {
		t.Fatalf("expected 14074.0, got %.2f", canonical)
	}

	canonical, ok = registry.Canonicalize("FT8", 14090.08)
	if !ok {
		t.Fatal("expected secondary FT8 dial to canonicalize")
	}
	if canonical != 14090.0 {
		t.Fatalf("expected 14090.0, got %.2f", canonical)
	}

	canonical, ok = registry.Canonicalize("FT4", 14082.25)
	if !ok {
		t.Fatal("expected FT4 frequency to canonicalize")
	}
	if canonical != 14080.0 {
		t.Fatalf("expected 14080.0, got %.2f", canonical)
	}
}

func TestFTDialRegistryCanonicalizeFailsOpenWhenOutOfWindow(t *testing.T) {
	registry := NewFTDialRegistry([]ModeSeed{
		{FrequencyKHz: 14074, Mode: "FT8"},
	})
	if registry == nil {
		t.Fatal("expected registry")
	}

	if canonical, ok := registry.Canonicalize("FT8", 14079.0); ok || canonical != 0 {
		t.Fatalf("expected out-of-window FT8 frequency to fail open, got %.2f ok=%v", canonical, ok)
	}
	if canonical, ok := registry.Canonicalize("FT4", 14080.0); ok || canonical != 0 {
		t.Fatalf("expected missing FT4 seed to fail open, got %.2f ok=%v", canonical, ok)
	}
}

func TestFTDialRegistryCanonicalizeAllowsSmallNegativeSlack(t *testing.T) {
	registry := NewFTDialRegistry([]ModeSeed{
		{FrequencyKHz: 14074, Mode: "FT8"},
	})
	if registry == nil {
		t.Fatal("expected registry")
	}

	canonical, ok := registry.Canonicalize("FT8", 14073.95)
	if !ok {
		t.Fatal("expected slightly-low FT8 frequency to canonicalize")
	}
	if canonical != 14074.0 {
		t.Fatalf("expected 14074.0, got %.2f", canonical)
	}
}

func TestSpotEffectiveObservedFrequencyFallsBackToOperational(t *testing.T) {
	s := NewSpot("K1ABC", "W1XYZ", 14074.0, "FT8")
	if got := s.EffectiveObservedFrequency(); got != 14074.0 {
		t.Fatalf("expected operational frequency fallback, got %.2f", got)
	}
	s.ObservedFrequency = 14076.11
	if got := s.EffectiveObservedFrequency(); got != 14076.11 {
		t.Fatalf("expected observed frequency, got %.2f", got)
	}
}

func BenchmarkFTDialRegistryCanonicalize(b *testing.B) {
	registry := NewFTDialRegistry([]ModeSeed{
		{FrequencyKHz: 14074, Mode: "FT8"},
		{FrequencyKHz: 14090, Mode: "FT8"},
		{FrequencyKHz: 14080, Mode: "FT4"},
	})
	if registry == nil {
		b.Fatal("expected registry")
	}

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, ok := registry.Canonicalize("FT8", 14076.11); !ok {
			b.Fatal("expected canonicalization")
		}
	}
}
