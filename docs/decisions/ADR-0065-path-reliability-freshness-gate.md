# ADR-0065: Path Reliability Freshness Gate

- Status: Accepted
- Date: 2026-04-20
- Decision Origin: Design

## Context
Path reliability stores recent observed SNR in decaying buckets. The bucket
power sum and weight decay together, so the estimated SNR stays at the observed
mean while confidence falls. That is mathematically correct for an empirical
mean, but it can leave a strong old opening visible until weight falls below
`min_effective_weight` or the bucket is purged.

HF band state can change abruptly at sunrise, sunset, MUF transitions, flare
recovery, and short high-band openings. In those cases, old positive evidence is
not a weaker path; it is stale state evidence.

## Decision
Path reliability applies a band-scaled freshness gate after fine/coarse sample
selection and before receive/transmit merge.

The YAML knob is:

```yaml
max_prediction_age_half_life_multiplier: 1.25
```

The maximum age is `ceil(band_half_life * multiplier)`. A value of `0` disables
the gate. Negative configured values normalize to `0`.

If selected receive or transmit evidence is older than the maximum age, that
direction is discarded. If no eligible evidence remains, the result is
`INSUFFICIENT` with reason `stale`. If stale evidence is dropped and only
underweight fresh evidence remains, the result is also reported as stale so the
operator can distinguish state freshness from ordinary low confidence.

Fine/coarse blended samples use weighted effective age, not youngest age. This
prevents a small fresh sample from hiding a large stale regional contribution.

## Alternatives considered
1. Decay the stored SNR value toward zero or a weak prior. Rejected because SNR
   should remain the observed conditional mean; confidence should decay, not the
   measured value.
2. Blend observed mean with a propagation prior as data ages. Deferred because
   the approved scope uses only ingested cluster data and no external model.
3. Rely only on existing bucket decay and purge. Rejected because it can keep
   stale high-SNR evidence visible through fast HF state transitions.
4. Add separate enable and age knobs. Rejected; `0` on the multiplier is the
   single disable path.

## Consequences
### Benefits
- Stale positive evidence becomes `INSUFFICIENT` instead of misleading weaker
  glyph tiers.
- The age threshold remains band-scaled by existing half-life tuning.
- No new retained bucket state, maps, caches, or indexes are added.
- PATH filters and displayed glyphs share the same freshness behavior.
- Five-minute path logs and propagation reports expose stale counts separately
  from no-sample and low-weight cases.

### Risks
- Quiet periods will show more `INSUFFICIENT` glyphs.
- Operators comparing historical logs across this boundary will see fewer stale
  positive glyphs and a new `stale=` prediction field.
- A hard cutoff can hide a path that remains open but has not produced recent
  decodes. That is the intended conservative tradeoff.

### Operational impact
- Telnet glyph slot and command syntax are unchanged.
- `PATH` filters treat stale predictions as `INSUFFICIENT`.
- `Path predictions (5m)` now includes `stale=<n>`.
- Persisted user records are unchanged.
- Slow clients, broadcast queues, reconnect handling, and shutdown behavior are
  unchanged.

## Links
- Related issues/PRs/commits:
- Related tests: `pathreliability/normalize_test.go`, `pathreliability/config_test.go`, `telnet/server_prediction_stats_test.go`, `internal/propreport/report_test.go`
- Related docs: `data/config/path_reliability.yaml`, `pathreliability/README.md`, `data/config/PATH_PREDICTIONS.md`, `docs/OPERATOR_GUIDE.md`
- Related TSRs:
- Supersedes / superseded by:
