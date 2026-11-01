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

The watcher compiles immediately, then fingerprints the names and contents of top-level `.mgo` sources. It triggers a directory compile only after changes, and `writeIfChanged` avoids rewriting identical outputs. This is **incremental change detection**, not yet selective per-file compilation. Errors are printed and the watcher keeps observing files until cancellation. Generated Go files are ignored to prevent infinite compile loops.

## Generated test contracts

Successful compilation also writes compiler-owned `zz_nestgo_wiring_generated_test.go` assertions and developer-editable `zz_nestgo_stubs_test.go` with skipped TODO tests for controllers and providers. Existing tests are preserved; `@Dto()` opts into JSON and validation-related stub templates. The TODO tests do not replace actual unit tests or validation libraries.

See the [CLI guide](../cli/overview.md) for supported flags and [compiler reference](https://github.com/kevenmiano/nestgo/blob/main/docs/COMPILER.md) for details.

## Verify generated artifacts

```bash
nestgo compile ./myapp
nestgo check ./myapp
```

`check` is a read-only consistency gate for compiler-owned Go and metadata outputs. It detects missing or outdated generated files and prints a recompile instruction. Developer-edited test scaffolds are excluded by design. Use it in CI alongside `go test ./...`.

## Validation and invocation

The compiler validates declared module references, local dependency graphs, interface matching in supported cases and malformed annotations. It generates optional `StaticInvoke` wrappers for conservative, zero-argument, single-result routes. Other method signatures use the reflective runtime path. The direct path is an optimization, not a removal of dynamic execution.

Explicit module imports support qualified Go type references. External type resolution and end-to-end cross-package validation remain areas under development. Do not assume all possible external Go provider types are supported by compiler analysis.

For supported syntax and diagnostics, consult the [compiler reference](https://github.com/kevenmiano/nestgo/blob/main/docs/COMPILER.md).

## Property transformations

DTO structs can use `@Expose`, `@Exclude`, `@Transform` and `@Type` on exported fields. Generated `MarshalJSON` and `UnmarshalJSON` methods apply across both HTTP adapters. Read [DTO transformers](./dto-transformers.md) for examples and custom functions.
