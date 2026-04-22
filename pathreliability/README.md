# Path Reliability

This directory owns path-reliability scoring, H3 grid mapping, decaying bucket storage, and the final class/glyph mapping used by telnet path display and PATH filters.

## Operator Summary

The main landing page [`../README.md`](../README.md) explains path reliability in operator language. This file records the implementation details behind that explanation.

## Data Sources

The predictor accepts only these ingest modes:

- `FT8`
- `FT4`
- `CW`
- `RTTY`
- `PSK`
- `WSPR`

`USB` and `LSB` are display-only. They can be classified with their own thresholds, but `BucketForIngest(...)` does not ingest them into path buckets.

Path-only PSKReporter modes, such as `WSPR`, can update the predictor without entering the normal dedup, archive, telnet, or peer paths.

## Grid To H3 Mapping

The implementation converts a Maidenhead grid to the center of the represented area:

- 4-character grid: center of the `2 x 1` degree square
- 6-character grid: center of the finer subsquare

That point is mapped into:

- H3 resolution 2 for fine cells
- H3 resolution 1 for coarse cells

The runtime uses stable `uint16` proxy IDs built from the precomputed H3 tables in [`../data/h3`](../data/h3). If the grid is invalid or the mapping tables are unavailable, the cell becomes invalid and prediction can fall back to insufficient.

## FT8-Equivalent Conversion

Every accepted report is normalized to an FT8-equivalent dB value before entering the store.

The shipped config in [`../data/config/path_reliability.yaml`](../data/config/path_reliability.yaml) currently sets:

- `FT4: 0`
- `CW: -7`
- `RTTY: -7`
- `PSK: -19`
- `WSPR: -26`

After that conversion, the value is clamped to the shipped `clamp_min` and `clamp_max`, then converted from dB into linear power for storage.

## Bucket Storage

The predictor stores directional buckets keyed by:

- receiver cell
- sender cell
- band
- fine or coarse resolution

Each bucket stores:

- accumulated power
- accumulated weight
- last update time

Updates apply exponential decay using the band's half-life before adding the new sample.

The shipped config currently uses:

- half-lives ranging from `600s` on `160m` and `80m` down to `240s` on `12m`, `10m`, and `6m`
- `stale_after_half_life_multiplier: 3`
- `stale_after_seconds: 1800` as the fallback purge window
- `max_prediction_age_half_life_multiplier: 1.25` as a display/filter freshness gate

## Sample Selection And Merge

Prediction uses two directions:

- receive sample: DX to user
- transmit sample: user to DX

For each direction, `SelectSample(...)` chooses between fine and coarse evidence:

- if fine is below `min_fine_weight`, coarse wins
- if fine is above `fine_only_weight`, fine wins outright
- otherwise, fine and coarse are blended by weight

When fine and coarse evidence are blended, the selected sample age is also a
weighted effective age. A small fresh sample therefore cannot hide a large stale
regional contribution.

After sample selection, the predictor applies the freshness gate. If selected
evidence is older than `ceil(band_half_life * max_prediction_age_half_life_multiplier)`,
that direction is discarded before receive/transmit merge. A value of `0`
disables this gate. Stale positive evidence returns `INSUFFICIENT`; it does not
fade through weaker glyph tiers just because it got older.

The shipped config currently uses:

- `min_effective_weight: 0.6`
- `min_fine_weight: 5`
- `fine_only_weight: 20`
- `reverse_hint_discount: 0.5`
- `merge_receive_weight: 0.6`
- `merge_transmit_weight: 0.4`

If only one direction exists, the predictor still uses it, but discounts the effective weight with `reverse_hint_discount`.

Noise is applied only to the DX-to-user side in the power domain. The shipped
table uses P.372-17-informed operational receive penalties: low bands retain
strong local-noise penalties, while 10m and 6m are tapered because absolute
external noise falls and receiver/system noise matters more near VHF.

| Band | Quiet | Rural | Suburban | Urban | Industrial |
| --- | ---: | ---: | ---: | ---: | ---: |
| 160m | 0 | 6 | 14 | 22 | 28 |
| 80m | 0 | 5 | 13 | 20 | 26 |
| 60m | 0 | 5 | 12 | 19 | 24 |
| 40m | 0 | 4 | 11 | 17 | 22 |
| 30m | 0 | 3 | 9 | 14 | 18 |
| 20m | 0 | 3 | 7 | 11 | 15 |
| 17m | 0 | 2 | 6 | 9 | 12 |
| 15m | 0 | 2 | 5 | 8 | 11 |
| 12m | 0 | 1 | 4 | 6 | 9 |
| 10m | 0 | 1 | 3 | 5 | 7 |
| 6m | 0 | 0 | 2 | 3 | 5 |

## Class Mapping

Prediction returns either:

- `HIGH`
- `MEDIUM`
- `LOW`
- `UNLIKELY`
- `INSUFFICIENT`

`INSUFFICIENT` is returned when there is no usable sample, selected evidence is
too old for the freshness gate, or the merged effective weight stays below
`min_effective_weight`.

The shipped glyph symbols are:

- `>` for `HIGH`
- `=` for `MEDIUM`
- `<` for `LOW`
- `-` for `UNLIKELY`
- space for `INSUFFICIENT`

The shipped threshold table is:

| Mode | High | Medium | Low | Unlikely |
| --- | ---: | ---: | ---: | ---: |
| FT8 | -13 | -17 | -21 | -21 |
| FT4 | -5 | -10 | -14 | -17 |
| CW | 0 | -5 | -9 | -12 |
| RTTY | 12 | 4 | 0 | -3 |
| PSK | 5 | 0 | -4 | -7 |
| USB | 22 | 17 | 13 | 10 |
| LSB | 22 | 17 | 13 | 10 |

## Config Ownership

Runtime path reliability settings are owned by [`../data/config/path_reliability.yaml`](../data/config/path_reliability.yaml). Startup loads that file through the central config registry and fails if required settings are missing or malformed.

`DefaultConfig()` remains a package-local test/helper baseline for constructing in-memory fixtures. It is not a runtime fallback, and production behavior should be documented from YAML.

## Solar Overrides

The predictor itself returns the normal class and glyph. Optional `R` and `G` solar-weather overrides are applied later by the telnet layer.

For user-facing behavior, see [`../README.md`](../README.md) and [`../telnet/README.md`](../telnet/README.md).
