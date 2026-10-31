<div align="center">

<img src="website/static/img/nestgo-mascot.svg" alt="NestGo mascot" width="150" />

# NestGo

**Structured applications. Native Go.**

*Independent project inspired by NestJS and built with Go; not affiliated with either project.*

[![Go quality gates](https://github.com/kevenmiano/nestgo/actions/workflows/go-quality.yml/badge.svg)](https://github.com/kevenmiano/nestgo/actions/workflows/go-quality.yml)
[![Documentation site](https://github.com/kevenmiano/nestgo/actions/workflows/docs-site.yml/badge.svg)](https://github.com/kevenmiano/nestgo/actions/workflows/docs-site.yml)
[![Go version](https://img.shields.io/badge/Go-1.23%2B-00ADD8?logo=go&logoColor=white)](go.mod)
[![Status](https://img.shields.io/badge/status-experimental-orange)](https://github.com/kevenmiano/nestgo/issues)
[![GitHub Project](https://img.shields.io/badge/roadmap-Project%20%232-ce0038)](https://github.com/users/kevenmiano/projects/2)

[Documentation](website/docs/intro.md) · [Project roadmap](.planning/ROADMAP.md) · [Contributing](CONTRIBUTING.md)

</div>

**NestGo Compiler v3** is an experimental Go source-to-source compiler and HTTP runtime inspired by NestJS decorators. Write annotated `.mgo` files, generate standard Go source and route metadata, and serve requests with Go's `net/http`.

> **Status:** Experimental proof of concept, not production-ready. NestGo is not a drop-in replacement for NestJS.

## Requirements

- Go 1.23 or newer
- No external runtime dependencies

## Getting started

```bash
git clone https://github.com/kevenmiano/nestgo.git
cd nestgo
go run ./cmd/modforge compile ./examples/shop
go test ./...
go vet ./...
```

The compiler accepts a directory or `.mgo` path (compiling its containing directory). It writes `*_generated.go` files and `zz_modforge_metadata.go`. Generated Go code is checked into the example so the tests can run with normal Go tooling.

## Example

From [`examples/shop/users.mgo`](examples/shop/users.mgo):

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

@Get("/{id}")
func (c *UsersController) FindOne(id string) string {
    return id
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

To serve registered routes, import the generated package into your application and create the HTTP handler:

```go
package main

import (
    "log"
    "net/http"

    _ "github.com/kevenmiano/nestgo/examples/shop"
    "github.com/kevenmiano/nestgo/runtime"
)

func main() {
    handler, err := runtime.FromRegistry("shop")
    if err != nil {
        log.Fatal(err)
    }
    log.Fatal(http.ListenAndServe(":8080", handler))
}
```

The example includes tests for `GET /users/`, `GET /users/{id}`, and `POST /users/`.

## Supported decorators

| Area | Decorators |
| --- | --- |
| Types | `@Controller`, `@Injectable`, `@Module` |
| HTTP methods | `@Get`, `@Post`, `@Put`, `@Patch`, `@Delete`, `@HttpCode` |
| Parameters | `@Body`, `@Param`, `@Query` |
| Compiler-only recognition | `@Headers` (**runtime binding not implemented**) |

## Architecture

```text
.mgo files
    |
    v
Decorator lexer + Go AST parser
    |
    v
Compiler validation + metadata generation
    |
    +--> *_generated.go
    +--> zz_modforge_metadata.go
                |
                v
        metadata.Register
                |
                v
        runtime.FromRegistry
                |
                v
          net/http ServeMux
```

| Directory | Responsibility |
| --- | --- |
| `cmd/modforge/` | Compiler command-line entry point |
| `compiler/` | Lexer, AST validation, source and metadata generation |
| `metadata/` | Process-wide metadata registry and descriptors |
| `runtime/` | Routing, request binding, reflection-based handler calls, field injection |
| `examples/shop/` | Sample annotated source, generated Go and HTTP tests |
| `docs/` | Detailed compiler, runtime, architecture and development references |

## Documentation

- [Architecture and design](docs/ARCHITECTURE.md)
- [Compiler reference](docs/COMPILER.md)
- [HTTP runtime and dependency injection](docs/RUNTIME.md)
- [Development guide](docs/DEVELOPMENT.md)
- [Contributing](CONTRIBUTING.md)

## Limitations

NestGo currently provides basic reflection-based field injection, JSON body binding and method routing. It does **not** implement constructor injection, request-scoped providers, guards, pipes, interceptors, dynamic modules, a language server, or full NestJS compatibility. Generated metadata and public APIs may change.

## License

No license has been declared. All rights remain with the copyright holder until a license is added.
