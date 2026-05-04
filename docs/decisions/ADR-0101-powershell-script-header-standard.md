# ADR-0101: PowerShell Script Header Standard

- Status: Accepted
- Date: 2026-05-04
- Decision Origin: Design

## Context

The scripts directory contains operationally important PowerShell tooling for
release packaging, profiling, PGO builds, workflow checks, console sizing, and
local Codex skill installation. Several scripts were self-explanatory to a
developer reading the body, but they did not consistently state prerequisites,
side effects, and safety boundaries at the top of the file for operators,
support agents, or future developers.

## Decision

No durable decision change.

Standardize tracked first-party PowerShell scripts on comment-based help that
describes purpose, usage, prerequisites, side effects, and safety boundaries.
The headers are local context only. The script body remains authoritative for
actual parameters, commands, paths, process launch behavior, release publishing
behavior, and generated artifacts.

## Alternatives considered

1. Reuse the YAML header format.
   - Rejected because scripts are executable tooling and PowerShell
     comment-based help is the native format.
2. Document scripts only in README prose.
   - Rejected because local top-of-file context is useful before running an
     operationally sharp script.
3. Refactor script parameters and hard-coded paths while adding headers.
   - Rejected as outside this comment/help-only pass.

## Consequences

### Benefits

- Operators and developers can see a script's purpose and side effects before
  execution.
- Support agents can route build, release, profiling, and script questions
  without copying script bodies into support docs.
- Existing script behavior remains unchanged.

### Risks

- Header comments can drift if future script behavior changes without updating
  comment-based help.
- Readers may over-trust comments; the script body and current repo docs remain
  authoritative when evidence conflicts.

### Operational impact

- No build, release, profiling, GitHub publishing, Codex skill installation, or
  console behavior changes.
- No script parameters, commands, generated paths, or process launch behavior
  changed.

## Links

- Related issues/PRs/commits: none
- Related tests: PowerShell parser checks over `scripts/*.ps1`,
  `git diff --check`
- Related docs: `scripts/README.md`, `scripts/*.ps1`,
  `customgpt/source-map.md`, `customgpt/gpt-instructions.md`
- Related TSRs: none
- Supersedes / superseded by: none
