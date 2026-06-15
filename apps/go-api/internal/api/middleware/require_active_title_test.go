// Package middleware_test — require_active_title_test.go : gate MT-22 (PMT-8).
//
// Oracle DOUBLE du parity-gate PMT-8 :
//   - (a) Parité Halo : un titre StatusActive (halo_infinite) passe le gate
//     sans changement observable (next appelé, pas de 503).
//   - (b) Synthetic : un descripteur synthetic_b coming_soon ET un archived
//     routent vers 503 title_unavailable avec le bon status ; dégradation
//     propre (body machine-readable, jamais panic/500). C'est la seule preuve
//     que le seam route vraiment sur Status.
package middleware_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"levelup/go-api/internal/api/middleware"
	"levelup/go-api/internal/ctxkeys"
	titlePkg "levelup/go-api/internal/domain/title"
)

// registryWithSynthetic enregistre, en plus de halo_infinite (actif), un titre
// coming_soon et un titre archived — fixtures de l'oracle (b).
func registryWithSynthetic() *titlePkg.Registry {
	reg := titlePkg.NewRegistry()
	reg.Register(&titlePkg.TitleDescriptor{
		Slug: "synthetic_b", Name: "Synthetic B", Status: titlePkg.StatusComingSoon,
	})
	reg.Register(&titlePkg.TitleDescriptor{
		Slug: "synthetic_archived", Name: "Synthetic Archived", Status: titlePkg.StatusArchived,
	})
	return reg
}

// invokeRequireActiveTitle exécute le gate avec un title_slug injecté dans le
// contexte (comme le ferait TitleExtractor) et retourne le recorder + si next a
// été appelé.
func invokeRequireActiveTitle(reg *titlePkg.Registry, slug string) (*httptest.ResponseRecorder, bool) {
	nextCalled := false
	gate := middleware.RequireActiveTitle(reg)
	h := gate(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/players/test-player/pages/career", nil)
	req = req.WithContext(ctxkeys.WithTitleSlug(req.Context(), slug))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	return w, nextCalled
}

// TestRequireActiveTitle_ActivePasses — oracle (a) : halo_infinite (actif) passe.
func TestRequireActiveTitle_ActivePasses(t *testing.T) {
	w, nextCalled := invokeRequireActiveTitle(registryWithSynthetic(), titlePkg.DefaultSlug)
	if !nextCalled {
		t.Fatal("titre actif : next doit être appelé")
	}
	if w.Code != http.StatusOK {
		t.Errorf("titre actif : attendu 200, got %d", w.Code)
	}
}

// TestRequireActiveTitle_RejectsNonActive — oracle (b) : coming_soon, archived
// et inconnu → 503 title_unavailable avec le bon status, body machine-readable.
func TestRequireActiveTitle_RejectsNonActive(t *testing.T) {
	cases := []struct {
		name       string
		slug       string
		wantStatus string
	}{
		{"coming_soon", "synthetic_b", "coming_soon"},
		{"archived", "synthetic_archived", "archived"},
		{"unknown", "title_inexistant_xyz", "unknown"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w, nextCalled := invokeRequireActiveTitle(registryWithSynthetic(), tc.slug)
			if nextCalled {
				t.Fatalf("%s : next ne doit PAS être appelé", tc.name)
			}
			if w.Code != http.StatusServiceUnavailable {
				t.Fatalf("%s : attendu 503, got %d", tc.name, w.Code)
			}
			var body map[string]any
			if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
				t.Fatalf("%s : body non-JSON (dégradation impropre) : %v", tc.name, err)
			}
			if body["code"] != "title_unavailable" {
				t.Errorf("%s : code attendu title_unavailable, got %v", tc.name, body["code"])
			}
			if body["status"] != tc.wantStatus {
				t.Errorf("%s : status attendu %q, got %v", tc.name, tc.wantStatus, body["status"])
			}
			if body["title_slug"] != tc.slug {
				t.Errorf("%s : title_slug attendu %q, got %v", tc.name, tc.slug, body["title_slug"])
			}
			if body["retryable"] != false {
				t.Errorf("%s : retryable attendu false, got %v", tc.name, body["retryable"])
			}
			if msg, ok := body["message"].(string); !ok || msg == "" {
				t.Errorf("%s : message utilisateur manquant", tc.name)
			}
		})
	}
}

// TestRegistry_ActiveExcludesNonActive — Active() exclut coming_soon ET archived ;
// NonArchived() garde coming_soon mais exclut archived (switcher).
func TestRegistry_ActiveVsNonArchived(t *testing.T) {
	reg := registryWithSynthetic()

	active := reg.Active()
	for _, td := range active {
		if td.Status != titlePkg.StatusActive {
			t.Errorf("Active() ne doit contenir que des actifs, got %s (%s)", td.Slug, td.Status)
		}
	}

	var hasComingSoon, hasArchived bool
	for _, td := range reg.NonArchived() {
		switch td.Slug {
		case "synthetic_b":
			hasComingSoon = true
		case "synthetic_archived":
			hasArchived = true
		}
	}
	if !hasComingSoon {
		t.Error("NonArchived() doit inclure le titre coming_soon")
	}
	if hasArchived {
		t.Error("NonArchived() ne doit PAS inclure le titre archived")
	}
}
