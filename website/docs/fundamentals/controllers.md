---
title: Controllers
---

# Controllers

Controllers map requests to handler methods. NestGo follows the
[NestJS controller model](https://docs.nestjs.com/controllers), using an
optional source-to-source compiler for decorated `.mgo` files.

```go
@Controller("/users")
type UsersController struct{}

@Get("/{id}")
func (c *UsersController) FindOne(@Param("id") id string) string {
    return id
}
```

Unlike JavaScript/TypeScript, **standard Go does not support these
decorators**. Run `nestgo compile ./myapp` or
`go run ./cmd/modforge compile ./myapp` from the repository root to
generate ordinary `.go` sources and routing metadata.

Supported HTTP methods: `@Get`, `@Post`, `@Put`, `@Patch`,
`@Delete`, and `@HttpCode`. Supported parameter bindings include
`@Body`, `@Param` and `@Query`. Although `@Headers` is recognized
by the parser, it is not yet a supported binding contract.

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
