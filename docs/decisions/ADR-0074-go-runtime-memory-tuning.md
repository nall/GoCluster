# ADR-0074 - Go Runtime Memory Tuning

Status: Accepted
Date: 2026-04-24
Decision Makers: project maintainer and Codex
Technical Area: config, startup, runtime memory
Decision Origin: Design
Troubleshooting Record(s): none
Tags: memory, config, runtime

## Context

The dashboard exposes Go `runtime.MemStats` values for heap and system memory.
Profiling showed modest CPU use, a live heap around the 500 MiB range after
warmup, and steady allocation churn. Running the binary through a PowerShell
profiling wrapper can set `GOMEMLIMIT` and `GOGC`, but production deployment is
normally a direct binary launch. Runtime memory tuning therefore needs to be
operator-owned YAML, not a wrapper-script requirement.

## Decision

Add required `go_runtime` settings to the merged runtime config:

- `go_runtime.memory_limit_mib`
- `go_runtime.gc_percent`

Positive values are applied once immediately after config load and before large
retained stores are initialized, using `debug.SetMemoryLimit` and
`debug.SetGCPercent`. A value of `0` leaves the Go runtime or environment value
unchanged. The effective configured behavior is printed/logged during startup.

## Alternatives Considered

1. Keep using wrapper scripts.
   - Pros: no runtime code change.
   - Cons: does not match direct-binary production deployment.
2. Hard-code `750MiB` and `50`.
   - Pros: simplest code.
   - Cons: requires rebuilds to tune memory and hides an operational setting.
3. Add YAML-owned runtime tuning.
   - Pros: auditable, production-friendly, matches existing config contract.
   - Cons: changes startup behavior and requires validation/testing.

## Consequences

- Positive outcomes:
  - Operators can launch `gocluster.exe` directly and still apply Go runtime
    memory controls.
  - Runtime memory tuning is visible in checked-in/operator YAML.
  - The `0` sentinel preserves environment/default behavior when desired.
- Negative outcomes / risks:
  - A too-low memory limit can increase GC CPU and GC pause pressure.
  - Changing runtime tuning affects the whole process, not one component.
- Operational impact:
  - Shipped config applies a 750 MiB soft memory target and GC percent 50.
  - Existing alternate config directories must add the required `go_runtime`
    block.
- Follow-up work required:
  - Re-profile under production-like load before lowering the memory target.

## Validation

- Config load tests cover shipped values, zero sentinels, missing section, and
  invalid values.
- Startup-unit tests cover runtime application and zero-sentinel no-op behavior.
- Live-profile confirmation is still required before claiming memory improvement
  under production load.

## Rollout and Reversal

- Rollout plan: ship YAML keys with conservative values and monitor dashboard
  Heap, Sys, GC p99, and CPU.
- Backward compatibility impact: alternate config directories missing
  `go_runtime` now fail load under the existing required-YAML policy.
- Reversal plan: set both keys to `0` to preserve Go/env defaults, or revert the
  config and startup application changes.

## References

- Related ADR(s): ADR-0067
- Docs: `data/config/README.md`, `data/config/runtime.yaml`
