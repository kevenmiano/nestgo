---
title: Compiler
---
# Compiler

The NestGo compiler translates decorator-annotated `.mgo` files into ordinary Go code and typed route/module metadata.

```bash
go run ./cmd/nestgo compile ./examples/shop
```

The legacy `cmd/modforge` entry point remains available. Compilation produces `*_generated.go` files and `zz_modforge_metadata.go`; generation is deterministic and checked by CI against the committed `shop` example.

## Validation and invocation

The compiler validates declared module references, local dependency graphs, interface matching in supported cases and malformed annotations. It generates optional `StaticInvoke` wrappers for conservative, zero-argument, single-result routes. Other method signatures use the reflective runtime path. The direct path is an optimization, not a removal of dynamic execution.

Explicit module imports support qualified Go type references. External type resolution and end-to-end cross-package validation remain areas under development. Do not assume all possible external Go provider types are supported by compiler analysis.

For supported syntax and diagnostics, consult the [compiler reference](https://github.com/kevenmiano/nestgo/blob/main/docs/COMPILER.md).


### Automatic compilation and DTO test stubs

Run `nestgo compile --watch ./myapp` during development. The compiler checks top-level `.mgo` sources and regenerates only on changes, preserving editable test stubs. Use `@Dto()` to opt a Go struct into serialization/deserialization TODO test stubs. A nonempty `validate` struct tag also adds a validation TODO. Existing tests are preserved; TODO stubs are skipped until you add assertions. Watch mode does not execute tests.
