# Developer Guide Index

Use this index for Go developers who want to understand, debug, or extend
GoCluster. It routes to existing sources and avoids duplicating the repo's
workflow rules.

## First Stops

- Start with `README.md` for repo layout and package ownership.
- Read the relevant package README before reading code.
- Use `AGENTS.md` and `docs/change-workflow.md` before planning changes.
- Use `docs/decision-log.md` and `docs/troubleshooting-log.md` before changing
  behavior with decision history.

## Package Ownership

| Area | Start here |
| --- | --- |
| Live runtime | `internal/cluster/`, `README.md` |
| Telnet sessions, filters, and queues | `telnet/README.md`, `telnet/` |
| Command HELP and command dispatch | `commands/README.md`, `commands/` |
| Spot record, formatting, confidence, correction | `spot/README.md`, `spot/` |
| Path reliability | `pathreliability/README.md`, `pathreliability/` |
| Config loading and validation | `data/config/README.md`, `config/` |
| RBN ingest | `rbn/README.md`, `rbn/` |
| PSKReporter ingest | `pskreporter/README.md`, `pskreporter/` |
| DXSummit ingest | `dxsummit/README.md`, `dxsummit/` |
| Peer protocol and forwarding | `peer/README.md`, `peer/` |
| Reputation gate and lookups | `reputation/`, `data/config/reputation.yaml` |
| UI/dashboard | `ui/`, `internal/cluster/dashboard.go` |

## Workflow

- Small vs Non-trivial classification is owned by `AGENTS.md` and
  `docs/change-workflow.md`.
- Non-trivial work requires a Scope Ledger and exact `Approved vN` before code.
- Config, protocol, parser, concurrency, queue, retained-state, hot-path, or
  operator-visible changes are normally Non-trivial.
- Workflow-doc or repo-managed skill edits require the workflow-drift audit in
  `docs/change-workflow.md`.

## Audits And Risk Areas

| Change area | Required routing |
| --- | --- |
| YAML/config/defaults/schema | `docs/change-workflow.md`, `data/config/README.md` |
| Retained maps/caches/stores/indexes | `docs/code-quality.md` |
| Hot paths, fan-out, parsing loops, queues | `docs/code-quality.md`, `docs/dev-runbook.md` |
| Concurrency, lifecycle, shutdown, timers | `docs/domain-contract.md`, `docs/dev-runbook.md` |
| Protocol/parser behavior | `docs/domain-contract.md`, package tests |
| Operator-visible behavior | `README.md`, package README, HELP/docs tests |
| Decisions or reversals | `docs/decision-memory.md`, `docs/decision-log.md` |
| Troubleshooting or incident learnings | `docs/decision-memory.md`, `docs/troubleshooting-log.md` |

## Validation

Use `docs/dev-runbook.md` as the command source. Use `VALIDATION.md` as the
Non-trivial compliance rubric.

- Small changes need targeted checks and normally `go test ./...`.
- Non-trivial changes need the full runbook sequence.
- Race checks are mandatory for concurrency, queues, timers, cancellation,
  lifecycle, long-lived connections, or shared mutable state.
- Fuzzing is expected for parser/protocol work.
- Benchmarks and pprof are expected for hot-path or performance claims.

## Answering Developer Questions

For custom GPT responses:

- Route to docs and tests before giving implementation advice.
- Say when a change likely triggers Non-trivial workflow.
- Say when effective YAML or current code must be inspected.
- Do not summarize old ADRs as current behavior without checking the current
  doc/code path.
