# ADR-0116: Blank Beacon Comment Fallback

- Status: Accepted
- Date: 2026-05-06
- Decision Origin: Design

## Context
RBN rows carry a spot-class field that is separate from RF transmission mode.
The class values `BEACON` and `NCDXF B` identify the DX as a beacon, while
`tx_mode` still identifies the RF mode such as `CW` or `RTTY`.

Before this decision, blank beacon comments stayed blank in user-facing spot
lines. That made beacon rows less clear to telnet users even though the runtime
already knew the spot was a beacon. The requested behavior is to display
`BEACON` for blank beacon comments while preserving normal SNR/report text,
path glyphs, grid, confidence, and time columns.

Peer forwarding is a separate compatibility surface. A display fallback should
not rewrite the comment that is forwarded to peers or become a synthetic human
comment for toxicity classification.

## Decision
The spot model keeps source-class beacon state separately from comment text.
When RBN live ingest, replay, or call-correction tooling sees a source spot
class of `BEACON` or `NCDXF B`, it records that source-class beacon state and
refreshes the canonical `IsBeacon` flag from source metadata plus existing
DX/comment beacon heuristics.

Fixed-width spot formatting displays synthetic beacon text only when a spot is
a beacon and the sanitized comment is blank. Generic blank beacon comments show
`BEACON`; blank `NCDXF B` source-class beacon comments show `NCDXF BEACON`.
Explicit comments still win over the fallback.

Archive persistence creates a narrow archive-only snapshot for blank beacon
comments and stores the same synthetic fallback in newly archived rows. The
live shared spot and peer-forwarded comment remain unchanged. Existing archive
rows are not migrated.

## Alternatives considered
1. Mutate `Spot.Comment` globally when a source-class beacon is parsed.
   Rejected because it would change peer-forwarded comments and could become
   classifier input in paths that treat comments as human text.
2. Make the fallback display-only.
   Rejected because archive-backed history should show the same operator-facing
   beacon comment for newly persisted rows.
3. Add an archive schema migration for existing rows.
   Rejected because the requested behavior is forward-looking and old rows may
   not retain enough source-class evidence to reconstruct the same decision
   deterministically.

## Consequences
### Benefits
- Telnet users and archive-backed history see `BEACON` instead of a blank
  comment for blank generic beacon rows, and `NCDXF BEACON` for blank
  `NCDXF B` rows.
- RF mode and report text remain intact, so examples such as `CW 5 dB BEACON`
  keep their normal mode/report shape.
- Peer forwarding preserves the original upstream comment compatibility
  surface.

### Risks
- Older archive rows stored before this decision are not rewritten and can still
  display according to their stored fields.
- Source-class beacon state is now part of spot cloning semantics and must be
  preserved by constructors and replay tooling.

### Operational impact
- No config changes are required.
- Support answers should route blank beacon comment questions to the telnet,
  spot, and RBN READMEs.
- The behavior is a display/archive convention, not a change to RBN RF mode
  parsing or peer protocol output.

## Links
- Related issues/PRs/commits: current working tree
- Related ADRs: ADR-0087
- Related tests: `spot/spot_test.go`, `rbn/parse_spot_test.go`,
  `cmd/rbn_replay/rbn_history_test.go`,
  `internal/cluster/output_pipeline_ownership_test.go`,
  `internal/cluster/output_pipeline_toxicity_test.go`,
  `archive/record_test.go`
- Related docs: `telnet/README.md`, `spot/README.md`, `rbn/README.md`,
  `customgpt/common-questions.md`, `customgpt/source-map.md`
- Related TSRs: -
- Supersedes / superseded by: -
