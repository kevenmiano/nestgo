# Contributing to NestGo

Thank you for helping improve NestGo.

Please open an issue or pull request with a focused description of the proposed change. Use English for all repository-facing documentation, issue descriptions, and commit messages.

## Local validation

```bash
go test ./...
go vet ./...
```

For changes to the `.mgo` syntax or compiler, add parser tests and verify generated Go output.

## Commit style

Follow Conventional Commits, for example `feat(compiler): validate route decorators`, `fix(runtime): reject invalid request bodies`, or `docs: clarify development setup`.

## Compatibility

Preserve standard Go compatibility for generated files. Treat compiler output and runtime APIs as experimental until an explicit stable release is announced.
