# Development guide

This guide describes the contributor workflow for NestGo Compiler v3.

## Prerequisites

Install Go 1.27 or later and clone the repository.

## Build and test

```bash
go test ./...
go vet ./...
go run ./cmd/nestgo compile ./examples/shop
```

To check formatting:

```bash
gofmt -l ./compiler ./metadata ./runtime ./cmd ./examples
```

Re-run the compiler after changing `.mgo` files. Generate CLI reference with `go run ./cmd/nestgo-docs ./website/docs/cli/reference`. Inspect the generated `_generated.go` and `zz_modforge_metadata.go` files before committing them.

## Source layout

- `compiler/`: decorator parsing, validation and source generation.
- `metadata/`: registration data structures.
- `runtime/`: HTTP handling and dependency injection.
- `internal/cli/`: Cobra command tree and CLI tests.\n- `cmd/nestgo/`: CLI entry point.\n- `cmd/nestgo-docs/`: generated CLI reference.
- `examples/shop/`: runnable source example and tests.

## Commit conventions

Use [Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/):

```text
<type>(<scope>): <imperative summary>
```

Examples:

```text
feat(compiler): support controller route metadata
fix(runtime): handle invalid JSON request bodies
test(compiler): cover malformed decorator arguments
docs: document compiler architecture
refactor(metadata): simplify registry types
chore: remove legacy generated binaries
```

Recommended types: `feat`, `fix`, `docs`, `test`, `refactor`, `chore`, `build`, and `ci`.

Use English for commit messages, PR titles, code comments, and documentation. Keep each commit focused on a single logical change.

## Pull request checklist

- [ ] The change has a clear English description.
- [ ] New behavior has tests.
- [ ] `go test ./...` passes.
- [ ] `go vet ./...` passes.
- [ ] Generated code is consistent with the `.mgo` source.
- [ ] Documentation is updated when public behavior changes.
- [ ] Experimental limitations are stated explicitly.

## Known gaps

NestGo is not yet production-ready. There is no stable LSP, comprehensive dependency-injection system, or full NestJS compatibility. Avoid describing unsupported capabilities as implemented.
