package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games/mappings"
)

// newAdminTitlesHandlerForTest construit le handler sur un vrai title.Registry
// (qui enregistre halo_infinite) + un stub de capabilities (réutilisé de
// capabilities_test.go).
func newAdminTitlesHandlerForTest(set *mappings.CapabilityMappingSet) *AdminTitlesHandler {
	return NewAdminTitlesHandler(titlePkg.NewRegistry(), &stubCapabilitiesRegistry{set: set}, nil)
}

func TestAdminTitles_List(t *testing.T) {
	t.Parallel()
	set := mappings.NewCapabilityMappingSet(titlePkg.DefaultSlug, 1, map[string]string{
		"match.history": mappings.CapStatusSupported,
	})
	r := chi.NewRouter()
	r.Get("/admin/titles", newAdminTitlesHandlerForTest(set).List)

	req := httptest.NewRequest("GET", "/admin/titles", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", w.Code, w.Body.String())
	}
	var body adminTitlesListResponse
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if body.Count < 1 || len(body.Titles) != body.Count {
		t.Fatalf("count incohérent: count=%d len=%d", body.Count, len(body.Titles))
	}
	found := false
	for _, ti := range body.Titles {
		if ti.Slug == titlePkg.DefaultSlug {
			found = true
			if ti.Status != titlePkg.StatusActive {
				t.Errorf("halo_infinite status = %q, want %q (MT-22 : Status lu/exposé)", ti.Status, titlePkg.StatusActive)
			}
			if !ti.HasMappings {
				t.Errorf("halo_infinite has_mappings = false, want true (caps présentes)")
			}
		}
	}
	if !found {
		t.Errorf("halo_infinite absent de la liste")
	}
}

func TestAdminTitles_Detail_OK(t *testing.T) {
	t.Parallel()
	set := mappings.NewCapabilityMappingSet(titlePkg.DefaultSlug, 2, map[string]string{
		"match.history":        mappings.CapStatusSupported,
		"analytics.timeseries": mappings.CapStatusNotExposed,
	})
	r := chi.NewRouter()
	r.Get("/admin/titles/{slug}", newAdminTitlesHandlerForTest(set).Detail)

	req := httptest.NewRequest("GET", "/admin/titles/"+titlePkg.DefaultSlug, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", w.Code, w.Body.String())
	}
	var body adminTitleDetail
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if body.Slug != titlePkg.DefaultSlug || !body.HasMappings || body.SchemaVersion != 2 {
		t.Errorf("detail meta: slug=%q has_mappings=%v schema=%d", body.Slug, body.HasMappings, body.SchemaVersion)
	}
	if body.DeclaredCapabilities["match.history"] != mappings.CapStatusSupported {
		t.Errorf("declared match.history = %q, want supported", body.DeclaredCapabilities["match.history"])
	}
	// La feature-matrix doit être calculée (réutilisation de games.ComputeFeatureMatrix).
	if len(body.FeatureMatrix) == 0 {
		t.Errorf("feature_matrix vide (attendu calculé via 1.7b)")
	}
}

func TestAdminTitles_Detail_NotFound(t *testing.T) {
	t.Parallel()
	r := chi.NewRouter()
	r.Get("/admin/titles/{slug}", newAdminTitlesHandlerForTest(nil).Detail)

	req := httptest.NewRequest("GET", "/admin/titles/does_not_exist", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404, body=%s", w.Code, w.Body.String())
	}
}

// Détail d'un titre EXISTANT mais sans capabilities.toml chargé : 200, mais
// has_mappings=false et feature_matrix omise (dégradation propre).
func TestAdminTitles_Detail_NoMappings(t *testing.T) {
	t.Parallel()
	r := chi.NewRouter()
	r.Get("/admin/titles/{slug}", newAdminTitlesHandlerForTest(nil).Detail)

	req := httptest.NewRequest("GET", "/admin/titles/"+titlePkg.DefaultSlug, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", w.Code, w.Body.String())
	}
	var body adminTitleDetail
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if body.Slug != titlePkg.DefaultSlug {
		t.Errorf("slug = %q, want %q", body.Slug, titlePkg.DefaultSlug)
	}
	if body.HasMappings {
		t.Errorf("has_mappings = true, want false (pas de caps chargées)")
	}
	if len(body.FeatureMatrix) != 0 {
		t.Errorf("feature_matrix non vide alors qu'aucune capability n'est chargée")
	}
}
