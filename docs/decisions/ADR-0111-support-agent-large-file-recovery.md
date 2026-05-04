# ADR-0111: Support-Agent Large-File Recovery

- Status: Accepted
- Date: 2026-05-04
- Decision Origin: Design

## Context

The support-agent Worker caps full-file content so action responses remain
bounded. Large integration files can therefore return `truncated: true` even
though the action successfully retrieved usable content, headers, related
paths, and source URL. The support agent needs to treat routing docs as sparse
neighborhood maps and then discover focused source or test files from repo
structure instead of relying on hand-maintained exhaustive route tables.

## Decision

Keep the full-file cap, but make large-file recovery explicit:

1. `truncated: true` and `source_truncated: true` are partial evidence, not
   retrieval failure.
2. Support routing remains sparse and biased toward common support questions;
   routes identify neighborhoods rather than every implementation file.
3. `/doc` and `/file` support bounded line-window retrieval through
   `start_line` and `line_count`, capped at 400 lines.
4. `/list-dir` and `/find-files` support bounded repo-driven discovery by
   directory and filename/path substring. They do not perform full-text search.
5. Responses include retrieval limit metadata so the GPT can choose a package
   README, smaller file, test, or bounded line window before refusing.

## Alternatives considered

1. Increase the full-file character cap.
   - Rejected because it moves the boundary without fixing recovery behavior
     and can make action responses too large for GPT use.
2. Add full-text search.
   - Deferred because bounded line windows plus better routes address the
     observed failure with less contract surface.
3. Add concrete source/test links to every route.
   - Rejected because `customgpt/` is a routing layer, not a stale file index.
4. Split every large source file.
   - Rejected as a support-agent workaround; source organization should follow
     runtime cohesion, not retrieval limits alone.

## Consequences

### Benefits

- The GPT can distinguish partial retrieval from retrieval failure.
- Developer support can discover focused source/test files without inventing
  code locations or expanding route tables.
- The line-window API stays bounded and deterministic.

### Risks

- The GPT must still pick relevant paths from route neighborhoods, package
  READMEs, related paths, directory listings, filename discovery, or prior
  retrieved files; discovery endpoints are not a substitute for source-map
  quality.
- GPT Builder must accept the new optional query parameters in the action
  schema and new discovery operations.

### Operational impact

- No cluster runtime, config, parser, telnet, archive, peer, replay, queue,
  timing, or long-lived connection behavior changes.
- The support-agent action contract gains optional bounded retrieval
  parameters and bounded discovery operations.

## Links

- Related issues/PRs/commits: none
- Related tests: Worker syntax check, OpenAPI YAML parse, local Worker
  truncation, line-window, directory-listing, and filename-discovery smoke
  tests, instruction character-count check
- Related docs: `customgpt/support-agent/agent-instructions.txt`,
  `customgpt/support-agent/actions-schema.yaml`,
  `customgpt/support-agent/cloudflare-worker.js`,
  `customgpt/source-map.md`, `customgpt/troubleshooting-index.md`
- Related TSRs: none
- Supersedes / superseded by: extends ADR-0108 and ADR-0109
