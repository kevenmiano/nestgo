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
// Import the generated package to register its metadata.
app, err := runtime.New(runtime.WithRegistry("shop"))
if err != nil { log.Fatal(err) }
defer app.Shutdown(context.Background())
log.Fatal(app.Listen(":8080"))
```

`runtime.New(options ...runtime.Option)` returns `*runtime.Application`. Options:

| Option | Function |
| --- | --- |
| `WithRoutes(routes...)` | Register Go HTTP functions directly (no codegen) |
| `WithRegistry(names...)` | Resolve registered packages and transitive module imports |
| `WithPackages(packages...)` | Provide metadata directly |
| `WithAdapter(adapter)` | Choose HTTP engine, default fasthttp |
| `WithLogger(logger)` | Enable structured bootstrap logging |
| `WithContext(ctx)` | Pass a cancellable context to bootstrap lifecycle hooks |
| `WithDebug(true)` | Enable detailed per-provider diagnostics |
| `WithBootstrapMetrics(callback)` | Observe successful bootstrap durations |

Do not combine `WithRegistry` and `WithPackages`. A nil option is rejected. `Application` provides `Listen`, `Serve`, `Shutdown(ctx)`, `Close()` and `HTTPAdapter`; it does not expose a second `http.Handler` bootstrap.

## Lifecycle

```mermaid
flowchart TD
  New["runtime.New"] --> Packages{"Metadata source"}
  Packages -->|"WithRegistry"| Closure["Registered package import closure"]
  Packages -->|"WithPackages"| Direct["Explicit metadata"]
  Closure --> Graph["Validate module scopes and imports"]
  Direct --> Graph
  Graph --> Providers["Instantiate and inject providers"]
  Providers --> Controllers["Instantiate and inject controllers"]
  Controllers --> Routes["Validate and register HTTP routes"]
  New --> Raw["WithRoutes: predeclared Go handlers"]
  Routes --> Registration["Register direct routes in same adapter"]
  Raw --> Registration
  Registration --> Hooks["OnModuleInit / OnApplicationBootstrap"]
  Hooks --> Application["Application ready"]
```

Providers are initialized in stable type-name order and controllers follow metadata order. Field injection supports exported pointer and interface fields. The supported explicit markers are `nestgo:"inject"` (required), `nestgo:"optional"` (missing provider is permitted), and `nestgo:"-"` (exclude a field from injection even if a provider exists). Invalid tags, tagged unexported fields, tagged non-pointer fields, ambiguous interface matches, and nil providers fail at bootstrap. These failures return `*runtime.InjectionError` with stable `Code`, `Target`, `Field` and `Dependency` fields for programmatic checks through `errors.As`; messages remain descriptive for logs. Dependencies for each object are validated before writing any field, preventing partially injected objects after validation failure. Module scopes control imports and exports.

**Breaking change:** untagged pointer and interface fields are ordinary user state and are **never** injected, even if a compatible provider exists. Add `nestgo:"inject"` to every required dependency and `nestgo:"optional"` to every optional dependency in .mgo definitions and ordinary Go structs. `nestgo:"-"` explicitly documents intentionally ignored fields. The compiler validates missing local concrete providers only for `inject`-tagged fields. No legacy implicit-injection compatibility mode is retained.

Direct Go handlers execute without reflective controller dispatch. For decorator-generated controllers, generated invokers can bypass reflective calls, while other handlers use reflection as needed. Both native `platform/fasthttp` and `platform/nethttp` use the same route namespace and lifecycle. Conflicts between direct and controller-generated routes are rejected at bootstrap.

`app.Shutdown(ctx)` runs `BeforeApplicationShutdown`, adapter shutdown, `OnModuleDestroy`, then `OnApplicationShutdown` in reverse instance order. It is idempotent and joins errors. Failed bootstrap attempts best-effort cleanup, but registered adapter routes are not transactionally rolled back. The caller owns OS signal handling and deadlines. `Close()` delegates to `Shutdown(context.Background())`; use `Shutdown(ctx)` when a timeout or cancellation deadline is required. `WithContext` applies only to bootstrap lifecycle hooks and does not implicitly cancel serving requests or shut down an already started application.

## Observability and constraints

Use `WithBootstrapMetrics` for route registration, lifecycle initialization and total durations. The `BenchmarkApplicationBootstrapProviders` benchmark and CI baseline comparison cover 1/10/100 providers.

Constructor injection, request scopes, automatic OS signals, universal filters/guards/interceptors and arbitrary resource disposal are not implemented. Successful application construction does not mean the HTTP port is bound.

See [HTTP adapters](HTTP_ADAPTERS.md) and [lifecycle guide](../website/docs/fundamentals/lifecycle.md).
