---
title: Application lifecycle
description: Bootstrap, dependency injection, request dispatch and shutdown in NestGo.
---

# Application lifecycle

NestGo takes inspiration from NestJS's module-oriented application model, but **its current lifecycle is not the NestJS lifecycle**. This page describes the implementation on `main`; the final section explicitly distinguishes planned capabilities.

## At a glance

```mermaid
flowchart TD
  Source[".mgo + generated Go metadata"] --> Registry["metadata.Register (Go init)"]
  Registry --> Entry{"Entry point"}
  Entry -->|"ApplicationFromRegistry"| Imports["registryImportClosure"]
  Entry -->|"NewApplication with packages"| App["NewApplication"]
  Imports --> App
  App --> Adapter["Select HTTP adapter (fasthttp default)"]
  Adapter --> Routes["registerRoutes"]
  Routes --> Instances["Instantiate providers"]
  Instances --> Graph["Validate modules and visibility"]
  Graph --> Inject["Inject provider fields"]
  Inject --> Controllers["Instantiate and inject controllers"]
  Controllers --> Map["Validate and register routes"]
  Map --> Ready["Application constructed"]
  Ready --> Listen["Listen / Serve (blocking)"]
  Listen --> Requests["HTTP dispatch"]
  Ready --> Shutdown["Shutdown(ctx)"]
  Requests -. "Caller coordinates shutdown" .-> Shutdown
```

The registry-based path resolves the **transitive registered package import closure** before constructing the application. Explicit `NewApplication` callers provide metadata packages themselves.

## Bootstrap metrics

The adapter-based factory accepts an optional `ApplicationOptions.OnBootstrap` callback with a `BootstrapMetrics` value. It reports `RouteRegistration`, `LifecycleInitialization` and `Total` durations on successful construction. Use `go test ./runtime -run '^

The adapter-based API is the recommended entry point when selecting an HTTP engine or controlling shutdown.

```go
app, err := runtime.ApplicationFromRegistry(runtime.ApplicationOptions{}, "shop")
if err != nil {
    log.Fatal(err)
}
log.Fatal(app.Listen(":8080"))
```

The generated shop package must be imported so its `init()` registration has executed. The registry name is the Go package name.

```mermaid
sequenceDiagram
  autonumber
  participant Main as main
  participant Registry as metadata registry
  participant Factory as runtime.ApplicationFromRegistry
  participant DI as registerRoutes / DI
  participant Adapter as HTTP adapter
  Main->>Factory: ApplicationFromRegistry(options, names...)
  Factory->>Registry: registryImportClosure(names)
  Registry-->>Factory: packages (or error)
  Factory->>Adapter: Select options.Adapter or fasthttp.New()
  Factory->>DI: registerRoutes(adapter, packages)
  DI->>DI: Instantiate provider pointers
  DI->>DI: Validate module visibility / imports / cycles
  DI->>DI: Inject provider fields
  DI->>DI: Create and inject controllers
  loop Each route
    DI->>Adapter: Register(Route)
  end
  DI-->>Factory: nil or bootstrap error
  Factory-->>Main: *Application or error
  Main->>Adapter: Listen -> Serve(listener)
  Note over Adapter: Serve blocks while serving requests
```

### Module boundaries and dependency injection

- Modules describe `Imports`, `Providers`, `Controllers` and `Exports` through generated metadata.
- Module validation rejects duplicate modules, unknown imports, invalid exports, missing registered providers, conflicting ownership and module import cycles.
- Each provider is instantiated as a pointer to a new struct. Exported pointer and interface fields are injected once at bootstrap, not per request.
- Pointer dependencies match exactly; interface dependencies require exactly one compatible visible provider. Missing required or ambiguous dependencies fail bootstrap. A field tagged `nestgo:"optional"` may remain unset.
- Explicit modules limit visibility to local providers and directly imported modules' exports. Legacy metadata without explicit module scopes retains global DI behavior.
- This is field injection, **not constructor injection**. An import-cycle check is **not** equivalent to detection of every provider dependency cycle; no general constructor/lifecycle ordering guarantee exists.

```mermaid
flowchart LR
  A["FeatureModule"] -->|"imports"| B["SharedModule"]
  B -->|"exports"| C["SharedService"]
  A -->|"providers"| D["FeatureService"]
  A -->|"controllers"| E["FeatureController"]
  C -->|"visible provider"| D
  D -->|"field injection"| E
```

### Startup failure behavior

Construction returns an error when dependency/route validation or registration fails. The current `NewApplication` does not expose transactional rollback of already registered adapter routes or resource cleanup for arbitrary user-managed provider resources. Do not reuse an adapter after a failed bootstrap without checking its state.

## Two APIs, two dispatch implementations

| Aspect | `NewApplication` / `ApplicationFromRegistry` | `New` / `FromRegistry` |
| --- | --- | --- |
| Return value | `*runtime.Application` | `http.Handler` |
| HTTP platform | `fasthttp` default; customizable adapter | Go `net/http` ServeMux |
| DI | `registerRoutes` in `runtime/http_routes.go` | `NewWithOptions` in `runtime/runtime.go` |
| HTTP serving | `Application.Listen/Serve` | Caller runs `http.Server` |
| Shutdown | `Application.Shutdown(ctx)` delegates to adapter | Caller owns server shutdown |
| Static zero-arg invoker | Uses generated `StaticInvoke` when eligible | Uses generated `StaticInvoke` when eligible |
| Lifecycle logs | No equivalent structured bootstrap events in `NewApplication` | `NewWithOptions` emits bootstrap/DI/route events |

These implementations are **not guaranteed to have identical routing, performance, errors or instrumentation**. Check both paths before assuming feature parity.

## Request lifecycle (currently implemented)

```mermaid
sequenceDiagram
  autonumber
  participant Client
  participant Adapter as HTTP adapter
  participant Handler as Registered handler closure
  participant Controller
  Client->>Adapter: HTTP request
  Adapter->>Handler: Handler(Context)
  Handler->>Handler: Bind Body / Param / Query
  Handler->>Controller: Reflective method call
  Controller-->>Handler: Value[, error]
  alt Application error
    Handler-->>Adapter: Error response
  else Success
    Handler->>Handler: JSON encode + select status
    Handler-->>Adapter: Write response
  end
  Adapter-->>Client: HTTP response
```

In the adapter path, `@Body` uses JSON decoding with a 1 MiB check, `@Param` and `@Query` require string parameters, and results are JSON encoded. Defaults are HTTP 201 for POST and 200 otherwise. Route metadata can override the status. There is currently no documented, unified NestJS-style guard/pipe/interceptor/filter chain.

## Shutdown (currently implemented)

```go
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()
if err := app.Shutdown(ctx); err != nil {
    log.Printf("shutdown: %v", err)
}
```

```mermaid
flowchart TD
  Signal["Caller receives termination signal"] --> Caller["Caller creates shutdown context"]
  Caller --> API["Application.Shutdown(ctx)"]
  API --> Adapter["Adapter.Shutdown(ctx)"]
  Adapter --> Native{"HTTP platform"}
  Native -->|"fasthttp"| Fast["fasthttp ShutdownWithContext"]
  Native -->|"net/http"| Net["http.Server.Shutdown"]
  Fast --> Done["Return error / nil"]
  Net --> Done
```

NestGo does **not** automatically register OS signal handlers. The adapter-based Application API now invokes provider and controller shutdown hooks when implemented; legacy http.Handler entry points do not. The application owner must orchestrate signal handling, deadline, and termination.

## Lifecycle hooks in the adapter-based API

Provider and controller instances may implement the optional interfaces `OnModuleInit(context.Context) error`, `OnApplicationBootstrap(context.Context) error`, `BeforeApplicationShutdown(context.Context) error`, `OnModuleDestroy(context.Context) error`, and `OnApplicationShutdown(context.Context) error`. The application initializes hooks after routes are registered. Initialization runs providers in stable type-name order, followed by controllers in metadata order. Shutdown runs these instances in reverse order. Shutdown hooks run around the adapter shutdown, collecting errors with `errors.Join`, and repeated calls return the same outcome.

These hooks currently exist only in `NewApplication` / `ApplicationFromRegistry`, not the legacy net/http handler constructor. There is no automatic signal handling. If bootstrap or initialization fails, the adapter-based factory attempts best-effort shutdown hooks and adapter shutdown, joining cleanup errors with the original failure. Previously registered adapter routes cannot currently be transactionally undone.

## Comparison with NestJS: implemented vs proposed

| Concept inspired by NestJS | NestGo status |
| --- | --- |
| Module imports / providers / controllers / exports | Implemented in generated module metadata and runtime visibility |
| Singleton-style bootstrap provider instances | Implemented for registered provider types |
| Injectable constructor factories, request/transient scopes | Not implemented in these runtime paths |
| `OnModuleInit` | Implemented for adapter-based applications |
| `OnApplicationBootstrap` | Implemented for adapter-based applications |
| `BeforeApplicationShutdown` | Implemented for adapter-based applications |
| `OnModuleDestroy` | Implemented for adapter-based applications |
| `OnApplicationShutdown` | Implemented for adapter-based applications |
| Explicit application shutdown via adapter | Implemented |
| Automatic shutdown signals | Not implemented |
| Unified bootstrap across both HTTP APIs | Not implemented |

### Proposed unified lifecycle (design target; currently only the adapter path has hooks)

```mermaid
stateDiagram-v2
  [*] --> Created
  Created --> ResolvingModules
  ResolvingModules --> InitializingProviders
  InitializingProviders --> ModuleInitHooks
  ModuleInitHooks --> AppBootstrapHooks
  AppBootstrapHooks --> Ready
  Ready --> Listening
  Listening --> BeforeShutdownHooks
  BeforeShutdownHooks --> AdapterShutdown
  AdapterShutdown --> ModuleDestroyHooks
  ModuleDestroyHooks --> AppShutdownHooks
  AppShutdownHooks --> Closed
  ResolvingModules --> Failed
  InitializingProviders --> Failed
  ModuleInitHooks --> Failed
  Failed --> Closed
```

Before unifying this lifecycle, specify dependency-topological ordering, cancellation and deadlines, complete rollback, and HTTP platform parity. The hooks above are available only in adapter-based applications; this diagram remains a design target for shared bootstrap semantics.

## Source of truth

- [`runtime/application.go`](https://github.com/kevenmiano/nestgo/blob/main/runtime/application.go): application factory, Serve/Listen/Shutdown.
- [`runtime/http_routes.go`](https://github.com/kevenmiano/nestgo/blob/main/runtime/http_routes.go): adapter-route DI and request dispatch.
- [`runtime/runtime.go`](https://github.com/kevenmiano/nestgo/blob/main/runtime/runtime.go): legacy net/http bootstrap and logs.
- [`runtime/module_visibility.go`](https://github.com/kevenmiano/nestgo/blob/main/runtime/module_visibility.go): graph validation and provider visibility.
- [`runtime/registry_imports.go`](https://github.com/kevenmiano/nestgo/blob/main/runtime/registry_imports.go): registered package closure.
- [`httpadapter/adapter.go`](https://github.com/kevenmiano/nestgo/blob/main/httpadapter/adapter.go): platform abstraction.
 -bench BenchmarkApplicationBootstrapProviders -benchmem` to compare startup overhead for 1, 10 and 100 providers. The legacy handler factory does not emit this callback.

## Bootstrap sequence: adapter-based application

The adapter-based API is the recommended entry point when selecting an HTTP engine or controlling shutdown.

```go
app, err := runtime.ApplicationFromRegistry(runtime.ApplicationOptions{}, "shop")
if err != nil {
    log.Fatal(err)
}
log.Fatal(app.Listen(":8080"))
```

The generated shop package must be imported so its `init()` registration has executed. The registry name is the Go package name.

```mermaid
sequenceDiagram
  autonumber
  participant Main as main
  participant Registry as metadata registry
  participant Factory as runtime.ApplicationFromRegistry
  participant DI as registerRoutes / DI
  participant Adapter as HTTP adapter
  Main->>Factory: ApplicationFromRegistry(options, names...)
  Factory->>Registry: registryImportClosure(names)
  Registry-->>Factory: packages (or error)
  Factory->>Adapter: Select options.Adapter or fasthttp.New()
  Factory->>DI: registerRoutes(adapter, packages)
  DI->>DI: Instantiate provider pointers
  DI->>DI: Validate module visibility / imports / cycles
  DI->>DI: Inject provider fields
  DI->>DI: Create and inject controllers
  loop Each route
    DI->>Adapter: Register(Route)
  end
  DI-->>Factory: nil or bootstrap error
  Factory-->>Main: *Application or error
  Main->>Adapter: Listen -> Serve(listener)
  Note over Adapter: Serve blocks while serving requests
```

### Module boundaries and dependency injection

- Modules describe `Imports`, `Providers`, `Controllers` and `Exports` through generated metadata.
- Module validation rejects duplicate modules, unknown imports, invalid exports, missing registered providers, conflicting ownership and module import cycles.
- Each provider is instantiated as a pointer to a new struct. Exported pointer and interface fields are injected once at bootstrap, not per request.
- Pointer dependencies match exactly; interface dependencies require exactly one compatible visible provider. Missing required or ambiguous dependencies fail bootstrap. A field tagged `nestgo:"optional"` may remain unset.
- Explicit modules limit visibility to local providers and directly imported modules' exports. Legacy metadata without explicit module scopes retains global DI behavior.
- This is field injection, **not constructor injection**. An import-cycle check is **not** equivalent to detection of every provider dependency cycle; no general constructor/lifecycle ordering guarantee exists.

```mermaid
flowchart LR
  A["FeatureModule"] -->|"imports"| B["SharedModule"]
  B -->|"exports"| C["SharedService"]
  A -->|"providers"| D["FeatureService"]
  A -->|"controllers"| E["FeatureController"]
  C -->|"visible provider"| D
  D -->|"field injection"| E
```

### Startup failure behavior

Construction returns an error when dependency/route validation or registration fails. The current `NewApplication` does not expose transactional rollback of already registered adapter routes or resource cleanup for arbitrary user-managed provider resources. Do not reuse an adapter after a failed bootstrap without checking its state.

## Two APIs, two dispatch implementations

| Aspect | `NewApplication` / `ApplicationFromRegistry` | `New` / `FromRegistry` |
| --- | --- | --- |
| Return value | `*runtime.Application` | `http.Handler` |
| HTTP platform | `fasthttp` default; customizable adapter | Go `net/http` ServeMux |
| DI | `registerRoutes` in `runtime/http_routes.go` | `NewWithOptions` in `runtime/runtime.go` |
| HTTP serving | `Application.Listen/Serve` | Caller runs `http.Server` |
| Shutdown | `Application.Shutdown(ctx)` delegates to adapter | Caller owns server shutdown |
| Static zero-arg invoker | Uses generated `StaticInvoke` when eligible | Uses generated `StaticInvoke` when eligible |
| Lifecycle logs | No equivalent structured bootstrap events in `NewApplication` | `NewWithOptions` emits bootstrap/DI/route events |

These implementations are **not guaranteed to have identical routing, performance, errors or instrumentation**. Check both paths before assuming feature parity.

## Request lifecycle (currently implemented)

```mermaid
sequenceDiagram
  autonumber
  participant Client
  participant Adapter as HTTP adapter
  participant Handler as Registered handler closure
  participant Controller
  Client->>Adapter: HTTP request
  Adapter->>Handler: Handler(Context)
  Handler->>Handler: Bind Body / Param / Query
  Handler->>Controller: Reflective method call
  Controller-->>Handler: Value[, error]
  alt Application error
    Handler-->>Adapter: Error response
  else Success
    Handler->>Handler: JSON encode + select status
    Handler-->>Adapter: Write response
  end
  Adapter-->>Client: HTTP response
```

In the adapter path, `@Body` uses JSON decoding with a 1 MiB check, `@Param` and `@Query` require string parameters, and results are JSON encoded. Defaults are HTTP 201 for POST and 200 otherwise. Route metadata can override the status. There is currently no documented, unified NestJS-style guard/pipe/interceptor/filter chain.

## Shutdown (currently implemented)

```go
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()
if err := app.Shutdown(ctx); err != nil {
    log.Printf("shutdown: %v", err)
}
```

```mermaid
flowchart TD
  Signal["Caller receives termination signal"] --> Caller["Caller creates shutdown context"]
  Caller --> API["Application.Shutdown(ctx)"]
  API --> Adapter["Adapter.Shutdown(ctx)"]
  Adapter --> Native{"HTTP platform"}
  Native -->|"fasthttp"| Fast["fasthttp ShutdownWithContext"]
  Native -->|"net/http"| Net["http.Server.Shutdown"]
  Fast --> Done["Return error / nil"]
  Net --> Done
```

NestGo does **not** automatically register OS signal handlers. The adapter-based Application API now invokes provider and controller shutdown hooks when implemented; legacy http.Handler entry points do not. The application owner must orchestrate signal handling, deadline, and termination.

## Lifecycle hooks in the adapter-based API

Provider and controller instances may implement the optional interfaces `OnModuleInit(context.Context) error`, `OnApplicationBootstrap(context.Context) error`, `BeforeApplicationShutdown(context.Context) error`, `OnModuleDestroy(context.Context) error`, and `OnApplicationShutdown(context.Context) error`. The application initializes hooks after routes are registered. Initialization runs providers in stable type-name order, followed by controllers in metadata order. Shutdown runs these instances in reverse order. Shutdown hooks run around the adapter shutdown, collecting errors with `errors.Join`, and repeated calls return the same outcome.

These hooks currently exist only in `NewApplication` / `ApplicationFromRegistry`, not the legacy net/http handler constructor. There is no automatic signal handling. If bootstrap or initialization fails, the adapter-based factory attempts best-effort shutdown hooks and adapter shutdown, joining cleanup errors with the original failure. Previously registered adapter routes cannot currently be transactionally undone.

## Comparison with NestJS: implemented vs proposed

| Concept inspired by NestJS | NestGo status |
| --- | --- |
| Module imports / providers / controllers / exports | Implemented in generated module metadata and runtime visibility |
| Singleton-style bootstrap provider instances | Implemented for registered provider types |
| Injectable constructor factories, request/transient scopes | Not implemented in these runtime paths |
| `OnModuleInit` | Implemented for adapter-based applications |
| `OnApplicationBootstrap` | Implemented for adapter-based applications |
| `BeforeApplicationShutdown` | Implemented for adapter-based applications |
| `OnModuleDestroy` | Implemented for adapter-based applications |
| `OnApplicationShutdown` | Implemented for adapter-based applications |
| Explicit application shutdown via adapter | Implemented |
| Automatic shutdown signals | Not implemented |
| Unified bootstrap across both HTTP APIs | Not implemented |

### Proposed unified lifecycle (design target; currently only the adapter path has hooks)

```mermaid
stateDiagram-v2
  [*] --> Created
  Created --> ResolvingModules
  ResolvingModules --> InitializingProviders
  InitializingProviders --> ModuleInitHooks
  ModuleInitHooks --> AppBootstrapHooks
  AppBootstrapHooks --> Ready
  Ready --> Listening
  Listening --> BeforeShutdownHooks
  BeforeShutdownHooks --> ModuleDestroyHooks
  ModuleDestroyHooks --> AdapterShutdown
  AdapterShutdown --> AppShutdownHooks
  AppShutdownHooks --> Closed
  ResolvingModules --> Failed
  InitializingProviders --> Failed
  ModuleInitHooks --> Failed
  Failed --> Closed
```

Before implementing this target, specify dependency topological ordering, deterministic hooks, cancellation/deadlines, failures and rollback, idempotent shutdown, and parity between HTTP platforms. Do not attach hook names to user code until those contracts exist.

## Source of truth

- [`runtime/application.go`](https://github.com/kevenmiano/nestgo/blob/main/runtime/application.go): application factory, Serve/Listen/Shutdown.
- [`runtime/http_routes.go`](https://github.com/kevenmiano/nestgo/blob/main/runtime/http_routes.go): adapter-route DI and request dispatch.
- [`runtime/runtime.go`](https://github.com/kevenmiano/nestgo/blob/main/runtime/runtime.go): legacy net/http bootstrap and logs.
- [`runtime/module_visibility.go`](https://github.com/kevenmiano/nestgo/blob/main/runtime/module_visibility.go): graph validation and provider visibility.
- [`runtime/registry_imports.go`](https://github.com/kevenmiano/nestgo/blob/main/runtime/registry_imports.go): registered package closure.
- [`httpadapter/adapter.go`](https://github.com/kevenmiano/nestgo/blob/main/httpadapter/adapter.go): platform abstraction.
