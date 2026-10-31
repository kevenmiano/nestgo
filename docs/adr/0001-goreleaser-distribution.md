# ADR 0001: Use GoReleaser for NestGo CLI distribution

- Status: Accepted for the release automation implementation (pending a real tagged release)
- Date: 2026-10-31
- Context: M006 — Developer experience and release

## Discuss

The repository currently contains manual GOOS/GOARCH loops, checksum generation and GitHub CLI asset uploads in `framework-release.yml`. Those operations duplicate mature GoReleaser OSS features. NestGo has one Go module with public `compiler`, `runtime` and `metadata` packages and two installable CLI paths (`cmd/nestgo`, `cmd/modforge`).

## Decision

- Use GoReleaser OSS v2, invoked through `goreleaser/goreleaser-action@v7`, to build and publish the primary `nestgo` CLI binaries and release archives.
- Keep three independent release workflows: `docs-release.yml` for GitHub Pages, `go-packages.yml` for Go library importability, and `framework-release.yml` for CLI releases.
- Use `.goreleaser.yaml` as the **single source of truth** for platform targets, archive formats, checksums and changelog settings.
- Retain `verify-release-tag.sh` as a security/Go-module-compatibility guard: GoReleaser does not enforce this repository's v0/v1-only policy or that a tag's commit belongs to main.
- Retain external `go install ...@tag` smoke testing, since successful binary builds do not prove a consumer can install a tagged Go module.
- Keep `cmd/modforge` installable through Go modules for compatibility, but publish platform archives for the public `nestgo` CLI only.
- A GitHub Release is published first, triggering the workflow. GoReleaser adds assets and **keeps existing release notes** using `release.mode: keep-existing`.

## Plan / Execute

1. Replace the manual cross-platform build/upload jobs with GoReleaser.
2. Add config validation and no-publish snapshot builds to PR CI.
3. Document the security controls, explicit supported platforms, and independent workflows.
4. Reconcile README, Docusaurus and GSD state.

## Verify

- `Go quality gates`, `Documentation site`, `Go packages` and `Release policy` must all be green before merge.
- `goreleaser check` and `goreleaser release --snapshot --clean` must pass on the PR, without publishing.
- The first real release must independently confirm the exact tagged `go install`, uploaded archives and checksums, and docs deployment.

## Consequences

**Benefits:** fewer bespoke scripts, centralized configuration, established changelog and artifact naming, easy cross-platform expansion.

**Trade-offs:** external GoReleaser action and upstream binary dependencies, releases remain independently green/red across workflows, and the first live tag is necessary to fully verify publication. GoReleaser does not independently publish the library packages: Go's root-module version tag does.

## Ship

Merge only with green PR checks and record the merge SHA in `.planning/STATE.md`. No first-release publication is asserted by this ADR.

References: https://goreleaser.com/customization/ci/actions/ and https://goreleaser.com/customization/
