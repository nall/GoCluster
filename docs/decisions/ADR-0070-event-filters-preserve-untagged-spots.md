# ADR-0070: EVENT Filters Preserve Untagged Spots

- Status: Accepted
- Date: 2026-04-23
- Decision Origin: Design

## Context

ADR-0068 introduced EVENT metadata and PASS/REJECT EVENT filters for activation
families such as POTA, SOTA, IOTA, LLOTA, and WWFF. The initial filter contract
treated EVENT as a spot-level gate: `PASS EVENT POTA` hid spots without EVENT
tags, and `REJECT EVENT ALL` rejected every spot.

That behavior made EVENT filters too broad for normal cluster use. Operators use
EVENT filters to include or exclude tagged activation spots, not to suppress
ordinary spots that simply do not mention an activation family.

## Decision

EVENT filters apply only to spots with recognized EVENT metadata. Spots with no
EVENT tag always pass the EVENT filter domain.

Tagged spots keep the existing deny-first allow/block semantics:

- `PASS EVENT <list>` allows tagged spots with at least one listed family.
- `REJECT EVENT <list>` rejects tagged spots with any listed family.
- If a tagged spot matches both allow and reject families, reject wins.
- `PASS EVENT ALL` allows every tagged EVENT family.
- `REJECT EVENT ALL` rejects every tagged EVENT family.

Other filter domains still apply to untagged spots. This decision only changes
the EVENT domain.

## Alternatives considered

1. Keep ADR-0068 spot-level EVENT filtering.
   - Rejected because explicit EVENT filters would continue hiding ordinary
     untagged spots.
2. Preserve untagged spots only for `PASS EVENT <list>`.
   - Rejected because `REJECT EVENT ALL` would still be surprising as a global
     spot suppressor.
3. Add a separate operator knob for untagged EVENT behavior.
   - Rejected because the desired semantics are simple and do not require a new
     persisted setting.

## Consequences

- `REJECT EVENT ALL` now means all EVENT-tagged spots are rejected; untagged
  spots still pass the EVENT domain.
- Per-user filter YAML schema does not change. Existing `blockallevents: true`
  records keep loading, but the runtime meaning narrows to tagged spots.
- Telnet `SHOW FILTER` and command responses must say that no-event spots pass.
- Live fan-out remains bounded and allocation-free; the matcher checks the
  fixed EVENT bitmask before map lookups.

## Links

- Supersedes: ADR-0068
- Related tests:
  - `filter/filter_test.go`
  - `telnet/server_filter_test.go`
- Related docs:
  - `README.md`
  - `telnet/README.md`
