# NestGo roadmap — production-ready HTTP core

**Source of truth:** [Issues](https://github.com/kevenmiano/nestgo/issues) · [Project #2](https://github.com/users/kevenmiano/projects/2) · [Release gate #70](https://github.com/kevenmiano/nestgo/issues/70) · [Acceptance evidence](../docs/RELEASE_ACCEPTANCE.md)

## Release contract: NestGo Core v1.0

**Goal:** a small, reliable Go HTTP framework with a deterministic optional `.mgo` compiler, well-defined module/DI boundaries and a safe request lifecycle. This is a **release target**, not a production-ready claim. The public v1.0 release must not be approved until all gates in #70 pass on the exact candidate SHA.

Supported core surface (subject to final acceptance):

- One idiomatic entry point: `runtime.New(options ...Option)`; `app.Listen`, `app.Serve`, `app.Shutdown(ctx)` and `app.Close()`. No legacy runtime factories.
- Both `fasthttp` (default) and `net/http` via `WithAdapter`, with explicit parity/limitations.
- `WithRoutes` for ordinary Go HTTP handlers; optional generated `@Module`, `@Controller`, providers and imported/exported module metadata using automatic discovery from compiled-package imports or explicit `WithPackages`.
- Singleton **explicit** field injection (`nestgo:"inject"`, `nestgo:"optional"`) and fail-closed, actionable DI/route errors; no implicit injection.
- JSON body/query/path/header binding, `http.Response[T]`, `http.Result`, explicit status/headers and typed HTTP errors.
- A **bounded** NestJS-inspired request pipeline: middleware, guards, parameter/body validation pipes and exception filters, with documented order and scopes; do not promise full NestJS behavior.
- Operational minimum: limits/timeouts, panic recovery, safe error redaction, structured logs, liveness/readiness, SIGTERM/in-flight graceful shutdown and reproducible release verification.

**Not part of Core v1.0:** generated SQL repositories/ORM/migrations, an opinionated configuration provider, OpenAPI/Swagger generation, caching, queues, scheduling, events, gRPC/microservice transports, WebSocket gateways, GraphQL, constructor DI, request/transient scopes, interceptors, advanced DTO groups/transformer factories, full LSP/watch tooling and broad NestJS feature parity.

These are possible optional packages or application-level integrations, **not dependencies of the HTTP core release**. A package being optional does not make an incomplete or unsafe implementation production-supported: SQL/persistence work (#80, #82, #84, #87-#94) must pass its own security and integration gates before release.

## Execution order — remaining minimum gates

| Order | Gate | Priority | Track | Acceptance and current distinction |
| --- | --- | --- | --- | --- |
| 1 | Dependency-aware lifecycle | P0 | [#119](https://github.com/kevenmiano/nestgo/issues/119) | Implemented in merged PR #157 and closed #119; re-run topology, rollback, cancellation and concurrency regression on final candidate SHA (both adapters) |
| 2 | Request pipeline and validation | P0 bounded slice | [PR #122](https://github.com/kevenmiano/nestgo/pull/122), [#13](https://github.com/kevenmiano/nestgo/issues/13), [#14](https://github.com/kevenmiano/nestgo/issues/14), [#15](https://github.com/kevenmiano/nestgo/issues/15) | Implemented in merged PR #122; verify global/controller/route ordering, guard rejection, invalid DTO 4xx, filters, typed/static/reflected and direct route parity on candidate SHA; defer interceptors and request scopes |
| 3 | Request/runtime security | P0 | [#112](https://github.com/kevenmiano/nestgo/issues/112), [#151](https://github.com/kevenmiano/nestgo/pull/151) | PR #151 and proxy tests #140 already merged; close #112 only after all static/reflected/direct/adapter 500s and panics redact sensitive errors. Repeat in-flight SIGTERM and actual Nginx tests on final SHA |
| 4 | Minimum operational contract | P0 bounded slice | [#16](https://github.com/kevenmiano/nestgo/issues/16), [#50](https://github.com/kevenmiano/nestgo/issues/50) | Tested configuration validation at application bootstrap, liveness/readiness (including draining state), correlation IDs and safe structured request/error logs; allow plain Go integrations |
| 5 | Release verification and sign-off | P0 | [#4](https://github.com/kevenmiano/nestgo/issues/4), [#10](https://github.com/kevenmiano/nestgo/issues/10), [#25](https://github.com/kevenmiano/nestgo/issues/25), [#35](https://github.com/kevenmiano/nestgo/issues/35), [#36](https://github.com/kevenmiano/nestgo/issues/36), [#37](https://github.com/kevenmiano/nestgo/issues/37), [#55](https://github.com/kevenmiano/nestgo/issues/55), [#70](https://github.com/kevenmiano/nestgo/issues/70) | Evidence on the candidate SHA: deterministic cross-package compiler E2E, metadata negatives, race, fuzz, Go/vuln/mod integrity, real listeners, proxy, benchmarks, external consumer via immutable tag/checksum, docs/release artifacts |

Execution order is a **release-blocking dependency chain**, not a requirement to resolve every historical unchecked bullet in each broad issue. Close an issue only when its actual acceptance criteria are met; otherwise retain its deferred scope explicitly.

## Current implementation versus remaining evidence

- **Present on main:** single `runtime.New` API, strict explicit DI, module graph checks, response/error helpers, native adapters, bounded server limits, deterministic/diagnostic workflows, indexed FastHTTP router, structured logging, external-consumer and release-scale workflows.
- **Already merged but requiring release-SHA evidence:** dependency/toolchain hardening (#115), bounded HTTP limits (#113), Nginx proxy security tests (PR #140), process-level and in-flight SIGTERM (PRs #145/#151), dependency-ordered lifecycle (PR #157), request pipeline (PR #122), and watch-test stabilization with repeated/race CI (PR #160). Reverify without recreating merged work.
- **Open work:** error/panic response and correlation coverage (#112); minimum liveness/readiness, draining and validated startup configuration (#16); exact-candidate-SHA validation for merged lifecycle #157, pipeline #122 and in-flight termination #151; final independent sign-off (#70).
- **Performance/developer experience:** #97 and broad diagnostic/tooling work do **not** block the HTTP core by default. Promote a measured regression to P0 only when the supported workload misses a documented acceptance threshold. Review stale #98 against the indexed router already on main.
- **Version policy:** an open release-please PR is **not** approval to publish 1.0.0. PR #73 was closed without merge; the newly generated release PR #162 must also remain unmerged until independent sign-off in #70. Never move immutable published Go tags or checksums.

## Architecture tracks after Core v1.0

| Track | Planned contract | Priority |
| --- | --- | --- |
| Core extensions | advanced scopes/tokens/interceptors, DTO serialization strategies, API versioning | P1 |
| Optional configuration | `config` (Viper as a replaceable adapter), typed validation and secret redaction; keep standard-library options | P1 |
| Optional persistence | `database`/SQL adapters, named connections, transactions and generated repositories after the SQL correctness gates | P1 (separate release gate) |
| API/documentation | OpenAPI generator, CLI improvements, LSP and developer experience | P1/P2 |
| Additional transports | events, queues, cache, gRPC/microservices, WebSocket, GraphQL | P2 |

The split is a **support and release boundary** first. Separate Go modules/packages only where independently versioned APIs or dependencies justify the maintenance cost; never introduce a dependency on optional modules into the HTTP core.

## Production gate

See [Release acceptance](../docs/RELEASE_ACCEPTANCE.md) and [#70](https://github.com/kevenmiano/nestgo/issues/70). Every P0 item must have a test, an exact SHA, CI/artifact link, owner and recorded decision. Release sign-off requires a reproducible public-Go-proxy install with checksums enabled and no local `replace` directives.

## Priority semantics

- **P0:** blocks declaring the supported HTTP core production-ready.
- **P1:** valuable but independent from core release; no silent promotion to a P0 requirement.
- **P2:** future integrations and exploratory capabilities.

Follow **Discuss → Plan → Execute → Verify → Review → Ship**. Retain M001–M006 as the Project taxonomy rather than constructing separate duplicate roadmaps.
