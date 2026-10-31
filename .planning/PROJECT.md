# NestGo — GSD Core Project

## Vision

Build a Go-first, source-to-source compiler and HTTP runtime inspired by NestJS decorators while preserving idiomatic Go output.

## Current baseline

NestGo Compiler v3 already provides `.mgo` lexing, AST parsing, Go code generation, metadata registration, a basic reflection-based HTTP runtime, an example package and tests. The implementation is experimental.

## Operating model

Use GSD Core's **discuss → plan → execute → verify → review → ship** lifecycle. GitHub Issues are the work intake and audit trail; GitHub Projects is the intended visual board. GSD planning files hold project context and phase decisions.

Every implementation task must have a GitHub issue, a phase, acceptance criteria, a linked PR, and verification evidence. Unplanned changes must first be captured as issues.

## Links

- Repository: https://github.com/kevenmiano/nestgo
- Issue tracker: https://github.com/kevenmiano/nestgo/issues
- Governance setup: https://github.com/kevenmiano/nestgo/issues/2

## Non-goals for the current milestone

Full NestJS parity, production deployment guarantees, and an LSP.
