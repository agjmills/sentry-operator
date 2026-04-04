package controller

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	sentryv1alpha1 "github.com/agjmills/sentry-operator/api/v1alpha1"
	"github.com/agjmills/sentry-operator/internal/sentry"
)

func newKeysTestServer(t *testing.T, handler http.HandlerFunc) *sentry.Client {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return sentry.NewClient(srv.URL, "test-token")
}

func encode(t *testing.T, w http.ResponseWriter, v interface{}) {
	t.Helper()
	if err := json.NewEncoder(w).Encode(v); err != nil {
		t.Errorf("encode response: %v", err)
	}
}

func makeKey(id, label, dsn string) map[string]interface{} {
	return map[string]interface{}{
		"id":    id,
		"label": label,
		"dsn":   map[string]string{"public": dsn},
	}
}

func TestReconcileKeys_Fallback(t *testing.T) {
	client := newKeysTestServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		encode(t, w, []interface{}{
			makeKey("k1", "Default", "https://abc@sentry.io/1"),
		})
	})

	data, _, err := reconcileKeys(context.Background(), client, "org", "slug", nil, nil, nil, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if data["SENTRY_DSN"] != "https://abc@sentry.io/1" {
		t.Errorf("unexpected DSN: %s", data["SENTRY_DSN"])
	}
}

func TestReconcileKeys_Fallback_NoKeys(t *testing.T) {
	client := newKeysTestServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		encode(t, w, []interface{}{})
	})

	_, _, err := reconcileKeys(context.Background(), client, "org", "slug", nil, nil, nil, true)
	if err == nil {
		t.Fatal("expected error when no keys exist, got nil")
	}
}

func TestReconcileKeys_ExistingKeys(t *testing.T) {
	client := newKeysTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			w.Header().Set("Content-Type", "application/json")
			encode(t, w, []interface{}{
				makeKey("k1", "backend", "https://abc@sentry.io/1"),
				makeKey("k2", "frontend", "https://def@sentry.io/2"),
			})
		}
	})

	specs := []sentryv1alpha1.KeySpec{
		{Name: "backend"},
		{Name: "frontend"},
	}

	data, statuses, err := reconcileKeys(context.Background(), client, "org", "slug", specs, nil, nil, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if data["SENTRY_DSN"] != "https://abc@sentry.io/1" {
		t.Errorf("unexpected first DSN: %s", data["SENTRY_DSN"])
	}
	if data["SENTRY_DSN_FRONTEND"] != "https://def@sentry.io/2" {
		t.Errorf("unexpected second DSN: %s", data["SENTRY_DSN_FRONTEND"])
	}
	if len(statuses) != 2 || statuses[0].ID != "k1" || statuses[1].ID != "k2" {
		t.Errorf("unexpected statuses: %+v", statuses)
	}
}

func TestReconcileKeys_CustomSecretKey(t *testing.T) {
	client := newKeysTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			w.Header().Set("Content-Type", "application/json")
			encode(t, w, []interface{}{
				makeKey("k1", "mykey", "https://abc@sentry.io/1"),
			})
		}
	})

	specs := []sentryv1alpha1.KeySpec{
		{Name: "mykey", SecretKey: "MY_CUSTOM_DSN"},
	}

	data, _, err := reconcileKeys(context.Background(), client, "org", "slug", specs, nil, nil, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if data["MY_CUSTOM_DSN"] != "https://abc@sentry.io/1" {
		t.Errorf("unexpected DSN at custom key: %s", data["MY_CUSTOM_DSN"])
	}
}

func TestReconcileKeys_CreatesMissingKey(t *testing.T) {
	created := false
	client := newKeysTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodGet {
			encode(t, w, []interface{}{})
		} else if r.Method == http.MethodPost {
			created = true
			w.WriteHeader(http.StatusCreated)
			encode(t, w, makeKey("k1", "newkey", "https://new@sentry.io/1"))
		}
	})

	specs := []sentryv1alpha1.KeySpec{{Name: "newkey"}}
	data, _, err := reconcileKeys(context.Background(), client, "org", "slug", specs, nil, nil, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !created {
		t.Error("expected key to be created via POST")
	}
	if data["SENTRY_DSN"] != "https://new@sentry.io/1" {
		t.Errorf("unexpected DSN: %s", data["SENTRY_DSN"])
	}
}

func TestReconcileKeys_RefusesMissingKey(t *testing.T) {
	client := newKeysTestServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		encode(t, w, []interface{}{})
	})

	specs := []sentryv1alpha1.KeySpec{{Name: "missing"}}
	_, _, err := reconcileKeys(context.Background(), client, "org", "slug", specs, nil, nil, false)
	if err == nil {
		t.Fatal("expected error when createMissing=false and key not found")
	}
}

func TestReconcileKeys_DefaultRateLimit(t *testing.T) {
	rateLimitCalled := false
	client := newKeysTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodGet {
			encode(t, w, []interface{}{
				makeKey("k1", "mykey", "https://abc@sentry.io/1"),
			})
		} else if r.Method == http.MethodPut {
			rateLimitCalled = true
			w.WriteHeader(http.StatusOK)
			encode(t, w, map[string]interface{}{})
		}
	})

	defaultRL := &sentryv1alpha1.RateLimitSpec{Count: 500, Window: 3600}
	specs := []sentryv1alpha1.KeySpec{{Name: "mykey"}}

	_, _, err := reconcileKeys(context.Background(), client, "org", "slug", specs, defaultRL, nil, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !rateLimitCalled {
		t.Error("expected rate limit PUT to be called")
	}
}

func TestReconcileKeys_PerKeyRateLimitOverridesDefault(t *testing.T) {
	var putBody map[string]interface{}
	client := newKeysTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodGet {
			encode(t, w, []interface{}{
				makeKey("k1", "mykey", "https://abc@sentry.io/1"),
			})
		} else if r.Method == http.MethodPut {
			if err := json.NewDecoder(r.Body).Decode(&putBody); err != nil {
				t.Errorf("decode PUT body: %v", err)
			}
			w.WriteHeader(http.StatusOK)
			encode(t, w, map[string]interface{}{})
		}
	})

	defaultRL := &sentryv1alpha1.RateLimitSpec{Count: 500, Window: 3600}
	perKeyRL := &sentryv1alpha1.RateLimitSpec{Count: 100, Window: 60}
	specs := []sentryv1alpha1.KeySpec{{Name: "mykey", RateLimit: perKeyRL}}

	_, _, err := reconcileKeys(context.Background(), client, "org", "slug", specs, defaultRL, nil, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	rl := putBody["rateLimit"].(map[string]interface{})
	if rl["count"].(float64) != 100 {
		t.Errorf("expected per-key count 100, got %v", rl["count"])
	}
}

func TestReconcileKeys_AdoptsRenamedKeyByID(t *testing.T) {
	// Simulate a key that was renamed in the Sentry UI from "backend" to "backend-renamed".
	// The operator should match by ID (from previous status) rather than creating a duplicate.
	client := newKeysTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			w.Header().Set("Content-Type", "application/json")
			// The key now has label "backend-renamed" in Sentry but same ID.
			encode(t, w, []interface{}{
				makeKey("k1", "backend-renamed", "https://abc@sentry.io/1"),
			})
		} else if r.Method == http.MethodPost {
			t.Error("should not create a new key — existing key should be adopted by ID")
		}
	})

	specs := []sentryv1alpha1.KeySpec{{Name: "backend"}}

	// Previous status records the key ID from when the key was originally named "backend".
	prevStatuses := []sentryv1alpha1.KeyStatus{
		{Name: "backend", ID: "k1", SecretKey: "SENTRY_DSN"},
	}

	data, _, err := reconcileKeys(context.Background(), client, "org", "slug", specs, nil, prevStatuses, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if data["SENTRY_DSN"] != "https://abc@sentry.io/1" {
		t.Errorf("unexpected DSN: %s", data["SENTRY_DSN"])
	}
}
