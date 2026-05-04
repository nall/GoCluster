# ADR-0103: YAML Documentation Rigor Workflow Gate

- Status: Accepted
- Date: 2026-05-04
- Decision Origin: Design

## Context

ADR-0100 and ADR-0102 standardized first-party YAML file headers and
key-comment rigor for operator, support-agent, and developer use. The workflow
still needed an explicit gate so future YAML additions and edits keep the same
standard instead of relying on memory from the original documentation pass.

## Decision

Add a workflow gate for checked-in first-party YAML changes. The gate points to
`data/config/README.md` as the authoritative header/key-comment standard,
requires a YAML comment/header audit in the Non-trivial template, adds YAML
comment concerns to Review Pass focus, and introduces a read-only PowerShell
checker for mechanical validation.

No durable runtime decision change.

## Alternatives considered

1. Rely on ADR-0100 and ADR-0102 only.
   - Rejected because future Codex runs need an execution gate, not just a
     historical decision record.
2. Enforce subjective comment quality entirely in a script.
   - Rejected because usefulness and drift require review against code, docs,
     loaders, and operational intent.
3. Duplicate the complete YAML comment standard in `AGENTS.md`.
   - Rejected because `AGENTS.md` should remain the entry contract and route to
     the owning workflow/config docs.

## Consequences

### Benefits

- Future YAML changes have an explicit support-oriented documentation audit.
- Mechanical checks catch missing headers and accidental YAML token changes in
  comment-only work.
- Review remains responsible for subjective clarity and drift.

### Risks

- The checker can warn on boolean comments that are justified by side effects;
  those warnings must be reviewed rather than treated as automatic failures.
- Workflow text can drift if future changes update `data/config/README.md`
  without updating the gate.

### Operational impact

- No runtime, config-loader, schema, protocol, telnet, queue, logging, or
  support-agent behavior changes.
- Future YAML documentation work has one additional read-only validation step.

## Links

- Related issues/PRs/commits: none
- Related tests: `scripts/check-yaml-doc-rigor.ps1`,
  `scripts/check-yaml-doc-rigor.ps1 -CommentOnlyCompare`, `git diff --check`
- Related docs: `AGENTS.md`, `docs/change-workflow.md`,
  `docs/templates/non-trivial-change-template.md`, `docs/review-checklist.md`,
  `docs/dev-runbook.md`, `data/config/README.md`
- Related TSRs: none
- Supersedes / superseded by: extends ADR-0100 and ADR-0102
