# Contributing

Contributions are welcome. This document covers how to get set up, the development workflow, and what to expect when opening a PR.

## Prerequisites

- Go 1.24+
- A Kubernetes cluster for manual testing ([kind](https://kind.sigs.k8s.io/) works well)
- A Sentry account and auth token with `project:read` and `project:write` scopes
- `make`, `helm`, `kubectl`
- [golangci-lint](https://golangci-lint.run/welcome/install/) v1.64+
- [goreleaser](https://goreleaser.com/install/) v2+ _(optional — for snapshot builds)_

## Getting started

```bash
git clone https://github.com/agjmills/sentry-operator
cd sentry-operator

# Set up git hooks (runs lint/tests before commit/push)
make install-hooks

# Install CRDs into your current cluster
make install

# Run the operator locally against your current kubeconfig
SENTRY_TOKEN=sntrys_... make run SENTRY_ORG=my-org SENTRY_TEAM=my-team
```

## Making changes

### Changing CRD types

After editing files in `api/v1alpha1/`, regenerate the CRD manifests and deepcopy methods:

```bash
make manifests generate
```

This requires `controller-gen`, which `make` will install into `./bin/` automatically.

### Running tests

```bash
make test
```

### Linting

```bash
make lint
```

### Testing the release build locally

If you're changing `.goreleaser.yaml` or `Dockerfile.goreleaser`, verify the snapshot build works before pushing:

```bash
make snapshot
```

This builds binaries and Docker images for both architectures locally without publishing anything.

## PR checklist

Before opening a PR, make sure the following all pass locally:

```bash
make install-hooks   # only needed once after cloning
make build           # compiles cleanly
make test            # all tests pass
make lint            # no lint errors
helm lint charts/sentry-operator/
```

If you've changed `.goreleaser.yaml` or `Dockerfile.goreleaser`:

```bash
make snapshot        # full local release build, no push
```

## Pull requests

- Open a PR against `main`
- All CI checks must pass (build, test, lint, helm lint, goreleaser check)
- **The squash commit title must follow [Conventional Commits](https://www.conventionalcommits.org/)** — this drives automated versioning via release-please:

  | Prefix | Effect |
  |---|---|
  | `fix:` | patch release |
  | `feat:` | minor release |
  | `feat!:` or `BREAKING CHANGE:` footer | major release |
  | `chore:`, `docs:`, `test:` | no release |

- You don't need to worry about the version number — release-please handles that when the maintainer is ready to cut a release

## Project structure

```
api/v1alpha1/           CRD types and deepcopy
internal/
  controller/           Reconcilers and shared logic (keys.go)
  sentry/               Thin Sentry REST API client
cmd/main.go             Operator entrypoint and flag definitions
charts/sentry-operator/ Helm chart
config/crd/bases/       Generated CRD manifests
```

## Reporting issues

Please open a GitHub issue. Include the operator version, Kubernetes version, and relevant logs from the operator pod (`kubectl logs -n sentry-operator deploy/sentry-operator`).
