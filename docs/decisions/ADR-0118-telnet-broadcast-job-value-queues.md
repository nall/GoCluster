# ADR-0118: Telnet Broadcast Job Value Queues

- Status: Accepted
- Date: 2026-05-06
- Decision Origin: Design

## Context
Post-optimization profiling from the 2026-05-06 runtime still showed
`telnet.(*Server).dispatchSpotToWorkers` as a flat allocation source. The
current fan-out path allocates one `*broadcastJob` per non-empty worker shard
before enqueueing it to a per-worker channel. The job is immutable after
construction and contains only the spot snapshot pointer, policy booleans, the
cached client shard slice, and the original enqueue timestamp.

The goal is to remove the per-shard heap object without changing telnet
delivery semantics, queue capacity, drop policy, shard ownership, batching,
shutdown, filtering, or latency metrics.

## Decision
Use value-based worker jobs for telnet spot fan-out:

- Store worker queues as `chan broadcastJob` instead of `chan *broadcastJob`.
- Construct one `broadcastJob` value per non-empty shard and send it with the
  existing non-blocking enqueue/drop select.
- Keep worker queues as never-closed runtime channels; shutdown continues to
  use `Server.shutdown`.
- Keep batch workers' shutdown behavior unchanged: flush only jobs already
  received into the in-memory batch and do not drain the worker channel.
- Treat zero-value jobs as no-ops inside workers and delivery, without making
  closed worker channels a supported production path.
- Preserve the existing immutable shard and owned spot snapshot contracts.

## Alternatives considered
1. Keep pointer jobs.
   - Rejected because the allocation source is directly tied to the pointer job
     representation and remains visible in alloc profiles.
2. Use `sync.Pool` for `broadcastJob`.
   - Rejected because asynchronous worker ownership makes pool lifetime harder
     to reason about, and a value send removes the allocation without reuse
     risk.
3. Replace per-worker queues with one shared queue.
   - Rejected because that changes worker/shard ordering and backpressure
     distribution rather than only changing job representation.
4. Drain worker queues on shutdown.
   - Rejected because it changes existing best-effort shutdown semantics and
     can extend shutdown latency under backlog.

## Consequences
### Benefits
- Removes the per-shard job heap allocation in the accepted worker-queue path.
- Preserves existing queue capacities and non-blocking drop behavior.
- Keeps immutable spot snapshot and client shard ownership unchanged.
- Avoids object-pool lifetime risk.

### Risks
- Value channels retain larger bounded queue buffers than pointer channels.
- Batch workers also retain larger bounded scratch buffers while batching is
  enabled.
- Tests must avoid accidentally measuring the drop path when proving accepted
  enqueue allocation behavior.

### Operational impact
- No operator-visible behavior change.
- No config, command, HELP, archive schema, filter, or protocol change.
- Slow-client behavior, queue drops, latency metrics, and shutdown semantics are
  intended to remain unchanged.
- Retained memory increases by a bounded amount proportional to
  `broadcast_workers * (worker_queue_size + broadcast_batch_max)`, while
  cumulative allocation churn decreases.

## Links
- Related issues/PRs/commits: current working tree
- Related tests:
  - `telnet/server_broadcast_worker_test.go`
  - `telnet/dedupe_policy_test.go`
  - `telnet/server_filter_test.go`
  - `telnet/server_spot_snapshot_test.go`
- Related docs:
  - `docs/decisions/ADR-0042-telnet-reject-workers-and-writer-microbatching.md`
  - `docs/decisions/ADR-0056-local-self-spots-operator-authoritative-v-path.md`
  - `docs/decisions/ADR-0117-hot-path-duplicate-work-removal.md`
- Related TSRs: none
- Supersedes / superseded by: none
