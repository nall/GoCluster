# ADR-0102: YAML Key Comment Standard

- Status: Accepted
- Date: 2026-05-04
- Decision Origin: Design

## Context

After adding compact file headers to checked-in config YAML, several files still
had keys whose purpose, units, sentinel values, or side effects were not clear
from the nearby context. At the same time, blanket comments on every repeated
table row or obvious boolean toggle would make the config harder to read.

## Decision

No durable decision change.

Clarify the config documentation rule: comment keys when the key name and
section do not make purpose, units, sentinel values, ownership, side effects,
runtime consequences, or safe-edit boundaries clear. Do not add comments for
obvious boolean toggles unless they have non-obvious consequences. When keys
repeat across homogeneous sections or list entries, document the first
occurrence or use a field guide instead of repeating the same comment on every
row.

## Alternatives considered

1. Comment every YAML key literally.
   - Rejected because repeated taxonomy, seed, region, peer, and profile tables
     would become harder to edit.
2. Rely only on file headers and external docs.
   - Rejected because local context is useful for operationally important
     units, paths, retention windows, queues, and sentinel values.
3. Add loader/schema metadata for documentation.
   - Rejected as outside this comment-only pass.

## Consequences

### Benefits

- Operators get more local context for paths, queues, timers, source identity,
  sentinels, calibration units, and repeated table schemas.
- Large repeated tables stay readable because field guides document repeated
  keys once.
- Existing config values, schemas, loader behavior, defaults, and runtime
  semantics remain unchanged.

### Risks

- Comments can drift if future config behavior changes without a matching docs
  pass.
- Calibration comments must stay conservative and source-grounded.

### Operational impact

- No runtime, protocol, startup, validation, logging, queue, telnet, or support
  behavior changes.
- Existing config directories remain compatible because no YAML keys or values
  changed.

## Links

- Related issues/PRs/commits: none
- Related tests: tracked YAML value comparison, `go test ./config`,
  `go test ./...`, `go vet ./...`, `staticcheck ./...`,
  `golangci-lint run ./... --config=.golangci.yaml`, `git diff --check`
- Related docs: `data/config/README.md`, checked-in `data/config/*.yaml`,
  `customgpt/source-map.md`, `customgpt/gpt-instructions.md`
- Related TSRs: none
- Supersedes / superseded by: extends ADR-0100
