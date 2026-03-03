# ADR-0047 - Peering Outbound Sessions Require Explicit Enablement

Status: Accepted
Date: 2026-03-03
Decision Makers: Core maintainers
Technical Area: config, peer
Decision Origin: Troubleshooting chat
Troubleshooting Record(s): TSR-0011
Tags: peering, config contract, outbound connections

## Context
- Operators use placeholder peer entries in `peering.yaml` during staged rollout.
- Runtime behavior showed placeholder peers being dialed despite `enabled: false`.
- The existing normalization logic auto-enabled peers when host and port were set, which violated operator intent and created avoidable reconnect noise.

## Decision
- Remove host/port-based auto-enable behavior for `peering.peers[]`.
- Outbound peer activation is explicit:
  - `enabled: true` -> dial peer.
  - `enabled: false` -> never dial peer.
  - omitted `enabled` -> defaults to disabled.

## Alternatives Considered
1. Keep auto-enable and add warnings only
   - Pros: backward compatible for implicit configs.
   - Cons: continues unwanted dial loops for placeholder entries; weak operator control.
2. Auto-enable only when `enabled` key omitted
   - Pros: preserves explicit false while keeping implicit convenience.
   - Cons: still dials peers by default when omitted; conflicts with desired strict opt-in semantics.
3. Add global compatibility toggle
   - Pros: migration flexibility.
   - Cons: more config complexity and operator confusion for a narrow issue.

## Consequences
- Positive outcomes:
  - Deterministic operator control over outbound dialing.
  - Safe placeholders in config without accidental connection attempts.
- Negative outcomes / risks:
  - Existing configs that relied on omitted `enabled` will no longer dial until set to `true`.
- Operational impact:
  - Reduced reconnect/DNS error noise from inactive placeholder peers.
- Follow-up work required:
  - None required; behavior covered by config tests.

## Validation
- Added tests covering omitted/false/true enable states for peers.
- Full repository checks passed:
  - `go test ./...`
  - `go vet ./...`
  - `staticcheck ./...`
- This decision would be invalidated if operational requirements demand implicit auto-dial for omitted `enabled`.

## Rollout and Reversal
- Rollout plan:
  - Deploy change and set `enabled: true` explicitly for intended peers.
- Backward compatibility impact:
  - Behavioral change for configs that omitted peer `enabled`.
- Reversal plan:
  - Reintroduce legacy implicit behavior (preferably behind an explicit compatibility flag).

## References
- Issue(s): none
- PR(s): none
- Commit(s): pending
- Related ADR(s): none
- Troubleshooting Record(s): TSR-0011
- Docs: `data/config/peering.yaml`
