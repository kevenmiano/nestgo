---
title: Install NestGo
sidebar_position: 2
---

# Install NestGo

NestGo uses one Go module with an optional installable CLI and importable library packages. Check the minimum supported Go version in the [root `go.mod`](https://github.com/kevenmiano/nestgo/blob/main/go.mod); the module currently declares Go 1.27.

## 1. Install the CLI

```bash
go install github.com/kevenmiano/nestgo/cmd/nestgo@latest
nestgo compile ./myapp
```

The CLI is needed only for decorated `.mgo` code or diagnostics; ordinary Go routes run without it. See the [CLI command reference](./cli.md) for commands, arguments, flags, reports and troubleshooting.

For reproducible builds, replace `@latest` with a published release tag such as `@v0.1.0`. Check [available releases](https://github.com/kevenmiano/nestgo/releases) before pinning a version.

The legacy `modforge` entrypoint still exists for older development workflows, but new projects should use `nestgo`.

## 2. Use the runtime in your application

Inside your application module, run:

```bash
go mod init example.com/myapp
go get github.com/kevenmiano/nestgo@latest
```

Then import the packages you need:

```go
import (
  "github.com/kevenmiano/nestgo/runtime"
  "github.com/kevenmiano/nestgo/metadata"
)
```

`compiler`, `runtime` and `metadata` are part of **one Go module**. You do not need to install separate dependencies for them.

:::warning Installation distinction
`go install` installs the `nestgo` executable and resolves its build dependencies. It does **not** add NestGo to your application's `go.mod`. Use `go get` for that.
:::

## 3. Understand releases

The [Docusaurus](https://github.com/kevenmiano/nestgo/blob/main/.github/workflows/docs-release.yml), [Go libraries](https://github.com/kevenmiano/nestgo/blob/main/.github/workflows/go-packages.yml) and [framework CLI](https://github.com/kevenmiano/nestgo/blob/main/.github/workflows/framework-release.yml) use **independent release pipelines**, all driven by the same versioned GitHub Release from `main`.

The Go module version ensures the runtime, metadata and compiler contracts are compatible. Separate GitHub workflows do not imply independently versioned submodules.

For supported capabilities, see [the introduction](./intro.md). The framework remains experimental and is not production-ready.

For technical details and the full GSD release checklist, see the [distribution guide](https://github.com/kevenmiano/nestgo/blob/main/docs/DISTRIBUTION.md).
