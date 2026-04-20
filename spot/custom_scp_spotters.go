package spot

type customSCPSpotterEntry struct {
	spotter  string
	seenUnix int64
	cellRes1 uint16
}

// customSCPEntry spotters are owned by CustomSCPStore under s.mu. The slice is
// sorted by spotter for deterministic lookup, duplicate prevention, and stable
// overflow ties. Retained entries store interned spotter strings; pending load
// entries use the same sorted shape before promotion into the store.
func makeCustomSCPSpotters(maxSpotters int) []customSCPSpotterEntry {
	capacity := customSCPSpotterInitialCap(maxSpotters)
	if capacity == 0 {
		return nil
	}
	return make([]customSCPSpotterEntry, 0, capacity)
}

func customSCPSpotterInitialCap(maxSpotters int) int {
	if maxSpotters <= 0 {
		return 0
	}
	if maxSpotters < 4 {
		return maxSpotters + 1
	}
	return 4
}

func findSpotterIndex(spotters []customSCPSpotterEntry, spotter string) (int, bool) {
	lo, hi := 0, len(spotters)
	for lo < hi {
		mid := int(uint(lo+hi) >> 1)
		if spotters[mid].spotter < spotter {
			lo = mid + 1
			continue
		}
		hi = mid
	}
	return lo, lo < len(spotters) && spotters[lo].spotter == spotter
}

func entryHasSpotter(entry *customSCPEntry, spotter string) bool {
	if entry == nil || spotter == "" {
		return false
	}
	_, ok := findSpotterIndex(entry.spotters, spotter)
	return ok
}

func entrySpotterObs(entry *customSCPEntry, spotter string) (customSCPSpotterObs, bool) {
	if entry == nil || spotter == "" {
		return customSCPSpotterObs{}, false
	}
	idx, ok := findSpotterIndex(entry.spotters, spotter)
	if !ok {
		return customSCPSpotterObs{}, false
	}
	item := entry.spotters[idx]
	return customSCPSpotterObs{seenUnix: item.seenUnix, cellRes1: item.cellRes1}, true
}

func entryUpsertSpotter(entry *customSCPEntry, spotter string, obs customSCPSpotterObs, maxSpotters int) (inserted bool, updated bool) {
	if entry == nil || spotter == "" {
		return false, false
	}
	idx, ok := findSpotterIndex(entry.spotters, spotter)
	if ok {
		if obs.seenUnix <= entry.spotters[idx].seenUnix {
			return false, false
		}
		entry.spotters[idx].seenUnix = obs.seenUnix
		entry.spotters[idx].cellRes1 = obs.cellRes1
		return false, true
	}
	item := customSCPSpotterEntry{spotter: spotter, seenUnix: obs.seenUnix, cellRes1: obs.cellRes1}
	entry.spotters = insertCustomSCPSpotter(entry.spotters, idx, item, maxSpotters)
	return true, true
}

func insertCustomSCPSpotter(spotters []customSCPSpotterEntry, idx int, item customSCPSpotterEntry, maxSpotters int) []customSCPSpotterEntry {
	if len(spotters) < cap(spotters) {
		spotters = append(spotters, customSCPSpotterEntry{})
		copy(spotters[idx+1:], spotters[idx:])
		spotters[idx] = item
		return spotters
	}
	next := make([]customSCPSpotterEntry, len(spotters)+1, customSCPSpotterGrowthCap(cap(spotters), len(spotters)+1, maxSpotters))
	copy(next, spotters[:idx])
	next[idx] = item
	copy(next[idx+1:], spotters[idx:])
	return next
}

func customSCPSpotterGrowthCap(currentCap, needed, maxSpotters int) int {
	nextCap := currentCap * 2
	if nextCap < 4 {
		nextCap = 4
	}
	maxCap := needed
	if maxSpotters > 0 {
		maxCap = maxSpotters + 1
	}
	if nextCap > maxCap {
		nextCap = maxCap
	}
	if nextCap < needed {
		nextCap = needed
	}
	return nextCap
}

func entryDeleteSpotter(entry *customSCPEntry, spotter string) (customSCPSpotterEntry, bool) {
	if entry == nil || spotter == "" {
		return customSCPSpotterEntry{}, false
	}
	idx, ok := findSpotterIndex(entry.spotters, spotter)
	if !ok {
		return customSCPSpotterEntry{}, false
	}
	return entryDeleteSpotterAt(entry, idx), true
}

func entryDeleteSpotterAt(entry *customSCPEntry, idx int) customSCPSpotterEntry {
	removed := entry.spotters[idx]
	copy(entry.spotters[idx:], entry.spotters[idx+1:])
	last := len(entry.spotters) - 1
	entry.spotters[last] = customSCPSpotterEntry{}
	entry.spotters = entry.spotters[:last]
	if len(entry.spotters) == 0 {
		entry.spotters = nil
	}
	return removed
}

func oldestCustomSCPSpotterIndex(entry *customSCPEntry) (int, bool) {
	if entry == nil || len(entry.spotters) == 0 {
		return 0, false
	}
	victim := 0
	for i := 1; i < len(entry.spotters); i++ {
		current := entry.spotters[i]
		selected := entry.spotters[victim]
		if current.seenUnix < selected.seenUnix || (current.seenUnix == selected.seenUnix && current.spotter < selected.spotter) {
			victim = i
		}
	}
	return victim, true
}

func compactCustomSCPSpotters(spotters []customSCPSpotterEntry) []customSCPSpotterEntry {
	if len(spotters) == 0 {
		return nil
	}
	compact := make([]customSCPSpotterEntry, len(spotters))
	copy(compact, spotters)
	return compact
}
