# GoCluster DX Cluster

GoCluster is a Go-based DX cluster for amateur radio operators. It collects spots from skimmer and operator feeds, adds CTY metadata, applies protection and cleanup stages, and serves fixed-width telnet output with filtering, confidence tags, and optional path hints.

## HELP

The section below mirrors the default `go` dialect `HELP` output from [`commands/processor.go`](commands/processor.go) using the shipped config in [`data/config`](data/config).

<!-- BEGIN DEFAULT_GO_HELP -->
```text
Available commands:
HELP - Show command list or command-specific help.
DX - Post a spot (human entry).
SHOW DX - Alias of SHOW MYDX.
SH DX - Alias of SHOW DX.
SHOW MYDX - Show filtered spot history.
SHOW DXCC - Look up DXCC/ADIF and zones.
SHOW DEDUPE - Show dedupe policy.
SET DEDUPE - Select dedupe policy.
SET DIAG - Toggle diagnostic comments.
SET GRID - Set your grid (4-6 chars).
SET NOISE - Set noise class.
PASS NEARBY - Toggle nearby filtering.
SHOW FILTER - Display filter state.
PASS - Allow filter matches.
REJECT - Block filter matches.
RESET FILTER - Reset filters to defaults.
DIALECT - Show or switch dialect.
BYE - Disconnect.
Type HELP <command> for details.

Filter core rules:
PASS <type> <list> adds to allowlist and removes from blocklist.
REJECT <type> <list> adds to blocklist and removes from allowlist.
If an item appears in both lists, block wins.

ALL keyword (type-scoped):
PASS <type> ALL - allow everything for that type
REJECT <type> ALL - block everything for that type
RESET FILTER resets all filters to configured defaults for new users.

Feature toggles (not list-based):
PASS BEACON | REJECT BEACON
PASS WWV | REJECT WWV
PASS WCY | REJECT WCY
PASS ANNOUNCE | REJECT ANNOUNCE
PASS SELF | REJECT SELF
PASS NEARBY ON|OFF

Confidence glyphs:
  ? - One reporter only; no prior/static support promoted it to S.
  S - One reporter only, but the call has static or recent on-band support.
  P - Resolver modes: lower-confidence multi-spotter support. FT modes:
    corroboration burst support at or above the configured P threshold but
    below the configured V threshold.
  V - Resolver modes: higher-confidence multi-spotter support. FT modes:
    corroboration burst support at or above the configured V threshold.
  C - The call was corrected.
  B - A correction was attempted, but base-call or CTY validation failed, so
    the original call was kept.

Path reliability glyphs:
  ">" - HIGH: favorable path.
  "=" - MEDIUM: workable path.
  "<" - LOW: weak or marginal path.
  "-" - UNLIKELY: poor path.
  " " - INSUFFICIENT: not enough recent evidence.
  PATH filters use HIGH, MEDIUM, LOW, UNLIKELY, INSUFFICIENT.

List types:
  BAND, MODE, SOURCE, DXCALL, DECALL, DXGRID2, DEGRID2, DXCONT, DECONT, DXZONE
  DEZONE, DXDXCC, DEDXCC, CONFIDENCE, PATH

Supported modes:
  CW, FT2, FT4, FT8, JS8, LSB, USB, RTTY, MSK144, PSK, SSTV, UNKNOWN

Supported bands:
  2200m, 630m, 160m, 80m, 60m, 40m, 30m, 20m, 17m, 15m, 12m, 10m, 6m, 2m
  1.25m, 70cm, 33cm, 23cm, 13cm
```
<!-- END DEFAULT_GO_HELP -->

## What The Cluster Does

- Ingests spots from RBN CW/RTTY, RBN digital, PSKReporter, local `DX` commands, and optional peer feeds.
- Normalizes callsigns, frequencies, modes, and reports before shared validation and enrichment.
- Adds CTY metadata and optional FCC license checks where that policy applies.
- Deduplicates and fans out spots to telnet clients with per-user filters.
- Optionally derives path-reliability glyphs from recent reports between your grid and the DX grid.

The main operator-facing configuration lives in [`data/config`](data/config). The config directory layout is described in [`data/config/README.md`](data/config/README.md).

## Quick Start

1. Install Go `1.25+`.
2. Review the config files in [`data/config`](data/config).
   At minimum, set your callsigns in `ingest.yaml` and your telnet port in `runtime.yaml`.
3. Run:

   ```pwsh
   go mod tidy
   go run .
   ```

4. Connect with telnet:

   ```text
   telnet localhost 9300
   ```

5. Log in with your callsign and type `HELP`.

You can point the server at another config directory with `DXC_CONFIG_PATH`.

## Confidence Tags

Confidence tags appear in the telnet confidence column and can be filtered with `PASS CONFIDENCE` and `REJECT CONFIDENCE`.

There are two confidence paths in the code:

- Resolver-capable modes: `CW`, `RTTY`, `USB`, `LSB`
- FT corroboration modes: `FT2`, `FT4`, `FT8`

### Resolver-capable modes

For resolver-capable modes, the runtime builds a resolver snapshot for the call that will actually be emitted. The current implementation is in [`internal/correctionflow/shared.go`](internal/correctionflow/shared.go) and [`main.go`](main.go).

- `?`: only one reporter supported the emitted call, or there was no usable resolver snapshot.
- `P`: multiple reporters were present, but the emitted call's confidence was 50% or lower.
- `V`: multiple reporters were present and the emitted call's confidence was 51% or higher.
- Split or uncertain resolver states are intentionally conservative: one-reporter cases stay `?`; multi-reporter contested cases downgrade to `P`, not `V`.

The runtime then applies a floor:

- `S`: the spot was still `?`, but the DX call has static support or recent on-band support.

The runtime can also replace the usual glyph entirely:

- `C`: the DX call was corrected and the corrected call passed validation.
- `B`: a correction was attempted, but the suggested call failed base-call or CTY validation, so the original call was kept.

Operationally, think of resolver-mode glyphs this way:

- `?`: very little evidence
- `S`: only one current report, but the call already looks plausible from prior knowledge or recent local history
- `P`: some corroboration, but not strong
- `V`: strong corroboration
- `C` and `B`: correction outcomes, not raw confidence grades

### FT2, FT4, and FT8

FT modes do not use the resolver mutation path for confidence. They use a separate bounded corroboration stage in the main output pipeline.

The live burst key is:

- normalized DX call
- exact FT mode
- canonical dial frequency

The current behavior is defined by [`main.go`](main.go), with the governing decisions recorded in ADR-0060 and ADR-0061 under [`docs/decisions`](docs/decisions).

Burst handling works like this:

- a new burst starts when the first matching FT report arrives
- the burst stays open while new corroborating reports keep arriving inside the quiet-gap window
- the burst releases at `min(last_seen + quiet_gap, first_seen + hard_cap)`
- PSKReporter and RBN-digital can corroborate each other when the burst key matches and the spotter calls differ

The shipped config in [`data/config/pipeline.yaml`](data/config/pipeline.yaml) currently exposes these knobs:

- `call_correction.p_min_unique_spotters`
- `call_correction.v_min_unique_spotters`
- `call_correction.ft8_quiet_gap_seconds`
- `call_correction.ft8_hard_cap_seconds`
- `call_correction.ft4_quiet_gap_seconds`
- `call_correction.ft4_hard_cap_seconds`
- `call_correction.ft2_quiet_gap_seconds`
- `call_correction.ft2_hard_cap_seconds`

With the current shipped defaults:

- `P` means exactly 2 unique reporters in the burst
- `V` means 3 or more unique reporters in the burst
- FT8 uses `6s` quiet gap and `12s` hard cap
- FT4 uses `5s` quiet gap and `10s` hard cap
- FT2 uses `3s` quiet gap and `6s` hard cap

The `S` floor can still promote a single-reporter FT spot when the call has static or recent support. Local non-test `DX` self-spots are treated as operator-authoritative in the live runtime path and are forced to `V`.

For the full confidence pipeline, including correction rails and related config, see [`spot/README.md`](spot/README.md).

## Path Reliability Tags

Path reliability is optional telnet output driven by your grid, the DX grid, recent reports, and the shipped tuning in [`data/config/path_reliability.yaml`](data/config/path_reliability.yaml).

Important distinction:

- The values below describe the shipped config file.
- Code fallbacks also exist in [`pathreliability/config.go`](pathreliability/config.go).
- Those two are not always the same, so operator docs should follow the shipped config unless you have changed it locally.

### What feeds the predictor

The predictor accepts only these ingest modes:

- `FT8`
- `FT4`
- `CW`
- `RTTY`
- `PSK`
- `WSPR`

Voice modes `USB` and `LSB` are display-only for path classification. They can be classified for output and filtering, but they do not create path buckets.

Some PSKReporter modes can also be configured as path-only modes. When that happens, those spots update path reliability and bypass dedup, telnet, archive, and peer output.

### How a spot becomes a path sample

For a spot to contribute to path reliability, the runtime needs:

- path reliability enabled
- a supported ingest mode
- a real report/SNR
- valid DX and DE grids
- a valid band inside the allowed band list
- valid H3 mapping for at least fine or coarse cells

If those conditions are met:

1. The SNR is converted to an FT8-equivalent dB value.
2. The value is clamped to the shipped range `-25 dB` to `35 dB`.
3. The clamped dB value is converted to linear power.
4. The sample is stored as a directional path bucket with exponential decay.

The shipped mode offsets are:

- `FT4: 0`
- `CW: -7`
- `RTTY: -7`
- `PSK: -19`
- `WSPR: -26`

That means the predictor first normalizes different modes onto one FT8-like scale before comparing them.

### How grids become cells

The predictor converts a Maidenhead grid to the center of that grid square and then maps that point into:

- H3 resolution 2 for finer local buckets
- H3 resolution 1 for coarser regional buckets

The H3 proxy tables live in [`data/h3`](data/h3). The table format and regeneration notes are documented in [`data/h3/README.md`](data/h3/README.md).

### How the score is merged

Prediction uses two directional hints:

- DX to you
- you to DX

The current shipped logic then:

- prefers fine cells only when fine weight is strong enough
- falls back to coarse cells when fine evidence is too weak
- discounts one-direction-only hints with `reverse_hint_discount: 0.5`
- merges DX-to-you and you-to-DX using shipped weights `0.6` and `0.4`
- applies your selected noise penalty only to the DX-to-you side

The shipped noise penalties are:

- `QUIET: 0 dB`
- `RURAL: 4 dB`
- `SUBURBAN: 12 dB`
- `URBAN: 17 dB`
- `INDUSTRIAL: 20 dB`

The shipped weight gates are:

- `min_effective_weight: 0.6`
- `min_fine_weight: 5`
- `fine_only_weight: 20`

If the merged weight stays below `min_effective_weight`, the predictor does not guess. It returns the insufficient glyph instead.

### How classes become glyphs

The telnet glyphs are mapped from path classes. In the shipped config, the symbols are:

- `>` = `HIGH`
- `=` = `MEDIUM`
- `<` = `LOW`
- `-` = `UNLIKELY`
- space = `INSUFFICIENT`

The shipped per-mode threshold table is:

| Mode | High | Medium | Low | Unlikely |
| --- | ---: | ---: | ---: | ---: |
| FT8 | -13 | -17 | -21 | -21 |
| FT4 | -5 | -10 | -14 | -17 |
| CW | 0 | -5 | -9 | -12 |
| RTTY | 12 | 4 | 0 | -3 |
| PSK | 5 | 0 | -4 | -7 |
| USB | 22 | 17 | 13 | 10 |
| LSB | 22 | 17 | 13 | 10 |

For display, the classing rule is straightforward:

- at or above the `high` threshold: `HIGH`
- else at or above `medium`: `MEDIUM`
- else at or above `low`: `LOW`
- otherwise: `UNLIKELY`

`INSUFFICIENT` is separate. It means the runtime did not have enough usable weight or geometry to rate the path at all.

### When the result stays insufficient

The predictor stays at the insufficient glyph when:

- path reliability is disabled
- no usable path samples exist
- the merged weight is below `min_effective_weight`
- the band is unsupported or missing
- the grids are missing or invalid
- the H3 tables are unavailable

### Optional solar overrides

If solar weather support is enabled, the normal path glyph can be replaced by:

- `R` for a radio-blackout override on a relevant sunlit path
- `G` for a geomagnetic-storm override on a relevant high-latitude path

Those overrides never replace the insufficient glyph.

For the full scoring path, bucket math, and config/fallback details, see [`pathreliability/README.md`](pathreliability/README.md).

## Deeper Docs

Implementation-heavy material now lives next to the relevant code:

- [`commands/README.md`](commands/README.md) - HELP source of truth, dialects, and command/filter behavior
- [`telnet/README.md`](telnet/README.md) - login flow, output lines, path display, and filter persistence
- [`spot/README.md`](spot/README.md) - confidence calculation, correction flow, and related policy knobs
- [`pathreliability/README.md`](pathreliability/README.md) - path bucket math and shipped versus fallback tuning
- [`rbn/README.md`](rbn/README.md) - structural RBN parsing and comment handoff
- [`pskreporter/README.md`](pskreporter/README.md) - MQTT normalization, path-only modes, and FT frequency handling
- [`peer/README.md`](peer/README.md) - peer forwarding, receive-only behavior, and control-plane details

Additional operator references:

- [`data/config/README.md`](data/config/README.md)
- [`docs/OPERATOR_GUIDE.md`](docs/OPERATOR_GUIDE.md)
