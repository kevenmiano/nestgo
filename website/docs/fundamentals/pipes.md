---
title: Pipes
---

# Pipes

Pipes run **after guards and before a controller method**. In the spirit of
[NestJS pipes](https://docs.nestjs.com/pipes), they can validate or
transform incoming parameters. NestGo keeps a Go-typed contract:

```go
type Pipe interface {
    Transform(value any, metadata httpadapter.ArgumentMetadata) (any, error)
}
```

## Create a validation pipe

In a `.mgo` source file:

```go
@Injectable()
type RequiredPipe struct{}

func (*RequiredPipe) Transform(v any, meta httpadapter.ArgumentMetadata) (any, error) {
    if meta.Kind == "Param" && v.(string) == "" {
        return nil, httpadapter.NewHTTPException(400, "id is required")
    }
    return v, nil
}
```

`ArgumentMetadata` includes the argument's `Index`, `Kind` (Body,
Param or Query), `Name`, and `reflect.Type`. To transform a DTO, return
the same concrete Go type. An incompatible replacement yields HTTP 400.

## Controller and method bindings

```go
@UsePipes(TrimPipe)
@Controller("/users")
type UsersController struct{}

@UsePipes(RequiredPipe)
@Get("/{id}")
func (*UsersController) FindOne(@Param("id") id string) string { return id }
```

Register the pipe classes as `@Injectable()` providers in the module.
For an individual bound value, the global pipe sequence runs before the
controller pipe sequence, then method pipes. `@UsePipes([A, B])` is
supported and preserves declaration order.

## Global pipes

```go
runtime.New(runtime.WithPipes(&AuditPipe{}))
```

For compiler-generated handlers, `@Body`, `@Param` and `@Query` all
pass through pipes in both static and reflective paths.

For **plain Go** `runtime.WithRoutes(...)` handlers, pipe invocation is
explicit. Use the helpers on the wrapped context:

```go
runtime.Route{Method: "GET", Path: "/users/{id}",
    Handler: func(ctx runtime.Context) error {
        id, _, err := httpadapter.BindParam(ctx, "id", 0)
        if err != nil { return err }
        return ctx.Write([]byte(id))
    },
}
```

The same package also provides `BindQuery` and `DecodeBodyAt[T]`.

:::caution NestJS parity
There is no built-in `ValidationPipe`, `ParseIntPipe`, class-transformer
bridge, or parameter-level `@Param("id", PipeClass)` binding yet.
NestJS processes multiple route parameters in reverse parameter order;
NestGo currently processes decorated parameters in **handler declaration
order**. Do not assume these orders are equivalent.
:::

See [exception filters](./exception-filters.md) and
[the full example](https://github.com/kevenmiano/nestgo/tree/main/examples/pipeline).