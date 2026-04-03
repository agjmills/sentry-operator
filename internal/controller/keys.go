package controller

import (
	"context"
	"fmt"
	"strings"

	sentryv1alpha1 "github.com/agjmills/sentry-operator/api/v1alpha1"
	"github.com/agjmills/sentry-operator/internal/sentry"
)

// reconcileKeys ensures the desired DSN keys exist in Sentry, applies rate limits,
// and returns a map of Secret key name → DSN value plus updated key statuses.
//
// existingStatuses contains the key IDs tracked from the previous reconcile.
// Matching by ID is attempted first so that externally renamed keys are adopted
// rather than duplicated.
//
// If keySpecs is empty, the first existing Sentry key is used and written as
// SENTRY_DSN (backward-compatible single-key mode).
func reconcileKeys(
	ctx context.Context,
	sc *sentry.Client,
	org, slug string,
	keySpecs []sentryv1alpha1.KeySpec,
	defaultRL *sentryv1alpha1.RateLimitSpec,
	existingStatuses []sentryv1alpha1.KeyStatus,
	createMissing bool,
) (map[string]string, []sentryv1alpha1.KeyStatus, error) {
	existing, err := sc.ListKeys(ctx, org, slug)
	if err != nil {
		return nil, nil, fmt.Errorf("list keys: %w", err)
	}

	// Single-key fallback.
	if len(keySpecs) == 0 {
		if len(existing) == 0 {
			return nil, nil, fmt.Errorf("project %s/%s has no DSN keys", org, slug)
		}
		k := existing[0]
		statuses := []sentryv1alpha1.KeyStatus{{Name: k.Label, ID: k.ID, SecretKey: "SENTRY_DSN"}}
		return map[string]string{"SENTRY_DSN": k.DSN.Public}, statuses, nil
	}

	// Index existing Sentry keys by ID and by label for lookup.
	byID := make(map[string]sentry.DSNKey, len(existing))
	byLabel := make(map[string]sentry.DSNKey, len(existing))
	for _, k := range existing {
		byID[k.ID] = k
		byLabel[k.Label] = k
	}

	// Index previous statuses by spec name for ID lookup.
	prevByName := make(map[string]sentryv1alpha1.KeyStatus, len(existingStatuses))
	for _, s := range existingStatuses {
		prevByName[s.Name] = s
	}

	secretData := make(map[string]string, len(keySpecs))
	newStatuses := make([]sentryv1alpha1.KeyStatus, 0, len(keySpecs))

	for i, spec := range keySpecs {
		var key sentry.DSNKey
		found := false

		// Prefer ID-based match from previous status (survives label renames).
		if prev, ok := prevByName[spec.Name]; ok {
			if k, ok := byID[prev.ID]; ok {
				key = k
				found = true
			}
		}

		// Fall back to label match.
		if !found {
			if k, ok := byLabel[spec.Name]; ok {
				key = k
				found = true
			}
		}

		if !found {
			if !createMissing {
				labels := make([]string, 0, len(existing))
				for _, k := range existing {
					labels = append(labels, k.Label)
				}
				return nil, nil, fmt.Errorf("key %q not found in project %s/%s (available: %s)",
					spec.Name, org, slug, strings.Join(labels, ", "))
			}
			created, err := sc.CreateKey(ctx, org, slug, spec.Name)
			if err != nil {
				return nil, nil, fmt.Errorf("create key %q: %w", spec.Name, err)
			}
			key = *created
		}

		// Resolve effective rate limit: key-level overrides project default.
		rl := defaultRL
		if spec.RateLimit != nil {
			rl = spec.RateLimit
		}
		if rl != nil {
			if err := sc.UpdateKeyRateLimit(ctx, org, slug, key.ID, rl.Count, rl.Window); err != nil {
				return nil, nil, fmt.Errorf("update rate limit for key %q: %w", spec.Name, err)
			}
		}

		// Resolve secret key name.
		secretKey := spec.SecretKey
		if secretKey == "" {
			if i == 0 {
				secretKey = "SENTRY_DSN"
			} else {
				secretKey = "SENTRY_DSN_" + strings.ToUpper(strings.ReplaceAll(spec.Name, "-", "_"))
			}
		}

		secretData[secretKey] = key.DSN.Public
		newStatuses = append(newStatuses, sentryv1alpha1.KeyStatus{
			Name:      spec.Name,
			ID:        key.ID,
			SecretKey: secretKey,
		})
	}

	return secretData, newStatuses, nil
}
