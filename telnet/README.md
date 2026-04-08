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

## Grid, Noise, And Nearby

- `SET GRID` stores the user's Maidenhead grid for path reliability
- `SET NOISE` stores the user's noise class and applies a DX-to-user path penalty
- `PASS NEARBY ON` requires a grid and temporarily suspends location filters

`NEARBY` uses H3 cells:

- coarse resolution on `160m`, `80m`, and `60m`
- finer resolution on the other supported bands

## Path Display

The telnet layer asks the path predictor for a class and glyph when path display is enabled and the user has a grid.

- normal classes come from [`../pathreliability`](../pathreliability)
- optional `R` and `G` solar-weather overrides are applied afterward
- the insufficient state is preserved and is not replaced by solar overrides

For command HELP and dialect details, see [`../commands/README.md`](../commands/README.md).
