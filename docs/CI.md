# Phase 1 — Go CI quality gates

**Tracking:** [Issue #3](https://github.com/kevenmiano/nestgo/issues/3) · **GSD phase:** 1 · **Milestone:** M001

## Objective

Make compiler and runtime changes verifiable in GitHub Actions before merge.

## Implementation

The `Go quality gates` workflow runs on pull requests to `main`, pushes to `main`, and manual dispatch. It uses the Go version from `go.mod` and performs:

1. `go test ./...` — unit/integration tests.
2. `go vet ./...` — static analysis.
3. `gofmt -l` — formatting verification for Go files changed relative to the PR base or prior push.
4. `go run ./cmd/modforge compile ./examples/shop` — regenerate the checked-in example.
5. `git diff --exit-code -- examples/shop` and untracked-file checks — reject stale or missing generated output.

## GSD verification gates

- **Plan:** CI scope and known limitations recorded here.
- **Execute:** Workflow added in a feature branch and linked to issue #3.
- **Verify:** GitHub Actions must pass on the PR; failures must be corrected and rerun.
- **Review:** A reviewer confirms CI results and the generated-code diff.
- **Ship:** Merge only after successful checks. Configure branch protection separately; workflow files cannot enforce required checks by themselves.

## Known limitations

- Formatting currently applies only to **changed** Go files. Existing files may not be gofmt-clean; a full cleanup should be tracked separately.
- This workflow checks deterministic example regeneration against committed generated files; it does not establish a general compiler determinism proof.
- No GitHub Actions execution or branch protection setting is asserted as successful until verified in GitHub.
