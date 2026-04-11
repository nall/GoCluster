    1: # DX Cluster Server
    2: 
    3: A modern Go-based DX cluster that aggregates amateur radio spots, enriches them with CTY metadata, and broadcasts them to telnet clients.
    4: 
    5: ## Quickstart
    6: 
    7: 1. Install Go `1.25+` (see `go.mod`).
    8: 2. Edit `data/config/` (at minimum: set your callsigns in `ingest.yaml` and `telnet.port` in `runtime.yaml`). If you plan to peer with other DXSpider nodes, populate `peering.yaml` (local callsign, peers, ACL/passwords). You can override the path with `DXC_CONFIG_PATH` if you want to point at a different config directory.
    9: 3. Run:
   10:    ```pwsh
   11:    go mod tidy
   12:    go run .
   13:    ```
   14:    Build an identifiable executable (recommended for deployments):
   15:    ```pwsh
   16:    $version = "v0.1.0"
   17:    $commit = (git rev-parse --short=12 HEAD)
   18:    $built = (Get-Date).ToUniversalTime().ToString("yyyy-MM-ddTHH:mm:ssZ")
   19:    go build -ldflags "-X main.Version=$version -X main.Commit=$commit -X main.BuildTime=$built" -o gocluster.exe .
   20:    .\gocluster.exe --version
   21:    ```
   22:    The PGO pipeline (`scripts/consolidate-and-build-pgo.ps1`, used by `launch-cluster.ps1`) now stamps version metadata automatically on every build.
   23: 4. Connect: `telnet localhost 9300` (or whatever `telnet.port` is set to in `data/config/runtime.yaml`).
   24: 
   25: ## Diagnostics (optional)
   26: 
   27: - `DXC_PPROF_ADDR` enables the `/debug/pprof/*` server (example: `localhost:6061`).
   28: - `DXC_HEAP_LOG_INTERVAL` logs heap stats every interval (e.g., `60s`).
   29: - `DXC_MAP_LOG_INTERVAL` logs key map sizes (call-quality, stats cardinality, dedup caches, path buckets) every interval.
   30: - `DXC_BLOCK_PROFILE_RATE` enables block profiling (Go duration or nanoseconds).
   31: - `DXC_MUTEX_PROFILE_FRACTION` enables mutex profiling (integer 1/N).
   32: 
   33: ## Architecture and Spot Sources
   34: 
   35: 1. **Telnet Server** (`telnet/server.go`) handles client connections, commands, and spot broadcasting using worker goroutines.
   36: 2. **RBN Clients** (`rbn/client.go`) maintain connections to the CW/RTTY (port 7000) and Digital (port 7001) feeds. Each line is parsed and normalized, then sent through the shared ingest CTY/ULS gate for validation and enrichment before queuing.
   37:    - Parsing uses a single left-to-right token walk assisted by an Aho–Corasick (AC) keyword scanner so the line only needs to be scanned once.
   38:    - The parser supports both `DX de CALL: 14074.0 ...` and the glued form `DX de CALL:14074.0 ...` by splitting the spotter token into `DECall` + optional attached frequency.
   39:    - Frequency is the first token that parses as a plausible dial frequency (currently `100.0`-`3,000,000.0` kHz), rather than assuming a fixed column index.
   40:    - Mode is taken from the first explicit mode token (`CW`, `USB`, `JS8`, `SSTV`, `FT8`, `MSK144`, etc.). If mode is absent, it is inferred from `data/config/mode_allocations.yaml` (with a simple fallback: `USB` >= 10 MHz else `CW`).
   41:    - Report/SNR is recognized in both `+5 dB` and `-13dB` forms; `HasReport` is set whenever a report is present (RBN/RBN-Digital zero-SNR reports are dropped before ingest).
   42:    - Ingest burst protection is sized per source via `rbn.slot_buffer` / `rbn_digital.slot_buffer` in `data/config/ingest.yaml`; overflow logs are tagged by source for easier diagnosis.
   43: 3. **PSKReporter MQTT** (`pskreporter/client.go`) subscribes to a single catch-all `pskr/filter/v2/+/+/#` topic and filters modes downstream according to `pskreporter.modes`. It converts JSON payloads into canonical spots and preserves locator-based grids. Set `pskreporter.append_spotter_ssid: true` if you want receiver callsigns that lack SSIDs to pick up a `-#` suffix for deduplication. PSKReporter spots no longer carry a comment string; DX/DE grids stay in metadata and are shown in the fixed tail of telnet output. Configure `pskreporter.max_payload_bytes` to guard against oversized payloads; CTY caching is handled by the unified call metadata cache. PSKReporter spots with explicit `0 dB` reports (rp=0) are dropped before ingest; missing reports are treated as absent. `pskreporter.path_only_modes` routes specific modes (e.g., WSPR) directly into path prediction only—they never reach dedup, telnet, archive, or peer output, and they bypass CTY validation. MQTT ingest is bounded by `pskreporter.mqtt_inbound_workers`, `pskreporter.mqtt_inbound_queue_depth`, and `pskreporter.mqtt_qos12_enqueue_timeout_ms` (QoS0 drops when full; QoS1/2 disconnect after the enqueue timeout).
   44:    - PSK modes are normalized to a canonical `PSK` family for filtering, dedupe, and stats while preserving the reported variant (PSK31/63/125) in telnet/archive output.
   45: 4. **CTY Database** (`cty/parser.go` + `data/cty/cty.plist`) performs longest-prefix lookups; when a callsign includes slashes, it prefers the shortest matching segment (portable/location prefix), so `N2WQ/VE3` and `VE3/N2WQ` both resolve to `VE3` (Canada) for metadata. The in-memory CTY DB is paired with a unified call metadata cache so repeated lookups do not thrash the trie.
   46: 5. **Dedup Engine** (`dedup/deduplicator.go`) filters duplicates before they reach the ring buffer. A zero-second window effectively disables dedup, but the pipeline stays unified. A secondary, broadcast-only dedupe runs after call correction/harmonic/frequency adjustments to collapse repeat DX reports without altering ring/history. It hashes band + DE ADIF (DXCC) + DE grid2 prefix (FAST/MED) or DE CQ zone (SLOW) + normalized DX call + source class (human vs skimmer); the time window is enforced by the cache, so one spot per window per key reaches clients while the ring/history remain intact. Three secondary policies are available: **fast** (120s, grid2), **med** (300s, grid2), and **slow** (480s, CQ zone), each with its own `secondary_*_prefer_stronger_snr` toggle in `data/config/dedupe.yaml`. Telnet clients select with `SET DEDUPE FAST|MED|SLOW` (use `SHOW DEDUPE` to confirm); default is MED. Archive uses the MED policy. Peer publishing uses the MED policy plus source gating: only local non-test human/manual spots are published (RBN/FT8/FT4/PSKReporter, upstream, and peer-origin spots are not peer-published). When call-correction stabilizer is enabled, telnet MED dedupe remains per-client while archive/peer MED dedupe is split to an independent instance so delayed telnet release does not change archive/peer suppression timing. The console pipeline line reports per-policy output as `<count>/<percent> (F) / <count>/<percent> (M) / <count>/<percent> (S)`. When a policy's prefer-stronger flag is true, the stronger SNR duplicate replaces the cached entry and is broadcast for that policy. Spotter SSID display is controlled at broadcast time (see `rbn.keep_ssid_suffix`); when disabled, telnet output, archive, and filters use stripped DE calls while peers keep the raw calls.
   47: 6. **Frequency Averager** (`spot/frequency_averager.go`) merges CW/RTTY skimmer reports by averaging corroborating reports within a tolerance and rounding to 0.01 kHz (10 Hz) once the minimum corroborators is met.
   48: 7. **Call/Harmonic/License Guards** (`spot/correction.go`, `spot/harmonics.go`, `main.go`) apply resolver-primary call correction, suppress harmonics, and finally run FCC license gating for DX right before broadcast/buffering (CTY validation runs in the ingest gate; corrected calls are re-validated against CTY before acceptance). Resolver winner admission uses shared family rails, optional neighborhood competition (`resolver_neighborhood_*`), and the conservative one-short recent corroborator rail (`resolver_recent_plus1_*`). Family behavior is configured under `call_correction.family_policy` (slash precedence, truncation rails, and telnet family suppression including optional contested edit-neighbor suppression). A conservative split-signal ambiguity guard rejects corrections when top candidates have strong support but highly disjoint spotter sets in the same narrow frequency neighborhood (`reason=ambiguous_multi_signal`). Calls ending in `/B` (standard beacon IDs) are auto-tagged and bypass correction/harmonic/license drops (only user filters can hide them). The license gate uses a license-normalized base call (e.g., `W6/UT5UF` -> `UT5UF`) to decide if FCC checks apply and which call to query, while CTY metadata still reflects the portable/location prefix (so `N2WQ/VE3` reports Canada for DXCC but uses `N2WQ` for licensing); drops appear in the "Unlicensed US Calls" pane.
   49: 8. **Skimmer Frequency Corrections** (`cmd/rbnskewfetch`, `skew/`, `rbn/client.go`, `pskreporter/client.go`) download SM7IUN's skew list, convert it to JSON, and apply per-spotter multiplicative factors before any callsign normalization for every CW/RTTY skimmer feed.
   50: 
   51: ### Aho–Corasick Spot Parsing (Non-PSKReporter)
   52: 
   53: Non-PSKReporter sources (RBN CW/RTTY, RBN digital, and upstream/human telnet feeds) arrive as DX-cluster style text lines (e.g., `DX de ...`). The parser in `rbn/client.go` uses a small Aho–Corasick (AC) automaton to recognize keywords in a single pass and drive a left-to-right extraction.
   54: 
   55: High-level flow:
   56: 
   57: - **Keyword dictionary**: `DX`, `DE`, `DB`, `WPM`, plus all supported mode tokens (`CW`, `SSB` as an alias normalized to `USB`/`LSB`, `USB`, `LSB`, `JS8`, `SSTV`, `RTTY`, `FT4`, `FT8`, `MSK144`, and common variants like `FT-8`).
   58: - **Automaton build (once)**: patterns are compiled into a trie and failure links are built with a BFS. This runs once and is reused for every line.
   59: - **Per-line scan**:
   60:   - Tokenize the raw line on whitespace while tracking token byte offsets.
   61:   - Run the AC scan over the uppercased line to find keyword hits.
   62:   - Classify each token by checking for an exact hit that spans the token (fallback: scan the token text itself when whitespace/punctuation causes slight drift).
   63: - **Single pass extraction**: walk tokens left-to-right, consuming fields as they are discovered: spotter call (and optional `CALL:freq` attachment), frequency, DX call, mode, report (`<signed int> dB` or `<signed int>dB`), time (`HHMMZ`), then treat any remaining unconsumed tokens as the free-form comment.
   64: - **Mode inference**: when no explicit mode token exists, infer from `data/config/mode_allocations.yaml` by frequency (fallback: `USB` ≥ 10 MHz else `CW`).
   65: - **Report semantics**: `HasReport` is strictly “report was present in the source line” (or PSKReporter rp field), so `0 dB` is distinct from “missing report” on sources that retain it (RBN/PSKReporter zero-SNR drops happen before ingest).
   66: 
   67: ### Call-Correction Distance Tuning
   68: - CW distance can be Morse-aware with weighted/normalized dot-dash costs (configurable via `call_correction.morse_weights`: `insert`, `delete`, `sub`, `scale`; defaults 1/1/2/2).
   69: - RTTY distance can be ITA2-aware with similar weights (configurable via `call_correction.baudot_weights`: `insert`, `delete`, `sub`, `scale`; defaults 1/1/2/2).
   70: - If you prefer plain rune-based Levenshtein, set `call_correction.distance_model_cw: plain` and/or `distance_model_rtty: plain`.
   71: - You can optionally enable confusion-model ranking via `call_correction.confusion_model_enabled`, `call_correction.confusion_model_file`, and `call_correction.confusion_model_weight`. In resolver-primary flow this resolves winner ties when top candidates are tied on weighted support and support count. Hard correction gates are unchanged.
   72: - Confusion-model enablement example (`data/config/pipeline.yaml`):
   73:   ```yaml
   74:   call_correction:
   75:     confusion_model_enabled: true
   76:     confusion_model_file: "data/rbn/priors/confusion_model.json"
   77:     confusion_model_weight: 1
   78:   ```
   79: - `call_correction.resolver_neighborhood_*` enables bounded cross-bucket winner competition in resolver-primary mode so near-boundary signals do not fork into adjacent buckets, with anchor-scoped comparability rails (vote-key/slash/truncation family and `resolver_neighborhood_max_distance`).
   80: - `call_correction.resolver_recent_plus1_*` enables a conservative resolver-primary corroborator rail: only one-short-on-min-reports winners are eligible, with rails for distance/family, winner recent support minimum, and optional subject-weaker requirement.
   81: - `call_correction.bayes_bonus.*` adds an optional default-off Bayesian-style gate bonus for distance-1/2 near-threshold winners. Caps remain conservative (`+1` report-gate bonus and `+1` advantage tie-break only), and validation/recent-evidence rails must still pass.
   82: - Call correction can reject split-evidence clusters with `reason=ambiguous_multi_signal` when two strong candidates in the same narrow frequency neighborhood have highly disjoint spotter sets.
   83: - `call_correction.stabilizer_*` adds an optional telnet-only delay gate for delay-eligible glyphs (`?`, `S`, `P`) and targeted holds for resolver-ambiguous (`split`/`uncertain`), low-confidence `P`, or contested edit-neighbor cases (`stabilizer_edit_neighbor_*`). `V` and `C` always pass through stabilizer delay. `stabilizer_max_checks` controls baseline delay-check cycles before timeout action; it includes the first delayed check (`1` preserves legacy single-check behavior). On timeout you can `release` or `suppress`; queue overflow is fail-open (immediate release). Delayed spots are admitted to recent-on-band support only after delay resolution; suppressed timeouts do not reinforce recent support.
   84: - `call_correction.temporal_decoder.*` enables a bounded fixed-lag sequence decoder shared by runtime and replay. It can hold uncertain candidates briefly (`scope=uncertain_only` or `all_correction_candidates`), score short call sequences with deterministic beam/Viterbi logic, and then either apply, fallback to resolver, abstain, or bypass based on `min_score`, `min_margin_score`, and `overflow_action`.
   85: - You can down-weight noisy reporters via `call_correction.spotter_reliability_file` (global fallback), plus optional mode-specific overrides `call_correction.spotter_reliability_file_cw` and `call_correction.spotter_reliability_file_rtty` (format: `SPOTTER WEIGHT 0-1`). `call_correction.min_spotter_reliability` defines the reporter floor used by resolver voting.
   86: - Slash precedence uses `call_correction.family_policy.slash_precedence_min_reports` (default `2`): when a slash-explicit variant reaches that threshold inside the same base bucket, the bare form is excluded from that bucket's voting and anchor path.
   87: - Truncation families are controlled by `call_correction.family_policy.truncation.*` (`max_length_delta` controls whether one-char only or wider deltas are considered). Advantage relaxation rails are configured by `call_correction.family_policy.truncation.relax_advantage.*`; optional truncation-only min-reports bonus uses `call_correction.family_policy.truncation.length_bonus.*`; optional stricter delta-2 gates use `call_correction.family_policy.truncation.delta2_rails.*`.
   88: 
   89: ## UI Modes (local console)
   90: 
   91: - `ui.mode: ansi` (default) draws the fixed 90x68 ANSI console in the server's terminal when stdout is a TTY. The layout is 12 stats lines, a blank line, then Dropped/Corrected/Unlicensed/Harmonics/System Log panes with 10 lines each. Pane headers render as `<<<<<<<<<< Dropped >>>>>>>>>>`, `<<<<<<<<<< Corrected >>>>>>>>>>`, etc. Telnet clients do **not** see this UI; if the terminal is smaller than 90x68, ANSI disables and logs continue.
   92: - `ui.mode: tview` enables the legacy framed tview dashboard (requires an interactive console).
   93: - `ui.mode: tview-v2` enables the page-based tview dashboard with bounded buffers and navigation keys.
   94:   - The v2 stream panes use virtualized viewport rendering with bounded rings to reduce CPU/heap pressure at sustained update rates.
   95: - `ui.mode: headless` disables the local console; logs continue to stdout/stderr.
   96: - `ui.pane_lines` controls the visible heights of tview panes; ANSI uses a fixed 12/10-line layout and ignores pane_lines.
   97: - `logging.enabled` in `app.yaml` duplicates system logs to daily files in `logging.dir` (local time, `logging.retention_days` controls retention).
   98: - Config block (excerpt):
   99:   ```yaml
  100:   ui:
  101:     mode: "ansi"       # ansi | tview | tview-v2 | headless
  102:     refresh_ms: 250    # ANSI minimum redraw spacing; 0 renders on every event
  103:     color: true        # ANSI coloring for marked-up lines
  104:     clear_screen: true # Ignored by ANSI; preserved for compatibility
  105:     pane_lines:
  106:       stats: 8
  107:       calls: 20
  108:       unlicensed: 20
  109:       harmonics: 20
  110:       system: 40
  111:   ```
  112: 
  113: ## Propagation Reports (Daily)
  114: 
  115: The cluster can generate a daily propagation report from the prior day's log file. It triggers on log rotation and also on a fixed UTC schedule so quiet systems still produce reports.
  116: 
  117: Config block (excerpt):
  118: ```yaml
  119: prop_report:
  120:   enabled: true
  121:   refresh_utc: "00:05" # UTC time to enqueue yesterday's report
  122: ```
  123: 
  124: ## Data Flow and Spot Record Format
  125: 
  126: ```
  127: [Source: RBN/PSKReporter] → Parser → Ingest CTY/ULS Gate → Dedup (window-driven) → Ring Buffer → Telnet Broadcast
  128: ```
  129: 
  130: ```
  131: ┏━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┓
  132: ┃                         DXCluster Spot Ingestion & Delivery                         ┃
  133: ┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┛
  134: 	┌────────────────────┐    ┌────────────────────┐    ┌─────────────────────────┐
  135: 	│ RBN CW/RTTY client │    │ RBN FT4/FT8 client │    │ PSKReporter MQTT client │
  136: 	└──────────┬─────────┘    └──────────┬─────────┘    └──────────┬──────────────┘
  137: 		   │                         │                         │
  138: 		   ▼                         ▼                         ▼
  139: 	┌────────────────────┐    ┌────────────────────┐    ┌─────────────────────────┐
  140: 	│ RBN line parsers   │    │ RBN digital parsers│    │ PSKReporter worker pool │
  141: 	└──────────┬─────────┘    └──────────┬─────────┘    └──────────┬──────────────┘
  142: 		   │                         │                         │
  143: 		   ├─────────────────────────┴─────────────────────────┤
  144: 		   ▼                                                   ▼
  145: 	 ┌────────────────────────────────────────────────────────────────────┐
  146: 	 │ Normalize callsigns → ingest CTY/ULS checks → enrich metadata      │
  147: 	 │ (shared logic in `spot` + `cty` packages)                           │
  148: 	 └──────────────────────────────┬──────────────────────────────────────┘
  149: 					│
  150: 					▼
  151: 			  ┌──────────────────────────────┐
  152: 			  │ Dedup engine (cluster/user)  │
  153: 			  └───────────────┬──────────────┘
  154: 					  │
  155: 					  ▼
  156: 			  ┌──────────────────────────────┐
  157: 			  │ Ring buffer (`buffer/`)      │
  158: 			  └───────────────┬──────────────┘
  159: 					  │
  160: 					  ▼
  161: 			  ┌──────────────────────────────┐
  162: 			  │ Telnet server (`telnet/`)    │
  163: 			  └───────────────┬──────────────┘
  164: 					  │
  165: 					  ▼
  166: 			  Connected telnet clients + filters
  167: ```
  168: 
  169: ### Telnet Spot Line Format
  170: 
  171: The telnet server broadcasts spots as fixed-width DX-cluster lines:
  172: 
  173: - Exactly **78 characters**, followed by **CRLF** (line endings are normalized in `telnet.Client.Send`).
  174: - Column numbering below is **1-based** (column 1 is the `D` in `DX de `).
  175: - The left side is padded so **mode always starts at column 40**.
  176: - Frequency is rendered in kHz with exactly **two decimal places**.
  177: - The displayed DX callsign uses the canonical normalized call (portable suffixes stripped) and is truncated to 10 characters to preserve fixed columns (the full normalized callsign is still stored and hashed). During correction, slash-explicit winners are preserved when slash precedence applies, so output can intentionally show variants like `W1AW/1`.
  178: - The right-side tail is fixed so clients can rely on it:
  179:   - Grid: columns 67-70 (4 chars; blank if unknown)
  180:   - Confidence: column 72 (1 char; blank if unknown)
  181:   - Time: columns 74-78 (`HHMMZ`)
  182: - Any free-form comment text is sanitized (tabs/newlines removed) and truncated so it can never push the grid/confidence/time tail; a space always separates the comment area from the fixed tail.
  183: 
  184: Report formatting:
  185: 
  186: - If a report is present: `MODE <report> dB` (e.g., `FT8 -12 dB`, `CW 27 dB`, `MSK144 +7 dB`).
  187: - Human spots without SNR omit the report entirely and show only `MODE`.
  188: 
  189: Example:
  190: 
  191: ```
  192: DX de W3LPL:      7009.50  K1ABC       FT8 -5 dB                          FN20 S 0615Z
  193: ```
  194: 
  195: Each `spot.Spot` stores:
  196: - **ID** - monotonic identifier
  197: - **DXCall / DECall** - normalized callsigns (portable suffixes stripped, location prefixes like `/VE3` retained; validation accepts 3-15 characters; telnet display truncates DX to 10 as described above)
  198: - **Frequency** (kHz), **Band**, **Mode**, **Report** (dB/SNR), **HasReport** (distinguishes missing SNR from a real 0 on sources that retain it; RBN/PSKReporter zero-SNR are dropped at ingest)
  199: - **Time** - UTC timestamp from the source
  200: - **Comment** - free-form message (human ingest strips mode/SNR/time tokens before storing)
  201: - **SourceType / SourceNode** - origin tags (`RBN`, `FT8`, `FT4`, `PSKREPORTER`, `UPSTREAM`, etc.)
  202: - **TTL** - hop count preventing loops
  203: - **IsHuman** - whether the spot was reported by a human operator (RBN/PSKReporter spots are skimmers; peer/upstream/manual spots are human)
  204: - **IsTestSpotter** - true for CTY-valid telnet test calls (suffix `TEST` or `TEST-SSID`); such spots are broadcast locally but never peered, archived, or stored in the ring buffer
  205: - **IsBeacon** - true when the DX call ends with `/B` or the comment mentions `NCDXF`/`BEACON` (used to suppress beacon corrections/filtering)
  206: - **DXMetadata / DEMetadata** - structured `CallMetadata` each containing:
  207: 	- `Continent`
  208: 	- `Country`
  209: 	- `CQZone`
  210: 	- `ITUZone`
  211: 	- `ADIF` (DXCC/ADIF country code)
  212: 	- `Grid`
  213: 
  214: All ingest sources run through a shared CTY/ULS validation gate before deduplication. Callsigns are normalized once (uppercased, dots converted to `/`, trailing slashes removed, and portable suffixes like `/P`, `/M`, `/MM`, `/AM`, `/QRP` stripped) before CTY lookup, so W1AW and W1AW/P collapse to the same canonical call for hashing and filters. Location prefixes (for example `/VE3`) are retained; CTY lookup then chooses the shortest matching slash segment so `N2WQ/VE3` and `VE3/N2WQ` both resolve to `VE3` for metadata. Validation still requires at least one digit to avoid non-amateur identifiers before malformed or unknown calls are filtered out. Automated feeds mark the `IsHuman` flag as `false` so downstream processors can tell which spots originated from telescopic inputs versus human operator submissions. Call correction re-validates suggested DX calls against CTY before accepting them; FCC license gating runs after correction using a license-normalized base call (for example, `W6/UT5UF` is evaluated as `UT5UF`) and drops unlicensed US calls (beacons bypass this drop; user filters still apply).
  215: 
  216: ## Commands
  217: 
  218: Telnet clients can issue commands via the prompt once logged in. The processor, located in `commands/processor.go`, requires a logged-in callsign and ignores unauthenticated commands with `No logged user found. Command ignored.` It supports the following general commands:
  219: 
  220: - Test spotter calls: when a logged-in callsign ends with `TEST` (optionally `-<SSID>`) and has no slash segments, the base call (SSID stripped) must resolve in CTY or the `DX` command is rejected with a message. Accepted test spots are still filtered/broadcast locally and subject to reputation gating; they bypass FCC ULS validation, but are not stored in the ring buffer, not archived, and never peered.
  221: 
  222: - `HELP [command]` / `H` - list commands for the active dialect or show detailed help for a specific command (for example, `HELP DX`).
  223: - `SHOW DX [N]` / `SHOW/DX [N]` - alias of `SHOW MYDX`, streaming the most recent `N` filtered spots (`N` ranges from 1-250, default 50). Optional DXCC selector forms are supported: `SHOW DX <prefix|callsign> [N]` and `SHOW DX [N] <prefix|callsign>` (same for `SHOW/DX`). The selector is resolved through CTY portable lookup, and only spots whose `DXMetadata.ADIF` matches that DXCC are shown. Archive-only: if the Pebble archive is unavailable, the command returns `No spots available.` The command accepts the alias `SH DX` (or `SH/DX` in cc) as well.
  224: - `SHOW MYDX [N]` - stream the most recent `N` spots that match your filters (self-spots always pass; `N` ranges from 1-250, default 50). Optional DXCC selector forms are supported: `SHOW MYDX <prefix|callsign> [N]` and `SHOW MYDX [N] <prefix|callsign>`. The selector resolves via CTY and filters to matching DX ADIF/DXCC. If a selector is provided while CTY is unavailable, the command returns `CTY database is not available.` / `CTY database is not loaded.` Archive-only: if the Pebble archive is unavailable, the command returns `No spots available.` Very narrow filters may return fewer than `N` results.
  225: - `SET DIAG <ON|OFF>` - replace the comment field with a diagnostic tag: `<source><DEDXCC><DEGRID2><band><policy>`, where source is `R` (RBN), `P` (PSK), or `H` (human/peer) and policy is `F`/`S`.
  226: - `SET SOLAR <15|30|60|OFF>` - opt into wall-clock aligned solar summaries (OFF by default).
  227: - `BYE`, `QUIT`, `EXIT` - request a graceful logout; the server replies with `73!` and closes the connection.
  228: 
  229: Filter management commands use a table-driven engine in `telnet/server.go` with explicit dialect selection. The default `go` dialect uses `PASS`/`REJECT`/`SHOW FILTER`. A CC-style subset is available via `DIALECT cc` (aliases: `SET/ANN`, `SET/NOANN`, `SET/BEACON`, `SET/NOBEACON`, `SET/WWV`, `SET/NOWWV`, `SET/WCY`, `SET/NOWCY`, `SET/SELF`, `SET/NOSELF`, `SET/SKIMMER`, `SET/NOSKIMMER`, `SET/<MODE>`, `SET/NO<MODE>`, `SET/FILTER DXBM/PASS|REJECT <band>` mapping CC DXBM bands to our band filters, `SET/NOFILTER`, plus `SET/FILTER`/`UNSET/FILTER`/`SHOW/FILTER`). `DIALECT LIST` shows the available dialects, the chosen dialect is persisted per callsign along with filter state, and HELP renders the verbs for the active dialect. Classic/go commands operate on each client's `filter.Filter` and fall into `PASS`, `REJECT`, and `SHOW FILTER` groups:
  230: 
  231: - Operator note: `RESET FILTER` re-applies the configured defaults from `data/config/runtime.yaml` (`filter.default_modes` and `filter.default_sources`). `SET/NOFILTER` is CC-only and resets to a fully permissive "all pass" state; it does not use configured defaults.
  232: 
  233: - `SHOW FILTER` - prints a full snapshot of filter state (allow/block + effective) for bands, modes, sources, continents, zones, DXCC, grids, confidence, path classes, callsigns, and toggles.
  234: 
  235: Tokenized `SHOW FILTER <type>` / `SHOW/FILTER <type>` forms are deprecated; they return the full snapshot with a warning.
  236: Effective labels in the snapshot use a fixed vocabulary: `all pass`, `all except: <items>`, `only: <items>`, `none pass`, `all blocked`.
  237: - `PASS BAND <band>[,<band>...]` - enables filtering for the comma- or space-separated list (each item normalized via `spot.NormalizeBand`), or specify `ALL` to accept every band; use the band names from `spot.SupportedBandNames()`.
  238: - `PASS MODE <mode>[,<mode>...]` - enables one or more modes (comma- or space-separated) that must exist in `filter.SupportedModes`, or specify `ALL` to accept every mode.
  239: - `PASS SOURCE <HUMAN|SKIMMER|ALL>` - filter by spot origin: `HUMAN` passes only spots with `IsHuman=true`, `SKIMMER` passes only spots with `IsHuman=false`, and `ALL` disables source filtering.
  240: - `PASS DXCONT <cont>[,<cont>...]` / `DECONT <cont>[,<cont>...]` - enable only the listed DX/spotter continents (AF, AN, AS, EU, NA, OC, SA), or `ALL`.
  241: - `PASS DXZONE <zone>[,<zone>...]` / `DEZONE <zone>[,<zone>...]` - enable only the listed DX/spotter CQ zones (1-40), or `ALL`.
  242: - `PASS DXGRID2 <grid>[,<grid>...]` - enable only the listed 2-character DX grid prefixes. Tokens longer than two characters are truncated (e.g., `FN05` -> `FN`); `ALL` resets to accept every DX 2-character grid.
  243: - `PASS DEGRID2 <grid>[,<grid>...]` - enable only the listed 2-character DE grid prefixes (same parsing/truncation as DXGRID2); `ALL` resets to accept every DE 2-character grid.
  244: - `PASS DXCALL <pattern>[,<pattern>...]` - begins delivering only spots with DX calls matching the supplied patterns.
  245: - `PASS DECALL <pattern>[,<pattern>...]` - begins delivering only spots with DE/spotter calls matching the supplied patterns.
  246: - `PASS CONFIDENCE <symbol>[,<symbol>...]` - enables the comma- or space-separated list of consensus glyphs (valid symbols: `?`, `S`, `C`, `P`, `V`, `B`; use `ALL` to accept every glyph).
  247: - `PASS PATH <class>[,<class>...]` - enables the comma- or space-separated list of path prediction classes (HIGH/MEDIUM/LOW/UNLIKELY/INSUFFICIENT; use `ALL` to accept every class). When the path predictor is disabled, PATH commands are ignored with a warning.
  248: - `PASS NEARBY ON|OFF` - when ON, deliver spots whose DX or DE H3 cell matches your grid (L1 for 160/80/60m, L2 otherwise). Location filters (DX/DE CONT, ZONE, GRID2, DXCC) are suspended while NEARBY is ON, and attempts to change them are rejected with a warning. OFF restores the prior location filter state. State persists across sessions and a login warning is shown when active. Requires `SET GRID`.
  249: - `PASS BEACON` - explicitly enable delivery of beacon spots (DX calls ending `/B`; enabled by default).
  250: - `PASS SELF` - always deliver spots where the DX callsign matches your normalized callsign (even if filters would normally block).
  251: - `REJECT BAND <band>[,<band>...]` - disables only the comma- or space-separated list of bands provided (use `ALL` to block every band).
  252: - `REJECT MODE <mode>[,<mode>...]` - disables only the comma- or space-separated list of modes provided (specify `ALL` to block every mode).
  253: - `REJECT SOURCE <HUMAN|SKIMMER|ALL>` - blocks one origin category (human/operator spots vs automated/skimmer spots), or `ALL`.
  254: - `REJECT DXCONT` / `DECONT` / `DXZONE` / `DEZONE` - block continent/zone filters (use `ALL` to block all).
  255: - `REJECT DXGRID2 <grid>[,<grid>...]` - remove specific 2-character DX grid prefixes (tokens truncated to two characters); `ALL` blocks every DX 2-character grid.
  256: - `REJECT DEGRID2 <grid>[,<grid>...]` - remove specific 2-character DE grid prefixes (tokens truncated to two characters); `ALL` blocks every DE 2-character grid.
  257: - `REJECT DXCALL <pattern>[,<pattern>...]` - blocks the supplied DX callsign patterns.
  258: - `REJECT DECALL <pattern>[,<pattern>...]` - blocks the supplied DE callsign patterns.
  259: - `REJECT CONFIDENCE <symbol>[,<symbol>...]` - disables only the comma- or space-separated list of glyphs provided (use `ALL` to block every glyph).
  260: - `REJECT PATH <class>[,<class>...]` - disables the comma- or space-separated list of path prediction classes (use `ALL` to block every class).
  261: - `REJECT BEACON` - drop beacon spots entirely (they remain tagged internally for future processing).
  262: - `REJECT SELF` - suppress all spots where the DX callsign matches your normalized callsign.
  263: 
  264: Confidence glyphs are only emitted for modes that run call-correction logic (CW/RTTY/USB/LSB voice modes). FT8/FT4 spots carry no confidence glyphs, so confidence filters do not affect them. After correction assigns `P`/`V`/`C`/`?`, any remaining `?` is upgraded to `S` when the DX call is admitted by static custom-SCP membership (when `call_correction.custom_scp.enabled=true`) or admitted by recent evidence rails.
  265: 
  266: Band, mode, confidence, PATH, and DXGRID2/DEGRID2 commands share identical semantics: they accept comma- or space-separated lists, ignore duplicates/case, and treat the literal `ALL` as a shorthand to allow or block everything for that type. PASS/REJECT add to allow/block lists and remove the same items from the opposite list. DXGRID2 applies only to the DX grid when it is exactly two characters long; DEGRID2 applies only to the DE grid when it is exactly two characters long. 4/6-character or empty grids are unaffected, and longer tokens provided by the user are truncated to their first two characters before validation.
  267: SELF matches the normalized DX callsign only; when a spot is suppressed by secondary dedupe, a matching client still receives it if SELF is enabled. This delivery is per-client and does not bypass secondary dedupe for the global broadcast stream.
  268: 
  269: Confidence indicator legend in telnet output:
  270: 
  271: - `?` - Unknown/low support
  272: - `S` - post-correction confidence would otherwise be `?` and DX call is admitted by custom-SCP static membership or recent-evidence admission
  273: - `P` - 25-50% consensus for the subject call (no correction applied)
  274: - `V` - More than 50% consensus for the subject call (no correction applied)
  275: - `B` - Correction was suggested but CTY validation failed (call left unchanged)
  276: - `C` - Callsign was corrected and CTY-validated
  277: 
  278: ### Telnet Reputation Gate
  279: 
  280: The passwordless reputation gate throttles telnet `DX` commands based on call history, ASN/geo consistency, and prefix pressure. It is designed to slow down new or suspicious senders while keeping known-good calls flowing. Drops are silent to clients, but surfaced in the console stats and system pane.
  281: 
  282: Core behavior:
  283: - New calls wait an initial probation window before any spots are accepted.
  284: - Per-band limits ramp by one each window up to a cap; total cap increases after ramp completion.
  285: - Country mismatch (IP vs CTY) or Cymru/IPinfo disagreement adds an extra delay before ramping.
  286: - New ASN or geo flips reset the call to probation.
  287: - Prefix token buckets (/24, /48) shed load before per-call limits.
  288: 
  289: Data sources:
  290: - IPinfo Lite CSV is imported into a local Pebble DB; IPv4 ranges are loaded into RAM for microsecond lookups, IPv6 stays on disk and is served via Pebble + cache.
  291: - Team Cymru DNS TXT is a fallback when Pebble misses or is unavailable; answers are cached for 24h with a tight timeout.
  292: - IPinfo live API is the last resort when both the local store and Cymru miss.
  293:   - The downloader uses `curl` with the configured token, imports into Pebble, and cleans up the CSV if configured.
  294:   - Optional full compaction after import reduces read amplification; see `ipinfo_pebble_compact` in the reputation config.
  295: 
  296: Configuration:
  297: - See the `reputation` section in `data/config/reputation.yaml` for lookup order, TTLs, download/API tokens, and thresholds; extensive comments document each knob for operators.
  298: 
  299: Use `PASS CONFIDENCE` with the glyphs above to whitelist the consensus levels you want to see (for example, `PASS CONFIDENCE P,V` keeps strong/very strong reports while dropping `?`/`S`/`B` entries).
  300: 
  301: Use `REJECT BEACON` to suppress DX beacons when you only want live operator traffic; `PASS BEACON` re-enables them, and `SHOW FILTER` reports the current state. Regardless of delivery, `/B` spots are excluded from call-correction, frequency-averaging, and harmonic checks.
  302: Errors during filter commands return a usage message (e.g., invalid bands or modes refer to the supported lists) and the `SHOW FILTER` command helps confirm the active settings.
  303: 
  304: Continent and CQ-zone filters behave like the band/mode whitelists: start permissive, tighten with `PASS`, reset with `ALL`. When a continent/zone filter is active, spots missing that metadata are rejected so the whitelist cannot be bypassed by incomplete records.
  305: 
  306: New-user filter defaults are configured in `data/config/runtime.yaml` under `filter:` and are only applied when a callsign has no saved filter file in `data/users/`:
  307: 
  308: - `filter.default_modes`: initial mode selection for `PASS/REJECT MODE`.
  309: - `filter.default_sources`: initial SOURCE selection (`HUMAN` for `IsHuman=true`, `SKIMMER` for `IsHuman=false`). Omit the field or list both categories to disable SOURCE filtering (equivalent to `PASS SOURCE ALL`).
  310: 
  311: Existing users keep whatever is stored in their `data/users/<CALL>.yaml` file; changing these defaults only affects newly created users.
  312: 
  313: ## RBN Skew Corrections
  314: 
  315: 1. Enable the `skew` block in `data/config/data.yaml` (the server writes to `skew.file` after each refresh):
  316: 
  317: ```yaml
  318: skew:
  319:   enabled: true
  320:   url: "https://sm7iun.se/rbnskew.csv"
  321:   file: "data/skm_correction/rbnskew.json"
  322:   min_abs_skew: 1
  323: ```
  324: 
  325: 2. (Optional) Run `go run ./cmd/rbnskewfetch -out data/skm_correction/rbnskew.json` once to pre-seed the JSON file before enabling the feature.
  326: 3. Restart the cluster. At startup, it loads the JSON file (if present) and then fetches the CSV at the next `skew.refresh_utc` boundary (default `00:30` UTC). The built-in scheduler automatically refreshes the list every day at that UTC time and rewrites `skew.file`, so no external cron job is required.
  327: 
  328: Each RBN spot uses the *raw* spotter string (SSID intact, before any normalization) to look up the correction. If found, the original frequency is multiplied by the factor before any dedup, CTY validation, call correction, or harmonic detection runs. This keeps SSID-specific skew data aligned with the broadcast nodes.
  329: 
  330: To match 10 Hz resolution end-to-end, corrected frequencies are rounded to the nearest 0.01 kHz (half-up) before continuing through the pipeline.
  331: 
  332: ## CTY Database Refresh
  333: 
  334: 1. Configure the `cty` block in `data/config/data.yaml`:
  352: 
  353: ```yaml
  354: cty:
  355:   enabled: true
  356:   url: "https://www.country-files.com/cty/cty.plist"
  357:   file: "data/cty/cty.plist"
  358:   refresh_utc: "00:45"
  359: ```
  360: 
  361: 2. On startup the server downloads `cty.plist` if it is missing and a URL is configured, then loads it into the in-memory CTY database. A `.status.json` sidecar tracks ETag/Last-Modified so subsequent refreshes can skip unchanged files.
  362: 3. When `cty.enabled` is true, the scheduler checks the plist daily at `cty.refresh_utc` using conditional requests and reloads the CTY DB only when content changes; the unified call metadata cache is cleared on each CTY reload. Failures log a warning and retry with backoff; the last-good CTY DB remains active.
  363: 4. The stats pane includes a `CTY: age ...` line that shows how long it has been since the last successful refresh (and a failure count when retries are failing), so staleness is visible at a glance.
  364: 
  365: ## FCC ULS Downloads
  366: 
  367: 1. Configure the `fcc_uls` block in `data/config/data.yaml`:
  368: 
  369: ```yaml
  370: fcc_uls:
  371:   enabled: true
  372:   url: "https://data.fcc.gov/download/pub/uls/complete/l_amat.zip"
  373:   archive_path: "data/fcc/l_amat.zip"
  374:   db_path: "data/fcc/fcc_uls.db"
  375:   refresh_utc: "02:15"
  376: ```
  377: 
  378: 2. On startup the cluster launches a background job that checks for the SQLite DB. If the DB is missing, it immediately downloads the archive (ignoring cache headers), extracts the AM/EN/HD tables, and builds a fresh SQLite database at `fcc_uls.db_path`. If the DB is present, it waits for the scheduled refresh time. Both the ZIP and DB are written via temp files and swapped atomically; metadata/status is stored at `archive_path + ".status.json"` (the previous `.meta.json` is still read for compatibility).
  379: 3. During the load, only active licenses are kept (`HD.license_status = 'A'`). HD is slimmed to a few useful fields (unique ID, call sign, status, service, grant/expire/cancel/last-action dates), and AM is reduced to just unique ID + call sign for active records. EN is not loaded. The downloaded ZIP is deleted after a successful build to save space.
  380: 4. When `fcc_uls.enabled` is true, a built-in scheduler refreshes the archive and rebuilds the database once per day at `fcc_uls.refresh_utc` (UTC). The refresh uses conditional requests when metadata is present, even if the archive was deleted after the prior build. The job runs independently of spot processing, so the rest of the cluster continues handling spots while the download, unzip, and load proceed.
  381: 5. The console/TUI stats include an FCC line showing active-record counts and the DB size.
  382: 
  383: ## Grid Persistence and Caching
  384: 
  385: - Grids and CTY metadata (ADIF/CQ/ITU/continent/country) are stored in Pebble at `grid_db` (default `data/grids/pebble`, a directory). Each batch is committed with `Sync` for durability; the in-memory cache continues serving while backfills rebuild on new spots.
  386: - Gridstore checkpoints are created hourly under `grid_db/checkpoint` and retained for 24 hours. On startup, corruption triggers an automatic restore from the newest verified checkpoint; the cluster continues running while the restore rebuilds the Pebble directory. A daily integrity scan runs at 05:00 UTC and logs the result.
  387: - When a call lacks a stored grid, the CTY prefix latitude/longitude is used to derive a 4-character Maidenhead grid and persist it as "derived"; derived grids never overwrite non-derived entries. Telnet output renders derived grids in lowercase while internal storage and computations remain uppercase.
  388: - Writes are batched by `grid_flush_seconds` (default `60s`); a final flush runs during shutdown.
  389: - The unified call metadata cache is a bounded LRU of size `grid_cache_size` (default `100000`). It caches grid/CTY lookups and only applies the TTL (`grid_cache_ttl_seconds`) to grid entries; CTY refreshes clear the cache. Cache misses fall back to Pebble via the async backfill path when `grid_db_check_on_miss` is true; RBN grid misses also attempt a tight-timeout sync lookup to seed the cache before secondary dedupe/path reliability.
  390: - Pebble tuning knobs (defaults tuned for read-heavy durability): `grid_block_cache_mb=64`, `grid_bloom_filter_bits=10`, `grid_memtable_size_mb=32`, `grid_l0_compaction_threshold=4`, `grid_l0_stop_writes_threshold=16`, `grid_write_queue_depth=64`.
  391: - The stats line `Grids: <TOTAL|UPDATED> / <hit%> / <lookups/min> | Drop aX sY` reports gridstore totals (or updates since start if the DB is unavailable), cache hit rate, lookup rate per minute, and async/sync lookup queue drops.
  392: - If you set `grid_ttl_days > 0`, the store purges rows whose `updated_at` timestamp is older than that many days during the configured maintenance pass. Live grid updates keep records fresh automatically.
  393: - `grid_preflight_timeout_ms` is ignored for the Pebble prototype (retained for config compatibility).
  394: 
  395: ## Runtime Logs and Corrections
  396: 
  397: - **Call corrections**: `2025/11/19 18:50:45 Call corrected: VE3N -> VE3NE at 7011.1 kHz (8 / 88%)`
  398: - **Frequency averaging**: `2025/11/19 18:50:45 Frequency corrected: VE3NE 7011.3 -> 7011.1 kHz (8 / 88%)`
  399: - **Harmonic suppression**: `2025/11/19 18:50:45 Harmonic suppressed: VE3NE 14022.0 -> 7011.0 kHz (3 / 18 dB)` plus a paired frequency-corrected line indicating the fundamental retained.
  400: - **Stats ticker** (per `stats.display_interval_seconds`): `PSKReporter: <TOTAL> TOTAL / <CW> CW / <RTTY> RTTY / <FT8> FT8 / <FT4> FT4 / <MSK144> MSK144 / <PSK31/63> PSK31/63`
  401: 
  402: ### Sample Session
  403: 
  404: Below is a hypothetical telnet session showing the documented commands in action (server replies are shown after each input):
  405: 
  406: ```
  407: telnet localhost 9300
  408: Experimental DX Cluster
  409: Please login with your callsign
  410: 
  411: Enter your callsign:
  412: N1ABC
  413: Hello N1ABC, you are now connected.
  414: Type HELP for available commands.
  415: HELP
  416: Available commands:
  417: ... (supported modes/bands summary)
  418: SHOW/DX 5
  419: DX1 14.074 FT8 599 N1ABC>W1XYZ
  420: DX2 14.070 FT4 26 N1ABC>W2ABC
  421: ...
  422: PASS BAND 20M
  423: Filter set: Band 20m
  424: PASS MODE FT8,FT4
  425: Filter set: Modes FT8, FT4
  426: PASS CONFIDENCE P,V
  427: Confidence symbols enabled: P, V
  428: SHOW FILTER
  429: Current filters: BAND=only: 20m | MODE=only: FT8, FT4 | CONFIDENCE=only: P, V | ...
  430: BAND: allow=20m block=NONE (effective: only: 20m)
  431: MODE: allow=FT8, FT4 block=NONE (effective: only: FT8, FT4)
  432: CONFIDENCE: allow=P, V block=NONE (effective: only: P, V)
  433: ...
  434: REJECT MODE FT4
  435: Mode filters disabled: FT4
  436: RESET FILTER
  437: Filters reset to defaults
  438: BYE
  439: 73!
  440: ```
  441: 
  442: Use these commands interactively to tailor the spot stream to your operating preferences.
  443: 
  444: ### Telnet Throughput Controls
  445: 
  446: The telnet server fans every post-dedup spot to every connected client. When PSKReporter or both RBN feeds spike, the broadcast queue can saturate and you'll see `Broadcast channel full, dropping spot` along with a rising `Telnet drops` metric in the stats ticker (Q/C/W = broadcast queue drops / per-client queue drops / sender write-failure disconnects). Tune the `telnet` block in `data/config/runtime.yaml` to match your load profile:
  447: 
  448: - `broadcast_workers` keeps the existing behavior (`0` = auto at half your CPUs, minimum 2).
  449: - `broadcast_queue_size` controls the global queue depth ahead of the worker pool (default `2048`); larger buffers smooth bursty ingest before anything is dropped.
  450: - `worker_queue_size` controls how many per-shard jobs each worker buffers before dropping a shard assignment (default `128`).
  451: - `client_buffer_size` defines how many spots a single telnet session can fall behind before its personal queue starts dropping (default `128`).
  452: - `control_queue_size` bounds per-client control output (bulletins, prompts, keepalives). Control always drains before spots; a full control queue disconnects the client.
  453: - `writer_batch_max_bytes` and `writer_batch_wait_ms` control per-connection control-first writer micro-batching (defaults `16384` bytes and `5ms`).
  454: - `reject_workers`, `reject_queue_size`, and `reject_write_deadline_ms` move reject-banner I/O off the accept loop with bounded resources (defaults `2`, `1024`, `500ms`).
  455: - `broadcast_batch_interval_ms` micro-batches outbound broadcasts to reduce mutex/IO churn (default `250`; set to `0` for immediate sends). Each shard flushes on interval or when the batch reaches its max size, preserving order per shard.
  456: - `login_line_limit` caps how many bytes a user can enter at the login prompt (default `32`). Keep this tight to prevent hostile clients from allocating massive buffers before authentication.
  457: - `command_line_limit` caps how long any post-login command may be (default `128`). Raise this when operators expect comma-heavy filter commands or scripted clients that send longer payloads.
  458: - `max_prelogin_sessions` hard-caps unauthenticated sockets (default `256`) so floods cannot consume unbounded resources before callsign login.
  459: - `prelogin_timeout_seconds` caps total accept->callsign time for unauthenticated sessions (default `15`).
  460: - `accept_rate_per_ip` and `accept_burst_per_ip` enforce per-IP prelogin admission (Go `x/time/rate`; defaults `3/s` and `6`).
  461: - `accept_rate_per_subnet` and `accept_burst_per_subnet` enforce per-subnet admission (`/24` IPv4, `/48` IPv6; defaults `24/s` and `48`).
  462: - `accept_rate_global` and `accept_burst_global` enforce cluster-wide prelogin admission (defaults `300/s` and `600`).
  463: - `accept_rate_per_asn` and `accept_burst_per_asn` enforce per-ASN prelogin admission from IPinfo metadata (defaults `40/s` and `80`).
  464: - `accept_rate_per_country` and `accept_burst_per_country` enforce per-country prelogin admission from IPinfo metadata (defaults `120/s` and `240`).
  465: - `prelogin_concurrency_per_ip` limits simultaneous unauthenticated sessions per source IP (default `3`).
  466: - `admission_log_interval_seconds`, `admission_log_sample_rate`, and `admission_log_max_reason_lines_per_interval` guard reject logging volume (defaults `10`, `0.05`, and `20`).
  467: - `read_idle_timeout_seconds` refreshes the read deadline for logged-in sessions (default `86400`); timeouts do not disconnect, they simply continue waiting for input.
  468: - `login_timeout_seconds` remains as a legacy fallback knob; Tier-A prelogin gating uses `prelogin_timeout_seconds`.
  469: - `drop_extreme_rate`, `drop_extreme_window_seconds`, and `drop_extreme_min_attempts` enforce a safety valve for slow clients: once the drop rate crosses the threshold over the window (after the minimum attempts), the session is disconnected.
  470: - `keepalive_seconds` emits a CRLF to every connected client on a cadence (default `120`; `0` disables). Blank lines sent by clients are treated as keepalives and get an immediate CRLF reply so idle TCP sessions stay open.
  471: 
  472: Increase the queue sizes if you see the broadcast-channel drop message frequently, or raise `broadcast_workers` when you have CPU headroom and thousands of concurrent clients.
  473: 
  474: ### Archive Durability (Pebble)
  475: 
  476: The optional Pebble archive is built to stay out of the hot path: enqueue is non-blocking and drops when backpressure builds. With the archive enabled, you can tune durability vs throughput:
  477: 
  478: - `archive.synchronous`: defaults to `off` for maximum throughput when the archive is disposable; `normal`, `full`, or `extra` enable fsync for stronger crash safety.
  479: - `archive.auto_delete_corrupt_db`: when true, the server deletes the archive directory on startup if Pebble reports corruption (or the path is not a directory), then recreates an empty store.
  480: - `archive.busy_timeout_ms` and `archive.preflight_timeout_ms` are ignored for Pebble (retained for compatibility).
  481: 
  482: Operational guidance: enable `auto_delete_corrupt_db` only if the archive is truly disposable. If you need to preserve data through crashes, leave auto-delete off and raise synchronous to `normal`/`full` (or disable the archive entirely).
  483: 
  484: ## Project Structure
  485: 
  486: ```
  487: .
  488: ├─ data/config/            # Runtime configuration (split YAML files)
  489: │  ├─ app.yaml             # Server identity, stats interval, console UI
  490: │  ├─ ingest.yaml          # RBN/PSKReporter/human ingest + call cache
  491: │  ├─ dedupe.yaml          # Primary/secondary dedupe policy windows
  492: │  ├─ pipeline.yaml        # Call correction, harmonics, spot policy
  493: │  ├─ data.yaml            # CTY/FCC/skew + grid DB tuning
  494: │  ├─ runtime.yaml         # Telnet server settings, buffer/filter defaults
  495: │  └─ mode_allocations.yaml # Mode inference for RBN/human ingest
  496: ├─ config/                 # YAML loader + defaults (merges config directory)
  497: ├─ cmd/                    # Helper binaries (CTY lookup, skew fetch, analysis)
  498: ├─ rbn/, pskreporter/, telnet/, dedup/, filter/, spot/, stats/, gridstore/  # Core packages
  499: ├─ data/cty/cty.plist      # CTY prefix database for metadata lookups
  500: ├─ go.mod / go.sum         # Go module definition + checksums
  501: └─ main.go                 # Entry point wiring ingest, protections, telnet server
  502: ```
  503: ## Code Walkthrough
  504: 
  505: - `main.go` glues together ingest clients (RBN/PSKReporter), protections (dedup, call correction, harmonics, frequency averaging), persistence (grid store), telnet server, dashboard, schedulers (FCC ULS, skew), and graceful shutdown. Helpers are commented so you can follow the pipeline without prior cluster context.
  506: - `internal/correctionflow/` is the authoritative shared call-correction core (runtime parameter resolution, settings mapping, confidence mapping, resolver selection/gates, stabilizer delay policy, temporal decoder policy, apply/suppress rails). `main.go` calls this package directly; `cmd/rbn_replay` uses the same policy path and keeps replay-only wrappers in `cmd/rbn_replay/pipeline.go` for harness wiring.
  507: - `telnet/server.go` documents the connection lifecycle, broadcast sharding, filter commands, and how per-client filters interact with the shared ring buffer.
  508: - `buffer/` explains the lock-free ring buffer used by SHOW/DX and broadcasts; it stores atomic spot pointers and IDs to avoid partial reads.
  509: - `config/` describes the YAML schema, default normalization, and `Print` diagnostics. The “Configuration Loader Defaults” section mirrors these behaviors.
  510: - `cty/` covers longest-prefix CTY lookups and cache metrics. `spot/` holds the canonical spot struct, formatting, hashing, validation, callsign utilities, harmonics/frequency averaging/correction helpers, and custom-SCP support.
  511: - `dedup/`, `filter/`, `gridstore/`, `skew/`, and `uls/` each have package-level docs and function comments outlining how they feed or persist data without blocking ingest.
  512: - `rbn/` and `pskreporter/` detail how each source is parsed, normalized, skew-corrected, and routed into the ingest CTY/ULS gate before deduplication.
  513: - `commands/` and `cmd/*` binaries include focused comments explaining the helper CLIs for SHOW/DX, CTY lookup, and skew prefetching.
  514: - `cmd/rbn_replay/` keeps replay-only metrics/reporting concerns (AB metrics, run artifacts) while sharing correction policy helpers with runtime. Replay is resolver-primary only (no legacy comparison path/artifacts) and tracks resolver outcomes including neighborhood, recent-plus1, stabilizer-delay proxy, and temporal decoder counters; see `docs/rbn_replay.md`.
  515: 
  516: ## Getting Started
  517: 
  518: 1. Update `data/config/ingest.yaml` with your preferred callsigns for the `rbn`, `rbn_digital`, optional `human_telnet`, and optional `pskreporter` sections. Optionally list `pskreporter.modes` (e.g., [`FT8`, `FT4`]) to accept only those modes after subscribing to the catch-all topic. If you enable `human_telnet`, review `data/config/mode_allocations.yaml` (used to infer CW vs LSB/USB when the incoming spot line does not include an explicit mode token).
  519: 2. If peering with DXSpider clusters, edit `data/config/peering.yaml`: set `local_callsign`, optional `listen_port`, hop/version fields, and add one or more peers (host/port/password/prefer_pc9x). Outbound peer sessions are explicit opt-in: each peer must set `enabled: true` to dial; omitted `enabled` defaults to disabled. ACLs are available for inbound (`allow_ips`/`allow_callsigns`). Topology persistence is optional (disabled by default); set `peering.topology.db_path` to enable SQLite caching (WAL mode) with the configured retention.
  520: 2. Optionally enable/tune `call_correction` (master `enabled` switch, minimum corroborating spotters, required advantage, confidence percent, recency window, max edit distance, per-mode distance models, and `invalid_action` failover). `distance_model_cw` switches CW between the baseline rune-based Levenshtein (`plain`) and a Morse-aware cost function (`morse`), `distance_model_rtty` toggles RTTY between `plain` and a Baudot/ITA2-aware scorer (`baudot`), while USB/LSB voice modes always stay on `plain` because those reports are typed by humans. Family behavior is tuned under `call_correction.family_policy` (slash precedence, truncation matching/relaxation, and telnet family suppression). Resolver-primary contested behavior is tuned with `resolver_neighborhood_*`, `resolver_recent_plus1_*`, and optional conservative Bayesian gate knobs in `call_correction.bayes_bonus.*`. Confusion-model behavior is controlled with `call_correction.confusion_model_enabled`, `call_correction.confusion_model_file`, and `call_correction.confusion_model_weight` (`0` disables effect). Custom SCP replacement behavior is configured under `call_correction.custom_scp.*` (runtime-learned static membership + long-horizon recency tiers with CW/RTTY SNR floors). To prevent self-reinforcing confidence loops, custom SCP admission records only spots with confidence glyph `V`; `S`, `P`, `C`, and `?` are not admitted. For telnet cleanup of newly-seen/busted calls, tune `call_correction.stabilizer_enabled`, `stabilizer_delay_seconds`, `stabilizer_max_checks`, `stabilizer_p_delay_confidence_percent`, `stabilizer_p_delay_max_checks`, `stabilizer_ambiguous_max_checks`, `stabilizer_edit_neighbor_*`, `stabilizer_timeout_action`, and `stabilizer_max_pending`; stabilizer delay eligibility is limited to `?`, `S`, and `P` (`V`/`C` always pass through). For fixed-lag temporal decoding across runtime and replay, tune `call_correction.temporal_decoder.*` (`enabled`, `scope`, `lag_seconds`, `max_wait_seconds`, beam/candidate bounds, transition penalties, commit gates, and `overflow_action`).
  521: 3. Optionally enable/tune `harmonics` to drop harmonic CW/USB/LSB/RTTY spots (master `enabled`, recency window, maximum harmonic multiple, frequency tolerance, and minimum report delta).
  522: 4. Set `spot_policy.max_age_seconds` to drop stale spots before they're processed further. For CW/RTTY frequency smoothing, tune `spot_policy.frequency_averaging_seconds` (window), `spot_policy.frequency_averaging_tolerance_hz` (allowed deviation), and `spot_policy.frequency_averaging_min_reports` (minimum corroborating reports).
  523: 5. (Optional) Enable `skew.enabled` after generating `skew.file` via `go run ./cmd/rbnskewfetch` (or let the server fetch it at the next 00:30 UTC window). The server applies each skimmer's multiplicative correction before normalization so SSIDs stay unique.
  524: 6. Grids and CTY metadata are persisted in Pebble (`grid_db`, default `data/grids/pebble`). Tune `grid_flush_seconds` for batch cadence, `grid_cache_size` for the unified call metadata LRU, `grid_cache_ttl_seconds` for grid TTL inside the cache, `grid_block_cache_mb`/`grid_bloom_filter_bits`/`grid_memtable_size_mb`/`grid_l0_stop_writes_threshold` for Pebble read/write tuning, and `grid_write_queue_depth`/`grid_ttl_days` for buffering and retention.
  525: 7. Adjust `stats.display_interval_seconds` in `data/config/app.yaml` to control how frequently runtime statistics print to the console (defaults to 30 seconds).
  526: 8. Install dependencies and run:
  528: 	 ```pwsh
  529: 	 go mod tidy
  530: 	 go run .
  531: 	 ```
  531: 9. Connect via `telnet localhost 9300` (or your configured `telnet.port`), enter your callsign, and the server will immediately stream real-time spots.
  533: 
  534: ## Path Reliability (telnet)
  535: 
  536: - Maintains a single directional FT8-equivalent bucket family per path: FT8/FT4/CW/RTTY/PSK all feed the same buckets. Voice modes (LSB/USB) are display-only. Buckets store linear power (FT8-equivalent) with exponential decay, using H3 res-2 buckets plus coarse H3 res-1 buckets, per-band half-lives, and staleness purging (per-band stale window).
  537: - H3 size reference (average edge length): res-2 ≈ 158 km; res-1 ≈ 418 km.
  538: - Maidenhead grids (4–6 chars) are converted to a representative lat/lon by taking the center of the grid square (4‑char: 2° × 1°). That point is mapped into H3 res‑2 (fine/local) and res‑1 (coarse/regional) cells so we can blend local and regional evidence deterministically.
  539: - H3 cells are stored as stable 16‑bit proxy IDs via precomputed tables in `data/h3`. If grids are invalid or H3 tables are unavailable, the path is treated as insufficient data.
  540: - Telnet lines show a single glyph in the comment area when enabled, reflecting a merged path estimate adjusted for the user's noise class. Glyph symbols are configurable via `glyph_symbols` (defaults: `+` high, `=` medium, `-` low, `!` unlikely); insufficient data uses `glyph_symbols.insufficient` (default `?`).
  541: - Commands: `SET GRID <grid>` to set/confirm your location (4-6 char), `SET NOISE <QUIET|RURAL|SUBURBAN|URBAN|INDUSTRIAL>` to apply a DX->you noise penalty. Defaults: QUIET/0 dB.
  542: - System log: every 5 minutes emits `Path predictions (5m)` (combined vs insufficient, with no-sample vs low-weight split), `Path buckets (5m)` (per-band bucket counts), and `Path weight dist (5m)` (per-band combined weight histogram).
  543: - Config: `data/config/path_reliability.yaml` controls clamps (-25..15 dB FT8-equiv), per-band half-life, stale window multiplier (`stale_after_half_life_multiplier`), min effective weights (`min_effective_weight`), min fine weight blending (`min_fine_weight`), reverse hint discount, merge weights, per-mode glyph thresholds (`mode_thresholds` with `high/medium/low/unlikely` keys, including LSB/USB), fallback `glyph_thresholds`, glyph symbols (`glyph_symbols`), beacon weight cap (default 1), mode offsets (FT4/CW/RTTY/PSK; CW/RTTY/PSK defaults assume 500 Hz -> 2500 Hz bandwidth correction of -7 dB), and noise offsets.
  544: 
  545: ## Propagation Glyphs (operator meaning)
  546: - We build a path score from recent spots on each band, weighted so newer reports matter most and noise environment (QUIET/RURAL/SUBURBAN/URBAN/INDUSTRIAL) is accounted for.
  547: - The score maps to a likelihood glyph: High / Medium / Low / Unlikely. If there is not enough data, we show the Insufficient glyph instead of guessing.
  548: - Only after a normal glyph is chosen do we check for rare space‑weather overrides.
  549: - `R` (radio blackout) appears only when the path is mostly sunlit and the X‑ray level meets the configured R‑threshold; it is band‑specific.
  550: - `G` (geomagnetic storm) appears only when the path is high‑latitude and Kp meets the configured G‑threshold; it is band‑specific.
  551: - Overrides are intentional and rare: they mean strong, path‑relevant space weather is likely to invalidate the normal estimate.
  552: 
  553: ## Solar Weather Overrides (optional)
  554: - Optional single-slot glyph overrides for strong space-weather events: `R` (radio blackout, R2+ thresholds) and `G` (geomagnetic storm, G2+ thresholds). Overrides only appear when the event is active *and* the path is relevant (sunlit fraction for `R`, high-latitude exposure for `G`), and they never replace the insufficient-data symbol.
  555: - Overrides are band-aware: each R/G severity level has an explicit band list, and unknown/empty bands never receive overrides. R has precedence per-band; if a band is not eligible for the active R level, an eligible G can still apply.
  556: - Inputs: GOES X-ray primary feed (corrected 0.1–0.8 nm flux) and observed 3-hour Kp. Fetches run every 60 seconds with conditional GET and in-memory caching.
  557: - Config: `data/config/solarweather.yaml` (disabled by default) pins thresholds, band lists, hold-down windows, hysteresis rules, and gating tolerances with detailed operator notes.
  558: 
  559: ## Configuration Loader Defaults
  560: 
  561: `config.Load` accepts a directory (merging all YAML files); the server defaults to `data/config`. It normalizes missing fields and refuses to start when time strings are invalid. Key fallbacks:
  562: 
  563: - Stats tickers default to `30s` when unset. Telnet queues fall back to `broadcast_queue_size=2048`, `worker_queue_size=128`, `client_buffer_size=128`, and friendly greeting/duplicate-login messages are injected if blank.
  564: - Call correction uses conservative resolver-primary baselines unless overridden: `min_consensus_reports=4`, `family_policy.slash_precedence_min_reports=2`, `family_policy.truncation.max_length_delta=1`, `family_policy.truncation.min_shorter_length=3`, `family_policy.truncation.relax_advantage.min_advantage=0`, `min_advantage=1`, `min_confidence_percent=70`, `recency_seconds=45`, `max_edit_distance=2`, `frequency_tolerance_hz=500`, `voice_frequency_tolerance_hz=2000`, `invalid_action=broadcast`, `resolver_recent_plus1_enabled=true`, `resolver_recent_plus1_min_unique_winner=3`, `resolver_recent_plus1_require_subject_weaker=true`, `resolver_recent_plus1_max_distance=1`, and `resolver_neighborhood_max_distance=1`. Bayesian gate defaults are conservative and disabled by default: `bayes_bonus.enabled=false`, `weight_distance1_milli=350`, `weight_distance2_milli=200`, `weighted_smoothing_milli=1000`, `recent_smoothing=2`, `obs_log_cap_milli=350`, `prior_log_min_milli=-200`, `prior_log_max_milli=600`, `report_threshold_distance1_milli=450`, `report_threshold_distance2_milli=650`, `advantage_threshold_distance1_milli=700`, `advantage_threshold_distance2_milli=950`, `advantage_min_weighted_delta_distance1_milli=200`, `advantage_min_weighted_delta_distance2_milli=300`, `advantage_extra_confidence_distance1=3`, `advantage_extra_confidence_distance2=5`, `require_candidate_validated=true`, and `require_subject_unvalidated_distance2=true`. Temporal defaults are `temporal_decoder.scope=uncertain_only`, `lag_seconds=2`, `max_wait_seconds=6`, `beam_size=8`, `max_obs_candidates=8`, `stay_bonus=120`, `switch_penalty=160`, `family_switch_penalty=60`, `edit1_switch_penalty=90`, `min_score=0`, `min_margin_score=80`, `overflow_action=fallback_resolver`, `max_pending=25000`, `max_active_keys=6000`, `max_events_per_key=32`. If `resolver_neighborhood_enabled=true`, `resolver_neighborhood_bucket_radius` is clamped to `[1..2]` (otherwise `[0..2]`) and `resolver_neighborhood_max_distance<=0` is normalized to `1`. Empty distance models default to `plain`; negative distance-3 extras and reliability/confusion weights are clamped to zero. When `custom_scp.enabled=true`, defaults include `history_horizon_days=60`, `static_horizon_days=395`, `min_snr_db_cw=4`, `min_snr_db_rtty=3`, `resolver_min_score=5`, `stabilizer_min_score=5`, `s_floor_min_score=3`, and bounded storage defaults (`max_keys=500000`, `max_spotters_per_key=64`).
  565: - Harmonic suppression clamps to sane minimums (`recency_seconds=120`, `max_harmonic_multiple=4`, `frequency_tolerance_hz=20`, `min_report_delta=6`, `min_report_delta_step>=0`).
  566: - Spot policy defaults prevent runaway averaging: `max_age_seconds=120`, `frequency_averaging_seconds=45`, `frequency_averaging_tolerance_hz=300`, `frequency_averaging_min_reports=4`.
  567: - Archive defaults keep writes lightweight: `queue_size=10000`, `batch_size=500`, `batch_interval_ms=200`, `cleanup_interval_seconds=3600`, `synchronous=off`, `auto_delete_corrupt_db=false` (`busy_timeout_ms`/`preflight_timeout_ms` are ignored for Pebble).
  568: - CTY falls back to `data/cty/cty.plist`.
  569: - FCC ULS fetches use the official URL/paths (`archive_path=data/fcc/l_amat.zip`, `db_path=data/fcc/fcc_uls.db`, `temp_dir` inherits from `db_path`), and refresh times must parse as `HH:MM` or loading fails fast.
  570: - Grid store defaults: `grid_db=data/grids/pebble`, `grid_flush_seconds=60`, `grid_cache_size=100000`, `grid_cache_ttl_seconds=0`, `grid_block_cache_mb=64`, `grid_bloom_filter_bits=10`, `grid_memtable_size_mb=32`, `grid_l0_compaction_threshold=4`, `grid_l0_stop_writes_threshold=16`, `grid_write_queue_depth=64`, with TTL/retention floors of zero to avoid negative durations.
  571: - Dedup windows are coerced to zero-or-greater; `output_buffer_size` defaults to `1000` so bursts do not immediately drop spots.
  572: - Buffer capacity defaults to `300000` spots; skew downloads default to SM7IUN's CSV (`url=https://sm7iun.se/rbnskew.csv`, `file=data/skm_correction/rbnskew.json`, `refresh_utc=00:30`) with `min_abs_skew=1`.
  573: - `config.Print` writes a concise summary of the loaded settings to stdout for easy startup diagnostics.
  574: 
  575: ## Testing & Tooling
  576: 
  577: - `go test ./...` validates packages (not all directories contain tests yet).
  578: - `gofmt -w ./...` keeps formatting consistent.
  579: - `go run cmd/ctylookup -data data/cty/cty.plist` lets you interactively inspect CTY entries used for validation (portable calls resolve by location prefix).
  580: - `go run cmd/rbnskewfetch -out data/skm_correction/rbnskew.json` forces an immediate skew-table refresh (the server still performs automatic downloads at 00:30 UTC daily).
  581: 
  582: Let me know if you want diagrams, sample logs, or scripted deployment steps added next.
