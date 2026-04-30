# Troubleshooting Index

Use this file when a user reports a problem or asks why GoCluster behaved a
certain way. This is a routing layer, not the source of truth. Cite this file,
then cite the underlying repo docs or TSRs.

## Rules

- Ask for the exact symptom, command, config path, and log text when missing.
- For config-sensitive answers, tell the user to check the effective YAML loaded
  through `DXC_CONFIG_PATH`.
- For service issues, prefer local `systemctl` and `journalctl` output before
  guessing.
- For spot-output issues, check per-user telnet state before assuming ingest
  failure.
- For old incidents, route to `docs/troubleshooting-log.md` and the matching
  TSR, then verify current docs or code before treating the TSR as current
  behavior.
- Do not invent config keys, command syntax, error messages, source behavior, or
  default values.

## Symptom Routes

| Symptom or question | First checks | Route to | Do not guess |
| --- | --- | --- | --- |
| Startup fails or config load fails | Confirm `DXC_CONFIG_PATH`; confirm the config directory is complete; check for unknown `.yaml` files, missing required files, missing keys, unknown keys, malformed reference tables, and placeholder-only private settings. | `data/config/README.md`, `docs/OPERATOR_GUIDE.md`, `docs/decisions/ADR-0067-centralized-yaml-settings-enforcement.md` | Do not assume hidden Go defaults fill YAML-owned settings. |
| Linux service will not start or restarts | Run `sudo systemctl status gocluster`; inspect `journalctl -u gocluster -n 200 --no-pager`; confirm `WorkingDirectory`, `ExecStart`, `DXC_CONFIG_PATH`, file ownership, and `ui.mode: headless`. | `docs/OPERATOR_GUIDE.md`, `external-authorities.md` | Do not diagnose without service logs and the unit file. |
| Windows local run fails | Confirm Go is installed; confirm the binary or `go run .` command; set `$env:DXC_CONFIG_PATH` when using private config; read startup output. | `docs/OPERATOR_GUIDE.md`, `README.md`, `external-authorities.md` | Do not recommend Linux `systemd` steps for Windows. |
| Telnet cannot connect | Confirm the configured telnet port in `runtime.yaml`; confirm the process is running; test from the host first; check firewall or service binding only after local failure is confirmed. | `docs/OPERATOR_GUIDE.md`, `data/config/README.md`, `telnet/README.md` | Do not assume the default port if config may differ. |
| Failed or blocked login attempts are not visible in console/UI | Check `logging.login_attempts` in the effective `app.yaml`; these events are file-only daily logs and are not routed to local UI panes or console output. | `data/config/README.md`, `docs/OPERATOR_GUIDE.md`, `docs/decisions/ADR-0093-file-only-connection-and-gate-event-logs.md` | Do not tell users to look for these new events in the console or UI. |
| Login prompt or greeting is wrong | Check `runtime.yaml` message settings and token support; confirm callsign login behavior and dialect state. | `data/config/README.md`, `telnet/README.md` | Do not invent unsupported message tokens. |
| A command is unknown or behaves unlike expected | Check active dialect; use `HELP` and `HELP <command>`; route parser/help ownership to commands or telnet docs. | `commands/README.md`, `telnet/README.md`, `README.md` | Do not assume DXSpider syntax unless the `cc` dialect supports it. |
| No spots appear after login | Run `SHOW FILTER`; check `PASS`/`REJECT`, MODE `UNKNOWN`, EVENT filters, `PASS NEARBY`, dedupe policy, `SET GRID`, active sources, and effective YAML. | `telnet/README.md`, `commands/README.md`, `data/config/README.md`, `README.md` | Do not assume upstream ingest is down before checking user filters and dedupe. |
| Expected spots disappear or repeat less often | Check `SHOW DEDUPE`; check secondary dedupe policy, bulletin dedupe, spot age gates, and source-specific admission. | `telnet/README.md`, `data/config/README.md`, `README.md` | Do not describe dedupe windows without checking effective config. |
| `REJECT EVENT ALL` still shows untagged spots | Check EVENT filter docs; untagged/no-event spots are not EVENT-tagged spots. | `README.md`, `telnet/README.md`, `docs/decisions/ADR-0070-event-filters-preserve-untagged-spots.md` | Do not tell users this blocks all untagged spots. |
| `UNKNOWN` mode filtering is confusing | Check MODE filter docs and mode taxonomy; route historical confusion to TSR-0019. | `telnet/README.md`, `README.md`, `data/config/README.md`, `docs/troubleshooting/TSR-0019-mode-unknown-filter-feedback.md` | Do not conflate blank mode with an unsupported mode token. |
| Displayed mode looks wrong | Use `SET DIAG MODE`; check source/comment/frequency/regional provenance; for RBN, distinguish RF mode from RBN spot class. | `docs/OPERATOR_GUIDE.md`, `data/config/README.md`, `rbn/README.md`, `docs/troubleshooting/TSR-0020-rbn-mode-column-confusion.md` | Do not treat historical RBN `mode` and `tx_mode` fields as the same thing. |
| Confidence tag looks wrong | Check current confidence docs; for correction questions, route to spot docs, ADRs, and TSR history. | `README.md`, `spot/README.md`, `docs/decision-log.md`, `docs/troubleshooting-log.md` | Do not summarize an old ADR as current behavior without checking current docs or code. |
| `?`, `S`, `P`, `V`, `C`, or `B` is confusing | Route to confidence tag docs and current HELP output. | `README.md`, `spot/README.md`, `commands/README.md` | Do not invent new glyph meanings. |
| Path glyph is blank or weaker than expected | Confirm `SET GRID`; check `SET NOISE`, `SET PATHSAMPLES`, `SET DIAG PATH`, sample count, effective weight, stale evidence, and effective path config. | `README.md`, `docs/OPERATOR_GUIDE.md`, `pathreliability/README.md`, `data/config/PATH_PREDICTIONS.md`, `data/config/README.md` | Do not call a blank glyph a bad path; it may be insufficient, low-count, low-weight, or stale evidence. |
| `PASS NEARBY ON` hides too much or ignores location filters | Confirm user grid and NEARBY state; explain that NEARBY suspends other location filters while active. | `telnet/README.md`, `README.md` | Do not describe NEARBY as a propagation prediction. |
| RBN, PSKReporter, DXSummit, or peer spots look absent | Check the source's enabled config, source-specific README, dashboard/source visibility, and source-specific admission behavior. | `rbn/README.md`, `pskreporter/README.md`, `dxsummit/README.md`, `peer/README.md`, `data/config/README.md` | Do not assume all sources emit user-visible spots immediately after startup. |
| Reputation-gated spots, ingest lifecycle, or peer lifecycle events are not visible in console/UI | Check `logging.reputation_drops`, `logging.ingest_connections`, and `logging.peer_connections` in the effective `app.yaml`; these new event streams are separate file-only daily logs. Existing reputation console display is still controlled by reputation config. | `data/config/README.md`, `docs/OPERATOR_GUIDE.md`, `docs/decisions/ADR-0093-file-only-connection-and-gate-event-logs.md` | Do not conflate file-only event logs with existing system log or reputation console display. |
| DXSummit is connected but emits no startup spots | Check `dxsummit.startup_backfill_seconds`; seed-only startup can establish the high-water cursor without emitting historical rows. | `dxsummit/README.md`, `data/config/README.md`, `docs/decisions/ADR-0066-dxsummit-http-ingest.md` | Do not call seed-only startup an ingest failure. |
| Peer connection or duplicate bulletin behavior is surprising | Check peer family, enabled state, passwords/hosts, topology cache, and bulletin dedupe settings. | `peer/README.md`, `telnet/README.md`, `data/config/README.md`, `docs/troubleshooting/TSR-0018-peer-bulletin-duplicate-fanout.md` | Do not expose private peer hosts or passwords in examples. |
| Developer asks how to investigate a bug | Start with package README, tests, `docs/troubleshooting-log.md`, relevant TSRs, ADRs, and workflow docs. | `customgpt/developer-guide-index.md`, `docs/change-workflow.md`, `docs/decision-memory.md` | Do not give implementation steps without workflow and validation routing. |

## Evidence To Ask For

- Exact command typed, including telnet dialect if known.
- Exact startup error, log line, or telnet response.
- Active config directory from `DXC_CONFIG_PATH`.
- Relevant YAML snippet with secrets removed.
- `SHOW FILTER`, `SHOW DEDUPE`, `SHOW BUILD`, `SET DIAG <mode>` output when
  the issue is telnet-visible.
- `systemctl status gocluster` and `journalctl -u gocluster -n 200 --no-pager`
  for Linux service failures.
