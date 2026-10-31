# Phase 1 — Compiler test coverage

**Tracking:** [Issue #4](https://github.com/kevenmiano/nestgo/issues/4) · **Milestone:** M001

## GSD iteration

- **Discuss:** Focus on parser diagnostics, invalid decorator usage, multi-file generation and parameter binding.
- **Plan:** Add table-driven negative cases and deterministic output assertions without changing compiler behavior.
- **Execute:** Add `compiler/coverage_test.go`.
- **Verify:** Run `go test ./...`, `go vet ./...` and `go test ./compiler -coverprofile=coverage.out` in CI or locally; record the coverage percentage after execution.
- **Review:** Confirm the diagnostic expectations match intended language semantics and investigate any failures.
- **Ship:** Merge only after checks pass; close issue #4 only when all acceptance criteria, including coverage baseline, are met.

## Tests added

- Invalid, dangling, inline, and unclosed decorators with file and line diagnostics.
- Illegal decorator targets and invalid parameter arguments.
- Invalid HTTP statuses, invalid route arguments, duplicate route methods and orphan routes.
- Multi-file package generation, repeated-output byte equality and mixed-package rejection.
- Parameter binding indices, kinds and names for `@Param`, `@Query`, `@Headers` and `@Body`.

## Known gaps

- This is an incremental coverage expansion, not a complete positive/negative matrix for every supported decorator.
- No golden-file snapshot is included in this slice; deterministic output is asserted by comparing repeated generation and by the existing CI example regeneration check.
- Coverage percentage is **pending actual test execution**. Do not report an estimated baseline.
