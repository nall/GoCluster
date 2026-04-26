# ADR-0084: SET DIAG Modes and Path Observation Count

- Status: Accepted
- Date: 2026-04-26
- Decision Origin: Design

## Context
`SET DIAG` was a boolean telnet option that replaced the spot comment with one
opaque compact dedupe tag. The operator wanted separate diagnostic use cases
for dedupe keys, source provenance, confidence display, and path reliability
evidence while preserving the spot mode/report and the fixed-width DX cluster
tail.

The comment area is intentionally small, and the normal spot line already shows
DX call, frequency, mode/report, confidence glyph, DX grid, and time. Showing
all diagnostics at once would either truncate unpredictably or obscure the most
useful field for a given troubleshooting question.

Path reliability previously retained only decaying power, decaying weight, and
last update time per directional bucket. That made weight and age available for
path diagnostics, but not the raw number of observations that contributed to
the selected cell-pair evidence.

## Decision
Replace the boolean diagnostic state with explicit per-client diagnostic modes:

- `OFF`
- `DEDUPE`
- `SOURCE`
- `CONF`
- `PATH`
- `MODE`

Require the explicit `SET DIAG DEDUPE` command for dedupe diagnostics. The
legacy boolean-style `SET DIAG ON` form is not retained as an alias.

Diagnostic payloads are compact and replace only the free-form comment field.
They omit the diagnostic type marker because the active `SET DIAG` mode already
provides that context:

- `DEDUPE`: `<DE-DXCC>|<DE-key>|<src>|<policy>`
- `SOURCE`: `<source>`, with bounded peer labels as `P:<peer>`
- `CONF`: `<score>%` when the pipeline calculated a confidence percent,
  otherwise `--%`
- `PATH`: `n<count>|w<weight>|a<age>` for usable evidence, or
  `n<count>|<reason>` for insufficient evidence.
- `MODE`: `<mode>|<provenance>` using the final normalized mode and the
  existing mode provenance recorded on the spot.

The confidence score uses the pipeline-calculated percent when available. It is
not derived from the public confidence glyph. FT confidence currently has no
percent calculation, so FT spots can show `--%` in confidence diagnostics.

Path reliability buckets retain a raw observation count inside the existing
bucket object. The count is not decayed; decayed weight remains the freshness
and strength indicator. When fine and coarse evidence are blended, the
diagnostic count uses the larger selected layer count rather than summing both
layers, because one report can update both resolutions. Receive/transmit
directional evidence remains additive. The count is capped by its unsigned
integer type and is bounded in memory by the existing bucket maps, stale purge,
and compact paths.

## Alternatives considered
1. Keep one `SET DIAG ON|OFF` mode and extend the opaque tag.
   Rejected because the comment area cannot carry all requested diagnostics
   clearly.
2. Show raw path power/SNR in `PATH` diagnostics.
   Rejected because the predictor value is an internal power-domain model value
   and is easy to confuse with an on-air report.
3. Approximate path sample size from decayed weight.
   Rejected because weight and sample count answer different operator questions.
4. Store a separate path diagnostic side table.
   Rejected because the existing bucket object already owns the retained path
   evidence and adding a side table would create unnecessary retention risk.

## Consequences
### Benefits
- Operators can choose the diagnostic view that matches the question being
  debugged.
- Mode/report placement and fixed tail columns are preserved.
- Peer-source diagnostics can identify the actual peer within a bounded label.
- Path diagnostics distinguish sample count from decayed effective weight.
- Mode diagnostics expose whether the mode was source/comment explicit,
  inferred from reusable evidence, inferred from digital frequency, or assigned
  by regional fallback policy.

### Risks
- `CONF` can show `--%` when no pipeline confidence percent exists, including
  FT confidence because that rail currently maps unique spotter counts to glyphs
  without a percent value.
- `PATH` diagnostics add one counter field to each retained path bucket.
- `PATH` diagnostics may require path prediction work for clients who enable
  that mode.

### Operational impact
- Normal spot output is unchanged when diagnostics are off.
- Diagnostic modes remain per-client and do not change ingest, dedupe,
  confidence, filtering, archive, or peer behavior.
- Path bucket cardinality is unchanged; no new maps, indexes, caches, or queues
  are added.
- Slow-client, queue, reconnect, and shutdown behavior are unchanged.

## Links
- Related issues/PRs/commits:
- Related tests: `telnet/diag_command_test.go`, `pathreliability/normalize_test.go`
- Related docs: `README.md`, `docs/OPERATOR_GUIDE.md`, `pathreliability/README.md`
- Related TSRs:
- Supersedes / superseded by:
