# ADR-0081: Operator Docs Completeness Pass

- Status: Accepted
- Date: 2026-04-25
- Decision Origin: Design

## Context

The repository documentation needed a visitor-first operator path covering
cluster usage, release downloads, required config edits, Windows builds, Linux
builds, and Linux service operation. Existing docs contained much of the
command and config detail, but the operator entry points were split across
multiple files and did not clearly cover Linux service operation.

## Decision

No durable decision change.

This pass updates operator-facing documentation only. It keeps the existing
release policy from ADR-0076: GitHub Releases publish the current ready-to-run
Windows amd64 package, while Linux operators build from source. It also keeps
the existing config-loader and UI-mode contracts unchanged.

## Alternatives considered

1. Create a new runtime or packaging contract for Linux releases.
2. Move all config details into the top-level README.
3. Leave Linux service operation undocumented until a packaged Linux release exists.

## Consequences

### Benefits

- First-time visitors get a clearer path from download to config to connect.
- Linux operators get source-build and `systemd` guidance without changing
  release packaging.
- Packaged users receive the same minimum operator path in the generated
  release README.

### Risks

- Linux deployment guidance is documentation-only and assumes operators build
  and install the binary themselves.
- Future release packaging changes must keep these docs aligned.

### Operational impact

- No runtime behavior, config defaults, release scripts, or generated artifacts
  changed.
- Operators are directed to use `ui.mode: headless` for unattended Linux
  service operation and an interactive terminal for local console inspection.

## Links

- Related issues/PRs/commits:
- Related tests: `go test ./commands`, `go test ./cmd/release_readme ./config`, `GOOS=linux GOARCH=amd64 go build -trimpath -o tmp\gocluster-linux-amd64 .`, `GOOS=windows GOARCH=amd64 go build -trimpath -o tmp\gocluster-windows-amd64.exe .`, `git diff --check`
- Related docs: `README.md`, `download/README.md`, `docs/OPERATOR_GUIDE.md`, `docs/release/README.md.template`, `telnet/README.md`
- Related TSRs:
- Supersedes / superseded by:
