# GoCluster Operator Guide

This guide is for running a GoCluster node and connecting to it as a telnet
DX-cluster user. For implementation details, use the package READMEs linked
from the repository root.

## Get A Binary

The current ready-to-run release asset is Windows amd64:

```text
https://github.com/N2WQ/GoCluster/releases/latest
gocluster-windows-amd64.zip
```

Download that asset, not GitHub's automatic source-code archives. Extract it
and open `ready_to_run/`.

Linux operators currently build from source:

```sh
GOOS=linux GOARCH=amd64 go build -trimpath -o gocluster .
```

## Configure A Real Node

The packaged and checked-in `data/config` directory is public example config.
For a real node:

1. Copy the whole directory to a private complete directory, for example
   `data/config.local`.
2. Edit the private copy.
3. Start the server with `DXC_CONFIG_PATH` pointing at that directory.

Review these files before real operation:

- `app.yaml`: server node ID, console/headless mode, and logging paths.
- `runtime.yaml`: telnet port, default filters, buffers, and memory controls.
- `ingest.yaml`: RBN, PSKReporter, DXSummit, and human/manual ingest settings.
- `peering.yaml`: only if this node connects to peer clusters.
- `reputation.yaml`: only if IPinfo/Cymru reputation enrichment is enabled.
- `solarweather.yaml`: only if solar/geomagnetic path overrides are enabled.
- `data.yaml`: CTY, FCC, H3, skew, and runtime data paths.
- `spot_taxonomy.yaml`: only when changing supported modes, events, or
  PSKReporter mode routing.

Keep real callsigns, peer hosts/IPs, passwords, and service tokens out of the
public example config and out of shared archives.

At minimum, replace the public placeholder identity before connecting a real
node: change `server.node_id` in `app.yaml` from `N0CALL-1`, change the RBN
login callsigns in `ingest.yaml` from `N0CALL-1`, and update any private
upstream telnet `host` and login fields you enable. If peering is enabled,
also replace peer hosts, login callsigns, and passwords in `peering.yaml`.

## Run On Windows

From the extracted `ready_to_run` directory:

```pwsh
$env:DXC_CONFIG_PATH = "data/config.local"
.\gocluster.exe
```

From a source checkout:

```pwsh
$env:DXC_CONFIG_PATH = "data/config.local"
go run .
```

To compile from source on Windows:

```pwsh
go test ./...
go build -trimpath -o gocluster.exe .
```

## Run On Linux

Build from the repository root with Go `1.26+`:

```sh
go test ./...
GOOS=linux GOARCH=amd64 go build -trimpath -o gocluster .
```

Install the binary and the required runtime data together, for example under
`/opt/gocluster`. Keep a complete private config directory at a stable path
such as `/opt/gocluster/data/config.local`.

Runtime data commonly needed beside the binary includes `data/cty`, `data/h3`,
`data/peers/topology.db`, and `data/skm_correction/rbnskew.json` when those
inputs are used by your config.

For unattended service operation, set `ui.mode: headless` in the private
`app.yaml`.

Create the service account, install directory, binary, config, and runtime
data, then assign ownership to the service user:

```sh
sudo useradd -r -s /bin/false gocluster
sudo mkdir -p /opt/gocluster
sudo cp gocluster /opt/gocluster/
sudo cp -R data /opt/gocluster/
sudo chown -R gocluster:gocluster /opt/gocluster
```

Save this unit file as `/etc/systemd/system/gocluster.service`:

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

Enable and inspect the service:

```sh
sudo systemctl daemon-reload
sudo systemctl enable --now gocluster
sudo systemctl status gocluster
journalctl -u gocluster -f
```

The interactive local console requires the process to run in a real terminal.
For console inspection, stop the service, edit `app.yaml` in the private config
directory, change `ui.mode` to `ansi` or `tview-v2`, then run the binary
manually:

```sh
sudo systemctl stop gocluster
cd /opt/gocluster
DXC_CONFIG_PATH=/opt/gocluster/data/config.local ./gocluster
```

After inspection, set `ui.mode` back to `headless` before returning to
unattended service mode.

## Connect And Use Commands

Connect to the configured telnet port from `runtime.yaml`:

```text
telnet localhost 8300
```

Log in with your callsign. Useful first commands:

- `HELP`: show the command list.
- `HELP <command>`: show command-specific help.
- `SHOW MYDX` or `SHOW DX`: show filtered spot history.
- `SHOW DXCC <call>`: look up DXCC/ADIF and zones.
- `WHOSPOTSME [band]`: show recent spotter countries for your call.
- `SET GRID <grid>`: set your 4-6 character Maidenhead grid.
- `SET NOISE QUIET|RURAL|SUBURBAN|URBAN|INDUSTRIAL`: set receive noise class.
- `SET DIAG OFF|DEDUPE|SOURCE|CONF|PATH|MODE`: replace spot comments with compact per-session diagnostics.
- `SET SOLAR 15|30|60|OFF`: opt into or stop periodic solar summaries.
- `DIALECT`, `DIALECT LIST`, `DIALECT <go|cc>`: show or switch command dialect.
- `SHOW FILTER`: display active filters.
- `PASS <type> <list>`: allow matching spots.
- `REJECT <type> <list>`: block matching spots.
- `RESET FILTER`: restore default filters.
- `PASS NEARBY ON|OFF`: toggle nearby local-area filtering.
- `SHOW DEDUPE`: show your dedupe policy.
- `SET DEDUPE FAST|MED|SLOW`: change your dedupe policy.
- `DX <freq> <call> <comment>`: post a local human spot.
- `BYE`: disconnect.

The top-level repository README contains the generated default `HELP` output.
That block is checked against the command processor in tests.

`SET DIAG MODE` is useful when the displayed mode is surprising. It shows
`<mode>|<provenance>`, where blank modes are shown as `--`. Provenance tokens
are `SRC` source explicit, `CMT` comment explicit, `EVD` recent evidence, `FQ`
digital frequency, `RCW` regional CW default, `RVO` regional voice default,
`RMIX` regional mixed blank, `RUNK` regional unknown blank, and `UNK` unknown.

### Reading `SET DIAG PATH`

`SET DIAG PATH` replaces the normal spot comment with the path-reliability data
used for that spot. The mode/report and fixed tail columns are preserved.

The compact format omits the path class glyph because the normal path
column already shows it when path display is enabled:

```text
n<count>|w<weight>|a<age>
```

Insufficient evidence is shown as:

```text
n<count>|<reason>
```

- `n<count>` is the raw number of selected observations behind the displayed
  path decision. It is a sample-size clue, not a confidence percent.
- `w<weight>` is the rounded effective weight after decay and path selection.
  It is not SNR or dB. Weight is an evidence-strength gate, not the displayed
  path class itself.
- `a<age>` is the effective age of the selected evidence. Bare numbers are
  seconds; `m` and `h` mean minutes and hours.
- `none` means no usable selected sample existed.
- `loww` means selected samples existed, but their effective weight was below
  the configured minimum.
- `stale` means selected samples existed, but the selected evidence was too old
  for the band's freshness gate.

The fixed-width cluster format may clip the right edge of a long diagnostic
comment to keep the grid, confidence, and time columns aligned. The leftmost
fields remain the important ones: count and effective weight or reason.

Example readings:

- `n18|w7`: 18 selected raw observations, rounded effective weight 7.
- `n0|none`: no usable selected sample.
- `n1|loww`: one selected observation existed, but the effective weight was
  below the minimum.
- `n32|w1`: large raw sample count but low rounded effective weight.

## Logs And Health

System logs and optional dropped-call logs are configured in `app.yaml`.
Under `systemd`, stdout/stderr also go to journald and can be tailed with:

```sh
journalctl -u gocluster -f
```

Common startup failures are usually config-path or config-content issues:

- `DXC_CONFIG_PATH` must point at a complete config directory, not one YAML file.
- Unknown YAML files or unknown keys fail startup.
- Required YAML-owned settings and reference tables must be present.
- The default config directory is `data/config` when `DXC_CONFIG_PATH` is not set.

For config loader details, see `data/config/README.md`.
