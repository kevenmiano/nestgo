# NestGo Core v1.0 — release acceptance matrix

Tracking gate: [#70](https://github.com/kevenmiano/nestgo/issues/70) · [execution roadmap](../.planning/ROADMAP.md).

**Current classification: experimental, not yet signed off for production.** The matrix below records implementation and remaining evidence, not certification. The current source can evolve while a test executes; only a single immutable candidate SHA with complete passing runs may be approved. Collect the release tag, SHA, toolchain, environment, owner, date, CI URLs and artifacts before marking a release criterion complete.

## Supported Core v1.0 target

- Exactly one factory: `runtime.New(runtime.With...)`, plus `Listen`, `Serve`, `Shutdown(ctx)` and `Close()`; generated metadata is discovered from imported packages or provided via `WithPackages`; `WithRegistry` and legacy factories are not public APIs.
- Native `fasthttp` and `net/http` adapters. Both are supported only after their same-SHA parity and reverse-proxy tests pass.
- Compiler-free `WithRoutes` and optional deterministic `.mgo` metadata compilation for modules, controllers, routes and explicit singleton DI.
- Typed `http.Response[T]`, `http.Result`, JSON request/response, status, headers and safe HTTP errors.
- Minimum pipeline: middleware, guards, validation pipes and exception filters with stable, documented ordering. A draft PR is **not** an implemented guarantee.
- Operational defaults: bounded HTTP reads/headers/body, deadlines, unexpected-error/panic redaction, structured logs, readiness/liveness and graceful in-flight SIGTERM handling.

Excluded from the core support contract: ORM/SQL code generation, config providers beyond simple validated Go configuration, GraphQL, OpenAPI generation, queues, WebSockets, gRPC transport, request/transient-scoped DI, constructor injection, interceptors and advanced DTO strategies. These may ship as optional packages with separate quality gates.

## Release evidence matrix

| Area / evidence | Existing implementation or work | Remaining acceptance | Release gate |
| --- | --- | --- | --- |
| Compiler + DI | Module graph, cross-package tests, explicit field injection and structured errors in core | Golden positive/negative cases; missing/ambiguous/cyclic dependencies; deterministic regeneration on **candidate SHA** | P0 — #4 #10 #35 #36 |
| Lifecycle | PR #157 merged dependency-ordered hooks and partial-bootstrap cleanup; #119 closed | Re-run hook ordering, cancellation, rollback, shutdown concurrency and adapter parity on the exact candidate SHA | P0 evidence — #119 #70 |
| HTTP + typed responses | PR #107 merged; adapters and server limits on main | Real-listener parity on candidate SHA incl. 404/405, JSON, headers/cookies, status/error handling, timeouts and request limits | P0 — #70 |
| Request pipeline | PR #122 merged middleware, guards, validation pipes and exception filters | Prove ordering, 4xx/5xx, fail-closed behavior, direct/codegen/static and reflective parity on the exact candidate SHA, both adapters | P0 bounded evidence — #13 #14 #15 |
| Unexpected errors and panics | Safe 5xx regression coverage already exists | Unified panic/recovery and redaction behavior through reflective/static/direct routes and both adapters; request correlation/logging | P0 — #112 |
| HTTP boundaries and proxy | #113 and #115 closed; PR #140 merged with local Nginx tests | Rerun with actual Nginx/proxy and malicious framing/slow header/resource ceilings on candidate SHA | P0 — #70 |
| Shutdown | Process SIGTERM and in-flight tests merged in PR #145 and #151 | Repeat real TCP SIGTERM and deadline/continuity checks on exact candidate SHA, both adapters; verify no half-writes | P0 evidence — #70 |
| Operations | `logging/slog`, bootstrap diagnostics and optional typed config loading exist | Supply framework/application liveness and readiness contract, draining readiness transition, fail-closed required config and safe correlation-aware logs in a deployable example | P0 bounded — #16 #112 |
| Static/race/security | Quality/race/vulnerability workflows plus merged PR #160 compiler watch-stability regression (50 repeats and 10 race repeats on PR SHA) | Re-run *all* checks on a single main candidate SHA; retain fuzz, dependency provenance and artifact evidence | P0 — #25 #70 |
| Scale and benchmarks | Router/bootstrap comparison, profiling, #150 measurements and #153 startup/summary CI | Repeat 1/10/100/1000 startup, p95/p99, throughput, allocations on exact candidate SHA; approve regression thresholds and decision | P0 — #37 #55 #70 |
| External release | External-consumer source and tagged proxy workflows, GoReleaser and docs exist | External Go project installs **approved immutable tag** from `proxy.golang.org` with `sum.golang.org`, without `replace`; build/sample/CLI/docs releases pass | P0 — #70 |
| Stability and security policy | `SECURITY.md`, upgrade notes, Go semantic release automation | API/version policy, exclusions, rollback, owner sign-off, exact SHA/tag links; do **not** merge release PR #73 until approved | P0 — #70 |

Rows with code merged are **not automatically accepted**; no checkboxes are marked complete here without fresh independent evidence.

## Mandatory final sign-off

1. Record immutable **candidate main SHA**, `go version`, `go env`, runner environment, owner and date.
2. Run `go mod tidy -diff`, `go mod verify`, `govulncheck ./...`, `go test ./... -count=1`, `go test -race ./... -count=1`, `go vet ./...`, gofmt and generated-output/idempotency checks; retain artifacts.
3. Run multi-package external fixture from candidate SHA with successful and rejected modules/DI, malformed decorators, handler signatures and generated code. Verify both compiler-first and `WithRoutes` code-first use.
4. Test real TCP listeners for FastHTTP/net/http, matched-route pipeline, error/panic redaction, 400/403/404/405/413/429/500, JSON body limits, timeouts, headers and graceful shutdown. Repeat proxy-security tests through actual Nginx.
5. Demonstrate readiness before, during and after SIGTERM; active requests complete or hit an explicit deadline, provider cleanup runs once, resources close, no data leaks or hangs.
6. Reproduce startup, p95/p99/throughput and allocations for 1/10/100/1000 providers/routes; archive raw samples, environmental metadata and thresholds, not unsupported production SLAs.
7. Publish only after the release candidate is approved, with **immutable semver tag** and previous-tag rollback plan. Verify full external consumer via `GOPROXY=https://proxy.golang.org`, `GOSUMDB=sum.golang.org`, no local `replace`, published GoReleaser binaries/checksums and docs.
8. Capture links to every green run/artifact and record reviewers' sign-off in #70. If any requirement is not evidenced, leave #70 open and keep Core v1.0 unreleased.

## Public API and rollback

Production users opt into only the tested HTTP core contract. Dynamic/request DI scopes, broad NestJS parity and optional integrations are experimental/not included until independently released. A Go application may supply its own `database/sql`, config library or authentication middleware without NestGo requiring an ORM.

**Breaking change:** DI requires explicit `nestgo:"inject"` / `nestgo:"optional"`; `runtime.New(...Option)` is the only supported factory. Remove old factories and deprecated implicit injection from consumer code before upgrading.

Rollback by deploying a previously **verified immutable tag** with its corresponding configuration and binaries. Never retag a version; verify external DB migration compatibility separately.

References: [semantic releases](SEMANTIC_RELEASES.md), [distribution](DISTRIBUTION.md), [security](../SECURITY.md).

## Reproducing release-scale measurements

The `.github/workflows/release-scale.yml` job runs `BenchmarkReleaseScale`
for 1, 10, 100 and 1,000 routes and singleton providers on the **same** runner,
for both `fasthttp` and `net/http`, using three repetitions:

```sh
GOMAXPROCS=2 go test ./runtime -run '^$' -bench '^BenchmarkReleaseScale$' \
  -benchtime=250ms -count=3 -benchmem -timeout=20m
```

The downloadable `release-scale-<sha>` artifact contains `environment.txt`
(Go version, OS, architecture, CPU and commit SHA), raw `bench.txt`,
machine-readable `summary.json` and a median `summary.md`. CI fails on
missing or inconsistent samples. The `startup-ms` metric includes NestGo
metadata/DI initialization and route registration, but excludes TCP listener
initialization and request warmup. HTTP p95/p99 and throughput include the Go
client, local TCP stack and adapter, so they are **not** isolated handler or
routing costs. The 250 ms sample duration is useful for tracking regressions,
not a high-confidence production load test.

**Release sign-off is still blocked:** collect measurements for the exact
candidate release SHA, compare them to an approved baseline on comparable
hardware, define explicit allowed deviations for p95/p99, startup, throughput
and allocations, then test under production-like concurrency and real
ingress. Do not turn a successful benchmark run into an automatic performance
SLA.
