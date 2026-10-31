# Architecture

NestGo is an experimental source-to-source compiler and HTTP runtime written in Go.

## Design goals

- Preserve Go as the execution language.
- Keep the generated output inspectable and compatible with standard Go tooling.
- Separate compilation from request handling.
- Minimize runtime dependencies and magic.
- Keep the compiler and runtime modular.

## Compilation pipeline

```text
.mgo source
    |
    v
Decorator lexer (compiler/lexer.go)
    |
    v
Go parser and decorator validation (compiler/compiler.go)
    |
    +--> *_generated.go
    |
    +--> zz_modforge_metadata.go
                  |
                  v
         metadata.Registry
                  |
                  v
         runtime.FromRegistry
                  |
                  v
           net/http Handler
```

The lexer removes supported decorators while preserving offsets so the standard `go/parser` package can parse the remaining Go syntax. The compiler then inspects declarations, associates annotations with types and methods, and emits metadata.

## Components

### Compiler

The compiler reads `.mgo` files, validates supported annotations, generates plain Go source, and writes a package-level metadata registry. Generated files should be treated as build artifacts.

### Metadata

The `metadata` package contains the controller, provider, module, route, and parameter descriptors used by the runtime. Registration happens during generated package initialization.

### Runtime

The runtime creates a `net/http` handler using Go 1.23 `ServeMux`. It instantiates controller and provider structs, injects compatible exported fields, binds selected request parameters, invokes handlers with reflection, and serializes JSON responses.

## Limitations and trade-offs

- Reflection simplifies the proof of concept but does not offer compile-time dependency validation.
- Provider injection is basic and not a full dependency-injection container.
- The current compiler does not implement a full language grammar or robust source mapping.
- Only some parameter decorators have corresponding runtime bindings.
- Metadata registration is global, so package names must be unique.
- Compiler output and public interfaces may change between revisions.

## Next milestones

1. Add strict decorator grammar and semantic validation.
2. Improve error reporting and source positions.
3. Add deterministic code generation and golden-file tests.
4. Define module and provider lifecycle semantics.
5. Introduce CI and integration tests across supported Go versions.
6. Evaluate an LSP after the language syntax stabilizes.
