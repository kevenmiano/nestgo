---
title: Exception filters
---

# Exception filters

Exception filters customize the HTTP response when middleware, guards,
pipes or handlers return an error. The design is informed by
[NestJS exception filters](https://docs.nestjs.com/exception-filters).

## Define a filter

```go
@Injectable()
type BadRequestFilter struct{}

func (*BadRequestFilter) Catch(err error, exec httpadapter.ExecutionContext) (bool, error) {
    if httpadapter.StatusOf(err) != 400 {
        return false, nil // let another filter or the default responder handle it
    }
    exec.Request.Status(400)
    exec.Request.SetHeader("Content-Type", "application/json")
    return true, exec.Request.Write([]byte(`{"message":"invalid request"}`))
}
```

The `Catch` result is `(handled bool, writeErr error)`. If handled,
stop. If not handled, NestGo tries the next filter. Unlike NestJS's
single-filter selection behavior, NestGo deliberately supports
**fall-through** when a filter returns `false`.

## Method, controller and global registration

```go
@UseFilters(BadRequestFilter)
@Post("/")
func (*UsersController) Create(@Body() dto CreateUserDTO) (CreateUserDTO, error) {
    return dto, nil
}
```

You can also decorate a controller with `@UseFilters(BadRequestFilter)`
or register global instances with `runtime.WithFilters(&GlobalFilter{})`.
Filters resolve in **method → controller → global** order.

## Typed HTTP exceptions and safe defaults

```go
return nil, httpadapter.NewHTTPException(400, "name is required")
```

A plain `error` results in a **500** with a redacted response body.
`HTTPException` 4xx errors may expose their declared message. Errors
from middleware are only offered to **global** filters because middleware
runs outside method/controller exception scopes.

:::warning Security
Filters write raw responses and can accidentally leak internal error
details. Do not send `err.Error()` to clients for unexpected 5xx errors.
Never assume a custom filter is safe solely because it is registered.
:::

:::caution NestJS parity
NestGo does not currently support `@Catch(ExceptionType)` matching,
NestJS `ArgumentsHost`, or `APP_FILTER` injection tokens. Filter
selection and response formatting are implemented with Go interfaces,
not TypeScript exception classes.
:::

Read the [request lifecycle](./request-pipeline.md) or run the
[exception filter sample](https://github.com/kevenmiano/nestgo/tree/feat/http-request-pipeline/examples/pipeline).