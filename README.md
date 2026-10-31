# NestGo Compiler v3

Experimental Go framework with NestJS-style decorators in `.mgo` files, a compiler and HTTP runtime.

## Run

```bash
go run ./cmd/modforge compile ./examples/shop
go test ./...
go vet ./...
```

Decorators: `@Controller`, `@Get`, `@Post`, `@Put`, `@Patch`, `@Delete`, `@HttpCode`, `@Body`, `@Param`, `@Query`, `@Injectable`, `@Module`.

The compiler produces ordinary Go files and a metadata registry. The runtime registers routes using Go 1.23 ServeMux, binds parameters, decodes JSON and injects exported provider fields.

**Experimental:** DI is basic; no dynamic modules, constructor injection, scopes, LSP, guards, pipes, or interceptors. Compile `.mgo` files before running Go tooling.
