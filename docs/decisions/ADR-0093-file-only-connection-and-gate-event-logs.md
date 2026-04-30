# ADR-0093: File-Only Connection and Gate Event Logs

- Status: Accepted
- Date: 2026-04-30
- Decision Origin: Design

## Context
Operators need durable event trails for login failures, reputation-gated manual
spots, telnet session lifecycle, ingest source lifecycle, and peer lifecycle.
The existing system log and UI panes are useful for live operation, but they mix
unrelated events and can become noisy during reconnect churn or abuse.

## Decision
Add separate daily file logs under `logging` for:

- `login_attempts`
- `reputation_drops`
- `telnet_connections`
- `ingest_connections`
- `peer_connections`

These logs are file-only. New events are not routed to UI panes or console-only
surfaces. Each log has its own directory, retention, and bounded de-dupe window
while reusing the existing daily sink pattern.

Log records use timestamped `key=value` fields with sanitized bounded values.
Passwords, raw commands, raw peer frames, and payload bodies are not logged.

## Alternatives considered
1. Put all new events in the system log
   - Simpler, but makes operational triage harder and duplicates UI/system
     noise during reconnect storms.
2. Add UI panes for the new events
   - Useful for live viewing, but the requirement is durable separate files and
     avoiding additional console/UI churn.
3. Use JSON lines
   - Easier for machines, but inconsistent with existing operator-facing daily
     log files.

## Consequences
- Operators get separate files for login, reputation, telnet lifecycle, ingest
  lifecycle, and peer lifecycle analysis.
- Flood-prone paths use bounded de-dupe state so logging does not grow by total
  historical input cardinality.
- Successful login lifecycle appears in `telnet_connections`; failed or blocked
  login attempts appear in `login_attempts`.

## Links
- Related ADRs: ADR-0048, ADR-0090
- Related TSRs: none
- Code: `internal/cluster/event_file_log.go`, `telnet/server.go`, `peer/manager.go`, `internal/cluster/ingest_health.go`
- Config/docs: `data/config/app.yaml`, `data/config/README.md`, `docs/OPERATOR_GUIDE.md`
