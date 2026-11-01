# Runtime and lifecycle

NestGo has **one application factory** and one lifecycle owner for all HTTP engines.

## Get started

### Plain Go HTTP server (no NestGo compiler)

```go
app, err := runtime.New(runtime.WithRoutes(runtime.Route{
    Method: "GET",
    Path: "/health",
    Handler: func(ctx runtime.Context) error {
        ctx.Status(200)
        return ctx.Write([]byte("OK"))
    },
}))
if err != nil { log.Fatal(err) }
log.Fatal(app.Listen(":8080"))
```

The `runtime` package is a regular Go dependency: no prebuilt runtime binary, NestGo code-generation step, or runtime compiler invocation is required. Go builds and links the imported package with the application, as it does for gRPC.

### Optional generated modules

```go
// The compiled package must be imported somewhere in the binary.
// import _ "example.com/yourapp/internal/app"
app, err := runtime.New()
if err != nil { log.Fatal(err) }
defer app.Shutdown(context.Background())
log.Fatal(app.Listen(":8080"))
```

`runtime.New(options ...runtime.Option)` returns `*runtime.Application`. Options:

| Option | Function |
| --- | --- |
| `WithRoutes(routes...)` | Register Go HTTP functions directly (no codegen) |
| `WithPackages(packages...)` | Provide metadata directly |
| `WithAdapter(adapter)` | Choose HTTP engine, default fasthttp |
| `WithLogger(logger)` | Enable structured bootstrap logging |
| `WithContext(ctx)` | Supply a cancellable bootstrap context |
| `WithDebug(true)` | Enable detailed per-provider diagnostics |
| `WithBootstrapMetrics(callback)` | Observe successful bootstrap durations |

`WithPackages` is reserved for explicit metadata fixtures and advanced embedding. Generated modules are loaded automatically when linked. A nil option is rejected. `Application` provides `Listen`, `Serve`, `Shutdown(ctx)`, `Close()` and `HTTPAdapter`; it does not expose a second `http.Handler` bootstrap.

## Lifecycle

```mermaid
flowchart TD
  New["runtime.New"] --> Packages{"Metadata source"}
  Packages -->|"auto-discovery"| Closure["Registered package import closure"]
  Packages -->|"WithPackages"| Direct["Explicit metadata"]
  Closure --> Graph["Validate module scopes and imports"]
  Direct --> Graph
  Graph --> Providers["Instantiate and inject providers"]
  Providers --> Order["Dependency-first lifecycle order; module imports before dependents"]
  Order --> Controllers["Instantiate and inject controllers"]
  Controllers --> Routes["Validate and register HTTP routes"]
  New --> Raw["WithRoutes: predeclared Go handlers"]
  Routes --> Registration["Register direct routes in same adapter"]
  Raw --> Registration
  Registration --> Hooks["OnModuleInit / OnApplicationBootstrap"]
  Hooks --> Application["Application ready"]
```

Providers initialize in dependency-first topological order using their actually resolved, explicitly tagged injections. Imported modules precede their importers when providers are independent; controllers run after providers. Self and multi-provider cycles fail bootstrap as `*runtime.InjectionError` with code `dependency_cycle` and a field-aware cycle path. A missing optional dependency creates no ordering edge; external provider instances and generated factories remain injectable. Module scopes control visibility. Generated factory callbacks can resolve arbitrary types dynamically; if their lifecycle depends on another service, declare an explicit injectable field to make the dependency visible to the ordering graph. Factory-only resolver calls cannot be inferred as lifecycle ordering edges.

Direct Go handlers execute without reflective controller dispatch. For decorator-generated controllers, generated invokers can bypass reflective calls, while other handlers use reflection as needed. Both native `platform/fasthttp` and `platform/nethttp` use the same route namespace and lifecycle. Conflicts between direct and controller-generated routes are rejected at bootstrap.

`app.Shutdown(ctx)` runs `BeforeApplicationShutdown`, adapter shutdown, `OnModuleDestroy`, then `OnApplicationShutdown` in reverse instance order. It is idempotent and joins errors. Failed bootstrap attempts best-effort cleanup of successfully constructed provider factories and external services, including cancellation via a bounded independent cleanup context. Registered adapter routes are not transactionally rolled back, but an application that fails construction is never served. The caller owns OS signal handling and deadlines.

## Observability and constraints

Use `WithBootstrapMetrics` for route registration, lifecycle initialization and total durations. The `BenchmarkApplicationBootstrapProviders` benchmark and CI baseline comparison cover 1/10/100 providers.

Constructor injection, request scopes, automatic OS signals, universal filters/guards/interceptors and arbitrary resource disposal are not implemented. Successful application construction does not mean the HTTP port is bound.

See [HTTP adapters](HTTP_ADAPTERS.md) and [lifecycle guide](../website/docs/fundamentals/lifecycle.md).
