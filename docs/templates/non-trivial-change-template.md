# docs/templates/non-trivial-change-template.md

Use this exact structure for Non-trivial tasks unless the user explicitly requests a different reporting shape.

## 1. Skill check
- Skill check: selected <skill>
or
- Skill check: none applicable

## 2. Current-State Discovery Evidence
- Inspected entry points/surfaces:
- Caller/callee flow checked:
- Persisted/config/archive/schema surfaces checked:
- User-visible/help/docs surfaces checked:
- Existing tests checked:
- Unknown from inspected code:

## 3. Proposed Scope Ledger vN
- Version: vN
- Objective:
- In scope:
  - [Agreed|Pending|Rejected|Deferred] item 1
  - [Agreed|Pending|Rejected|Deferred] item 2
- Out of scope:
- Risks requiring attention:
- Initial classification: Non-trivial
- Reasoning budget: <low|medium|high|xhigh> (lowest sufficient). Rationale: <one sentence>; escalation trigger: <one phrase or "none expected">.

Stop here and wait for the exact approval token:
`Approved vN`

No code, diffs, file writes, or full validation commands before that approval.

---

## 4. Git preflight
- Git preflight: branch=<name>; worktree=<clean|dirty acknowledged>; rollback=<hash/tag/branch>

## 5. Current-State Understanding Note
- Current flow:
- Likely impacted files/packages/functions:
- Invariants that must not break:
- Top 3 failure modes if changed incorrectly:

## 6. Requirements & Edge Cases Note
- Functional requirements:
- Non-functional requirements:
- Compatibility constraints:
- Operational behavior:
- Observability expectations:
- Edge cases:

## 7. Dependency impact
- Dependency rigor: Light | Full
- Touched files/packages:
- Upstream callers/sources reviewed:
- Downstream consumers reviewed:
- Shared components/interfaces reviewed:
- Config/metrics/logs/docs affected:
- Dependency scan evidence: <required for Full rigor>

## 8. Config Contract Audit
- Required? yes/no
- Touched YAML files and classification:
- Single loader path:
- Unknown-key behavior:
- Missing-key behavior:
- Null behavior:
- Explicit `0` / `false` behavior:
- Defaults audit evidence:
- Downstream consumers reviewed:
- Consumer-level tests planned:
- Docs/ADR impact:

## 9. Implementation Plan
- Objective:
- In scope:
- Out of scope:
- Files/packages to inspect and likely files to change:
- Tests to add/update:
- Validation commands in execution order:
- Milestones:
  1.
  2.
- Rollback note:

## 10. Architecture Note
- Concurrency model:
- Ownership/lifetime:
- Backpressure and queue policy:
- Failure/recovery behavior:
- Resource bounds:
- Timeout/deadline behavior:
- Shutdown sequencing:
- Determinism guarantees:
- Alternatives considered:

## 11. User Impact and Determinism Note
- User-visible behavior changes:
- Operator-visible behavior changes:
- Slow-client behavior:
- Overload behavior:
- Reconnect behavior:
- Determinism statement:

## 12. Decision-memory scan
- Relevant ADRs:
- Relevant TSRs:
- Decision refs:
- If none: `No relevant ADR found` / `No relevant TSR found`

## 13. Implementation slices
For each milestone:
- what changed
- files touched
- incremental checks run
- result
- remaining risk before next slice

## 14. Review Pass
- Findings by severity:
- Confirmed fixes:
- Rerun checks:

If no material findings:
- `Review Pass findings: none material`

## 15. Tests and checker results
- Command:
- Purpose:
- Result:
- Incremental or final:

## 16. Performance evidence
- Required? yes/no
- If yes:
  - benchmark results
  - allocs/op
  - pprof highlights
  - conclusion

## 17. Documentation and README
- README impact: Required | Not required
- README action:
- Docs/comments updated:

## 18. ADR/TSR update
- ADR created/updated:
- TSR created/updated:
- or `No decision change`

## 19. Self-Audit
Use:
- Scope completeness: PASS|FAIL|N/A - note
- Dependency coverage: PASS|FAIL|N/A - note
- Contract disclosure: PASS|FAIL|N/A - note
- Correctness/protocol semantics: PASS|FAIL|N/A - note
- Config contract integrity: PASS|FAIL|N/A - note
- Concurrency/lifecycle: PASS|FAIL|N/A - note
- Backpressure/drop/disconnect semantics: PASS|FAIL|N/A - note
- Resource bounds: PASS|FAIL|N/A - note
- Performance evidence: PASS|FAIL|N/A - note
- Security/robustness: PASS|FAIL|N/A - note
- Testing adequacy: PASS|FAIL|N/A - note
- Checker discipline: PASS|FAIL|N/A - note
- Documentation/README: PASS|FAIL|N/A - note
- Decision-memory obligations: PASS|FAIL|N/A - note
- Traceability completeness: PASS|FAIL|N/A - note
- Validation block completeness: PASS|FAIL|N/A - note

## 20. PR-style summary
- Summary:
- Tradeoffs:
- Risks and mitigations:
- Contracts and compatibility:
- Config contract audit:
- User impact and determinism:
- Observability impact:
- README impact:
- Skill check:
- Verification commands and results:
- Dependency scan evidence:
- Decision refs:

## 21. Scope-to-Code Traceability
For every Scope Ledger item that was `Agreed` or `Pending` at the start of implementation:
- Ledger item:
- Code locations:
- Tests:
- Docs/comments:
- Decision refs:

## 22. Validation block
Validation Score: X/6
Failed items: none | <comma-separated failed item numbers/names>
Auto-fail conditions triggered: no | yes (<conditions>)
