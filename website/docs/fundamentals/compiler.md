---
title: Compiler
---
# Compiler

The `modforge` CLI translates NestGo `.mgo` files into ordinary Go code.

```bash
go run ./cmd/modforge compile ./examples/shop
```

Output includes `*_generated.go` files and `zz_modforge_metadata.go`. Re-running generation should produce deterministic output; generated files can be checked by normal Go tests.

Read the [compiler reference](https://github.com/kevenmiano/nestgo/blob/main/docs/COMPILER.md) for supported syntax, diagnostics and restrictions.
