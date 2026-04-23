# ADR-0073: Root Live-Runtime Boundary and Root Artifact Hygiene

- Status: Accepted
- Date: 2026-04-23
- Decision Origin: Design

## Context

The repo root had accumulated three different kinds of material in one place:

- live-binary `package main` implementation files
- standalone and offline tooling artifacts
- historical analysis notes, generated binaries, and protocol reference material

That layout made it difficult to tell what actually compiled into the live server, what belonged to auxiliary tooling, and what was merely historical context. The repo had already moved toward extraction-first structure in the monolith refactor stages, but the root still looked like a mixed workspace instead of a clear live-binary entrypoint.

## Decision

Adopt the following durable repo-structure rules:

1. The repo root keeps the live binary entrypoint only.
2. Live-binary-only implementation moves under `internal/cluster`.
3. Standalone executables live under `cmd/<tool>/`.
4. Historical analysis notes and protocol/reference artifacts do not remain at the repo root; they move under `docs/archive/analysis` or `docs/reference`.
5. This is a structural clarification only; it must not change YAML meaning, protocol behavior, startup/shutdown ordering, or hot-path semantics.

## Alternatives considered

1. Keep the current mixed root layout.
2. Do only conservative root cleanup while leaving live runtime spread across root `package main` files.
3. Split the live runtime immediately into multiple new domain packages instead of using a private `internal/cluster` boundary.

## Consequences

### Benefits

- The compile path is obvious from the repo layout alone.
- The root becomes easier to scan for operator-facing and build-entry files.
- Live-runtime code has a clear private ownership boundary without turning into a public shared API.
- Auxiliary tools and historical material stop obscuring the production binary.

### Risks

- Structural moves can break tests that depended on root-package-private helpers.
- Historical docs and scripts may reference old root paths and need follow-up maintenance.
- `internal/cluster` can become another catch-all if future changes ignore domain-package boundaries.

### Operational impact

- `go build .` remains the live-binary build path from repo root.
- Standalone tool builds remain under `cmd/...`.
- Operator-visible behavior is unchanged; only code ownership and artifact placement changed.

## Links

- Related issues/PRs/commits:
- Related tests: `go test .`, `go test ./internal/cluster`, `go test ./cmd/...`, `go test ./...`, `go test -race ./...`
- Related docs: `README.md`, `docs/monolith-refactor-stage1-4.md`
- Related TSRs:
- Supersedes / superseded by:
