# ADR-0113: Support-Agent PowerShell Script Retrieval

- Status: Accepted
- Date: 2026-05-04
- Decision Origin: Design

## Context

Tracked first-party PowerShell scripts under `scripts/` are support-critical
operational tooling for release packaging, profiling, PGO builds, workflow
checks, console setup, and Codex skill state. The support-agent routing docs
already route build, release, profiling, and helper-script questions through
`scripts/README.md` and script comment-based help, but the action safe-file
allowlist did not include `.ps1`. That meant the GPT could discover the script
directory documentation without retrieving the authoritative script help.

## Decision

Allow `.ps1` files through the support-agent action safe-file policy and
classify them as `script`.

Keep the existing binary, archive, credential, hidden-path, data-log, and
`customgpt/support-agent/*` deny rules. Do not allow broader executable script
families such as `.bat`, `.cmd`, `.sh`, or `.psm1`.

## Alternatives considered

1. Keep routing through `scripts/README.md` only.
   - Rejected because exact parameter, side-effect, and troubleshooting
     details live in each script's comment-based help and implementation.
2. Copy script help into Markdown docs.
   - Rejected because it would duplicate executable tooling contracts and
     drift from the scripts.
3. Allow all text-like script files.
   - Rejected because the support need is specifically first-party PowerShell
     tooling already documented under `scripts/`.

## Consequences

### Benefits

- The support agent can answer script questions from authoritative
  comment-based help and script code.
- Release, profiling, workflow-check, and local tooling support no longer stops
  at the script README.

### Risks

- `.ps1` files are executable text. The support agent must treat retrieved
  script content as evidence for explanation and troubleshooting, not as code
  to execute.

### Operational impact

- No cluster runtime, telnet, config, archive, peer, replay, queue, timing, or
  long-lived connection behavior changes.
- The support-agent action retrieval contract expands to include first-party
  PowerShell script text.

## Links

- Related issues/PRs/commits: none
- Related tests: Worker syntax check, OpenAPI YAML parse, `.ps1`
  discovery/retrieval smoke tests, support-agent bundle denial smoke tests
- Related docs: `scripts/README.md`, `scripts/*.ps1`,
  `customgpt/source-map.md`, `customgpt/support-agent/actions-schema.yaml`,
  `customgpt/support-agent/cloudflare-worker.js`
- Related TSRs: none
- Supersedes / superseded by: extends ADR-0101 and ADR-0112
