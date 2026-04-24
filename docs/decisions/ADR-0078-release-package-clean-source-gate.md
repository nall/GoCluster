# ADR-0078: Release Package Clean Source Gate

- Status: Accepted
- Date: 2026-04-24
- Decision Origin: Design

## Context

The GitHub Release package is intended to be reproducible from committed
source-controlled inputs. A local release package built from uncommitted files
can contain binary behavior, rendered README content, or config examples that
do not correspond to any commit. Separately, stale `go.mod` or `go.sum`
metadata can make a release build depend on implicit local cleanup.

## Decision

The release packaging script fails by default when `git status --porcelain`
reports any uncommitted worktree changes. It also runs `go mod tidy -diff`
before staging or compiling; any required module-file change fails the package
build without modifying files.

For local validation only, the script accepts `-AllowDirty`. Dirty local test
packages keep the existing `+dirty` version suffix so they are visibly distinct
from clean release packages.

## Alternatives considered

1. Warn on dirty source and continue.
2. Automatically run `go mod tidy` and package the resulting local files.
3. Prompt interactively when the worktree is dirty.

## Consequences

### Benefits

- Normal release packages correspond to committed source.
- GitHub Actions and local release builds use the same non-interactive policy.
- Module hygiene problems are reported before a zip is created.
- Local test packages remain possible, but require an explicit switch and are
  marked in the binary version.

### Risks

- Developers must commit or stash unrelated local edits before creating a
  normal package.
- `go mod tidy -diff` requires a Go toolchain version that supports the flag.

### Operational impact

- Operators should not receive dirty release packages from the normal release
  path.
- A binary version ending in `+dirty` means the package came from an explicit
  local test build, not the default release path.

## Links

- Related issues/PRs/commits:
- Related tests: `go mod tidy -diff`, `scripts/build-release-package.ps1 -OutputDir .tmp\release-validation`, `scripts/build-release-package.ps1 -AllowDirty -OutputDir .tmp\release-validation`
- Related docs: `README.md`, `scripts/build-release-package.ps1`, `docs/decisions/ADR-0076-github-release-package.md`, `docs/decisions/ADR-0077-compile-date-binary-version.md`
- Related TSRs:
- Supersedes / superseded by:
