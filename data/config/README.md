# Runtime Config Layout

Configuration is split by concern so you only edit the relevant file:

- `app.yaml` - server identity, stats interval, console UI, and logging options (including dropped-call dedupe window).
- `ingest.yaml` - RBN/PSKReporter/human ingest plus the shared call cache.
- `dedupe.yaml` - primary/secondary dedupe policy windows.
- `floodcontrol.yaml` - shared-ingest flood rails, actions, windows, and per-source thresholds.
- `pipeline.yaml` - call correction, harmonics, spot policy.
- `data.yaml` - CTY/known_calls/FCC/skew sources, grid DB tuning, and H3 table path.
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
