---
title: Architecture
---
# Architecture

NestGo compiles decorator-annotated `.mgo` sources into standard Go source and a metadata registry. At runtime, `runtime.FromRegistry` builds an HTTP handler on top of `net/http`.

```text
.mgo → lexer + Go AST → compiler → generated Go + metadata
                                         ↓
                                 metadata.Register
                                         ↓
                                 runtime.FromRegistry
                                         ↓
                                      net/http
```

The Go runtime has no external dependencies. The documentation website uses Node.js tooling **only for building the website**.

For deeper implementation details, see [the repository architecture reference](https://github.com/kevenmiano/nestgo/blob/main/docs/ARCHITECTURE.md).
