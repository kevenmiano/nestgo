---
title: Providers and modules
---
# Providers and modules

NestGo supports `@Injectable`, `@Controller` and declarative `@Module` metadata. Explicit module contracts control the visibility of registered providers.

```go
@Injectable()
type UsersService struct{}

@Module({providers: [UsersService], exports: [UsersService]})
type UsersModule struct{}

@Module({imports: [UsersModule], controllers: [UsersController]})
type AppModule struct{}
```

A provider is available within its declaring module. A consuming module can inject it when it imports a module that exports that provider. External module references can use qualified Go types in `imports` when their package aliases are declared in the `.mgo` source.

At bootstrap, the runtime validates missing imports, duplicate ownership, invalid exports, import cycles and unavailable required dependencies. `runtime.New()` discovers the registered metadata packages of imported modules transitively. Those Go packages must still be linked into the binary so their `init` registration runs.

:::caution Experimental limitations
Field injection is supported; constructor injection, request-scoped providers, complete cross-package compiler type validation and advanced lifecycle hooks are not stable features. An import does not automatically export all providers.
:::

See the [architecture guide](./architecture.md) and [roadmap](../roadmap.md).
