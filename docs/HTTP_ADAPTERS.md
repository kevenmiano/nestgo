# HTTP platform adapters

All HTTP engines use **the same** `runtime.New` factory, dependency injection, route metadata, lifecycle hooks and shutdown.

Plain Go handler registration needs no metadata or code-generation step:

```go
app, err := runtime.New(
    runtime.WithRoutes(runtime.Route{
        Method: "GET", Path: "/ping",
        Handler: func(ctx runtime.Context) error { return ctx.Write([]byte("pong")) },
    }),
)
if err != nil { log.Fatal(err) }
log.Fatal(app.Listen(":8080"))
```

Generated controller metadata is optional. Native fasthttp (default):

```go
app, err := runtime.New(runtime.WithRegistry("shop"))
if err != nil { log.Fatal(err) }
log.Fatal(app.Listen(":8080"))
```

Go standard-library HTTP engine:

```go
app, err := runtime.New(
    runtime.WithRegistry("shop"),
    runtime.WithAdapter(nethttp.New()),
)
if err != nil { log.Fatal(err) }
log.Fatal(app.Listen(":8080"))
```

Custom engines implement `httpadapter.Adapter`: `Register(httpadapter.Route) error`, `Serve(net.Listener) error` and `Shutdown(context.Context) error`. The common `httpadapter.Context` exposes request and response operations. The application owns the adapter; always call `app.Shutdown(ctx)`.

Fasthttp uses native `fasthttp.RequestCtx` (no net/http adaptor); the standard-library engine uses `http.ServeMux`. Middleware written for net/http might require an adapter-aware integration.

## Verification priorities

Validate route precedence, 404/405, trailing slash, JSON body-size limits, headers, status codes, and graceful shutdown across engines. Benchmark startup, allocations and p95/p99 HTTP latency before production use. WebSockets and streaming are not guaranteed by the current adapter contract.

No separate legacy `http.Handler` constructor exists.

## Request limits and network security

The built-in transports enforce secure defaults for untrusted HTTP traffic. Configure them with functional options, not an additional wrapper server:

```go
limits := httpadapter.ServerLimits{
    ReadHeaderTimeout: 5 * time.Second,
    ReadTimeout:       10 * time.Second,
    WriteTimeout:      15 * time.Second,
    IdleTimeout:       30 * time.Second,
    MaxHeaderBytes:    16 << 10,
    MaxBodyBytes:      1 << 20,
}
app, err := runtime.New(runtime.WithAdapter(nethttp.New(nethttp.WithLimits(limits))))
// Or: runtime.WithAdapter(fastadapter.New(fastadapter.WithLimits(limits)))
```

All fields are positive-only overrides: zero or negative values retain the defaults. The `net/http` engine enforces its header-specific timeout, the total read deadline, a maximum header size, and a bounded body read (up to configured limit plus one byte) **before** invoking the handler; oversized Content-Length or chunked bodies return 413. The FastHTTP engine enforces the maximum request body size during parsing, a bounded request/header read buffer, and a total read deadline. FastHTTP's public server API does not provide an independent `ReadHeaderTimeout`, so `ReadTimeout` protects the entire request. On both engines, oversized requests return 413.

The generated JSON binder also applies a 1 MiB safety ceiling. Raising `MaxBodyBytes` beyond 1 MiB does not override that application-layer ceiling for generated JSON endpoints. Direct `WithRoutes` handlers can accept up to the configured adapter limit.

For Internet-facing production deployments, enforce equivalent or stricter ingress/reverse-proxy limits (headers, body size, requests per connection, idle timeouts). Proxy configuration and the application must agree on request framing to avoid request smuggling. Tightening read deadlines may reject legitimate slow uploads; adjust positive limits for the deployment and validate with the actual reverse proxy. Test suites exercise incomplete headers, chunked oversized requests, concurrent requests and graceful shutdown on a loopback listener.

## Engine-independent responses

Import `github.com/kevenmiano/nestgo/http` for transport-independent response helpers.
Controllers may return ordinary DTOs, `http.Response[DTO]` for typed response metadata,
or `http.Result` for untyped response contracts. HTTP errors and status constants
also live in this package; see [HTTP responses](HTTP_RESPONSES.md).
