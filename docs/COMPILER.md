# Compiler reference

NestGo Compiler v3 translates Go-like `.mgo` source into regular Go files and a package metadata registry.

## Command-line interface

```bash
go run ./cmd/modforge compile ./examples/shop
```

The command also accepts a `.mgo` file path, but it compiles **the entire containing directory**. The compiler searches only for `*.mgo` files directly inside that directory, not recursively.

For every input `name.mgo`, the compiler writes `name_generated.go`. It also writes one `zz_modforge_metadata.go` for the directory. Output files are rewritten only when their content changes. All source files in the directory must declare the same Go package.

## Supported annotations

| Annotation | Placement | Purpose |
| --- | --- | --- |
| `@Controller("/prefix")` | Struct declaration | Defines an HTTP controller and its URL prefix |
| `@Injectable()` | Struct declaration | Registers a provider |
| `@Module("name")` | Struct declaration | Adds module metadata |
| `@Get("/path")` | Controller method | Registers a GET route |
| `@Post("/path")` | Controller method | Registers a POST route |
| `@Put("/path")` | Controller method | Registers a PUT route |
| `@Patch("/path")` | Controller method | Registers a PATCH route |
| `@Delete("/path")` | Controller method | Registers a DELETE route |
| `@HttpCode(201)` | Controller method | Overrides the default response status |
| `@Body()` | Method parameter | Decodes a JSON request body |
| `@Param("id")` | Method parameter | Binds a string URL path variable |
| `@Query("page")` | Method parameter | Binds a string query value |
| `@Headers("name")` | Method parameter | Recognized by the compiler; **not supported by the runtime** |

Declaration annotations must occupy their own line. Route decorators must be attached to methods with a receiver whose controller type is declared in the same `.mgo` file. Only a single HTTP method decorator is permitted per handler.

The compiler accepts only a subset of annotation expressions. Controller, module and route arguments are string literals; `@HttpCode` takes a numeric HTTP status.

## Compilation stages

1. `lexDecorators` scans source while skipping quoted strings and comments.
2. Decorator tokens are replaced with spaces, preserving source offsets and line numbers.
3. The Go parser validates the remaining source and discovers decorated declarations.
4. `Generate` produces metadata describing controllers, providers, modules, routes and parameter bindings.
5. `CompileDir` formats and writes generated Go files.

The compiler reports invalid annotations, dangling decorators, unsupported placements, duplicate method decorators, invalid status codes and mixed package declarations.

## Generated files

The generated `*_generated.go` files contain the original Go code with annotations removed. `zz_modforge_metadata.go` contains an `init()` function that calls `metadata.Register` for the package.

Both files are Go source code and should be inspected in code review. They are **not** dynamically interpreted at runtime.

## Known limitations

- No recursive directory compilation.
- No full source mapping or language server integration.
- No decorator support on arbitrary expressions or non-struct types.
- No cross-file controller receiver lookup for decorated methods.
- No complete semantic validation of every handler signature.
- No stable generated metadata schema.

See [Runtime](RUNTIME.md) for supported request and response behavior.
