# PSKReporter Ingest

This directory owns PSKReporter MQTT ingest and normalization into canonical spots.

## What It Does

- subscribes to the PSKReporter MQTT feed
- filters accepted modes downstream
- converts JSON payloads into canonical spot records
- preserves locator-based grids in metadata
- can route selected modes into path reliability only

## Key Runtime Rules

- the client subscribes to one catch-all topic and filters modes after receipt
- spots with explicit `0 dB` reports or missing reports are dropped before ingest
- PSK variants are normalized into the `PSK` family for filtering and stats while keeping the original variant for display
- PSKReporter spots do not carry a free-form comment string in the runtime pipeline

## FT Frequency Handling

For `FT2`, `FT4`, and `FT8`, the runtime canonicalizes the operational frequency to the nearest configured dial frequency from `mode_inference.digital_seeds`.

That canonical dial frequency is then used for:

- dedup
- mode inference
- archive storage
- FT confidence burst grouping
- telnet output

The raw observed RF frequency is still kept separately for diagnostics and archive records.

## Path-Only Modes

Modes listed in `pskreporter.path_only_modes` go directly to path reliability and bypass:

- CTY validation
- dedup
- telnet fan-out
- archive
- peer output

This is how the runtime can use WSPR-like reports for propagation hints without publishing them as normal DX-cluster spots.

## Queueing And Backpressure

The MQTT ingest path is bounded by config:

- inbound worker count
- inbound queue depth
- enqueue timeout for QoS1 and QoS2

Under pressure:

- QoS0 drops when the queue is full
- QoS1 and QoS2 disconnect after the configured enqueue timeout

For the operator-facing summary, see [`../README.md`](../README.md). For path scoring details, see [`../pathreliability/README.md`](../pathreliability/README.md).
