---
title: CLI command reference
sidebar_position: 3
slug: /cli
---

# NestGo CLI

`nestgo` is an optional **build-time developer tool** for applications that use `.mgo` decorators and for analyzing compiled application metadata. You do **not** need the CLI to run a plain Go server with `runtime.New`.

The current CLI exposes **two commands**: `compile` and `diagnose`. Project scaffolding, generators, watch mode, `nestgo start`, `nestgo build`, and `nestgo new` are **not implemented**. The [Cobra migration](https://github.com/kevenmiano/nestgo/issues/143) will add conventional per-command help and shell completion; do not rely on them yet.

## Install

Use a Go toolchain compatible with the version declared in the [repository go.mod](https://github.com/kevenmiano/nestgo/blob/main/go.mod). For an application using the library, also add the dependency with `go get`.

```bash
go install github.com/kevenmiano/nestgo/cmd/nestgo@latest
go get github.com/kevenmiano/nestgo@latest
```

`go install` installs an executable; `go get` adds NestGo to the current application's `go.mod`. For reproducible builds, replace `@latest` with a [published release tag](https://github.com/kevenmiano/nestgo/releases). If the shell cannot find `nestgo`, check `go env GOBIN` and `go env GOPATH` and put the installation directory on `PATH`.

When working in the cloned NestGo repository, use `go run ./cmd/nestgo <command>` instead of installing a global copy.

## Command summary

| Command | Purpose |
| --- | --- |
| `nestgo compile <directory\|file.mgo>` | Convert decorated `.mgo` sources into Go source and metadata |
| `nestgo diagnose -example [flags]` | Inspect the built-in diagnostic fixture |
| `nestgo diagnose -app <go-package> [flags]` | Inspect an importable compiled application package |

### `nestgo compile`

```bash
nestgo compile ./myapp
nestgo compile ./myapp/users.mgo
# From a checkout of this repository:
go run ./cmd/nestgo compile ./examples/shop
```

**Arguments:** exactly one existing directory or `.mgo` file. Passing a file causes the CLI to compile the **containing directory**, not just that individual source. The compiler discovers `.mgo` sources, validates supported decorators and emits ordinary Go files and generated metadata in that directory. The default generator currently creates files such as `*_generated.go` and `zz_modforge_metadata.go`. Generated outputs should be reviewed and kept in sync with source files.

This command currently has **no flags** and no `--watch` or `--check` mode. Compiler errors are printed to stderr; the process exits unsuccessfully.

See [Compiler](./fundamentals/compiler.md) for the decorator model.

### `nestgo diagnose`

Diagnose either an example or a real, **importable** Go package containing compiled NestGo metadata.

```bash
nestgo diagnose -example -out ./diagnostics-output
nestgo diagnose -example -scale -samples 15 -iterations 100 -cold-samples 5
nestgo diagnose -app ./myapp -out ./diagnostics-real -strict
# From the repository checkout:
go run ./cmd/nestgo diagnose -app ./examples/shop -out ./diagnostics-real
```

| Flag | Default | Meaning |
| --- | --- | --- |
| `-example` | `false` | Use the built-in sample instead of an application package |
| `-app <package>` | unset | Importable Go package path or resolvable directory |
| `-out <directory>` | `nestgo-diagnostics` | Write the report files to this directory |
| `-samples <n>` | `15` | Number of independent warm measurement batches |
| `-iterations <n>` | `100` | Number of `runtime.New` calls per warm batch |
| `-cold-samples <n>` | `3` | Number of cold process launches, from 0 to 50; 0 disables |
| `-scale` | `false` | Measure synthetic fixtures with 1, 10, and 100 providers; example only |
| `-strict` | `false` | Exit unsuccessfully for unresolved dependency findings or cycles |

The `-app` and `-example` modes are mutually exclusive; `-scale` cannot be used with `-app`. The application must be importable (not `package main`), its generated metadata must register on import, and `go list` must resolve it from your working Go module. Compilation **must run first** when using `.mgo` sources.

**Security:** application diagnostics build and execute a temporary runner that imports your package. Importing a package executes its `init()` functions. Diagnose only trusted code.

**Outputs** include `report.html`, `report.json`, `graph.json`, `graph.dot`, `modules.json` and `modules.dot`. The offline HTML provides dependency graphs and measured startup data; benchmark values vary by machine and are not full HTTP readiness measurements. For an explanation of methodology, see [Diagnostics](./diagnostics.md).

## Development workflow

1. Author `.mgo` files in an importable Go package (only when opting into decorators).
2. Run `nestgo compile ./path/to/package` after changing decorated sources.
3. Run `go test ./...` to compile and test the generated application.
4. Optionally run `nestgo diagnose -app ./path/to/package -strict` and review `report.html`.
5. Build/deploy your application with normal Go commands, such as `go build ./...`; **the NestGo CLI is not a production runtime**.

## Common issues

| Symptom | What to check |
| --- | --- |
| `nestgo: command not found` | Check `GOBIN`, `GOPATH/bin` and `PATH`. |
| `expected .mgo file` | Pass a directory or a file ending in `.mgo`. |
| `go list ...` or `package main cannot be imported` | Run from the correct Go module and point `-app` to an importable package. |
| `use -app <importable Go package> or -example` | Choose one diagnostic mode explicitly. |
| `-app cannot be combined with -example or -scale` | Remove the conflicting diagnostic flag. |
| Dependency findings under `-strict` | Inspect `report.html`, `graph.json` and `modules.json` before retrying. |

## Current limitations

This page documents what is implemented, **not proposed NestJS command parity**. The existing CLI has limited top-level usage text and does not yet expose a conventional `nestgo --help` / `nestgo compile --help` interface. See [CLI modernization #143](https://github.com/kevenmiano/nestgo/issues/143) and [compiler tooling #24](https://github.com/kevenmiano/nestgo/issues/24) for improvements.
