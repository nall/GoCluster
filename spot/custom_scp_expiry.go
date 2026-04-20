package spot

import "container/heap"

const customSCPImmediateCleanupDueUnix int64 = -1 << 63

type customSCPEntryExpiryItem struct {
	key     customSCPKey
	dueUnix int64
	index   int
}

type customSCPEntryExpiryHeap []*customSCPEntryExpiryItem

func (h customSCPEntryExpiryHeap) Len() int { return len(h) }

func (h customSCPEntryExpiryHeap) Less(i, j int) bool {
	if h[i].dueUnix != h[j].dueUnix {
		return h[i].dueUnix < h[j].dueUnix
	}
	return customSCPKeyLess(h[i].key, h[j].key)
}

func (h customSCPEntryExpiryHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
	h[i].index = i
	h[j].index = j
}

func (h *customSCPEntryExpiryHeap) Push(x any) {
	item := x.(*customSCPEntryExpiryItem)
	item.index = len(*h)
	*h = append(*h, item)
}

func (h *customSCPEntryExpiryHeap) Pop() any {
	old := *h
	n := len(old)
	item := old[n-1]
	old[n-1] = nil
	item.index = -1
	*h = old[:n-1]
	return item
}

type customSCPStaticExpiryItem struct {
	call    string
	dueUnix int64
	index   int
}

type customSCPStaticExpiryHeap []*customSCPStaticExpiryItem

func (h customSCPStaticExpiryHeap) Len() int { return len(h) }

func (h customSCPStaticExpiryHeap) Less(i, j int) bool {
	if h[i].dueUnix != h[j].dueUnix {
		return h[i].dueUnix < h[j].dueUnix
	}
	return h[i].call < h[j].call
}

func (h customSCPStaticExpiryHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
	h[i].index = i
	h[j].index = j
}

func (h *customSCPStaticExpiryHeap) Push(x any) {
	item := x.(*customSCPStaticExpiryItem)
	item.index = len(*h)
	*h = append(*h, item)
}

func (h *customSCPStaticExpiryHeap) Pop() any {
	old := *h
	n := len(old)
	item := old[n-1]
	old[n-1] = nil
	item.index = -1
	*h = old[:n-1]
	return item
}

func (s *CustomSCPStore) refreshEntryAgesLocked(entry *customSCPEntry) {
	if entry == nil || len(entry.spotters) == 0 {
		if entry != nil {
			entry.lastSeen = 0
			entry.oldestSeenUnix = 0
		}
		return
	}
	latest := int64(0)
	oldest := int64(0)
	for _, spotter := range entry.spotters {
		if spotter.seenUnix > latest {
			latest = spotter.seenUnix
		}
		if oldest == 0 || spotter.seenUnix < oldest {
			oldest = spotter.seenUnix
		}
	}
	entry.lastSeen = latest
	entry.oldestSeenUnix = oldest
}

func (s *CustomSCPStore) entryCleanupDueUnix(entry *customSCPEntry) int64 {
	if entry == nil || len(entry.spotters) == 0 {
		return 0
	}
	if len(entry.spotters) > s.opts.MaxSpottersPerKey {
		return customSCPImmediateCleanupDueUnix
	}
	return entry.oldestSeenUnix
}

func (s *CustomSCPStore) upsertEntryExpiryLocked(key customSCPKey, entry *customSCPEntry) {
	if s == nil {
		return
	}
	if s.entryExpiryItems == nil {
		s.entryExpiryItems = make(map[customSCPKey]*customSCPEntryExpiryItem, 16)
	}
	if entry == nil || len(entry.spotters) == 0 {
		s.deleteEntryExpiryLocked(key)
		return
	}
	dueUnix := s.entryCleanupDueUnix(entry)
	if dueUnix == 0 {
		s.deleteEntryExpiryLocked(key)
		return
	}
	if item := s.entryExpiryItems[key]; item != nil {
		item.dueUnix = dueUnix
		heap.Fix(&s.entryExpiry, item.index)
		return
	}
	item := &customSCPEntryExpiryItem{key: key, dueUnix: dueUnix}
	heap.Push(&s.entryExpiry, item)
	s.entryExpiryItems[key] = item
}

func (s *CustomSCPStore) deleteEntryExpiryLocked(key customSCPKey) {
	if s == nil {
		return
	}
	item := s.entryExpiryItems[key]
	if item == nil {
		return
	}
	heap.Remove(&s.entryExpiry, item.index)
	delete(s.entryExpiryItems, key)
}

func (s *CustomSCPStore) markEntryForCleanupLocked(key customSCPKey, entry *customSCPEntry) {
	if s == nil {
		return
	}
	s.refreshEntryAgesLocked(entry)
	s.upsertEntryExpiryLocked(key, entry)
}

func (s *CustomSCPStore) popDueEntryExpiryLocked(observationCutoff int64) (customSCPKey, *customSCPEntry, bool) {
	for s != nil && len(s.entryExpiry) > 0 {
		item := s.entryExpiry[0]
		entry := s.entries[item.key]
		if entry == nil {
			heap.Pop(&s.entryExpiry)
			delete(s.entryExpiryItems, item.key)
			continue
		}
		if item.dueUnix != s.entryCleanupDueUnix(entry) {
			s.upsertEntryExpiryLocked(item.key, entry)
			continue
		}
		if item.dueUnix >= observationCutoff {
			return customSCPKey{}, nil, false
		}
		heap.Pop(&s.entryExpiry)
		delete(s.entryExpiryItems, item.key)
		return item.key, entry, true
	}
	return customSCPKey{}, nil, false
}

func (s *CustomSCPStore) upsertStaticExpiryLocked(call string, seenUnix int64) {
	if s == nil || call == "" || seenUnix <= 0 {
		s.deleteStaticExpiryLocked(call)
		return
	}
	if s.staticExpiryItems == nil {
		s.staticExpiryItems = make(map[string]*customSCPStaticExpiryItem, 16)
	}
	if item := s.staticExpiryItems[call]; item != nil {
		item.dueUnix = seenUnix
		heap.Fix(&s.staticExpiry, item.index)
		return
	}
	item := &customSCPStaticExpiryItem{call: call, dueUnix: seenUnix}
	heap.Push(&s.staticExpiry, item)
	s.staticExpiryItems[call] = item
}

func (s *CustomSCPStore) deleteStaticExpiryLocked(call string) {
	if s == nil || call == "" {
		return
	}
	item := s.staticExpiryItems[call]
	if item == nil {
		return
	}
	heap.Remove(&s.staticExpiry, item.index)
	delete(s.staticExpiryItems, call)
}

func (s *CustomSCPStore) popDueStaticExpiryLocked(staticCutoff int64) (string, bool) {
	for s != nil && len(s.staticExpiry) > 0 {
		item := s.staticExpiry[0]
		seen := s.static[item.call]
		if seen <= 0 {
			heap.Pop(&s.staticExpiry)
			delete(s.staticExpiryItems, item.call)
			continue
		}
		if item.dueUnix != seen {
			s.upsertStaticExpiryLocked(item.call, seen)
			continue
		}
		if item.dueUnix >= staticCutoff {
			return "", false
		}
		heap.Pop(&s.staticExpiry)
		delete(s.staticExpiryItems, item.call)
		return item.call, true
	}
	return "", false
}
