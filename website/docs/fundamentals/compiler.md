---
title: Compiler
---
# Compiler

The NestGo compiler translates decorator-annotated `.mgo` files into ordinary Go code and typed route/module metadata.

```bash
go run ./cmd/nestgo compile ./examples/shop
```

The legacy `cmd/modforge` entry point exists for older workflows, but `nestgo compile` is the documented CLI. See the [command reference](../cli.md) for arguments and troubleshooting. Compilation produces `*_generated.go` files and `zz_modforge_metadata.go`; generation is deterministic and checked by CI against the committed `shop` example.

## Validation and invocation

The compiler validates declared module references, local dependency graphs, interface matching in supported cases and malformed annotations. It generates optional `StaticInvoke` wrappers for conservative, zero-argument, single-result routes. Other method signatures use the reflective runtime path. The direct path is an optimization, not a removal of dynamic execution.

Explicit module imports support qualified Go type references. External type resolution and end-to-end cross-package validation remain areas under development. Do not assume all possible external Go provider types are supported by compiler analysis.

For supported syntax and diagnostics, consult the [compiler reference](https://github.com/kevenmiano/nestgo/blob/main/docs/COMPILER.md).
