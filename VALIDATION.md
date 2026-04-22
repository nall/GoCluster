# VALIDATION.md - Non-trivial Task Compliance Rubric

Use this scorecard after any Non-trivial Codex task to verify that Codex actually followed `AGENTS.md` and did not merely produce plausible output.

## Purpose
This rubric is designed to catch false-complete work:
- code that was written before approval
- plans that skipped current-state understanding
- missing dependency analysis
- missing or overstated validation
- hidden behavior changes
- config/schema changes that skip YAML contract review
- missing docs, review, or traceability

A task can compile and still fail this rubric.

## How to use
- Score each of the 6 items as `0` or `1`.
- Total the score out of `6`.
- Apply the automatic fail rules even if the numeric score looks acceptable.
- If evidence is missing or ambiguous, score the item `0`.
- Do not give partial credit.

## Required final output block
For every Non-trivial task, Codex must end its final response with this exact 3-line block:

Validation Score: X/6
Failed items: none | <comma-separated failed item numbers/names>
Auto-fail conditions triggered: no | yes (<conditions>)

Do not accept paraphrases or extra lines inside that block.

## Scorecard

### 1) Scope gate and approval discipline
Score `1` only if all are true:
- Codex showed `Proposed Scope Ledger vN` or explicitly referenced the currently approved ledger version.
- Codex did not start code, diffs, file edits, or full validation before you replied with the exact approval token `Approved vN`.
- Codex did not silently expand scope after approval.
- The final summary mapped implemented work back to approved scope items.

Score `0` if any are true:
- Codex assumed approval.
- Codex skipped the ledger.
- Codex started implementation before `Approved vN`.
- Scope drift was not surfaced and re-approved.
- The final summary did not clearly trace back to approved scope.

### 2) Skill and workflow discipline
Score `1` only if all are true:
- Codex explicitly said which skill it used, or explicitly stated `Skill check: none applicable`.
- Codex classified the task appropriately as `Small` or `Non-trivial`.
- For a Non-trivial task, Codex followed the full workflow rather than skipping directly to code.

Score `0` if any are true:
- No skill check was shown.
- An obviously relevant installed skill was skipped.
- The task was treated as Small even though blast radius or behavior change clearly made it Non-trivial.
- Codex skipped required workflow stages for a Non-trivial task.

### 3) Current-state understanding and dependency rigor
Score `1` only if all are true:
- Before coding, Codex produced a `Current-State Understanding Note`.
- That note identified concrete files, packages, functions, control flow, and invariants.
- Codex stated `Dependency rigor: Light` or `Dependency rigor: Full`.
- Codex listed:
  - touched files/packages
  - upstream callers or sources reviewed
  - downstream consumers reviewed
  - shared components or interfaces reviewed
  - config, metrics, logs, or docs affected
- If `Full` rigor applied, Codex included:
  `Dependency scan evidence: <commands/steps>; reviewed files/packages: <list>`
- For config/schema work, Codex included a `Config Contract Audit`.

Score `0` if any are true:
- Codex coded without a concrete current-state read of the system.
- Dependency coverage was vague or implied.
- Upstream or downstream impact was skipped.
- Shared components or interfaces were not examined when clearly relevant.
- A Full-rigor task lacked `Dependency scan evidence`.
- Config/schema work omitted the `Config Contract Audit`.

### 4) Pre-code design discipline
Score `1` only if, before coding, Codex provided all of the following:
- contract changes, or `No contract changes`
- user-visible changes, or `No user-visible behavior changes`
- a distinct `Implementation Plan`
- a short `Architecture Note`
- a `User Impact and Determinism Note`
- for config/schema work, a `Config Contract Audit` that covers required-key,
  null, explicit `0`, explicit `false`, defaults, and downstream consumers

The plan must include:
- objective
- in-scope
- out-of-scope
- likely files/packages to change
- tests to add/update
- validation commands
- milestones
- rollback note

The Architecture Note must cover, when applicable:
- concurrency model
- ownership/lifetime
- backpressure and queue policy
- failure/recovery behavior
- resource bounds
- timeout/deadline behavior
- shutdown sequencing

Score `0` if any are true:
- Codex jumped into code without that pre-code framing.
- The plan was missing or merged into vague prose.
- Architecture implications were not stated for a system-level change.
- Contract or user-visible behavior disclosure was omitted.

### 5) Verification and review discipline
Score `1` only if all are true:
- Codex updated relevant tests, or explicitly explained why no test change was needed.
- Codex showed concrete verification commands.
- Required checks were run for the task type, including race and static checks when applicable.
- Checks were run incrementally, not only once at the very end.
- A `Review Pass` was completed after implementation and before final closeout.
- The final summary reported command results honestly.

Expected checks for most Non-trivial tasks:
- `go test ./...`
- `go vet ./...`
- `staticcheck ./...`

Also required when applicable:
- `go test -race ./...` for concurrency, lifecycle, cancellation, timers, queues, long-lived connections, or shared mutable state
- fuzzing for parser/protocol changes
- benchmarks and profiling for hot-path or performance claims

Score `0` if any are true:
- tests or checks were missing, vague, or only implied
- commands were listed but not clearly run
- a required checker was skipped without explicit waiver
- no Review Pass occurred
- performance claims were made without evidence

### 6) Documentation, decision memory, and traceability
Score `1` only if all are true:
- Codex explicitly stated `README impact: Required` or `README impact: Not required`.
- Docs were reviewed and updated when needed.
- Comments on invariants/ownership/concurrency/drop policy were updated where applicable.
- Codex checked decision memory and explicitly cited relevant ADRs or TSRs, or stated `No decision change`.
- The final summary mapped scope items to code, tests, docs/comments, and decision refs in `Scope-to-Code Traceability`.
- The final response ended with the exact 3-line validation block.

Score `0` if any are true:
- README review status was omitted
- docs were not reviewed for a change that obviously required it
- durable design changes were made with no decision-memory handling
- traceability was missing or incomplete
- the final validation block was missing or malformed

## Scoring interpretation
- `6/6` = Fully followed the contract
- `5/6` = Strong compliance, minor gap
- `4/6` = Partial compliance, review carefully
- `3/6 or below` = Did not reliably follow the contract

A high score does not override an automatic fail.

## Automatic fail conditions
Mark the task non-compliant regardless of numeric score if any of the following happened:

1. Codex implemented, produced diffs, edited files, or ran full validation before `Approved vN`.
2. Codex claimed validation that was not actually performed.
3. Codex skipped repo-wide or shared-component dependency review for a change that clearly required it.
4. Codex omitted `README impact: Required|Not required` on a Non-trivial task.
5. Codex introduced user-visible behavior changes without explicitly disclosing them.
6. Codex omitted `go test -race ./...` for a change that touched concurrency, lifecycle, queues, cancellation, timers, long-lived connections, or shared mutable state, unless you explicitly waived it.
7. Codex left placeholders, stubs, `TODO`, or deferred-hardening markers in touched files.
8. Codex failed to include Scope-to-Code Traceability for approved scope items.
9. Codex omitted the exact final 3-line validation block.
10. Codex changed YAML/config/schema/defaulting behavior without a Config Contract Audit.
11. Codex introduced or preserved a runtime fallback for a YAML-owned setting without explicitly documenting and approving that exception.
12. Codex changed documented zero/false sentinel behavior without consumer-level regression tests.

## Waivers
Waivers are allowed only when explicit, narrowly scoped, and time-bounded.

A valid waiver must state:
- what was waived
- why it was waived
- who approved it
- mitigation
- expiry date

If the waived item is part of an automatic fail condition, the task still fails unless you explicitly override the rubric for that task.

## Quick-use checklist
- Scope approved
- Skill checked
- Current state understood
- Dependencies scanned
- Config Contract Audit completed when config/schema/defaulting behavior changed
- Pre-code design stated
- Tests and checks run
- Review Pass completed
- README/docs reviewed
- Decision memory handled
- Scope-to-Code Traceability done
- Final validation block present

## Copy-paste review template

```text
Non-trivial task validation

1) Scope gate and approval discipline: 0/1
2) Skill and workflow discipline: 0/1
3) Current-state understanding and dependency rigor: 0/1
4) Pre-code design discipline: 0/1
5) Verification and review discipline: 0/1
6) Documentation, decision memory, and traceability: 0/1

Total: __ / 6

Automatic fail triggered: Yes / No
Reason:

Notes:
- 
- 
- 

Required final block present: Yes / No
Block content:
Validation Score: 
Failed items: 
Auto-fail conditions triggered:
