package sentry_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/agjmills/sentry-operator/internal/sentry"
)

func newTestServer(t *testing.T, handler http.HandlerFunc) (*httptest.Server, *sentry.Client) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return srv, sentry.NewClient(srv.URL, "test-token")
}

func encode(t *testing.T, w http.ResponseWriter, v interface{}) {
	t.Helper()
	if err := json.NewEncoder(w).Encode(v); err != nil {
		t.Errorf("encode response: %v", err)
	}
}

func TestGetProject_Found(t *testing.T) {
	srv, client := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		encode(t, w, map[string]string{"id": "1", "slug": "myapp", "name": "myapp"})
	})
	_ = srv

	project, err := client.GetProject(context.Background(), "my-org", "myapp")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if project == nil {
		t.Fatal("expected project, got nil")
	}
	if project.Slug != "myapp" {
		t.Errorf("expected slug myapp, got %s", project.Slug)
	}
}

func TestGetProject_NotFound(t *testing.T) {
	srv, client := newTestServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})
	_ = srv

	project, err := client.GetProject(context.Background(), "my-org", "missing")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if project != nil {
		t.Errorf("expected nil project, got %+v", project)
	}
}

func TestCreateProject(t *testing.T) {
	srv, client := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		w.WriteHeader(http.StatusCreated)
		w.Header().Set("Content-Type", "application/json")
		encode(t, w, map[string]string{"id": "2", "slug": "newapp", "name": "newapp"})
	})
	_ = srv

	project, err := client.CreateProject(context.Background(), "my-org", "my-team", "newapp", "newapp", "go")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if project.Slug != "newapp" {
		t.Errorf("expected slug newapp, got %s", project.Slug)
	}
}

func TestListKeys(t *testing.T) {
	srv, client := newTestServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		encode(t, w, []map[string]interface{}{
			{"id": "k1", "label": "Default", "dsn": map[string]string{"public": "https://abc@o1.ingest.sentry.io/1"}},
			{"id": "k2", "label": "backend", "dsn": map[string]string{"public": "https://def@o1.ingest.sentry.io/2"}},
		})
	})
	_ = srv

	keys, err := client.ListKeys(context.Background(), "my-org", "myapp")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(keys) != 2 {
		t.Fatalf("expected 2 keys, got %d", len(keys))
	}
	if keys[0].Label != "Default" {
		t.Errorf("expected label Default, got %s", keys[0].Label)
	}
}

func TestCreateKey(t *testing.T) {
	srv, client := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		w.WriteHeader(http.StatusCreated)
		w.Header().Set("Content-Type", "application/json")
		encode(t, w, map[string]interface{}{
			"id":    "k3",
			"label": "frontend",
			"dsn":   map[string]string{"public": "https://ghi@o1.ingest.sentry.io/3"},
		})
	})
	_ = srv

	key, err := client.CreateKey(context.Background(), "my-org", "myapp", "frontend")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if key.Label != "frontend" {
		t.Errorf("expected label frontend, got %s", key.Label)
	}
}

func TestUpdateKeyRateLimit(t *testing.T) {
	called := false
	srv, client := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		called = true
		w.WriteHeader(http.StatusOK)
		encode(t, w, map[string]interface{}{})
	})
	_ = srv

	err := client.UpdateKeyRateLimit(context.Background(), "my-org", "myapp", "k1", 1000, 3600)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Error("expected PUT to be called")
	}
}

func TestDeleteProject(t *testing.T) {
	srv, client := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		w.WriteHeader(http.StatusNoContent)
	})
	_ = srv

	if err := client.DeleteProject(context.Background(), "my-org", "myapp"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDeleteProject_AlreadyGone(t *testing.T) {
	srv, client := newTestServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})
	_ = srv

	if err := client.DeleteProject(context.Background(), "my-org", "gone"); err != nil {
		t.Fatalf("unexpected error on 404: %v", err)
	}
}

func TestAPIError(t *testing.T) {
	srv, client := newTestServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"detail":"Authentication credentials were not provided."}`))
	})
	_ = srv

	_, err := client.GetProject(context.Background(), "my-org", "myapp")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	apiErr, ok := err.(*sentry.APIError)
	if !ok {
		t.Fatalf("expected *sentry.APIError, got %T", err)
	}
	if apiErr.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", apiErr.StatusCode)
	}
}
