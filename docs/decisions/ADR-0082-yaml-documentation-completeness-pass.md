# ADR-0082: YAML Documentation Completeness Pass

- Status: Accepted
- Date: 2026-04-25
- Decision Origin: Design

## Context

The repository's first-party YAML files are part of the operator contract.
Several files were accurate and loadable but not equally easy to understand:
the config README omitted required registered files, the taxonomy reference
table lacked a field guide, and a few small YAML files looked like unexplained
fixtures or terse controls.

## Decision

No durable decision change.

This pass updates documentation and YAML comments only. It does not change YAML
values, config schema, loader behavior, runtime defaults, or any generated
artifact. Existing config ownership from ADR-0067 and taxonomy ownership from
ADR-0069 remain unchanged.

## Alternatives considered

1. Change YAML schema or split taxonomy into smaller files.
2. Move all YAML explanations into external Markdown only.
3. Leave terse YAML comments in place because strict loader tests already cover behavior.

## Consequences

### Benefits

- Operators can see every registered config file in the config README.
- High-impact YAML surfaces include local field guidance near the values being edited.
- Sample/generated YAML is less likely to be mistaken for required operator config.

### Risks

- Inline comments can drift if future behavior changes without a docs pass.
- The comments describe existing behavior but do not make unsupported YAML
  capability combinations valid.

### Operational impact

- No runtime, protocol, config, service, queue, or retained-state behavior changed.
- Existing config directories remain compatible because no YAML keys or values changed.

## Links

- Related issues/PRs/commits:
- Related tests: `go test ./config`, `go test ./commands`, `go test ./...`, `go vet ./...`, `staticcheck ./...`, `golangci-lint run ./... --config=.golangci.yaml`, `git diff --check`
- Related docs: `data/config/README.md`, `data/config/spot_taxonomy.yaml`, `data/config/mode_seeds.yaml`, `data/config/prop_report.yaml`, `data/config/pipeline.yaml`, `.golangci.yaml`, `reputation/data/reputation/K1ABC.yaml`
- Related TSRs:
- Supersedes / superseded by:
