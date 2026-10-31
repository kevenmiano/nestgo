# NestGo dependency graph and startup diagnostics — GSD

## Discuss

NestJS documents circular dependencies, dependency resolution errors and module/provider introspection. NestGo's current public `metadata.Provider` contracts contain Go `reflect.Type` values but no module imports/exports graph, so this implementation analyzes **provider injection edges** and intentionally does not pretend to model imports/exports, missing compiler annotations, or future provider scopes.

The graph structure can be deterministic. Wall-clock initialization time cannot: it depends on the CPU, Go runtime, GC, process scheduling and concurrent load. Do not claim exact repeatability of nanoseconds or impose arbitrary CI millisecond thresholds.

## Plan

- **Graph:** canonical sorted provider nodes/edges, strongly connected components for cycles, and findings for missing, ambiguous, invalid or duplicated provider registrations.
- **Performance:** repeat `runtime.New(packages...)` on a fixed fixture, measure batches and use a separate Go benchmark for bytes and allocations per startup.
- **Reporting:** offline standalone HTML with embedded SVG diagrams, plus `graph.json`, `graph.dot`, and `report.json` for machine processing.
- **CI:** correctness tests and an example artifact on PRs, with no mandatory absolute timing baseline.

## Execute / reproduce

```bash
go test ./diagnostics ./cmd/nestgo
go build -o ./nestgo-local ./cmd/nestgo
./nestgo-local diagnose -example -out diagnostics-output -samples 15 -iterations 100 -cold-samples 5 -scale -strict
# Open diagnostics-output/report.html locally, without internet access.
```

The CLI's `-example` runs a transparent demo fixture containing three providers and one HTTP route. To analyze **real application metadata**, use `-app` with an importable Go package whose generated code calls `metadata.Register`:

```bash
go run ./cmd/nestgo diagnose -app ./examples/shop -out diagnostics-real -samples 10 -iterations 50 -cold-samples 3 -strict
```

The CLI resolves the package using `go list`, creates an ephemeral Go runner that blank-imports the package, compiles it once, then executes the runner to inspect `metadata.All()`. It does not modify application source files or embed fixture providers in the results. It measures warm `runtime.New` and process cold launch-to-exit from the linked executable. `-scale` is only supported with `-example`. **Importing and initializing application packages executes their Go `init()` functions and their dependencies.** Only diagnose code you trust.

Current limitations: `-app` requires an importable non-`main` package with compiled/generated NestGo registration. It does not automatically resolve the entire application startup, nor infer module imports/exports, which are not represented in the current `metadata.Module` contract. Package-local unregistered controllers/providers are invisible. Direct use of `diagnostics.Analyze` and `MeasureStartup` remains supported for programmatic use.

### Inspect your own application

Call the Go API from an application that has access to its typed provider metadata:

```go
package main

import (
    "os"
    "github.com/kevenmiano/nestgo/diagnostics"
    "github.com/kevenmiano/nestgo/metadata"
)

func main() {
    // Replace with the actual, typed metadata.Package values of your application.
    packages := []metadata.Package{ /* ... */ }
    graph := diagnostics.Analyze(packages...)
    measured, err := diagnostics.MeasureStartup("myapp", packages, 15, 100, os.Getenv("GITHUB_SHA"))
    if err != nil { panic(err) }
    report := diagnostics.Report{Graph: graph, Startup: measured}
    page, err := report.HTML()
    if err != nil { panic(err) }
    if err := os.WriteFile("nestgo-diagnostics.html", page, 0644); err != nil { panic(err) }
}
```

For custom apps, ensure your provider graph passes validation and `runtime.New` accepts the supplied packages. A cyclic provider relationship may be reported structurally even where pointer wiring is technically possible; the diagnostics don't mutate or enforce the runtime dependency policy.

## Output contract

- `graph.json`: stable graph schema, sorted vertices/edges/findings and sorted SCC cycles; no wall-clock values.
- `graph.dot`: byte-stable Graphviz-compatible dependency diagram. Arrow: consumer → dependency.
- `report.json`: graph plus measured warm samples, optional **cold subprocess** samples, scale benchmark series and runtime platform context.
- `report.html`: self-contained SVG graph and warm startup sample chart, **cold subprocess startup** median/p95, synthetic 1/10/100-provider comparison table, B/op, allocs/op, findings and environment context. No external CDNs.
- `runtime.New` benchmark measures **warm handler initialization from prebuilt metadata**, including provider materialization/injection, controller wiring, route signature validation and `net/http` handler registration. It **excludes** OS process cold start, module compilation, generated `init()` metadata registration and socket listening.
- `-cold-samples N` (default 3; `0` disables) runs the **already-built CLI executable** as N new OS processes. Each subprocess creates the same provider/route fixture and exits. This includes OS process launch, Go runtime/package `init`, CLI argument processing, and `runtime.New`, but **not** compiler invocation, Go builds, network listening or HTTP readiness. Cold timings are not comparable to a complete production server becoming ready.
- `-scale` measures three reproducible synthetic fixtures of **1, 10 and 100 providers**, with **no dependencies and no HTTP routes**. They quantify provider registration/materialization overhead, not large service applications. All timing samples are nondeterministic and carry runner metadata.
- `go run` can still execute the sample, but for cold timing consistency build the executable once as illustrated above. The subprocess does not invoke `go run` or a shell.

Graph cycles are reported as strongly connected provider sets, **not necessarily ordered minimal cycle paths**. Current injector rules skip unexported/anonymous fields. Static analysis cannot guarantee runtime behavior for not-yet-modeled request scopes or future dynamic provider factories.

## Verify

- [ ] `go test` covers shuffling provider registrations, two-node and self-cycles, absent dependencies, ambiguous interface implementations, duplicate providers and report generation.
- [ ] Running graph output with the same input has identical JSON/DOT bytes.
- [ ] CI uploads an offline HTML report and real sample measurements with environment metadata.
- [ ] Warm startup timing is clearly labelled, separated from deterministic graph data.
- [ ] A future issue adds full module graph and cross-module import/export analysis (see #10, #11, #36).
- [ ] Baseline provider-scale comparison (1/10/100) and cold process launch-to-exit samples added as incremental deliverables for #37; full application readiness and large route matrices remain pending.
- [ ] Cold timing and scale data displayed in the same offline report under #38; graph layout refinement and arbitrary application discovery remain pending.

## Review / Ship

PR correctness checks are required before merge. Link the relevant issues but do **not** close #36–#38 until their complete acceptance criteria are met. The initial implementation supplies a deterministic provider graph and warm startup reporting. This iteration adds a **bounded cold subprocess** and synthetic provider scale benchmarks; full application cold readiness, package-level imports/exports, full route scale sweeps and strict compiler integration remain incremental work.

**GSD:** Discuss → Plan → Execute → Verify → Review → Ship.

## Module import/export graph (GSD iteration)

The module graph is now exported as `modules.json` and `modules.dot`, included in `report.json` and displayed in the standalone HTML. This graph has its own deterministic schema and is separate from provider field injection.

For modules registered programmatically, use the backwards-compatible metadata contract:

```go
metadata.Module{
    Type: reflect.TypeOf((*UsersModule)(nil)),
    Name: "users",
    Imports: []reflect.Type{reflect.TypeOf((*DatabaseModule)(nil))},
    Exports: []reflect.Type{reflect.TypeOf((*UsersService)(nil))},
}
```

The validator detects directed module import cycles (Tarjan SCC), missing imports, invalid module identifiers, duplicate module types and exported providers absent from the registered provider inventory. `-strict` returns nonzero when any module or provider graph errors are present. All nodes and edges are ordered canonically; timing is still measured and nondeterministic.

**Important compatibility boundary:** the current `@Module("name")` compiler output only generates `Type` and `Name`. It does not infer `Imports` or `Exports` from existing `.mgo` source code. Therefore, existing generated applications report registered modules but no module edges until explicit import/export metadata is supplied. The current runtime does not yet enforce module visibility; this is a **diagnostic contract**, not a completed scoped DI container. Full compiler-declared ownership and import/export enforcement remain follow-ups on #10, #11 and #36.

**Verify:** golden-order test covers shuffled modules and two-node cycle; additional tests validate missing modules, self-cycles, legacy metadata, duplicate registration and invalid exports/imports. The app CI verifies `UsersModule` occurs in `modules.json`.

## Compiler-generated module relationships (GSD iteration)

The compiler now supports explicit, **static local type names** on a module, without needing to handwrite `metadata.Module`:

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

The `@Imports` and `@Exports` decorators accept quoted, comma-separated **local type names** from any `.mgo` file in the same package directory. They are validated against `@Module` and `@Injectable` declarations, deduplicated by source validation and sorted before code generation. Unresolved, duplicate, non-identifier, or non-literal references cause a source-local compiler diagnostic. The legacy `@Module("users")` syntax remains valid.

`nestgo compile ./myapp` emits the typed `Imports []reflect.Type` and `Exports []reflect.Type` metadata. Then `nestgo diagnose -app ./myapp -strict` uses the generated relationships to detect module import cycles, alongside the existing provider graph and measured startup timings.

**Intentional limitation:** this first implementation supports module and provider references in the **same Go package**, not qualified cross-package imports or a NestJS-style module ownership/resolution container. The compiler validates declared references and the diagnostic checks runtime-registered types; the runtime still does not enforce exports or module visibility. Cross-package ownership and runtime enforcement remain under #10 and #11. The compiler also does not infer module edges from arbitrary Go field types; relationships must be declared explicitly.

## NestJS-style module configuration with type-reference arrays

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

