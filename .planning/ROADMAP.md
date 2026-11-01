# NestGo roadmap — minimal production-ready HTTP framework

**Source of truth:** [Issues](https://github.com/kevenmiano/nestgo/issues) · [Project #2](https://github.com/users/kevenmiano/projects/2) · [Production gate #70](https://github.com/kevenmiano/nestgo/issues/70)

## Supported MVP target

A deployable, maintainable Go HTTP framework with deterministic `.mgo` compilation; controllers; explicit module imports/exports; singleton field injection; JSON request/response; safe failures; pluggable HTTP adapter contracts; graceful shutdown; basic health/logging; verified tagged distribution.

**Not MVP requirements:** constructor DI, request/transient scopes, guards/interceptors framework, GraphQL, OpenAPI generation, ORM, caching, queues, WebSockets, microservices, watch mode. These must not hold the minimal production release hostage.

## Strict implementation order

| Order | Gate | Priority | Issues | Acceptance evidence |
| --- | --- | --- | --- | --- |
| 0 | Correctness and metadata | P0 | #4 #5 #6 #35 #36 #53 | Compiler goldens, deterministic diagnostics, no DI panics/ambiguity |
| 1 | Module/DI integration | P0 | #10 #12 #52; singleton subset of #11 | Real multi-package compiler-to-runtime E2E, import/export isolation |
| 2 | HTTP runtime safety | P0 | #48 #12 #13 (minimal middleware subset) #56 | Adapter conformance, cancellation, errors, body limits, timeouts, graceful shutdown, race |
| 3 | Minimum operations | P0 | #16 #50; safe-default subset of #18 | Structured logging, liveness/readiness, recovery, error redaction |
| 4 | Release qualification | P0 | #25 #37 #41 #55 #70 | Reproducible benchmarks, full CI, external installation, integrity-checked immutable version and docs deployment |
| 5 | Post-MVP extension | P1 | #7 #11 #14 #15 #19 #24 #38 #54 | Expanded lifecycle, validators, docs, tooling |
| 6 | Optional integrations | P2 | #17 #18 #20 #21 #22 #23 | Data, security integrations, distributed capabilities, GraphQL |

M001–M006 remain the existing Project Roadmap taxonomy, not mandatory sequential release phases. Each broad issue can have a bounded P0 slice and defer the rest without claiming the entire issue completed.

## Production gate: #70

No production-ready claim before **every** #70 acceptance criterion is verified: deterministic compilation, fail-closed DI, real-listener HTTP adapter contract, integration and negative tests, `go test -race ./...`, `go vet ./...`, formatting, generated output checks, representative fuzz testing, startup/per-request benchmarks, clean shutdown, minimum operational safety, immutable semver tags and external module checksum verification.

Production-supported adapters must demonstrate parity; otherwise identify the unsupported adapter explicitly as experimental.

## Priority semantics

- **P0:** release blocker (correctness, request/DI safety, tests and packaging).
- **P1:** useful capabilities safely deferred until after minimal production release.
- **P2:** integrations, distributed features and exploratory work.

Follow **Discuss → Plan → Execute → Verify → Review → Ship** per issue. All merged PRs require checks on exact head SHA; an issue stays open if acceptance criteria are not fully met. Never reuse or move `v0.1.0`; versioned Go module checksums are immutable.

See [Semantic releases](../docs/SEMANTIC_RELEASES.md) and [distribution](../docs/DISTRIBUTION.md).
