# ADR-0059 - Slot-Aware FT Confidence Timing and RBN Arrival Fallback

Status: Superseded
Date: 2026-04-08
Decision Makers: Cluster maintainers
Technical Area: main output pipeline, FT confidence, docs
Decision Origin: Troubleshooting chat
Troubleshooting Record(s): TSR-0016
Tags: ft8, ft4, ft2, confidence, timing, output-pipeline

Superseded by: ADR-0060

## Context

- ADR-0057 introduced bounded FT corroboration in the main output pipeline using a fixed arrival-based hold of about 2 seconds.
- ADR-0058 fixed PSKReporter FT frequency semantics by canonicalizing operational frequency to dial values while preserving the observed RF frequency separately.
- Live validation after ADR-0058 showed that the canonical-frequency fix alone did not produce practical FT `P`/`V` tagging:
  - corroborating FT groups existed on the same canonical dial frequency,
  - but their reports arrived over much longer spans than 2 seconds,
  - so the arrival-only hold released groups too early and left live output almost entirely at `?`/`S`.
- RBN digital FT8/FT4 spots only carry `HHMM` source timestamps, so their parsed source time lacks second-level precision and cannot safely identify a transmission slot by itself.

## Decision

- Keep the bounded main-loop FT corroboration architecture from ADR-0057.
- Replace the fixed arrival-based hold with slot-aware grouping and release:
  - key FT corroboration by normalized DX call, exact FT mode, canonical dial frequency, and transmission slot
  - derive the slot from source-observed time when the source timestamp is precise enough
  - for RBN FT8/FT4, derive the slot from arrival time instead, because the source timestamp has only minute resolution
- Use mode-specific timing:
  - FT8: 15 second slot, release at slot end + 6 seconds grace
  - FT4: 7.5 second slot, release at slot end + 3 seconds grace
  - FT2: 3.8 second slot, release at slot end + 2 seconds grace
- Preserve existing FT glyph meanings and floors:
  - `?` = one unique reporter without static/recent promotion
  - `S` = one unique reporter with static/recent promotion
  - `P` = exactly two unique reporters in the same slot
  - `V` = three or more unique reporters in the same slot
- Preserve fail-open behavior, but raise pending-state bounds enough to cover the longer slot windows under the expected FT ingest rate.

## Alternatives Considered

1. Increase the existing arrival hold to a larger fixed constant.
   - Rejected because it still groups by first arrival instead of transmission cadence.
2. Keep the 2 second hold and accept sparse live `P`/`V`.
   - Rejected because it does not satisfy the corroboration goal after frequency semantics were fixed.
3. Add a separate per-mode worker or goroutine for FT timing.
   - Rejected because the existing single-owner output-pipeline model is simpler and already has deterministic release semantics.

## Consequences

### Benefits
- Live FT `P`/`V` formation now tracks actual transmission slots rather than arbitrary arrival jitter.
- Canonical dial frequency from ADR-0058 and slot timing now work together, addressing the two main causes of FT corroboration fragmentation.
- Archive, telnet, peer, and ring-buffer outputs still see the same final FT glyph.

### Risks
- FT output latency increases materially compared with the earlier 2 second hold.
- Pending FT state remains longer in memory, so bounds must stay explicit and overload must continue to fail open.
- FT2 timing confidence is lower than FT4/FT8 because the project ecosystem is smaller and source timing evidence is less mature.

### Operational impact
- Slow clients remain unaffected because the slot-aware hold still happens before fan-out queues.
- Overload still degrades by reducing promotion quality rather than blocking the main loop.
- No new goroutines, network deadlines, or shutdown phases are introduced; forced drain remains the shutdown escape hatch.

## Validation

- Added/updated tests:
  - `ft_confidence_runtime_test.go`
- Commands:
  - `go test . -run "TestOutputPipelineFT|TestFTConfidenceController|TestBuildFTConfidence"`
  - `go test . -run ^$ -bench "Benchmark(OutputPipelineFTHoldAndRelease|FTConfidenceControllerObserveAndDrain)$" -benchmem`
  - `go test ./...`
  - `go vet ./...`
  - `staticcheck ./...`
  - `go test -race ./...`

## Rollout and Reversal

- Rollout plan:
  - Deploy normally; no config migration is required.
- Backward compatibility impact:
  - FT2/FT4/FT8 now wait for slot-end grace rather than a fixed ~2 second hold.
  - FT `P`/`V` semantics now explicitly mean corroboration within the same transmission slot.
- Reversal plan:
  - Revert the slot-aware timing change and restore the previous arrival-only FT hold.

## References

- Related ADR(s):
  - `docs/decisions/ADR-0055-ft2-explicit-mode-support-and-filter-contract.md`
  - `docs/decisions/ADR-0057-ft-confidence-corroboration.md`
  - `docs/decisions/ADR-0058-pskreporter-ft-frequency-canonicalization.md`
- Troubleshooting Record(s):
  - `docs/troubleshooting/TSR-0016-ft-confidence-arrival-window-misses-live-corroboration.md`
- Docs:
  - `README.md`
