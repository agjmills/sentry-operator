# sentry-operator

A Kubernetes operator that automatically provisions [Sentry](https://sentry.io) projects and injects DSNs as Kubernetes Secrets.

Instead of manually creating Sentry projects and copy-pasting DSNs into your manifests, you declare a `SentryProject` resource alongside your app. The operator provisions the project in Sentry, fetches the DSN, and writes it into a Secret in the same namespace — ready to be mounted as an environment variable.

```yaml
apiVersion: sentry-operator.io/v1alpha1
kind: SentryProject
metadata:
  name: myapp
  namespace: myapp
spec:
  platform: go
```

```
$ kubectl get sentryprojects -n myapp
NAME    PROJECT   SECRET         READY   AGE
myapp   myapp     myapp-sentry   True    10s
```

```
$ kubectl get secret myapp-sentry -n myapp -o jsonpath='{.data.SENTRY_DSN}' | base64 -d
https://abc123@o123.ingest.sentry.io/456
```

---

## How it works

1. You create a `SentryProject` resource in your app's namespace.
2. The operator calls the Sentry API to create the project if it doesn't already exist.
3. It fetches the project's DSN and writes it into a Kubernetes Secret in the same namespace.
4. Your Deployment references the Secret via `envFrom`. Tools like [Reloader](https://github.com/stakater/Reloader) will automatically restart pods if the Secret is ever updated.
5. If the `SentryProject` is deleted, the Secret is cleaned up. The Sentry project itself is retained by default (configurable).

The operator re-validates each project against the Sentry API every 24 hours (configurable), so if a project or DSN key is manually deleted from Sentry it will be automatically restored.

---

## Installation

### Helm

```bash
helm upgrade --install sentry-operator oci://ghcr.io/agjmills/charts/sentry-operator \
  --namespace sentry-operator \
  --create-namespace \
  --set operator.defaultOrganization=my-org \
  --set operator.defaultTeam=my-team \
  --set sentryToken=sntrys_...
```

The Sentry auth token requires the `project:read` and `project:write` scopes. For production use, provide it via an existing Secret rather than `--set sentryToken`:

```bash
# Create the secret (e.g. via ExternalSecrets, Vault, etc.)
kubectl create secret generic sentry-token \
  --from-literal=SENTRY_TOKEN=sntrys_... \
  -n sentry-operator

helm upgrade --install sentry-operator oci://ghcr.io/agjmills/charts/sentry-operator \
  --namespace sentry-operator \
  --create-namespace \
  --set operator.defaultOrganization=my-org \
  --set operator.defaultTeam=my-team \
  --set existingSecret=sentry-token
```

### Self-hosted Sentry

Point the operator at your own instance with `--set operator.sentryURL=https://sentry.example.com`.

---

## Usage

### Minimal

With `defaultOrganization` and `defaultTeam` set at the operator level, most apps only need:

```yaml
apiVersion: sentry-operator.io/v1alpha1
kind: SentryProject
metadata:
  name: myapp
  namespace: myapp
spec:
  platform: go
```

### Full spec

```yaml
apiVersion: sentry-operator.io/v1alpha1
kind: SentryProject
metadata:
  name: myapp
  namespace: myapp
spec:
  # Sentry organization slug. Overrides the operator default.
  organization: my-org

  # Sentry team slug. Overrides the operator default.
  team: my-team

  # Sentry platform identifier. See https://docs.sentry.io/platforms/
  platform: python-django

  # Override the Sentry project slug. Defaults to metadata.name.
  projectSlug: myapp-production

  # Override the output Secret name. Defaults to "<name>-sentry".
  secretName: myapp-sentry-dsn

  # Whether to retain the Sentry project if this resource is deleted.
  # Defaults to true. Set to false to cascade-delete the Sentry project.
  retainOnDelete: true

  # Customize the key names written into the Secret.
  secretKeys:
    dsn: SENTRY_DSN               # default
    environment: SENTRY_ENVIRONMENT
    release: SENTRY_RELEASE
```

### Referencing the Secret in your Deployment

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: myapp
  namespace: myapp
spec:
  template:
    spec:
      containers:
        - name: myapp
          image: myapp:latest
          envFrom:
            - secretRef:
                name: myapp-sentry  # matches spec.secretName (or <name>-sentry by default)
```

This injects `SENTRY_DSN` (and any other configured keys) directly into the container's environment. No code changes required beyond initialising your Sentry SDK with `SENTRY_DSN`.

---

## Configuration reference

### Operator flags

| Flag | Default | Description |
|---|---|---|
| `--sentry-url` | `https://sentry.io` | Base URL of the Sentry instance |
| `--sentry-token-env` | `SENTRY_TOKEN` | Env var name containing the auth token |
| `--default-organization` | `""` | Fallback org slug for projects that don't set `spec.organization` |
| `--default-team` | `""` | Fallback team slug |
| `--default-platform` | `""` | Fallback platform |
| `--default-retain-on-delete` | `true` | Fallback `retainOnDelete` value |
| `--requeue-interval` | `24h` | How often to re-validate projects against the Sentry API |
| `--leader-elect` | `false` | Enable leader election (recommended for multiple replicas) |

All flags are also exposed as `values.yaml` keys under `operator.*`.

### `SentryProject` spec

| Field | Default | Description |
|---|---|---|
| `spec.organization` | operator default | Sentry org slug |
| `spec.team` | operator default | Sentry team slug |
| `spec.platform` | operator default | Sentry platform (e.g. `go`, `python-django`, `javascript`) |
| `spec.projectSlug` | `metadata.name` | Sentry project slug |
| `spec.secretName` | `<name>-sentry` | Name of the output Secret |
| `spec.retainOnDelete` | `true` | Retain Sentry project on CRD deletion |
| `spec.secretKeys.dsn` | `SENTRY_DSN` | Key name for the DSN in the Secret |
| `spec.secretKeys.environment` | `SENTRY_ENVIRONMENT` | Key name for the environment annotation |
| `spec.secretKeys.release` | `SENTRY_RELEASE` | Key name for the release annotation |

### `SentryProject` status

```
$ kubectl describe sentryproject myapp -n myapp
Status:
  Conditions:
    Type:                Ready
    Status:              True
    Reason:              ProjectProvisioned
    Message:             Sentry project provisioned and secret synced
    Last Transition Time: 2026-04-03T12:00:00Z
  Last Sync Time:        2026-04-03T12:00:00Z
  Project Slug:          myapp
  Secret Name:           myapp-sentry
```

---

## Development

### Prerequisites

- Go 1.23+
- A Kubernetes cluster (or [kind](https://kind.sigs.k8s.io/))
- A Sentry auth token

### Running locally

```bash
# Install CRDs into your current cluster
make install

# Run the operator against your current kubeconfig
SENTRY_TOKEN=sntrys_... make run SENTRY_ORG=my-org SENTRY_TEAM=my-team
```

### Regenerating CRDs and deepcopy after changing types

```bash
make manifests generate
```

### Running tests

```bash
make test
```

---

## Releases

Releases are automated via [release-please](https://github.com/googleapis/release-please). Merging commits to `main` that follow [Conventional Commits](https://www.conventionalcommits.org/) will automatically open a release PR. Merging the release PR tags the repo and triggers the publish workflow, which builds and pushes:

- A multi-arch Docker image to `ghcr.io/agjmills/sentry-operator`
- A Helm chart to `ghcr.io/agjmills/charts`

| Commit prefix | Version bump |
|---|---|
| `fix:` | patch |
| `feat:` | minor |
| `feat!:` / `BREAKING CHANGE:` | major |

---

## License

Apache 2.0 — see [LICENSE](LICENSE).
