# Peer Behavior

This directory owns DXSpider-style cluster peering.

## What The Peer Layer Does

- accepts inbound peer sessions
- optionally opens outbound peer sessions
- exchanges spot-bearing and control-plane frames
- maintains topology and keepalive traffic

## Enablement And Direction

Peering is explicit:

- each outbound peer must be individually enabled
- the node can run receive-only peering when `forward_spots` is false or omitted

In receive-only mode:

- inbound peer spots still ingest locally
- maintenance traffic still runs
- only local `DX` command spots are peer-published

With `forward_spots: true`:

- normal transit forwarding is re-enabled
- local acceptance still gates whether relayed traffic continues onward

## Publishing Rules

The runtime is intentionally conservative about what it republishes to peers.

- local non-test human and manual spots are eligible for normal peer publishing
- skimmer-origin spots are not blindly republished as local human traffic
- local `DX` command spots remain the operator-authored exception

## Control Plane

Keepalive and topology traffic has higher priority than normal outbound spot backlog.

If the control lane saturates:

- the session closes
- reconnect backoff takes over

That is preferred over silently drifting until the remote side times out.

## Operator View

The main landing page now keeps only the high-level peering summary. The detailed forwarding and receive-only behavior belongs here because it is implementation-heavy and changes more often than the operator quickstart.

For the high-level overview, see [`../README.md`](../README.md).
