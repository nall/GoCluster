# Peer Behavior

This directory owns DXSpider-style cluster peering.

## What The Peer Layer Does

- accepts inbound peer sessions
- optionally opens outbound peer sessions
- exchanges spot-bearing and control-plane frames
- maintains topology and keepalive traffic

## Enablement And Direction

Peering is explicit:

- each peer record in `peering.peers[]` must be individually enabled
- `direction: outbound` dials the peer
- `direction: inbound` waits for the peer to connect
- `direction: both` allows either side to establish the link
- omitted `direction` defaults to `outbound`
- omitted `family` defaults to `dxspider`
- the node can run receive-only peering when `forward_spots` is false or omitted

Inbound admission is explicit:

- the global `peering.acl.*` block is only a coarse prefilter
- a connecting peer must also match an enabled peer record by `remote_callsign`
- `allow_ips` on a peer record optionally pins that peer to specific source IPs/CIDRs
- if `direction: both` produces simultaneous inbound/outbound attempts, the first established session wins and later duplicates are rejected

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

Configured peer family drives inbound startup:

- `dxspider` peers keep the strict inbound path and still need remote `PC20` to finish startup
- `ccluster` peers can complete inbound startup from a `PC18` banner carrying `CC Cluster Version:` or from the first valid `PC92`
- outbound CC behavior is unchanged in this slice

Keepalive and topology traffic has higher priority than normal outbound spot backlog.

If the control lane saturates:

- the session closes
- reconnect backoff takes over

That is preferred over silently drifting until the remote side times out.

## Operator View

The main landing page now keeps only the high-level peering summary. The detailed forwarding and receive-only behavior belongs here because it is implementation-heavy and changes more often than the operator quickstart.

For the high-level overview, see [`../README.md`](../README.md).
