# ADR-0066: DXSummit HTTP Ingest

- Status: Accepted
- Date: 2026-04-21
- Decision Origin: Design

## Context

GoCluster already ingests RBN, PSKReporter, human telnet relay, local `DX`
commands, and peer spots. DXSummit exposes a live HTTP JSON endpoint that can
provide additional human spot coverage for HF, VHF, and UHF.

Live API checks showed that `include=HF,VHF,UHF` is accepted, that `from_time`
without `to_time` can still return a full limited page, and that UHF can include
13cm rows in the 2390-2450 MHz segment. The existing band table only modeled the
2300-2310 MHz 13cm segment, so those rows could not be classified correctly.

DXSummit spotter calls may carry a trailing `-@` marker. That marker is source
provenance, similar to an RBN `-#` marker, not part of the base callsign for
metadata or licensing.

## Decision

Add an optional, disabled-by-default DXSummit HTTP polling client.

Note: ADR-0067 later made checked-in/operator YAML the runtime source of truth.
The values below document this ADR's original design surface; current startup
behavior is determined by `data/config/ingest.yaml` and missing DXSummit keys
fail config load rather than receiving loader defaults.

The YAML surface is:

```yaml
dxsummit:
  enabled: false
  name: "DXSUMMIT"
  base_url: "http://www.dxsummit.fi/api/v1/spots"
  poll_interval_seconds: 30
  max_records_per_poll: 500
  request_timeout_ms: 10000
  lookback_seconds: 300
  startup_backfill_seconds: 0
  include_bands:
    - "HF"
    - "VHF"
    - "UHF"
  spot_channel_size: 1000
  max_response_bytes: 1048576
```

The client polls with `limit`, `from_time`, `to_time`, `include`, and `refresh`.
It keeps one high-water ID, emits new rows oldest-to-newest, and does not keep a
per-ID map. Startup is seed-only unless `startup_backfill_seconds` is positive.

Accepted rows enter the existing shared pipeline as human upstream spots with
`SourceType=UPSTREAM`, `SourceNode=DXSUMMIT`, and `IsHuman=true`. DXSummit spots
are not peer-published and do not create a new `SourceType`.

Preserve a final `-@` marker on DXSummit spotter calls for display and archive
provenance. Reject embedded `@` or malformed marker forms. Strip the marker for
metadata, ULS/license, and grid-cache lookups.

Do not derive Maidenhead grids from DXSummit latitude/longitude fields in this
version. CTY and existing grid-cache enrichment remain the only grid sources.

Refactor band matching so a band can have multiple disjoint segments, and add
13cm coverage for `2390000..2450000` kHz while preserving `2300000..2310000`.

## Alternatives considered

1. IRC/CQDX bridge. Rejected for v1 because the HTTP API is simpler to bound,
   poll, and shut down.
2. Persistent cursor. Deferred because seed-only startup plus bounded lookback
   avoids replay without introducing durable state.
3. New `DXSUMMIT` source type. Rejected because downstream contracts already
   understand human upstream spots; fixed source stats provide operator
   visibility without broad source-type churn.
4. Use DXSummit coordinates for grids. Deferred because the coordinate authority
   and sign semantics need separate validation before affecting path filters.
5. Add mode include/exclude filters. Deferred to keep v1 focused on source
   ingest, bounds, and source semantics.
6. Broad microwave expansion. Rejected for this scope; only the observed 13cm
   UHF gap is required.

## Consequences

### Benefits

- Operators can enable DXSummit with explicit poll cadence and max-record knobs.
- Polling and memory are bounded by timeout, response cap, channel cap, and
  O(1) cursor state.
- DXSummit provenance remains visible through the `-@` marker.
- Existing stale admission, CTY/ULS validation, flood control, dedupe,
  correction, archive, filters, and fan-out remain the downstream enforcement
  points.
- 13cm DXSummit UHF rows around 2.4 GHz classify correctly.

### Risks

- If a poll returns exactly `max_records_per_poll`, the lookback window may be
  clipped. The client increments a truncation-warning counter and logs a
  rate-limited warning.
- Seed-only startup intentionally ignores currently visible historical rows;
  operators who want startup replay must configure `startup_backfill_seconds`.
- HTTP endpoint behavior is external to GoCluster. Timeouts, non-200 responses,
  decode errors, and oversized payloads are treated as source health failures,
  not cluster-wide failures.

### Operational impact

- Telnet DX line syntax is unchanged.
- DXSummit spots appear as human upstream spots and are not published to peers.
- Source stats use `DXSUMMIT`; broad path-report source buckets may remain
  `UPSTREAM`.
- Slow clients and existing broadcast queues are unchanged.
- Shutdown gains one explicit DXSummit client stop when the feed is enabled.

## Links

- Related issues/PRs/commits:
- Related tests: `dxsummit/client_test.go`, `config/dxsummit_config_test.go`, `spot/bands_test.go`, `main_stats_test.go`, `peer/forwarding_policy_test.go`
- Related docs: `data/config/ingest.yaml`, `data/config/README.md`, `dxsummit/README.md`, `README.md`
- Related TSRs:
- Supersedes / superseded by:
