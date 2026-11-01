# Semantic releases

NestGo uses **Release Please** to propose versions and maintain `CHANGELOG.md`, and **GoReleaser** to build CLI binaries and checksums. All Go library packages share a **single root Go module version**.

## Process

1. Merge changes into `main` using Conventional Commit-style squash titles.
2. The [Prepare semantic release](../.github/workflows/release-please.yml) workflow opens or updates a release PR.
3. Review the proposed version and changelog. Merge the release PR once checks pass.
4. Release Please creates an immutable `vX.Y.Z` tag and GitHub Release.
5. Its workflow dispatches three **independent** jobs: `go-packages.yml`, `framework-release.yml` (GoReleaser) and `docs-release.yml`.
6. Check **all three** workflow results. A release tag alone does not guarantee that assets or documentation deployed successfully.

| Commit title | Typical bump |
| --- | --- |
| `fix(runtime): correct module visibility` | patch |
| `feat(runtime): add HTTP adapter` | minor |
| `feat!: change application bootstrap` with breaking-change description | major (prerelease/0.x strategy may differ) |
| `docs:`, `chore:`, `test:` alone | no version bump by default |

GoReleaser **does not determine semantic versions**. It creates release assets for the tag selected by Release Please.

## First-run setup

Under **Settings → Actions → General**, permit GitHub Actions to create pull requests and enable read/write repository permissions as required by the workflow. The workflow requests `contents: write`, `pull-requests: write`, `issues: write` and `actions: write` at job level.

The default `GITHUB_TOKEN` can create release PRs and tags, but those events do not automatically trigger downstream workflows. To avoid a PAT dependency, the release workflow explicitly invokes the three existing `workflow_dispatch` endpoints with the new immutable tag. **Release Please release PRs created with `GITHUB_TOKEN` may not trigger PR checks automatically**. Run the validation workflows manually or configure a GitHub App token/PAT with appropriately scoped permissions if required by branch protection.

Set GitHub Pages `github-pages` environment deployment rules to allow version tags (such as `v*`), and use **Settings → Pages → GitHub Actions** as the build/deployment source.

## Version integrity

`.release-please-manifest.json` initializes the last known version as `0.1.0` so automation advances the number. Do **not** move, delete and recreate, or reuse `v0.1.0`: the Go checksum database already recorded different bytes for that tag. Release new content as `v0.1.1` or a later version as determined by conventional commits. Tags must be immutable and commit ancestry is validated by downstream jobs.

After the release completes:

```bash
GOPROXY=https://proxy.golang.org go install github.com/kevenmiano/nestgo/cmd/nestgo@v0.1.1
```

Replace `v0.1.1` with the version actually published. Never turn off `GOSUMDB` to work around a mismatch.

## Retry

If asset building or Pages deployment fails, rerun the failed workflow with its `tag` input. Do not recreate the tag or merge an unrelated change solely to retry deployment.
