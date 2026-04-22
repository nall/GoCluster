# Runtime Config Layout

Configuration is split by concern so you only edit the relevant file:

- `app.yaml` - server identity, stats interval, console UI, system logging, and optional dropped-call logs.
- `ingest.yaml` - RBN/PSKReporter/human/DXSummit ingest plus the shared call cache.
- `dedupe.yaml` - primary/secondary dedupe policy windows.
- `floodcontrol.yaml` - shared-ingest flood rails, actions, windows, and per-source thresholds.
- `pipeline.yaml` - call correction, harmonics, spot policy.
- `data.yaml` - CTY/FCC/skew sources, grid DB tuning, and H3 table path.
- `path_reliability.yaml` - path reliability aggregation thresholds, glyph tuning, and allowed band list.
- `runtime.yaml` - telnet server settings, buffer capacity, and filter defaults.
- `reputation.yaml` - telnet reputation gate thresholds and IPinfo/Cymru enrichment.
- `peering.yaml` - DXSpider peer configuration (inbound/outbound, ACLs, topology cache).
- `iaru_regions.yaml` - DXCC/ADIF to IARU region mapping used by final regional mode policy.
- `iaru_mode_inference.yaml` - region-aware frequency classification table for final mode labeling.

Loader behavior:
- The server defaults to this directory (`data/config`). Override with `DXC_CONFIG_PATH` to point at another directory.
- All `.yaml`/`.yml` files here are merged in lexical order; when two files set the same key, the later file wins. Keep top-level keys unique to avoid surprises.
- `floodcontrol.yaml` is required. Startup fails fast if the file is missing or the `flood_control` block is incomplete.

Telnet message tokens (usable in `runtime.yaml`):
- Pre-login `welcome_message`, `login_prompt`, `login_empty_message`, `login_invalid_message`: `<CALL>`, `<CLUSTER>`, `<DATE>`, `<TIME>`, `<DATETIME>`, `<UPTIME>`, `<USER_COUNT>`, `<LAST_LOGIN>`, `<LAST_IP>`, `<DIALECT>`, `<DIALECT_SOURCE>`, `<DIALECT_DEFAULT>`, `<DEDUPE>`, `<GRID>`, `<NOISE>`.
- Post-login `login_greeting`: same tokens as above (with real values after login).
- Input guardrails `input_too_long_message`/`input_invalid_char_message`: `<CONTEXT>`, `<MAX_LEN>`, `<ALLOWED>`.
- Dialect status `dialect_welcome_message`: `<DIALECT>`, `<DIALECT_SOURCE>`, `<DIALECT_DEFAULT>` (source labels come from `dialect_source_default_label`/`dialect_source_persisted_label`).
- Path reliability `path_status_message`: `<GRID>`, `<NOISE>`.

Input behavior:
- Human telnet input is normalized to uppercase as it is read; the echoed characters are uppercase as well.
- Telnet IAC negotiation bytes (including subnegotiation) are stripped from input before validation.

Bulletin behavior:
- `telnet.bulletin_dedupe_window_seconds` suppresses identical WWV, WCY, and `TO ALL` announcement lines across all bulletin sources before they enter client control queues.
- Set `telnet.bulletin_dedupe_window_seconds: 0` to disable bulletin dedupe.
- `telnet.bulletin_dedupe_max_entries` is the hard cap on retained bulletin keys while dedupe is enabled.

DXSummit ingest:
- `dxsummit.enabled` loader default is `false` when omitted. The checked-in `ingest.yaml` block is the operator-editable runtime config for this checkout; its explicit values override loader defaults.
- When enabled, one HTTP polling goroutine reads `dxsummit.base_url` and forwards accepted rows into the shared ingest pipeline as human `UPSTREAM` spots with `SourceNode=DXSUMMIT`.
- `dxsummit.poll_interval_seconds` controls poll cadence. The loader default is `30` when omitted; use the effective YAML value to know what this node will run.
- `dxsummit.max_records_per_poll` maps directly to the DXSummit `limit` query parameter. The shipped default is `500`; valid range is `1..10000`.
- Normal polls use `from_time=now-lookback_seconds`, `to_time=now`, `limit=max_records_per_poll`, and `include=HF,VHF,UHF`.
- `dxsummit.include_bands` is limited to `HF`, `VHF`, and `UHF`.
- `dxsummit.startup_backfill_seconds: 0` means seed-only startup: the initial page sets the high-water cursor and emits no historical rows.
- `dxsummit.spot_channel_size` and `dxsummit.max_response_bytes` bound retained queue and response memory.
- DXSummit spotter calls ending in `-@` preserve that marker for display/archive provenance, while metadata and license lookups use the base callsign without the marker.
- DXSummit latitude/longitude fields are not used to populate grids. Existing CTY and grid-cache enrichment may fill grids later from callsign-derived data.
- The console dashboard counts DXSummit in `Ingest Sources` only when `dxsummit.enabled` is true. It shows `DXSUMMIT` connected after a recent successful poll, including seed-only startup polls that emit no spots.
