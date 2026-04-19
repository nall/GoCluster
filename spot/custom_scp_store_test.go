package spot

import (
	"encoding/binary"
	"path/filepath"
	"testing"
	"time"
)

func assertCustomSCPInternerInvariant(t *testing.T, store *CustomSCPStore) {
	t.Helper()
	if store == nil {
		return
	}
	store.mu.RLock()
	defer store.mu.RUnlock()

	expected := make(map[string]int)
	for call := range store.static {
		expected[call]++
	}
	for key, entry := range store.entries {
		expected[key.call]++
		expected[key.band]++
		expected[key.bucket]++
		for spotter := range entry.spotters {
			expected[spotter]++
		}
	}
	expectedRefs := 0
	for _, refs := range expected {
		expectedRefs += refs
	}
	if store.interner.releaseMisses != 0 {
		t.Fatalf("expected no interner release misses, got %d", store.interner.releaseMisses)
	}
	if store.interner.totalRefs != expectedRefs {
		t.Fatalf("expected %d interner refs, got %d", expectedRefs, store.interner.totalRefs)
	}
	if len(store.interner.refs) != len(expected) {
		t.Fatalf("expected %d interned strings, got %d", len(expected), len(store.interner.refs))
	}
	for value, want := range expected {
		ref, ok := store.interner.refs[value]
		if !ok {
			t.Fatalf("missing interner ref for %q", value)
		}
		if ref.value != value {
			t.Fatalf("interner canonical value mismatch for %q: %q", value, ref.value)
		}
		if ref.refs != want {
			t.Fatalf("expected %d refs for %q, got %d", want, value, ref.refs)
		}
	}
	for value := range store.interner.refs {
		if _, ok := expected[value]; !ok {
			t.Fatalf("unexpected interner ref for %q", value)
		}
	}
}

func retainCustomSCPTestEntryLocked(store *CustomSCPStore, key customSCPKey, entry *customSCPEntry) customSCPKey {
	retainedKey := customSCPKey{
		call:   store.retainInternedStringLocked(key.call),
		band:   store.retainInternedStringLocked(key.band),
		bucket: store.retainInternedStringLocked(key.bucket),
	}
	retainedSpotters := make(map[string]customSCPSpotterObs, len(entry.spotters))
	for spotter, obs := range entry.spotters {
		retainedSpotters[store.retainInternedStringLocked(spotter)] = obs
	}
	entry.spotters = retainedSpotters
	store.entries[retainedKey] = entry
	return retainedKey
}

func retainCustomSCPTestStaticLocked(store *CustomSCPStore, call string, seenUnix int64) string {
	retained := store.retainStaticCallLocked(call, seenUnix)
	store.upsertStaticExpiryLocked(retained, store.static[retained])
	return retained
}

func TestCustomSCPStoreCWAndRTTYSNRThresholds(t *testing.T) {
	opts := CustomSCPOptions{
		Path:           filepath.Join(t.TempDir(), "scp"),
		CoreMinScore:   1,
		CoreMinH3Cells: 1,
		MinSNRDBCW:     4,
		MinSNRDBRTTY:   3,
	}
	store, err := OpenCustomSCPStore(opts)
	if err != nil {
		t.Fatalf("open custom store: %v", err)
	}
	t.Cleanup(func() {
		_ = store.Close()
	})

	now := time.Now().UTC()
	callCW := "K1CW"
	callRTTY := "K1RY"
	band := "40m"

	// Below threshold: CW report=3 should be rejected when min_snr_db_cw=4.
	store.recordObservation(callCW, band, "CW", "N0AAA", 101, 3, true, now)
	if got := store.RecentSupportCount(callCW, band, "CW", now); got != 0 {
		t.Fatalf("expected no CW support below threshold, got %d", got)
	}

	// At threshold: CW report=4 should be accepted.
	store.recordObservation(callCW, band, "CW", "N0BBB", 102, 4, true, now)
	if got := store.RecentSupportCount(callCW, band, "CW", now); got != 1 {
		t.Fatalf("expected one CW support at threshold, got %d", got)
	}

	// Missing report should be rejected when the RTTY SNR gate is enabled.
	store.recordObservation(callRTTY, band, "RTTY", "N0CCC", 201, 0, false, now)
	if got := store.RecentSupportCount(callRTTY, band, "RTTY", now); got != 0 {
		t.Fatalf("expected no RTTY support without report when gate enabled, got %d", got)
	}

	// At threshold: RTTY report=3 should be accepted.
	store.recordObservation(callRTTY, band, "RTTY", "N0DDD", 202, 3, true, now)
	if got := store.RecentSupportCount(callRTTY, band, "RTTY", now); got != 1 {
		t.Fatalf("expected one RTTY support at threshold, got %d", got)
	}
}

func TestCustomSCPStoreVoiceBucketSharesUSBAndLSB(t *testing.T) {
	store, err := OpenCustomSCPStore(CustomSCPOptions{
		Path:           filepath.Join(t.TempDir(), "scp"),
		CoreMinScore:   1,
		CoreMinH3Cells: 1,
	})
	if err != nil {
		t.Fatalf("open custom store: %v", err)
	}
	t.Cleanup(func() {
		_ = store.Close()
	})

	now := time.Now().UTC()
	call := "K1SSB"
	band := "40m"

	store.recordObservation(call, band, "USB", "N0AAA", 101, 0, false, now)

	if got := store.RecentSupportCount(call, band, "LSB", now); got != 1 {
		t.Fatalf("expected USB evidence to be visible via LSB voice bucket, got %d", got)
	}
	if got := store.RecentSupportCount(call, band, "CW", now); got != 0 {
		t.Fatalf("expected CW bucket to stay isolated from voice, got %d", got)
	}
}

func TestCustomSCPStoreH3CellDiversityGate(t *testing.T) {
	store, err := OpenCustomSCPStore(CustomSCPOptions{
		Path:           filepath.Join(t.TempDir(), "scp"),
		CoreMinScore:   1,
		CoreMinH3Cells: 2,
	})
	if err != nil {
		t.Fatalf("open custom store: %v", err)
	}
	t.Cleanup(func() {
		_ = store.Close()
	})

	now := time.Now().UTC()
	call := "K1DIVERSE"
	band := "40m"

	// Two unique spotters in the same coarse H3 cell are not diverse enough.
	store.recordObservation(call, band, "CW", "N0AAA", 101, 10, true, now)
	store.recordObservation(call, band, "CW", "N0BBB", 101, 10, true, now)
	if store.HasRecentSupport(call, band, "CW", 2, now) {
		t.Fatalf("expected H3 diversity gate to reject same-cell support")
	}

	// Add a third spotter in a distant grid to satisfy coarse-cell diversity.
	store.recordObservation(call, band, "CW", "N0CCC", 202, 10, true, now)
	if !store.HasRecentSupport(call, band, "CW", 2, now) {
		t.Fatalf("expected H3 diversity gate to pass after multi-cell evidence")
	}
}

func TestCustomSCPStoreRecordSpotAdmitsVOnly(t *testing.T) {
	store, err := OpenCustomSCPStore(CustomSCPOptions{
		Path:           filepath.Join(t.TempDir(), "scp"),
		CoreMinScore:   1,
		CoreMinH3Cells: 1,
	})
	if err != nil {
		t.Fatalf("open custom store: %v", err)
	}
	t.Cleanup(func() {
		_ = store.Close()
	})

	now := time.Now().UTC()
	makeSpot := func(spotter, confidence string) *Spot {
		return &Spot{
			DXCall:     "K1ABC",
			DECall:     spotter,
			DEGridNorm: "FN20",
			Frequency:  7020,
			Band:       "40m",
			Mode:       "CW",
			HasReport:  true,
			Report:     12,
			Time:       now,
			Confidence: confidence,
		}
	}

	store.RecordSpot(makeSpot("N0AAA", "S"))
	if store.StaticContains("K1ABC") {
		t.Fatalf("expected S confidence to be rejected from static membership")
	}

	store.RecordSpot(makeSpot("N0BBB", "P"))
	if store.StaticContains("K1ABC") {
		t.Fatalf("expected P confidence to be rejected from static membership")
	}

	store.RecordSpot(makeSpot("N0CCC", "C"))
	if store.StaticContains("K1ABC") {
		t.Fatalf("expected C confidence to be rejected from static membership")
	}

	store.RecordSpot(makeSpot("N0DDD", "V"))
	if !store.StaticContains("K1ABC") {
		t.Fatalf("expected V-admitted call to enter static membership")
	}
	if got := store.ActiveCallCount(now); got != 1 {
		t.Fatalf("expected exactly one active call after V admission, got %d", got)
	}
}

func TestCustomSCPStoreRecordSpotAdmitsFT8VOnly(t *testing.T) {
	store, err := OpenCustomSCPStore(CustomSCPOptions{
		Path:           filepath.Join(t.TempDir(), "scp"),
		CoreMinScore:   1,
		CoreMinH3Cells: 1,
	})
	if err != nil {
		t.Fatalf("open custom store: %v", err)
	}
	t.Cleanup(func() {
		_ = store.Close()
	})

	now := time.Now().UTC()
	makeSpot := func(spotter, confidence string) *Spot {
		return &Spot{
			DXCall:     "K1FT8",
			DECall:     spotter,
			DEGridNorm: "FN31",
			Frequency:  14074.1,
			Band:       "20m",
			Mode:       "FT8",
			HasReport:  true,
			Report:     -10,
			Time:       now,
			Confidence: confidence,
		}
	}

	store.RecordSpot(makeSpot("N0AAA", "P"))
	if store.StaticContains("K1FT8") {
		t.Fatalf("expected non-V FT8 confidence to be rejected from static membership")
	}

	store.RecordSpot(makeSpot("N0BBB", "V"))
	if !store.StaticContains("K1FT8") {
		t.Fatalf("expected V-admitted FT8 call to enter static membership")
	}
	if snap := store.snapshotFor("K1FT8", "20m", "FT8", now); snap.uniqueSpotters != 1 {
		t.Fatalf("expected FT8 snapshot to track exact FT mode, got %+v", snap)
	}
	if snap := store.snapshotFor("K1FT8", "20m", "FT4", now); snap.uniqueSpotters != 0 {
		t.Fatalf("expected FT4 snapshot to remain isolated from FT8 evidence, got %+v", snap)
	}
}

func TestCustomSCPStoreLoadPrunesLegacyOversizedKey(t *testing.T) {
	path := filepath.Join(t.TempDir(), "scp")
	opts := CustomSCPOptions{
		Path:                   path,
		HorizonDays:            30,
		StaticHorizonDays:      30,
		MaxSpottersPerKey:      2,
		CoreMinScore:           1,
		CoreMinH3Cells:         1,
		SFloorMinScore:         1,
		SFloorExactMinH3Cells:  1,
		SFloorFamilyMinH3Cells: 1,
	}

	store, err := OpenCustomSCPStore(opts)
	if err != nil {
		t.Fatalf("open custom store: %v", err)
	}
	key := customSCPKey{call: "K1ABC", band: "40m", bucket: "cw"}
	now := time.Now().UTC()
	spotters := []struct {
		call string
		seen int64
		cell uint16
	}{
		{call: "N0AAA", seen: now.Add(-4 * time.Minute).Unix(), cell: 101},
		{call: "N0BBB", seen: now.Add(-3 * time.Minute).Unix(), cell: 102},
		{call: "N0CCC", seen: now.Add(-2 * time.Minute).Unix(), cell: 103},
		{call: "N0DDD", seen: now.Add(-1 * time.Minute).Unix(), cell: 104},
	}
	for _, entry := range spotters {
		setRawObservation(t, store, key, entry.call, entry.seen, entry.cell)
	}
	if err := store.Close(); err != nil {
		t.Fatalf("close custom store: %v", err)
	}

	reopened, err := OpenCustomSCPStore(opts)
	if err != nil {
		t.Fatalf("reopen custom store: %v", err)
	}
	t.Cleanup(func() {
		_ = reopened.Close()
	})

	stats := reopened.StatsSnapshot()
	if stats.ObservationSpotters != 2 {
		t.Fatalf("expected 2 retained spotters after load trim, got %+v", stats)
	}
	if stats.OversizedKeysSeenOnLoad != 1 {
		t.Fatalf("expected one oversized key on load, got %+v", stats)
	}
	if stats.OverflowObservationsPruned != 2 {
		t.Fatalf("expected two overflow deletions on load, got %+v", stats)
	}

	entry := reopened.entries[key]
	if entry == nil {
		t.Fatalf("expected loaded entry for %+v", key)
	}
	if len(entry.spotters) != 2 {
		t.Fatalf("expected 2 retained spotters in memory, got %d", len(entry.spotters))
	}
	if _, ok := entry.spotters["N0CCC"]; !ok {
		t.Fatalf("expected newest retained spotter N0CCC")
	}
	if _, ok := entry.spotters["N0DDD"]; !ok {
		t.Fatalf("expected newest retained spotter N0DDD")
	}
	if got := countObservationRecords(t, reopened, key); got != 2 {
		t.Fatalf("expected 2 persisted observation records after load trim, got %d", got)
	}
	assertCustomSCPInternerInvariant(t, reopened)
}

func TestCustomSCPStoreRecordObservationOverflowRetainsNewestDeterministically(t *testing.T) {
	opts := CustomSCPOptions{
		Path:                   filepath.Join(t.TempDir(), "scp"),
		HorizonDays:            30,
		StaticHorizonDays:      30,
		MaxSpottersPerKey:      2,
		CoreMinScore:           1,
		CoreMinH3Cells:         1,
		SFloorMinScore:         1,
		SFloorExactMinH3Cells:  1,
		SFloorFamilyMinH3Cells: 1,
	}
	store, err := OpenCustomSCPStore(opts)
	if err != nil {
		t.Fatalf("open custom store: %v", err)
	}
	t.Cleanup(func() {
		_ = store.Close()
	})

	seenAt := time.Date(2026, 4, 11, 22, 0, 0, 0, time.UTC)
	for _, spotter := range []string{"N0AAA", "N0BBB", "N0CCC"} {
		store.recordObservation("K1ABC", "40m", "CW", spotter, 101, 0, false, seenAt)
	}

	key := customSCPKey{call: "K1ABC", band: "40m", bucket: "cw"}
	entry := store.entries[key]
	if entry == nil {
		t.Fatalf("expected retained custom SCP entry")
	}
	if len(entry.spotters) != 2 {
		t.Fatalf("expected 2 retained spotters, got %d", len(entry.spotters))
	}
	if _, ok := entry.spotters["N0AAA"]; ok {
		t.Fatalf("expected lexical tie-break to evict N0AAA")
	}
	if _, ok := entry.spotters["N0BBB"]; !ok {
		t.Fatalf("expected N0BBB retained after overflow trim")
	}
	if _, ok := entry.spotters["N0CCC"]; !ok {
		t.Fatalf("expected N0CCC retained after overflow trim")
	}
	assertCustomSCPInternerInvariant(t, store)
}

func TestCustomSCPInternerReleasesRecordPrunedSpotters(t *testing.T) {
	opts := CustomSCPOptions{
		Path:                   filepath.Join(t.TempDir(), "scp"),
		HorizonDays:            1,
		StaticHorizonDays:      30,
		MaxSpottersPerKey:      4,
		CoreMinScore:           1,
		CoreMinH3Cells:         1,
		SFloorMinScore:         1,
		SFloorExactMinH3Cells:  1,
		SFloorFamilyMinH3Cells: 1,
	}
	store, err := OpenCustomSCPStore(opts)
	if err != nil {
		t.Fatalf("open custom store: %v", err)
	}
	t.Cleanup(func() {
		_ = store.Close()
	})

	now := time.Now().UTC()
	key := customSCPKey{call: "K1PRUNE", band: "40m", bucket: "cw"}
	staleEntry := &customSCPEntry{
		spotters: map[string]customSCPSpotterObs{
			"N0OLD": {seenUnix: now.Add(-48 * time.Hour).Unix(), cellRes1: 101},
		},
		lastSeen: now.Add(-48 * time.Hour).Unix(),
	}
	store.mu.Lock()
	key = retainCustomSCPTestEntryLocked(store, key, staleEntry)
	store.observationSpotters = 1
	store.markEntryForCleanupLocked(key, staleEntry)
	store.mu.Unlock()

	store.recordObservation("K1PRUNE", "40m", "CW", "N0NEW", 102, 0, false, now)

	entry := store.entries[key]
	if entry == nil {
		t.Fatalf("expected entry to survive after fresh observation")
	}
	if _, ok := entry.spotters["N0OLD"]; ok {
		t.Fatalf("expected stale spotter to be pruned")
	}
	if _, ok := store.interner.refs["N0OLD"]; ok {
		t.Fatalf("expected stale spotter interner ref to be released")
	}
	assertCustomSCPInternerInvariant(t, store)
}

func TestCustomSCPInternerReleasesSnapshotPrunedEntry(t *testing.T) {
	opts := CustomSCPOptions{
		Path:                   filepath.Join(t.TempDir(), "scp"),
		HorizonDays:            1,
		StaticHorizonDays:      30,
		MaxSpottersPerKey:      4,
		CoreMinScore:           1,
		CoreMinH3Cells:         1,
		SFloorMinScore:         1,
		SFloorExactMinH3Cells:  1,
		SFloorFamilyMinH3Cells: 1,
	}
	store, err := OpenCustomSCPStore(opts)
	if err != nil {
		t.Fatalf("open custom store: %v", err)
	}
	t.Cleanup(func() {
		_ = store.Close()
	})

	now := time.Date(2026, 4, 12, 12, 0, 0, 0, time.UTC)
	key := customSCPKey{call: "K1SNAP", band: "40m", bucket: "cw"}
	entry := &customSCPEntry{
		spotters: map[string]customSCPSpotterObs{
			"N0OLD": {seenUnix: now.Add(-48 * time.Hour).Unix(), cellRes1: 101},
		},
		lastSeen: now.Add(-48 * time.Hour).Unix(),
	}
	store.mu.Lock()
	key = retainCustomSCPTestEntryLocked(store, key, entry)
	store.observationSpotters = 1
	store.markEntryForCleanupLocked(key, entry)
	store.mu.Unlock()

	if snap := store.snapshotFor("K1SNAP", "40m", "CW", now); snap.uniqueSpotters != 0 {
		t.Fatalf("expected stale snapshot to return no support, got %+v", snap)
	}
	if _, ok := store.entries[key]; ok {
		t.Fatalf("expected stale entry to be deleted by snapshot pruning")
	}
	assertCustomSCPInternerInvariant(t, store)
}

func TestCustomSCPInternerReleasesMaxKeyEviction(t *testing.T) {
	opts := CustomSCPOptions{
		Path:                   filepath.Join(t.TempDir(), "scp"),
		HorizonDays:            30,
		StaticHorizonDays:      30,
		MaxKeys:                1,
		MaxSpottersPerKey:      4,
		CoreMinScore:           1,
		CoreMinH3Cells:         1,
		SFloorMinScore:         1,
		SFloorExactMinH3Cells:  1,
		SFloorFamilyMinH3Cells: 1,
	}
	store, err := OpenCustomSCPStore(opts)
	if err != nil {
		t.Fatalf("open custom store: %v", err)
	}
	t.Cleanup(func() {
		_ = store.Close()
	})

	now := time.Date(2026, 4, 12, 12, 0, 0, 0, time.UTC)
	store.recordObservation("K1OLD", "40m", "CW", "N0OLD", 101, 0, false, now.Add(-time.Minute))
	store.recordObservation("K1NEW", "20m", "RTTY", "N0NEW", 202, 3, true, now)

	if len(store.entries) != 1 {
		t.Fatalf("expected one retained entry after max-key eviction, got %d", len(store.entries))
	}
	if _, ok := store.interner.refs["N0OLD"]; ok {
		t.Fatalf("expected evicted entry spotter ref to be released")
	}
	if _, ok := store.interner.refs["40m"]; ok {
		t.Fatalf("expected evicted entry band ref to be released")
	}
	if _, ok := store.interner.refs["cw"]; ok {
		t.Fatalf("expected evicted entry bucket ref to be released")
	}
	assertCustomSCPInternerInvariant(t, store)
}

func TestCustomSCPStatsSnapshotReportsRetainedStateCardinality(t *testing.T) {
	opts := CustomSCPOptions{
		Path:                   filepath.Join(t.TempDir(), "scp"),
		HorizonDays:            30,
		StaticHorizonDays:      30,
		MaxSpottersPerKey:      4,
		CoreMinScore:           1,
		CoreMinH3Cells:         1,
		SFloorMinScore:         1,
		SFloorExactMinH3Cells:  1,
		SFloorFamilyMinH3Cells: 1,
	}
	store, err := OpenCustomSCPStore(opts)
	if err != nil {
		t.Fatalf("open custom store: %v", err)
	}
	t.Cleanup(func() {
		_ = store.Close()
	})

	now := time.Date(2026, 4, 12, 12, 0, 0, 0, time.UTC)
	store.recordObservation("K1STAT", "40m", "CW", "N0AAA", 101, 0, false, now)

	stats := store.StatsSnapshot()
	if stats.InternedStrings != 4 {
		t.Fatalf("expected 4 distinct interned strings, got %+v", stats)
	}
	if stats.InternedRefs != 5 {
		t.Fatalf("expected 5 interned refs, got %+v", stats)
	}
	if stats.InternReleaseMisses != 0 {
		t.Fatalf("expected no interner release misses, got %+v", stats)
	}
	if stats.EntryExpiryItems != 1 || stats.StaticExpiryItems != 1 {
		t.Fatalf("expected expiry index stats to track retained state, got %+v", stats)
	}
	assertCustomSCPInternerInvariant(t, store)
}

func TestCustomSCPTrimSpottersLockedDoesNotRefreshLastSeen(t *testing.T) {
	store := &CustomSCPStore{
		opts: sanitizeCustomSCPOptions(CustomSCPOptions{MaxSpottersPerKey: 1}),
	}
	entry := &customSCPEntry{
		spotters: map[string]customSCPSpotterObs{
			"N0AAA": {seenUnix: 10, cellRes1: 101},
			"N0BBB": {seenUnix: 20, cellRes1: 102},
		},
		lastSeen: 999,
	}
	store.entries = make(map[customSCPKey]*customSCPEntry, 1)
	retainCustomSCPTestEntryLocked(store, customSCPKey{call: "K1TRIM", band: "40m", bucket: "cw"}, entry)
	store.observationSpotters = len(entry.spotters)

	if trimmed := store.trimSpottersLocked(entry, nil); trimmed != 1 {
		t.Fatalf("expected one trimmed spotter, got %d", trimmed)
	}
	if entry.lastSeen != 999 {
		t.Fatalf("expected normal overflow trim to leave lastSeen unchanged, got %d", entry.lastSeen)
	}
	assertCustomSCPInternerInvariant(t, store)
}

func TestCustomSCPStoreCleanupPrunesOverflowAndStaleStatic(t *testing.T) {
	opts := CustomSCPOptions{
		Path:                   filepath.Join(t.TempDir(), "scp"),
		HorizonDays:            30,
		StaticHorizonDays:      30,
		MaxSpottersPerKey:      2,
		CoreMinScore:           1,
		CoreMinH3Cells:         1,
		SFloorMinScore:         1,
		SFloorExactMinH3Cells:  1,
		SFloorFamilyMinH3Cells: 1,
	}
	store, err := OpenCustomSCPStore(opts)
	if err != nil {
		t.Fatalf("open custom store: %v", err)
	}
	t.Cleanup(func() {
		_ = store.Close()
	})

	now := time.Now().UTC()
	key := customSCPKey{call: "K1BIG", band: "20m", bucket: "cw"}
	store.mu.Lock()
	entry := &customSCPEntry{
		spotters: map[string]customSCPSpotterObs{
			"N0AAA": {seenUnix: now.Add(-4 * time.Minute).Unix(), cellRes1: 201},
			"N0BBB": {seenUnix: now.Add(-3 * time.Minute).Unix(), cellRes1: 202},
			"N0CCC": {seenUnix: now.Add(-2 * time.Minute).Unix(), cellRes1: 203},
			"N0DDD": {seenUnix: now.Add(-1 * time.Minute).Unix(), cellRes1: 204},
		},
		lastSeen: now.Add(-1 * time.Minute).Unix(),
	}
	key = retainCustomSCPTestEntryLocked(store, key, entry)
	store.observationSpotters = 4
	retainCustomSCPTestStaticLocked(store, "K1OLD", now.Add(-31*24*time.Hour).Unix())
	store.markEntryForCleanupLocked(key, store.entries[key])
	store.mu.Unlock()
	for spotter, obs := range store.entries[key].spotters {
		setRawObservation(t, store, key, spotter, obs.seenUnix, obs.cellRes1)
	}
	store.persistStaticLocked("K1OLD", now.Add(-31*24*time.Hour).Unix())
	if item := store.entryExpiryItems[key]; item == nil || item.dueUnix != customSCPImmediateCleanupDueUnix {
		t.Fatalf("expected oversized entry queued for immediate cleanup, item=%+v", item)
	}

	store.cleanup(now)

	entry = store.entries[key]
	if entry == nil {
		t.Fatalf("expected retained entry after cleanup")
	}
	if len(entry.spotters) != 2 {
		t.Fatalf("expected cleanup to retain 2 newest spotters, got %d", len(entry.spotters))
	}
	if _, ok := entry.spotters["N0CCC"]; !ok {
		t.Fatalf("expected cleanup to retain N0CCC")
	}
	if _, ok := entry.spotters["N0DDD"]; !ok {
		t.Fatalf("expected cleanup to retain N0DDD")
	}
	if got := countObservationRecords(t, store, key); got != 2 {
		t.Fatalf("expected cleanup to reduce persisted observation records to 2, got %d", got)
	}
	if store.StaticContains("K1OLD") {
		t.Fatalf("expected stale static membership to age out")
	}
	if got := countStaticRecord(t, store, "K1OLD"); got != 0 {
		t.Fatalf("expected stale static record deleted from Pebble, got %d", got)
	}
	stats := store.StatsSnapshot()
	if stats.StaleStaticPruned == 0 {
		t.Fatalf("expected stale static prune counter to increase, got %+v", stats)
	}
	if stats.OverflowObservationsPruned < 2 {
		t.Fatalf("expected overflow prune counter to record cleanup trimming, got %+v", stats)
	}
	assertCustomSCPInternerInvariant(t, store)
}

func TestCustomSCPStoreCleanupIgnoresUpdatedEntryExpiry(t *testing.T) {
	opts := CustomSCPOptions{
		Path:                   filepath.Join(t.TempDir(), "scp"),
		HorizonDays:            30,
		StaticHorizonDays:      30,
		MaxSpottersPerKey:      2,
		CoreMinScore:           1,
		CoreMinH3Cells:         1,
		SFloorMinScore:         1,
		SFloorExactMinH3Cells:  1,
		SFloorFamilyMinH3Cells: 1,
	}
	store, err := OpenCustomSCPStore(opts)
	if err != nil {
		t.Fatalf("open custom store: %v", err)
	}
	t.Cleanup(func() {
		_ = store.Close()
	})

	now := time.Now().UTC()
	key := customSCPKey{call: "K1FRESH", band: "20m", bucket: "cw"}
	store.mu.Lock()
	entry := &customSCPEntry{
		spotters: map[string]customSCPSpotterObs{
			"N0AAA": {seenUnix: now.Add(-29 * 24 * time.Hour).Unix(), cellRes1: 201},
		},
	}
	key = retainCustomSCPTestEntryLocked(store, key, entry)
	store.observationSpotters = 1
	store.markEntryForCleanupLocked(key, entry)
	entry.spotters["N0AAA"] = customSCPSpotterObs{seenUnix: now.Add(-1 * time.Hour).Unix(), cellRes1: 201}
	store.markEntryForCleanupLocked(key, entry)
	store.mu.Unlock()

	store.cleanup(now)

	store.mu.RLock()
	entry = store.entries[key]
	if entry == nil {
		store.mu.RUnlock()
		t.Fatalf("expected refreshed entry to survive cleanup")
	}
	if len(entry.spotters) != 1 {
		store.mu.RUnlock()
		t.Fatalf("expected refreshed entry to retain one spotter, got %d", len(entry.spotters))
	}
	oldestSeen := entry.oldestSeenUnix
	cutoff := store.observationHorizonCutoffUnix(now)
	store.mu.RUnlock()
	if oldestSeen < cutoff {
		t.Fatalf("expected updated expiry to reflect fresh observation, got oldest=%d cutoff=%d", oldestSeen, cutoff)
	}
	assertCustomSCPInternerInvariant(t, store)
}

func TestCustomSCPStoreLoadDeletesStaleStaticMembership(t *testing.T) {
	path := filepath.Join(t.TempDir(), "scp")
	opts := CustomSCPOptions{
		Path:                   path,
		HorizonDays:            30,
		StaticHorizonDays:      30,
		MaxSpottersPerKey:      2,
		CoreMinScore:           1,
		CoreMinH3Cells:         1,
		SFloorMinScore:         1,
		SFloorExactMinH3Cells:  1,
		SFloorFamilyMinH3Cells: 1,
	}

	store, err := OpenCustomSCPStore(opts)
	if err != nil {
		t.Fatalf("open custom store: %v", err)
	}
	staleSeen := time.Now().UTC().Add(-31 * 24 * time.Hour).Unix()
	store.persistStaticLocked("K1STALE", staleSeen)
	if err := store.Close(); err != nil {
		t.Fatalf("close custom store: %v", err)
	}

	reopened, err := OpenCustomSCPStore(opts)
	if err != nil {
		t.Fatalf("reopen custom store: %v", err)
	}
	t.Cleanup(func() {
		_ = reopened.Close()
	})

	if reopened.StaticContains("K1STALE") {
		t.Fatalf("expected stale static membership to be ignored on load")
	}
	if got := countStaticRecord(t, reopened, "K1STALE"); got != 0 {
		t.Fatalf("expected stale static record deleted on load, got %d", got)
	}
	if stats := reopened.StatsSnapshot(); stats.StaleStaticPruned == 0 {
		t.Fatalf("expected stale static prune count on load, got %+v", stats)
	}
}

func TestCustomSCPStoreStaticMembershipOutlivesObservationHorizon(t *testing.T) {
	path := filepath.Join(t.TempDir(), "scp")
	opts := CustomSCPOptions{
		Path:                   path,
		HorizonDays:            60,
		StaticHorizonDays:      395,
		MaxSpottersPerKey:      2,
		CoreMinScore:           1,
		CoreMinH3Cells:         1,
		SFloorMinScore:         1,
		SFloorExactMinH3Cells:  1,
		SFloorFamilyMinH3Cells: 1,
	}

	store, err := OpenCustomSCPStore(opts)
	if err != nil {
		t.Fatalf("open custom store: %v", err)
	}

	call := "K1YEAR"
	key := customSCPKey{call: call, band: "20m", bucket: "cw"}
	now := time.Now().UTC()
	staticSeen := now.Add(-120 * 24 * time.Hour).Unix()
	obsSeen := now.Add(-61 * 24 * time.Hour).Unix()

	store.mu.Lock()
	retainCustomSCPTestStaticLocked(store, call, staticSeen)
	store.mu.Unlock()
	store.persistStaticLocked(call, staticSeen)
	setRawObservation(t, store, key, "N0OLD", obsSeen, 101)
	if err := store.Close(); err != nil {
		t.Fatalf("close custom store: %v", err)
	}

	reopened, err := OpenCustomSCPStore(opts)
	if err != nil {
		t.Fatalf("reopen custom store: %v", err)
	}
	t.Cleanup(func() {
		_ = reopened.Close()
	})

	if !reopened.StaticContains(call) {
		t.Fatalf("expected static membership to survive beyond observation horizon")
	}
	if got := reopened.RecentSupportCount(call, "20m", "CW", now); got != 0 {
		t.Fatalf("expected observation support to age out after 60-day horizon, got %d", got)
	}
	if got := countObservationRecords(t, reopened, key); got != 0 {
		t.Fatalf("expected stale observation record to be deleted on load, got %d", got)
	}
	if got := countStaticRecord(t, reopened, call); got != 1 {
		t.Fatalf("expected static membership record to remain, got %d", got)
	}
	assertCustomSCPInternerInvariant(t, reopened)
}

func setRawObservation(t *testing.T, store *CustomSCPStore, key customSCPKey, spotter string, seenUnix int64, cell uint16) {
	t.Helper()
	value := make([]byte, 10)
	binary.BigEndian.PutUint64(value[:8], uint64(seenUnix))
	binary.BigEndian.PutUint16(value[8:10], cell)
	if err := store.db.Set([]byte(observationKeyString(key, spotter)), value, nil); err != nil {
		t.Fatalf("set raw observation: %v", err)
	}
}

func countObservationRecords(t *testing.T, store *CustomSCPStore, key customSCPKey) int {
	t.Helper()
	iter, err := store.db.NewIter(nil)
	if err != nil {
		t.Fatalf("new iter: %v", err)
	}
	defer iter.Close()
	count := 0
	prefix := observationPrefixForKey(key)
	for iter.First(); iter.Valid(); iter.Next() {
		if len(iter.Key()) >= len(prefix) && string(iter.Key()[:len(prefix)]) == prefix {
			count++
		}
	}
	if err := iter.Error(); err != nil {
		t.Fatalf("iter error: %v", err)
	}
	return count
}

func countStaticRecord(t *testing.T, store *CustomSCPStore, call string) int {
	t.Helper()
	iter, err := store.db.NewIter(nil)
	if err != nil {
		t.Fatalf("new iter: %v", err)
	}
	defer iter.Close()
	key := customSCPMetaPrefix + call
	count := 0
	for iter.First(); iter.Valid(); iter.Next() {
		if string(iter.Key()) == key {
			count++
		}
	}
	if err := iter.Error(); err != nil {
		t.Fatalf("iter error: %v", err)
	}
	return count
}
