# ADR-0098: Support-Agent Raw GitHub Source URLs

Status: Accepted
Date: 2026-05-02
Decision Origin: Design

## Context

The support-agent docs in `customgpt/` route users to authoritative GoCluster
docs and source files. Those routes mostly used repository-relative paths,
which are readable in a checkout but less convenient for a retrieval-oriented
support agent.

## Decision

No durable decision change.

This change updates support-agent routing references to point at raw GitHub
file URLs under `https://raw.githubusercontent.com/N2WQ/GoCluster/main/`.
It preserves the existing contract that `customgpt/` is a routing layer and
not a second maintained copy of runtime, operator, or developer behavior.

## Alternatives considered

1. Keep repository-relative paths and let the support agent resolve them.
2. Use regular GitHub blob URLs.
3. Pin links to a specific commit SHA.

## Consequences

### Benefits

- Support-agent retrieval can fetch linked docs directly from raw GitHub.
- The linked branch tracks current `main` documentation.
- Directory-only package hints remain plain paths instead of pretending a raw
  directory URL exists.

### Risks

- Links track `main`, so older deployed binaries may need current-code caveats.
- Long raw URLs make routing tables wider.

### Operational impact

- No runtime behavior, config schema, parser behavior, command behavior,
  protocol behavior, or operator setting changed.

## Links

- Related issues/PRs/commits:
- Related tests: raw-link text checks over `customgpt/`
- Related docs: `customgpt/README.md`, `customgpt/source-map.md`,
  `customgpt/common-questions.md`, `customgpt/operator-guide-index.md`,
  `customgpt/troubleshooting-index.md`, `customgpt/developer-guide-index.md`,
  `customgpt/gpt-instructions.md`
- Related TSRs:
- Supersedes / superseded by:
