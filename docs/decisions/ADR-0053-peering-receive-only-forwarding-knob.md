# ADR-0053 - Peering Receive-Only Forwarding Knob With Local DX Exception

Status: Accepted
Date: 2026-03-07
Decision Makers: Core maintainers
Technical Area: config, peer, main output pipeline
Decision Origin: Design
Troubleshooting Record(s): none
Tags: peering, forwarding contract, receive-only mode

## Context
- Operators need a single YAML control that can stop peer data-plane forwarding without tearing down peer sessions.
- The current system has two peer spot egress paths:
  - local publish from `main.processOutputSpots` into `peer.Manager.PublishDX`
  - transit relay inside `peer.Manager.HandleFrame` for inbound `PC11`/`PC61` plus raw `PC26`
- Existing peer enablement semantics from ADR-0047 already require explicit outbound session opt-in, but they do not provide receive-only behavior once sessions are up.
- Existing publish-source semantics from ADR-0049 restrict local peer publish to local non-test human/manual spots, but they do not suppress transit relay.

## Decision
- Add `peering.forward_spots` as a single operator-facing data-plane forwarding knob.
- Interpret omitted or `false` as disabled.
- Define disabled behavior as:
  - keep peer sessions, login/auth, keepalive, and config refresh active
  - keep ingest of inbound peer traffic active
  - suppress transit relay of inbound `PC11`/`PC61` spot frames
  - suppress raw relay of inbound `PC26` merge frames
  - still allow outbound publish of locally created `DX` command spots
- Define enabled behavior as:
  - preserve the prior local publish policy from ADR-0049
  - allow transit relay of inbound `PC11`/`PC61`/`PC26` spot data-plane traffic
- Centralize this decision in the `peer` package so both the local publish path and the transit relay path consult the same policy surface.

## Alternatives Considered
1. Two knobs: one for local publish and one for transit relay
   - Pros:
     - More operator flexibility.
   - Cons:
     - More config complexity for a narrow operational mode; easier to misconfigure.
2. Collapse everything into one physical egress path
   - Pros:
     - One send site in code.
   - Cons:
     - Local publish and transit relay have different protocol timing and frame-shape requirements, especially for raw `PC26`.
3. Disable peering entirely when forwarding should stop
   - Pros:
     - Very simple semantics.
   - Cons:
     - Fails the receive-only requirement and drops useful inbound peer traffic.

## Consequences
- Positive outcomes:
  - Operators can run receive-only peering without losing local `DX` posting to peers.
  - Local publish and transit relay use one deterministic policy surface.
  - Peer write pressure drops when forwarding is disabled because no transit relay is attempted.
- Negative outcomes / risks:
  - The local `DX` exception currently relies on the runtime invariant that `DX` is the only production path creating `SourceManual` spots.
  - Operators who expect historical transit relay behavior must now set `peering.forward_spots: true` explicitly.
- Operational impact:
  - Session liveness/reconnect behavior is unchanged.
  - Inbound peer spots still appear locally through the normal ingest pipeline.
- Follow-up work required:
  - Revisit the exception if additional local manual spot producers are introduced.

## Validation
- Added/updated tests:
  - `config/peering_forward_spots_test.go`
  - `peer/forwarding_policy_test.go`
  - `peer/manager_test.go`
- Validation commands:
  - `go test ./config ./peer . -run "TestPeering|TestHandleFrame|TestPublishDX|TestCloneSpotForPeerPublish|TestShouldPublishLocalSpot|TestShouldRelayDataFrame"`
  - `go test ./...`
  - `go vet ./...`
  - `staticcheck ./...`
  - `go test -race ./...`

## Rollout and Reversal
- Rollout plan:
  - Deploy with `peering.forward_spots: false` for receive-only mode or `true` to preserve prior transit relay behavior.
- Backward compatibility impact:
  - Omitted `peering.forward_spots` now means receive-only data-plane behavior rather than full forwarding.
- Reversal plan:
  - Remove the knob and restore unconditional transit relay/local publish behavior, then mark this ADR superseded.

## References
- Issue(s): none
- PR(s): none
- Commit(s): pending
- Related ADR(s): ADR-0047, ADR-0049, ADR-0050
- Troubleshooting Record(s): none
- Docs: `README.md`, `data/config/peering.yaml`
