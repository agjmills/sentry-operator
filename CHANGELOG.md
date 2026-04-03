# Changelog

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
