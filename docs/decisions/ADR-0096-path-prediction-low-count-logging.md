# ADR-0096: Path Prediction Low Count Logging

- Status: Accepted
- Date: 2026-04-30
- Decision Origin: Design

## Context
Path prediction already distinguishes `InsufficientLowCount` from
`InsufficientLowWeight` in `pathreliability.Result`. The five-minute system log
and propagation report aggregation only exposed `low_weight`, so low-count
insufficient predictions were indistinguishable from true low-weight
predictions in operator telemetry.

## Decision
Expose `low_count` separately in path prediction statistics, five-minute system
logs, and propagation reports. Keep existing prediction math, thresholds, path
classes, receiver-cap behavior, and PATH filtering unchanged.

Historical propagation-report parsing remains compatible with logs that do not
include `low_count`; those records parse as `low_count=0`.

## Alternatives considered
1. Keep using `low_weight` for both low-count and low-weight failures. Rejected
   because it hides whether `min_observation_count` or `min_effective_weight`
   is the binding gate.
2. Rename `low_weight` to a generic `low_evidence`. Rejected because it would
   be less precise and would break existing log/report consumers.
3. Add only `SET DIAG PATH` examples. Rejected because the operator question is
   about aggregate log behavior, not individual spot diagnostics.

## Consequences
### Benefits
- Operators can tell whether high insufficient rates are count-limited or
  weight-limited.
- Propagation reports preserve the same distinction in hourly summaries.
- The change is additive for new logs and compatible with historical logs.

### Risks
- Consumers that parse the whole `Path predictions (5m)` line positionally
  must tolerate the new `low_count` token.
- Historical reports before this change cannot reconstruct low-count counts;
  they remain under the old `low_weight` aggregate.

### Operational impact
- `Path predictions (5m)` includes `low_count=<n>` before `low_weight=<n>`.
- Propagation report JSON includes `avg_low_count`.
- Path glyphs, filters, config, retained state, slow-client handling,
  broadcast queues, reconnect handling, and shutdown behavior are unchanged.

## Links
- Related issues/PRs/commits:
- Related tests: `telnet/server_prediction_stats_test.go`, `internal/propreport/report_test.go`
- Related docs: `README.md`, `docs/OPERATOR_GUIDE.md`, `data/config/PATH_PREDICTIONS.md`, `pathreliability/README.md`, `customgpt/common-questions.md`, `customgpt/operator-guide-index.md`
- Related TSRs:
- Supersedes / superseded by:
