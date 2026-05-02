# ADR-0099: README Telnet-User First Ordering

Status: Accepted
Date: 2026-05-02
Decision Origin: Design

## Context

The top-level README is the first route for project overview, commands,
filters, path reliability, and operator setup. Its previous order put release,
configuration, build, and Linux service sections before the telnet command
experience, which made the page less useful for DX-cluster users who only need
to connect and operate a session.

## Decision

No durable decision change.

This change rearranges the top-level README so telnet users see connection,
first commands, generated HELP, filters, dedupe, confidence, diagnostics, and
path hints before node setup, build, service, logging, and repository-layout
details. It preserves the generated HELP block and does not change command
semantics, config defaults, runtime behavior, packaging, or service behavior.

## Alternatives considered

1. Leave README operator-first and rely on `docs/OPERATOR_GUIDE.md` for users.
2. Split telnet usage into a new top-level user guide.
3. Remove build and service details from README entirely.

## Consequences

### Benefits

- Telnet users can find connection and command behavior before operator setup.
- Operator and builder material remains available later in the README and in
  `docs/OPERATOR_GUIDE.md`.
- The generated HELP block remains the command reference anchor.

### Risks

- Node operators must scroll farther for setup details.
- The README is still long because it keeps both user and operator routes.

### Operational impact

- No runtime behavior, config schema, parser behavior, command behavior,
  protocol behavior, release packaging, or service behavior changed.

## Links

- Related issues/PRs/commits:
- Related tests: `go test ./commands -run TestReadmeDefaultGoHelpBlockMatchesProcessor`
- Related docs: `README.md`, `docs/OPERATOR_GUIDE.md`,
  `customgpt/source-map.md`
- Related TSRs:
- Supersedes / superseded by:
