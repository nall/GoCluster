package spot

import (
	"testing"
	"time"
)

func TestRecentBandStoreAdmissionAndExpiry(t *testing.T) {
	now := time.Date(2026, 2, 14, 12, 0, 0, 0, time.UTC)
	store := NewRecentBandStoreWithOptions(RecentBandOptions{
		Window:             time.Hour,
		Shards:             1,
		MaxEntries:         128,
		CleanupInterval:    time.Hour,
		MaxSpottersPerCall: 8,
	})

	store.Record("K1ABC", "40m", "CW", "W1AAA", now.Add(-30*time.Minute))
	store.Record("K1ABC", "40m", "CW", "W2BBB", now.Add(-20*time.Minute))

	if got := store.RecentSupportCount("K1ABC", "40m", "CW", now); got != 2 {
		t.Fatalf("expected 2 unique spotters, got %d", got)
	}
	if !store.HasRecentSupport("K1ABC", "40m", "CW", 2, now) {
		t.Fatalf("expected recent support admission at threshold=2")
	}
	if store.HasRecentSupport("K1ABC", "40m", "CW", 3, now) {
		t.Fatalf("expected no admission at threshold=3")
	}

	expiredAt := now.Add(2 * time.Hour)
	if store.HasRecentSupport("K1ABC", "40m", "CW", 2, expiredAt) {
		t.Fatalf("expected admission to expire outside window")
	}
	if got := store.RecentSupportCount("K1ABC", "40m", "CW", expiredAt); got != 0 {
		t.Fatalf("expected 0 unique spotters after expiry, got %d", got)
	}
}

func TestRecentBandStoreNormalizesKeys(t *testing.T) {
	now := time.Now().UTC()
	store := NewRecentBandStoreWithOptions(RecentBandOptions{
		Window:             12 * time.Hour,
		Shards:             1,
		MaxEntries:         128,
		CleanupInterval:    time.Hour,
		MaxSpottersPerCall: 8,
	})

	store.Record("k1abc", "40M", "cw", "w1aaa", now)
	store.Record("K1ABC", "40m", "CW", "W2BBB", now)

	if !store.HasRecentSupport("K1ABC", "40m", "CW", 2, now) {
		t.Fatalf("expected key normalization across case differences")
	}
}

func TestRecentBandStoreActiveCallCount(t *testing.T) {
	now := time.Date(2026, 2, 16, 12, 0, 0, 0, time.UTC)
	store := NewRecentBandStoreWithOptions(RecentBandOptions{
		Window:             time.Hour,
		Shards:             1,
		MaxEntries:         128,
		CleanupInterval:    time.Hour,
		MaxSpottersPerCall: 8,
	})

	store.Record("K1ABC", "40m", "CW", "W1AAA", now.Add(-10*time.Minute))
	store.Record("K1ABC", "20m", "CW", "W2BBB", now.Add(-5*time.Minute))
	store.Record("N0XYZ", "40m", "CW", "W3CCC", now.Add(-8*time.Minute))
	store.Record("OLD1", "40m", "CW", "W4DDD", now.Add(-2*time.Hour))
	store.cleanup(now)

	if got := store.ActiveCallCount(now); got != 2 {
		t.Fatalf("expected 2 active calls, got %d", got)
	}
	store.cleanup(now.Add(2 * time.Hour))
	if got := store.ActiveCallCount(now.Add(2 * time.Hour)); got != 0 {
		t.Fatalf("expected 0 active calls after expiry, got %d", got)
	}
}

func TestRecentBandStoreActiveCallCountsByBand(t *testing.T) {
	now := time.Date(2026, 2, 16, 12, 0, 0, 0, time.UTC)
	store := NewRecentBandStoreWithOptions(RecentBandOptions{
		Window:             time.Hour,
		Shards:             1,
		MaxEntries:         128,
		CleanupInterval:    time.Hour,
		MaxSpottersPerCall: 8,
	})

	// Same call appears in two modes on 40m and should only count once on 40m.
	store.Record("K1ABC", "40m", "CW", "W1AAA", now.Add(-10*time.Minute))
	store.Record("K1ABC", "40m", "RTTY", "W2BBB", now.Add(-8*time.Minute))
	store.Record("K1ABC", "20m", "CW", "W3CCC", now.Add(-7*time.Minute))
	store.Record("N0XYZ", "40m", "CW", "W4DDD", now.Add(-6*time.Minute))
	store.Record("OLD1", "15m", "CW", "W5EEE", now.Add(-2*time.Hour))
	store.cleanup(now)

	got := store.ActiveCallCountsByBand(now)
	if got["40m"] != 2 {
		t.Fatalf("expected 40m active-call count 2, got %d", got["40m"])
	}
	if got["20m"] != 1 {
		t.Fatalf("expected 20m active-call count 1, got %d", got["20m"])
	}
	if _, ok := got["15m"]; ok {
		t.Fatalf("did not expect expired 15m entry in counts")
	}
}

func TestRecentBandStoreRecordOverflowRetainsNewestDeterministically(t *testing.T) {
	now := time.Date(2026, 4, 11, 22, 0, 0, 0, time.UTC)
	store := NewRecentBandStoreWithOptions(RecentBandOptions{
		Window:             time.Hour,
		Shards:             1,
		MaxEntries:         128,
		CleanupInterval:    time.Hour,
		MaxSpottersPerCall: 2,
	})

	for _, spotter := range []string{"W1AAA", "W1BBB", "W1CCC"} {
		store.Record("K1ABC", "40m", "CW", spotter, now)
	}

	key, ok := store.normalizeKey("K1ABC", "40m", "CW")
	if !ok {
		t.Fatalf("expected normalized recent-band key")
	}
	entry := store.shards[0].entries[key]
	if entry == nil {
		t.Fatalf("expected recent-band entry after overflow record")
	}
	if len(entry.spotters) != 2 {
		t.Fatalf("expected 2 retained spotters, got %d", len(entry.spotters))
	}
	if _, ok := entry.spotters["W1AAA"]; ok {
		t.Fatalf("expected lexical tie-break to evict W1AAA")
	}
	if _, ok := entry.spotters["W1BBB"]; !ok {
		t.Fatalf("expected W1BBB retained after overflow trim")
	}
	if _, ok := entry.spotters["W1CCC"]; !ok {
		t.Fatalf("expected W1CCC retained after overflow trim")
	}
}

func TestRecentBandTrimSpottersLockedDoesNotRefreshLastSeen(t *testing.T) {
	store := NewRecentBandStoreWithOptions(RecentBandOptions{
		Window:             time.Hour,
		Shards:             1,
		MaxEntries:         128,
		CleanupInterval:    time.Hour,
		MaxSpottersPerCall: 1,
	})
	entry := &recentBandEntry{
		spotters: map[string]time.Time{
			"W1AAA": time.Unix(10, 0).UTC(),
			"W1BBB": time.Unix(20, 0).UTC(),
		},
		lastSeen: time.Unix(999, 0).UTC(),
	}

	store.trimSpottersLocked(entry)
	if got := entry.lastSeen.Unix(); got != 999 {
		t.Fatalf("expected normal overflow trim to leave lastSeen unchanged, got %d", got)
	}
}
