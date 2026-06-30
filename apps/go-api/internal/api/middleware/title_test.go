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

// MT-22 (PMT-8) : le SEAM résout n'importe quel titre CONNU, y compris
// coming_soon — c'est le gate RequireActiveTitle (et non le seam) qui rejette
// un titre non-actif en 503. Ici on prouve que le titre coming_soon demandé est
// bien injecté dans le contexte (pas masqué par un fallback silencieux).
func TestTitleExtractor_ComingSoonHeader_Resolved(t *testing.T) {
	registry := titlePkg.NewRegistry()
	registry.Register(&titlePkg.TitleDescriptor{
		Slug: "futur_titre", Name: "Futur", Status: titlePkg.StatusComingSoon,
	})
	mw := middleware.TitleExtractor(registry)

	var captured string
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured = ctxkeys.TitleSlug(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-LevelUp-Title", "futur_titre")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if captured != "futur_titre" {
		t.Errorf("le seam doit résoudre le titre coming_soon connu, attendu %q, got %q",
			"futur_titre", captured)
	}
}

// La locale UI est extraite du header X-LevelUp-Locale et placée dans le contexte
// (ctxkeys.Locale) — alimente les lectures localisées (noms de commendations H5).
func TestTitleExtractor_LocaleFromHeader(t *testing.T) {
	cases := []struct {
		header string
		want   string
	}{
		{"en", "en"},
		{"en-US", "en"},
		{"fr", "fr"},
		{"fr-FR", "fr"},
		{"", "fr"},   // absent → défaut fr
		{"de", "fr"}, // inconnue → défaut fr
	}
	mw := middleware.TitleExtractor(titlePkg.NewRegistry())
	for _, c := range cases {
		var captured string
		handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			captured = ctxkeys.Locale(r.Context())
			w.WriteHeader(http.StatusOK)
		}))
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		if c.header != "" {
			req.Header.Set("X-LevelUp-Locale", c.header)
		}
		handler.ServeHTTP(httptest.NewRecorder(), req)
		if captured != c.want {
			t.Errorf("X-LevelUp-Locale=%q → locale %q, want %q", c.header, captured, c.want)
		}
	}
}
