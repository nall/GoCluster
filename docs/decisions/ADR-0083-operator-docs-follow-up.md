# ADR-0083: Operator Docs Follow-Up

- Status: Accepted
- Date: 2026-04-25
- Decision Origin: Design

## Context

The operator documentation completeness pass in ADR-0081 made the repository
visitor-first, but follow-up review found a few remaining operator guidance
gaps: the practical command list omitted some live commands, Linux service
setup did not show account/unit-file/ownership steps, and the config checklist
did not explicitly name placeholder values that must be replaced before a real
node connects to upstream feeds.

## Decision

No durable decision change.

This pass updates operator-facing documentation only. It keeps the release
policy, config loader behavior, YAML values, Linux self-install model, and
runtime UI-mode behavior unchanged.

## Alternatives considered

1. Leave the operator guide abbreviated and rely on the generated README HELP
   block for the full command list.
2. Document only the top-level README and leave packaged release guidance less
   explicit.
3. Add a packaged Linux release path before documenting service installation.

## Consequences

### Benefits

- Operators see the missing command surfaces in the practical guide and
  packaged README template.
- First-time node operators are explicitly told to replace public placeholder
  node and ingest identities before connecting real feeds.
- Linux service setup is more copyable and names the service account, unit-file
  location, install ownership, and console-mode workflow.

### Risks

- Linux service guidance remains documentation-only and assumes an operator
  performs a source build and self-install.
- Future command or release packaging changes must keep the guide and release
  template aligned with the generated README HELP block and release script.

### Operational impact

- No runtime behavior, config defaults, command behavior, build behavior, or
  generated artifacts changed.
- Existing deployments remain compatible.

## Links

- Related issues/PRs/commits:
- Related tests: `go test ./commands`, `go test ./cmd/release_readme ./config`, `go test ./...`, `go vet ./...`, `staticcheck ./...`, `golangci-lint run ./... --config=.golangci.yaml`, `git diff --check`
- Related docs: `README.md`, `docs/OPERATOR_GUIDE.md`, `docs/release/README.md.template`
- Related ADRs: ADR-0081, ADR-0082
- Related TSRs:
- Supersedes / superseded by:
