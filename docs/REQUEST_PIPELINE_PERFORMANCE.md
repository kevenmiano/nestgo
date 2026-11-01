# Request pipeline performance — PR #122

This report records **measured**, not projected, incremental costs of the
NestGo request pipeline. It is separate from the existing HTTP transport and
typed-binding benchmarks.

- Source: [GitHub Actions run 41339194871](https://github.com/kevenmiano/nestgo/actions/runs/41339194871)
- Fixture: `runtime/http_pipeline_bench_test.go` (`BenchmarkRequestPipeline`)
- Command: `GOMAXPROCS=2 go test ./runtime -run '^$' -bench '^BenchmarkRequestPipeline$' -benchtime=300ms -count=5 -benchmem`
- Environment: Linux GitHub Actions hosted runner, Go version recorded in job logs
- Method: Median of five samples, identical compiled controller, path input,
  JSON response, and in-process mock `httpadapter.Context`. Application
  bootstrap and socket/network processing are excluded.

## Isolated results

| Scenario | Median ns/op | Change vs static baseline | B/op | allocs/op |
| --- | ---: | ---: | ---: | ---: |
| baseline/static | 323.1 | baseline | 336 | 6 |
| middleware_1/static | 458.0 | +41.8% | 440 | 10 |
| middleware_3/static | 551.1 | +70.6% | 512 | 14 |
| guard_1/static | 325.3 | +0.7% | 336 | 6 |
| guard_3/static | 335.0 | +3.7% | 336 | 6 |
| pipe_1/static | 363.3 | +12.4% | 384 | 7 |
| pipe_3/static | 375.7 | +16.3% | 384 | 7 |
| filter_1_success/static | 322.4 | -0.2% | 336 | 6 |
| combined/static | 501.6 | +55.2% | 488 | 11 |
| baseline/reflection | 916.4 | +183.6% | 408 | 10 |
| combined/reflection | 1091.0 | +237.7% | 560 | 15 |

The combined case registers **one middleware, one guard, one identity pipe
and one fallback exception filter**. The filter runs only on the error path;
this benchmark exercises successful requests.

## Findings and engineering follow-up

1. **Middleware is the dominant measured incremental cost**. One
   pass-through middleware adds 134.9 ns/op, 104 B/op and four allocations
   relative to the identical no-middleware route. Three middlewares add
   228.0 ns/op, 176 B/op and eight allocations. Inspect request-time
   `runMiddleware` closures and `Next` invocation recursion; preserve
   short-circuit, error propagation, exactly-once `next`, and scope order.
2. **Guards cost little in the synthetic success path** and do not create
   additional allocations. Small percentage differences may be benchmark
   noise; no strict sub-5% performance claim is warranted.
3. **Pipes add one allocation per operation** when activating the
   context adapter; moving from one to three identity pipes has little
   additional allocation cost.
4. **Exception filters have negligible successful-path cost** in the
   measured synthetic fixture. Error-path latency and allocations still
   require an isolated test, including catch/delegation and 5xx redaction.
5. **Static dispatch remains faster** than reflection under the same
   combined pipeline (501.6 vs 1091.0 ns/op).

## Benchmark interpretation

These results are **not** end-to-end requests per second or p95/p99
network latency and do not represent production throughput. They exclude
transport parsing, TLS, database calls and actual middleware business logic.
A median of five run-level measurements is useful for directional analysis,
not a performance service-level guarantee.

Suggested follow-up: optimize middleware closure allocation on the hot
path, then re-run the same fixture and validate the guard/pipes/filter
behavior with `go test -race`. Compare server-level HTTP benchmarks
and CPU/heap profiles before/after any runtime change.
