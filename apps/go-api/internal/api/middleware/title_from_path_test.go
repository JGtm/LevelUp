package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/ctxkeys"
)

func TestTitleSlugFromPath_SetsCtxFromPathParam(t *testing.T) {
	r := chi.NewRouter()
	var got string
	r.Route("/titles/{slug}", func(r chi.Router) {
		r.Use(TitleSlugFromPath("slug"))
		r.Get("/x", func(w http.ResponseWriter, req *http.Request) {
			got = ctxkeys.TitleSlug(req.Context())
			w.WriteHeader(http.StatusOK)
		})
	})

	req := httptest.NewRequest(http.MethodGet, "/titles/halo_5/x", nil)
	r.ServeHTTP(httptest.NewRecorder(), req)

	if got != "halo_5" {
		t.Fatalf("ctx title attendu halo_5 (depuis le path), reçu %q", got)
	}
}

func TestTitleSlugFromPath_EmptyParam_LeavesFallback(t *testing.T) {
	r := chi.NewRouter()
	var got string
	// Pas de {slug} dans le motif → param vide → fallback ctxkeys (halo_infinite).
	r.With(TitleSlugFromPath("slug")).Get("/y", func(w http.ResponseWriter, req *http.Request) {
		got = ctxkeys.TitleSlug(req.Context())
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/y", nil)
	r.ServeHTTP(httptest.NewRecorder(), req)

	if got != "halo_infinite" {
		t.Fatalf("fallback attendu halo_infinite, reçu %q", got)
	}
}
