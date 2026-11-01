# NestGo P0 runtime verification

The release-readiness gate is tracked in [issue #70](https://github.com/kevenmiano/nestgo/issues/70).
Passing this checklist demonstrates the tested runtime behavior; it does **not**
alone authorize an unqualified production-ready release.

## Required commands

Execute on the exact commit intended for release, with the Go version and
architecture recorded in the verification artifact:

```sh
go test ./... -count=1
go test -race ./... -count=1
go test -race ./runtime ./platform/... -count=3
go vet ./...
go run ./cmd/modforge compile ./examples/shop
git diff --exit-code -- examples/shop
```

The `P0 runtime race and dependency safety` GitHub Actions workflow also
saves the all-package race log, repeated runtime/HTTP-adapter results and
atomic coverage for `runtime` under a 30-day artifact retention period.

## Test isolation and negative scenarios

Runtime tests register metadata in a process-wide registry. Registration is
strict: duplicating a package name must fail. Fixtures which register the
same name on repeated test runs should use `sync.Once` to avoid a false
failure under `go test -count=3`; this does **not** relax runtime duplicate
registration validation.

Supported negative bootstrap scenarios include malformed provider/controller
metadata, unresolved or ambiguous dependencies, invalid module imports and
cycles, unsafe handler return types, invalid route parameter indexes,
duplicate bindings and unsupported decorators. Both fasthttp and net/http
are exercised with actual listeners for routing, 404/405 and shutdown.

A successful race run is evidence for the scenarios exercised, not proof
that all application code is race-free. Explicit panics inside user handlers,
rate limiting, panic recovery, audit logging, shutdown under overload,
and real-world stress remain separate release-gate obligations.

## Performance

Keep profiling artifacts distinct from uninstrumented before/after
benchmarks. See [HTTP benchmarks](BENCHMARKS.md). Do not compare uninstrumented
throughput with CPU/memory-profiled runs or with results from different
runners as if they were controlled A/B experiments.
