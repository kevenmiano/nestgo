---
title: Roadmap
---
# NestGo roadmap

**Current status:** Core v1.0 is a production-readiness target, not yet certified for production.

The [GitHub Project #2](https://github.com/users/kevenmiano/projects/2), [repository roadmap](https://github.com/kevenmiano/nestgo/blob/main/.planning/ROADMAP.md) and [release gate #70](https://github.com/kevenmiano/nestgo/issues/70) define the supported release contract.

## Next steps to qualify Core v1.0

| Sequence | Focus | Tracking |
| --- | --- | --- |
| 1 | Dependency-aware lifecycle, safe rollback and shutdown | [#119](https://github.com/kevenmiano/nestgo/issues/119) |
| 2 | Middleware, guards, validation pipes and filters | [PR #122](https://github.com/kevenmiano/nestgo/pull/122) |
| 3 | Unexpected error/panic redaction and active-request SIGTERM | [#112](https://github.com/kevenmiano/nestgo/issues/112), [PR #151](https://github.com/kevenmiano/nestgo/pull/151) |
| 4 | Config validation, health/readiness and logging | [#16](https://github.com/kevenmiano/nestgo/issues/16) |
| 5 | Exact-SHA E2E, race, security, benchmarks and tagged consumer | [#70](https://github.com/kevenmiano/nestgo/issues/70) |

The supported **HTTP core** uses one `runtime.New(...Option)` API, pluggable FastHTTP/net/http adapters, optional compiled modules, explicit singleton DI and typed HTTP responses.

**Optional integrations after the core gate:** database/repositories, full DTO transformation strategies, OpenAPI, caching, queues, other transports, GraphQL and extended DI scopes. They do not gate Core v1.0 and require separate acceptance before production support.

The detailed acceptance evidence checklist lives in [docs/RELEASE_ACCEPTANCE.md](https://github.com/kevenmiano/nestgo/blob/main/docs/RELEASE_ACCEPTANCE.md).

**Workflow:** Discuss → Plan → Execute → Verify → Review → Ship.

:::info Release policy
Passing individual PR tests does not certify NestGo for production. Publish Core v1.0 only after every P0 gate has passing evidence on the approved immutable release SHA.
:::
