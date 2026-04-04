# sentry-operator

A Kubernetes operator that automatically provisions [Sentry](https://sentry.io) projects and injects DSNs as Kubernetes Secrets.

Instead of manually creating Sentry projects and copy-pasting DSNs into your manifests, you declare a `SentryProject` resource alongside your app. The operator provisions the project in Sentry, fetches the DSN, and writes it into a Secret in the same namespace — ready to be mounted as an environment variable.

> **Sentry plan requirement:** Creating projects via the Sentry API requires a **Business plan** or above on sentry.io. If you are on a free or Team plan, create your projects manually in the Sentry UI and use `SentryProjectRef` instead — it references an existing project and fetches the DSN without needing elevated API permissions.

## Installation

```bash
helm upgrade --install sentry-operator oci://ghcr.io/agjmills/charts/sentry-operator \
  --namespace sentry-operator \
  --create-namespace \
  --set operator.defaultOrganization=my-org \
  --set operator.defaultTeam=my-team \
  --set sentryToken=sntrys_...
```

For production, provide the token via an existing Secret rather than `--set`:

```bash
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

The Sentry auth token requires `project:read` and `project:write` scopes. If you are only using `SentryProjectRef`, `project:read` alone is sufficient.

## Usage

### SentryProject — create and manage a project

```yaml
apiVersion: sentry-operator.io/v1alpha1
kind: SentryProject
metadata:
  name: myapp
  namespace: myapp
spec:
  platform: go
```

### SentryProjectRef — reference an existing project

```yaml
apiVersion: sentry-operator.io/v1alpha1
kind: SentryProjectRef
metadata:
  name: myapp
  namespace: myapp
spec:
  projectSlug: myapp
```

Both resources write a Secret containing `SENTRY_DSN` into the same namespace.

### Referencing the Secret in your Deployment

```yaml
envFrom:
  - secretRef:
      name: myapp-sentry
```

## Configuration

| Value | Default | Description |
|---|---|---|
| `operator.defaultOrganization` | `""` | Fallback Sentry org slug |
| `operator.defaultTeam` | `""` | Fallback Sentry team slug |
| `operator.defaultPlatform` | `""` | Fallback platform |
| `operator.defaultRetainOnDelete` | `true` | Retain Sentry project on CRD deletion |
| `operator.requeueInterval` | `24h` | How often to re-validate against the Sentry API |
| `operator.sentryURL` | `https://sentry.io` | Base URL (for self-hosted Sentry) |
| `sentryToken` | `""` | Sentry auth token (use `existingSecret` for production) |
| `existingSecret` | `""` | Name of an existing Secret containing `SENTRY_TOKEN` |
| `replicaCount` | `1` | Number of operator replicas |
| `leaderElect` | `false` | Enable leader election (recommended with multiple replicas) |

## Full documentation

See the [GitHub repository](https://github.com/agjmills/sentry-operator) for the full CRD reference, multi-key DSN configuration, rate limiting, and self-hosted Sentry setup.
