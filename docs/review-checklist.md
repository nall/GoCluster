# docs/review-checklist.md

This document defines the mandatory review posture for Non-trivial Codex tasks.
The compact output shape is owned by
`docs/templates/non-trivial-change-template.md`.

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
- YAML schema, required-key, null, and sentinel-value behavior
- hidden runtime defaults or downstream config re-defaulting
- edge cases
- concurrency and lifecycle safety
- cancellation and shutdown
- backpressure, queue, drop, and disconnect semantics
- memory/allocation risks
- performance regressions
- maintainability and readability
- missing tests
- documentation gaps
- support-agent routing drift when operator docs or operator-visible behavior changed

If there are no material findings, say:
- `Review Pass findings: none material`

## Self-Audit
After the Review Pass, produce a Self-Audit with pass/fail for each category below.

### Required categories
- Scope and dependency coverage
- Contract, config, and protocol correctness
- Concurrency, backpressure, and resource bounds
- Verification and checker discipline
- Documentation, decision memory, and traceability
- Validation block completeness

### Self-Audit rules
- Use `PASS`, `FAIL`, or `N/A` only.
- `N/A` is allowed only when the category truly does not apply.
- Every `FAIL` must include a short explanation and next action.
- Do not hide uncertainty. If evidence is incomplete, fail the category.
- Use one short note per grouped category. Reference earlier review evidence when
  that already establishes the point.

## Closeout evidence
Every Non-trivial task must end with the template's `CLOSEOUT`,
`TRACEABILITY`, and `VALIDATION` markers. Keep the closeout concise and refer
to earlier markers instead of repeating evidence.

## Scope-to-Code Traceability
Map every Scope Ledger item with status `Agreed` or `Pending` as of the start of the implementation cycle to:
- code locations
- tests
- docs/comments updated
- support-agent docs updated or explicitly not impacted
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
The `VALIDATION` marker must end with these exact three lines:

Validation Score: X/6
Failed items: none | <comma-separated failed item numbers/names>
Auto-fail conditions triggered: no | yes (<conditions>)
