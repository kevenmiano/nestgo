---
title: Compiler
---
# Compiler

The NestGo compiler translates decorator-annotated `.mgo` files into ordinary Go code and typed route/module metadata.

```bash
go run ./cmd/nestgo compile ./examples/shop
```

Use the [NestGo CLI guide](../cli/overview.md) for usage, supported commands, flags, and troubleshooting. Compilation produces `*_generated.go` files and `zz_modforge_metadata.go`; generation is deterministic and checked by CI against the committed `shop` example.

## Watch and incremental recompilation

```bash
nestgo compile --watch ./myapp
# Optional polling frequency
nestgo compile ./myapp --watch --poll-interval=500ms
```

The watcher compiles immediately, then fingerprints top-level `.mgo` and handwritten `.go` sources (including same-size edits). After the initial build, unchanged `.mgo` units reuse their parsed decorators and formatted local Go output; edited files are reparsed. Package-wide metadata, SQL/DTO generation, dependency contracts and wiring assertions are still validated each time to avoid cross-file stale results. The cache is process-local, never affects `check`, and is discarded when the watcher exits. Removed source units have their compiler-owned Go artifacts cleaned up. Developer-owned tests remain untouched.

Errors are reported without stopping the watcher. Generated Go outputs are ignored by the change detector to prevent recursive builds. See the [incremental build contract](https://github.com/kevenmiano/nestgo/blob/main/docs/COMPILER.md) for semantics and benchmarks.

## Generated test contracts

Successful compilation also writes compiler-owned `zz_nestgo_wiring_generated_test.go` assertions and developer-editable `zz_nestgo_stubs_test.go` with skipped TODO tests for controllers and providers. Existing tests are preserved; `@Dto()` opts into JSON and validation-related stub templates. The TODO tests do not replace actual unit tests or validation libraries.

See the [CLI guide](../cli/overview.md) for supported flags and [compiler reference](https://github.com/kevenmiano/nestgo/blob/main/docs/COMPILER.md) for details.

## Verify generated artifacts

```bash
nestgo compile ./myapp
nestgo check ./myapp
```

`check` is a read-only consistency gate for compiler-owned Go and metadata outputs. It detects missing or outdated generated files and prints a recompile instruction. Developer-edited test scaffolds are excluded by design. Use it in CI alongside `go test ./...`.

## Structured errors

For integrations and CI, run `nestgo check --format=json ./myapp` or
`nestgo compile --format=json ./myapp`. The CLI emits a single schema-v1
JSON result, with stable error codes for decorators, Go syntax, and generated
artifact drift. Failed checks preserve nonzero exit status. Watch mode
continues to use readable terminal messages.

## Validation and invocation

The compiler validates declared module references, local dependency graphs, interface matching in supported cases and malformed annotations. It generates optional `StaticInvoke` wrappers for conservative, zero-argument, single-result routes. Other method signatures use the reflective runtime path. The direct path is an optimization, not a removal of dynamic execution.

Explicit module imports support qualified Go type references. External type resolution and end-to-end cross-package validation remain areas under development. Do not assume all possible external Go provider types are supported by compiler analysis.

For supported syntax and diagnostics, consult the [compiler reference](https://github.com/kevenmiano/nestgo/blob/main/docs/COMPILER.md).

## Property transformations

Decorate DTO fields with `@Expose`, `@Exclude`, `@Transform` and validators such as `@IsEmail` and `@IsCPF`. The compiler emits typed JSON methods, which work with either HTTP adapter. See [DTO transformers](./dto-transformers.md).
