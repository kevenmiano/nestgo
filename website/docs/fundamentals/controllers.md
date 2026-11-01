---
title: Controllers
---
# Controllers

Controllers group HTTP handlers. NestGo's decorators are compiled from `.mgo`, not supported by the Go compiler directly. NestGo supports `@Controller`,
`@Get`, `@Post`, `@Put`, `@Patch`, `@Delete`,
`@HttpCode`, `@Header` and `@Redirect`.

```go
import "github.com/kevenmiano/nestgo/http"

@Controller("/users")
type UsersController struct{}

@Get("/{id}")
@Header("Cache-Control", "no-store")
func (c *UsersController) FindOne(@Param("id") id string) http.Response[map[string]string] {
    return http.OK(map[string]string{"id":id})
}

@Post("/")
func (c *UsersController) Create(@Body() dto UserDTO) http.Response[map[string]string] {
    return http.Created(dto)
}
```

`@Body`, `@Param`, `@Query`, `@Headers`, `@Res` and `@Response`
bind parameters. `@Res()` injects a transport-independent
`*http.Writer`; `@Res({passthrough:true})` allows a handler
to set response metadata while returning a value for automatic serialization.

The same controllers run with native fasthttp (default), net/http or any
compatible HTTP adapter.

For status codes, headers, redirects, cookies, typed HTTP exceptions,
explicit responses and examples, see the
[HTTP responses guide](https://github.com/kevenmiano/nestgo/blob/main/docs/HTTP_RESPONSES.md).

## Route pipeline

Use class-based decorators on controllers and methods:

```go
@UseGuards(ApiKeyGuard)
@UsePipes(TrimPipe)
@Controller("/users")
type UsersController struct{}

@UsePipes(RequiredPipe)
@UseFilters(BadRequestFilter)
@Post("/")
func (c *UsersController) Create(@Body() dto CreateUserDTO) (CreateUserDTO, error) {
    return dto, nil
}
```

These types must implement the relevant `httpadapter` Go interfaces,
carry `@Injectable()` and belong to the module's provider list.
Components are constructed once at bootstrap.

Read [middleware](./middleware.md), [guards](./guards.md),
[pipes](./pipes.md) and [exception filters](./exception-filters.md),
or follow the [request lifecycle](./request-pipeline.md).

For a complete example, see
[examples/pipeline](https://github.com/kevenmiano/nestgo/tree/main/examples/pipeline).
