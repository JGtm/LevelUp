package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/games/mappings"
)

func newFeatureMatrixHandler() *FeatureMatrixHandler {
	set := mappings.NewCapabilityMappingSet("test_title", 1, map[string]string{
		"match.history":        mappings.CapStatusSupported,
		"pve.firefight_stats":  mappings.CapStatusSupported,
		"analytics.timeseries": mappings.CapStatusNotExposed,
	})
	return NewFeatureMatrixHandler(&stubCapabilitiesRegistry{set: set}, nil)
}

func TestFeatureMatrixHandler_Success(t *testing.T) {
	t.Parallel()
	r := chi.NewRouter()
	r.Route("/api/v1", func(r chi.Router) { newFeatureMatrixHandler().Mount(r) })

	req := httptest.NewRequest("GET", "/api/v1/titles/test_title/feature-matrix", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", w.Code, w.Body.String())
	}
	var body featureMatrixResponse
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if body.TitleSlug != "test_title" || body.SchemaVersion != 1 {
		t.Errorf("meta: got (%q, %d), want (test_title, 1)", body.TitleSlug, body.SchemaVersion)
	}
	// Cascade attendue depuis les caps stub.
	checks := map[string]string{
		"match_history": "available",
		"pve_stats":     "available",
		"timeseries":    "unavailable", // analytics.timeseries=not_exposed
		"citations":     "unavailable", // capability absente du set
	}
	for feat, want := range checks {
		if got := body.Features[feat]; got != want {
			t.Errorf("feature %q = %q, want %q", feat, got, want)
		}
	}
	// Headers (cohérence endpoints frères).
	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}
	if w.Header().Get("ETag") == "" {
		t.Errorf("ETag absent")
	}
	if cc := w.Header().Get("Cache-Control"); cc != "public, max-age=300" {
		t.Errorf("Cache-Control = %q", cc)
	}
}

func TestFeatureMatrixHandler_NotFound(t *testing.T) {
	t.Parallel()
	r := chi.NewRouter()
	r.Route("/api/v1", func(r chi.Router) { newFeatureMatrixHandler().Mount(r) })

	req := httptest.NewRequest("GET", "/api/v1/titles/unknown/feature-matrix", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", w.Code)
	}
}

func TestFeatureMatrixHandler_ETag304(t *testing.T) {
	t.Parallel()
	r := chi.NewRouter()
	r.Route("/api/v1", func(r chi.Router) { newFeatureMatrixHandler().Mount(r) })

	req := httptest.NewRequest("GET", "/api/v1/titles/test_title/feature-matrix", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	etag := w.Header().Get("ETag")
	if etag == "" {
		t.Fatalf("ETag absent au 1er appel")
	}

	req2 := httptest.NewRequest("GET", "/api/v1/titles/test_title/feature-matrix", nil)
	req2.Header.Set("If-None-Match", etag)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	if w2.Code != http.StatusNotModified {
		t.Errorf("status = %d, want 304", w2.Code)
	}
}

// TestFeatureMatrixHandler_EmptySlug appelle handleGet directement avec un slug
// vide (le routeur ne matcherait pas) pour exercer le garde 400.
func TestFeatureMatrixHandler_EmptySlug(t *testing.T) {
	t.Parallel()
	_, err := newFeatureMatrixHandler().handleGet(context.Background(), &featureMatrixInput{Slug: ""})
	se, ok := err.(huma.StatusError)
	if !ok {
		t.Fatalf("erreur = %v, want huma.StatusError", err)
	}
	if se.GetStatus() != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", se.GetStatus())
	}
}

// TestFeatureMatrixHandler_InvalidCapabilities : un TOML déclarant une capability
// hors vocabulaire produit → CapabilityMapFromMappings échoue → 500.
func TestFeatureMatrixHandler_InvalidCapabilities(t *testing.T) {
	t.Parallel()
	set := mappings.NewCapabilityMappingSet("bad_title", 1, map[string]string{
		"made.up.capability": mappings.CapStatusSupported,
	})
	h := NewFeatureMatrixHandler(&stubCapabilitiesRegistry{set: set}, nil)

	r := chi.NewRouter()
	r.Route("/api/v1", func(r chi.Router) { h.Mount(r) })
	req := httptest.NewRequest("GET", "/api/v1/titles/bad_title/feature-matrix", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", w.Code)
	}
}
