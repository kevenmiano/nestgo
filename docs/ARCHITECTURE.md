# Architecture

NestGo Compiler v3 separates **build-time code generation** from **runtime HTTP request handling**. It is a Go-first experimental framework: decorators are parsed before compilation and are not interpreted at request time.

## Design principles

- Emit ordinary, inspectable Go code.
- Keep the compiler independent of the HTTP runtime.
- Use standard Go parsing, formatting and HTTP primitives.
- Store only declarative controller, provider, module and route metadata.
- Prefer explicit limitations to undocumented behavior.

## System overview

```text
Developer                         Build time                          Runtime
---------                         ----------                          -------
*.mgo files ──────────────> lexDecorators + go/parser
                                     |
                              AST declarations
                                     |
                           semantic checks + Generate
                                     |
                        +------------+-------------+
                        |                          |
                  *_generated.go          zz_modforge_metadata.go
                        |                          |
                        +------ Go compilation ----+
                                                   |
                                         generated init() function
                                                   |
                                            metadata.Register
                                                   |
                                            runtime.New(runtime.WithRegistry("shop"))
                                                   |
                                    provider/controller construction
                                                   |
                                            httpadapter.Adapter (fasthttp default)
                                                   |
                                           HTTP request handler
```

## Compiler subsystem

The `compiler` package contains `lexDecorators`, `Parse`, `Generate` and `CompileDir`.

1. **Lexing:** Skip comments and quoted strings. Replace decorator bytes with spaces while preserving byte offsets and line numbers.
2. **Parsing:** Parse the resulting source with `go/parser` and inspect `go/ast` declarations.
3. **Validation:** Reject unsupported placements, unknown or dangling annotations, mixed packages, invalid HTTP status values and multiple method decorators.
4. **Generation:** Produce ordinary Go source and one package-level metadata registration file.
5. **Writing:** Format generated files and avoid rewriting unchanged output.

The CLI lives at `cmd/modforge/main.go`. It compiles only the selected directory, not an entire repository recursively.

See [Compiler reference](COMPILER.md) for syntax and constraints.

## Metadata subsystem

The `metadata` package defines:

- `Package`: controllers, providers and modules.
- `Controller`: Go type, URL prefix and routes.
- `Route`: HTTP method, path, handler name, status and parameter bindings.
- `Provider`: provider type and declared scope.
- `Module`: module type and name.
- `Parameter`: method argument index, decorator kind and binding name.

Generated `init()` functions call `metadata.Register(packageName, packageMetadata)`. A mutex protects the global registry. Duplicate package names cause a panic; `Lookup` and `All` expose registered packages. This is **registration metadata**, not a complete module dependency graph.

## Runtime subsystem

The `runtime` package exposes **one** constructor: `New(options ...Option) (*Application, error)`. Use `WithRegistry(names...)` or `WithPackages(packages...)` to select metadata, and `WithAdapter` to select the HTTP engine.

During application construction, the runtime:

1. Instantiates provider structs.
2. Injects registered providers into exported pointer or interface fields.
3. Instantiates controller structs and injects their dependencies.
4. Validates route contracts and chooses generated invokers or reflection fallback.
5. Registers handlers with the selected HTTP adapter and runs initialization hooks.

At request time, it binds JSON bodies, string path parameters and string query parameters; calls the reflected handler; and writes a JSON response. POST defaults to HTTP 201, other methods to 200, unless `@HttpCode` overrides the status.

See [Runtime reference](RUNTIME.md) for signatures, errors, and injection constraints.

## Example end-to-end flow

`examples/shop/users.mgo` defines a `UsersController`, `UsersService` and `UsersModule`. Compilation produces `users_generated.go` and `zz_modforge_metadata.go`. Importing the package registers its metadata. Calling `runtime.New(runtime.WithRegistry("shop"))` creates routes for:

| Method | Pattern | Handler |
| --- | --- | --- |
| GET | `/users/` | `FindAll` |
| GET | `/users/{id}` | `FindOne` |
| POST | `/users/` | `Create` |

These routes are exercised by `examples/shop/runtime_test.go`.

## Boundaries and limitations

- Decorator parsing is not a full new-language grammar.
- Generated metadata uses reflection and has no stable compatibility guarantee.
- Provider injection does not implement constructor injection or request scopes; lifecycle hooks exist on applications.
- `@Headers` is recognized by the compiler but not handled by the runtime.
- The runtime does not provide guards, validation pipes, interceptors or authentication.
- Global metadata registration requires unique package names.
- Build-time compilation is explicit; there is no file watcher or LSP.

## Future directions

Improve semantic diagnostics and golden-file testing, stabilize generated APIs, define module/provider lifecycle behavior, expand runtime parameter binding and introduce CI coverage before considering production use.
