---
title: Middleware
---

# Middleware

Middleware executes before guards, pipes and the route handler. Use it for
request logging, tracing headers or early termination. This follows the
[NestJS middleware concept](https://docs.nestjs.com/middleware), but NestGo
uses a transport-neutral `httpadapter.Context` instead of Express/Fastify
request and response types.

## Class-based middleware with a static contract

Reusable middleware can implement `httpadapter.MiddlewareHandler`:

```go
type AuditMiddleware struct{}

func (*AuditMiddleware) Use(ctx httpadapter.Context, next httpadapter.Next) error {
    ctx.SetHeader("X-Audit", "enabled")
    return next()
}

// The Go compiler rejects a missing or mismatched Use method.
var _ httpadapter.MiddlewareHandler = (*AuditMiddleware)(nil)

app, err := runtime.New(runtime.WithMiddlewareHandlers(&AuditMiddleware{}))
// Or: runtime.WithRouteMiddlewareHandlers("GET", "/users/{id}", &AuditMiddleware{})
```

Class-based middleware is registered at global or route scope and runs in
the same sequence as functional middleware. It is resolved once during
application construction and adds no reflection to the request path.
Instances are reused concurrently; typed-nil handlers fail at bootstrap.

## Functional middleware

```go
import (
    "github.com/kevenmiano/nestgo/httpadapter"
    "github.com/kevenmiano/nestgo/runtime"
)

app, err := runtime.New(
        runtime.WithMiddleware(func(ctx httpadapter.Context, next httpadapter.Next) error {
        ctx.SetHeader("X-Example", "pipeline")
        return next() // continue to the next middleware/guard/handler
    }),
)
```

A middleware has the signature `func(httpadapter.Context, httpadapter.Next) error`.
Calling `next()` is optional: a middleware can send its own response and
stop the chain. Calling it twice is an error. Middleware is executed in
registration order. You can return an error to the exception layer.

## Apply middleware to one route

```go
runtime.WithRouteMiddleware(
    "GET", "/users/{id}",
    func(ctx httpadapter.Context, next httpadapter.Next) error {
        ctx.SetHeader("X-Route", "users")
        return next()
    },
)
```

Use the **full registered route template**, including the controller
prefix, and an exact HTTP method. A wildcard path is not supported.

## Middleware vs guards

Middleware processes transport-level concerns; guards know which handler
is being executed and decide whether it may run. For authorization logic,
prefer [guards](./guards.md).

:::caution NestJS parity
NestJS supports module middleware via `MiddlewareConsumer.apply().forRoutes()`
and global pre-routing middleware. NestGo currently runs framework
middleware **only after route matching**: it does not execute for 404/405.
It does not implement `exclude()`, wildcard application, or
module-bound middleware decorators. Errors from middleware reach only global
[exception filters](./exception-filters.md).
:::

See the [runnable example](https://github.com/kevenmiano/nestgo/tree/main/examples/pipeline)
or the [request lifecycle](./request-pipeline.md).