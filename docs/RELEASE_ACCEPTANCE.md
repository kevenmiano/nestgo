# NestGo Core v1.0 — release acceptance matrix

Tracking gate: [#70](https://github.com/kevenmiano/nestgo/issues/70) · [execution roadmap](../.planning/ROADMAP.md).

**Current classification: experimental, not yet signed off for production.** The matrix below records implementation and remaining evidence, not certification. The current source can evolve while a test executes; only a single immutable candidate SHA with complete passing runs may be approved. Collect the release tag, SHA, toolchain, environment, owner, date, CI URLs and artifacts before marking a release criterion complete.

## Supported Core v1.0 target

- Exactly one factory: `runtime.New(runtime.With...)`, plus `Listen`, `Serve`, `Shutdown(ctx)` and `Close()`; no legacy factories.
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
| Lifecycle | Hooks and partial rollback exist | Dependency-topological init/teardown, cancellation, idempotent concurrent shutdown; #119 regression suite | P0 — #119 |
| HTTP + typed responses | PR #107 merged; adapters and server limits on main | Real-listener parity on candidate SHA incl. 404/405, JSON, headers/cookies, status/error handling, timeouts and request limits | P0 — #70 |
| Request pipeline | PR #122 is draft | Middleware/guards/validation pipes/filters, reject/transform cases, ordering, fail-closed defaults, code-first/generation, both adapters | P0 bounded — #13 #14 #15 |
| Unexpected errors and panics | Safe 5xx regression coverage already exists | Unified panic/recovery and redaction behavior through reflective/static/direct routes and both adapters; request correlation/logging | P0 — #112 |
| HTTP boundaries and proxy | #113 and #115 closed; PR #140 merged with local Nginx tests | Rerun with actual Nginx/proxy and malicious framing/slow header/resource ceilings on candidate SHA | P0 — #70 |
| Shutdown | Process SIGTERM smoke from PR #145 merged | PR #151 in-flight request completes or fails only by explicit deadline, no half-writes; both adapters | P0 — #151 |
| Operations | `logging/slog` and bootstrap diagnostics available | Health/live and health/ready, readiness flips before draining, config validation, safe request logs; external runnable example | P0 bounded — #16 |
| Static/race/security | `go-quality.yml` and `runtime-race-gates.yml` include tests, race, govulncheck, vet, formatting and module verification | All checks pass on candidate SHA; fuzz of metadata/parsing; no known reproducible panic, race or vulnerable supported dependency | P0 — #25 #70 |
| Scale and benchmarks | Router/bootstrap comparison, profiling, release-scale fixtures | Record reproducible 1/10/100/1000 provider/route samples, p95/p99, allocs, runner/toolchain, thresholds and regression decision | P0 — #37 #55 #70 |
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
