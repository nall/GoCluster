# ADR-0080: Custom SCP Retained Heap Layout

- Status: Accepted
- Date: 2026-04-25
- Decision Origin: Design

## Context
Profiling showed Custom SCP retained heap was dominated by loaded in-memory
state rather than transient allocation churn. The largest repo-owned symbols
included `spot.compactCustomSCPSpotters`,
`CustomSCPStore.upsertEntryExpiryLocked`, `CustomSCPStore.loadFromDB.func1`,
and `customSCPInterner.retain`.

The optimization scope is behavior-preserving: reduce bytes retained by Custom
SCP's in-memory representation without changing admission policy, scoring,
evidence windows, YAML defaults, on-disk encoding, or operator-visible output.

## Decision
Store Custom SCP expiry queues as value-backed typed heaps with key-to-index
maps instead of pointer-backed heap items. The queues remain bounded one-for-one
by retained observation entries and static calls, and delete paths remain
coupled to primary-entry removal.

Store retained spotter observation timestamps as a compact unsigned Unix-second
value inside `customSCPSpotterEntry`. Runtime APIs, persisted Pebble values,
cutoff calculations, and scoring continue to use `int64` Unix seconds at their
boundaries.

Keep Custom SCP load synchronous and keep existing YAML defaults. Do not use
async/lazy load, pooling, or global interners for this change.

## Alternatives considered
1. Lower Custom SCP YAML caps or horizons.
2. Change the on-disk Custom SCP observation encoding.
3. Load Custom SCP asynchronously after startup.
4. Replace exact per-spotter observations with aggregate evidence.
5. Keep pointer-backed expiry heap items and accept the retained heap cost.

## Consequences
### Benefits
- Expiry tracking removes one class of per-entry heap-item object retention.
- Retained spotter slices hold fewer bytes per observation.
- Custom SCP operator semantics and persisted data compatibility remain
  unchanged.

### Risks
- Typed heap code must preserve due-time and deterministic key/call tie-breaks.
- Expiry index repair must remain correct after cleanup pops due items.
- Compact spotter timestamps rely on Unix seconds fitting in `uint32`; out of
  range values are clamped at the representation boundary.

### Operational impact
- No operator-visible behavior change.
- No config, protocol, scoring, admission, output, or evidence-window change.

## Links
- Related tests:
  - `spot/custom_scp_store_test.go`
  - `spot/bounded_store_bench_test.go`
- Related docs:
  - `docs/decisions/ADR-0039-custom-scp-runtime-evidence-and-shared-pebble-resilience.md`
  - `docs/decisions/ADR-0043-custom-scp-v-only-admission.md`
- Related TSRs: none
- Supersedes / superseded by: none
