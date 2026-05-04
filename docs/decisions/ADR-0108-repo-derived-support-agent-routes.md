# ADR-0108: Repo-Derived Support-Agent Routes

- Status: Accepted
- Date: 2026-05-04
- Decision Origin: Design

## Context

The support agent should route questions automatically from repository-owned
documentation instead of a separate known-topic alias table. The existing
source-map and troubleshooting-index files already own topic and symptom
routing, but the action returned them only as Markdown content and generic
related paths. That forced the GPT to infer table structure itself.

## Decision

Have the Cloudflare Worker parse repo-owned routing Markdown into optional
structured fields:

1. `routes` from `customgpt/source-map.md`
2. `symptom_routes` from `customgpt/troubleshooting-index.md`

The parsed routes are routing hints only. The GPT still must retrieve the
authoritative file with `getDoc` or a concrete `getBundle` file before
answering unless the routing document itself is the authority.

No hand-maintained known-topic table is added. Route quality is improved by
editing the repository routing docs.

## Alternatives considered

1. Add a known-topic alias table to instructions.
   - Rejected because it creates another maintained source of truth.
2. Add `/topic` and `/search` endpoints.
   - Rejected for now because source-map and troubleshooting-index parsing is a
     smaller, repo-driven step.
3. Keep only Markdown content and related paths.
   - Rejected because structured route fields reduce GPT routing ambiguity
     without duplicating knowledge.

## Consequences

### Benefits

- Routing remains driven by checked-in repo docs.
- The GPT receives machine-readable route candidates without losing the
  human-readable source documents.
- Future route quality work happens in `customgpt/source-map.md` and
  `customgpt/troubleshooting-index.md`.

### Risks

- Markdown table parsing is intentionally conservative and depends on the
  current table shapes.
- Structured fields are hints; the GPT must not answer from them without
  retrieving authoritative content.

### Operational impact

- No cluster runtime, config, protocol, parser, queue, timing, telnet, archive,
  peer, or replay behavior changes.
- No authentication change.

## Links

- Related issues/PRs/commits: none
- Related tests: instruction character-count check, OpenAPI YAML parse, Worker
  syntax check, local Worker source-map/troubleshooting route smoke tests,
  stale operation text check
- Related docs: `customgpt/support-agent/agent-instructions.md`,
  `customgpt/support-agent/actions-schema.yaml`,
  `customgpt/support-agent/cloudflare-worker.js`,
  `customgpt/support-agent/README.md`,
  `customgpt/source-map.md`, `customgpt/troubleshooting-index.md`
- Related TSRs: none
- Supersedes / superseded by: extends ADR-0107
