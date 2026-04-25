# GoCluster DX Cluster

GoCluster is a Go-based DX cluster for amateur radio operators. It collects spots from skimmer and operator feeds, adds CTY metadata, applies protection and cleanup stages, and serves fixed-width telnet output with filtering, confidence tags, and optional path hints.

## Download Latest Ready-To-Run Binary

Compiled ready-to-run packages are published on GitHub Releases. The current
published binary package is Windows amd64:

1. Open the latest release:
   [`https://github.com/N2WQ/GoCluster/releases/latest`](https://github.com/N2WQ/GoCluster/releases/latest)
2. Download the release asset named `gocluster-windows-amd64.zip`.
3. Extract the zip and open the `ready_to_run/` directory.
4. Start with the packaged `ready_to_run/README.md`.

Do not use GitHub's automatic `Source code (zip)` or `Source code (tar.gz)`
downloads unless you want the developer source tree. Those archives are not the
ready-to-run package.

More detail is in [`download/README.md`](download/README.md).

## Configure A Real Node

The checked-in [`data/config`](data/config) directory is public example config.
For a real node, copy the whole directory to a private complete config
directory such as ignored `data/config.local`, edit that private copy, and run
with `DXC_CONFIG_PATH` pointing at the directory:

```pwsh
$env:DXC_CONFIG_PATH = "data/config.local"
```

Minimum files to review before running a real node:

- `app.yaml`: set `server.node_id`, choose local UI mode, and confirm log paths.
- `runtime.yaml`: confirm telnet port, filter defaults, buffers, and memory controls.
- `ingest.yaml`: configure RBN, PSKReporter, DXSummit, and local/human ingest settings.
- `peering.yaml`: edit only if this node peers with other clusters.
- `reputation.yaml`: edit only if IPinfo/Cymru reputation enrichment is enabled.
- `solarweather.yaml`: edit only if solar/geomagnetic path overrides are enabled.
- `data.yaml`: adjust CTY, FCC, H3, skew, and data paths if your deployment layout differs.
- `spot_taxonomy.yaml`: edit only when changing supported modes, event families, or PSKReporter mode routing.

The loader expects a complete config directory, rejects unknown YAML files and
unknown keys, and fails fast when required YAML-owned settings or reference
tables are missing. Keep private callsigns, peer hostnames/IPs, passwords, and
tokens out of committed example config.

At minimum, replace the public placeholder identity before connecting a real
node: change `server.node_id` in `app.yaml` from `N0CALL-1`, change the RBN
login callsigns in `ingest.yaml` from `N0CALL-1`, and update any private
upstream telnet `host` and login fields you enable. If peering is enabled,
also replace peer hosts, login callsigns, and passwords in `peering.yaml`.

See [`data/config/README.md`](data/config/README.md) for the full config layout.

## Run And Connect

From a ready-to-run Windows package:

```pwsh
cd ready_to_run
$env:DXC_CONFIG_PATH = "data/config.local"
.\gocluster.exe
```

From a source checkout:

```pwsh
$env:DXC_CONFIG_PATH = "data/config.local"
go run .
```

Then connect with telnet using the configured port from
`data/config/runtime.yaml`:

```text
telnet localhost 8300
```

Log in with your callsign and type `HELP`.

## Build From Source

GoCluster builds from the repo root with Go `1.26+`.

Windows amd64 binary:

```pwsh
go test ./...
go build -trimpath -o gocluster.exe .
```

Windows release-style package for local testing:

```pwsh
.\scripts\create-release.ps1 -PackageOnly -AllowDirty
```

Clean publishable Windows release package:

```pwsh
.\scripts\create-release.ps1
```

The release script refuses a dirty worktree, runs `go mod tidy -diff`, stages
`ready_to_run/`, writes `gocluster-windows-amd64.zip`, creates and pushes the
Git tag, and publishes the GitHub Release asset.

Linux amd64 binary from source:

```sh
go test ./...
GOOS=linux GOARCH=amd64 go build -trimpath -o gocluster .
```

Deploy the Linux binary together with a complete config directory and required
runtime data such as `data/cty`, `data/h3`, `data/peers/topology.db`, and
`data/skm_correction/rbnskew.json` when those inputs are used by your config.
There is not currently a published Linux ready-to-run release asset.

## Run As A Linux Service

For unattended Linux operation, use a private config directory and set
`ui.mode: headless` in that config's `app.yaml`. The interactive local console
requires a real terminal and is not shown by a normal `systemd` service.

Prepare the install path and service account before enabling the unit:

```sh
sudo useradd -r -s /bin/false gocluster
sudo mkdir -p /opt/gocluster
sudo cp gocluster /opt/gocluster/
sudo cp -R data /opt/gocluster/
sudo chown -R gocluster:gocluster /opt/gocluster
```

Create the unit file as `/etc/systemd/system/gocluster.service`:

```ini
[Unit]
Description=GoCluster DX Cluster
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=gocluster
Group=gocluster
WorkingDirectory=/opt/gocluster
Environment=DXC_CONFIG_PATH=/opt/gocluster/data/config.local
ExecStart=/opt/gocluster/gocluster
Restart=on-failure
RestartSec=5s

[Install]
WantedBy=multi-user.target
```

Typical service commands:

```sh
sudo systemctl daemon-reload
sudo systemctl enable --now gocluster
sudo systemctl status gocluster
journalctl -u gocluster -f
telnet localhost 8300
```

To inspect the interactive console on Linux, stop the service and run the
binary manually from an interactive terminal with a UI mode such as `ansi` or
`tview-v2` in your private `app.yaml`:

```sh
sudo systemctl stop gocluster
cd /opt/gocluster
DXC_CONFIG_PATH=/opt/gocluster/data/config.local ./gocluster
```

Return production service config to `ui.mode: headless` before starting it
again under `systemd`.

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
WHOSPOTSME - Show recent spotter countries.
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
PASS/REJECT MODE <list> are deltas; modes not listed are unchanged.
UNKNOWN is the MODE token for blank-mode spots.
If an item appears in both lists, block wins.

ALL keyword (type-scoped):
PASS <type> ALL - allow everything for that type
REJECT <type> ALL - block everything for most types
REJECT EVENT ALL - block only tagged EVENT spots
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

Event filters:
  EVENT recognizes the taxonomy EVENT families as standalone comment tokens or
    acronym-prefixed references such as POTA-1234. Only the event family is
    filtered; the reference remains in the comment.
  Spots with no recognized EVENT tag are not affected by EVENT filters,
    including REJECT EVENT ALL.

Path reliability glyphs:
  ">" - HIGH: favorable path.
  "=" - MEDIUM: workable path.
  "<" - LOW: weak or marginal path.
  "-" - UNLIKELY: poor path.
  " " - INSUFFICIENT: not enough recent evidence.
  PATH filters use HIGH, MEDIUM, LOW, UNLIKELY, INSUFFICIENT.

List types:
  BAND, MODE, SOURCE, EVENT, DXCALL, DECALL, DXGRID2, DEGRID2, DXCONT, DECONT
  DXZONE, DEZONE, DXDXCC, DEDXCC, CONFIDENCE, PATH

Supported modes:
  CW, FT2, FT4, FT8, JS8, LSB, USB, RTTY, MSK144, PSK, SSTV, UNKNOWN

Supported events:
  LLOTA, IOTA, POTA, SOTA, WWFF

Supported bands:
  2200m, 630m, 160m, 80m, 60m, 40m, 30m, 20m, 17m, 15m, 12m, 10m, 6m, 2m
  1.25m, 70cm, 33cm, 23cm, 13cm
```
<!-- END DEFAULT_GO_HELP -->

## What The Cluster Does

- Ingests spots from RBN CW/RTTY, RBN digital, PSKReporter, optional DXSummit HTTP polling, local `DX` commands, and optional peer feeds.
- Shows enabled ingest sources in the console dashboard; DXSummit appears as `DXSUMMIT` when enabled and recently polling.
- Normalizes callsigns, frequencies, modes, and reports before shared validation and enrichment.
- Adds CTY metadata and optional FCC license checks where that policy applies.
- Applies shared-ingest flood policy before primary dedupe using the shipped `floodcontrol.yaml` rails.
- Deduplicates and fans out spots to telnet clients with per-user filters.
- Optionally derives path-reliability glyphs from recent reports between your grid and the DX grid.

The main operator-facing configuration lives in [`data/config`](data/config). Startup uses an explicit filename registry, rejects unknown YAML files/keys, and fails fast when required YAML-owned settings or reference tables are missing. Documented zero sentinels, such as disabled keepalives or immediate broadcast delivery, are preserved as operator choices rather than replaced by code defaults. The config directory layout is described in [`data/config/README.md`](data/config/README.md).

## Repo Layout

The repo root now follows a simple ownership rule:

- `main.go` is the live binary entrypoint only.
- `internal/cluster` contains the live runtime implementation and cluster-local helpers.
- `cmd/` contains standalone tools and offline runners.
- Domain packages such as `spot`, `peer`, `telnet`, `config`, and `pathreliability` remain reusable subsystems with their own tests and package-local docs.

Historical analysis notes and protocol reference material live under [`docs/archive/analysis`](docs/archive/analysis) and [`docs/reference`](docs/reference) rather than competing with the live binary at the repo root.

## Dropped Call Logs

`logging.dropped_calls` can write optional daily files for dropped calls without changing any drop policy. The shipped config enables it; set `logging.dropped_calls.enabled: false` to disable those files. When enabled, the cluster writes separate files for bad DE/DX calls, FCC no-license drops, and harmonic suppressions under `logging.dropped_calls.dir`.

Each entry uses the same timestamped daily-file logger as the system log and records only the ingestion source, dropped role, reason, call, DE, DX, mode, and a short detail field. Frequency, category, and dashboard text are intentionally omitted.

```yaml
logging:
  dropped_calls:
    enabled: true
    dir: "data/logs/dropped_calls"
    retention_days: 7
    dedupe_window_seconds: 120
    bad_de_dx: true
    no_license: true
    harmonics: true
```

## Dedupe Policies

The cluster already removes upstream duplicates before spots reach users. `SET DEDUPE` controls the second, operator-facing dedupe stage that decides how aggressively repeated live spots are hidden in your telnet feed.

Separately, shared-ingest flood control is configured in [`data/config/floodcontrol.yaml`](data/config/floodcontrol.yaml). That stage runs before primary dedupe, is not per-user, and can `observe`, `suppress`, or `drop` by actor rail. The shipped file starts in `observe` mode on every rail, but the file itself is required at startup.

Each user can choose a policy for their own session:

- `FAST`: `120s` window. Keyed by band + DE DXCC + DE 2-character grid + DX call.
- `MED`: `300s` window. Uses the same key as `FAST`. This is the default for new users.
- `SLOW`: `480s` window. Keyed by band + DE DXCC + DE CQ zone + DX call.

In plain terms:

- `FAST` shows more repeats from the same general area.
- `MED` is the middle ground and the shipped default.
- `SLOW` suppresses more repeats because CQ zone is broader than a 2-character grid square.

Useful commands:

- `SHOW DEDUPE` shows your active policy and whether `FAST`, `MED`, and `SLOW` are enabled server-side.
- `SET DEDUPE FAST|MED|SLOW` stores your preference by callsign.
- If you request a disabled policy, the server automatically chooses the nearest enabled policy and tells you what it picked.

WWV, WCY, and `TO ALL` announcement bulletins have a separate server-wide duplicate guard because they are delivered as telnet control traffic rather than spots. The shipped `runtime.yaml` suppresses identical bulletin lines for `600s` across peer and relay sources; set `telnet.bulletin_dedupe_window_seconds: 0` to disable that behavior.

## EVENT Filtering

`PASS EVENT` and `REJECT EVENT` filter spots by comment-derived activation/event family. Supported families come from `data/config/spot_taxonomy.yaml`; the shipped config defines `LLOTA`, `IOTA`, `POTA`, `SOTA`, and `WWFF`. Spots with no recognized EVENT tag are not affected by EVENT filters.

Event recognition is intentionally family-level. A comment token such as `POTA` or `POTA-1234` marks the spot as `POTA`; the reference text stays in the comment and is not a separate filter key. Slash forms such as `POTA/SOTA` and event-specific reference grammars without the acronym prefix are not interpreted by this filter.

Useful commands:

- `PASS EVENT POTA,SOTA` shows spots tagged with either family.
- `REJECT EVENT WWFF` hides WWFF-tagged spots.
- `PASS EVENT ALL` disables EVENT filtering.
- `REJECT EVENT ALL` hides all EVENT-tagged spots; spots with no event tag still pass this filter domain.

## MODE And EVENT Taxonomy

`data/config/spot_taxonomy.yaml` is the single operator-editable table for supported MODE tokens, EVENT families, PSKReporter mode routing, and existing mode capability classes. `ingest.yaml` keeps only PSKReporter transport/runtime settings; mode admission moved to `pskreporter_route` in the taxonomy.

This is a binary+config contract. Deploy or roll back the binary and config directory together, and restart the cluster after changing taxonomy entries.

## NEARBY Filtering

`NEARBY` is a quick local-area filter for operators who want spots near their own location without building manual continent, zone, DXCC, or grid lists.

How it works:

- First set your grid with `SET GRID <4-6 char maidenhead>`.
- Turn it on with `PASS NEARBY ON`.
- While it is on, the cluster keeps spots whose DX side or DE side falls in your nearby area.

Band handling is intentionally simple:

- `160m`, `80m`, and `60m` use a coarser local area.
- All other supported bands use a finer local area.

`NEARBY` also changes how location filters behave:

- While `NEARBY` is on, the regular location filters are suspended: `DXGRID2`, `DEGRID2`, `DXCONT`, `DECONT`, `DXZONE`, `DEZONE`, `DXDXCC`, and `DEDXCC`.
- Attempts to change those filters while `NEARBY` is on are rejected with a warning.
- `PASS NEARBY OFF` restores the saved location-filter state from before `NEARBY` was enabled.

Session behavior:

- `NEARBY` persists across logins.
- The login greeting warns you when `NEARBY` is active.
- If your stored grid is missing or the H3 mapping tables are unavailable, `NEARBY` stays stored but inactive until the grid/H3 state becomes usable again.
- `SHOW FILTER` includes the current `NEARBY` state.

## Confidence Tags

Confidence tags appear in the telnet confidence column and can be filtered with `PASS CONFIDENCE` and `REJECT CONFIDENCE`.

There are two confidence families:

- Resolver-capable modes: `CW`, `RTTY`, `USB`, `LSB`
- FT corroboration modes: `FT2`, `FT4`, `FT8`

### Resolver-capable modes

- `?`: only one reporter supported the emitted call, or there was no usable corroboration.
- `S`: still a one-reporter spot, but the call has static support or recent on-band support.
- `P`: more than one reporter was involved, but support is weaker or the result is contested.
- `V`: strong multi-reporter support.
- `C`: the DX call was corrected and the corrected call passed validation.
- `B`: a correction was attempted, but the suggested call failed validation, so the original call was kept.

Operationally, think of resolver-mode glyphs this way:

- `?`: very little evidence
- `S`: only one current report, but the call already looks plausible from prior knowledge or recent local history
- `P`: some corroboration, but not strong
- `V`: strong corroboration
- `C` and `B`: correction outcomes, not raw confidence grades

### FT2, FT4, and FT8

FT modes use a separate corroboration rule. The cluster groups reports by normalized DX call, exact FT mode, and canonical dial frequency into a bounded live burst.

With the shipped defaults:

- `P` means exactly 2 unique reporters in the same burst.
- `V` means 3 or more unique reporters in the same burst.
- `S` can still appear for a one-reporter spot if static or recent support promotes it.

Local non-test `DX` self-spots are treated as operator-authoritative and are forced to `V`.

For the exact FT timing knobs, burst rules, and decision history, see [`spot/README.md`](spot/README.md).

## Path Reliability Tags

Path reliability is an optional telnet hint based on your grid, the DX grid, recent reports, and the shipped tuning in [`data/config/path_reliability.yaml`](data/config/path_reliability.yaml).

At a high level, the cluster:

1. accepts recent reports from supported path modes such as `FT8`, `FT4`, `CW`, `RTTY`, `PSK`, and `WSPR`
2. converts those reports onto a common FT8-like signal scale
3. groups them by coarse and fine geographic cells derived from Maidenhead grids
4. combines recent DX-to-you and you-to-DX evidence with decay over time
5. rejects selected evidence that is too old for the band's freshness gate
6. applies your selected noise class on the receive side using a band-specific penalty
7. maps the result to `HIGH`, `MEDIUM`, `LOW`, `UNLIKELY`, or `INSUFFICIENT`

What the classes mean to an operator:

- `HIGH`: recent evidence suggests a favorable path
- `MEDIUM`: workable path
- `LOW`: weak or marginal path
- `UNLIKELY`: poor path
- `INSUFFICIENT`: not enough recent evidence to rate it

Important operational notes:

- You need `SET GRID` for path hints to be useful.
- `SET NOISE` changes the receive-side penalty used in the calculation. The
  penalty is band-specific: low bands get stronger local-noise corrections than
  10m and 6m.
- Stale evidence becomes `INSUFFICIENT`; age alone does not demote a strong
  path through weaker glyph tiers.
- If grids are missing, evidence is stale or too weak, or the H3 tables are unavailable, the result stays `INSUFFICIENT`.
- `PATH` filters work on the class names, not on the glyph characters.

If solar-weather support is enabled, a normal path glyph can be replaced by:

- `R` for a radio-blackout override
- `G` for a geomagnetic-storm override

Those overrides never replace `INSUFFICIENT`.

For the exact thresholds, per-mode offsets, weight rules, and shipped tables, see [`pathreliability/README.md`](pathreliability/README.md).

## Deeper Docs

Implementation-heavy material now lives next to the relevant code:

- [`commands/README.md`](commands/README.md) - HELP source of truth, dialects, and command/filter behavior
- [`telnet/README.md`](telnet/README.md) - login flow, output lines, dedupe, `NEARBY`, path display, and filter persistence
- [`spot/README.md`](spot/README.md) - confidence calculation, correction flow, and FT policy knobs
- [`pathreliability/README.md`](pathreliability/README.md) - path bucket math and shipped YAML tuning
- [`rbn/README.md`](rbn/README.md) - structural RBN parsing and comment handoff
- [`pskreporter/README.md`](pskreporter/README.md) - MQTT normalization, path-only modes, and FT frequency handling
- [`dxsummit/README.md`](dxsummit/README.md) - HTTP polling, DXSummit source markers, and HF/VHF/UHF scope
- [`peer/README.md`](peer/README.md) - peer forwarding, receive-only behavior, and control-plane details

Additional operator references:

- [`data/config/README.md`](data/config/README.md)
- [`docs/OPERATOR_GUIDE.md`](docs/OPERATOR_GUIDE.md)
