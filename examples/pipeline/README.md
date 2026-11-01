# Request pipeline: runnable NestGo example

This example demonstrates **middleware → guards → pipes → handler**, with
exception filters on decorated controllers, while remaining independent of
the HTTP engine. It includes a compiler-free `/health` route.

The files `pipeline_generated.go` and `zz_modforge_metadata.go` are
**generated**, not handwritten. To regenerate:

```bash
go run ./cmd/modforge compile ./examples/pipeline
git diff --exit-code -- examples/pipeline
```

Run from the repository root:

```bash
export DEMO_API_KEY=local-demo-key
go run ./examples/pipeline/cmd/server
```

Try each HTTP outcome:

```bash
# Public code-first route; global middleware runs on this matched route too.
curl -i http://localhost:8080/health

# Guard rejects missing API key with 403.
curl -i http://localhost:8080/users/ada

# Successful route: guard, TrimPipe, RequiredPipe.
curl -i -H 'X-API-Key: local-demo-key' http://localhost:8080/users/ada

# Body DTO is trimmed before validation and returned with HTTP 201.
curl -i -X POST http://localhost:8080/users/ \
  -H 'X-API-Key: local-demo-key' -H 'Content-Type: application/json' \
  -d '{"name":"  Ada  "}'

# Missing name triggers 400; @UseFilters returns a JSON error.
curl -i -X POST http://localhost:8080/users/ \
  -H 'X-API-Key: local-demo-key' -H 'Content-Type: application/json' \
  -d '{"name":"  "}'
```

Run the integration tests against both default FastHTTP and net/http:

```bash
go test ./examples/pipeline/... -v
go test -race ./examples/pipeline/...
```

The `DEMO_API_KEY` header comparison is for demonstration only, not
constant-time token verification or a production authentication scheme.
Never check secrets into source control.

- [Middleware](../../website/docs/fundamentals/middleware.md)
- [Guards](../../website/docs/fundamentals/guards.md)
- [Pipes](../../website/docs/fundamentals/pipes.md)
- [Exception filters](../../website/docs/fundamentals/exception-filters.md)
