# ADR-0117: Hot Path Duplicate Work Removal

- Status: Accepted
- Date: 2026-05-06
- Decision Origin: Design

## Context
Profiling from the 2026-05-05 runtime showed avoidable allocation and CPU work
in already-bounded hot paths. The goal was surgical cleanup: remove repeated
normalization, repeated route splitting, avoidable timer resets, and Custom SCP
expiry-index rebuilds without changing output semantics, config, protocol,
archive format, or retained-state bounds.

## Decision
Keep the existing behavioral contracts and remove duplicate runtime work where
the owner and invariants were already clear:

- Spot constructors populate normalized fields directly and no longer call a
  second normalization pass before beacon refresh.
- `EnsureNormalized` treats populated normalized DX fields as trusted pipeline
  state and only canonicalizes from the raw DX field when the normalized field
  is empty.
- Output-pipeline final metrics no longer forces a second normalization pass
  when no stage mutated normalized fields.
- FT confidence observation no longer drains due groups inside the per-spot
  observe stage after the output loop has already performed the release pass,
  and the FT timer is not reset when the next due time is unchanged.
- Custom SCP observation and static expiry heaps keep key indexes current on
  root pops instead of marking indexes dirty and rebuilding the full map later.
- MQTT route matching caches route levels at route registration and scans the
  incoming topic without allocating a split slice on every publish.
- Telnet self-only delivery uses the existing owned-snapshot path where the
  output pipeline can prove no later mutation will occur, or defers self-only
  delivery until the final shared snapshot exists.

Resolver snapshot publishes remain unchanged. Although repeated snapshot
publication allocates, the snapshot `EvaluatedAt` value participates in resolver
tie-break behavior, so suppressing equivalent-looking publishes would be a
semantic change rather than duplicate-work removal.

## Alternatives considered
1. Keep the existing implementation and rely on future live profiling.
   Rejected because today's profile already identified direct duplicate work.
2. Use object pools for spots, FT groups, or broadcast payloads.
   Rejected for this scope because ownership/lifetime risk is higher than
   removing known repeated operations.
3. Suppress resolver snapshot publishes when only `EvaluatedAt` changes.
   Rejected because `EvaluatedAt` affects resolver snapshot ordering.

## Consequences
### Benefits
- MQTT route matching avoids per-message route/topic split allocation.
- Custom SCP cleanup no longer turns ordinary root pops into full index rebuild
  opportunities.
- FT confidence scheduling does less repeated scan/reset work under sustained
  ingest.
- Self-only telnet delivery avoids an extra spot clone on owned final-output
  paths.
- Constructor and finalization paths avoid redundant normalization calls.

### Risks
- Custom SCP heap/index coupling must remain exact after root pop, delete, and
  stale-entry cleanup.
- FT timer due tracking must not hide a changed deadline.
- The MQTT matcher must preserve wildcard and shared-subscription semantics.
- Trusting populated normalized DX fields relies on the existing construction
  boundary that normalized fields are canonical after ingest/materialization.

### Operational impact
- No operator-visible behavior change.
- No config, command, HELP, archive schema, filter, or protocol change.
- Resource bounds are unchanged; the patch removes churn inside existing
  bounded structures.

## Links
- Related issues/PRs/commits: current working tree
- Related tests:
  - `spot/custom_scp_store_test.go`
  - `spot/bounded_store_bench_test.go`
  - `spot/spot_test.go`
  - `rbn/parse_spot_test.go`
  - `dxsummit/client_test.go`
  - `pskreporter/client_test.go`
  - `peer/parse_test.go`
  - `commands/processor_test.go`
  - `internal/cluster/ft_confidence_runtime_test.go`
  - `internal/cluster/output_pipeline_ownership_test.go`
  - `third_party/paho.mqtt.golang/unit_router_test.go`
- Related docs:
  - `docs/decisions/ADR-0075-output-pipeline-context-allocation.md`
  - `docs/decisions/ADR-0080-custom-scp-retained-heap-layout.md`
  - `docs/decisions/ADR-0115-dx-numeric-ssid-canonicalization.md`
- Related TSRs: none
- Supersedes / superseded by: none
