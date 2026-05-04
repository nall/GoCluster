# GoCluster Support Agent

This folder contains the deployable custom GPT support-agent bundle:

- `agent-instructions.md` - instruction text pasted into the GPT builder.
- `actions-schema.yaml` - OpenAPI action schema for the Cloudflare Worker.
- `cloudflare-worker.js` - read-only Worker that retrieves public repo files
  from `N2WQ/GoCluster` on GitHub.

## Contract

The agent must retrieve GoCluster evidence through actions before answering:

1. `getSourceMap` or `getTroubleshootingIndex` for routing.
2. Use returned `routes` or `symptom_routes` when present; these are parsed
   from repo-owned Markdown routing docs, not from a separate alias table.
3. `getDoc` for the authoritative file.
4. `listDir` or `findFiles` for bounded repo-driven discovery when a route
   identifies a package neighborhood but not the exact file.
5. `getBundle` only after concrete paths are known.
6. `getExternalAuthorities` only for directly related Go, GitHub,
   Linux/systemd, or PowerShell behavior.

If route quality is weak, improve `customgpt/source-map.md` or
`customgpt/troubleshooting-index.md`. Do not add a separate hand-maintained
known-topics table, and do not turn routing docs into exhaustive file indexes.

The Worker adds optional structured fields:

- `routes` on `customgpt/source-map.md` responses.
- `symptom_routes` on `customgpt/troubleshooting-index.md` responses.

Do not document or require action operations that are not present in both
`actions-schema.yaml` and `cloudflare-worker.js`.

## Maintenance Checks

- Keep `agent-instructions.md` at 8000 characters or fewer.
- Keep schema operation IDs aligned with Worker routes.
- Keep structured routing derived from repo docs only.
- Keep `/bundle` all-or-error so the GPT never receives mixed file and error
  objects in a successful bundle response.
- Keep `/privacy` public.
- Keep large-file recovery explicit: full-file content is capped at 140000
  characters, bundles at 12 files, related paths at 80 entries, and
  `/doc`/`/file` line windows at 400 lines.
- Keep discovery bounded: `/list-dir` returns at most 80 entries and
  `/find-files` returns at most 80 filename/path matches.
- Treat `truncated: true` or `source_truncated: true` as partial evidence, not
  retrieval failure. Prefer smaller authoritative sibling files and tests
  before using a bounded line window from a large integration file.

## Authentication

All repository retrieval endpoints require:

```text
Authorization: Bearer <token>
```

Set the Worker secret binding with:

```powershell
wrangler secret put GOCLUSTER_DOCS_ACTION_TOKEN
```

Configure the GPT action authentication as a bearer/API key using the same
value. Do not commit the real token, paste it into docs, or print it in test
output. Local smoke tests should use dummy values only.
