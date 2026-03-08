# Monolith Refactor Stage 1-4 Note

Status: Implemented
Date: 2026-03-08
Scope: Structural refactor of the longest repo routines into smaller private helpers and controller types
Decision refs reviewed: ADR-0015, ADR-0023, ADR-0028, ADR-0030, ADR-0038, ADR-0047, ADR-0048, ADR-0049, ADR-0053

## 1) Intent

Stages 1-4 reduced the largest repo-owned routines into smaller reviewable units without changing runtime behavior, configuration meaning, protocol contracts, or operator-facing semantics.

This note documents how the refactor was organized and what invariants were intentionally preserved.

## 2) Non-goals

- No YAML schema changes
- No CLI changes
- No protocol or wire-format changes
- No startup or shutdown semantic changes
- No output ordering changes
- No timer, queue, or goroutine topology redesign
- No replay-vs-runtime policy consolidation beyond structural extraction

## 3) Before and After

### Stage 1: config loader

Before:
- `config.Load` in `config/config.go` mixed directory merge, unmarshal, raw key-presence checks, defaults, normalization, and validation for many subsystems.

After:
- `config.Load` remains the orchestration entrypoint.
- Section-specific normalization and validation moved into private helpers.
- Raw omitted-vs-explicit key presence moved into a private carrier rather than a long ad hoc boolean list.

Primary outcome:
- Config behavior stayed the same, but section ownership and review boundaries became explicit.

### Stage 2: replay entrypoint

Before:
- `cmd/rbn_replay/main.go` mixed flag/config handling, runtime bootstrap, replay loop, temporal handling, counters, and artifact writing in one routine.

After:
- `cmd/rbn_replay/main.go` is a thin entrypoint.
- Replay lifecycle moved into a private runner in `cmd/rbn_replay/runner.go`.

Primary outcome:
- Replay runtime construction and replay-loop responsibilities are now isolated without changing replay outputs.

### Stage 3: root composition root

Before:
- `main.go` mixed config load, runtime construction, subsystem wiring, goroutine startup, signal handling, and cleanup sequencing.

After:
- `main.go` is a thin composition entrypoint.
- Runtime startup and shutdown phases moved into `main_runtime.go` under `clusterRuntime`.

Primary outcome:
- Startup and shutdown sequencing became phase-oriented and auditable without changing lifecycle behavior.

### Stage 4: output hot path

Before:
- `processOutputSpots` in `main.go` contained resolver application, temporal hold/release, stabilizer release logic, recent-band updates, sink gating, and archive/telnet/peer fan-out in one monolithic routine.

After:
- `processOutputSpots` remains the public package-level entrypoint but now delegates to a private `outputPipeline`.
- Pipeline construction and run loop live in `output_pipeline.go`.
- Spot-stage processing lives in `output_pipeline_stages.go`.
- Stabilizer release handling and delivery planning live in `output_pipeline_delivery.go`.
- Temporal pending-state ownership moved behind `runtimeTemporalController` in `temporal_runtime.go`.

Primary outcome:
- The hot path is still single-consumer and behavior-preserving, but its internal stages are now explicit and independently reviewable.

## 4) Responsibility Map

| Area | Before | After |
|---|---|---|
| Config load | `config.Load` monolith | orchestration plus private section normalizers |
| Replay runtime | replay `main` monolith | replay `main` plus private `replayRunner` |
| Root startup/shutdown | root `main` monolith | root `main` plus private `clusterRuntime` |
| Output pipeline | `processOutputSpots` monolith | wrapper plus private `outputPipeline` and temporal controller |

## 5) Invariants Preserved

The refactor was intentionally extraction-first. These invariants were treated as non-negotiable:

- `config.Load` preserves omitted-key versus explicit `0` or `false` behavior.
- Root startup order, goroutine launch order, and shutdown ordering remain unchanged.
- Replay input order, artifact naming, manifest shape, and replay gating remain unchanged.
- Deduplicated output is still consumed by a single main-loop owner.
- Temporal pending-state, heap order, and timer scheduling remain single-owner within the output loop.
- Stabilizer release remains a separate bounded goroutine, as before.
- Telnet-only suppression does not leak into archive or peer behavior.
- Peer publish eligibility remains governed by the same local-human and receive-only rules as before the refactor.

## 6) Why This Was Not Documented As A New ADR

No ADR was added because these stages were not intended to change system policy or architecture contracts. They changed code shape, not system semantics.

An ADR would have been appropriate only if the refactor had introduced a durable new choice such as:

- different concurrency ownership
- changed backpressure or timer policy
- changed sink suppression boundaries
- changed bootstrap sequencing contracts
- new shared abstractions that altered package boundaries materially

That did not happen in Stages 1-4.

## 7) Validation Approach

Each stage followed the same pattern:

1. identify the monolithic routine and its invariants
2. extract only one structural seam at a time
3. keep call order and ownership unchanged
4. add narrow characterization tests where new pure or stateful seams were introduced
5. run package checks incrementally, then run the full repo checker set

For Stage 4 specifically, `go test -race ./...` remained mandatory because timer ownership, goroutine lifecycle, and hot-path state boundaries were touched structurally.

## 8) Optional Stage 5

Stage 5 is intentionally not part of the completed refactor set.

Its purpose would be to evaluate whether any remaining duplicated bootstrap or policy wiring should be shared across:

- root runtime startup
- replay runtime bootstrap
- output-pipeline helper paths

Stage 5 is optional because it has a different risk profile:

- it is no longer about reducing monolith size
- it introduces a higher chance of abstraction drift
- it can blur the boundary between live runtime code and replay/testing code

Recommended rule:
- do Stage 5 only when a focused duplication audit shows repeated maintenance cost that is larger than the abstraction risk.

## 9) Current End State

After Stages 1-4, the original repo goal for this refactor program was met:

- the identified repo-owned routines above the 500-line threshold were removed
- the major composition and hot-path responsibilities now have explicit private owners
- behavior-preserving structure is documented without claiming a new architectural decision
