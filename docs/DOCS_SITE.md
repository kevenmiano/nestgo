# NestGo documentation site — GSD delivery

## Discuss
Model the documentation navigation after NestJS's developer journey (introduction → fundamentals → recipes → reference), but use original NestGo text and Go-native examples. Avoid copying NestJS assets or suggesting API parity.

## Plan
The Docusaurus source lives in `website/` and is separate from the Go compiler/runtime. The visual identity uses the **approved red gopher-cat mascot with speed trails** from the supplied artwork. `website/static/img/nestgo-mascot.svg` is a clean transparent vector trace of the approved image for scalable rendering across the homepage, navbar, favicon, and README. The prior N+G monogram and side-by-side upstream project logos are superseded. The mark and project are independent from NestJS and the Go team.

## Execute
```bash
cd website
npm install
npm run start
npm run build
```

The site targets GitHub Pages at `https://kevenmiano.github.io/nestgo/`. Pull requests and changes on `main` run the documentation build; **publishing** is handled separately by the release deployment workflow, only when a release tag points to a commit reachable from `main`.

## Verify
- `npm run build` succeeds in the documentation CI workflow.
- Navbar, sidebar, dark mode and links render correctly.
- Every documented capability matches the current compiler and runtime.
- Experimental status remains visible.
- The Go runtime remains dependency-free.

## Review
Check navigation, mascot fidelity, transparent rendering, dark/light contrast, mobile layout, accessibility, documentation accuracy and issue references.

## Ship
Merge after green CI. Publish Docusaurus using [docs-release.yml](../.github/workflows/docs-release.yml) when a new semantic GitHub Release is published for a commit on `main`. The deployed homepage shows the release version and a changelog link. Do not claim the URL is live until the deployment job succeeds.

### Release-tag publishing pipeline

Documentation is one of **three independent** release workflows. The others are [Go package verification](../.github/workflows/go-packages.yml) and [CLI/framework publication](../.github/workflows/framework-release.yml). All three are triggered by the same published GitHub Release, but have independent tests and status. See [distribution architecture](DISTRIBUTION.md).

1. In repository **Settings → Pages → Build and deployment**, select **GitHub Actions** as the source. The repository owner must perform this configuration; it is not changed by committing a workflow.
2. Ensure repository Actions permit GitHub Pages deployments. The workflow uses `pages: write` and `id-token: write`, and deploys to the `github-pages` environment. Environment restrictions must allow the release event/tag (for example `v*`), otherwise GitHub can block the deployment.
3. Merge all desired website updates into `main`, and verify the `Documentation site` CI job.
4. Create and publish a GitHub Release with a semantic tag such as `v0.1.0`, targeting `main` (not another branch). The workflow also checks that the tag's actual commit is an ancestor of `main`; if not, it fails without publishing.
5. GitHub Actions checks out the exact tagged commit, installs the Node 22 dependencies, builds Docusaurus with `NESTGO_RELEASE_TAG`, validates the generated site and uploads it as a Pages artifact. Deployment happens only if the build passes.
6. Confirm the `Publish Docusaurus on release` workflow succeeded and visit the deployed Pages URL. Future published releases update the site automatically.
7. For a known existing release tag, use the workflow's `Run workflow` action with the `tag` input. This uses the same ancestry and build checks.

```bash
# Create the tag on the intended main commit.
git switch main
git pull --ff-only
git tag -a v0.1.0 -m "NestGo v0.1.0"
git push origin v0.1.0
# Publish the Release for v0.1.0 on GitHub to trigger deployment.
```

Tag pushes alone do not publish the site; the trigger is **GitHub Release published**. PRs and ordinary main pushes validate the site without deploying. A GitHub Release that is only drafted does not deploy.

The homepage exposes the release tag as a badge linked to the matching GitHub Release. This describes the most recently deployed version; it does not claim to implement versioned multi-release documentation or API stability.

## References
- https://docs.nestjs.com/
- https://docusaurus.io/docs/installation
- https://docusaurus.io/docs/deployment
