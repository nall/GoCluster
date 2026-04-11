package spot

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"dxcluster/pathreliability"
	"dxcluster/strutil"

	"github.com/cockroachdb/pebble"
	"github.com/cockroachdb/pebble/bloom"
)

const (
	customSCPDefaultHorizonDays       = 60
	customSCPDefaultStaticHorizonDays = 395
	customSCPDefaultMaxKeys           = 500000
	customSCPDefaultMaxSpotters       = 64
	customSCPDefaultCleanupInterval   = 10 * time.Minute
	customSCPDefaultCacheSizeBytes    = int64(64 << 20)
	customSCPDefaultBloomBits         = 10
	customSCPDefaultMemTableSize      = uint64(32 << 20)
	customSCPDefaultL0Compaction      = 4
	customSCPDefaultL0StopWrites      = 16
	customSCPDefaultCoreMinScore      = 5
	customSCPDefaultCoreMinH3Cells    = 2
	customSCPDefaultFloorMinScore     = 3
	customSCPDefaultFloorExactCells   = 2
	customSCPDefaultFloorFamilyCells  = 3

	customSCPMetaPrefix = "m|"
	customSCPObsPrefix  = "o|"
)

const (
	customSCPScoreT0 = 10 // <= 12h
	customSCPScoreT1 = 7  // <= 48h
	customSCPScoreT2 = 5  // <= 7d
	customSCPScoreT3 = 3  // <= 30d
	customSCPScoreT4 = 2  // <= 90d
	customSCPScoreT5 = 1  // <= 180d
	customSCPScoreT6 = 0  // <= 395d
)

const (
	customSCPTierT0 = iota
	customSCPTierT1
	customSCPTierT2
	customSCPTierT3
	customSCPTierT4
	customSCPTierT5
	customSCPTierT6
)

type customSCPKey struct {
	call   string
	band   string
	bucket string
}

type customSCPSpotterObs struct {
	seenUnix int64
	cellRes1 uint16
}

type customSCPEntry struct {
	spotters map[string]customSCPSpotterObs
	lastSeen int64
}

// CustomSCPOptions configures runtime and persistence behavior.
type CustomSCPOptions struct {
	Path string

	HorizonDays       int
	StaticHorizonDays int
	MaxKeys           int
	MaxSpottersPerKey int
	CleanupInterval   time.Duration

	CacheSizeBytes        int64
	BloomFilterBitsPerKey int
	MemTableSizeBytes     uint64
	L0CompactionThreshold int
	L0StopWritesThreshold int

	CoreMinScore   int
	CoreMinH3Cells int

	SFloorMinScore         int
	SFloorExactMinH3Cells  int
	SFloorFamilyMinH3Cells int

	MinSNRDBCW   int
	MinSNRDBRTTY int
}

type customSCPSnapshot struct {
	uniqueSpotters int
	uniqueCells    int
	latestSeenUnix int64
	score          int
}

// CustomSCPStatsSnapshot reports bounded custom-SCP cardinality and cleanup
// counters for opt-in diagnostics.
type CustomSCPStatsSnapshot struct {
	StaticCalls                int
	ObservationKeys            int
	ObservationSpotters        int
	OversizedKeysSeenOnLoad    uint64
	OverflowObservationsPruned uint64
	StaleObservationsPruned    uint64
	StaleStaticPruned          uint64
}

type customSCPDiagnostics struct {
	oversizedKeysSeenOnLoad    uint64
	overflowObservationsPruned uint64
	staleObservationsPruned    uint64
	staleStaticPruned          uint64
}

// CustomSCPStore holds runtime-learned static membership and recent-evidence
// rails for the correction/confidence path.
type CustomSCPStore struct {
	mu sync.RWMutex

	opts CustomSCPOptions

	entries map[customSCPKey]*customSCPEntry
	static  map[string]int64

	observationSpotters int
	diag                customSCPDiagnostics

	quit      chan struct{}
	cleanupMu sync.Mutex
	db        *pebble.DB
}

// OpenCustomSCPStore opens (or creates) a Pebble-backed custom SCP store.
func OpenCustomSCPStore(opts CustomSCPOptions) (*CustomSCPStore, error) {
	opts = sanitizeCustomSCPOptions(opts)
	if strings.TrimSpace(opts.Path) == "" {
		return nil, errors.New("custom_scp: path is empty")
	}
	if err := os.MkdirAll(opts.Path, 0o755); err != nil {
		return nil, fmt.Errorf("custom_scp: mkdir: %w", err)
	}
	pebbleOpts := &pebble.Options{
		MemTableSize:          opts.MemTableSizeBytes,
		L0CompactionThreshold: opts.L0CompactionThreshold,
		L0StopWritesThreshold: opts.L0StopWritesThreshold,
	}
	if opts.CacheSizeBytes > 0 {
		pebbleOpts.Cache = pebble.NewCache(opts.CacheSizeBytes)
	}
	if opts.BloomFilterBitsPerKey > 0 {
		filter := bloom.FilterPolicy(opts.BloomFilterBitsPerKey)
		level := pebble.LevelOptions{
			FilterPolicy: filter,
			FilterType:   pebble.TableFilter,
		}
		pebbleOpts.Levels = make([]pebble.LevelOptions, 7)
		for i := range pebbleOpts.Levels {
			pebbleOpts.Levels[i] = level
		}
	}
	db, err := pebble.Open(opts.Path, pebbleOpts)
	if err != nil {
		if pebbleOpts.Cache != nil {
			pebbleOpts.Cache.Unref()
		}
		return nil, fmt.Errorf("custom_scp: open: %w", err)
	}
	store := &CustomSCPStore{
		opts:    opts,
		entries: make(map[customSCPKey]*customSCPEntry, 1024),
		static:  make(map[string]int64, 1024),
		db:      db,
	}
	if err := store.loadFromDB(); err != nil {
		_ = store.Close()
		return nil, err
	}
	return store, nil
}

// Close closes the underlying Pebble DB.
func (s *CustomSCPStore) Close() error {
	if s == nil {
		return nil
	}
	s.StopCleanup()
	if s.db == nil {
		return nil
	}
	err := s.db.Close()
	s.db = nil
	return err
}

// StartCleanup starts periodic in-memory and on-disk stale-data cleanup.
func (s *CustomSCPStore) StartCleanup() {
	if s == nil || s.opts.CleanupInterval <= 0 {
		return
	}
	startPeriodicCleanup(&s.cleanupMu, &s.quit, s.opts.CleanupInterval, func() {
		s.cleanup(time.Now().UTC())
	})
}

// StopCleanup stops periodic cleanup.
func (s *CustomSCPStore) StopCleanup() {
	if s == nil {
		return
	}
	stopPeriodicCleanup(&s.cleanupMu, &s.quit)
}

// Checkpoint creates a Pebble checkpoint at dest.
func (s *CustomSCPStore) Checkpoint(dest string) error {
	if s == nil || s.db == nil {
		return errors.New("custom_scp: store not initialized")
	}
	if strings.TrimSpace(dest) == "" {
		return errors.New("custom_scp: checkpoint destination is empty")
	}
	if err := s.db.Checkpoint(dest, pebble.WithFlushedWAL()); err != nil {
		return fmt.Errorf("custom_scp: checkpoint: %w", err)
	}
	return nil
}

// Verify performs a bounded full scan over observation and static-membership keys.
func (s *CustomSCPStore) Verify(ctx context.Context, maxDuration time.Duration) (int64, error) {
	if s == nil || s.db == nil {
		return 0, errors.New("custom_scp: store not initialized")
	}
	return verifyCustomSCPDB(ctx, s.db, maxDuration)
}

// VerifyCustomSCPCheckpoint verifies a checkpoint directory by opening it
// read-only and scanning all records.
func VerifyCustomSCPCheckpoint(ctx context.Context, path string, maxDuration time.Duration) (int64, error) {
	if strings.TrimSpace(path) == "" {
		return 0, errors.New("custom_scp: checkpoint path is empty")
	}
	info, err := os.Stat(path)
	if err != nil {
		return 0, fmt.Errorf("custom_scp: checkpoint stat %s: %w", path, err)
	}
	if !info.IsDir() {
		return 0, fmt.Errorf("custom_scp: checkpoint %s is not a directory", path)
	}
	db, err := pebble.Open(path, &pebble.Options{ReadOnly: true})
	if err != nil {
		return 0, fmt.Errorf("custom_scp: checkpoint open %s: %w", path, err)
	}
	defer db.Close()
	return verifyCustomSCPDB(ctx, db, maxDuration)
}

func verifyCustomSCPDB(ctx context.Context, db *pebble.DB, maxDuration time.Duration) (int64, error) {
	if db == nil {
		return 0, errors.New("custom_scp: database is nil")
	}
	start := time.Now().UTC()
	deadline := time.Time{}
	if maxDuration > 0 {
		deadline = start.Add(maxDuration)
	}
	iter, err := db.NewIter(nil)
	if err != nil {
		return 0, fmt.Errorf("custom_scp: verify iterator: %w", err)
	}
	defer iter.Close()
	var scanned int64
	for iter.First(); iter.Valid(); iter.Next() {
		if ctx != nil {
			select {
			case <-ctx.Done():
				return scanned, ctx.Err()
			default:
			}
		}
		if !deadline.IsZero() && time.Now().UTC().After(deadline) {
			return scanned, errors.New("custom_scp: integrity scan timed out")
		}
		key := iter.Key()
		if strings.HasPrefix(string(key), customSCPMetaPrefix) {
			if len(iter.Value()) != 8 {
				return scanned, errors.New("custom_scp: invalid static-membership value")
			}
		}
		if strings.HasPrefix(string(key), customSCPObsPrefix) {
			if len(iter.Value()) != 10 {
				return scanned, errors.New("custom_scp: invalid observation value")
			}
		}
		scanned++
	}
	if err := iter.Error(); err != nil {
		return scanned, fmt.Errorf("custom_scp: verify iterate: %w", err)
	}
	return scanned, nil
}

// RecordSpot records one accepted spot into custom SCP runtime evidence.
func (s *CustomSCPStore) RecordSpot(sp *Spot) {
	if s == nil || sp == nil || sp.IsBeacon {
		return
	}
	// Admission is intentionally limited to strongest-confidence output to avoid
	// self-reinforcement loops where SCP-backed S/C outputs feed SCP evidence.
	if strutil.NormalizeUpper(sp.Confidence) != "V" {
		return
	}
	mode := sp.ModeNorm
	if mode == "" {
		mode = sp.Mode
	}
	if _, ok := customSCPBucketForMode(mode); !ok {
		return
	}
	call := sp.DXCallNorm
	if call == "" {
		call = sp.DXCall
	}
	call = NormalizeCallsign(call)
	if call == "" {
		return
	}
	band := sp.BandNorm
	if band == "" || band == "???" {
		band = FreqToBand(sp.Frequency)
	}
	band = NormalizeBand(band)
	if band == "" || band == "???" {
		return
	}
	spotter := sp.DECallNorm
	if spotter == "" {
		spotter = sp.DECall
	}
	spotter = strutil.NormalizeUpper(spotter)
	if spotter == "" {
		return
	}
	seenAt := sp.Time.UTC()
	if seenAt.IsZero() {
		seenAt = time.Now().UTC()
	}
	cell := uint16(0)
	grid := strings.TrimSpace(sp.DEGridNorm)
	if grid == "" {
		grid = strings.TrimSpace(sp.DEMetadata.Grid)
	}
	if grid != "" {
		cell = uint16(pathreliability.EncodeCoarseCell(grid))
	}

	keys := CorrectionFamilyKeys(call)
	if len(keys) == 0 {
		keys = []string{call}
	}
	for _, k := range keys {
		s.recordObservation(k, band, mode, spotter, cell, sp.Report, sp.HasReport, seenAt)
	}
}

func (s *CustomSCPStore) recordObservation(call, band, mode, spotter string, cellRes1 uint16, report int, hasReport bool, seenAt time.Time) {
	if s == nil {
		return
	}
	call = NormalizeCallsign(call)
	band = NormalizeBand(band)
	bucket, ok := customSCPBucketForMode(mode)
	if !ok || call == "" || band == "" || band == "???" || spotter == "" {
		return
	}
	if !s.snrPasses(bucket, report, hasReport) {
		return
	}
	seenUnix := seenAt.UTC().Unix()
	if seenUnix <= 0 {
		seenUnix = time.Now().UTC().Unix()
	}
	cutoff := s.observationHorizonCutoffUnix(time.Now().UTC())
	if seenUnix < cutoff {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.static[call] = maxInt64(s.static[call], seenUnix)

	key := customSCPKey{call: call, band: band, bucket: bucket}
	entry := s.entries[key]
	if entry == nil {
		if len(s.entries) >= s.opts.MaxKeys {
			s.evictOldestKeyLocked()
		}
		if len(s.entries) >= s.opts.MaxKeys {
			return
		}
		entry = &customSCPEntry{spotters: make(map[string]customSCPSpotterObs, 4)}
		s.entries[key] = entry
	}
	removed := s.pruneEntryLocked(entry, cutoff)
	prev, exists := entry.spotters[spotter]
	if !exists || seenUnix > prev.seenUnix {
		if !exists {
			s.observationSpotters++
		}
		entry.spotters[spotter] = customSCPSpotterObs{seenUnix: seenUnix, cellRes1: cellRes1}
	}
	if seenUnix > entry.lastSeen {
		entry.lastSeen = seenUnix
	}
	if len(entry.spotters) > s.opts.MaxSpottersPerKey {
		overflow := s.trimSpottersLocked(entry)
		s.diag.overflowObservationsPruned += uint64(len(overflow))
		removed = append(removed, overflow...)
	}
	if len(entry.spotters) == 0 {
		s.deleteEntryLocked(key)
		s.deleteObservationPrefixLocked(key)
		return
	}
	s.persistStaticLocked(call, s.static[call])
	if obs, ok := entry.spotters[spotter]; ok {
		s.persistObservationLocked(key, spotter, obs)
	}
	s.deleteObservationSpottersLocked(key, entry, removed, nil)
}

// StaticContains reports whether call is in persisted custom SCP membership.
func (s *CustomSCPStore) StaticContains(call string) bool {
	if s == nil {
		return false
	}
	call = NormalizeCallsign(call)
	if call == "" {
		return false
	}
	s.mu.RLock()
	seen, ok := s.static[call]
	s.mu.RUnlock()
	if !ok {
		return false
	}
	return seen >= s.staticHorizonCutoffUnix(time.Now().UTC())
}

// StatsSnapshot reports current custom-SCP cardinalities and bounded cleanup
// counters for opt-in diagnostics.
func (s *CustomSCPStore) StatsSnapshot() CustomSCPStatsSnapshot {
	if s == nil {
		return CustomSCPStatsSnapshot{}
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return CustomSCPStatsSnapshot{
		StaticCalls:                len(s.static),
		ObservationKeys:            len(s.entries),
		ObservationSpotters:        s.observationSpotters,
		OversizedKeysSeenOnLoad:    s.diag.oversizedKeysSeenOnLoad,
		OverflowObservationsPruned: s.diag.overflowObservationsPruned,
		StaleObservationsPruned:    s.diag.staleObservationsPruned,
		StaleStaticPruned:          s.diag.staleStaticPruned,
	}
}

// HasRecentSupport implements RecentSupportStore for resolver/stabilizer rails.
func (s *CustomSCPStore) HasRecentSupport(call, band, mode string, minUnique int, now time.Time) bool {
	snapshot := s.snapshotFor(call, band, mode, now)
	if snapshot.score < s.opts.CoreMinScore {
		return false
	}
	if snapshot.uniqueCells < s.opts.CoreMinH3Cells {
		return false
	}
	return snapshot.uniqueSpotters >= normalizeMinUnique(minUnique, s.opts.MaxSpottersPerKey)
}

// RecentSupportCount implements RecentSupportStore for resolver/stabilizer rails.
// Count is returned only when core score/cell gates pass.
func (s *CustomSCPStore) RecentSupportCount(call, band, mode string, now time.Time) int {
	snapshot := s.snapshotFor(call, band, mode, now)
	if snapshot.score < s.opts.CoreMinScore || snapshot.uniqueCells < s.opts.CoreMinH3Cells {
		return 0
	}
	return snapshot.uniqueSpotters
}

// HasSFloorSupportExact reports exact-call support for S-floor promotion.
func (s *CustomSCPStore) HasSFloorSupportExact(call, band, mode string, minUnique int, now time.Time) bool {
	snapshot := s.snapshotFor(call, band, mode, now)
	if snapshot.score < s.opts.SFloorMinScore {
		return false
	}
	if snapshot.uniqueCells < s.opts.SFloorExactMinH3Cells {
		return false
	}
	return snapshot.uniqueSpotters >= normalizeMinUnique(minUnique, s.opts.MaxSpottersPerKey)
}

// HasSFloorSupportFamily reports family-fallback support for S-floor promotion.
func (s *CustomSCPStore) HasSFloorSupportFamily(calls []string, band, mode string, minUnique int, now time.Time) bool {
	if s == nil {
		return false
	}
	best := customSCPSnapshot{}
	for _, call := range calls {
		snap := s.snapshotFor(call, band, mode, now)
		if snap.score > best.score || (snap.score == best.score && snap.uniqueSpotters > best.uniqueSpotters) {
			best = snap
		}
	}
	if best.score < s.opts.SFloorMinScore {
		return false
	}
	if best.uniqueCells < s.opts.SFloorFamilyMinH3Cells {
		return false
	}
	return best.uniqueSpotters >= normalizeMinUnique(minUnique, s.opts.MaxSpottersPerKey)
}

// ActiveCallCount returns distinct active calls within horizon.
func (s *CustomSCPStore) ActiveCallCount(now time.Time) int {
	if s == nil {
		return 0
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	cutoff := s.observationHorizonCutoffUnix(now.UTC())
	s.mu.Lock()
	defer s.mu.Unlock()
	calls := make(map[string]struct{})
	for key, entry := range s.entries {
		s.pruneEntryLocked(entry, cutoff)
		if len(entry.spotters) == 0 {
			s.deleteEntryLocked(key)
			continue
		}
		calls[key.call] = struct{}{}
	}
	return len(calls)
}

// ActiveCallCountsByBand returns distinct active call counts per band.
func (s *CustomSCPStore) ActiveCallCountsByBand(now time.Time) map[string]int {
	if s == nil {
		return nil
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	cutoff := s.observationHorizonCutoffUnix(now.UTC())
	s.mu.Lock()
	defer s.mu.Unlock()
	byBand := make(map[string]map[string]struct{})
	for key, entry := range s.entries {
		s.pruneEntryLocked(entry, cutoff)
		if len(entry.spotters) == 0 {
			s.deleteEntryLocked(key)
			continue
		}
		calls := byBand[key.band]
		if calls == nil {
			calls = make(map[string]struct{})
			byBand[key.band] = calls
		}
		calls[key.call] = struct{}{}
	}
	out := make(map[string]int, len(byBand))
	for band, calls := range byBand {
		out[band] = len(calls)
	}
	return out
}

func (s *CustomSCPStore) snapshotFor(call, band, mode string, now time.Time) customSCPSnapshot {
	if s == nil {
		return customSCPSnapshot{}
	}
	call = NormalizeCallsign(call)
	band = NormalizeBand(band)
	bucket, ok := customSCPBucketForMode(mode)
	if !ok || call == "" || band == "" || band == "???" {
		return customSCPSnapshot{}
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	cutoff := s.observationHorizonCutoffUnix(now.UTC())

	s.mu.Lock()
	defer s.mu.Unlock()
	key := customSCPKey{call: call, band: band, bucket: bucket}
	entry := s.entries[key]
	if entry == nil {
		return customSCPSnapshot{}
	}
	s.pruneEntryLocked(entry, cutoff)
	if len(entry.spotters) == 0 {
		s.deleteEntryLocked(key)
		return customSCPSnapshot{}
	}
	cells := make(map[uint16]struct{}, len(entry.spotters))
	latestSeen := int64(0)
	for _, obs := range entry.spotters {
		if obs.seenUnix > latestSeen {
			latestSeen = obs.seenUnix
		}
		if obs.cellRes1 != 0 {
			cells[obs.cellRes1] = struct{}{}
		}
	}
	return customSCPSnapshot{
		uniqueSpotters: len(entry.spotters),
		uniqueCells:    len(cells),
		latestSeenUnix: latestSeen,
		score:          scoreForTier(tierForAge(now.UTC().Unix() - latestSeen)),
	}
}

func (s *CustomSCPStore) cleanup(now time.Time) {
	if s == nil {
		return
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	observationCutoff := s.observationHorizonCutoffUnix(now.UTC())
	staticCutoff := s.staticHorizonCutoffUnix(now.UTC())

	staleStaticCalls := make([]string, 0)
	overflowDeletes := make([]customSCPDeleteRequest, 0)
	s.mu.Lock()
	for key, entry := range s.entries {
		removed := s.pruneEntryLocked(entry, observationCutoff)
		if len(removed) > 0 {
			s.diag.staleObservationsPruned += uint64(len(removed))
			overflowDeletes = append(overflowDeletes, customSCPDeleteRequest{key: key, spotters: removed})
		}
		if len(entry.spotters) > s.opts.MaxSpottersPerKey {
			trimmed := s.trimSpottersLocked(entry)
			if len(trimmed) > 0 {
				s.diag.overflowObservationsPruned += uint64(len(trimmed))
				overflowDeletes = append(overflowDeletes, customSCPDeleteRequest{key: key, spotters: trimmed})
			}
		}
		if len(entry.spotters) == 0 {
			s.deleteEntryLocked(key)
		}
	}
	for call, seen := range s.static {
		if seen < staticCutoff {
			delete(s.static, call)
			s.diag.staleStaticPruned++
			staleStaticCalls = append(staleStaticCalls, call)
		}
	}
	for len(s.entries) > s.opts.MaxKeys {
		s.evictOldestKeyLocked()
	}
	s.mu.Unlock()

	// Best-effort disk cleanup: drop stale and overflow records so disk
	// retention converges to the bounded in-memory view.
	if s.db != nil {
		iter, err := s.db.NewIter(nil)
		if err != nil {
			return
		}
		defer iter.Close()
		batch := s.db.NewBatch()
		defer batch.Close()
		pending := 0
		for _, call := range staleStaticCalls {
			_ = batch.Delete([]byte(customSCPMetaPrefix+call), nil)
			pending++
		}
		for _, req := range overflowDeletes {
			pending += s.deleteObservationSpottersFromRequestLocked(req, batch)
		}
		for iter.First(); iter.Valid(); iter.Next() {
			key := string(iter.Key())
			switch {
			case strings.HasPrefix(key, customSCPMetaPrefix):
				value := iter.Value()
				if len(value) != 8 {
					_ = batch.Delete(iter.Key(), nil)
					pending++
					continue
				}
				seen := int64(binary.BigEndian.Uint64(value))
				if seen < staticCutoff {
					_ = batch.Delete(iter.Key(), nil)
					pending++
				}
			case strings.HasPrefix(key, customSCPObsPrefix):
				value := iter.Value()
				if len(value) != 10 {
					_ = batch.Delete(iter.Key(), nil)
					pending++
				} else {
					seen := int64(binary.BigEndian.Uint64(value[:8]))
					if seen < observationCutoff {
						_ = batch.Delete(iter.Key(), nil)
						pending++
					}
				}
			default:
				continue
			}
			if pending >= 512 {
				_ = batch.Commit(pebble.Sync)
				batch.Reset()
				pending = 0
			}
		}
		if pending > 0 {
			_ = batch.Commit(pebble.Sync)
		}
	}
}

func (s *CustomSCPStore) loadFromDB() error {
	if s == nil || s.db == nil {
		return nil
	}
	iter, err := s.db.NewIter(nil)
	if err != nil {
		return fmt.Errorf("custom_scp: load iterator: %w", err)
	}
	defer iter.Close()
	observationCutoff := s.observationHorizonCutoffUnix(time.Now().UTC())
	staticCutoff := s.staticHorizonCutoffUnix(time.Now().UTC())
	batch := s.db.NewBatch()
	defer batch.Close()
	pendingDeletes := 0
	var currentKey customSCPKey
	haveCurrentKey := false
	overflowSeenForCurrentKey := false
	for iter.First(); iter.Valid(); iter.Next() {
		k := string(iter.Key())
		v := iter.Value()
		switch {
		case strings.HasPrefix(k, customSCPMetaPrefix):
			if len(v) != 8 {
				_ = batch.Delete(iter.Key(), nil)
				pendingDeletes++
				continue
			}
			call := strings.TrimPrefix(k, customSCPMetaPrefix)
			seen := int64(binary.BigEndian.Uint64(v))
			if call == "" {
				continue
			}
			if seen < staticCutoff {
				_ = batch.Delete(iter.Key(), nil)
				s.diag.staleStaticPruned++
				pendingDeletes++
				continue
			}
			s.static[NormalizeCallsign(call)] = seen
		case strings.HasPrefix(k, customSCPObsPrefix):
			call, band, bucket, spotter, ok := parseObservationKey(k)
			if !ok || len(v) != 10 {
				_ = batch.Delete(iter.Key(), nil)
				pendingDeletes++
				continue
			}
			seen := int64(binary.BigEndian.Uint64(v[:8]))
			if seen < observationCutoff {
				_ = batch.Delete(iter.Key(), nil)
				s.diag.staleObservationsPruned++
				pendingDeletes++
				continue
			}
			cell := binary.BigEndian.Uint16(v[8:10])
			key := customSCPKey{call: call, band: band, bucket: bucket}
			if !haveCurrentKey || key != currentKey {
				currentKey = key
				haveCurrentKey = true
				overflowSeenForCurrentKey = false
			}
			entry := s.entries[key]
			if entry == nil {
				if len(s.entries) >= s.opts.MaxKeys {
					s.evictOldestKeyLocked()
				}
				if len(s.entries) >= s.opts.MaxKeys {
					continue
				}
				entry = &customSCPEntry{spotters: make(map[string]customSCPSpotterObs, 4)}
				s.entries[key] = entry
			}
			prev, exists := entry.spotters[spotter]
			if !exists || seen > prev.seenUnix {
				if !exists {
					s.observationSpotters++
				}
				entry.spotters[spotter] = customSCPSpotterObs{seenUnix: seen, cellRes1: cell}
			}
			if seen > entry.lastSeen {
				entry.lastSeen = seen
			}
			if len(entry.spotters) > s.opts.MaxSpottersPerKey {
				if !overflowSeenForCurrentKey {
					s.diag.oversizedKeysSeenOnLoad++
					overflowSeenForCurrentKey = true
				}
				trimmed := s.trimSpottersLocked(entry)
				s.diag.overflowObservationsPruned += uint64(len(trimmed))
				pendingDeletes += s.deleteObservationSpottersLocked(key, entry, trimmed, batch)
			}
		}
		if pendingDeletes >= 512 {
			_ = batch.Commit(pebble.NoSync)
			batch.Reset()
			pendingDeletes = 0
		}
	}
	if err := iter.Error(); err != nil {
		return fmt.Errorf("custom_scp: load iterate: %w", err)
	}
	if pendingDeletes > 0 {
		_ = batch.Commit(pebble.NoSync)
	}
	return nil
}

func (s *CustomSCPStore) persistStaticLocked(call string, seenUnix int64) {
	if s == nil || s.db == nil || call == "" {
		return
	}
	value := make([]byte, 8)
	binary.BigEndian.PutUint64(value, uint64(seenUnix))
	_ = s.db.Set([]byte(customSCPMetaPrefix+call), value, pebble.NoSync)
}

func (s *CustomSCPStore) persistObservationLocked(key customSCPKey, spotter string, obs customSCPSpotterObs) {
	if s == nil || s.db == nil {
		return
	}
	value := make([]byte, 10)
	binary.BigEndian.PutUint64(value[:8], uint64(obs.seenUnix))
	binary.BigEndian.PutUint16(value[8:10], obs.cellRes1)
	_ = s.db.Set([]byte(observationKeyString(key, spotter)), value, pebble.NoSync)
}

func (s *CustomSCPStore) evictOldestKeyLocked() {
	var victim customSCPKey
	set := false
	var oldest int64
	for key, entry := range s.entries {
		if !set || entry.lastSeen < oldest {
			victim = key
			oldest = entry.lastSeen
			set = true
		}
	}
	if !set {
		return
	}
	s.deleteEntryLocked(victim)
	s.deleteObservationPrefixLocked(victim)
}

func (s *CustomSCPStore) pruneEntryLocked(entry *customSCPEntry, cutoff int64) []string {
	if entry == nil || len(entry.spotters) == 0 {
		return nil
	}
	removed := make([]string, 0)
	for spotter, obs := range entry.spotters {
		if obs.seenUnix < cutoff {
			delete(entry.spotters, spotter)
			s.observationSpotters--
			removed = append(removed, spotter)
		}
	}
	s.refreshEntryLastSeenLocked(entry)
	return removed
}

func (s *CustomSCPStore) trimSpottersLocked(entry *customSCPEntry) []string {
	if entry == nil || len(entry.spotters) <= s.opts.MaxSpottersPerKey {
		return nil
	}
	type candidate struct {
		spotter string
		seen    int64
	}
	all := make([]candidate, 0, len(entry.spotters))
	for spotter, obs := range entry.spotters {
		all = append(all, candidate{spotter: spotter, seen: obs.seenUnix})
	}
	sort.Slice(all, func(i, j int) bool {
		if all[i].seen == all[j].seen {
			return all[i].spotter < all[j].spotter
		}
		return all[i].seen < all[j].seen
	})
	remove := len(all) - s.opts.MaxSpottersPerKey
	removed := make([]string, 0, remove)
	for i := 0; i < remove; i++ {
		delete(entry.spotters, all[i].spotter)
		s.observationSpotters--
		removed = append(removed, all[i].spotter)
	}
	s.refreshEntryLastSeenLocked(entry)
	return removed
}

type customSCPDeleteRequest struct {
	key      customSCPKey
	spotters []string
}

func (s *CustomSCPStore) observationHorizonCutoffUnix(now time.Time) int64 {
	return now.UTC().Add(-time.Duration(s.opts.HorizonDays) * 24 * time.Hour).Unix()
}

func (s *CustomSCPStore) staticHorizonCutoffUnix(now time.Time) int64 {
	return now.UTC().Add(-time.Duration(s.opts.StaticHorizonDays) * 24 * time.Hour).Unix()
}

func (s *CustomSCPStore) deleteEntryLocked(key customSCPKey) {
	entry := s.entries[key]
	if entry == nil {
		return
	}
	s.observationSpotters -= len(entry.spotters)
	if s.observationSpotters < 0 {
		s.observationSpotters = 0
	}
	delete(s.entries, key)
}

func (s *CustomSCPStore) refreshEntryLastSeenLocked(entry *customSCPEntry) {
	if entry == nil {
		return
	}
	latest := int64(0)
	for _, obs := range entry.spotters {
		if obs.seenUnix > latest {
			latest = obs.seenUnix
		}
	}
	entry.lastSeen = latest
}

func (s *CustomSCPStore) deleteObservationSpottersFromRequestLocked(req customSCPDeleteRequest, batch *pebble.Batch) int {
	return s.deleteObservationSpottersLocked(req.key, s.entries[req.key], req.spotters, batch)
}

func (s *CustomSCPStore) deleteObservationSpottersLocked(key customSCPKey, entry *customSCPEntry, spotters []string, batch *pebble.Batch) int {
	if s == nil || s.db == nil || len(spotters) == 0 {
		return 0
	}
	uniq := make(map[string]struct{}, len(spotters))
	deleted := 0
	for _, spotter := range spotters {
		if spotter == "" {
			continue
		}
		if entry != nil {
			if _, ok := entry.spotters[spotter]; ok {
				continue
			}
		}
		if _, ok := uniq[spotter]; ok {
			continue
		}
		uniq[spotter] = struct{}{}
		keyBytes := []byte(observationKeyString(key, spotter))
		if batch != nil {
			_ = batch.Delete(keyBytes, nil)
		} else {
			_ = s.db.Delete(keyBytes, pebble.NoSync)
		}
		deleted++
	}
	return deleted
}

func (s *CustomSCPStore) deleteObservationPrefixLocked(key customSCPKey) {
	if s == nil || s.db == nil {
		return
	}
	prefix := []byte(observationPrefixForKey(key))
	iter, err := s.db.NewIter(&pebble.IterOptions{
		LowerBound: prefix,
		UpperBound: append(prefix, 0xff),
	})
	if err != nil {
		return
	}
	defer iter.Close()
	batch := s.db.NewBatch()
	defer batch.Close()
	for iter.First(); iter.Valid(); iter.Next() {
		_ = batch.Delete(iter.Key(), nil)
	}
	_ = batch.Commit(pebble.NoSync)
}

func (s *CustomSCPStore) snrPasses(bucket string, report int, hasReport bool) bool {
	switch bucket {
	case "cw":
		if s.opts.MinSNRDBCW > 0 {
			return hasReport && report >= s.opts.MinSNRDBCW
		}
	case "rtty":
		if s.opts.MinSNRDBRTTY > 0 {
			return hasReport && report >= s.opts.MinSNRDBRTTY
		}
	}
	return true
}

func sanitizeCustomSCPOptions(opts CustomSCPOptions) CustomSCPOptions {
	if strings.TrimSpace(opts.Path) == "" {
		opts.Path = filepath.Join("data", "scp")
	}
	if opts.HorizonDays <= 0 {
		opts.HorizonDays = customSCPDefaultHorizonDays
	}
	if opts.StaticHorizonDays <= 0 {
		opts.StaticHorizonDays = customSCPDefaultStaticHorizonDays
	}
	if opts.MaxKeys <= 0 {
		opts.MaxKeys = customSCPDefaultMaxKeys
	}
	if opts.MaxSpottersPerKey <= 0 {
		opts.MaxSpottersPerKey = customSCPDefaultMaxSpotters
	}
	if opts.CleanupInterval <= 0 {
		opts.CleanupInterval = customSCPDefaultCleanupInterval
	}
	if opts.CacheSizeBytes <= 0 {
		opts.CacheSizeBytes = customSCPDefaultCacheSizeBytes
	}
	if opts.BloomFilterBitsPerKey <= 0 {
		opts.BloomFilterBitsPerKey = customSCPDefaultBloomBits
	}
	if opts.MemTableSizeBytes <= 0 {
		opts.MemTableSizeBytes = customSCPDefaultMemTableSize
	}
	if opts.L0CompactionThreshold <= 0 {
		opts.L0CompactionThreshold = customSCPDefaultL0Compaction
	}
	if opts.L0StopWritesThreshold <= opts.L0CompactionThreshold {
		opts.L0StopWritesThreshold = customSCPDefaultL0StopWrites
		if opts.L0StopWritesThreshold <= opts.L0CompactionThreshold {
			opts.L0StopWritesThreshold = opts.L0CompactionThreshold + 4
		}
	}
	if opts.CoreMinScore <= 0 {
		opts.CoreMinScore = customSCPDefaultCoreMinScore
	}
	if opts.CoreMinH3Cells <= 0 {
		opts.CoreMinH3Cells = customSCPDefaultCoreMinH3Cells
	}
	if opts.SFloorMinScore <= 0 {
		opts.SFloorMinScore = customSCPDefaultFloorMinScore
	}
	if opts.SFloorExactMinH3Cells <= 0 {
		opts.SFloorExactMinH3Cells = customSCPDefaultFloorExactCells
	}
	if opts.SFloorFamilyMinH3Cells <= 0 {
		opts.SFloorFamilyMinH3Cells = customSCPDefaultFloorFamilyCells
	}
	return opts
}

func customSCPBucketForMode(mode string) (string, bool) {
	switch strutil.NormalizeUpper(mode) {
	case "USB", "LSB":
		return "voice", true
	case "CW":
		return "cw", true
	case "RTTY":
		return "rtty", true
	case "FT2":
		return "ft2", true
	case "FT4":
		return "ft4", true
	case "FT8":
		return "ft8", true
	default:
		return "", false
	}
}

func normalizeMinUnique(minUnique int, max int) int {
	if minUnique <= 0 {
		minUnique = 2
	}
	if max > 0 && minUnique > max {
		return max
	}
	return minUnique
}

func tierForAge(ageSeconds int64) int {
	switch {
	case ageSeconds <= int64((12 * time.Hour).Seconds()):
		return customSCPTierT0
	case ageSeconds <= int64((48 * time.Hour).Seconds()):
		return customSCPTierT1
	case ageSeconds <= int64((7 * 24 * time.Hour).Seconds()):
		return customSCPTierT2
	case ageSeconds <= int64((30 * 24 * time.Hour).Seconds()):
		return customSCPTierT3
	case ageSeconds <= int64((90 * 24 * time.Hour).Seconds()):
		return customSCPTierT4
	case ageSeconds <= int64((180 * 24 * time.Hour).Seconds()):
		return customSCPTierT5
	default:
		return customSCPTierT6
	}
}

func scoreForTier(tier int) int {
	switch tier {
	case customSCPTierT0:
		return customSCPScoreT0
	case customSCPTierT1:
		return customSCPScoreT1
	case customSCPTierT2:
		return customSCPScoreT2
	case customSCPTierT3:
		return customSCPScoreT3
	case customSCPTierT4:
		return customSCPScoreT4
	case customSCPTierT5:
		return customSCPScoreT5
	default:
		return customSCPScoreT6
	}
}

func observationKeyString(key customSCPKey, spotter string) string {
	return customSCPObsPrefix + key.call + "|" + key.band + "|" + key.bucket + "|" + strutil.NormalizeUpper(spotter)
}

func observationPrefixForKey(key customSCPKey) string {
	return customSCPObsPrefix + key.call + "|" + key.band + "|" + key.bucket + "|"
}

func parseObservationKey(key string) (call, band, bucket, spotter string, ok bool) {
	raw := strings.TrimPrefix(key, customSCPObsPrefix)
	parts := strings.Split(raw, "|")
	if len(parts) != 4 {
		return "", "", "", "", false
	}
	call = NormalizeCallsign(parts[0])
	band = NormalizeBand(parts[1])
	bucket = strings.ToLower(strings.TrimSpace(parts[2]))
	spotter = strutil.NormalizeUpper(parts[3])
	if call == "" || band == "" || spotter == "" {
		return "", "", "", "", false
	}
	return call, band, bucket, spotter, true
}

func maxInt64(a, b int64) int64 {
	if b > a {
		return b
	}
	return a
}
