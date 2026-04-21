# DXSummit HTTP Ingest

This package owns the optional DXSummit HTTP polling feed.

## Runtime Contract

- Disabled by default under `dxsummit.enabled: false`.
- When enabled, one goroutine polls `api/v1/spots`; requests do not overlap.
- Each request is bounded by `request_timeout_ms` and `max_response_bytes`.
- The output queue is capped by `spot_channel_size`; full queues drop the newest DXSummit row and increment health counters.
- Shutdown cancels the polling context and waits for the goroutine to close its output channel.
- Cursor state is O(1): the client retains only the highest observed DXSummit row ID, not a per-ID history.

Normal request shape:

```text
GET {base_url}?limit={max_records_per_poll}&from_time={now-lookback_seconds}&to_time={now}&include=HF,VHF,UHF&refresh={unix_ms}
```

Startup is seed-only by default. With `startup_backfill_seconds: 0`, the first successful poll establishes the high-water ID and emits no spots. A positive startup backfill emits only rows newer than that startup window after high-water filtering.

## Spot Semantics

Accepted rows become normal upstream human spots:

- `SourceType=UPSTREAM`
- `SourceNode=DXSUMMIT`
- `IsHuman=true`

DXSummit spotter calls ending in `-@` preserve that marker for display and archive provenance. Embedded `@` forms are rejected. Metadata lookups, ULS checks, and grid-cache lookups strip the final marker before lookup.

DXSummit coordinates are intentionally not used to populate `DXMetadata.Grid` or `DEMetadata.Grid`. Existing CTY and grid-cache enrichment may still fill those fields later from callsign-derived data.

DXSummit spots are not skimmer spots. They do not seed skimmer-only mode evidence and remain in the existing `UPSTREAM` source class for broad path-report buckets. Source stats use the fixed label `DXSUMMIT`.

## Band Scope

The feed requests `HF,VHF,UHF`. UHF includes observed 13cm rows around 2.4 GHz, so the shared band table supports both 13cm segments:

- `2300000..2310000` kHz
- `2390000..2450000` kHz

Microwave bands beyond that 13cm coverage are out of scope for this ingest path.
