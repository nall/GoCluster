# ADR-0058 - PSKReporter FT Frequency Canonicalization With Preserved Observed RF

Status: Accepted
Date: 2026-04-08
Decision Makers: Cluster maintainers
Technical Area: pskreporter ingest, spot, archive, mode inference, FT confidence
Decision Origin: Troubleshooting chat
Troubleshooting Record(s): TSR-0015
Tags: pskreporter, ft8, ft4, ft2, frequency, archive, compatibility

## Context
- PSKReporter supplies a single FT frequency field that reflects observed RF frequency, not an explicit dial-frequency field.
- RBN digital ingest already uses dial frequency semantics, and `mode_inference.digital_seeds` are maintained as dial frequencies.
- The cluster reused `Spot.Frequency` across mode inference, FT corroboration, archive, telnet output, and diagnostics, so PSKReporter FT spots introduced a pipeline-wide semantic mismatch.
- Operators observed that this mismatch fragmented FT corroboration and inflated the effective digital frequency map.

## Decision
- Canonicalize PSKReporter `FT2`/`FT4`/`FT8` operational frequency at ingest to the nearest configured dial frequency from `mode_inference.digital_seeds`, subject to a bounded same-band audio-offset window.
- Keep `Spot.Frequency` as the operational pipeline frequency and use the canonical dial there when mapping succeeds.
- Preserve the raw source-reported PSKReporter RF frequency in `Spot.ObservedFrequency`; when no mapping succeeds, `ObservedFrequency` remains implicit via `Frequency`.
- Persist both operational and observed frequencies in archive records, with backward-compatible decoding for legacy records.
- Keep the dial registry immutable and derived from the same seed list used by mode inference so ingest and inference share one dial-frequency source of truth.

## Alternatives Considered
1. Leave PSKReporter FT frequencies raw and adjust only FT confidence grouping
   - Pros: smaller local change.
   - Cons: mode inference and archive semantics stay polluted.
2. Overwrite `Spot.Frequency` without preserving raw observed frequency
   - Pros: minimal storage churn.
   - Cons: loses auditability and source-truth diagnostics.
3. Add a separate canonical-frequency field and keep `Spot.Frequency` raw
   - Pros: preserves raw semantics everywhere.
   - Cons: much larger blast radius because every downstream consumer must choose between fields.

## Consequences
### Benefits
- PSKReporter FT spots now align with RBN digital and configured dial seeds on the operational path.
- FT corroboration and digital mode-inference buckets become more stable.
- Raw observed PSKReporter RF frequency remains available for diagnostics and archive introspection.

### Risks
- Canonicalization is heuristic because PSKReporter does not provide a true dial-frequency field.
- Incorrect dial tables or window assumptions can merge spots onto the wrong dial.
- Operator-visible FT frequency output changes for PSKReporter-derived FT spots.

### Operational impact
- No new goroutines, queues, or timeout behavior.
- Archive records gain an observed-frequency field and a new record version, while legacy records remain readable.
- Telnet/archive output shows canonical dial frequency for mapped PSKReporter FT spots.

## Links
- Related issues/PRs/commits:
  - pending
- Related tests:
  - `spot/ft_frequency_test.go`
  - `pskreporter/client_test.go`
  - `archive/record_test.go`
- Related docs:
  - `README.md`
- Related TSRs:
  - `docs/troubleshooting/TSR-0015-pskreporter-ft-frequency-semantic-mismatch.md`
- Supersedes / superseded by:
  - none
