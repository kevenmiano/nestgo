---
title: Architecture
---
# Architecture

NestGo compiles decorator-annotated `.mgo` source into standard Go files and statically generated metadata. Its runtime resolves module visibility and dependency injection at bootstrap.

```text
.mgo → decorator lexer + Go AST + typed checks
                     ↓
           generated Go + metadata
                     ↓
               metadata.Register
                     ↓
       runtime.FromRegistry (http.Handler)
                  OR
       runtime.NewApplication (Adapter)
                  ↓
       fasthttp (default) / net/http
```

## HTTP APIs

- `runtime.New` and `runtime.FromRegistry` retain standard-library `http.Handler` compatibility.
- `runtime.NewApplication` accepts an `ApplicationOptions{Adapter: ...}`; without an adapter, native fasthttp is selected.
- `httpadapter.Adapter` separates route registration, listening and shutdown from the transport.
- The new adapter route-dispatch path has its own implementation. Do not assume complete feature or performance parity with the legacy handler without verification.

## Dependency injection and compilation

Explicit module contracts define imports, providers, controllers and exports. The runtime enforces declared visibility, validates module graphs and discovers imported registered packages for `FromRegistry`. The compiler can generate optional direct invokers for eligible zero-argument handlers; all other handlers use reflection.

The framework uses normal Go modules. Runtime adapters may bring external Go dependencies; Node.js is needed only for building the documentation site.

Read the [architecture reference](https://github.com/kevenmiano/nestgo/blob/main/docs/ARCHITECTURE.md) and [HTTP adapters guide](https://github.com/kevenmiano/nestgo/blob/main/docs/HTTP_ADAPTERS.md).
