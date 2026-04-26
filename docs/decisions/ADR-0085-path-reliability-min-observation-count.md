# ADR-0085: Path Reliability Minimum Observation Count

- Status: Accepted
- Date: 2026-04-26
- Decision Origin: Design

## Context
Path reliability already gates usable path tags by selected evidence freshness
and decayed effective weight. `SET DIAG PATH` now exposes the raw selected
observation count, which made it clear that a tag can be emitted from a very
small sample when weight is high enough.

For troubleshooting and operator trust, sample size should be an explicit
eligibility gate. Weight answers how much usable evidence remains after decay
and discounts; raw count answers whether there was enough independent evidence
to label the path at all.

## Decision
Add required YAML setting
`data/config/path_reliability.yaml:min_observation_count` with shipped value
`19`.

The path predictor returns a usable path class only when selected evidence
meets all of these gates:

- selected evidence exists
- selected evidence is fresh enough for the display/filter freshness gate
- selected raw observation count is at least `min_observation_count`
- selected decayed effective weight is at least `min_effective_weight`

Evidence below the count gate returns `INSUFFICIENT` with reason
`low_count`. `SET DIAG PATH` renders that reason compactly as `lown`, for
example `n3|lown`.

Freshness remains higher precedence than low count. If stale evidence was
dropped and surviving evidence is also below count, the result remains reported
as stale so operators see the age problem that hid the stronger evidence.

## Alternatives considered
1. Keep the existing weight-only gate.
   Rejected because high-weight low-count evidence can look more reliable than
   its sample size supports.
2. Use count only in diagnostics without changing path emission.
   Rejected because it would reveal the problem but still emit the same path
   tags operators are trying to troubleshoot.
3. Add per-band or per-mode count thresholds immediately.
   Deferred because the current operator requirement is a simple global floor;
   per-band tuning can be added later if field data shows different sample-size
   requirements.
4. Fold count into effective weight.
   Rejected because weight and count are intentionally different evidence
   dimensions and are more debuggable when kept separate.

## Consequences
### Benefits
- Path tags require a minimum raw evidence base before they are emitted.
- `SET DIAG PATH` can distinguish sparse evidence (`lown`) from weak decayed
  evidence (`loww`) and stale evidence (`stale`).
- Operators can tune one YAML-owned floor without changing decay, weight, or
  glyph thresholds.

### Risks
- Quiet bands, uncommon paths, or startup periods will show more
  `INSUFFICIENT` tags until at least 19 selected observations accumulate.
- Historical comparisons across this change will show fewer usable path glyphs
  for sparse paths.
- A single global count floor may be conservative for some high-volume bands
  and aggressive for low-volume bands.

### Validation
- Config loading requires `min_observation_count` and rejects non-positive
  values.
- Predictor tests cover low-count evidence returning `INSUFFICIENT`.
- Operator docs describe `lown` and the configured minimum-count gate.

