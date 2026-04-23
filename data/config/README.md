# Runtime Config Layout

Configuration is split by concern so you only edit the relevant file:

- `app.yaml` - server identity, stats interval, console UI, system logging, and optional dropped-call logs.
- `ingest.yaml` - RBN/PSKReporter/human/DXSummit ingest plus the shared call cache.
- `dedupe.yaml` - primary/secondary dedupe policy windows.
- `floodcontrol.yaml` - shared-ingest flood rails, actions, windows, and per-source thresholds.
- `pipeline.yaml` - call correction, harmonics, spot policy.
- `data.yaml` - CTY/FCC/skew sources, grid DB tuning, and H3 table path.
- `path_reliability.yaml` - path reliability aggregation thresholds, glyph tuning, and allowed band list.
- `runtime.yaml` - telnet server settings, WHOSPOTSME window, buffer capacity, and filter defaults.
- `reputation.yaml` - telnet reputation gate thresholds and IPinfo/Cymru enrichment.
- `peering.yaml` - DXSpider peer configuration (inbound/outbound, ACLs, topology cache).
- `iaru_regions.yaml` - DXCC/ADIF to IARU region mapping used by final regional mode policy.
- `iaru_mode_inference.yaml` - region-aware frequency classification table for final mode labeling.
- `spot_taxonomy.yaml` - canonical supported MODE and EVENT families, parser tokens, PSKReporter routing, and mode capability flags.
- `solarweather.yaml` - solar/geomagnetic override gating for path reliability glyphs.
- `openai.yaml` - optional local LLM settings for propagation-report generation; this file is secret-bearing and ignored by git.

Loader behavior:
- The server defaults to this directory (`data/config`). Override with `DXC_CONFIG_PATH` to point at another directory.
- The loader uses a filename registry. Unknown `.yaml`/`.yml` files fail config load instead of being accidentally merged.
- Merged runtime files populate the main `config.Config` tree. `path_reliability.yaml` and `solarweather.yaml` keep their file-local shapes and are loaded as typed feature-root config.
- `iaru_regions.yaml`, `iaru_mode_inference.yaml`, and `spot_taxonomy.yaml` are required reference tables. Startup fails if they are missing or malformed; there is no built-in table fallback.
- Required YAML-owned settings must be present and non-null in YAML. Missing settings and unknown keys fail config load with a file/key error instead of receiving hidden Go defaults.
- Documented zero values are meaningful. For example, `telnet.broadcast_batch_interval_ms: 0` means immediate delivery, and `*_keepalive_seconds: 0` means the keepalive is disabled.
- `openai.yaml` is optional for server startup and `prop_report -no-llm`. When propagation-report LLM generation is enabled, the file is required and validated at that tool boundary. Secret values must not be logged or committed.
- Use `prop_report -config-dir <dir>` to point report generation at an alternate config directory. The older `-path-config` flag accepts either a directory or `path_reliability.yaml` path for compatibility.

Spot taxonomy:
- `spot_taxonomy.yaml` is the only YAML surface for supported MODE tokens, EVENT families, EVENT reference prefixes, and PSKReporter mode routing.
- `ingest.yaml` owns PSKReporter transport/runtime settings only. Legacy `pskreporter.modes` and `pskreporter.path_only_modes` are rejected; use `pskreporter_route: normal`, `path_only`, or `ignore` on taxonomy modes instead.
- EVENT filtering is family-level. Standalone tokens such as `POTA` and acronym-prefixed references such as `POTA-1234` both tag `POTA`; reference values are not retained or filterable.
- Adding taxonomy modes/events requires editing `spot_taxonomy.yaml` and restarting the cluster with the matching binary/config directory.

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
- `dxsummit.enabled` is owned by the checked-in/operator `ingest.yaml` block. Missing DXSummit settings fail config load rather than receiving loader defaults.
- When enabled, one HTTP polling goroutine reads `dxsummit.base_url` and forwards accepted rows into the shared ingest pipeline as human `UPSTREAM` spots with `SourceNode=DXSUMMIT`.
- `dxsummit.poll_interval_seconds` controls poll cadence. Use the effective YAML value in `ingest.yaml` to know what this node will run.
- `dxsummit.max_records_per_poll` maps directly to the DXSummit `limit` query parameter. The shipped value is `500`; valid range is `1..10000`.
- Normal polls use `from_time=now-lookback_seconds`, `to_time=now`, `limit=max_records_per_poll`, and `include=HF,VHF,UHF`.
- `dxsummit.include_bands` is limited to `HF`, `VHF`, and `UHF`.
- `dxsummit.startup_backfill_seconds: 0` means seed-only startup: the initial page sets the high-water cursor and emits no historical rows.
- `dxsummit.spot_channel_size` and `dxsummit.max_response_bytes` bound retained queue and response memory.
- DXSummit spotter calls ending in `-@` preserve that marker for display/archive provenance, while metadata and license lookups use the base callsign without the marker.
- DXSummit latitude/longitude fields are not used to populate grids. Existing CTY and grid-cache enrichment may fill grids later from callsign-derived data.
- The console dashboard counts DXSummit in `Ingest Sources` only when `dxsummit.enabled` is true. It shows `DXSUMMIT` connected after a recent successful poll, including seed-only startup polls that emit no spots.
