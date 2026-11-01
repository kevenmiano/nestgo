# NestGo production-readiness acceptance matrix

Tracking issue: [#70](https://github.com/kevenmiano/nestgo/issues/70).

**Status: experimental / not yet production certified.** A passing PR test suite
does not constitute a release sign-off. All P0 evidence must refer to the exact
immutable release commit and Go toolchain. Record the final tag, SHA, owner,
date, CI run and corresponding artifact before checking off a gate.

## Supported MVP API

- Single application factory: `runtime.New(runtime.With...)` and one lifecycle
  owner, with `Listen`, `Serve`, `Shutdown(ctx)` and `Close()`.
- Transport switch: `runtime.WithAdapter(fasthttp.New())` (default) or
  `runtime.WithAdapter(nethttp.New())`.
- Direct Go routes via `WithRoutes`, optionally combined with generated
  `@Module`/`@Controller`/`@Injectable` metadata via `WithRegistry`.
- Singleton provider and controller **field** DI only when explicitly tagged
  `nestgo:"inject"` (required) or `nestgo:"optional"`.
- HTTP methods, path/query/body binding, JSON responses, explicit status and
  typed error handling for supported signatures.

**Not in MVP:** request-scoped providers, constructor injection, websocket,
GraphQL, persistence connectors, or arbitrary legacy factories. Runtime
`NewApplication`/`NewApp` are not supported; `New` is canonical.

## Acceptance evidence

| Gate | Evidence | Status |
| --- | --- | --- |
| Single runtime and bootstrap | #110, #121, #128 | Implemented and tested |
| Explicit DI and startup validation | #125, #131, #141 | Implemented and tested |
| Handler signature safety and race | #108, #109, #116, #117 | Implemented and tested |
| Generated example is deterministic | `go-quality.yml` generated shop check | CI gate; repeat on release SHA |
| Both adapters use real listeners | `runtime/http_adapter_integration_test.go` | CI gate; expand operational matrix |
| SIGTERM, health, error redaction | #145 | Pending PR verification |
| External consumer using current SHA | #147, source-mode external-consumer workflow | Pending PR verification |
| Tagged consumer through public proxy and checksum DB | #147, release-mode workflow | **Not yet verified for release tag** |
| Race, vet, static checks, module integrity | `go-quality.yml`, `runtime-race-gates.yml` | CI gates; repeat on release SHA |
| Route/bootstrap throughput regressions | `release-scale.yml` and immutable SHA artifacts: 1/10/100/1000 routes/providers, 3 samples, startup, p95/p99, throughput, allocations for both adapters | Measured in CI; acceptable regression thresholds and exact release SHA sign-off pending |
| SIGTERM under in-flight load and timeout | Operational load test and trace | Pending |
| GoReleaser, tags, docs publication | Release workflow evidence | Pending exact tag sign-off |
| Security handling and disclosure | `SECURITY.md`, error-redaction tests | Guidance available; audit pending |

Do not turn the above **pending** cells into accepted claims without their
specific successful workflow runs and artifacts.

## Release checklist

1. Create a candidate commit on `main`; record full SHA, Go version, operating
   system and architecture. No mutable version tags.
2. Execute `go test ./... -count=1`, `go test -race ./... -count=1`,
   `go vet ./...`, gofmt checks, deterministic generated artifacts,
   `go mod verify` and external-consumer **source** mode on the SHA.
3. Verify both HTTP adapters on real TCP listeners, including error redaction,
   route precedence, headers, JSON, body/header limits, startup and SIGTERM.
4. Record controlled 1/10/100/1000 route/provider throughput, p95/p99,
   startup, CPU and heap baselines. State environment and regression thresholds.
5. Create an immutable semver tag pointing to the approved SHA; configure tag
   protection and run published-tag workflows with `GOPROXY=https://proxy.golang.org`
   and `GOSUMDB=sum.golang.org`. Verify the exact version and checksum.
6. Confirm GoReleaser artifacts, checksums, docs publication and the external
   consumer work **without a local replace directive**.
7. Document breaking changes (strict explicit DI), rollback to previous
   immutable tag, exclusions and sign-off. Keep #70 open if any gate is missing.

## Upgrade and rollback notes

The current DI contract intentionally breaks legacy implicit pointer/interface
injection. Add `nestgo:"inject"` to every required field and
`nestgo:"optional"` to optional fields before upgrading. Untagged fields are
ordinary application state and will not be filled automatically. The canonical
application entrypoint is `runtime.New(options...)`.

For a rollback, deploy the previous **verified immutable** tag and its matching
configuration/artifacts. Do not force-move or overwrite a published tag. Verify
external consumers and DB compatibility independently when introducing
non-MVP adapters. Experimental v0 releases may still change APIs before v1.

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
