# docs/review-checklist.md

This document defines the mandatory review posture for Non-trivial tasks.

## Review Pass
The Review Pass happens after implementation and before final closeout.

Purpose:
- switch from implementer mode to reviewer mode
- find hidden regressions, edge cases, and missing tests
- verify that the diff matches the approved scope

Required output:
- findings first, ordered by severity
- then confirmed fixes
- then rerun of affected validations

Review focus:
- correctness
- protocol/format compatibility
- hidden behavior drift
- edge cases
- concurrency and lifecycle safety
- cancellation and shutdown
- backpressure, queue, drop, and disconnect semantics
- memory/allocation risks
- performance regressions
- maintainability and readability
- missing tests
- documentation gaps

If there are no material findings, say:
- `Review Pass findings: none material`

## Self-Audit
After the Review Pass, produce a Self-Audit with pass/fail for each category below.

### Required categories
- Scope completeness vs Scope Ledger
- Dependency impact coverage
- Contract and user-visible behavior disclosure
- Correctness and protocol semantics
- Concurrency, cancellation, deadlines, and shutdown
- Backpressure, queue, drop, and disconnect policy
- Resource bounds
- Performance evidence where applicable
- Security and robustness
- Testing adequacy
- Checker discipline
- Documentation and README review
- Decision-memory obligations
- Traceability completeness
- Validation block completeness

### Self-Audit rules
- Use `PASS`, `FAIL`, or `N/A` only.
- `N/A` is allowed only when the category truly does not apply.
- Every `FAIL` must include a short explanation and next action.
- Do not hide uncertainty. If evidence is incomplete, fail the category.

## PR-style summary
Every Non-trivial task must end with a PR-style summary.

Required sections:
- Summary
- Tradeoffs
- Risks and mitigations
- Contracts and compatibility
- User impact and determinism
- Observability impact
- README impact
- Skill check
- Verification commands and results
- Dependency scan evidence (for Full-rigor tasks)
- Decision refs
- Scope-to-Code Traceability
- Validation block

## Scope-to-Code Traceability
Map every Scope Ledger item with status `Agreed` or `Pending` as of the start of the implementation cycle to:
- code locations
- tests
- docs/comments updated
- decision refs if applicable

No omissions allowed.

## Verification command reporting
For each major command, report:
- exact command
- why it was run
- result
- whether it was incremental or final

Example shape:
- `go test ./...` - baseline regression check - pass
- `go test -race ./...` - concurrency/lifecycle verification - pass
- `go test ./internal/cluster -run TestSlowClientDropPolicy` - targeted regression - pass

## Final validation block
The final three lines must be exactly:

Validation Score: X/6
Failed items: none | <comma-separated failed item numbers/names>
Auto-fail conditions triggered: no | yes (<conditions>)