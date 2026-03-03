# ADR-0049 - Peer Publish Limited to Local Non-Test Human/Manual Sources

Status: Accepted
Date: 2026-03-03
Decision Makers: Core maintainers
Technical Area: main output pipeline, peer
Decision Origin: Design
Troubleshooting Record(s): none
Tags: peering, source policy, forwarding contract

## Context
- Peer publishing previously accepted all non-test sources except `UPSTREAM` and `PEER`, which allowed local skimmer feeds (`RBN`, `FT8`, `FT4`, `PSKREPORTER`) to be forwarded to peers.
- Operators requested a stricter peering contract: outbound peer publish should carry local human/manual intent, not skimmer stream replication.
- Existing inbound peer relay semantics (hop-bounded `PC11`/`PC61` relay across peer sessions) are required to remain unchanged.
- The change must keep bounded-resource behavior unchanged: no new queues, no new goroutines, and no changes to writer backpressure/drop behavior.

## Decision
- Restrict outbound peer publishing eligibility in `main.shouldPublishToPeers`:
  - Allow: local non-test manual/human sources.
  - Deny: all skimmer sources (`RBN`, `FT8`, `FT4`, `PSKREPORTER`), plus existing denies for `UPSTREAM`, `PEER`, and `IsTestSpotter`.
- Keep inbound peer relay behavior unchanged in `peer.Manager.HandleFrame` for `PC11`/`PC61` hop-based forwarding.
- Keep archive/peer MED dedupe and per-session queue semantics unchanged.

## Alternatives Considered
1. Keep current behavior (forward skimmer + manual sources)
   - Pros:
     - Maximum peer propagation volume.
   - Cons:
     - Violates operator intent for human/manual-only peer publishing.
2. Manual-only allowlist (`SourceManual` only)
   - Pros:
     - Simple and strict.
   - Cons:
     - Excludes potential future local human sources that are non-skimmer and non-test by contract.
3. Configurable source allowlist in YAML
   - Pros:
     - Operator flexibility by deployment.
   - Cons:
     - Extra config complexity and validation surface for a narrow policy decision.

## Consequences
- Positive outcomes:
  - Peer output now aligns with local human/manual intent.
  - Reduced peer traffic volume from local skimmer ingest.
- Negative outcomes / risks:
  - Peers no longer receive local skimmer spots from this node.
  - Operators expecting previous skimmer-forward behavior must adjust expectations.
- Operational impact:
  - Stats/overview semantics for peer ingest remain unchanged.
  - Inbound relay remains active, so peer-origin traffic may still transit hop-bounded.
- Follow-up work required:
  - None.

## Validation
- Added/updated tests:
  - `main_test.go`: source policy matrix for `shouldPublishToPeers` (manual allow, test/skimmer/upstream/peer denies).
  - `peer/manager_test.go`: regression coverage that inbound `PC11`/`PC61` frames still relay to other peers with decremented hop and source exclusion.
- Full repository checks executed:
  - `go test ./...`
  - `go vet ./...`
  - `staticcheck ./...`
  - `go test -race ./...`
- This decision would be invalidated if operators require skimmer-source peer replication again.

## Rollout and Reversal
- Rollout plan:
  - Deploy normally; no config migration required.
- Backward compatibility impact:
  - Behavior change: local skimmer sources are no longer peer-published.
- Reversal plan:
  - Restore prior source gate behavior in `shouldPublishToPeers` and mark this ADR superseded.

## References
- Issue(s): none
- PR(s): none
- Commit(s): pending
- Related ADR(s): ADR-0047
- Troubleshooting Record(s): none
- Docs: `README.md`
