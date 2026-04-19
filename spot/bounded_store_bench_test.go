package spot

import (
	"fmt"
	"testing"
	"time"
)

func BenchmarkCustomSCPRecordOverflow(b *testing.B) {
	const maxSpotters = 64

	now := time.Now().UTC()
	oldestSpotter := "N000"
	newSpotter := "NZZZ"
	oldestObs := customSCPSpotterObs{
		seenUnix: now.Add(-time.Duration(maxSpotters) * time.Second).Unix(),
		cellRes1: 100,
	}
	entry := &customSCPEntry{
		spotters: make(map[string]customSCPSpotterObs, maxSpotters+1),
		lastSeen: now.Add(-time.Second).Unix(),
	}
	for i := 0; i < maxSpotters; i++ {
		spotter := fmt.Sprintf("N%03d", i)
		entry.spotters[spotter] = customSCPSpotterObs{
			seenUnix: now.Add(time.Duration(i-maxSpotters) * time.Second).Unix(),
			cellRes1: uint16(100 + i),
		}
	}

	store := &CustomSCPStore{
		opts:                sanitizeCustomSCPOptions(CustomSCPOptions{MaxSpottersPerKey: maxSpotters}),
		entries:             make(map[customSCPKey]*customSCPEntry, 1),
		entryExpiryItems:    make(map[customSCPKey]*customSCPEntryExpiryItem, 1),
		static:              make(map[string]int64, 1),
		observationSpotters: maxSpotters,
	}
	store.mu.Lock()
	retainCustomSCPTestStaticLocked(store, "K1BENCH", now.Unix())
	retainCustomSCPTestEntryLocked(store, customSCPKey{call: "K1BENCH", band: "40m", bucket: "cw"}, entry)
	store.mu.Unlock()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		store.recordObservation("K1BENCH", "40m", "CW", newSpotter, 999, 0, false, now)
		b.StopTimer()
		store.mu.Lock()
		delete(entry.spotters, newSpotter)
		store.releaseInternedStringLocked(newSpotter)
		entry.spotters[store.retainInternedStringLocked(oldestSpotter)] = oldestObs
		entry.lastSeen = now.Add(-time.Second).Unix()
		store.mu.Unlock()
		b.StartTimer()
	}
}

func BenchmarkRecentBandRecordOverflow(b *testing.B) {
	const maxSpotters = 64

	now := time.Now().UTC()
	store := NewRecentBandStoreWithOptions(RecentBandOptions{
		Window:             12 * time.Hour,
		Shards:             1,
		MaxEntries:         128,
		CleanupInterval:    time.Hour,
		MaxSpottersPerCall: maxSpotters,
	})
	key, ok := store.normalizeKey("K1BENCH", "40m", "CW")
	if !ok {
		b.Fatalf("normalize recent-band key")
	}
	entry := &recentBandEntry{
		spotters: make(map[string]time.Time, maxSpotters+1),
		lastSeen: now.Add(-time.Second),
	}
	oldestSpotter := "W000"
	oldestSeen := now.Add(-time.Duration(maxSpotters) * time.Second)
	for i := 0; i < maxSpotters; i++ {
		spotter := fmt.Sprintf("W%03d", i)
		entry.spotters[spotter] = now.Add(time.Duration(i-maxSpotters) * time.Second)
	}
	store.shards[0].entries = map[recentBandKey]*recentBandEntry{key: entry}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		store.Record("K1BENCH", "40m", "CW", "WZZZ", now)
		b.StopTimer()
		delete(entry.spotters, "WZZZ")
		entry.spotters[oldestSpotter] = oldestSeen
		entry.lastSeen = now.Add(-time.Second)
		b.StartTimer()
	}
}

func BenchmarkCustomSCPLoadOversizedKey(b *testing.B) {
	const maxSpotters = 64

	now := time.Now().UTC()
	store := &CustomSCPStore{
		opts: sanitizeCustomSCPOptions(CustomSCPOptions{MaxSpottersPerKey: maxSpotters}),
	}
	key := customSCPKey{call: "K1LOAD", band: "40m", bucket: "cw"}
	entry := &customSCPEntry{
		spotters: make(map[string]customSCPSpotterObs, maxSpotters+1),
		lastSeen: now.Unix(),
	}
	oldestSpotter := "L000"
	oldestObs := customSCPSpotterObs{
		seenUnix: now.Add(-time.Duration(maxSpotters) * time.Second).Unix(),
		cellRes1: 200,
	}
	for i := 0; i <= maxSpotters; i++ {
		spotter := fmt.Sprintf("L%03d", i)
		entry.spotters[spotter] = customSCPSpotterObs{
			seenUnix: now.Add(time.Duration(i-maxSpotters) * time.Second).Unix(),
			cellRes1: uint16(200 + i),
		}
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pendingOverflow := false
		store.trimPendingEntryOnLoadLocked(key, entry, &pendingOverflow, nil)
		b.StopTimer()
		entry.spotters[oldestSpotter] = oldestObs
		b.StartTimer()
	}
}

func BenchmarkCustomSCPCleanupFewExpired(b *testing.B) {
	benchmarkCustomSCPCleanup(b, 1024, 4)
}

func BenchmarkCustomSCPCleanupManyExpired(b *testing.B) {
	benchmarkCustomSCPCleanup(b, 1024, 1024)
}

func benchmarkCustomSCPCleanup(b *testing.B, totalEntries, expiredEntries int) {
	now := time.Now().UTC()
	opts := sanitizeCustomSCPOptions(CustomSCPOptions{
		HorizonDays:       30,
		StaticHorizonDays: 30,
		MaxSpottersPerKey: 64,
	})

	buildStore := func() *CustomSCPStore {
		store := &CustomSCPStore{
			opts:              opts,
			entries:           make(map[customSCPKey]*customSCPEntry, totalEntries),
			entryExpiryItems:  make(map[customSCPKey]*customSCPEntryExpiryItem, totalEntries),
			static:            make(map[string]int64),
			staticExpiryItems: make(map[string]*customSCPStaticExpiryItem),
		}
		for i := 0; i < totalEntries; i++ {
			seenAt := now.Add(-1 * time.Hour)
			if i < expiredEntries {
				seenAt = now.Add(-31 * 24 * time.Hour)
			}
			key := customSCPKey{
				call:   fmt.Sprintf("K%04d", i),
				band:   "40m",
				bucket: "cw",
			}
			entry := &customSCPEntry{
				spotters: map[string]customSCPSpotterObs{
					"N0AAA": {seenUnix: seenAt.Unix(), cellRes1: 101},
				},
			}
			key = retainCustomSCPTestEntryLocked(store, key, entry)
			store.observationSpotters++
			store.markEntryForCleanupLocked(key, entry)
		}
		return store
	}

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		store := buildStore()
		b.StartTimer()
		store.cleanup(now)
	}
}

func BenchmarkCustomSCPInternerChurn(b *testing.B) {
	values := make([]string, 4096)
	for i := range values {
		values[i] = fmt.Sprintf("S%04d", i)
	}
	var interner customSCPInterner

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		value := interner.retain(values[i&(len(values)-1)])
		interner.release(value)
	}
	if interner.totalRefs != 0 || len(interner.refs) != 0 || interner.releaseMisses != 0 {
		b.Fatalf("interner did not fully release: refs=%d distinct=%d misses=%d", interner.totalRefs, len(interner.refs), interner.releaseMisses)
	}
}
