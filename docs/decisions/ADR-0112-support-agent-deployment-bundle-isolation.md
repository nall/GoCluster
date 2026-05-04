# ADR-0112: Support-Agent Deployment Bundle Isolation

- Status: Accepted
- Date: 2026-05-04
- Decision Origin: Design

## Context

The `customgpt/support-agent/` directory stores deployable payloads for the
custom GPT and Cloudflare Worker. Those files are support tooling inputs, not
GoCluster runtime documentation. Letting the support-agent action retrieve or
discover its own deployment bundle can cause the GPT to read its own
instructions, action schema, or Worker implementation instead of using the
repo-owned routing docs and authoritative cluster sources.

## Decision

Treat `customgpt/support-agent/` as an isolated deployment bundle:

1. Keep only the three deployment payloads in the directory:
   `agent-instructions.txt`, `actions-schema.yaml`, and
   `cloudflare-worker.js`.
2. Do not keep a README or repo-documentation index inside the bundle.
3. Deny `customgpt/support-agent/*` through Worker path safety for direct file
   retrieval, bundles, directory listing, and filename discovery.
4. Keep support routing in the repo-owned `customgpt/` routing docs and the
   underlying package READMEs, ADRs, TSRs, config docs, and source comments.

## Alternatives considered

1. Add `customgpt/support-agent/` to `.gitignore`.
   - Rejected because the deployment payloads must stay versioned, and
     `.gitignore` does not prevent the action from reading committed files.
2. Keep a README in the bundle.
   - Rejected because it makes the bundle look like a crawlable documentation
     surface instead of paste/deploy input.
3. Let the GPT retrieve the action schema for debugging.
   - Rejected for the support workflow. Local repo inspection and deployment
     logs are better debugging tools for action implementation details.

## Consequences

### Benefits

- The GPT cannot loop through its own instructions or action schema through the
  GoCluster documentation action.
- The bundle remains versioned while staying outside the support evidence
  graph.
- Repo support answers continue to route through authoritative cluster docs and
  source, not deployment scaffolding.

### Risks

- The GPT cannot self-diagnose action implementation details by fetching the
  bundle through the action. Maintainers must inspect the local repo or GitHub
  directly when debugging the support-agent deployment.

### Operational impact

- No cluster runtime, telnet, config, archive, peer, replay, queue, timing, or
  long-lived connection behavior changes.
- The support-agent action retrieval contract intentionally excludes its own
  deployment bundle.

## Links

- Related issues/PRs/commits: none
- Related tests: Worker syntax check, instruction character-count check,
  support-agent bundle denial smoke tests, directory-discovery smoke tests
- Related docs: `customgpt/support-agent/agent-instructions.txt`,
  `customgpt/support-agent/actions-schema.yaml`,
  `customgpt/support-agent/cloudflare-worker.js`,
  `customgpt/source-map.md`, `customgpt/troubleshooting-index.md`
- Related TSRs: none
- Supersedes / superseded by: none
