package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/middleware"
	"levelup/go-api/internal/domain"
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

// TestAdminTitles_TOMLDraft — PMT-14 (D10) : le draft reprend les capabilities
// déclarées + le schema_version, et 404 pour un titre inconnu.
func TestAdminTitles_TOMLDraft(t *testing.T) {
	set := mappings.NewCapabilityMappingSet(titlePkg.DefaultSlug, 3, map[string]string{
		"match.history": mappings.CapStatusSupported,
	})
	h := newAdminTitlesHandlerForTest(set)
	r := chi.NewRouter()
	r.Get("/admin/titles/{slug}/toml-draft", h.TOMLDraft)

	req := httptest.NewRequest("GET", "/admin/titles/"+titlePkg.DefaultSlug+"/toml-draft", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d want 200, body=%s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	if !strings.Contains(body, "[capabilities]") || !strings.Contains(body, "schema_version = 3") {
		t.Errorf("draft incomplet: %q", body)
	}
	if !strings.Contains(body, `"match.history" = "supported"`) {
		t.Errorf("draft doit reprendre les capabilities déclarées, got: %q", body)
	}

	req404 := httptest.NewRequest("GET", "/admin/titles/inconnu_xyz/toml-draft", nil)
	w404 := httptest.NewRecorder()
	r.ServeHTTP(w404, req404)
	if w404.Code != http.StatusNotFound {
		t.Errorf("titre inconnu: status=%d want 404", w404.Code)
	}
}

// TestAdminTitles_NoOsWriteFile_D10 — garde-fou D10 : aucune écriture serveur
// dans les handlers titres (le draft est presse-papier côté front uniquement).
func TestAdminTitles_NoOsWriteFile_D10(t *testing.T) {
	for _, f := range []string{"admin_titles.go", "admin_title_diagnostic.go"} {
		src, err := os.ReadFile(f)
		if err != nil {
			t.Fatalf("lecture %s: %v", f, err)
		}
		if strings.Contains(string(src), "os.WriteFile") {
			t.Errorf("%s contient os.WriteFile — viole D10 (zéro écriture serveur dans les handlers titres)", f)
		}
	}
}

// TestAdminTitles_Gating_401_403 — PMT-14 : les routes admin titres sont gatées
// par la MÊME chaîne que server.go (RequireAuth + RequireAdmin) → 401 sans
// session, 403 non-admin, 200 admin.
func TestAdminTitles_Gating_401_403(t *testing.T) {
	set := mappings.NewCapabilityMappingSet(titlePkg.DefaultSlug, 1, map[string]string{
		"match.history": mappings.CapStatusSupported,
	})
	h := newAdminTitlesHandlerForTest(set)
	r := chi.NewRouter()
	r.Group(func(r chi.Router) {
		r.Use(middleware.RequireAuth(false, "password"))
		r.Use(middleware.RequireAdmin(false, "password"))
		r.Get("/admin/titles", h.List)
	})

	serve := func(sess *domain.SessionData) int {
		req := httptest.NewRequest("GET", "/admin/titles", nil)
		if sess != nil {
			req = req.WithContext(middleware.InjectSession(req.Context(), sess))
		}
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		return w.Code
	}

	if code := serve(nil); code != http.StatusUnauthorized {
		t.Errorf("sans session : status=%d, want 401", code)
	}

	user := "bob"
	roleUser := "user"
	if code := serve(&domain.SessionData{Username: &user, Role: &roleUser}); code != http.StatusForbidden {
		t.Errorf("non-admin : status=%d, want 403", code)
	}

	roleAdmin := "admin"
	if code := serve(&domain.SessionData{Username: &user, Role: &roleAdmin}); code != http.StatusOK {
		t.Errorf("admin : status=%d, want 200", code)
	}
}

// TestAdminTitles_SyntheticTitle_ListedWithDegradation — ORACLE (b) PMT-14 :
// enregistrer synthetic_title_b (coming_soon, sans capabilities.toml) → la liste
// le montre avec son Status ; le détail dégrade proprement (200, has_mappings
// false, feature_matrix omise, zéro panic). Preuve que la surface route vraiment
// sur le registre et pas sur halo_infinite en dur.
func TestAdminTitles_SyntheticTitle_ListedWithDegradation(t *testing.T) {
	reg := titlePkg.NewRegistry()
	reg.Register(&titlePkg.TitleDescriptor{
		Slug: "synthetic_title_b", Name: "Synthetic B", Status: titlePkg.StatusComingSoon,
	})
	// Caps pour HI uniquement → synthetic_title_b n'a pas de mappings (dégradation).
	set := mappings.NewCapabilityMappingSet(titlePkg.DefaultSlug, 1, map[string]string{
		"match.history": mappings.CapStatusSupported,
	})
	h := NewAdminTitlesHandler(reg, &stubCapabilitiesRegistry{set: set}, nil)

	// Liste : synthetic_title_b présent avec son Status coming_soon, has_mappings=false.
	rl := chi.NewRouter()
	rl.Get("/admin/titles", h.List)
	reqL := httptest.NewRequest("GET", "/admin/titles", nil)
	wL := httptest.NewRecorder()
	rl.ServeHTTP(wL, reqL)
	var list adminTitlesListResponse
	if err := json.Unmarshal(wL.Body.Bytes(), &list); err != nil {
		t.Fatalf("unmarshal list: %v", err)
	}
	var syn *adminTitleSummary
	for i := range list.Titles {
		if list.Titles[i].Slug == "synthetic_title_b" {
			syn = &list.Titles[i]
		}
	}
	if syn == nil {
		t.Fatal("synthetic_title_b absent de la liste (la surface ne route pas sur le registre)")
	}
	if syn.Status != titlePkg.StatusComingSoon {
		t.Errorf("synthetic_title_b status=%q, want coming_soon", syn.Status)
	}
	if syn.HasMappings {
		t.Errorf("synthetic_title_b has_mappings=true, want false (pas de capabilities.toml)")
	}

	// Détail : dégradation propre (200, pas de panic, feature_matrix vide).
	rd := chi.NewRouter()
	rd.Get("/admin/titles/{slug}", h.Detail)
	reqD := httptest.NewRequest("GET", "/admin/titles/synthetic_title_b", nil)
	wD := httptest.NewRecorder()
	rd.ServeHTTP(wD, reqD)
	if wD.Code != http.StatusOK {
		t.Fatalf("détail synthetic : status=%d, want 200", wD.Code)
	}
	var detail adminTitleDetail
	if err := json.Unmarshal(wD.Body.Bytes(), &detail); err != nil {
		t.Fatalf("unmarshal detail: %v", err)
	}
	if detail.HasMappings || len(detail.FeatureMatrix) != 0 {
		t.Errorf("dégradation impropre : has_mappings=%v feature_matrix=%d", detail.HasMappings, len(detail.FeatureMatrix))
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
