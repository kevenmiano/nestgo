# Project state

**Project:** NestGo Compiler v3
**Milestone:** M001 — Stabilization
**Current phase:** 1 — CI and compiler coverage
**Current status:** Phase 1 CI (#8) and compiler coverage (#9) merged; GitHub Project Roadmap synchronization repaired in PR #28; automatic backfill and post-merge PR status sync passed.

## Verified repository facts

- Compiler v3 is on `main`.
- GitHub Issues #2–#7 define the initial milestone backlog.
- GitHub Project #2 exists: https://github.com/users/kevenmiano/projects/2.
- PR #28 fixed the `Milestone` type collision by writing to a custom single-select `Roadmap` field, leaving GitHub's native Milestone unchanged. The automatic push backfill succeeded (run 41238808688); logs confirm issues #10–#25 synchronized with Roadmap/Priority metadata. The PR merge event also succeeded (run 41238809091), setting issue #2 to In Progress (not Done) for its plain `Refs #2` reference. Board UI layout has not been independently verified.
- A repository-level GSD-style planning scaffold exists; **GSD Core installation/onboarding has not been verified**.

## Phase 1 execution log

- **Discuss:** Scope from issue #3: Go 1.23 checks, formatting, deterministic compiler output.
- **Plan:** Introduce a single PR quality workflow and document verification and limitations.
- **Execute:** Add `.github/workflows/go-quality.yml` on branch `ci/phase-1-quality-gates`.
- **Verify:** PR #8 Go quality gates passed in GitHub Actions run 41235026178.
- **Review:** PR #8 merged after generated fixtures were aligned.
- **Ship:** PR #8 squash-merged as bb993f3583ef12c9595b34ebfef71926c2802e7d.

## Next actions

1. Review the actual Project #2 Kanban and Roadmap views after successful backfill; verify field visibility and grouping in the UI (views are not created by Actions).
2. Complete the remaining issue #4 compiler coverage acceptance criteria, including a measured baseline.
3. Keep issue statuses accurate: `Refs #N` means partial work, not completion.
4. Complete issue #2 only after actual GSD Core onboarding is performed and verified.
5. Keep GitHub Issues as the work-item source of truth; link PRs using `Closes #N` and update this file at each phase gate.

## Decisions and limitations

- English for repository content and Conventional Commits.
- CI checks `gofmt` for **changed Go files**, not all legacy Go files, to avoid blocking unrelated PRs on pre-existing formatting debt. Full-repository formatting remains outstanding.
- A PR cannot be marked Done merely because code was pushed: tests, review, and merge are required.
- GitHub Project fields and synchronization success cannot be inferred from workflow creation alone.

## Phase 1 iteration 2 — compiler coverage

- **Discuss/Plan:** Issue #4 defines decorator diagnostics, binding and deterministic output coverage.
- **Execute:** PR #9 adds table-driven tests and `docs/COMPILER_TESTING.md`.
- **Verify:** Go quality gates passed on commit 72f7464f5dd98d19c43020f1d05c46810cdddd5d before merge; measured coverage baseline remains outstanding.
- **Review/Ship:** PR #9 squash-merged as 65a11c37c2a83dafeb0109205f5cecdb9e9485e8. Issue #4 remains open until the full acceptance criteria are met.

## Project synchronization incident — GSD verification

- **Discuss:** Runs 41237925067, 41237971244 and 41237993520 failed with `Milestone must be a single-select field`.
- **Plan/Execute:** PR #28 renamed the custom destination to `Roadmap`, added metadata/reference policy tests, and prohibited `Refs #N` from completing open issues.
- **Verify:** PR checks `Project sync policy tests` (41238733802) and Go quality gates (41238733739) passed; post-merge push backfill (41238808688) and PR-target issue status sync (41238809091) passed.
- **Review/Ship:** Squash merged PR #28 as `4cb788e40d1e52e0d2535ae2caff67ce7c441bc4`. Confirmed the workflow log for #10–#25 and `#2 status=In Progress`. UI view configuration remains manual.

## Follow-up — existing Project Status options

- A PR opened after the Roadmap repair (run 41238977312) exposed a separate incompatible assumption: the board has no `In Review` status option.
- Correction: reuse safe existing statuses instead of requiring `In Review`. The status mapping is tested against `Todo`, `In Progress` and `Done`, and logs missing options without mutating the Project field configuration.
- Verify that both opening and merging a PR now complete the synchronization workflow; never mark `Refs #N` as Done.

## GitHub Project Status compatibility — follow-up verification

- **Failure:** run [41238977312](https://github.com/kevenmiano/nestgo/actions/runs/41238977312) failed on PR open with `Missing Status option In Review`.
- **Root cause:** Project #2's actual Status options are `Todo`, `In Progress` and `Done`, rather than the workflow's former `In Review` assumption.
- **Execute/Review:** PR #30 added safe Status aliases and regression coverage without changing the Project's field options.
- **Verify:** Project policy tests [41239221015](https://github.com/kevenmiano/nestgo/actions/runs/41239221015) and Go quality gates [41239220984](https://github.com/kevenmiano/nestgo/actions/runs/41239220984) passed. Post-merge backfill [41239288817](https://github.com/kevenmiano/nestgo/actions/runs/41239288817) and merged-PR status reconciliation [41239289370](https://github.com/kevenmiano/nestgo/actions/runs/41239289370) passed. The PR merge reconciled issue #2 to In Progress, not Done.
- **Ship:** PR #30 squash-merged as `bcc27ad7e7c9017467816e2c13545e333408b37b`. Kanban/Roadmap view configuration remains a manual review item.

## Go distribution and release automation — GSD workstream

- **Discuss:** Current root module exposes `compiler`, `runtime`, `metadata` as public packages; Go `internal/` would prevent consumer imports. `go install` of a command does not add module requirements to applications.
- **Plan:** Preserve one versioned root module, add an official `nestgo` command, and separate release workflows for docs, Go packages and framework binaries.
- **Execute:** Add `cmd/nestgo`, package and framework release workflows, tag verification, installation guides and CI verification.
- **Verify:** PR checks must pass before merging. No successful tagged CLI installation, release artifact upload or GitHub Pages deployment is claimed until a real GitHub Release triggers the corresponding workflows.
- **Review/Ship:** Pending PR review, CI and first official release. GSD issue #25 remains open.

## GoReleaser migration — GSD execution

- **Discuss:** Custom platform loops, sha256sum and gh release upload duplicate GoReleaser OSS capabilities.
- **Plan:** Preserve independent docs, libraries and framework workflows; replace only CLI packaging/upload with GoReleaser v2, keep tag ancestry and real go-install smoke checks.
- **Execute:** Add `.goreleaser.yaml`, GoReleaser GitHub Action v7, PR snapshot validation, [ADR 0001](../docs/adr/0001-goreleaser-distribution.md), and distribution guide updates.
- **Verify:** Run `goreleaser check` and a no-publish cross-platform snapshot in CI. A live tagged GitHub Release is still required to confirm publication and consumer installs.
- **Review/Ship:** Pending CI and PR review; issue #25 remains in progress.

## Deterministic diagnostics and startup instrumentation — GSD

- **Discuss:** Inspired by NestJS circular dependency diagnostics, prioritize a provider graph and measured Go-native startup performance; avoid claiming that the existing metadata contains a complete module graph.
- **Plan:** Issue #36 dependency SCC/finding diagnostics; #37 warm-startup benchmarks; #38 unified offline visual report.
- **Execute:** `diagnostics/` package, JSON/DOT and standalone SVG HTML exports, `nestgo diagnose -example`, CI artifacts and reproducible instructions.
- **Verify:** CI must test graph ordering, cycles, missing/ambiguous provider findings and the sample `runtime.New` timings. Measured latency is nondeterministic, so no fake baselines or absolute-ms gating.
- **Review/Ship:** This initial iteration does not complete cold process startup profiling, module-level graphing or large-scale benchmark matrices; issues #36–#38 remain open until those acceptance criteria are met.

## Diagnostics iteration — cold subprocess and scale (GSD)

- **Discuss:** Initial report only measured warm `runtime.New` on a 3-provider example. A full process launch and provider scaling need distinct, honestly labeled measurements.
- **Plan:** Extend the optional report with real fresh-process launch-to-exit samples and separate stable synthetic provider fixtures (1/10/100); preserve a deterministic graph JSON without timestamps.
- **Execute:** `diagnostics/process.go`, CLI `-cold-samples` and `-scale`, HTML comparison table, CI verification and tests.
- **Verify:** PR CI must show functional cold subprocess execution, scale fixture tests, graph correctness and no invented performance thresholds.
- **Review/Ship:** Real application HTTP readiness, module import/export analysis and representative benchmark matrix remain pending on #36–#38.

## Real-application diagnostics bridge — GSD

- **Discuss:** The prior CLI only accepted `-example`; a separately installed CLI cannot discover the consumer's Go type metadata without linking the application package.
- **Plan:** Add `nestgo diagnose -app <importable-package>` using a temporary Go runner that imports generated registration, with no application source changes.
- **Execute:** Resolve the target via `go list`, compile an ephemeral runner, inspect real `metadata.All()`, measure warm startup and linked-process cold startup, export JSON/DOT/HTML.
- **Verify:** CI diagnoses `./examples/shop` and checks that `UsersService` appears in the graph; Go quality and Docusaurus checks gate merging.
- **Review/Ship:** Module import/export graph edges remain unavailable in the current metadata contract; track in #10, #11 and #36. The import-time execution of application `init()` is explicitly documented.

## Module graph diagnostics — GSD iteration

- **Discuss:** `metadata.Module` originally held only Type and Name; module import/export edges were unavailable, so the report could not responsibly claim cross-module cycle detection.
- **Plan:** Add backwards-compatible `Imports []reflect.Type` and `Exports []reflect.Type`, canonical module graph export, Tarjan SCC, and strict validation of missing/unknown metadata.
- **Execute:** `diagnostics/modules.go`, `modules.json`/`modules.dot`, unified HTML and CI fixtures. Keep legacy compiler generated metadata compiling.
- **Verify:** Require shuffled-order stable JSON/DOT, cyclic and self imports, missing modules, unknown exports and real `UsersModule` generated metadata in Actions.
- **Review/Ship:** The compiler still emits `@Module` Type/Name only; module visibility and export enforcement in runtime are deferred to #10/#11/#36. PR merge requires green CI.
