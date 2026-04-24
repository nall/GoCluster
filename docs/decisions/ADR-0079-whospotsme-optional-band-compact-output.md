# ADR-0079: WHOSPOTSME Optional Band and Compact Output

- Status: Accepted
- Date: 2026-04-24
- Decision Origin: Design
- Supersedes: ADR-0071 operator-visible command/output contract

## Context

ADR-0071 introduced `WHOSPOTSME <band>` as a rolling country summary keyed by
logged-in callsign plus band. The retained-state, admission, rolling-window, and
counting decisions remain valid.

The operator command needed one usability refinement: when no band is supplied,
operators should see all bands that have rolling-window data. The selected-band
form should use the same compact row policy so output is consistent.

## Decision

`WHOSPOTSME` now accepts an optional band:

- `WHOSPOTSME <band>` shows only the selected band;
- `WHOSPOTSME` scans canonical supported bands and shows only bands that have
  rolling-window observations for the logged-in callsign;
- both forms omit continents with no spots;
- if no data exists for the selected scope, the command returns a single
  `no data` response.

The existing retained-state model is unchanged:

- observations are still recorded on the accepted-spot path before secondary
  broadcast dedupe;
- counts remain accepted-observation totals, not unique spotters;
- the rolling window remains YAML-owned under `who_spots_me.window_minutes`;
- the store remains keyed by normalized DX callsign plus normalized band;
- no new retained index by call or band is added.

## Alternatives considered

1. Keep selected-band output with empty continent rows
   - Rejected because it makes `WHOSPOTSME <band>` visually inconsistent with
     bare `WHOSPOTSME`.
2. Add a retained call-to-band index
   - Rejected because the canonical band list is small and bounded, so query
     scanning avoids new lifetime state and cleanup coupling.
3. Render every supported band when no band is supplied
   - Rejected because empty band sections add noise and do not answer which
     bands have recent observations.

## Consequences

- Operators can now use `WHOSPOTSME` as a compact all-band recent-heard summary.
- `WHOSPOTSME <band>` no longer shows explicit `(no data)` rows for empty
  continents.
- Bare command cost is bounded by the number of supported bands, not retained
  key cardinality.
- ADR-0071 remains the authority for retained-state architecture, admission
  timing, counting semantics, and config ownership.

## Links

- Superseded operator contract: `docs/decisions/ADR-0071-whospotsme-rolling-country-summary.md`
- Related tests: `commands/processor_test.go`
