# NestGo Gatling E2E (GSD)

This is a real HTTP loopback smoke/load test, not a compiler benchmark. It runs only against the local fixture, not public infrastructure.

## Reproduce

From repository root:

```sh
go build -o /tmp/nestgo-perf-server ./cmd/perfserver
/tmp/nestgo-perf-server -mode=nestgo &
cd performance/gatling
npm install
npm run test:load
```

Environment variables: `NESTGO_BASE_URL`, `NESTGO_RATE` (arrival rate), `NESTGO_WARMUP`, `NESTGO_DURATION`. Reports: `target/gatling/`.

## Experiment design

GSD: Discuss → Plan → Execute → Verify → Review → Ship.

This first smoke scenario covers GET /users/ on 10 requests/s at steady state and checks HTTP 200 and a non-empty body. Follow-up work must run **equivalent** NestGo and net/http handlers, with identical payload/JSON serialization, route mix, connection setup, and warmup. Then measure saturation rates and record p50/p95/p99, throughput, errors, GC, CPU, RSS, and full environment information. Run independent iterations with baseline and candidate on equal hardware; a shared CI runner cannot support reliable absolute latency performance gates.

The local `nethttp` mode is only a control candidate and is **not** yet a fully equivalent HTTP workload for all routes. Do not infer scalable microservice readiness from the smoke test alone.
