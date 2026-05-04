# ADR-0107: Support-Agent Action Contract Alignment

- Status: Accepted
- Date: 2026-05-04
- Decision Origin: Design

## Context

The custom GPT support-agent bundle under `customgpt/support-agent` requires
action-returned repository evidence before answering GoCluster questions. The
initial instruction text referenced topic/search operations that were not
present in the OpenAPI action schema or Cloudflare Worker routes. That mismatch
could make normal support questions fail before the agent retrieves source
evidence.

Authentication is intentionally out of scope while the retrieval flow is being
validated.

## Decision

Align the support-agent instructions, OpenAPI schema, and Worker behavior around
the implemented retrieval model:

1. route with `getSourceMap` or `getTroubleshootingIndex`
2. retrieve authoritative files with `getDoc`
3. use `getBundle` only after concrete paths are known
4. use `getExternalAuthorities` only for directly related external tool behavior

Keep `agent-instructions.md` under the custom GPT 8000-character limit. Make
bundle retrieval all-or-error so the GPT does not receive mixed file payloads
and error objects inside a successful bundle response.

## Alternatives considered

1. Add `/topic` and `/search` endpoints.
   - Rejected for now because that would create another routing/ranking layer
     before the source-map and troubleshooting docs are proven insufficient.
2. Leave instructions aspirational and implement endpoints later.
   - Rejected because the GPT must work against the deployed action contract.
3. Fold all routing into instructions only.
   - Rejected because routing should remain grounded in repo-returned docs.

## Consequences

### Benefits

- The GPT can call only operations that exist in the action schema and Worker.
- Answers still require authoritative `getDoc` or concrete bundle evidence.
- Bundle failures are explicit instead of producing schema-shaped ambiguity.

### Risks

- The agent depends on source-map and troubleshooting-index coverage until a
  future topic/search layer is explicitly designed.
- The deployed Worker must be kept in sync with checked-in Worker changes.

### Operational impact

- No cluster runtime, config, protocol, parser, queue, timing, telnet, archive,
  peer, or replay behavior changes.
- No authentication change.

## Links

- Related issues/PRs/commits: none
- Related tests: instruction character-count check, stale operation text check,
  OpenAPI YAML parse, Worker syntax check, local Worker endpoint smoke test,
  deployed Worker smoke test
- Related docs: `customgpt/support-agent/agent-instructions.md`,
  `customgpt/support-agent/actions-schema.yaml`,
  `customgpt/support-agent/cloudflare-worker.js`,
  `customgpt/support-agent/README.md`
- Related TSRs: none
- Supersedes / superseded by: none
