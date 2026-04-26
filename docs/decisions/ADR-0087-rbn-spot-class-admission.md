# ADR-0087: RBN Spot-Class Admission

- Status: Accepted
- Date: 2026-04-26
- Decision Origin: Troubleshooting chat

## Context
RBN history exports have two mode-like fields with different meanings:

- `mode`: RBN spot class (`CQ`, `DX`, `BEACON`, `NCDXF B`, or blank)
- `tx_mode`: RF/transmission mode (`CW`, `RTTY`, and similar explicit mode tokens)

Using the spot-class field as the cluster spot mode would confuse filter behavior and downstream correction logic. Ingesting `DX` class rows also admits second-hand DX-cluster-style reports rather than the RBN classes intended for skimmer evidence.

## Decision
RBN ingest must treat spot class and RF mode as separate contracts:

- Use `tx_mode` or the explicit RF mode token for `Spot.Mode`.
- Use RBN spot class only as an admission and beacon-tagging signal.
- Admit only `CQ`, `BEACON`, and `NCDXF B`.
- Drop `DX`, blank, and unknown spot classes before ingest/evidence.
- Tag admitted `BEACON` and `NCDXF B` rows as `IsBeacon=true`.

This applies to live RBN telnet parsing and CSV-backed historical/replay tooling. Minimal-parser human/upstream telnet feeds are unchanged.

## Alternatives considered
1. Continue ingesting every row with a nonblank RF mode.
   - Rejected because it admits `DX` class reports that are not the intended skimmer evidence.
2. Use CSV `mode` as `Spot.Mode`.
   - Rejected because `CQ`, `DX`, `BEACON`, and `NCDXF B` are spot classes, not RF transmission modes.
3. Drop all beacon-class rows.
   - Rejected because beacon reports are valid RBN observations, but downstream correction evidence must be protected by tagging them as beacons.

## Consequences
### Benefits
- Preserves `Spot.Mode` as the RF/transmission mode contract.
- Removes `DX` class rows from RBN-derived ingest/evidence.
- Keeps beacon observations explicit and filterable through existing beacon handling.
- Keeps live and replay RBN behavior aligned.

### Risks
- If a live RBN source omits spot class tokens, non-minimal RBN parsing will now drop those rows.
- Historical replay counts can change because `DX`, blank, and unknown classes no longer reach resolver evidence.

### Operational impact
- Operators should expect fewer RBN-derived spots where prior input included `DX` class rows.
- Replay manifests can show more skipped-mode rows because rejected spot classes share the existing skipped-mode bucket.
- No YAML/config migration is required.

## Links
- Related issues/PRs/commits:
- Related tests: `rbn/parse_spot_test.go`, `cmd/rbn_replay/rbn_history_test.go`, `cmd/callcorr_reveng_rebuilt/main_test.go`
- Related docs: `rbn/README.md`, `docs/rbn_replay.md`
- Related TSRs: `docs/troubleshooting/TSR-0020-rbn-mode-column-confusion.md`
- Supersedes / superseded by:
