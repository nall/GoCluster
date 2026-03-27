# ADR-0056 - Local Self-Spots Use an Operator-Authoritative V Path

Status: Accepted
Date: 2026-03-27
Decision Makers: Cluster maintainers
Technical Area: commands, main output pipeline, telnet fan-out, custom_scp, docs
Decision Origin: Troubleshooting chat
Troubleshooting Record(s): TSR-0014
Tags: self-spot, confidence, stabilizer, temporal, custom_scp

## Context

- Local `DX` command spots are human/manual runtime input and are already peer-published immediately under the existing local-DX policy.
- A self-spot (`DX == DE` after normalization) represents direct operator authority over the advertised callsign, but the runtime treated it like any other local manual voice/CW/RTTY spot.
- That allowed local self-spots to enter resolver-adjacent temporal/stabilizer rails before telnet fan-out, creating avoidable delay and out-of-sequence timestamps for an operator-authored spot.
- `custom_scp` admission is intentionally V-only to avoid reinforcement loops from weaker or derived confidence classes.

## Decision

- Define a local self-spot as a non-test `SourceManual` spot whose normalized `DX` and `DE` callsigns are identical.
- In the runtime main pipeline, treat local self-spots as operator-authoritative input:
  - bypass resolver mutation
  - bypass temporal hold
  - bypass telnet stabilizer delay
  - force confidence glyph `V`
- Keep local self-spots on the existing telnet and peer delivery queues; do not add a direct synchronous fast path at command entry.
- Allow correction-eligible local self-spots to enter `custom_scp` through the existing V-only admission rail.
- Do not extend this behavior to peer-origin or upstream-origin self-looking spots.
- Do not change replay/shared stabilizer policy; this is a runtime main-pipeline decision only.

## Alternatives Considered

1. Keep current behavior and let self-spots follow normal resolver/temporal/stabilizer rails.
   - Pros:
     - No new special case.
   - Cons:
     - Delays operator-authored self-spots and misses the requested `V`/`custom_scp` semantics.
2. Add a new synchronous telnet fast path directly in `handleDX()`.
   - Pros:
     - Minimal visible latency.
   - Cons:
     - Duplicates queue ownership, bypasses normal clone/delivery boundaries, and risks inconsistent peer/telnet behavior.
3. Add a dedicated persisted `Spot` field marking self-spots.
   - Pros:
     - Very explicit data model.
   - Cons:
     - Wider blast radius than needed for the current runtime, where local manual production is already tightly scoped.

## Consequences

### Positive outcomes
- Local self-spots reach telnet without resolver/temporal/stabilizer delay.
- Local self-spots always show `V`.
- `custom_scp` can learn from correction-eligible local self-spots through the existing V-only contract.
- Peer publish timing remains unchanged and bounded-resource delivery ownership stays intact.

### Negative outcomes / risks
- `custom_scp` now incorporates operator self-spots, which may strengthen later `S`-floor support for those calls.
- Future new local manual spot producers would implicitly inherit this behavior if they reuse the same manual/non-test/self predicate without revisiting the contract.

### Operational impact
- Stabilizer counters for self-spot-heavy workloads should skew toward immediate release.
- No new queues, goroutines, timeout knobs, or wire-format changes.
- Slow-client and overload behavior remain unchanged because delivery still uses existing queues and drop policy.

## Validation

- Added/updated tests:
  - `spot/self_spot_test.go`
  - `main_test.go`
  - `peer/forwarding_policy_test.go`
- Commands:
  - `go test ./spot ./peer . -run 'Test(IsLocalSelfSpot|EvaluateTelnetStabilizerDelayLocalSelfSpotPassesThrough|LocalSelfSpotResolverStageForcesVAndAdmitsCustomSCP|ShouldPublishLocalSelfSpotWhenForwardingDisabled)' -count=1`
  - `go test ./...`
  - `go vet ./...`
  - `staticcheck ./...`
  - `go test -race ./...`

## Rollout and Reversal

- Rollout plan:
  - Deploy normally; no config migration is required.
- Backward compatibility impact:
  - Operator-visible timing/confidence behavior changes only for local non-test self-spots.
- Reversal plan:
  - Remove the local self-spot predicate bypasses and mark this ADR superseded.

## References

- Issue(s): n/a
- PR(s): n/a
- Commit(s): pending
- Related ADR(s):
  - `docs/decisions/ADR-0043-custom-scp-v-only-admission.md`
  - `docs/decisions/ADR-0046-stabilizer-delay-eligibility-unknown-s-p-only.md`
  - `docs/decisions/ADR-0049-peer-publish-local-human-only.md`
- Troubleshooting Record(s):
  - `docs/troubleshooting/TSR-0014-local-self-spot-delay-and-confidence.md`
- Docs:
  - `README.md`
  - `docs/OPERATOR_GUIDE.md`
  - `docs/decision-log.md`
  - `docs/troubleshooting-log.md`
