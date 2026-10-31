# HTTP engines (initial compatibility implementation)

NestGo supports engine selection at application startup. The default is `fasthttp`;
`net/http` remains available. Controller decorators, dependency injection and
generated metadata are not configured per engine.

```go
app, err := runtime.ApplicationFromRegistry(runtime.ApplicationOptions{}, "shop")
if err != nil { log.Fatal(err) }
log.Fatal(app.Listen(":8080"))
```

To explicitly select the Go standard HTTP engine:

```go
app, err := runtime.ApplicationFromRegistry(
    runtime.ApplicationOptions{Engine: runtime.EngineNetHTTP}, "shop",
)
```

**Important performance caveat:** this first implementation uses
`fasthttpadaptor.NewFastHTTPHandler` to reuse the current `net/http` runtime.
It is a working fasthttp server selection mechanism, **not yet a zero-copy
native fasthttp execution path**. Conversion overhead must be measured.
The public `runtime.New` and `runtime.FromRegistry` APIs remain compatible
and return `http.Handler`. Application engines are selected on bootstrap,
not switched while serving requests.

## Next milestones

1. Extract precompiled controller invocation, route descriptors and bindings
   from the `http.ServeMux` registration code.
2. Implement a native `fasthttp.RequestCtx` router with identical route
   matching, JSON body limits, status codes, params and query behavior.
3. Add concurrent HTTP integration tests for both engines, startup benchmarks,
   requests/sec, allocs/op and p95/p99 latency using Gatling.
4. Add explicit engine feature capability checks (WebSocket, streaming,
   multipart, context cancellation and middleware compatibility).
5. Generate an up-to-date `go.sum` using `go mod tidy` before merging.
