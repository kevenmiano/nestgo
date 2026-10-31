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
go run ./cmd/nestgo diagnose -example -out diagnostics-output -samples 15 -iterations 100 -strict
# Open diagnostics-output/report.html locally, without internet access.
```

The CLI's `-example` runs a transparent demo fixture containing three providers and one HTTP route. **It does not automatically inspect an arbitrary user application**, because the compiled CLI binary cannot load Go types from an unrelated process without linking or a metadata export contract.

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
- `report.json`: graph plus measured samples and runtime platform context.
- `report.html`: self-contained SVG graph and startup sample bar chart, median, p95, B/op, allocs/op, findings and environment context. No external CDNs.
- `runtime.New` benchmark measures **warm handler initialization from prebuilt metadata**, including provider materialization/injection, controller wiring, route signature validation and `net/http` handler registration. It **excludes** OS process cold start, module compilation, generated `init()` metadata registration and socket listening.

Graph cycles are reported as strongly connected provider sets, **not necessarily ordered minimal cycle paths**. Current injector rules skip unexported/anonymous fields. Static analysis cannot guarantee runtime behavior for not-yet-modeled request scopes or future dynamic provider factories.

## Verify

- [ ] `go test` covers shuffling provider registrations, two-node and self-cycles, absent dependencies, ambiguous interface implementations, duplicate providers and report generation.
- [ ] Running graph output with the same input has identical JSON/DOT bytes.
- [ ] CI uploads an offline HTML report and real sample measurements with environment metadata.
- [ ] Warm startup timing is clearly labelled, separated from deterministic graph data.
- [ ] A future issue adds full module graph and cross-module import/export analysis (see #10, #11, #36).
- [ ] Broader startup and allocation comparison across provider graph sizes is scoped in #37; integrated reporting improvements in #38.

## Review / Ship

PR correctness checks are required before merge. Link the relevant issues but do **not** close #36–#38 until their complete acceptance criteria are met. This iteration supplies the deterministic provider graph and initial warm startup reporting; cold process boot, package-level imports/exports, full route scale sweeps and strict compiler integration remain incremental work.

**GSD:** Discuss → Plan → Execute → Verify → Review → Ship.
