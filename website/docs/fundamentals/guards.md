---
title: Guards
---

# Guards

Guards run **after middleware and before pipes/handlers**. As in
[NestJS guards](https://docs.nestjs.com/guards), they are intended for
authorization rather than raw request preprocessing.

## Create a guard

Create this inside a `.mgo` file:

```go
import (
    "os"
    "github.com/kevenmiano/nestgo/httpadapter"
)

@Injectable()
type ApiKeyGuard struct{}

func (*ApiKeyGuard) CanActivate(ctx httpadapter.ExecutionContext) (bool, error) {
    key := os.Getenv("DEMO_API_KEY")
    return key != "" && ctx.Request.Header("X-API-Key") == key, nil
}
```

`CanActivate` returns `(bool, error)`. Returning `false, nil` stops
the request with **403 Forbidden**. Errors are routed to the appropriate
exception filter. `ExecutionContext` exposes `Request` (the engine-neutral
HTTP context), `Controller`, and `Handler` (method name). For
code-first routes, `Controller` is nil and `Handler` is the route key.

:::warning Demonstration only
An API-key equality check is not a complete authentication system.
Use appropriate credential verification and secret management in production.
:::

The compiler emits `var _ httpadapter.Guard = (*ApiKeyGuard)(nil)`
automatically for decorated guard classes. Incorrect method signatures fail
at Go compile time, before the server starts.

## Apply guards

Decorate an entire controller or a particular method:

```go
@UseGuards(ApiKeyGuard)
@Controller("/users")
type UsersController struct{}

@UseGuards(RoleGuard)
@Get("/{id}")
func (*UsersController) FindOne(@Param("id") id string) string {
    return id
}
```

Both `ApiKeyGuard` and `RoleGuard` must be local `@Injectable()`
providers and included in the module's `providers: [...]` list.
Import/export visibility is validated at bootstrap. Multiple guards
can be specified as `@UseGuards([ApiKeyGuard, RoleGuard])`.

## Global guards

Use functional options for an application-wide guard instance:

```go
app, err := runtime.New(
        runtime.WithGuards(&MetricsGuard{}),
)
```

Global, controller and route guards run in that order. No guards are
instantiated per request: provider instances are resolved at bootstrap
and must be concurrency-safe.

:::caution Scope differences
NestGo does not yet have NestJS's `APP_GUARD` provider token, request-scoped
guards, reflection metadata helpers, or automatic JWT/roles decorators.
The current compiler accepts local provider *types*, not arbitrary
expressions or instances in `@UseGuards`.
:::

Continue with [pipes](./pipes.md) and
[the complete example](https://github.com/kevenmiano/nestgo/tree/main/examples/pipeline).