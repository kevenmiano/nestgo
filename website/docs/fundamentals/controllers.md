---
title: Controllers
---
# Controllers

Controllers group HTTP handlers. NestGo supports `@Controller`,
`@Get`, `@Post`, `@Put`, `@Patch`, `@Delete`,
`@HttpCode`, `@Header` and `@Redirect`.

```go
import "github.com/kevenmiano/nestgo/http"

@Controller("/users")
type UsersController struct{}

@Get("/{id}")
@Header("Cache-Control", "no-store")
func (c *UsersController) FindOne(@Param("id") id string) http.Response {
    return http.OK(map[string]string{"id":id})
}

@Post("/")
func (c *UsersController) Create(@Body() dto UserDTO) http.Response {
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
