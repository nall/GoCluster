# ADR-0105: Go Comment Intent Workflow Gate

- Status: Accepted
- Date: 2026-05-04
- Decision Origin: Design

## Context

ADR-0104 recorded the support-oriented Go comment pass for runtime-critical
packages and replay tooling. The workflow needed an explicit gate so future Go
changes preserve that intent-focused style instead of reverting to sparse,
mechanical, or stale comments.

## Decision

Add a Go comment intent gate to the Codex workflow. `docs/code-quality.md` owns
the source comment standard, `docs/change-workflow.md` owns when the audit is
triggered, the Non-trivial template records the audit result, the review
checklist adds comment-quality review focus, and the runbook documents the
mechanical checks for comment-only Go changes.

No durable runtime decision change.

## Alternatives considered

1. Rely on ADR-0104 only.
   - Rejected because future Codex runs need an execution gate, not just a
     historical record.
2. Require comments on every exported or unexported Go identifier.
   - Rejected because it would reward noisy mechanical comments over useful
     operational intent.
3. Add a mandatory script to judge comment quality.
   - Rejected because "why" quality, drift, and support usefulness require
     source-aware review. Mechanical checks are useful only for scope control on
     comment-only diffs.

## Consequences

### Benefits

- Future support-critical Go changes must review comment intent explicitly.
- The standard stays centralized in `docs/code-quality.md`.
- Comment-only Go work gets a mechanical diff-scope check without pretending a
  script can judge support usefulness.

### Risks

- The trigger is intentionally judgment-based; Codex must not use it as an
  excuse to over-comment unrelated code.
- Workflow docs can drift if the template or review checklist changes without
  preserving the same audit path.

### Operational impact

- No runtime, config, protocol, parser, queue, timing, telnet, archive, peer, or
  replay behavior changes.
- Future Go implementation work has one additional documentation-quality audit
  when support-critical code is touched.

## Links

- Related issues/PRs/commits: none
- Related tests: `git diff --check`, targeted workflow text checks
- Related docs: `AGENTS.md`, `docs/change-workflow.md`,
  `docs/code-quality.md`, `docs/review-checklist.md`,
  `docs/dev-runbook.md`, `docs/templates/non-trivial-change-template.md`
- Related TSRs: none
- Supersedes / superseded by: extends ADR-0104
