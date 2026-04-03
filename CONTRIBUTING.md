# Contributing

Contributions are welcome. This document covers how to get set up, the development workflow, and what to expect when opening a PR.

## Prerequisites

- Go 1.24+
- A Kubernetes cluster for manual testing ([kind](https://kind.sigs.k8s.io/) works well)
- A Sentry account and auth token with `project:read` and `project:write` scopes
- `make`, `helm`, `kubectl`

## Getting started

```bash
git clone https://github.com/agjmills/sentry-operator
cd sentry-operator

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

## Pull requests

- Open a PR against `main`
- All CI checks must pass (build, test, lint, helm lint, docker build)
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
