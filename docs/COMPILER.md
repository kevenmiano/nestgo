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

| `@Expose("alias")` | DTO field | Rename or expose a JSON property |
| `@Exclude()`, `@ExcludeInput()`, `@ExcludeOutput()` | DTO field | Input/output JSON visibility |
| `@Transform(Fn)`, `@TransformInput(Fn)`, `@TransformOutput(Fn)` | DTO field | Typed compile-time transformation |
| `@Type(T)` | DTO field | Validate nested field type contract |
| `@IsEmail()`, `@IsCPF()`, `@IsCNPJ()`, `@IsNotEmpty()`, `@IsOptional()` | DTO field | Compiled input validation |
| `@Regex("pattern")`, `@Matches("pattern")`, `@MinLength(n)`, `@MaxLength(n)`, `@Length(min,max)` | DTO field | String validation |
| `@Min(n)`, `@Max(n)`, `@Validate(CustomFn)` | DTO field | Numeric and custom validation |

See [DTO property transformers](DTO_TRANSFORMERS.md) and [DTO validation](DTO_VALIDATION.md). DTOs with field annotations also emit `*_transformers_generated.go`.

Declaration annotations must occupy their own line. Route decorators must be attached to methods with a receiver whose controller type is declared in the same `.mgo` file. Only a single HTTP method decorator is permitted per handler.

The compiler accepts only a subset of annotation expressions. Controller, module and route arguments are string literals; `@HttpCode` takes a numeric HTTP status.

## Compilation stages

1. `lexDecorators` scans source while skipping quoted strings and comments.
2. Decorator tokens are replaced with spaces, preserving source offsets and line numbers.
3. The Go parser validates the remaining source and discovers decorated declarations.
4. `Generate` produces metadata describing controllers, providers, modules, routes and parameter bindings.
5. `CompileDir` formats and writes generated Go files and the versioned module export contract.

The compiler reports invalid annotations, dangling decorators, unsupported placements, duplicate method decorators, invalid status codes and mixed package declarations.

## Generated files

The generated `*_generated.go` files contain the original Go code with annotations removed. `zz_modforge_metadata.go` contains an `init()` function that calls `metadata.Register` for the package.

The compiler also writes `zz_nestgo_modules.json` with schema version `1` and the modules' explicitly declared `providers` and `exports`. Commit this generated file together with the generated Go files when distributing an imported NestGo module. When the original `.mgo` is absent, the importing compiler checks this contract before accepting concrete cross-package DI. If neither `.mgo` nor the contract exists, compilation fails closed. The runtime continues to enforce module visibility. The contract does not replace Go type checking, and it does not yet support external interface-provider visibility checks.

The generated Go files should be inspected in code review. They are **not** dynamically interpreted at runtime.

## Automatic test scaffolding from compiled wiring

Every successful `nestgo compile <directory>` (or `modforge compile`) also produces two Go test files in the package:

- `zz_nestgo_wiring_generated_test.go` is **compiler-owned**. It is regenerated only when the compiled metadata changes. The test `TestNestGoCompiledWiring` verifies that the package registration contains the expected modules, providers, controllers, prefixes, and route contracts. It does not open network listeners, initialize databases, or construct application instances.
- `zz_nestgo_stubs_test.go` is **developer-owned**. It starts with one `Test<Type>` for each `@Injectable` or `@Controller` and one `Test<Controller>_<Method>` for each route method. Comments list fields found on each decorated type. These tests explicitly call `t.Skip("TODO: ...")` until you implement assertions; they are not claimed as passing behavioral tests.

The compiler never rewrites existing editable stubs. When a new controller, provider, or handler appears, it appends only the missing test functions. A test function already defined in any `*_test.go` file takes precedence and is not duplicated. Recompilation preserves developer edits verbatim. Removing a handler does **not** delete its previously created test; clean up obsolete tests yourself.

```bash
nestgo compile ./myapp
go test ./myapp
```

Example starter for `UsersController`:

```go
func TestUsersController_FindAll(t *testing.T) {
    // Wiring field: Service *UsersService
    subject := new(UsersController)
    _ = subject
    t.Skip("TODO: add assertions for TestUsersController_FindAll")
}
```


### Watch mode

Run `nestgo compile --watch ./myapp` (or `nestgo compile ./myapp -w`) to compile immediately and recompile whenever a top-level `.mgo` file is added, renamed, removed or edited. The watcher checks source contents every 250 ms by default (configurable with `--poll-interval=500ms`), including same-size edits; it intentionally ignores generated outputs to avoid self-triggering loops. Source diagnostics are printed but the watcher keeps running. Ctrl+C stops it. Watch mode does **not** automatically run `go test`.

### DTO scaffolding

Use `@Dto()` to opt in to DTO-specific test stubs:

```go
@Dto()
type CreateUserDTO struct {
    Name string // json:"name" validate:"required" (use a Go struct tag in your code)
}
```

The compiler creates `TestCreateUserDTO_Serialization` and `TestCreateUserDTO_Deserialization` TODOs. It creates `TestCreateUserDTO_Validation` only when at least one DTO field has a nonempty `validate` struct tag. Plain undecorated Go structs do not receive DTO stubs.

A `validate` tag is a contract hint: it does not enable validation in the runtime. Generated tests explicitly use `t.Skip`, so add real fixtures, validators and assertions before treating them as behavioral coverage. DTO property-transformer code is being built separately in PR #132; specialized transformation scaffolding is not yet integrated.

**Boundary:** The generated assertion checks *compiled registry consistency*, not full runtime dependency construction or correct application behavior. The editable unit stubs do not automatically create mocks, start servers, or execute HTTP requests. Add dependency fakes and assertions as needed. Test generation adds no cost to a production binary because all generated tests live in `*_test.go` files.

## Read-only compiler check

```bash
nestgo compile ./myapp
nestgo check ./myapp
```

`nestgo check` runs the same source parsing, metadata generation, SQL repository generation and wiring-test generation as a normal compile, but compares compiler-owned artifacts **in memory** against existing files. It never writes to disk. Missing or changed output files result in `compiler.ErrOutdated` (nonzero CLI exit), with deterministic sorted file paths and a suggestion to recompile.

Developer-owned `zz_nestgo_stubs_test.go` is intentionally excluded from equality checks: the compiler must not overwrite handwritten assertions or insist on a specific test body. A successful check verifies compiler-owned outputs, **not** behavioral correctness, and does not run `go test`. SQL dialect defaults match `CompileDir`; dedicated CLI dialect flags are not yet available.

## Structured compiler diagnostics (JSON schema v1)

`nestgo compile` and `nestgo check` accept `--format=json` for single-shot commands.
The default `--format=text` remains unchanged. JSON produces **exactly one JSON
object on stdout**, even when validation fails; the process still exits with a
nonzero code on failure and does not print a duplicate error message to stderr.
Cobra argument/unknown flag parsing failures occur before command execution
and still use standard textual usage errors.

```bash
nestgo compile --format=json ./myapp
nestgo check --format=json ./myapp
```

Example: a missing compiler-owned artifact.

```json
{
  "schema_version": 1,
  "command": "check",
  "ok": false,
  "diagnostics": [
    {
      "code": "NG2001",
      "severity": "error",
      "message": "generated artifact missing",
      "file": "myapp/zz_modforge_metadata.go"
    }
  ]
}
```

The `diagnostics` field is **always an array**, including on success. Paths
use the directory/file spelling passed to the CLI (not necessarily absolute).
Go source diagnostics contain `file`, 1-based `line` and, for parser errors,
1-based `column` when available. Absent coordinates are omitted rather than guessed.
Generated artifacts in `check` are ordered by file path.

| Code | Meaning |
| --- | --- |
| `NG1001` | NestGo decorator/semantic compiler diagnostic |
| `NG1002` | Go parser syntax diagnostic |
| `NG2001` | Compiler-owned artifact missing |
| `NG2002` | Compiler-owned artifact outdated |
| `NG2003` | Compiler-owned artifact obsolete |
| `NG9000` | Other error (I/O, arguments or currently unclassified error) |

`compiler.OutdatedError` also exposes typed `Artifacts` with `Path` and
`Reason` fields and preserves `errors.Is(err, compiler.ErrOutdated)`.
This is a v1 CLI **output contract**, not a declaration that every compiler
error has granular semantic classifications. `--format=json` is deliberately
rejected alongside `--watch`: watch remains a textual stream, not JSON/NDJSON.

## Known limitations

- No recursive directory compilation.
- No full source mapping or language server integration.
- No decorator support on arbitrary expressions or non-struct types.
- No cross-file controller receiver lookup for decorated methods.
- No complete semantic validation of every handler signature.
- No stable generated metadata schema.

See [Runtime](RUNTIME.md) for supported request and response behavior.
