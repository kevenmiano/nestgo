---
title: NestGo CLI
sidebar_position: 1
slug: /cli/overview
---

# NestGo CLI

The NestGo CLI is an **optional development tool**. It compiles `.mgo` decorator files into ordinary Go and generates dependency/startup diagnostics. Native Go applications using `runtime.New(...)` do not need it to run.

## Installation

```bash
go install github.com/kevenmiano/nestgo/cmd/nestgo@latest
nestgo --help
```

Use a published version tag instead of `@latest` for reproducible installations. Ensure `GOBIN` or `$(go env GOPATH)/bin` is in `PATH`. To add the runtime to your application module, run `go get github.com/kevenmiano/nestgo@latest` separately.

Inside the NestGo repository you can use `go run ./cmd/nestgo <command>` instead of installing the CLI.

## Command overview

| Command | Description |
| --- | --- |
| `nestgo compile <directory\|file.mgo>` | Compile a directory of `.mgo` files into Go sources and metadata |
| `nestgo diagnose -example [flags]` | Export a diagnostic report for the built-in fixture |
| `nestgo diagnose -app <package> [flags]` | Analyze an importable compiled Go package |
| `nestgo completion <shell>` | Generate bash, zsh, fish or PowerShell completions |
| `nestgo version` or `nestgo --version` | Display the CLI version (development builds say `dev`) |
| `nestgo --help` | Show available commands; use `nestgo <command> --help` for flags |

The command reference pages in the sidebar are **generated from the Cobra command definitions** at site start/build time. They are not maintained independently.

## Compile decorators

```bash
nestgo compile ./myapp
# Passing a file compiles the entire containing directory:
nestgo compile ./myapp/users.mgo
```

To watch edits during development:

```bash
nestgo compile --watch ./myapp
nestgo compile ./myapp -w --poll-interval=500ms
```

The watcher initially compiles the directory and repeats only when input fingerprints change. It ignores generated outputs, preserves existing editable tests, and prints errors while continuing to watch. It is not a runtime server and does not run `go test`. Source re-parsing still covers the whole package on each detected change.

A successful compilation emits ordinary Go files, registration metadata, a generated wiring test and editable testing TODOs. The compiler accepts one existing directory or `.mgo` file. Decorators are not syntax understood by `go build`; compile them first and then use native Go commands:

```bash
go build ./...
go test ./...
```

There is currently **no separate** `nestgo new`, `start`, `watch`, `generate`, or `test` subcommand; `--watch` is a flag on `compile`. Do not confuse this compiler with a long-running application server.

## Verify generated output in CI

```bash
nestgo compile ./myapp
nestgo check ./myapp
go test ./...
```

`nestgo check` validates decorated source and compares emitted Go metadata, module contracts, SQL repository outputs and generated wiring assertions with files already on disk. It exits unsuccessfully when files are stale or missing, printing their paths and a recompile hint. It **never writes files** and deliberately excludes editable test stubs from byte-for-byte comparison.

You can run it against a `.mgo` file; like `compile`, it operates on the containing directory. Use the same compiler version for generation and checking. It is not a substitute for `go test`, and does not start application processes or execute application `init()` functions.

## Machine-readable diagnostics

Use `--format=json` for single-shot compilation and read-only checks:

```bash
nestgo compile --format=json ./myapp
nestgo check --format=json ./myapp
```

Both commands produce one versioned JSON object containing `schema_version`,
`command`, `ok` and a `diagnostics` array. A failing compiler/check still
returns an unsuccessful process exit code; it does not add a second unstructured
error line to stderr. JSON is intended for CI, editor integrations and
automation. It identifies decorator errors, Go parser errors and missing,
outdated or obsolete generated artifacts using stable codes.

The full [compiler diagnostics contract](https://github.com/kevenmiano/nestgo/blob/main/docs/COMPILER.md)
documents all codes, source coordinates and the schema.
`compile --watch` remains text-only; JSON is **not** an NDJSON watch protocol.
Ordinary argument parsing errors from Cobra are still textual.

## Diagnose dependencies and startup

Run with the built-in fixture:

```bash
nestgo diagnose -example -out ./diagnostics-output -samples 15 -iterations 100
```

Run against your compiled application metadata:

```bash
nestgo compile ./myapp
nestgo diagnose -app ./myapp -out ./diagnostics-output -strict
```

The `-app` argument must resolve to an **importable Go package** (not `package main`). The command builds a temporary diagnostic runner and imports the application package, **executing its Go `init()` functions**. Only analyze trusted application code.

### Diagnostic flags

| Flag | Default | Purpose |
| --- | --- | --- |
| `-example` | `false` | Use the built-in fixture |
| `-app` | None | Importable Go package or relative directory |
| `-out` | `nestgo-diagnostics` | Directory for report files |
| `-samples` | `15` | Warm measurement batches |
| `-iterations` | `100` | Runtime initializations per batch |
| `-cold-samples` | `3` | Process launches; `0` disables, max `50` |
| `-scale` | `false` | Synthetic 1/10/100-provider comparison (fixture only) |
| `-strict` | `false` | Fail for graph cycles or missing/unresolved dependencies |

Select **either** `-app` or `-example`. The `-scale` option cannot be combined with `-app`. Reports contain `report.html`, `report.json`, `graph.json`, `graph.dot`, `modules.json` and `modules.dot`. The HTML file is self-contained and can be opened offline.

Warm startup timings cover runtime construction from metadata, not HTTP socket readiness or total application boot time. See the [diagnostics guide](../diagnostics.md).

## Shell completions

```bash
# Bash, current shell
source <(nestgo completion bash)

# Zsh, current shell
source <(nestgo completion zsh)
```

Use `nestgo completion --help` for detailed persistent installation instructions for bash, zsh, fish and PowerShell.


## Day-to-day workflow

1. Add `.mgo` files only when opting into NestGo decorators.
2. Run `nestgo compile ./path/to/package`, or `nestgo compile --watch ./path/to/package` during development.
3. Run `go test ./...` to test the generated Go application.
4. Optionally run `nestgo diagnose -app ./path/to/package -strict` to inspect registrations and graphs.
5. Use ordinary `go build` and deployment tooling. The NestGo CLI is not a production application runtime.

## Troubleshooting

| Symptom | Suggested fix |
| --- | --- |
| `nestgo: command not found` | Check `go env GOBIN`, `go env GOPATH`, and your shell `PATH`. |
| `expected .mgo file` | Pass a directory or file ending in `.mgo`. |
| `go list ...` fails during `diagnose` | Run inside the relevant Go module and provide a resolvable importable package. |
| `package main cannot be imported` | Point `-app` at an importable compiled package, not your executable entrypoint. |
| `use -app ... or -example` | Explicitly choose a diagnostic target. |
| `-app cannot be combined with -example or -scale` | Remove incompatible flags; `-scale` only operates on the fixture. |
| `-strict` reports errors | Inspect `report.html`, `graph.json` and `modules.json` to find dependency problems. |

## Contributing to the CLI

The Cobra command tree and tests live under `internal/cli`. The installed binary is `cmd/nestgo`, and its reference generator lives in `cmd/nestgo-docs`.

```bash
go test ./internal/cli ./cmd/nestgo ./cmd/nestgo-docs
go run ./cmd/nestgo diagnose --help
go run ./cmd/nestgo-docs ./website/docs/cli/reference
```

Docusaurus runs the generator automatically during `npm start` and `npm run build`. Do not hand-edit generated reference Markdown: update Cobra descriptions, flags, examples and tests instead.
