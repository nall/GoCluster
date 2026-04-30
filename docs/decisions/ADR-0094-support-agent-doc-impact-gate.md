# ADR-0094: Support-Agent Documentation Impact Gate

- Status: Accepted
- Date: 2026-04-30
- Decision Origin: Design

## Context
GoCluster has operator documentation and a separate `customgpt/` support-agent
routing layer. Operator-visible changes can update README/operator docs while
leaving support-agent routing stale, which causes the support assistant to send
operators to incomplete or outdated guidance.

The support-agent docs are intentionally not a second source of truth. They
should route to authoritative repo docs and call out caveats such as effective
YAML or current-code dependence.

## Decision
Add a required `Support-agent docs impact: Required | Not required` decision to
the Non-trivial workflow, parallel to `README impact`.

The impact is required for operator-support topics such as commands, HELP,
filters, modes, EVENT families, glyphs, diagnostics, YAML/config surfaces,
logging, observability, troubleshooting, startup, service, deployment, source,
ingest, peer, or connection behavior.

When required, Codex must inspect and update relevant `customgpt/` routing
files while keeping them as routing/support guidance rather than duplicating
operator documentation.

## Alternatives considered
1. Always update `customgpt/` whenever operator docs change.
   - Rejected because it would create noisy churn when routing is already
     adequate.
2. Rely on reviewer judgment without a named workflow field.
   - Rejected because the failure mode is silent omission.
3. Put the full rule in `AGENTS.md`.
   - Rejected because `AGENTS.md` should remain compact and already routes
     detailed workflow requirements through `docs/change-workflow.md`.

## Consequences
### Benefits
- Future operator-visible changes must make an explicit support-agent sync
  decision.
- Support routing stays aligned with authoritative operator docs.
- `customgpt/` remains a thin routing layer instead of becoming another
  maintained behavior manual.

### Risks
- The new field adds one more required Non-trivial evidence item.
- Contributors may over-update `customgpt/` unless the routing-layer constraint
  is followed.

### Operational impact
- No runtime behavior changes.
- Future Codex closeouts and traceability must report support-agent docs impact.

## Links
- Related issues/PRs/commits: none
- Related tests: targeted text checks for workflow phrases
- Related docs: `docs/change-workflow.md`, `docs/templates/non-trivial-change-template.md`, `docs/review-checklist.md`, `customgpt/README.md`
- Related TSRs: none
- Supersedes / superseded by: extends ADR-0092
