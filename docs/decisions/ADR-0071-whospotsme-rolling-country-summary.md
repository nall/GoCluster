# ADR-0071: WHOSPOTSME Rolling Country Summary

- Status: Accepted
- Date: 2026-04-23
- Decision Origin: Design

## Context

`WHOSPOTSME <band>` adds a new operator-visible telnet command that answers:
"Which countries, grouped by continent, have recently spotted my logged-in
callsign on this band?"

The current implementation introduces durable decisions in several areas:

- a new shared retained-state store owned by runtime startup and shutdown;
- a YAML-owned rolling window under `who_spots_me.window_minutes`;
- a specific ingest/query contract for which accepted observations count;
- an operator-visible output contract for continent ordering and formatting.

Those choices affect resource bounds, shared component behavior, and
operator-visible mode selection, so they need an ADR rather than only code and
tests.

## Decision

`WHOSPOTSME` uses a global rolling recent-heard index keyed by normalized DX
callsign plus normalized band. Telnet session state is used only to resolve
who "me" is for the current authenticated/logged-in user.

Accepted observations are recorded after correction, quality gating, beacon/test
rejection, and primary dedupe, but before secondary FAST/MED/SLOW suppression.
The command therefore reports absolute accepted observations for this node,
not the subset later suppressed by secondary dedupe and not unique spotter
counts.

The retained store is bounded and deterministic:

- second-resolution rolling buckets over a fixed configured window;
- sharded per-key aggregates for query speed;
- hard caps on active keys and countries per key;
- periodic cleanup plus oldest-entry eviction or country admission refusal when
  caps are reached;
- no raw event retention and no ingest backpressure from this feature.

Operator-visible behavior is:

- command: `WHOSPOTSME <band>`;
- window: YAML-owned `who_spots_me.window_minutes`, validated to `1..60`;
- result: top 5 countries per continent, sorted by count descending with
  deterministic tie ordering;
- token format: `<country>(<count>)` only;
- empty continents render as `(no data)`;
- country labels resolve to DXCC prefix on read/format, with deterministic
  fallback if needed.

The rolling window uses cluster acceptance time, not embedded source
timestamps, so the command answers what this node has admitted recently.

## Alternatives considered

1. Per-telnet-session state
   - Rejected because the command semantics are global recent-heard data, not a
     per-connection cache.
2. Record only post-secondary-dedupe survivors
   - Rejected because FAST/MED/SLOW policies are output suppression policies,
     not the truth source for accepted observations.
3. Count unique spotters per country
   - Rejected for v1 because it changes semantics, raises retained-state cost,
     and was explicitly out of scope.
4. Hard-code a 10-minute window
   - Rejected because the final approved scope required operator-owned YAML for
     the rolling window.

## Consequences

### Benefits

- Operators get a deterministic recent-heard summary by band without per-user
  retained state.
- The feature remains bounded by time and cardinality and does not retain raw
  spot events.
- HELP output can reflect the live configured window.
- Secondary dedupe tuning does not silently change the meaning of
  `WHOSPOTSME`.

### Risks

- Counts can exceed what users saw forwarded on telnet because secondary
  suppression happens after `WHOSPOTSME` admission.
- Countries with dense automated spotting can dominate because counts are
  absolute observations, not unique spotters.
- The feature adds one more retained shared structure on the accepted-spot path,
  so bounds and cleanup correctness matter.

### Operational impact

- Startup now requires valid `who_spots_me.window_minutes` in the active YAML
  config.
- Runtime memory use scales with configured window length and active
  `DX call + band` cardinality, subject to code-level caps.
- Changing the window requires restart because the store shape is built at
  startup.
- Empty or metadata-poor spot shapes are skipped and do not produce partial
  pseudo-countries.

## Links

- Related tests: `spot/who_spots_me_test.go`,
  `spot/who_spots_me_bench_test.go`, `config/who_spots_me_config_test.go`,
  `commands/processor_test.go`, `main_test.go`
- Related docs: `README.md`, `data/config/README.md`,
  `data/config/runtime.yaml`
- Related TSRs: none
