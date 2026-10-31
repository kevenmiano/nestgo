---
title: Diagnostics and startup performance
sidebar_position: 3
---

# Dependency graph and startup diagnostics

NestGo can export a deterministic provider dependency graph together with **measured** initialization performance, all in one offline HTML report.

## Generate the sample report

```bash
go run ./cmd/nestgo diagnose -example -out diagnostics-output -samples 15 -iterations 100
```

Open `diagnostics-output/report.html` locally. The report embeds the graph SVG, startup sample chart, median and p95 latency, per-startup memory and allocation statistics, missing or ambiguous provider findings and cycles.

The example uses a built-in fixture. For an **actual compiled NestGo package**, run:

```bash
go run ./cmd/nestgo diagnose -app ./examples/shop -out diagnostics-real -samples 10 -iterations 50 -cold-samples 3
```

The CLI compiles a temporary, linked Go diagnostic runner and imports the application's generated metadata registration. It does not edit user files. The package must be importable (not `package main`), and its generated metadata must register with NestGo. **Importing that package executes its `init()` functions**; analyze trusted application code only.

The current metadata only exposes provider/controller and module descriptors; module import/export edges are not yet defined. Thus the provider graph is real, but it is **not yet a complete NestJS-style module graph**.

## Files

| Artifact | Description |
| --- | --- |
| `report.html` | Self-contained offline visual graph and startup timing chart |
| `report.json` | Machine-readable graph, raw timings and environment |
| `graph.json` | Deterministically sorted nodes, edges, cycles and findings |
| `graph.dot` | Graphviz-compatible dependency graph |

## How reproducible is it?

The structural graph is deterministically ordered for identical input metadata. Latency timings depend on the machine, process scheduling and Go runtime; repeat runs can and should differ.

The current measurement covers warm `runtime.New` construction, including reflection-based provider initialization and HTTP route setup. It does not include compilation, launching a new process, package `init()` or opening a listening socket.

For real applications, embed the Go `diagnostics.Analyze` / `diagnostics.MeasureStartup` API with application-owned `metadata.Package` values. See the [technical and GSD guide](https://github.com/kevenmiano/nestgo/blob/main/docs/DIAGNOSTICS.md) for the Go integration example.

## Cold-process and provider-size comparisons

Build once and measure separately:

```bash
go build -o ./nestgo-local ./cmd/nestgo
./nestgo-local diagnose -example -cold-samples 5 -scale -out diagnostics-output
```

The HTML includes real **cold subprocess** median/p95 times measured from process launch through Go initialization, CLI execution and `runtime.New`. These are not full HTTP server readiness metrics. The optional scale table benchmarks **1, 10 and 100 synthetic providers with no dependencies or routes**, reporting warm-start latency and allocations. Fixture sizes and graph structure are deterministic; wall-clock samples are not.

## Planned improvements

Full module imports/exports validation, cold-process startup, large provider/route benchmark matrices and expanded CLI discovery are tracked in [#36](https://github.com/kevenmiano/nestgo/issues/36), [#37](https://github.com/kevenmiano/nestgo/issues/37) and [#38](https://github.com/kevenmiano/nestgo/issues/38).

## Module dependency graph

The analyzer now produces `modules.json` and `modules.dot` alongside the provider graph. Explicit `metadata.Module.Imports` and `metadata.Module.Exports` describe directed module imports and exported provider types. It reports module cycles and missing or invalid references in the same HTML report.

Existing `@Module("users")` generated metadata is compatible and appears as a module node. **The compiler does not yet generate imports/exports**, and runtime module visibility is not enforced; use explicit programmatic module metadata to model those relationships. See the [diagnostics GSD guide](https://github.com/kevenmiano/nestgo/blob/main/docs/DIAGNOSTICS.md).

## Declare module relationships in `.mgo`

```go
@Injectable()
type UsersService struct{}

@Module("database")
type DatabaseModule struct{}

@Module("users")
@Imports("DatabaseModule")
@Exports("UsersService")
type UsersModule struct{}
```

The compiler converts these decorators to typed `metadata.Module.Imports` and `.Exports`, with deterministic ordering and compile-time validation. References are limited to modules and providers declared in the same Go package; cross-package imports and runtime enforcement of module visibility are not yet supported. The legacy `@Module("name")` form remains compatible.

## Declarative module configuration with type-reference arrays

The preferred syntax is a **single** declarative module decorator, with actual type references rather than comma-separated strings:

```go
@Module({
    name: "users",
    imports: [DatabaseModule, CacheModule],
    controllers: [UsersController],
    providers: [UsersService],
    exports: [UsersService],
})
type UsersModule struct{}
```

The .mgo compiler parses these arrays statically, validates local @Module/@Injectable/@Controller declarations, rejects quoted items, unknown or duplicate entries, and emits ordered typed reflect metadata. A provider listed in exports must also be in providers when the providers property is present. `@Module("users")` and the earlier `@Imports/@Exports` forms remain available for compatibility, but the new object syntax is the preferred design. Type references currently must be declared in the same Go package; cross-package selectors and runtime ownership enforcement are not implemented. The object is compile-time .mgo syntax, not ordinary Go source.

