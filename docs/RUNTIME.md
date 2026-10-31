# HTTP runtime and dependency injection

The `runtime` package converts generated metadata into a standard `net/http` handler.

## Entry points

```go
handler, err := runtime.FromRegistry("shop")
if err != nil {
    log.Fatal(err)
}
log.Fatal(http.ListenAndServe(":8080", handler))
```

Import the generated `shop` package somewhere in your application so its `init()` registration runs before calling `FromRegistry`. The package name used by `FromRegistry` is the Go package name, not the module annotation.

Alternatively, call `runtime.New(packages ...metadata.Package)` with explicit metadata.

## Routing

Routes are registered with Go 1.23 `http.ServeMux` method patterns. The runtime combines the controller prefix and route path. For example, `@Controller("/users")` and `@Get("/{id}")` produce `GET /users/{id}`.

The runtime rejects duplicate method/path registrations. It uses reflection to find the handler method by its generated metadata name.

## Request parameters

| Decorator | Runtime behavior |
| --- | --- |
| `@Body()` | Decodes one JSON value into the method argument; body size is limited to 1 MiB |
| `@Param("id")` | Reads `r.PathValue("id")`; method argument must be `string` |
| `@Query("name")` | Reads `r.URL.Query().Get("name")`; method argument must be `string` |
| `@Headers("name")` | **Not implemented**; request returns an unsupported decorator error |

For undecorated `string` parameters, the runtime attempts to bind the first path variable found in the route pattern. Prefer explicit decorators for clarity.

## Responses and errors

- A handler must return one value or a value plus an `error`.
- Successful results are serialized as JSON.
- The default status is `201 Created` for POST and `200 OK` for other methods.
- `@HttpCode` overrides the default status.
- Invalid JSON bodies return HTTP 400.
- Non-nil handler errors return HTTP 500.
- Unsupported parameter decorators or parameter types can produce HTTP 500.

These are implementation behaviors, not a finalized public API. There is currently no built-in error mapping, validation pipeline or response interception.

## Dependency injection

Provider types registered by `@Injectable()` are instantiated as pointers to structs. The runtime then injects matching provider instances into exported pointer or interface fields on providers and controllers.

- Pointer fields match registered provider types exactly.
- Interface fields match a provider that implements the interface.
- Multiple matching providers for one interface are rejected as ambiguous.
- Duplicate provider types are rejected.
- There is no constructor injection, request scope, lifecycle hook or automatic cycle detection.

The metadata registry is process-global and protected by a mutex. Registering the same package name twice causes a panic.

## Security and production considerations

The runtime is an experimental prototype. It does not include authentication, authorization, request validation, rate limiting, structured error responses, observability or production hardening. Do not deploy untrusted public workloads without additional application-level safeguards.
