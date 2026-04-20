package peer

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	overlongOnce sync.Once
	overlongCh   chan overlongSample
)

type overlongSample struct {
	path    string
	host    string
	preview string
	length  int
	reason  string
	limit   int
	ts      time.Time
}

type overlongSummaryState struct {
	mu       sync.Mutex
	lastEmit time.Time
	reasons  map[string]uint64
}

var overlongSummary overlongSummaryState

const (
	overlongSummaryInterval = 30 * time.Second
	overlongMaxLogSizeBytes = 8 << 20
	overlongBackupFiles     = 2
)

// Purpose: Enqueue a preview of an overlong line for diagnostic logging.
// Key aspects: Truncates previews and drops when the queue is full.
// Upstream: Peer reader when a line exceeds max length.
// Downstream: overlongWorker goroutine.
func appendOverlongSample(path, host, preview string, length int, reason string, limit int) {
	preview = strings.TrimSpace(preview)
	if preview == "" {
		return
	}
	const maxPreview = 512
	if len(preview) > maxPreview {
		preview = preview[:maxPreview]
	}
	overlongOnce.Do(func() {
		overlongCh = make(chan overlongSample, 256)
		// Goroutine: write overlong samples to disk without blocking readers.
		go overlongWorker()
	})
	if overlongCh == nil {
		return
	}
	sample := overlongSample{
		path:    path,
		host:    strings.TrimSpace(host),
		preview: preview,
		length:  length,
		reason:  normalizeOverlongReason(reason),
		limit:   limit,
		ts:      time.Now().UTC(),
	}
	recordOverlongDrop(sample.reason, sample.limit)
	// Best-effort: drop if the queue is full so the read loop never blocks.
	select {
	case overlongCh <- sample:
	default:
	}
}

// Purpose: Persist overlong line samples to disk.
// Key aspects: Best-effort; skips on file errors to avoid backpressure.
// Upstream: appendOverlongSample goroutine.
// Downstream: os.OpenFile, f.WriteString.
func overlongWorker() {
	for sample := range overlongCh {
		if sample.preview == "" {
			continue
		}
		if dir := filepath.Dir(sample.path); dir != "" && dir != "." {
			if err := os.MkdirAll(dir, 0o755); err != nil {
				continue
			}
		}
		rotateOverlongLog(sample.path, overlongMaxLogSizeBytes, overlongBackupFiles)
		f, err := os.OpenFile(sample.path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
		if err != nil {
			continue
		}
		ts := sample.ts.Format(time.RFC3339)
		line := fmt.Sprintf(
			"%s host=%s reason=%s limit=%d len=%d preview=%s\n",
			ts,
			sample.host,
			sample.reason,
			sample.limit,
			sample.length,
			sample.preview,
		)
		if _, err := f.WriteString(line); err != nil {
			_ = f.Close()
			continue
		}
		_ = f.Close()
	}
}

func normalizeOverlongReason(reason string) string {
	reason = strings.TrimSpace(strings.ToLower(reason))
	if reason == "" {
		return "unknown"
	}
	return reason
}

func recordOverlongDrop(reason string, limit int) {
	reason = normalizeOverlongReason(reason)
	if limit < 0 {
		limit = 0
	}
	now := time.Now().UTC()

	overlongSummary.mu.Lock()
	defer overlongSummary.mu.Unlock()
	if overlongSummary.reasons == nil {
		overlongSummary.reasons = make(map[string]uint64, 4)
	}
	key := reason + ":" + strconv.Itoa(limit)
	overlongSummary.reasons[key]++
	if overlongSummary.lastEmit.IsZero() {
		overlongSummary.lastEmit = now
		return
	}
	if now.Sub(overlongSummary.lastEmit) < overlongSummaryInterval {
		return
	}

	keys := make([]string, 0, len(overlongSummary.reasons))
	for k := range overlongSummary.reasons {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	total := uint64(0)
	for _, k := range keys {
		v := overlongSummary.reasons[k]
		total += v
		parts = append(parts, fmt.Sprintf("%s=%d", k, v))
	}
	log.Printf("Peering: overlong drops summary interval=%s total=%d %s", overlongSummaryInterval, total, strings.Join(parts, " "))
	overlongSummary.reasons = make(map[string]uint64, len(overlongSummary.reasons))
	overlongSummary.lastEmit = now
}

func rotateOverlongLog(path string, maxBytes int64, keep int) {
	if strings.TrimSpace(path) == "" || maxBytes <= 0 || keep <= 0 {
		return
	}
	info, err := os.Stat(path)
	if err != nil {
		return
	}
	if info.Size() < maxBytes {
		return
	}
	oldest := path + "." + strconv.Itoa(keep)
	_ = os.Remove(oldest)
	for i := keep - 1; i >= 1; i-- {
		src := path + "." + strconv.Itoa(i)
		dst := path + "." + strconv.Itoa(i+1)
		ignoreOverlongRotateError(os.Rename(src, dst))
	}
	ignoreOverlongRotateError(os.Rename(path, path+".1"))
}

func ignoreOverlongRotateError(error) {
	// Overlong sample rotation is diagnostic-only and must never block or
	// fail peer read paths. Future samples can retry rotation.
}
