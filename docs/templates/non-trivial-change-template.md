# docs/templates/non-trivial-change-template.md

Use this exact structure for Non-trivial tasks unless the user explicitly requests a different reporting shape.

Later sections may reference earlier evidence instead of restating unchanged
facts. Only restate information when the later section adds a new conclusion,
delta, or final disposition.
Omit subsections that are not triggered by the task instead of filling them with
placeholder text.

## Phase A: Approval Packet

### 1. Skill check
- Skill check: selected <skill>
or
- Skill check: none applicable

### 2. Current-State Discovery Evidence
- Inspected entry points/surfaces:
- Caller/callee flow checked:
- Persisted/config/archive/schema surfaces checked:
- User-visible/help/docs surfaces checked:
- Existing tests checked:
- Unknown from inspected code:

### 3. Proposed Scope Ledger vN
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

## Phase B: Execution Closeout

### 4. Git preflight
- Git preflight: branch=<name>; worktree=<clean|dirty acknowledged>; rollback=<hash/tag/branch>

### 5. Design and impact frame
- Current-State Understanding Note:
  - Current flow:
  - Likely impacted files/packages/functions:
  - Invariants that must not break:
  - Top 3 failure modes if changed incorrectly:
- Requirements & Edge Cases Note:
  - Functional requirements:
  - Non-functional requirements:
  - Compatibility constraints:
  - Operational behavior:
  - Observability expectations:
  - Edge cases:
- Dependency impact:
  - Dependency rigor: Light | Full
  - Touched files/packages:
  - Upstream callers/sources reviewed:
  - Downstream consumers reviewed:
  - Shared components/interfaces reviewed:
  - Config/metrics/logs/docs affected:
  - Dependency scan evidence: <required for Full rigor>
- Triggered audits only:
  - Config Contract Audit when required
  - Retained-State Audit when required
  - Performance evidence when required
- Contract and surface declarations:
  - Contracts changed | No contract changes
  - User-visible behavior changes | No user-visible behavior changes
  - Operator-visible behavior changes when relevant
  - Slow-client / overload / reconnect behavior when relevant
  - README impact: Required | Not required
  - Checker set and execution order:
- Decision-memory scan:
  - Relevant ADRs:
  - Relevant TSRs:
  - ADR handling plan:
    - full ADR | updated ADR | lightweight ADR stub
  - Decision refs:
  - If none: `No relevant ADR found` / `No relevant TSR found`

### 6. Implementation Plan
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

### 7. Architecture and determinism
Include only the fields material to the change:
- Concurrency model
- Ownership/lifetime
- Backpressure and queue policy
- Failure/recovery behavior
- Resource bounds
- Timeout/deadline behavior
- Shutdown sequencing
- Determinism guarantees
- Alternatives considered when material

### 8. Implementation slices and verification
For each milestone:
- what changed
- files touched
- incremental checks run
- result
- remaining risk before next slice

### 9. Review Pass
- Findings by severity:
- Confirmed fixes:
- Rerun checks:

If no material findings:
- `Review Pass findings: none material`

### 10. Self-Audit
Use:
- Scope and dependency coverage: PASS|FAIL|N/A - note
- Contract, config, and protocol correctness: PASS|FAIL|N/A - note
- Concurrency, backpressure, and resource bounds: PASS|FAIL|N/A - note
- Verification and checker discipline: PASS|FAIL|N/A - note
- Documentation, decision memory, and traceability: PASS|FAIL|N/A - note
- Validation block completeness: PASS|FAIL|N/A - note

### 11. Closeout summary
- Summary:
- Tradeoffs:
- Risks and mitigations:
- Contracts and compatibility:
- User impact and determinism:
- README impact:
- Verification commands and results:
- ADR handling outcome:
- Decision refs:

### 12. Scope-to-Code Traceability
For every Scope Ledger item that was `Agreed` or `Pending` at the start of implementation:
- Ledger item:
- Code locations:
- Tests:
- Docs/comments:
- Decision refs:

### 13. Validation block
Validation Score: X/6
Failed items: none | <comma-separated failed item numbers/names>
Auto-fail conditions triggered: no | yes (<conditions>)
