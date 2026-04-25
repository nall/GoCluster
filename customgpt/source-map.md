# Source Map

Use this map to route questions to authoritative GoCluster sources. Prefer
linking or citing these docs over duplicating their content here.

| Topic | Primary source | Supporting source |
| --- | --- | --- |
| Project overview | `README.md` | `docs/OPERATOR_GUIDE.md` |
| Ready-to-run release package | `README.md` | `download/README.md` |
| Windows run steps | `docs/OPERATOR_GUIDE.md` | `README.md` |
| Linux build and service steps | `docs/OPERATOR_GUIDE.md` | `README.md` |
| Real-node configuration | `data/config/README.md` | `README.md`, `docs/OPERATOR_GUIDE.md` |
| Config loader behavior | `data/config/README.md` | `config/` tests |
| Secrets and private config safety | `data/config/README.md` | `README.md`, `docs/OPERATOR_GUIDE.md` |
| Telnet login and session behavior | `telnet/README.md` | `docs/OPERATOR_GUIDE.md` |
| Command HELP source | `commands/README.md` | `commands/processor.go`, `README.md` |
| Command dialects | `commands/README.md` | `telnet/README.md` |
| `DX`, `SHOW`, `SHOW MYDX`, `SHOW DXCC` | `commands/README.md` | `commands/processor.go` |
| `WHOSPOTSME` | `README.md` | `commands/README.md`, `spot/who_spots_me.go` |
| Filters | `telnet/README.md` | `commands/README.md`, `filter/` |
| EVENT filtering | `README.md` | `telnet/README.md`, `data/config/README.md` |
| MODE taxonomy | `data/config/README.md` | `README.md`, `spot/README.md` |
| `NEARBY` filtering | `README.md` | `telnet/README.md` |
| Dedupe policies | `README.md` | `telnet/README.md`, `dedup/` |
| Bulletin dedupe | `telnet/README.md` | `data/config/README.md` |
| Confidence tags | `README.md` | `spot/README.md` |
| Call correction | `spot/README.md` | `docs/decisions/`, `docs/troubleshooting/` |
| Path reliability tags | `README.md` | `pathreliability/README.md` |
| RBN ingest | `rbn/README.md` | `rbn/` tests |
| PSKReporter ingest | `pskreporter/README.md` | `pskreporter/` tests |
| DXSummit ingest | `dxsummit/README.md` | `data/config/README.md` |
| Peer behavior | `peer/README.md` | `docs/decisions/` |
| Repo layout | `README.md` | package READMEs |
| Developer workflow | `AGENTS.md` | `docs/WORKING_WITH_CODEX.md`, `docs/change-workflow.md` |
| Code quality | `docs/code-quality.md` | `AGENTS.md` |
| Domain contract | `docs/domain-contract.md` | `telnet/README.md`, `peer/README.md` |
| Validation commands | `docs/dev-runbook.md` | `VALIDATION.md` |
| Review and closeout | `docs/review-checklist.md` | `docs/templates/non-trivial-change-template.md` |
| ADR and TSR handling | `docs/decision-memory.md` | `docs/decision-log.md`, `docs/troubleshooting-log.md` |
| Accepted design decisions | `docs/decision-log.md` | `docs/decisions/` |
| Troubleshooting records | `docs/troubleshooting-log.md` | `docs/troubleshooting/` |

## Routing Rules

- For operator behavior, route to `README.md`, `docs/OPERATOR_GUIDE.md`, or a
  package README before using code.
- For developer behavior, route to `AGENTS.md` and the workflow docs before
  implementation advice.
- For config-sensitive answers, route to `data/config/README.md` and tell the
  user to check their effective YAML.
- For decision history, route to the ADR/TSR indexes before summarizing.
