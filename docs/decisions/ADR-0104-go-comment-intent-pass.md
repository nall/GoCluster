# ADR-0104: Go Comment Intent Pass

- Status: Accepted
- Date: 2026-05-04
- Decision Origin: Design

## Context

The repository now has support-agent-oriented YAML and PowerShell documentation
standards. Runtime/support-critical Go packages and replay tools also need
comments that help humans and agentic support workflows discover ownership,
operational intent, and troubleshooting boundaries without changing behavior.

## Decision

Add intent-focused Go comments to selected runtime/support-critical packages and
replay tooling. Comments should explain why code exists, what operational
boundary it owns, and how support should reason about delay/drop/fallback/state
paths. They should not become line-by-line restatements of mechanics.

No durable runtime decision change.

## Alternatives considered

1. Comment every function and type mechanically.
   - Rejected because it would add noise and make support-oriented comments
     harder to find.
2. Limit comments to exported identifiers only.
   - Rejected because many important troubleshooting boundaries are unexported
     runtime helpers.
3. Leave Go comments as-is after YAML/PowerShell work.
   - Rejected because support-agent discovery also depends on runtime and replay
     code intent.

## Consequences

### Benefits

- Runtime delay, drop, support-memory, and replay analysis paths are easier to
  discover from source.
- Comments document operational ownership without changing queue, timing,
  state, parser, or fanout behavior.
- Replay/profiling tools now better explain artifact and metric intent.

### Risks

- Comments can drift if future behavior changes without updating the nearby
  intent text.
- Over-commenting remains a risk if future passes expand beyond operationally
  meaningful boundaries.

### Operational impact

- No runtime, config, protocol, parser, queue, timing, telnet, archive, peer, or
  replay behavior changes.
- No support-agent routing doc changes required; this is source-level discovery
  documentation.

## Links

- Related issues/PRs/commits: none
- Related tests: `go test ./internal/cluster ./telnet`, `go test ./spot`,
  `go test ./internal/cluster ./reputation ./solarweather`,
  `go test ./cmd/rbn_replay ./cmd/callcorr_reveng_rebuilt`, `go test ./...`
- Related docs: runtime/support-critical Go source comments, replay tool source
  comments
- Related TSRs: none
- Supersedes / superseded by: none
