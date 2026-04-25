# Common Questions

This file is an index of likely questions. Answers should route to the source
docs listed here instead of restating the full behavior.

## Operator Questions

| Question | Route to |
| --- | --- |
| Where do I download GoCluster? | `README.md`, `download/README.md` |
| Which release asset should I use? | `README.md`, `docs/OPERATOR_GUIDE.md` |
| Why should I not use GitHub's source archive? | `README.md`, `download/README.md` |
| How do I configure a real node? | `data/config/README.md`, `docs/OPERATOR_GUIDE.md` |
| What is `DXC_CONFIG_PATH`? | `data/config/README.md` |
| Why does config load fail? | `data/config/README.md` |
| How do I run on Windows? | `docs/OPERATOR_GUIDE.md`, `README.md` |
| How do I run under Linux `systemd`? | `docs/OPERATOR_GUIDE.md`, `README.md` |
| How do I connect by telnet? | `docs/OPERATOR_GUIDE.md`, `telnet/README.md` |
| What commands should I try first? | `docs/OPERATOR_GUIDE.md`, `README.md` |
| What does `HELP <command>` show? | `commands/README.md`, `README.md` |
| What is the difference between `go` and `cc` dialects? | `commands/README.md` |
| How do I post a spot? | `commands/README.md`, `docs/OPERATOR_GUIDE.md` |
| How do I view recent spots? | `commands/README.md`, `docs/OPERATOR_GUIDE.md` |
| What does `WHOSPOTSME` show? | `README.md`, `commands/README.md` |
| How do `PASS` and `REJECT` work? | `telnet/README.md`, `README.md` |
| Why are no-event spots still visible after `REJECT EVENT ALL`? | `README.md`, `telnet/README.md` |
| What does `UNKNOWN` mean for mode filters? | `telnet/README.md`, `README.md` |
| What does `PASS NEARBY ON` do? | `README.md`, `telnet/README.md` |
| Why do I need `SET GRID`? | `README.md`, `telnet/README.md`, `pathreliability/README.md` |
| What do `FAST`, `MED`, and `SLOW` dedupe mean? | `README.md`, `telnet/README.md` |
| What do `?`, `S`, `P`, `V`, `C`, and `B` mean? | `README.md`, `spot/README.md` |
| What do path glyphs mean? | `README.md`, `pathreliability/README.md` |
| Why are path glyphs blank? | `README.md`, `pathreliability/README.md` |
| What sources feed spots into the cluster? | `README.md`, `rbn/README.md`, `pskreporter/README.md`, `dxsummit/README.md`, `peer/README.md` |

## Developer Questions

| Question | Route to |
| --- | --- |
| Where is the live binary entrypoint? | `README.md`, `main.go`, `internal/cluster/` |
| Which package owns a behavior? | `README.md`, package READMEs, `source-map.md` |
| How do I classify a change? | `AGENTS.md`, `docs/change-workflow.md` |
| When is `Approved vN` required? | `AGENTS.md`, `docs/change-workflow.md` |
| What is Current-State Discovery? | `docs/change-workflow.md` |
| When is a Config Contract Audit required? | `docs/change-workflow.md`, `data/config/README.md` |
| What are the bounded-state rules? | `docs/code-quality.md` |
| What are the hot-path rules? | `docs/code-quality.md`, `docs/dev-runbook.md` |
| What are the queue/fan-out/shutdown contracts? | `docs/domain-contract.md` |
| What tests should I run? | `docs/dev-runbook.md`, `VALIDATION.md` |
| When is `go test -race ./...` required? | `docs/dev-runbook.md`, `VALIDATION.md` |
| When are fuzzing or benchmarks required? | `docs/dev-runbook.md` |
| How should I do a review pass? | `docs/review-checklist.md` |
| When do I need an ADR or TSR? | `docs/decision-memory.md` |
| Where do I find prior decisions? | `docs/decision-log.md`, `docs/decisions/` |
| Where do I find incident/troubleshooting history? | `docs/troubleshooting-log.md`, `docs/troubleshooting/` |
| Where should I start before changing telnet filters? | `telnet/README.md`, `commands/README.md`, `filter/`, `docs/change-workflow.md` |
| Where should I start before changing confidence or correction? | `spot/README.md`, `docs/decision-log.md`, `docs/troubleshooting-log.md` |
| Where should I start before changing path reliability? | `pathreliability/README.md`, `data/config/path_reliability.yaml`, `docs/decision-log.md` |
