# ADR-0106: Go Crawler Entry Comments

- Status: Accepted
- Date: 2026-05-04
- Decision Origin: Design

## Context

ADR-0104 added intent-focused Go comments to support-critical runtime and
replay code. A follow-up crawlability review found the style works where it is
present, but important entry and integration files could still require a human
or support agent to read deep implementation before knowing why the file matters.

The missing standard is not "comment every helper." The useful boundary is a
short file/package entry note on files that act as subsystem entry points,
runtime integration points, support-critical leaves, or replay/tool entry
points.

## Decision

Add crawler-entry comments to high-value support-critical Go files discovered by
a repo-wide first-party source pass, including telnet filter command handling,
path prediction/store ownership, live runtime wiring, output pipeline
integration, reputation gating, persisted user records, call-confusion scoring,
signal resolver state, and RBN replay entry/pipeline/runner files.

Extend the workflow with a changed-file mechanical checker,
`scripts/check-go-crawler-entry-comments.ps1`, and require it for added or
materially changed support-critical Go files. The script is a review aid only;
human review remains responsible for judging whether comments explain ownership,
intent, related docs/tests, and troubleshooting meaning without mechanical
noise.

No runtime behavior, config values, protocol syntax, queue behavior, parser
behavior, or replay semantics change.

## Alternatives considered

1. Require crawler-entry comments on every Go file.
   - Rejected because helper and test files would accumulate low-value noise.
2. Keep only the subjective Go Comment Intent gate from ADR-0105.
   - Rejected because future additions need a mechanical prompt when changed
     support-critical files lack an entry comment.
3. Add a hand-maintained repo manifest.
   - Rejected for now because package READMEs, source-map routing, and local
     entry comments are enough if kept current.

## Consequences

### Benefits

- Support agents can rank high-value Go files faster during open-ended repo
  crawling.
- Future Codex runs get a concrete check for changed support-critical files.
- The standard reinforces intent/why comments without turning source into a
  generated manifest.

### Risks

- The checker can only identify likely missing entry comments; it cannot judge
  quality or drift.
- The support-critical path list must be reviewed when new subsystems are added.
- Over-commenting remains a risk if future agents apply the pattern to simple
  helpers instead of entry/integration boundaries.

### Operational impact

- None. This is comment and workflow documentation only.

## Links

- Related issues/PRs/commits: none
- Related tests: `scripts/check-go-crawler-entry-comments.ps1 -ChangedOnly -FailOnMissing`, `go test ./internal/cluster ./telnet ./pathreliability ./spot ./filter ./reputation ./cmd/rbn_replay`, `git diff --check`
- Related docs: `docs/code-quality.md`, `docs/change-workflow.md`,
  `docs/review-checklist.md`, `docs/dev-runbook.md`,
  `docs/templates/non-trivial-change-template.md`,
  `customgpt/developer-guide-index.md`, `scripts/README.md`
- Related TSRs: none
- Supersedes / superseded by: extends ADR-0104 and ADR-0105
