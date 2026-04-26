# Commands And HELP

This directory is the source of truth for telnet command help. The public `HELP` output is built in [`processor.go`](processor.go), not copied from a separate static document.

## What Lives Here

- command parsing for `HELP`, `DX`, `SHOW`, `BYE`, and related aliases
- per-dialect HELP catalogs
- build, dedupe, and path-glyph HELP notes injected from runtime snapshots
- history, DXCC, and build-info read paths used by `SHOW DX`, `SHOW MYDX`, `SHOW DXCC`, and `SHOW BUILD`

## HELP Source Of Truth

The top-level `HELP` output is assembled by:

- `buildHelpCatalog(...)`
- `filterHelpLines(...)`
- `pathGlyphHelpLines(...)`
- `dedupeHelpNotes(...)`

The main landing page [`../README.md`](../README.md) now includes a generated HELP block for the default `go` dialect. A test in this package checks that the README block still matches the processor output built from the shipped config files.

## Dialects

The runtime supports two command dialects:

- `go`: the default dialect shown in the main README
- `cc`: a DXSpider-style alias set with `SHOW/DX`, `SET/FILTER`, `UNSET/FILTER`, and `SET/ANN`-style shortcuts

Dialect selection is user-visible and persisted per callsign. HELP is rendered for the active dialect, so the command list and examples change with the selected dialect.

## Command Surface

The operator-facing commands handled here are:

- `HELP [topic]`
- `DX`
- `SHOW DX`
- `SHOW MYDX`
- `SHOW DXCC`
- `SHOW BUILD`
- `SHOW DEDUPE`
- `SET DEDUPE`
- `SET DIAG`
- `SET GRID`
- `SET NOISE`
- `SET PATHSAMPLES`
- `SET SOLAR`
- `RESET FILTER`
- `DIALECT`
- `BYE`

Filter mutation itself is handled in the telnet layer. This package documents those filters in HELP, but the parser for `PASS`, `REJECT`, `SHOW FILTER`, and the `cc` aliases lives under [`../telnet`](../telnet).

## Notes For Documentation

- If HELP text changes here, the main README HELP block must change too.
- Path-glyph notes should match the shipped glyph symbols from `data/config/path_reliability.yaml`.
- Dedupe HELP should match the effective secondary dedupe windows from `data/config`.

For session flow and filter behavior, see [`../telnet/README.md`](../telnet/README.md).
