# NestGo documentation site — GSD delivery

## Discuss
Model the documentation navigation after NestJS's developer journey (introduction → fundamentals → recipes → reference), but use original NestGo text and Go-native examples. Avoid copying NestJS assets or suggesting API parity.

## Plan
The Docusaurus source lives in `website/` and is separate from the Go compiler/runtime. The original logo mark is an angular **N + G** monogram inside a hexagonal frame, combining Go-inspired teal/cyan and structured-application geometry. It does not reuse the official NestJS owl or the Go gopher. Treat the mark as a **proposed identity**, not a final brand approval.

## Execute
```bash
cd website
npm install
npm run start
npm run build
```

The site is configured for GitHub Pages at `https://kevenmiano.github.io/nestgo/` but **is not published automatically**. A PR build workflow validates the static site. Deployment requires configuring Pages and a separate publish action.

## Verify
- `npm run build` succeeds in the documentation CI workflow.
- Navbar, sidebar, dark mode and links render correctly.
- Every documented capability matches the current compiler and runtime.
- Experimental status remains visible.
- The Go runtime remains dependency-free.

## Review
Check navigation, branding contrast, mobile layout, accessibility, documentation accuracy and issue references.

## Ship
Merge after green CI. Later configure Pages deployment, domain, search and versioned documentation. Do not claim the docs URL is live before deployment.

## References
- https://docs.nestjs.com/
- https://docusaurus.io/docs/installation
- https://docusaurus.io/docs/deployment
