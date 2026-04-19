# ADR-0063 - Configurable Telnet Bulletin Dedupe

Status: Accepted
Date: 2026-04-19
Decision Makers: Core maintainers
Technical Area: config, peer, telnet fan-out
Decision Origin: Troubleshooting chat
Troubleshooting Record(s): TSR-0018
Tags: bulletins, dedupe, bounded-state, peering

## Context
- WWV/WCY and `TO ALL` announcement lines are delivered as telnet control traffic rather than spot traffic.
- The shared spot dedupe pipeline therefore cannot suppress duplicate bulletins arriving from multiple peers or relay sources.
- Peer-level bulletin keys used raw frame text, which made hop-only route variants distinct.
- The operator requested a 10-minute dedupe window, but required that value to be YAML-configurable rather than hard-coded.

## Decision
- Add all-source telnet bulletin dedupe for WWV, WCY, and `TO ALL` announcements.
- Configure the behavior under `telnet`:
  - `bulletin_dedupe_window_seconds`
  - `bulletin_dedupe_max_entries`
- Ship defaults:
  - window: `600` seconds
  - max entries: `4096`
- Treat `bulletin_dedupe_window_seconds: 0` as disabled.
- Reject negative windows and reject max entries below `1` when dedupe is enabled.
- Use a bounded in-memory cache owned by `telnet.Server`, with TTL expiry and oldest-key eviction on cap.
- Key telnet bulletin dedupe by normalized bulletin kind plus the exact line delivered to users after newline normalization.
- Change peer WWV/WCY and PC93 dedupe keys to canonical payload keys so hop-only route variants collapse before telnet callbacks.

## Alternatives Considered
1. Peer-origin-only dedupe
   - Pros: Smaller behavior change.
   - Cons: Does not suppress duplicates from human/relay passthrough, which uses the same telnet bulletin surface.
2. Hard-code a 10-minute window
   - Pros: Smaller config surface.
   - Cons: Violates operator requirement and makes rollout tuning require a code change.
3. Reuse spot secondary dedupe
   - Pros: One dedupe subsystem.
   - Cons: Bulletins are not spots and do not have spot keys, timing, filters, or archive/peer semantics.

## Consequences
- Positive outcomes:
  - Telnet users see fewer repeated WWV/WCY and `TO ALL` lines.
  - Duplicate bulletins do not consume per-client control queue capacity.
  - Operators can disable or tune the window and retained key cap.
  - Retained state is bounded by both time and cardinality.
- Negative outcomes / risks:
  - Intentional repeated announcements with identical text inside the configured window are suppressed.
  - One additional mutex is taken per unique or duplicate bulletin; this is outside the spot hot path.
- Operational impact:
  - Control queue behavior for unique bulletins remains unchanged.
  - Direct PC93 talk messages remain outside telnet bulletin dedupe.
  - Stats expose accepted, suppressed, evicted, and tracked bulletin dedupe counts.

## Validation
- Added/updated tests:
  - Config default, disabled, invalid, and explicit setting tests.
  - Telnet duplicate, expiry, disabled, no-eligible-client, and churn-cap tests.
  - Peer hop-insensitive key and callback suppression tests.
- Required validation:
  - `go test ./config -run "Test.*BulletinDedupe" -count=1`
  - `go test ./peer -run "Test(WWVKey|PC93Key|HandleFrameWWV|HandleFramePC93Announcement).*" -count=1`
  - `go test ./telnet -run "TestBulletinDedupe" -count=1`
  - `go test ./...`
  - `go vet ./...`
  - `staticcheck ./...`
  - `go test -race ./...`

## Rollout and Reversal
- Rollout plan:
  - Deploy with shipped defaults.
  - Watch bulletin dedupe suppressed/evicted counts and telnet control drops.
- Backward compatibility impact:
  - Default behavior now suppresses identical WWV/WCY/announcement lines for 600 seconds.
  - Set `telnet.bulletin_dedupe_window_seconds: 0` to restore previous no-dedupe behavior.
- Reversal plan:
  - Disable the window in YAML or remove the telnet bulletin cache and mark this ADR superseded.

## References
- Related ADR(s): ADR-0050, ADR-0053, ADR-0054
- Troubleshooting Record(s): TSR-0018
- Docs: `README.md`, `telnet/README.md`, `peer/README.md`, `data/config/runtime.yaml`
