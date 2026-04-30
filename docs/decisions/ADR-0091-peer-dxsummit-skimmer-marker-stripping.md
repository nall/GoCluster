# ADR-0091: Peer and DXSummit Skimmer Marker Stripping

- Status: Accepted
- Date: 2026-04-30
- Decision Origin: Design

## Context

Peer nodes and DXSummit can relay spotter identities that carry a terminal
`-#` skimmer marker. That marker identifies source class provenance, not the
licensed spotter identity. Preserving it on accepted peer or DXSummit spots
creates operator-visible spotter identities that differ from the base station
call.

DXSummit also uses a separate terminal `-@` source marker. That marker is not
part of this change.

## Decision

Strip only a terminal `-#` skimmer marker from peer and DXSummit DE/spotter
calls before creating the shared `spot.Spot`.

Examples:

- `N2WQ-#` becomes `N2WQ`
- `N2WQ-1-#` becomes `N2WQ-1`
- `N2WQ-1` remains `N2WQ-1`
- `EA5JLX-@` remains `EA5JLX-@`

The rule applies only to DE/spotter identities from peer PC11, PC61, PC26, and
DXSummit ingest. It does not change DX call normalization, RBN/local skimmer
ingest, peer hop suffix handling, source stats, queues, or forwarding policy.

## Alternatives considered

1. Strip all source markers, including `-@`.
   Rejected because the approved scope only covers `-#`; DXSummit `-@`
   display behavior remains an accepted contract.
2. Strip numeric SSIDs along with `-#`.
   Rejected because numeric SSIDs can identify a station instance and were
   explicitly outside this change.
3. Change global callsign normalization.
   Rejected because only peer and DXSummit spotter fields need this behavior.

## Consequences

- Peer and DXSummit spotter identities no longer preserve terminal `-#` marker
  provenance in accepted spots.
- Numeric SSIDs remain visible when present.
- DXSummit `-@` behavior remains unchanged.
- No YAML/config, retention, concurrency, queue, timeout, or shutdown behavior
  changes.

## Links

- Related ADRs: ADR-0066
- Related tests: `peer/parse_test.go`, `dxsummit/client_test.go`
- Related docs: `peer/README.md`, `dxsummit/README.md`, `data/config/README.md`
