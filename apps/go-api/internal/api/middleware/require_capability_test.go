package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"levelup/go-api/internal/ctxkeys"
	titlePkg "levelup/go-api/internal/domain/title"
)

func newTestRegistryWithB(t *testing.T) *titlePkg.Registry {
	t.Helper()
	r := titlePkg.NewRegistry()
	// Titre B sans Forge ni Firefight — pour tester le rejet.
	r.Register(&titlePkg.TitleDescriptor{
		Slug:     "synthetic_title_b",
		Name:     "Title B",
		Provider: "synthetic",
		Status:   titlePkg.StatusActive,
		Capabilities: []titlePkg.Capability{
			titlePkg.CapMatchmaking, titlePkg.CapCareer,
		},
	})
	return r
}

func TestRequireCapability_AllowsHaloInfiniteForge(t *testing.T) {
	reg := newTestRegistryWithB(t)
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})
	mw := RequireCapability(reg, titlePkg.CapForge)(next)

	req := httptest.NewRequest(http.MethodGet, "/forge/test", nil)
	ctx := ctxkeys.WithTitleSlug(req.Context(), "halo_infinite")
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()
	mw.ServeHTTP(rec, req)

	if !called {
		t.Error("next handler should have been called for halo_infinite + CapForge")
	}
	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}
}

func TestRequireCapability_RejectsTitleBForge(t *testing.T) {
	reg := newTestRegistryWithB(t)
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { called = true })
	mw := RequireCapability(reg, titlePkg.CapForge)(next)

	req := httptest.NewRequest(http.MethodGet, "/forge/test", nil)
	ctx := ctxkeys.WithTitleSlug(req.Context(), "synthetic_title_b")
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()
	mw.ServeHTTP(rec, req)

	if called {
		t.Error("next handler should NOT have been called for title B + CapForge")
	}
	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want 503", rec.Code)
	}

	var body map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body["code"] != "capability_unavailable" {
		t.Errorf("body.code = %v, want capability_unavailable", body["code"])
	}
	if body["capability"] != "forge" {
		t.Errorf("body.capability = %v, want forge", body["capability"])
	}
	if body["title_slug"] != "synthetic_title_b" {
		t.Errorf("body.title_slug = %v, want synthetic_title_b", body["title_slug"])
	}
	if body["retryable"] != false {
		t.Errorf("body.retryable = %v, want false", body["retryable"])
	}
}

func TestRequireCapability_RejectsUnknownTitle(t *testing.T) {
	reg := newTestRegistryWithB(t)
	mw := RequireCapability(reg, titlePkg.CapCareer)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/career", nil)
	ctx := ctxkeys.WithTitleSlug(req.Context(), "title_inexistant")
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()
	mw.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want 503 (titre inconnu)", rec.Code)
	}
}

func TestRequireCapability_FallsBackToDefaultSlug(t *testing.T) {
	// Si le contexte ne contient pas de slug, on doit retomber sur halo_infinite
	// (qui supporte CapCareer).
	reg := titlePkg.NewRegistry()
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { called = true })
	mw := RequireCapability(reg, titlePkg.CapCareer)(next)

	req := httptest.NewRequest(http.MethodGet, "/career", nil)
	// Pas de WithTitleSlug → ctx vide
	req = req.WithContext(context.Background())
	rec := httptest.NewRecorder()
	mw.ServeHTTP(rec, req)

	if !called {
		t.Errorf("fallback halo_infinite devrait laisser passer CapCareer (status=%d)", rec.Code)
	}
}
