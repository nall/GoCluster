# docs/dev-runbook.md

This runbook defines the expected validation commands and when to use them.
For Non-trivial closeout, this file is the required checker source; do not rely
only on the abbreviated baseline in `AGENTS.md`.

## Principles
- Run the smallest useful check early, then broaden.
- Use incremental validation after each meaningful slice.
- Use the full suite before calling a Non-trivial task complete.
- Report commands and results honestly.

## Baseline environment
Expected Go toolchain:
- `go`
- `go test`
- `go vet`
- `staticcheck`
- `golangci-lint` pinned to the CI version in `.github/workflows/ci.yml`

If a tool is missing locally, report that fact explicitly and treat it as a validation gap, not a silent success.

## Default command set by task type

### Small change
Minimum expected sequence:
1. targeted test if available
2. `go test ./...`

Add `go vet ./...` and `staticcheck ./...` if the change touches exported/shared logic, parsing, or anything likely to affect multiple packages.

### Non-trivial change
Default full sequence:
1. targeted package test(s) during development
2. `go test ./...`
3. `go vet ./...`
4. `staticcheck ./...`
5. `golangci-lint run ./... --config=.golangci.yaml`

Also required as applicable:
- `go test -race ./...` for concurrency, queues, timers, cancellation, lifecycle, shutdown, or long-lived connections
- fuzzing for parser/protocol changes
- benchmarks for hot-path changes
- pprof for meaningful performance claims

## Targeted test examples
Use the narrowest useful targeted commands during implementation, then run the broader suite.

Examples:
- `go test ./internal/cluster -run TestSlowClientDropPolicy`
- `go test ./internal/parser -run TestRejectsMalformedControlBytes`
- `go test ./... -run TestGracefulShutdown`

## Fuzz guidance
Use fuzzing for parser/protocol work where malformed or adversarial input matters.

Examples:
- `go test ./internal/parser -fuzz=FuzzLineParser -fuzztime=10s`
- `go test ./... -fuzz=FuzzCommandDecoder -fuzztime=10s`

Rules:
- keep fuzz inputs bounded
- seed with real malformed cases when available
- report fuzz command and result

## Race guidance
Race checks are mandatory when touching:
- goroutines
- channels
- queue ownership
- timers/tickers
- connection lifecycle
- cancellation/shutdown
- shared mutable state

Command:
- `go test -race ./...`

If the repo has a narrower stable race target, it may be added in addition to, not instead of, the full run unless a temporary waiver is explicitly granted.

## Benchmark guidance
Benchmarks are expected for hot-path changes such as:
- fan-out/broadcast
- parser loops
- allocation-sensitive handlers
- queue operations
- lock-contention-sensitive paths

Examples:
- `go test ./internal/parser -bench . -benchmem`
- `go test ./internal/cluster -bench BenchmarkBroadcast -benchmem`

Report:
- ns/op
- allocs/op
- bytes/op
- before/after comparison when claiming improvement

## Profiling guidance
Use pprof when:
- benchmark numbers regress or surprise you
- lock contention is suspected
- memory growth or retention is in question
- CPU cost of a hot path changed materially

Examples:
- `go test ./internal/cluster -bench BenchmarkBroadcast -benchmem -cpuprofile cpu.out -memprofile mem.out`
- `go tool pprof -top cpu.out`
- `go tool pprof -top mem.out`

## Escape-analysis spot checks
Use when allocations or ownership are unclear:
- `go test ./... -gcflags=all=-m`

Do not dump noisy compiler output into the final summary. Summarize only relevant findings.

## Suggested execution sequence for Non-trivial work
Example cadence:
1. after milestone 1: targeted tests
2. after milestone 2: targeted tests + `go test ./...`
3. before closeout: `go vet ./...`, `staticcheck ./...`
4. final if applicable: `go test -race ./...`, fuzz, benchmark, pprof

## Reporting format
In the final summary, list each command with:
- command
- why it was run
- result
- whether it was incremental or final

Example:
- `go test ./internal/cluster -run TestSlowClientDropPolicy` - targeted drop-policy regression - pass - incremental
- `go test ./...` - baseline regression suite - pass - final
- `go test -race ./...` - lifecycle/concurrency verification - pass - final
