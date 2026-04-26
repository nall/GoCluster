# ADR-0088: FT2 Ingest-Rate Dashboard Counter

Status: Accepted
Date: 2026-04-26
Decision Origin: Design

## Context

`FT2` is already a supported explicit mode for RBN digital and PSKReporter ingest. The console dashboard per-minute ingest-rate block showed `FT8` and `FT4` counters but did not expose `FT2`, leaving an observability gap for an existing mode contract.

## Decision

Add `FT2` to the RBN and PSKReporter ingest-rate counter display, ordered as `FT8`, `FT4`, `FT2`. Keep the change limited to stats aggregation and dashboard formatting.

No durable decision change: this records observability parity for the existing `FT2` mode contract and does not change parsing, taxonomy, filtering, path reliability, FT confidence, or ingest behavior.

## Consequences

- Operators can see per-minute `FT2` volume alongside `FT8` and `FT4`.
- Dashboard formatting uses comma-aware field widths to keep separators aligned for expected operator-visible count ranges.
- Counts are not truncated or clamped if live values exceed the expected widths.

## Links

- Related ADRs: `ADR-0055`
- Related TSRs: `TSR-0013`
- Related code: `internal/cluster/bootstrap.go`, `ui/dashboard_v2.go`
- Related tests: `internal/cluster/main_stats_test.go`, `ui/dashboard_v2_test.go`
