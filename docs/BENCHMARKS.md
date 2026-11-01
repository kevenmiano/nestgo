# NestGo HTTP Benchmarks

The HTTP benchmarks are located in `runtime/http_benchmark_test.go` and
`runtime/http_engine_benchmark_test.go`.

## Run

```sh
go test ./runtime -run '^$' -bench 'BenchmarkHTTP' -benchmem -count=5 -timeout=15m
go test ./runtime -run '^$' -bench 'BenchmarkAdapter' -benchmem -count=5 -timeout=15m
```

Use fixed `GOMAXPROCS`, Go version and hardware for before/after comparisons.
For robust regressions, save raw output and compare using benchstat.

```sh
go test ./runtime -run '^$' -bench 'BenchmarkHTTP' -benchmem -count=10 > before.txt
# Repeat from the candidate commit under identical conditions.
go test ./runtime -run '^$' -bench 'BenchmarkHTTP' -benchmem -count=10 > after.txt
benchstat before.txt after.txt
```

`BenchmarkHTTPInvocation` uses an in-memory adapter to isolate handler
invocation. The direct `net/http` baseline uses a ServeMux and recorder, so
those measurements are NOT comparable end-to-end.

`BenchmarkHTTPEngines` measures loopback HTTP/1.1 requests with a shared
`net/http` client, including both server and client overhead. It provides
static (GET), path parameter, query parameter, and JSON body cases, each
against both actual NestGo HTTP adapters. The reported p95 and p99 values
come from measured per-request latencies and are descriptive benchmark
metrics, not service-level objectives.

`BenchmarkAdapterConcurrency` uses Go's parallel benchmark harness.
`BenchmarkAdapterBootstrap` measures route registration without networking.
Avoid interpreting differences smaller than run-to-run variance as
performance improvements.

Do not use these microbenchmarks to claim production throughput. A separate
long-running external load test with fixed connections, warm-up, and
latency histograms is needed to validate production p95/p99.
