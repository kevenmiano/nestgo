# NestGo

**NestGo Compiler v3** is an experimental Go framework that explores NestJS-style decorators through `.mgo` source files. It transpiles annotated code into ordinary Go and generates metadata consumed by a lightweight HTTP runtime.

> **Status:** Experimental proof of concept. Not production-ready. The project does not yet implement all NestJS features.

## Requirements

- Go 1.23 or later
- No third-party runtime dependencies

## Quick start

Clone the repository, then run:

```bash
go run ./cmd/modforge compile ./examples/shop
go test ./...
go vet ./...
```

The compiler reads `.mgo` files and writes generated Go files alongside them. Do not edit generated files manually.

## Decorator syntax

```go
package shop

@Controller("/users")
type UsersController struct {
    Service *UsersService
}

@Get("/")
func (c *UsersController) FindAll() []string {
    return []string{"Ada", "Grace"}
}

@Post("/")
@HttpCode(201)
func (c *UsersController) Create(@Body() dto CreateUserDTO) string {
    return dto.Name
}

type CreateUserDTO struct {
    Name string `json:"name"`
}

@Injectable()
type UsersService struct{}

@Module("users")
type UsersModule struct{}
```

The supported syntax includes:

| Category | Decorators |
| --- | --- |
| Types | `@Controller`, `@Injectable`, `@Module` |
| Routes | `@Get`, `@Post`, `@Put`, `@Patch`, `@Delete`, `@HttpCode` |
| Parameters | `@Body`, `@Param`, `@Query`, `@Headers` |

**Important:** Recognizing a decorator in the compiler does not guarantee full runtime support. For example, the runtime does not currently implement `@Headers` binding.

## Project structure

| Path | Responsibility |
| --- | --- |
| `cmd/modforge/` | Command-line compiler entry point |
| `compiler/` | Decorator lexing, parsing, validation, Go code and metadata generation |
| `metadata/` | Runtime metadata types and registry |
| `runtime/` | HTTP routing, parameter binding and basic dependency injection |
| `examples/shop/` | Example `.mgo` source, generated files and HTTP tests |
| `docs/` | Architecture, development workflow and contribution conventions |

## How it works

1. Write Go-like code with decorators in a `.mgo` file.
2. Run the compiler to remove decorators and produce a standard `_generated.go` file.
3. The compiler generates `zz_modforge_metadata.go` with controller and route metadata.
4. Register the generated package and construct an HTTP handler through `runtime.FromRegistry`.
5. Run and test the resulting application using normal Go tooling.

See [Architecture](docs/ARCHITECTURE.md) for the compilation pipeline and [Development](docs/DEVELOPMENT.md) for contributor workflows.

## Current limitations

- Basic field injection only; no constructor injection or request-scoped providers.
- No guards, pipes, interceptors, middleware abstraction or dynamic modules.
- No Language Server Protocol (LSP) implementation.
- Only a subset of decorator argument forms is supported.
- Parameter and response handling is intentionally minimal.
- The generated code and metadata format are not yet stable APIs.

## License

No license has been declared in this repository. All rights remain with the copyright holder until a license is added.
