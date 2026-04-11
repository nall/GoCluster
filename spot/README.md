# Spot Confidence And Correction

This directory owns the canonical spot record, fixed-width spot formatting, call-correction helpers, harmonics, frequency averaging, recent-support stores, and custom SCP persistence.

## Confidence Paths

The runtime has two separate confidence paths.

### Resolver-capable modes

Resolver-capable modes are:

- `CW`
- `RTTY`
- `USB`
- `LSB`

The live mapping is shared through [`../internal/correctionflow/shared.go`](../internal/correctionflow/shared.go).

For resolver-capable modes:

- the emitted call gets a call-specific confidence percent
- weighted support is used when available
- plain reporter support is the fallback
- `FormatConfidence` maps:
  - one reporter to `?`
  - multi-reporter confidence `<= 50%` to `P`
  - multi-reporter confidence `>= 51%` to `V`

Contested resolver states are intentionally conservative:

- `split` or `uncertain` become `?` for one-reporter cases
- `split` or `uncertain` become `P` for multi-reporter cases

### The `S` floor

`S` is not produced by `FormatConfidence`.

Instead, after the main confidence path runs, the runtime promotes `?` to `S` when:

- the call is in static support, or
- the call has recent on-band support

That floor only applies to confidence-capable modes and does not overwrite non-`?` results.

### Correction outcomes

Two glyphs represent correction outcomes rather than raw confidence:

- `C`: correction applied and validated
- `B`: correction attempted, but the candidate failed base-call or CTY validation, so the original call stayed on the spot

### FT2, FT4, And FT8

FT modes use a separate corroboration stage in the main output pipeline.

The burst key is:

- normalized DX call
- exact FT mode
- canonical dial frequency

Burst timing is controlled by the `call_correction` FT knobs in the shipped config:

- `p_min_unique_spotters`
- `v_min_unique_spotters`
- per-mode quiet-gap seconds
- per-mode hard-cap seconds

In the shipped repo config these live under `call_correction` in [`../data/config/pipeline.yaml`](../data/config/pipeline.yaml):

- `call_correction.p_min_unique_spotters`
- `call_correction.v_min_unique_spotters`
- `call_correction.ft8_quiet_gap_seconds`
- `call_correction.ft8_hard_cap_seconds`
- `call_correction.ft4_quiet_gap_seconds`
- `call_correction.ft4_hard_cap_seconds`
- `call_correction.ft2_quiet_gap_seconds`
- `call_correction.ft2_hard_cap_seconds`

With the current shipped defaults:

- `P` = exactly 2 unique reporters
- `V` = 3 or more unique reporters
- FT8 = `6s` quiet gap, `12s` hard cap
- FT4 = `5s` quiet gap, `10s` hard cap
- FT2 = `3s` quiet gap, `6s` hard cap

The runtime can corroborate across PSKReporter and RBN-digital if the burst key matches and the spotter calls differ.

### Special operator path

Local non-test `DX` self-spots are treated as operator-authoritative in the live runtime pipeline:

- they are forced to `V`
- they bypass resolver, temporal, and stabilizer delay in that runtime path

## Related Stores And Policy

- recent on-band support helps the `S` floor and some resolver gates
- custom SCP stores persistent support evidence
- custom SCP retention is bounded by `history_horizon_days`, `max_keys`, and `max_spotters_per_key`
- custom SCP static membership now ages out on the same horizon as recent evidence
- custom SCP admission is intentionally `V`-only to avoid reinforcement loops from weaker classes

## Related Files

- `spot.go` for the spot record and fixed-width formatter
- `correction.go` for correction gates and confidence inputs
- `recent_band.go` for bounded recent-support state
- `custom_scp_store.go` for persistent support storage

For the main operator summary, see [`../README.md`](../README.md). For parser-specific input behavior, see [`../rbn/README.md`](../rbn/README.md) and [`../pskreporter/README.md`](../pskreporter/README.md).
