# ADR-0057 - FT Confidence Uses Bounded Main-Loop Corroboration

Status: Superseded
Date: 2026-04-07
Decision Makers: Cluster maintainers
Technical Area: main output pipeline, filter, custom_scp, docs
Decision Origin: Design
Tags: ft8, ft4, ft2, confidence, custom_scp, output-pipeline

## Context

- FT spots are digitally decoded and do not need the existing resolver/call-correction path.
- Operators still want FT confidence glyphs that distinguish single-reporter observations from corroborated multi-reporter bursts.
- FT ingest volume can reach tens of thousands of spots per minute, so any solution must keep memory bounded and fail open under pressure.
- Existing `S` semantics and `custom_scp` admission rules already matter operationally:
  - `S` means one reporter plus static or recent support.
  - `custom_scp` admission is intentionally `V`-only to avoid self-reinforcing weaker confidence classes.

## Decision

- Keep FT2/FT4/FT8 outside resolver call mutation and telnet stabilizer semantics.
- Add a bounded FT-specific corroboration stage inside the single-owner main output pipeline:
  - hold FT spots before archive/telnet/peer fan-out
  - group corroboration by normalized DX call and exact FT mode, with timing/key details later refined by ADR-0059
  - fail open when pending bounds are exceeded by releasing the current spot immediately with the best-known count-only glyph
- Use FT confidence glyph semantics:
  - `?` = one unique reporter in the bounded corroboration window and no static/recent support promotion
  - `S` = one unique reporter plus static known-calls/custom-SCP membership or recent on-band support
  - `P` = exactly two unique reporters in the bounded corroboration window
  - `V` = three or more unique reporters in the bounded corroboration window
- Reuse recent-on-band floor semantics for FT via a dedicated FT recent-support store, independent of resolver/stabilizer recent-support state.
- Extend `custom_scp` mode bucketing to record exact FT2/FT4/FT8 modes, but preserve the existing `V`-only admission contract.
- Enable confidence filtering for FT2/FT4/FT8 now that those modes emit glyphs.

## Alternatives Considered

1. Reuse resolver/call-correction for FT.
   - Rejected because FT decodes should not enter call mutation logic.
2. Reuse the telnet stabilizer for FT corroboration.
   - Rejected because stabilizer only affects telnet output and would desynchronize archive/telnet/peer glyphs.
3. Emit FT glyphs immediately without a hold.
   - Rejected because the first spot in a burst would under-report corroboration too often.

## Consequences

### Positive outcomes
- FT glyphs become available without introducing FT call correction.
- Archive, telnet, peer, and ring-buffer outputs see the same FT confidence result.
- `S` semantics stay consistent with existing operator expectations.
- `custom_scp` can learn from corroborated FT `V` spots without admitting weaker FT classes.

### Negative outcomes / risks
- FT output now has a bounded added latency, with precise timing later refined by ADR-0059.
- Under extreme FT burst pressure, some spots will fail open and skip the hold promotion window.
- FT corroboration keys are deliberately coarse, so very rare adjacent observations near a bucket edge may under-group or over-group slightly.

### Operational impact
- Slow clients remain unaffected because the FT hold happens before fan-out queues.
- Overload degrades by reducing FT promotion quality rather than blocking the pipeline.
- No new goroutines are added; FT pending state remains owned by the existing output-pipeline goroutine.

## Validation

- Added/updated tests:
  - `ft_confidence_runtime_test.go`
  - `main_test.go`
  - `spot/custom_scp_store_test.go`
  - `filter/filter_test.go`
- Commands:
  - `go test . ./spot ./filter ./commands -run "TestOutputPipelineFT|TestFTConfidenceController|TestApplyKnownCallFloor|TestCustomSCPStoreRecordSpot|TestFT8ConfidenceFilterNoLongerExempt|TestHelpPerDialect"`
  - `go test . -run ^$ -bench "Benchmark(OutputPipelineFTHoldAndRelease|FTConfidenceControllerObserveAndDrain)$" -benchmem`
  - `go test ./...`
  - `go vet ./...`
  - `staticcheck ./...`
  - `go test -race ./...`

## Rollout and Reversal

- Rollout plan:
  - Deploy normally; no config migration is required.
- Backward compatibility impact:
  - FT2/FT4/FT8 now emit confidence glyphs and are subject to confidence filtering.
  - FT spots can be delayed by about 2 seconds before fan-out.
- Reversal plan:
  - Remove the FT corroboration controller, restore FT confidence filtering exemption, and keep FT out of `custom_scp` buckets.

## References

- Related ADR(s):
  - `docs/decisions/ADR-0010-s-glyph-recent-on-band-floor.md`
  - `docs/decisions/ADR-0039-custom-scp-runtime-evidence-and-shared-pebble-resilience.md`
  - `docs/decisions/ADR-0043-custom-scp-v-only-admission.md`
  - `docs/decisions/ADR-0046-stabilizer-delay-eligibility-unknown-s-p-only.md`
  - `docs/decisions/ADR-0055-ft2-explicit-mode-support-and-filter-contract.md`
  - `docs/decisions/ADR-0056-local-self-spots-operator-authoritative-v-path.md`
  - `docs/decisions/ADR-0059-slot-aware-ft-confidence-timing.md`
- Troubleshooting Record(s): none
- Docs:
  - `README.md`
  - `docs/decision-log.md`
