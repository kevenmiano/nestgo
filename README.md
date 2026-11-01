<p align="center">
  <a href="https://kevenmiano.github.io/nestgo/">
    <img src="website/static/img/nestgo-mascot.svg" width="130" alt="NestGo logo" />
  </a>
</p>

<h1 align="center">NestGo</h1>

<p align="center">
  A modular Go framework for building structured, maintainable server-side applications.
</p>

<p align="center">
  <a href="https://github.com/kevenmiano/nestgo/actions/workflows/go-quality.yml"><img src="https://github.com/kevenmiano/nestgo/actions/workflows/go-quality.yml/badge.svg" alt="Go quality" /></a>
  <a href="https://github.com/kevenmiano/nestgo/actions/workflows/docs-site.yml"><img src="https://github.com/kevenmiano/nestgo/actions/workflows/docs-site.yml/badge.svg" alt="Documentation build" /></a>
  <a href="https://github.com/kevenmiano/nestgo/actions/workflows/go-packages.yml"><img src="https://github.com/kevenmiano/nestgo/actions/workflows/go-packages.yml/badge.svg" alt="Go packages" /></a>
  <img src="https://img.shields.io/badge/Go-1.23%2B-00ADD8?logo=go&logoColor=white" alt="Go 1.23+" />
  <img src="https://img.shields.io/badge/status-experimental-orange" alt="Experimental" />
</p>

<p align="center">
  <a href="https://kevenmiano.github.io/nestgo/">Documentation</a>
  · <a href="https://github.com/kevenmiano/nestgo/tree/main/examples/shop">Example</a>
  · <a href="https://github.com/kevenmiano/nestgo/issues">Issues</a>
  · <a href="https://github.com/kevenmiano/nestgo/discussions">Discussions</a>
</p>

## Description

**NestGo** is an experimental, NestJS-inspired framework that brings a modular application structure to **Go**. It combines controllers, providers, dependency injection and declarative modules with a source-to-source compiler that turns decorated `.mgo` files into standard Go code.

NestGo offers a pluggable HTTP platform API with native **fasthttp** and **net/http** adapters. It also supports optional compiled handler invocation, with reflection as a fallback for other signatures.

> [!IMPORTANT]
> NestGo is an independent project inspired by NestJS. It is under active development, is not affiliated with NestJS and is not yet production-ready.

## Philosophy

Go is well suited to efficient network services, but application architecture is often left to each team. NestGo explores an opinionated **organization model**, while retaining Go's native compiler, type system and ecosystem.

Applications can be organized into focused modules with explicit imports and exports, making dependency relationships visible and easier to reason about.

## Getting started

Install the CLI with Go 1.23 or newer:

```bash
go install github.com/kevenmiano/nestgo/cmd/nestgo@latest
```

In your application's Go module, add the framework and compile your annotated source:

```bash
go get github.com/kevenmiano/nestgo@latest
nestgo compile ./myapp
```

To explore the included application:

```bash
git clone https://github.com/kevenmiano/nestgo.git
cd nestgo
go run ./cmd/nestgo compile ./examples/shop
go test ./...
```

## Example

NestGo's decorators are written in `.mgo` files and compiled into regular Go source:

```go
@Injectable()
type UsersService struct{}

@Controller("/users")
type UsersController struct {
    Service *UsersService
}

@Get("/")
func (c *UsersController) FindAll() []string {
    return []string{"Ada", "Grace"}
}

@Module({
    controllers: [UsersController],
    providers: [UsersService],
})
type UsersModule struct{}
```

Use the generated metadata with the application runtime. For example, after importing your generated package for registration:

```go
app, err := runtime.New(runtime.WithRegistry("shop")) // native fasthttp by default
if err != nil {
    log.Fatal(err)
}
log.Fatal(app.Listen(":8080"))
```

Pass `runtime.WithAdapter(nethttp.New())` to switch to the Go standard-library HTTP engine. Both adapters use the same bootstrap and lifecycle. See [HTTP adapters](docs/HTTP_ADAPTERS.md).

## Features

- **Modular architecture:** declare controllers, providers, module imports and exports.
- **Dependency injection:** resolve dependencies and validate module visibility at bootstrap.
- **Decorator compiler:** generate standard Go files and metadata from `.mgo` sources.
- **HTTP adapters:** use native fasthttp or net/http through a platform interface.
- **Hybrid invocation:** statically invoke eligible handlers while retaining reflection for other cases.
- **Diagnostics:** inspect module graphs and collect startup measurements.

Features are evolving. Constructor injection, request-scoped providers, guards, interceptors and full cross-package compilation support are not yet stable.

## Documentation

Read the [documentation portal](https://kevenmiano.github.io/nestgo/) or browse the [documentation sources](website/docs/). The site is built in CI; GitHub Pages publication currently happens through the separate release workflow.

Additional references: [Compiler](docs/COMPILER.md) · [Runtime](docs/RUNTIME.md) · [Architecture](docs/ARCHITECTURE.md) · [Development](docs/DEVELOPMENT.md).

## Contributing

Contributions, bug reports and design discussions are welcome. See [CONTRIBUTING.md](CONTRIBUTING.md), the [issue tracker](https://github.com/kevenmiano/nestgo/issues) and the [project roadmap](https://github.com/users/kevenmiano/projects/2).

## License

A license has not yet been declared for this repository. See the repository contents for the current licensing status.
