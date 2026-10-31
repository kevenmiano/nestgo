---
title: Providers and modules
---
# Providers and modules

NestGo recognizes `@Injectable` and `@Module` annotations and currently supports basic reflection-based field injection.

```go
@Injectable()
type UsersService struct{}

@Module("users")
type UsersModule struct{}
```

:::caution Not yet implemented
Constructor injection, provider scopes, module imports/exports and lifecycle hooks are roadmap items, not stable features.
:::

See [the core architecture roadmap](../roadmap.md).
