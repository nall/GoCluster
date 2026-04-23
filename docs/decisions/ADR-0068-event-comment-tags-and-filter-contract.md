# ADR-0068: EVENT Comment Tags and Filter Contract

- Status: Superseded
- Date: 2026-04-22
- Decision Origin: Design

## Context

Operators want to filter portable and activation spots by event family when the
family appears in normal spot comments. The relevant comments arrive from local
`DX` commands, peer frames, RBN comment text, and DXSummit rows, and the same
spot data is later used for live fan-out and archive-backed `SHOW DX` history.

Existing comment parsing already removes mode/report/time tokens while preserving
human comment text. EVENT recognition must not strip event text from output
comments, must not add work to telnet fan-out, and must remain compatible with
older archive records that do not have a persisted EVENT field.

## Decision

Support exactly these EVENT families: `LLOTA`, `IOTA`, `POTA`, `SOTA`, and
`WWFF`.

Recognize each family as either a standalone comment token or an acronym-prefixed
reference token such as `POTA-1234`, `SOTA-ABC`, or `WWFF-1234`. Store and filter
only the family. The reference text remains only in the original comment.

Do not recognize slash forms or event-specific reference grammars without the
acronym prefix in this version.

Store spot EVENT metadata as a fixed bitmask, persist it in a new archive record
version, and derive it from preserved comments when decoding older archive
records. Add map-backed per-user EVENT allow/block filters so they follow the
same command and YAML shape as other filter domains.

## Alternatives considered

1. Match only standalone acronyms.
   - Rejected because real comments include acronym-prefixed references such as
     `POTA-1234`.
2. Persist and filter specific reference values.
   - Rejected for this version because reference grammars differ by event family
     and would expand the command and archive contract beyond the requested
     family-level filter.
3. Derive EVENT tags during every filter evaluation from comment text.
   - Rejected because it would parse comments in live fan-out and history scans.

## Consequences

### Benefits

- Operators can use `PASS EVENT` and `REJECT EVENT` for common activation
  families across live and history output.
- Output comments remain unchanged, including event references.
- Fan-out matching is bounded and allocation-free once a spot is parsed.

### Risks

- Family-only filtering cannot distinguish `POTA-1234` from another POTA
  reference.
- Some real-world event references without acronym prefixes are intentionally not
  recognized until their grammar is explicitly designed.
- Archive records gain a new version and field.

### Operational impact

- No new goroutines, queues, timers, network paths, or shutdown steps.
- Per-user filter YAML gains bounded EVENT maps and booleans.
- Legacy archive records remain readable and derive EVENT tags from preserved
  comments when possible.

## Links

- Related issues/PRs/commits:
- Related tests:
  - `spot/comment_parser_test.go`
  - `filter/filter_test.go`
  - `filter/user_record_test.go`
  - `telnet/server_filter_test.go`
  - `archive/record_test.go`
  - `archive/recent_filtered_test.go`
- Related docs:
  - `README.md`
  - `telnet/README.md`
- Related TSRs:
- Supersedes / superseded by:
  - Superseded by ADR-0070
