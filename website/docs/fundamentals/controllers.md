---
title: Controllers
---
# Controllers

Controllers group HTTP handlers. NestGo currently recognizes `@Controller`, `@Get`, `@Post`, `@Put`, `@Patch`, `@Delete` and `@HttpCode`.

```go
@Controller("/users")
type UsersController struct{}

@Get("/{id}")
func (c *UsersController) FindOne(id string) string {
    return id
}
```

Parameter decorators include `@Body`, `@Param` and `@Query`. `@Headers` is recognized by the compiler but runtime binding is not implemented.

See the [runtime reference](https://github.com/kevenmiano/nestgo/blob/main/docs/RUNTIME.md) for current behavior and limitations.
