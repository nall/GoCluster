# docs/change-workflow.md

This document defines the full workflow for Non-trivial tasks and the deeper rules behind `AGENTS.md`.

## Core principle
For Non-trivial work, do not go directly from idea to code. Move through:
1. understand the current system
2. define scope
3. confirm approval state
4. plan the slices
5. implement one verified slice at a time
6. review the diff
7. close out with traceability and validation

## IDE context discipline
When using the Codex VS Code extension:
- Prefer the user's open files as the primary context.
- Ask the user to open the most relevant caller/callee files when context is thin.
- Use selected code as a focus anchor when a single function, method, or protocol path is under discussion.
- If Auto Context is available and on, still name the critical files explicitly in your analysis so the user can verify you are looking in the right place.

## Task classification
### Small
A task is Small only if it is tightly localized and does not change contracts, concurrency/lifecycle, operational behavior, or shared interfaces.

### Non-trivial
Anything with meaningful blast radius, uncertain impact, or operational consequences is Non-trivial.

When in doubt, choose Non-trivial.

## Git preflight
Required for every Non-trivial change:
- record branch name
- confirm working tree state
- identify rollback point
- note any unrelated dirty files that must not be touched

Output format:
- `Git preflight: branch=<name>; worktree=<clean|dirty acknowledged>; rollback=<hash/tag/branch>`

## Approval-state reporting
Required for every Non-trivial change before any implementation work.

Output format:
- `Ledger status: Approved vN found: yes`
or
- `Ledger status: Approved vN found: no`

Rules:
- For Non-trivial work, `yes` is required before code, diffs, file edits, or full validation commands.
- If approval is not present, stop after the proposed ledger.

## Current-State Understanding Note
This is mandatory before implementation planning.

Required content:
- current request/data/control flow
- key packages/files/functions involved
- boundaries and ownership
- invariants that must not break
- likely blast radius
- top 3 failure modes if changed incorrectly

Quality rules:
- ground statements in inspected code
- mention concrete file/package identifiers
- say `Unknown from inspected code` rather than guessing
- keep it concise but specific

## Requirements & Edge Cases Note
Required for Non-trivial work.

Cover:
- functional requirements
- non-functional requirements
- compatibility constraints
- overload behavior
- reconnect behavior
- shutdown behavior
- observability expectations
- safety/security constraints
- edge cases that must be tested or reasoned about

This is where hidden expectations should be surfaced before code.

## Contract and behavior disclosure
Required before coding for every Non-trivial task.

State explicitly:
- contract changes, or exactly `No contract changes`
- user-visible behavior changes, or exactly `No user-visible behavior changes`

Cover when applicable:
- protocol/format/compatibility
- ordering
- drop/disconnect semantics
- deadlines/timeouts
- metrics/logging
- configuration knobs
- operator-visible behavior
- error text

## Dependency rigor decision tree
Choose `Light` or `Full`.

### Light rigor
Use Light only when all are true:
- localized package
- no shared component/interface change
- no protocol/parser/config/schema change
- no user-visible or operator-visible contract change
- no concurrency/lifecycle impact outside the local package

Expected output:
- touched files/packages
- nearest upstream callers reviewed
- nearest downstream consumers reviewed
- tests/docs/config touched

### Full rigor
Use Full when any are true:
- shared package or interface
- parser/protocol/config/schema changes
- concurrency/lifecycle/timeout/backpressure/shutdown changes
- metrics/logging/observability contract changes
- user-visible or operator-visible behavior changes
- uncertain blast radius
- fan-out, queueing, caching, or hot-path changes

Required output:
- touched files/packages
- upstream callers/sources reviewed
- downstream consumers reviewed
- shared interfaces/components reviewed
- config/metrics/logs/docs affected
- exact one-line evidence block:
  `Dependency scan evidence: <repo search commands/steps used>; reviewed files/packages: <list>`

## README impact declaration
Required before coding for every Non-trivial task.

Use exactly one:
- `README impact: Required`
- `README impact: Not required`

Then add one sentence of reasoning.

## Implementation Plan
Distinct from the Scope Ledger. The ledger says what is approved. The plan says how to do it.

Required fields:
- objective
- in scope
- out of scope
- files/packages to inspect and likely files to change
- tests to add or update
- exact validation commands in execution order
- milestone breakdown
- rollback note

Rules:
- milestone 1 must be the smallest production-safe slice
- do not combine multiple uncertain changes into one slice
- keep the first slice easy to verify

## Architecture Note
Mandatory for every Non-trivial change before code.

Required content:
- concurrency model
- ownership/lifetime
- backpressure and queue policy
- failure/recovery behavior
- resource bounds
- timeout/deadline behavior
- shutdown sequencing
- determinism guarantees
- tradeoffs and rejected alternatives when material

## User Impact and Determinism Note
Required for every Non-trivial change.

State explicitly:
- whether user-visible behavior changes
- whether operator-visible behavior changes
- what slow clients experience
- what overload looks like
- what reconnect churn looks like
- how behavior remains deterministic
- if there is no user-visible change, say so explicitly

## Skill discipline
Skills may help with analysis or execution, but they do not replace this workflow.

Rules:
- no skill may bypass exact approval gating
- no skill may suppress required understanding, dependency, or documentation steps
- if a skill requires unavailable tools or privileged surfaces, skip it and state why
- for explanation-only review requests, suppress any skill-driven change suggestions

## Implementation slicing rules
- Implement only the current milestone.
- Run the milestone's checks before continuing.
- If results reveal hidden blast radius, stop and update the Scope Ledger.
- Keep diffs narrow.
- Do not sneak in opportunistic cleanup unless it is required for correctness or clarity and is called out explicitly.

## Testing and checker discipline
At minimum:
- `go test ./...`
- `go vet ./...`
- `staticcheck ./...`

Also required when applicable:
- `go test -race ./...` for concurrency/lifecycle
- fuzzing for parser/protocol work
- benchmarks and profiling for hot paths

Rules:
- run checks incrementally
- report commands and results honestly
- add regression tests for changed behavior when feasible
- explain why any test was not added

## Performance evidence
Required when behavior touches hot paths, fan-out, queueing, parsing, allocation pressure, timers, or lock contention.

Evidence should include as applicable:
- before/after benchmark numbers
- allocs/op
- pprof CPU or heap evidence
- lock/contention evidence
- explanation of why the change is safe under nominal and overload conditions

Do not make optimization claims without measurements.

## Documentation expectations
Review and update when applicable:
- README
- operator docs
- protocol docs
- comments on invariants/ownership/concurrency/drop policy
- ADR/TSR records
- test names and descriptions for operator-facing behavior

For every Non-trivial task, explicitly say:
- `README impact: Required`
- or `README impact: Not required`
with one sentence of reasoning

## Completion requirements
A Non-trivial task is not complete until:
- approval state is recorded as present
- code is implemented
- checks are run
- Review Pass is done
- docs are reviewed
- ADR/TSR obligations are satisfied
- Scope-to-Code Traceability is complete
- the exact 3-line validation block is present
