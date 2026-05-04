package main

import (
	"container/heap"
	"time"

	"dxcluster/internal/correctionflow"
	"dxcluster/spot"
)

// replayTemporalPending preserves the resolver evidence and selected snapshot
// for a held observation so lag-release scoring can use the same inputs runtime
// saw at hold time.
type replayTemporalPending struct {
	id          uint64
	spot        *spot.Spot
	evidence    spot.ResolverEvidence
	hasEvidence bool
	maxAt       time.Time
	selection   correctionflow.ResolverPrimarySelection
}

// replayTemporalItem orders pending temporal observations by release time and
// sequence for deterministic replay artifacts.
type replayTemporalItem struct {
	id  uint64
	due time.Time
	seq uint64
}

type replayTemporalHeap []*replayTemporalItem

func (h replayTemporalHeap) Len() int { return len(h) }

func (h replayTemporalHeap) Less(i, j int) bool {
	if h[i].due.Equal(h[j].due) {
		return h[i].seq < h[j].seq
	}
	return h[i].due.Before(h[j].due)
}

func (h replayTemporalHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
}

func (h *replayTemporalHeap) Push(x any) {
	item, ok := x.(*replayTemporalItem)
	if !ok {
		return
	}
	*h = append(*h, item)
}

func (h *replayTemporalHeap) Pop() any {
	old := *h
	n := len(old)
	if n == 0 {
		return nil
	}
	item := old[n-1]
	*h = old[:n-1]
	return item
}

// popReplayTemporalDue returns every item due at the replay timestamp. Stale or
// missing pending IDs are handled by the caller because the decoder owns the
// final decision state.
func popReplayTemporalDue(h *replayTemporalHeap, now time.Time) []*replayTemporalItem {
	if h == nil || h.Len() == 0 {
		return nil
	}
	out := make([]*replayTemporalItem, 0, 8)
	for h.Len() > 0 {
		head := (*h)[0]
		if head == nil || head.due.After(now) {
			break
		}
		itemAny := heap.Pop(h)
		if item, ok := itemAny.(*replayTemporalItem); ok && item != nil {
			out = append(out, item)
		}
	}
	return out
}
