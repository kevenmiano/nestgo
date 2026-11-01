# NestGo distribution and package architecture

## Overview

NestGo is a **single Go module**, declared in the root `go.mod`:

```text
module github.com/kevenmiano/nestgo

github.com/kevenmiano/nestgo/
├── compiler/       public Go library, source-to-source compiler
├── runtime/        public Go library, net/http serving and DI
├── metadata/       public Go library, registry and shared descriptors
├── cmd/nestgo/     installable executable: nestgo
└── cmd/modforge/   backward-compatible executable: modforge
```

Packages under `runtime/`, `compiler/` and `metadata/` are **not currently in a Go `internal/` directory**. They are importable public packages. In Go, `internal` has an access-control meaning: packages inside it can only be imported by code under their allowed parent tree. Moving current public APIs under `internal/` would break consumer applications and generated metadata imports.

The **single-module** decision is deliberate: the compiler emits imports for `github.com/kevenmiano/nestgo/metadata` and the runtime imports that same metadata package. The three packages must agree on metadata contracts. They share a version and are delivered in **one module archive**, not three imaginary independent Go packages in a registry.

## Installation — CLI versus library

The CLI is a `package main` executable. The documented command is:

```bash
go install github.com/kevenmiano/nestgo/cmd/nestgo@latest
nestgo compile ./path/to/myapp
```

Pin a version for repeatability:

```bash
go install github.com/kevenmiano/nestgo/cmd/nestgo@v0.1.0
```

`modforge` remains available for existing users:

```bash
go install github.com/kevenmiano/nestgo/cmd/modforge@v0.1.0
```

**Important:** The `go install ...@version` command installs the executable to `GOBIN` (or `GOPATH/bin`) and downloads/transitively resolves dependencies needed to build that CLI, but does **not** add `nestgo` to the application's `go.mod`. That behavior is by design in Go modules.

A consumer Go application that imports NestGo packages must add the dependency separately:

```bash
mkdir myapp && cd myapp
go mod init example.com/myapp
go get github.com/kevenmiano/nestgo@v0.1.0
```

Then in normal Go code:

```go
import (
    "github.com/kevenmiano/nestgo/metadata"
    "github.com/kevenmiano/nestgo/runtime"
)
```

The application only requires the **root module**; imports for `runtime`, `compiler` and `metadata` resolve from the same tagged module. The `compiler` library can be imported when direct programmatic compilation is necessary; otherwise the `nestgo` CLI is preferred.

The `@v0.1.0` snippets are **examples** until that release has been published. Verify [releases](https://github.com/kevenmiano/nestgo/releases) before using a version.

For automated version selection and the release PR lifecycle, see [Semantic releases](SEMANTIC_RELEASES.md). Release Please creates the tag and changelog; GoReleaser builds CLI artifacts for that tag.\n\n## Three independent GitHub Actions release workflows

A single GitHub **Release published** event for a valid root-module tag (for example `v0.1.0`) triggers three separate workflows. They do not call or depend on one another.

| Pipeline | Workflow | Output |
| --- | --- | --- |
| Documentation | [docs-release.yml](../.github/workflows/docs-release.yml) | Builds tagged Docusaurus content; deploys GitHub Pages |
| Go libraries | [go-packages.yml](../.github/workflows/go-packages.yml) | Validates `compiler`, `metadata`, `runtime`; builds and tests an external consumer against the exact module tag |
| CLI / framework | [framework-release.yml](../.github/workflows/framework-release.yml) | Validates `go install .../cmd/nestgo@tag` and `modforge@tag`; uses GoReleaser OSS v2 to build Linux/macOS/Windows CLI archives, SHA-256 checksums, and attach them to the existing GitHub Release |

**Go packages are published by pushing an immutable root-module version tag to GitHub.** There is no separate `npm publish`-style Go package upload. Go module proxies can resolve the tagged module and its public packages. The packages workflow acts as a **release conformance gate**, not as a nonexistent per-package registry push.

`go-packages.yml` also runs independently on library/CLI changes to `main` and PRs; `go-quality.yml` remains the repository-wide unit/format/generation CI. The `docs-site.yml` workflow runs on docs changes in PRs and main; *only* `docs-release.yml` deploys Pages. A package-only change will not deploy docs until a release is published.

### Release tagging and safety

- Versions must match `v0.x.y` or `v1.x.y` (optional prerelease suffix); the module does not yet support `v2` because Go requires a `/v2` module path for semantic import versioning.
- A release tag must identify a commit reachable from `main`. Workflows verify this before checking out the immutable tagged commit.
- Create and publish a GitHub Release from the same tag after Go and Docusaurus CI are green.
- Release workflows can be rerun manually for an existing tag. GoReleaser manages the existing release and its assets (`release.mode: keep-existing`). Tags themselves must never be moved; avoid re-running a published release unnecessarily.
- The docs workflow requires repository **Settings → Pages → Build and deployment: GitHub Actions** and Pages deployment permissions.
- The GoReleaser Action requires the published release to exist, full git history and a job-scoped Actions token with `contents: write`. GoReleaser itself performs archive creation, checksums, changelog and release asset upload. The Go 1.23 environment is configured separately via `actions/setup-go`.
- The repository's tag-check script is intentionally retained only to verify the v0/v1 module major version and ancestry on `main`, not to implement build/release features.
- PR CI validates `.goreleaser.yaml` with `goreleaser check` and builds no-publish snapshots with `goreleaser release --snapshot --clean`.
- Supported CLI archive targets: Linux/macOS amd64+arm64 and Windows amd64+arm64; Windows uses ZIP, the other platforms use tar.gz. `cmd/modforge` remains available via `go install` but does not generate an additional release binary archive.
- Decision record: [ADR 0001](adr/0001-goreleaser-distribution.md).
- Package, CLI and docs pipelines are **operationally independent**: one may succeed while another fails. The release is fully verified only when all three are green.
- Do **not** claim a release, GitHub Pages publication, Go proxy indexing or CLI installation has succeeded before the corresponding release jobs have run.

```bash
git checkout main
git pull --ff-only
git tag -a v0.1.0 -m "NestGo v0.1.0"
git push origin v0.1.0
# Create and publish the GitHub Release for v0.1.0
```

After publication, check the three workflows separately under [Actions](https://github.com/kevenmiano/nestgo/actions). Source code and Go import paths remain version-aligned.

## Why not three independent nested modules?

Moving `compiler/`, `runtime/`, and `metadata/` into their own `go.mod` files would introduce separate semantic tag prefixes (`compiler/v0.1.0`, `runtime/v0.1.0`, etc.), require versioned cross-module dependencies, and complicate generated Go imports and release ordering. Do so only after a deliberate ADR and tested migration. It is **not** necessary to install the current CLI or use its public packages.

## GSD release checklist

1. **Discuss:** package boundaries, public API, Go internal visibility, semantic version choice and compatibility.
2. **Plan:** independent docs, library and CLI workflows; stable tag verification; acceptance gates.
3. **Execute:** update the CLI, tests, workflows, README and Docusaurus installation guides.
4. **Verify:** `go test ./...`, race tests for Go libraries, `go vet ./...`, build Docusaurus, release script syntax checks and the external-consumer smoke test on a **real tagged release**.
5. **Review:** verify the root module has no local `replace` directives, confirm generated imports, platform binaries and the Pages URL.
6. **Ship:** merge only after green CI. Tag/release after review, then record each workflow run and its status in `.planning/STATE.md`.

Reference: https://go.dev/ref/mod and https://go.dev/doc/go-get-install-deprecation
