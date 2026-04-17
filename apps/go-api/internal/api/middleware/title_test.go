// Package middleware_test — title_test.go : tests du middleware TitleExtractor.
package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"levelup/go-api/internal/api/middleware"
	"levelup/go-api/internal/ctxkeys"
	titlePkg "levelup/go-api/internal/domain/title"
)

func TestTitleExtractor_FromHeader(t *testing.T) {
	registry := titlePkg.NewRegistry()
	mw := middleware.TitleExtractor(registry)

	var captured string
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured = ctxkeys.TitleSlug(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-LevelUp-Title", "halo_infinite")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if captured != "halo_infinite" {
		t.Errorf("expected halo_infinite from header, got %q", captured)
	}
}

func TestTitleExtractor_InvalidHeader_Fallback(t *testing.T) {
	registry := titlePkg.NewRegistry()
	mw := middleware.TitleExtractor(registry)

	var captured string
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured = ctxkeys.TitleSlug(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-LevelUp-Title", "nonexistent_game")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if captured != titlePkg.DefaultSlug {
		t.Errorf("expected fallback %q for invalid header, got %q", titlePkg.DefaultSlug, captured)
	}
}

func TestTitleExtractor_NoHeader_DefaultFallback(t *testing.T) {
	registry := titlePkg.NewRegistry()
	mw := middleware.TitleExtractor(registry)

	var captured string
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured = ctxkeys.TitleSlug(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if captured != titlePkg.DefaultSlug {
		t.Errorf("expected default slug %q, got %q", titlePkg.DefaultSlug, captured)
	}
}
