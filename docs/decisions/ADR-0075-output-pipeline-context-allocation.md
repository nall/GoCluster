# ADR-0075: Output Pipeline Context Allocation

- Status: Accepted
- Date: 2026-04-24
- Decision Origin: Design

## Context
Profiling showed repo-owned allocation pressure in the output pipeline. The
optimization scope was surgical and behavior-preserving: reduce allocation
churn without changing filtering, correction, output semantics, retained
evidence windows, YAML behavior, or operator-visible behavior.

## Decision
Use value-owned `outputSpotContext` data for normal output-pipeline flow, and
retain bounded FT confidence pending contexts as compact values rather than
pointers or full pipeline contexts.

No durable runtime policy changed. FT pending groups remain bounded by the
existing `ftConfidenceMaxPendingGroups` and `ftConfidenceMaxPendingSpots`
limits; the representation of each held context changed only to avoid
avoidable per-spot heap allocation and backing-array churn on the main flow.

Use a typed FT confidence heap instead of `container/heap` for the FT pending
queue. The queue ordering remains due time followed by sequence number, but
push/pop no longer route through `any` values.

Expose a family-key pair helper for hot correction-family lookups so callers
that only need to check or record one or two keys do not allocate a temporary
slice for ordinary single-key calls. The existing `CorrectionFamilyKeys` API
remains available for compatibility.

## Alternatives considered
1. Keep pointer-returning context creation and accept the allocation.
2. Add pooling for `outputSpotContext`.
3. Retain full output-pipeline contexts in FT groups.
4. Keep `container/heap` for FT due ordering.
5. Redesign FT confidence evidence windows or correction-family semantics.

## Consequences
### Benefits
- Normal `prepareSpotContext` calls can stay allocation-free.
- FT pending state keeps the existing bounds while retaining fewer bytes per
  held context.
- FT due-queue maintenance avoids interface boxing on push/pop.
- Hot family-key callers can avoid slice allocation in the common no-slash
  callsign case.

### Risks
- Future fields added to `outputSpotContext` that are needed after FT release
  must also be copied into `ftConfidencePendingContext`.
- The typed heap must preserve the previous due-time and sequence tie-break
  order exactly.

### Operational impact
- No operator-visible behavior change.
- No config, protocol, output, correction, filtering, or evidence-window change.

## Links
- Related tests:
  - `internal/cluster/output_pipeline_ownership_test.go`
  - `internal/cluster/output_pipeline_ownership_bench_test.go`
  - `internal/cluster/ft_confidence_runtime_test.go`
  - `spot/correction_family_test.go`
- Related docs: none
- Related TSRs: none
- Supersedes / superseded by: none
