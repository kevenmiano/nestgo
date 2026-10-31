---
sidebar_position: 1
title: Introduction
slug: /intro
---
# Introduction to NestGo

**NestGo** is an experimental Go source-to-source compiler and HTTP runtime inspired by the architecture and decorator ergonomics of NestJS. It is **not** NestJS for Go and is **not production-ready**.

## Requirements

- Go 1.23+
- No external Go runtime dependencies

## Quick start

```bash
git clone https://github.com/kevenmiano/nestgo.git
cd nestgo
go run ./cmd/modforge compile ./examples/shop
go test ./...
```

The compiler reads `.mgo` files and generates regular Go files and metadata. See [Compiler](./fundamentals/compiler.md) for details.

## Example

```go
@Controller("/users")
type UsersController struct{}

@Get("/")
func (c *UsersController) FindAll() []string {
    return []string{"Ada", "Grace"}
}
```

:::warning Experimental syntax
Decorator annotations are accepted by the NestGo compiler in `.mgo` files, **not** by the standard Go compiler. Compile them before using ordinary Go tooling.
:::

## Current scope

The current proof of concept supports basic controller routing, metadata generation, field injection and JSON body binding. Guards, interceptors, dynamic modules, request scopes and other NestJS concepts are **planned**, not implemented. See the [roadmap](./roadmap.md).
