# Changelog

## [1.0.0](https://github.com/kevenmiano/nestgo/compare/v0.1.0...v1.0.0) (2026-11-01)


### ⚠ BREAKING CHANGES

* **di:** enforce explicit-only field injection ([#131](https://github.com/kevenmiano/nestgo/issues/131))
* **runtime:** Replace NewApplication/ApplicationFromRegistry/NewWithOptions/FromRegistry with runtime.New(runtime.WithRegistry(...) or runtime.WithPackages(...)).

### Features

* **cli:** add read-only generated artifacts check ([#161](https://github.com/kevenmiano/nestgo/issues/161)) ([bfe57af](https://github.com/kevenmiano/nestgo/commit/bfe57af7350ebb8ff5c7552e9c6c246b69704ca8))
* **cli:** adopt Cobra and generate CLI reference ([#149](https://github.com/kevenmiano/nestgo/issues/149)) ([bcec402](https://github.com/kevenmiano/nestgo/commit/bcec40229c7beda48fb298dc64234f2da308c109))
* **compiler:** add Cobra watch mode and compiled test scaffolds ([#130](https://github.com/kevenmiano/nestgo/issues/130)) ([18a406f](https://github.com/kevenmiano/nestgo/commit/18a406f6ecc6ddf51f64828b601250fae4b42b1e))
* **compiler:** generate scoped guard pipe filter metadata and typed binding ([479c7ab](https://github.com/kevenmiano/nestgo/commit/479c7ab1a3baa00b11f0f659f26b6f1fa4b1932f))
* **compiler:** generate scoped guard pipe filter metadata and typed binding ([2d4f537](https://github.com/kevenmiano/nestgo/commit/2d4f5377f5d9ed71110be53f26f1d41d705f1989))
* **compiler:** generate scoped guard pipe filter metadata and typed binding ([405cf56](https://github.com/kevenmiano/nestgo/commit/405cf5672e61fdfb69ee9a1d108ec75eaf5703da))
* **compiler:** versioned module export contracts for source-less imports ([#85](https://github.com/kevenmiano/nestgo/issues/85)) ([4972139](https://github.com/kevenmiano/nestgo/commit/4972139efeb8403010e2688ceef5e127077344d4))
* **database:** compiled SQL repositories and automatic module bootstrap ([#82](https://github.com/kevenmiano/nestgo/issues/82)) ([d0a4037](https://github.com/kevenmiano/nestgo/commit/d0a40370012aa0fe8a5e5ed12b08cc2f5089de06))
* **dto:** compiled property transformers and validation decorators ([b483e05](https://github.com/kevenmiano/nestgo/commit/b483e054e246e4459b4f19ae00646aac8b0da321))
* **http:** add engine-neutral guards, pipes and exception contracts ([497ac54](https://github.com/kevenmiano/nestgo/commit/497ac548ad974be142d61db6e1b40d7e5156ca7f))
* **http:** add engine-neutral guards, pipes and exception contracts ([96ccbb7](https://github.com/kevenmiano/nestgo/commit/96ccbb7856bc10976dc8e023c7ac9c01b22a29ad))
* **http:** add engine-neutral guards, pipes and exception contracts ([991b075](https://github.com/kevenmiano/nestgo/commit/991b07591385916d4e2ca9e8e9a7e0eca6169aa0))
* **http:** engine-agnostic responses, typed contracts and HTTP exceptions ([#107](https://github.com/kevenmiano/nestgo/issues/107)) ([6720289](https://github.com/kevenmiano/nestgo/commit/6720289095801a0b572713de851affd8a5f3166d))
* **http:** middleware, guards, pipes and exception filters ([#122](https://github.com/kevenmiano/nestgo/issues/122)) ([73ba8c7](https://github.com/kevenmiano/nestgo/commit/73ba8c7718840351963765337420837e996e69f4))
* **runtime:** configure a code-first Go HTTP server without compiler ([#121](https://github.com/kevenmiano/nestgo/issues/121)) ([969526c](https://github.com/kevenmiano/nestgo/commit/969526c23b8e36a4e0809ce331aaae2c45844b87))
* **runtime:** context-aware bootstrap and idempotent Close ([#128](https://github.com/kevenmiano/nestgo/issues/128)) ([9b3a69e](https://github.com/kevenmiano/nestgo/commit/9b3a69e84c7e65f4098e7f05777122447d56eb90))
* **runtime:** execute middleware, guards, pipes and exception filters per route ([165bf95](https://github.com/kevenmiano/nestgo/commit/165bf9557132326e9d893ade6ece7b2109b4b115))
* **runtime:** lifecycle hooks, shared DI, benchmarks and Mermaid docs ([#95](https://github.com/kevenmiano/nestgo/issues/95)) ([3fb3de3](https://github.com/kevenmiano/nestgo/commit/3fb3de3c77badb88f37b525b95a2abadac569f43))
* **runtime:** register request pipeline and resolve class components ([180e3e3](https://github.com/kevenmiano/nestgo/commit/180e3e383b5c1f58fbada83bc723e38ed00a5669))
* **runtime:** register request pipeline and resolve class components ([af8c671](https://github.com/kevenmiano/nestgo/commit/af8c6712905e457306ab986e3be1c13b34d406f2))


### Bug Fixes

* **ci:** repair route lookup and registration benchmarks ([#159](https://github.com/kevenmiano/nestgo/issues/159)) ([7d4de61](https://github.com/kevenmiano/nestgo/commit/7d4de61c5144408271f803d05775f714f882ed64))
* **compiler:** diagnose unimported external concrete dependencies ([#81](https://github.com/kevenmiano/nestgo/issues/81)) ([e6d0cc0](https://github.com/kevenmiano/nestgo/commit/e6d0cc0897d5c994bfa12afcbd60b8feb2233a13))
* **compiler:** enforce concrete provider exports across source modules ([#83](https://github.com/kevenmiano/nestgo/issues/83)) ([67de6ab](https://github.com/kevenmiano/nestgo/commit/67de6ab50f38eaabb0de92680b6f8b1836954cff))
* **compiler:** module-aware import resolution and cross-package E2E ([#76](https://github.com/kevenmiano/nestgo/issues/76)) ([08d3900](https://github.com/kevenmiano/nestgo/commit/08d39001ff6d0302f7f2be0abbe513457d7c5933))
* **di:** stable structured injection errors ([#141](https://github.com/kevenmiano/nestgo/issues/141)) ([a83df87](https://github.com/kevenmiano/nestgo/commit/a83df87a55511ef1d5f5a17181ded40b425c3143))
* **di:** typed nil guards, field markers and atomic injection ([#125](https://github.com/kevenmiano/nestgo/issues/125)) ([8f7a7ec](https://github.com/kevenmiano/nestgo/commit/8f7a7ec0c83644b41de8abd8b6f692ef1c41ef4c))
* **dto:** restore formatter, go.mod, and generated test gates ([#164](https://github.com/kevenmiano/nestgo/issues/164)) ([29de476](https://github.com/kevenmiano/nestgo/commit/29de47670cb08ca3dbbdede719fe5618cc229a71))
* **integration:** format merged runtime and regenerate NestGo metadata ([a6345a0](https://github.com/kevenmiano/nestgo/commit/a6345a069faf817f7b4f57573c6d2b5b9a8bcf42))
* **integration:** regenerate and gofmt current NestGo pipeline ([101f17c](https://github.com/kevenmiano/nestgo/commit/101f17c4dd7740024b18e16894fd894a4a1df37f))
* **pipeline:** typed bindings, HTTP response assertions and 5xx redaction ([74691e6](https://github.com/kevenmiano/nestgo/commit/74691e665ad4dec849e6bb6770c5d9554abd6fea))
* **pipeline:** typed bindings, HTTP response assertions and 5xx redaction ([28df509](https://github.com/kevenmiano/nestgo/commit/28df5099d47a95f9fcf14bf57cffd8a0c3a34f43))
* **pipeline:** typed bindings, HTTP response assertions and 5xx redaction ([428e272](https://github.com/kevenmiano/nestgo/commit/428e272309fd8b6d266700485170ec9bb9933a42))
* **pipeline:** typed bindings, HTTP response assertions and 5xx redaction ([143d7ca](https://github.com/kevenmiano/nestgo/commit/143d7ca6410341d822f58d82f718321f4f855ca8))
* **runtime:** dependency-ordered lifecycle, rollback and shutdown safety ([#157](https://github.com/kevenmiano/nestgo/issues/157)) ([c21aadf](https://github.com/kevenmiano/nestgo/commit/c21aadf6348b289c177da2b5853853407c8a90c4))
* **runtime:** enforce module-scoped DI in HTTP adapter bootstrap ([#74](https://github.com/kevenmiano/nestgo/issues/74)) ([86ce715](https://github.com/kevenmiano/nestgo/commit/86ce715f8b7e71f564b726b27b41325c0034247a))
* **runtime:** preserve security error redaction after main integration ([12f4ed4](https://github.com/kevenmiano/nestgo/commit/12f4ed4f45e13638215bbe4b9f5982725521b783))
* **runtime:** preserve security error redaction after main integration ([dc47f7d](https://github.com/kevenmiano/nestgo/commit/dc47f7dccc78d157a88dfe65aac114df9984f23f))
* **runtime:** preserve security error redaction after main integration ([a8fac03](https://github.com/kevenmiano/nestgo/commit/a8fac03921eccf75a02f722993d1b6132482cfe8))
* **runtime:** reject unsafe handler and binding contracts ([#108](https://github.com/kevenmiano/nestgo/issues/108)) ([cc620db](https://github.com/kevenmiano/nestgo/commit/cc620db9c80004d5dac1533cb8039a4f5f9c5fdc))
* **runtime:** resolve imported module packages on adapter bootstrap ([#72](https://github.com/kevenmiano/nestgo/issues/72)) ([b28caad](https://github.com/kevenmiano/nestgo/commit/b28caadb3c15eb8464bf4f1dcbb1d10863d78a6a))
* **security:** bound HTTP resources across built-in adapters ([#126](https://github.com/kevenmiano/nestgo/issues/126)) ([3d76c75](https://github.com/kevenmiano/nestgo/commit/3d76c75bda14ebeb113b677f446667fccef9c5e1))
* **security:** harden runtime error boundaries and HTTP routing ([#123](https://github.com/kevenmiano/nestgo/issues/123)) ([d4aa407](https://github.com/kevenmiano/nestgo/commit/d4aa407ed864f2b2f40b9446b265c3d192799e1a))
* **security:** upgrade Go toolchain and fasthttp with vulnerability gates ([#127](https://github.com/kevenmiano/nestgo/issues/127)) ([8470632](https://github.com/kevenmiano/nestgo/commit/84706325b74dda347a0b8ce19308073f7e0976f9))


### Performance Improvements

* **fasthttp:** avoid quadratic literal route registration ([#156](https://github.com/kevenmiano/nestgo/issues/156)) ([effd75a](https://github.com/kevenmiano/nestgo/commit/effd75a9660120bb3e2fc74086f5464f897a083e))
* **fasthttp:** optimize indexed routing and benchmarks ([#101](https://github.com/kevenmiano/nestgo/issues/101)) ([a1bfc52](https://github.com/kevenmiano/nestgo/commit/a1bfc524869a1db786ffa6c308206f4aee6930d4))
* **release:** preserve 1-1000 route/provider p95 p99 baselines ([#150](https://github.com/kevenmiano/nestgo/issues/150)) ([11f80d0](https://github.com/kevenmiano/nestgo/commit/11f80d0e21ba6d2fdd52b59451290feaf8e678ec))
* **release:** record bootstrap latency and validated scale summaries ([#153](https://github.com/kevenmiano/nestgo/issues/153)) ([f35a3c1](https://github.com/kevenmiano/nestgo/commit/f35a3c1cf20298e746995b8cfcc7dac1ad19743f))
* **runtime:** benchmark code-first HTTP versus generated controllers ([#124](https://github.com/kevenmiano/nestgo/issues/124)) ([8016574](https://github.com/kevenmiano/nestgo/commit/801657494ee02ca0daadc4c8c72e620298124264))
* **runtime:** compile middleware dispatcher once per route ([c74a9dd](https://github.com/kevenmiano/nestgo/commit/c74a9dd8f5bb43d7a8a14825932fc2e7bd772f0a))
* **runtime:** compile typed string binding fallback ([#142](https://github.com/kevenmiano/nestgo/issues/142)) ([d54c56f](https://github.com/kevenmiano/nestgo/commit/d54c56f43696f6ea0bbc113a7098ced5a5a1e46f))
* **runtime:** eliminate unused bootstrap allocations ([#158](https://github.com/kevenmiano/nestgo/issues/158)) ([81d6766](https://github.com/kevenmiano/nestgo/commit/81d67664530da5f9515d3544d16dc8b1e812b8b6))
* **runtime:** generate typed HTTP parameter dispatch ([#106](https://github.com/kevenmiano/nestgo/issues/106)) ([433b3b6](https://github.com/kevenmiano/nestgo/commit/433b3b6158e66bc1067302b5ca160f9cc953216b))
* **runtime:** isolate middleware guard pipe filter and static reflection overhead ([53f0bcb](https://github.com/kevenmiano/nestgo/commit/53f0bcb523269e3c41f59dec1c1c57eabba0510c))
* **runtime:** reuse route-specific middleware chain for controllers and code-first handlers ([cf18925](https://github.com/kevenmiano/nestgo/commit/cf1892503b47ea7a5d839c50368ba365f803f5dd))
* **runtime:** skip metadata bootstrap for code-first-only servers ([#129](https://github.com/kevenmiano/nestgo/issues/129)) ([09de318](https://github.com/kevenmiano/nestgo/commit/09de318f4764ea0643916089d79799cbad39eedb))
* **runtime:** use static HTTP invokers and add benchmarks ([#86](https://github.com/kevenmiano/nestgo/issues/86)) ([e0bdeb5](https://github.com/kevenmiano/nestgo/commit/e0bdeb53f5cbb17c5de630f65bf36901e99815a9))


### Code Refactoring

* **di:** enforce explicit-only field injection ([#131](https://github.com/kevenmiano/nestgo/issues/131)) ([ddc5d16](https://github.com/kevenmiano/nestgo/commit/ddc5d16fb3c7374123fcd29ee16e00087801a4e7))
* **runtime:** one application factory and functional options ([#110](https://github.com/kevenmiano/nestgo/issues/110)) ([11dd209](https://github.com/kevenmiano/nestgo/commit/11dd209bd92ebaf6845f6c13e88c47ca105f4532))
