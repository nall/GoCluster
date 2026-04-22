# docs/change-workflow.md

This document is written for Codex. When `AGENTS.md` sends you here, read the applicable sections before code.

It defines the full workflow for Non-trivial tasks and the deeper rules behind `AGENTS.md`.

## Core principle
For Non-trivial work, do not go directly from idea to code. Move through:
1. understand the current system
2. define scope
3. plan the slices
4. implement one verified slice at a time
5. review the diff
6. close out with traceability and validation

## IDE context discipline
When using the Codex VS Code extension:
- Prefer the user's open files as the primary context.
- Ask the user to open the most relevant caller/callee files when context is thin.
- Use selected code as a focus anchor when a single function, method, or protocol path is under discussion.
- If Auto Context is available and on, still name the critical files explicitly in your analysis so the user can verify you are looking in the right place.

## Task classification
### Small
A task is Small only if it is tightly localized and does not change contracts, concurrency/lifecycle, operational behavior, or shared interfaces.

When a Small task changes code, state a brief classification justification before
editing. The justification should name why the change is localized, why blast
radius is low, and why no Non-trivial triggers apply.

### Non-trivial
Anything with meaningful blast radius, uncertain impact, or operational consequences is Non-trivial.

When in doubt, choose Non-trivial.

## Approval and pre-code gates
Required before every change:
- confirm the current Scope Ledger version and the status of each item
- classify the task as Small or Non-trivial
- record `Skill check: selected <skill>` or `Skill check: none applicable`

For Non-trivial changes:
- do not edit files, propose diffs, run formatters, or run full checker suites until the user has replied with the exact approval token: `Approved vN`
- record `Ledger status: Approved vN found: yes/no`
- do not treat discussion, "please implement", "go ahead", or any non-exact wording as approval
- every scope change after approval requires a new ledger version

### Current-State Discovery before Scope Ledger
Before proposing or confirming a Non-trivial Scope Ledger, perform a targeted
Current-State Discovery pass. The first ledger must be grounded in inspected
code and docs, not assumptions.

Minimum discovery:
- relevant entry points and command/API surfaces
- caller/callee flow at least one level where material
- persisted state, config, archive, or schema surfaces when relevant
- user-visible/operator-visible output and HELP/docs surfaces
- existing tests for the affected behavior
- applicable installed or repo-managed skills

Ask product or semantic questions only after discoverable code facts have been
checked. If a fact cannot be established from inspection, say
`Unknown from inspected code` and name what should be inspected next.

Before code, explicitly identify:
- impacted contracts, or `No contract changes`
- user-visible behavior changes, or `No user-visible behavior changes`
- README impact: `Required` or `Not required`
- checker set and validation command order

## Workflow-drift audit
Required when editing any workflow contract, validation rule, runbook, review
checklist, Codex guidance, or repo-managed skill, including:
- `AGENTS.md`
- `VALIDATION.md`
- `docs/change-workflow.md`
- `docs/review-checklist.md`
- `docs/dev-runbook.md`
- `docs/code-quality.md`
- `docs/WORKING_WITH_CODEX.md`
- `codex-skills/**/SKILL.md`

Audit requirements:
- preserve exact strings that other workflow docs or users rely on
- check that moved or shortened rules remain reachable from `AGENTS.md`
- verify that skill triggers, validation rules, runbook commands, and review
  expectations do not contradict each other
- run targeted text checks for the key workflow phrases touched by the change
- report the audit result in the final summary

## Git preflight
Required for every Non-trivial change:
- record branch name
- confirm working tree state
- identify rollback point
- note any unrelated dirty files that must not be touched

Output format:
- `Git preflight: branch=<name>; worktree=<clean|dirty acknowledged>; rollback=<hash/tag/branch>`

## Current-State Understanding Note
This is mandatory before implementation planning. It extends the pre-ledger
Current-State Discovery with the detail needed for implementation.

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

## Config Contract Audit
Required when a task touches YAML files, config structs, config loaders,
normalizers, runtime defaults, reference tables, operator settings, or optional
tool/secret config.

Config/schema changes require Full dependency rigor unless they are strictly
local test fixture changes.

Required output:
- touched YAML files and classification
- single loader path for each file
- unknown-key, missing-key, null, `0`, and `false` behavior
- defaults audit evidence
- downstream consumers reviewed
- tests proving loader behavior and consumer behavior
- README/config docs/ADR impact

The audit must distinguish YAML-owned operator settings from validation
constants, algorithm constants, compatibility boundaries, and test fixtures.

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

## Implementation slicing rules
- Implement only the current milestone.
- Run the milestone's checks before continuing.
- If results reveal hidden blast radius, stop and update the Scope Ledger.
- Keep diffs narrow.
- Do not sneak in opportunistic cleanup unless it is required for correctness or clarity and is called out explicitly.

## Testing and checker discipline
Use `docs/dev-runbook.md` as the required checker source for Non-trivial
closeout. The list below is the minimum baseline, not the full command set.

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
- code is implemented
- checks are run
- Review Pass is done
- docs are reviewed
- ADR/TSR obligations are satisfied
- Scope-to-Code Traceability is complete
- the exact 3-line validation block is present
