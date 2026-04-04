# Changelog

## [1.2.1](https://github.com/agjmills/sentry-operator/compare/v1.2.0...v1.2.1) (2026-04-04)


### Bug Fixes

* replace dockers_v2 with dockers+manifests, add goreleaser check to CI ([95d9e02](https://github.com/agjmills/sentry-operator/commit/95d9e0219125c7c7af5d5b007291a68969eafaa1))

## [1.2.0](https://github.com/agjmills/sentry-operator/compare/v1.1.1...v1.2.0) (2026-04-04)


### Features

* add artifacthub-pkg.yml to chart for richer Artifact Hub listing ([055e819](https://github.com/agjmills/sentry-operator/commit/055e819ed8483b3b67bdec5df37375a2d8aa2540))

## [1.1.1](https://github.com/agjmills/sentry-operator/compare/v1.1.0...v1.1.1) (2026-04-04)


### Bug Fixes

* gate release-please on CI passing and fix lint errors in keys_test ([a2699e5](https://github.com/agjmills/sentry-operator/commit/a2699e51e08ffd95dfb936872dbd47cd62019991))
* label operator-managed secrets with app.kubernetes.io/managed-by ([a4aebcb](https://github.com/agjmills/sentry-operator/commit/a4aebcbe9e53c7e96bc7655a7a888650e076f024))
* resolve lint errors in test files ([854e2a7](https://github.com/agjmills/sentry-operator/commit/854e2a7debdfe7c431a64b4a6bbe17c65e51b0ec))
* track key IDs in status to survive Sentry-side label renames ([5be8564](https://github.com/agjmills/sentry-operator/commit/5be856485edb84e634708e39f620c03a03bceb9a))

## [1.1.0](https://github.com/agjmills/sentry-operator/compare/v1.0.1...v1.1.0) (2026-04-03)


### Features

* add SentryProjectRef, multi-key DSNs, and rate limiting ([f7ffbb6](https://github.com/agjmills/sentry-operator/commit/f7ffbb6bbb8b2cbc9e0ac1b2a11a9df691742a5e))


### Bug Fixes

* install syft in publish workflow, migrate to dockers_v2 ([55d2de8](https://github.com/agjmills/sentry-operator/commit/55d2de8812069f5a101caeff2f50c89a8176ebdc))

## [1.0.1](https://github.com/agjmills/sentry-operator/compare/v1.0.0...v1.0.1) (2026-04-03)


### Bug Fixes

* allow manual trigger of release-please workflow ([66e5758](https://github.com/agjmills/sentry-operator/commit/66e5758390f82ec8954a8eb07dfd461867424d67))

## 1.0.0 (2026-04-03)


### Features

* initial implementation of sentry-operator ([e5a94ac](https://github.com/agjmills/sentry-operator/commit/e5a94ace43cb98a6a5992cfe6eddcc6f45bb8a99))


### Bug Fixes

* go 1.24 in Dockerfile and errors.New vet fix ([653eb83](https://github.com/agjmills/sentry-operator/commit/653eb838d367f6efb2eabb991727d57a6a36dc1b))
* gofmt cmd/main.go ([a46776f](https://github.com/agjmills/sentry-operator/commit/a46776f69109a9cee5c6658c51c2726993ad7ec5))
* use PAT for release-please so CI runs on its PRs ([87eab03](https://github.com/agjmills/sentry-operator/commit/87eab03ba9159833a42eddb9e638b05563958de7))
