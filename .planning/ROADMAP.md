# NestGo roadmap — NestJS-inspired, Go-native

**Source of truth:** GitHub Issues. **Project board:** https://github.com/users/kevenmiano/projects/2

This roadmap uses the [NestJS documentation](https://docs.nestjs.com/) as a **feature taxonomy**, not a promise of 1:1 TypeScript or runtime compatibility. NestGo APIs must remain idiomatic Go. Each feature requires a design discussion, tests, documentation, and CI verification before it can be marked Done.

## Milestones and dependency order

| Milestone | Goal | GitHub Issues | Priority | Entry gate |
| --- | --- | --- | --- | --- |
| M001 — Stabilization | CI, compiler coverage, runtime correctness, metadata contracts and GSD governance | #2–#7 | P0/P1 | Existing baseline |
| M002 — Core architecture | Module graph, dependency injection and controller routing | #10–#12 | P0 | M001 compiler/runtime contracts |
| M003 — Request pipeline | Middleware, execution context, guards, pipes, interceptors, filters, validation and versioning | #13–#15 | P0 | M002 routing and DI |
| M004 — Application capabilities | Configuration, logging, health, persistence, caching, security and OpenAPI | #16–#19 | P1 | M003 request pipeline |
| M005 — Distributed and advanced | Events, jobs, WebSockets, microservices, observability, reliability and GraphQL spike | #20–#23 | P2 | Stable core and adapters |
| M006 — Developer experience and release | CLI, incremental compilation, end-to-end samples, benchmarks and release gates | #24–#25 | P1 | Runs incrementally across milestones |

M004 and M006 can run in parallel after their relevant interfaces stabilize; M005 items should be independently scoped and may be postponed.

## Feature mapping

- **Core:** [Modules](https://docs.nestjs.com/modules), [Providers](https://docs.nestjs.com/providers), [Controllers](https://docs.nestjs.com/controllers), [Custom providers](https://docs.nestjs.com/fundamentals/custom-providers).
- **Request pipeline:** [Middleware](https://docs.nestjs.com/middleware), [Request lifecycle](https://docs.nestjs.com/faq/request-lifecycle), [Validation](https://docs.nestjs.com/techniques/validation).
- **Application and data:** [Configuration](https://docs.nestjs.com/techniques/configuration), [Caching](https://docs.nestjs.com/techniques/caching), [Authentication](https://docs.nestjs.com/security/authentication), [OpenAPI](https://docs.nestjs.com/openapi/introduction).
- **Distributed:** [Queues](https://docs.nestjs.com/techniques/queues), [Microservices](https://docs.nestjs.com/microservices/basics), [GraphQL](https://docs.nestjs.com/graphql/quick-start).
- **Quality and operations:** [Testing](https://docs.nestjs.com/fundamentals/testing), [Observability](https://docs.nestjs.com/observability/overview).

## GSD tracking

1. **Discuss:** clarify intended Go API, constraints, and ADR.
2. **Plan:** break an issue into verifiable tasks, identify dependencies.
3. **Execute:** feature branch and linked PR (`Refs #N` while in progress).
4. **Verify:** tests, generated output, docs, CI and measured coverage.
5. **Review:** reconcile implementation with acceptance criteria.
6. **Ship:** merge, close the issue and update `.planning/STATE.md`.

**Project automation:** `.github/workflows/project-sync.yml` adds newly opened issues to Project #2 **only if** the `PROJECTS_TOKEN` secret and workflow are operational. The connector cannot directly confirm or set GitHub Projects v2 fields. To import older issues, manually dispatch project-sync with backfill; note that it resets open items to Backlog. Priority and Milestone remain in issue bodies until Project field synchronization is implemented.

## Current iteration

M001 Phase 1: PR #8 was merged; PR #9 adds compiler tests, pending verification and review. Issues #2–#7 remain the stabilization baseline. Newly created #10–#25 are future work, not implemented features.
