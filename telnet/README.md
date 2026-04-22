# Telnet Surface

This directory owns the telnet session layer: login flow, prompt handling, filter commands, client fan-out, and path display.

## Login And Session Behavior

- Clients log in with a callsign before commands are accepted.
- The greeting can include dialect, grid, noise, and dedupe status from config.
- Dialect choice and filter state are persisted per callsign.
- If `NEARBY` is active, the login greeting warns the user.

The handshake transcript tests in this package cover the visible login sequence.

## Spot Line Format

Spot lines are formatted in [`../spot/spot.go`](../spot/spot.go) and sent by the telnet layer.

Key operator-visible facts:

- default line length is `78` characters before CRLF
- mode starts at 1-based column `40`
- the fixed tail holds:
  - grid at column `67`
  - confidence at column `72`
  - time at column `74`
- the telnet layer normalizes line endings to CRLF

The formatter keeps the right-side tail stable by truncating comment text before it can push the grid, confidence, or time columns around.

## Filters

The telnet layer owns the live filter parser and persistence rules.

- `PASS` adds to the allow list and removes from the block list
- `REJECT` adds to the block list and removes from the allow list
- if a value appears in both, block wins
- `RESET FILTER` restores configured defaults for new users

Path and confidence filters are operator-visible here:

- `PASS/REJECT CONFIDENCE` works with `?`, `S`, `C`, `P`, `V`, `B`
- `PASS/REJECT PATH` works with `HIGH`, `MEDIUM`, `LOW`, `UNLIKELY`, `INSUFFICIENT`
- `PASS/REJECT EVENT` works with `LLOTA`, `IOTA`, `POTA`, `SOTA`, `WWFF`, or `ALL`

EVENT filters are family-level. Standalone tokens such as `POTA` and acronym-prefixed references such as `POTA-1234` both match `POTA`; the reference stays in the comment and is not separately filterable.

## Dedupe Policies

The telnet server exposes per-user dedupe policy control through `SHOW DEDUPE` and `SET DEDUPE`.

- new users default to `MED`
- the selected policy is persisted per callsign
- `SHOW DEDUPE` reports the active policy and whether `FAST`, `MED`, and `SLOW` are enabled server-side
- if a user requests a disabled policy, the server falls back to the nearest enabled one and reports that in the response

The shipped policy windows are:

- `FAST`: `120s`, keyed by band + DE DXCC + DE grid2 + DX call
- `MED`: `300s`, keyed by band + DE DXCC + DE grid2 + DX call
- `SLOW`: `480s`, keyed by band + DE DXCC + DE CQ zone + DX call

This is why `SLOW` suppresses more repeats from one region than `FAST` or `MED`: CQ zone is broader than a 2-character grid square.

## Bulletin Dedupe

WWV, WCY, and `TO ALL` announcement lines are control traffic, not spots, so they do not pass through the spot dedupe pipeline.

The telnet server applies a separate all-source duplicate guard before those lines enter per-client control queues:

- `telnet.bulletin_dedupe_window_seconds` sets the duplicate window; the shipped default is `600`.
- `telnet.bulletin_dedupe_window_seconds: 0` disables bulletin dedupe.
- `telnet.bulletin_dedupe_max_entries` bounds retained bulletin keys; the shipped default is `4096`.
- The key is the normalized bulletin kind plus the exact line shown to users, after newline normalization.
- Direct talk messages are not included.

If a duplicate is suppressed, slow clients do not see another control-queue enqueue. Unique bulletins still use the normal control queue, where a full queue disconnects the client.

## Grid, Noise, And Nearby

- `SET GRID` stores the user's Maidenhead grid for path reliability
- `SET NOISE` stores the user's noise class and applies a band-specific DX-to-user path penalty
- `PASS NEARBY ON` requires a grid and keeps spots whose DX side or DE side falls in the user's nearby area

`NEARBY` uses H3 cells:

- coarse resolution on `160m`, `80m`, and `60m`
- finer resolution on the other supported bands

While `NEARBY` is active:

- the regular location filters are suspended
- attempts to change `DXGRID2`, `DEGRID2`, `DXCONT`, `DECONT`, `DXZONE`, `DEZONE`, `DXDXCC`, and `DEDXCC` are rejected with a warning
- `SHOW FILTER` reports `NEARBY: ON (location filters suspended)`

When `PASS NEARBY OFF` is used, the telnet layer restores the saved location-filter snapshot that existed before `NEARBY` was enabled.

`NEARBY` state is persisted. On login:

- the greeting warns when `NEARBY` is active
- if the user has no usable grid or H3 mapping is unavailable, the state remains stored but inactive until it can be reactivated cleanly

## Path Display

The telnet layer asks the path predictor for a class and glyph when path display is enabled and the user has a grid.

- normal classes come from [`../pathreliability`](../pathreliability)
- optional `R` and `G` solar-weather overrides are applied afterward
- the insufficient state is preserved and is not replaced by solar overrides

For command HELP and dialect details, see [`../commands/README.md`](../commands/README.md).
