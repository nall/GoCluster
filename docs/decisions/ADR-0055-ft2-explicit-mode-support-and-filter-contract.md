# ADR-0055 - FT2 Explicit Mode Support and Filter Contract

Status: Accepted
Date: 2026-03-27
Decision Makers: Maintainers
Technical Area: spot, pskreporter, filter, telnet, commands, docs
Decision Origin: Troubleshooting chat
Troubleshooting Record(s): TSR-0013
Tags: mode-token, parser, filter-contract, operator-surface

## Context
- Operators observed `FT2` in real spot traffic and wanted it treated as a first-class explicit mode rather than left as free-form comment text.
- Shared comment parsing is used by manual DX, peer PC11/PC61, and human telnet ingest. A missing token there creates ingest inconsistency across multiple sources.
- PSKReporter can already preserve explicit source modes when config allows them, so leaving operator-mode surfaces unchanged would create ingest/filter/help drift.
- Existing behavior for `FT4` and `FT8` does not accept hyphenated aliases in comment parsing, and operators explicitly requested that `FT-2` not be introduced as a special alias.

## Decision
- Add `FT2` as a first-class explicit mode token in the shared comment parser.
- Preserve explicit `FT2` through the existing ingest/output pipeline with normal explicit-mode provenance handling.
- Treat PSKReporter `FT2` as a valid explicit mode when the operator includes `FT2` in `pskreporter.modes`.
- Add `FT2` to `filter.SupportedModes`, telnet usage/help surfaces, and CC `SET/<MODE>` / `SET/NO<MODE>` shortcuts.
- Do not add `FT-2` alias recognition.
- Do not change mode inference, confidence glyph eligibility, or path-reliability semantics for `FT2` in this change.

## Alternatives Considered
1. Support `FT2` only in comment parsing
   - Pros:
   - Minimal code churn.
   - Cons:
   - Leaves PSKReporter/operator surfaces inconsistent and makes filtering/help incomplete.
2. Support `FT2` in PSKReporter only
   - Pros:
   - Minimal scope for MQTT ingest.
   - Cons:
   - Does not solve the reported human-spot behavior and keeps shared parser drift.
3. Add both `FT2` and `FT-2` aliases
   - Pros:
   - More permissive user input handling.
   - Cons:
   - Deviates from the current FT4/FT8 token policy without operator justification.

## Consequences
- Positive outcomes:
  - Human/manual, peer, human telnet, and PSKReporter ingest now share a consistent explicit `FT2` contract.
  - Operators can filter and request help for `FT2` using both classic and CC dialect surfaces.
  - The explicit-mode pipeline remains deterministic and table-driven.
- Negative outcomes / risks:
  - Explicit `FT2` may seed the trusted DX+frequency recent-evidence cache like other explicit modes.
  - Operator-visible supported-mode lists expand, which can affect expectations for saved filter profiles and docs.
- Operational impact:
  - No concurrency, queue, timeout, reconnect, or shutdown behavior changes.
  - No new backpressure or drop semantics are introduced.
  - `FT2` remains outside current confidence-glyph logic and path-reliability special handling.

## Validation
- `go test ./spot ./rbn ./pskreporter ./filter ./telnet ./commands .`
- `go test ./spot -fuzz=FuzzParseSpotComment -fuzztime=5s`
- Added regression coverage for:
  - comment parsing of `FT2`,
  - manual DX command parsing of `FT2`,
  - minimal human telnet parsing of `FT2`,
  - PSKReporter `FT2` acceptance when allowed,
  - filter matching and CC shortcut handling for `FT2`.

## Rollout and Reversal
- Rollout plan:
  - Deploy together with the operator config that includes `FT2` in `pskreporter.modes`.
  - Verify live human/PSKReporter FT2 spots are visible under intended mode filters.
- Backward compatibility impact:
  - `FT2` becomes a supported explicit mode token and filter/help target.
  - `FT-2` remains unsupported.
- Reversal plan:
  - Revert the token-table, filter-surface, and docs changes as one unit if operator feedback shows the new mode contract is undesirable.

## References
- Issue(s):
  - none
- PR(s):
  - none
- Commit(s):
  - pending
- Related ADR(s):
  - `ADR-0052`
  - `ADR-0018`
- Troubleshooting Record(s):
  - `TSR-0013`
- Docs:
  - `README.md`
