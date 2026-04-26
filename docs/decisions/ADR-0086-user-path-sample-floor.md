# ADR-0086: User Path Sample Floor

- Status: Accepted
- Date: 2026-04-26
- Decision Origin: Design

## Context
ADR-0085 added a cluster-wide `min_observation_count` gate before path
reliability can emit a usable tag. Some telnet users may want a stricter
personal evidence floor before they see path tags, but they must not be able to
weaken the cluster's trust floor.

## Decision
Add `SET PATHSAMPLES <count|DEFAULT>` as a per-callsign telnet setting.

Numeric values are accepted only when they are greater than the cluster default
from `path_reliability.yaml:min_observation_count`. The effective prediction
floor is therefore:

```text
max(cluster min_observation_count, user path_min_observation_count)
```

`SET PATHSAMPLES DEFAULT` clears the user override. The setting is persisted in
the existing user record and affects displayed path glyphs, `PATH` filter class
evaluation, and `SET DIAG PATH` low-count diagnostics. It does not change
ingestion, retained path buckets, decay, freshness, solar overrides, or global
YAML defaults.

## Alternatives considered
1. Allow users to lower the floor. Rejected because it would bypass the
   operator-owned cluster trust floor.
2. Add a YAML min/max range for user overrides. Deferred because the current
   requirement is stricter-only and a fixed high input cap is enough to reject
   accidental nonsense.
3. Implement as a `PASS PATH` option. Rejected because sample gating is not a
   PATH class allowlist and should not be confused with `PASS/REJECT PATH`.

## Consequences
### Benefits
- Users can make path tags more conservative for their own sessions.
- Cluster operators retain the global minimum evidence policy.
- Display, filtering, and diagnostics use the same effective count gate.

### Risks
- Users with high overrides will see more `INSUFFICIENT` path results.
- Persisted user records gain one optional scalar.
- Historical comparisons for users who enable the override will show fewer
  usable path glyphs than the cluster default view.

### Operational impact
- Slow-client handling, broadcast queues, reconnects, shutdown, and retained
  path bucket cardinality are unchanged.
- Existing users keep cluster-default behavior until they set an override.

## Links
- Related issues/PRs/commits:
- Related tests: `telnet/path_settings_test.go`, `filter/user_record_test.go`, `pathreliability/normalize_test.go`
- Related docs: `README.md`, `telnet/README.md`, `pathreliability/README.md`
- Related TSRs:
- Supersedes / superseded by:
