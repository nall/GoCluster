# Operator Guide Index

Use this index for operators and telnet users. It routes support questions to
existing operator-facing docs.

## Install And Run

| Need | Start here |
| --- | --- |
| Download the ready-to-run package | `README.md`, `download/README.md` |
| Configure a real node | `docs/OPERATOR_GUIDE.md`, `data/config/README.md` |
| Run on Windows | `docs/OPERATOR_GUIDE.md`, `README.md` |
| Build or run on Linux | `docs/OPERATOR_GUIDE.md`, `README.md` |
| Run under `systemd` | `docs/OPERATOR_GUIDE.md`, `README.md` |
| View logs and health | `docs/OPERATOR_GUIDE.md`, `README.md` |

## Config

| Need | Start here |
| --- | --- |
| Understand `DXC_CONFIG_PATH` | `data/config/README.md` |
| Know which YAML file owns a setting | `data/config/README.md` |
| Fix unknown file/key startup errors | `data/config/README.md` |
| Keep secrets out of public config | `data/config/README.md`, `docs/OPERATOR_GUIDE.md` |
| Change supported modes or events | `data/config/README.md`, `README.md` |

## Telnet Use

| Need | Start here |
| --- | --- |
| Connect and log in | `docs/OPERATOR_GUIDE.md`, `telnet/README.md` |
| Find available commands | `README.md`, `commands/README.md` |
| Get command-specific help | `commands/README.md` |
| Understand dialects | `commands/README.md` |
| Post spots | `commands/README.md` |
| Show recent filtered history | `commands/README.md` |
| Look up DXCC/ADIF/zones | `commands/README.md` |
| Show recent spotter countries | `README.md`, `commands/README.md` |

## Filtering And Output

| Need | Start here |
| --- | --- |
| `PASS`, `REJECT`, `SHOW FILTER`, `RESET FILTER` | `telnet/README.md`, `README.md` |
| MODE filtering and `UNKNOWN` | `telnet/README.md`, `README.md` |
| EVENT filtering | `README.md`, `telnet/README.md` |
| `NEARBY` filtering | `README.md`, `telnet/README.md` |
| Dedupe policies | `README.md`, `telnet/README.md` |
| Confidence tags | `README.md`, `spot/README.md` |
| Path reliability tags | `README.md`, `pathreliability/README.md` |
| Spot line format | `telnet/README.md`, `spot/spot.go` |

## Sources

| Need | Start here |
| --- | --- |
| What the cluster ingests | `README.md` |
| RBN behavior | `rbn/README.md` |
| PSKReporter behavior | `pskreporter/README.md` |
| DXSummit behavior | `dxsummit/README.md`, `data/config/README.md` |
| Peer behavior | `peer/README.md` |

## Troubleshooting

Start with `customgpt/troubleshooting-index.md` for symptom-based routing.

| Symptom | Start here |
| --- | --- |
| Startup or config load failure | `customgpt/troubleshooting-index.md`, `data/config/README.md` |
| Linux service failure | `customgpt/troubleshooting-index.md`, `docs/OPERATOR_GUIDE.md` |
| Telnet cannot connect | `customgpt/troubleshooting-index.md`, `docs/OPERATOR_GUIDE.md`, `telnet/README.md` |
| Missing spots after login | `customgpt/troubleshooting-index.md`, `telnet/README.md`, `data/config/README.md` |
| Surprising mode, confidence, or path glyph | `customgpt/troubleshooting-index.md`, `docs/OPERATOR_GUIDE.md`, package README |
| Source-specific missing spots | `customgpt/troubleshooting-index.md`, `rbn/README.md`, `pskreporter/README.md`, `dxsummit/README.md`, `peer/README.md` |
