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

For unattended service operation, set `ui.mode: headless` in the private
`app.yaml`.

Example `systemd` unit:

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
For console inspection, stop the service and run the binary manually with a UI
mode such as `ansi` or `tview-v2`:

```sh
sudo systemctl stop gocluster
cd /opt/gocluster
DXC_CONFIG_PATH=/opt/gocluster/data/config.local ./gocluster
```

Use `ui.mode: headless` again before returning to unattended service mode.

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
- `SET GRID <grid>`: set your 4-6 character Maidenhead grid.
- `SET NOISE QUIET|RURAL|SUBURBAN|URBAN|INDUSTRIAL`: set receive noise class.
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
