---
sidebar_position: 1
title: Introduction
slug: /intro
---
# Introduction to NestGo

**NestGo** is an experimental source-to-source Go framework inspired by the decorator ergonomics and modular architecture of NestJS. It is independent of NestJS, not a drop-in replacement, and not yet production-ready.

## Requirements

- Go 1.23 or newer
- Go module dependencies resolved through `go.mod` and `go.sum`
- Node.js 22 only when building the Docusaurus documentation website

## Install NestGo

Import the ordinary Go runtime using `go get github.com/kevenmiano/nestgo@latest`. No NestGo compiler is needed for plain Go handlers. The CLI (`go install github.com/kevenmiano/nestgo/cmd/nestgo@latest`) is optional and only required for `.mgo` decorators. See [installation](./installation.md).

## Quick start: plain Go server

```go
app, err := runtime.New(runtime.WithRoutes(runtime.Route{
    Method: "GET", Path: "/health",
    Handler: func(ctx runtime.Context) error {
        ctx.Status(200)
        return ctx.Write([]byte("OK"))
    },
}))
if err != nil { log.Fatal(err) }
log.Fatal(app.Listen(":8080"))
```

This is a normal imported Go library, similar to creating a gRPC server. Go links the dependency with your application; no precompiled standalone NestGo server executable is necessary.

## Optional compiler and decorators

```bash
git clone https://github.com/kevenmiano/nestgo.git
cd nestgo
go run ./cmd/modforge compile ./examples/shop
go test ./...
```

The compiler reads `.mgo` files and writes ordinary Go files plus metadata for controllers, providers, routes and modules.

```go
@Controller("/users")
type UsersController struct{}

@Get("/")
func (c *UsersController) FindAll() []string {
    return []string{"Ada", "Grace"}
}
```

:::warning Experimental syntax
Decorators are accepted in `.mgo` files by the NestGo compiler, **not** by the standard Go compiler.
:::

## Application runtime

The single factory `runtime.New(runtime.WithRoutes(...))` or `runtime.New(runtime.WithRegistry("shop"))` creates `*runtime.Application` with a native fasthttp adapter by default. Use `runtime.WithAdapter(nethttp.New())` to switch engines. Both engines share DI, route contracts, initialization hooks and explicit `app.Shutdown(ctx)`. Declarative modules support imports, providers, controllers and exports with runtime graph checks.

The compiler can emit direct invocation wrappers for eligible zero-argument handlers, retaining reflective invocation for other signatures. These are experimental features, not evidence of production-scale performance or complete adapter parity.

For implementation details, see [Architecture](./fundamentals/architecture.md), [Providers and modules](./fundamentals/providers.md), and [Compiler](./fundamentals/compiler.md).

## Documentation publishing

Pull requests and changes in `website/**` trigger a documentation **build check**. The hosted site is deployed separately from an eligible published GitHub Release tag by `docs-release.yml`. Merging documentation into `main` alone does not deploy GitHub Pages. Check [deployment instructions](https://github.com/kevenmiano/nestgo/blob/main/docs/DOCS_SITE.md).
