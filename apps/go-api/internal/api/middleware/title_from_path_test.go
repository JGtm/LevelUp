package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/ctxkeys"
	titlePkg "levelup/go-api/internal/domain/title"
)

// titleFromPathRouter monte TitleSlugFromPath sur /titles/{slug}/x et rend le titre
// vu par le handler (vide si le handler n'est pas atteint).
func titleFromPathRouter(reg *titlePkg.Registry, got *string) http.Handler {
	r := chi.NewRouter()
	r.Route("/titles/{slug}", func(r chi.Router) {
		r.Use(TitleSlugFromPath(reg, "slug"))
		r.Get("/x", func(w http.ResponseWriter, req *http.Request) {
			*got = ctxkeys.TitleSlug(req.Context())
			w.WriteHeader(http.StatusOK)
		})
	})
	return r
}

// Tout titre du registre, quel que soit son statut, est posé dans le contexte : la
// garde ne connaît aucun slug particulier.
func TestTitleSlugFromPath_SetsCtxFromPathParam(t *testing.T) {
	reg := titlePkg.NewRegistry()
	reg.Register(&titlePkg.TitleDescriptor{Slug: "synthetic_soon", Name: "Synthetic", Status: titlePkg.StatusComingSoon})
	for _, d := range reg.All() {
		t.Run(d.Slug, func(t *testing.T) {
			var got string
			rec := httptest.NewRecorder()
			titleFromPathRouter(reg, &got).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/titles/"+d.Slug+"/x", nil))
			if rec.Code != http.StatusOK || got != d.Slug {
				t.Fatalf("titre %q : code %d, contexte %q ; attendu 200 et le titre du chemin", d.Slug, rec.Code, got)
			}
		})
	}
}

// Un titre absent du registre (inconnu, casse différente, point, séparateur encodé)
// reçoit 404 title_not_found et n'atteint jamais le handler.
func TestTitleSlugFromPath_RefusesTitleOutsideRegistry(t *testing.T) {
	reg := titlePkg.NewRegistry()
	for _, seg := range []string{"titre_inconnu", "HALO_INCONNU", ".", "..", "%2E%2E", "a%2Fb", "a%5Cb", "%00"} {
		t.Run(seg, func(t *testing.T) {
			got := "non atteint"
			rec := httptest.NewRecorder()
			titleFromPathRouter(reg, &got).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/titles/"+seg+"/x", nil))
			if got != "non atteint" {
				t.Fatalf("titre %q : handler atteint avec le contexte %q", seg, got)
			}
			if rec.Code != http.StatusNotFound {
				t.Fatalf("titre %q : code %d, attendu 404", seg, rec.Code)
			}
			var body map[string]any
			if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil || body[errKeyCode] != "title_not_found" {
				t.Fatalf("titre %q : corps %q, attendu le code title_not_found", seg, rec.Body.String())
			}
		})
	}
}

func TestTitleSlugFromPath_EmptyParam_LeavesFallback(t *testing.T) {
	r := chi.NewRouter()
	var got string
	// Pas de {slug} dans le motif → param vide → fallback ctxkeys (titre par défaut).
	r.With(TitleSlugFromPath(titlePkg.NewRegistry(), "slug")).Get("/y", func(w http.ResponseWriter, req *http.Request) {
		got = ctxkeys.TitleSlug(req.Context())
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/y", nil)
	r.ServeHTTP(httptest.NewRecorder(), req)

	if got != titlePkg.DefaultSlug {
		t.Fatalf("fallback attendu %q, reçu %q", titlePkg.DefaultSlug, got)
	}
}
