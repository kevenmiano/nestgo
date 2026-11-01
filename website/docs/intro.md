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

Install the compiler CLI with `go install github.com/kevenmiano/nestgo/cmd/nestgo@latest`. In your application module, use `go get github.com/kevenmiano/nestgo@latest` to add the runtime dependency. See [installation](./installation.md).

## Quick start

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

## Current runtime APIs

The legacy `runtime.New` and `runtime.FromRegistry` return `http.Handler`. The pluggable `runtime.NewApplication` selects a native fasthttp adapter by default or an explicitly supplied `httpadapter.Adapter`, including the net/http adapter. Declarative modules support imports, providers, controllers and exports with runtime graph checks.

The compiler can emit direct invocation wrappers for eligible zero-argument handlers, retaining reflective invocation for other signatures. These are experimental features, not evidence of production-scale performance or complete adapter parity.

For implementation details, see [Architecture](./fundamentals/architecture.md), [Providers and modules](./fundamentals/providers.md), and [Compiler](./fundamentals/compiler.md).

## Documentation publishing

Pull requests and changes in `website/**` trigger a documentation **build check**. The hosted site is deployed separately from an eligible published GitHub Release tag by `docs-release.yml`. Merging documentation into `main` alone does not deploy GitHub Pages. Check [deployment instructions](https://github.com/kevenmiano/nestgo/blob/main/docs/DOCS_SITE.md).
