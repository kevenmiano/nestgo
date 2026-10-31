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

The sample uses a built-in Go fixture with three providers and one HTTP route. It **does not automatically inspect unrelated applications**.

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

## Planned improvements

Full module imports/exports validation, cold-process startup, large provider/route benchmark matrices and expanded CLI discovery are tracked in [#36](https://github.com/kevenmiano/nestgo/issues/36), [#37](https://github.com/kevenmiano/nestgo/issues/37) and [#38](https://github.com/kevenmiano/nestgo/issues/38).
