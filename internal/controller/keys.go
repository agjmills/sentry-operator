package controller

import (
	"context"
	"fmt"
	"strings"

	sentryv1alpha1 "github.com/agjmills/sentry-operator/api/v1alpha1"
	"github.com/agjmills/sentry-operator/internal/sentry"
)

// reconcileKeys ensures the desired DSN keys exist in Sentry, applies rate limits,
// and returns a map of Secret key name → DSN value.
//
// If keySpecs is empty, the first existing Sentry key is used and written as SENTRY_DSN
// (backward-compatible single-key mode).
func reconcileKeys(
	ctx context.Context,
	sc *sentry.Client,
	org, slug string,
	keySpecs []sentryv1alpha1.KeySpec,
	defaultRL *sentryv1alpha1.RateLimitSpec,
	createMissing bool,
) (map[string]string, error) {
	existing, err := sc.ListKeys(ctx, org, slug)
	if err != nil {
		return nil, fmt.Errorf("list keys: %w", err)
	}

	// Single-key fallback.
	if len(keySpecs) == 0 {
		if len(existing) == 0 {
			return nil, fmt.Errorf("project %s/%s has no DSN keys", org, slug)
		}
		return map[string]string{"SENTRY_DSN": existing[0].DSN.Public}, nil
	}

	// Index existing keys by label for O(1) lookup.
	byLabel := make(map[string]sentry.DSNKey, len(existing))
	for _, k := range existing {
		byLabel[k.Label] = k
	}

	result := make(map[string]string, len(keySpecs))

	for i, spec := range keySpecs {
		key, found := byLabel[spec.Name]

		if !found {
			if !createMissing {
				return nil, fmt.Errorf("key %q not found in project %s/%s", spec.Name, org, slug)
			}
			created, err := sc.CreateKey(ctx, org, slug, spec.Name)
			if err != nil {
				return nil, fmt.Errorf("create key %q: %w", spec.Name, err)
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
				return nil, fmt.Errorf("update rate limit for key %q: %w", spec.Name, err)
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

		result[secretKey] = key.DSN.Public
	}

	return result, nil
}
