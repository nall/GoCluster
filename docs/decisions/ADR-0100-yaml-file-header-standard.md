# ADR-0100: YAML File Header Standard

- Status: Accepted
- Date: 2026-05-04
- Decision Origin: Design

## Context

The first-party config YAML files are part of the operator contract and are
also useful context for support agents and developers. Earlier documentation
passes clarified YAML ownership classes and added comments to high-risk files,
but the top-of-file context was not consistently shaped across all registered
runtime config YAML.

## Decision

No durable decision change.

Standardize a compact five-line comment-only header on checked-in first-party
runtime config YAML under `data/config/*.yaml`. The header names purpose,
ownership, runtime behavior, safe-edit boundaries, and the authoritative source
route using the exact label shape documented in `data/config/README.md`. The
header is local context only; it does not define schema, defaults, loader
behavior, or runtime semantics.

## Alternatives considered

1. Add AI-specific prose to every YAML file.
   - Rejected because operator-facing comments should help humans and support
     agents without creating a separate AI-only contract.
2. Document the convention only in `customgpt/`.
   - Rejected because `customgpt/` is a routing layer, while the config README
     owns YAML guidance.
3. Change loader/schema behavior to enforce ownership classes.
   - Rejected as outside this comment-only clarity pass.

## Consequences

### Benefits

- Operators can identify purpose, ownership, runtime behavior, and safe-edit
  boundaries at the top of each first-party config file.
- Support agents can use local YAML context while still routing to
  authoritative docs.
- The change preserves existing config values, schema, loader behavior,
  defaults, and runtime semantics.

### Risks

- Header comments can drift if future config behavior changes without updating
  the matching YAML and docs.
- Readers may over-trust comments; the config README, package docs, current
  code, and ADRs remain authoritative when evidence conflicts.

### Operational impact

- No runtime, protocol, startup, validation, queue, retention, logging, or
  telnet behavior changes.
- Existing config directories remain compatible because no YAML keys or values
  changed.

## Links

- Related issues/PRs/commits: none
- Related tests: `go test ./config`, `go test ./...`, `go vet ./...`,
  `staticcheck ./...`, `golangci-lint run ./... --config=.golangci.yaml`,
  `git diff --check`
- Related docs: `data/config/README.md`, checked-in `data/config/*.yaml`,
  `customgpt/source-map.md`, `customgpt/gpt-instructions.md`
- Related TSRs: none
- Supersedes / superseded by: extends ADR-0082 and ADR-0097
