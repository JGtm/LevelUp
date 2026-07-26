package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/config"
	session_platform "levelup/go-api/internal/platform/session"
)

// docsTestDoc — document OpenAPI minimal, suffisant pour la sérialisation.
func docsTestDoc() *huma.OpenAPI {
	return &huma.OpenAPI{
		OpenAPI:    "3.1.0",
		Info:       &huma.Info{Title: "LevelUp API", Version: "1.0.0"},
		Components: &huma.Components{},
	}
}

func docsRouter(t *testing.T, cfg *config.AppConfig) http.Handler {
	t.Helper()
	r := chi.NewRouter()
	mountOpenAPIDocs(r, cfg, docsTestDoc())
	return r
}

func getStatus(t *testing.T, h http.Handler, path string) (int, string, http.Header) {
	t.Helper()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
	return rec.Code, rec.Body.String(), rec.Header()
}

// TestMountOpenAPIDocs_DevServesUIAndSources — hors déploiement exposé, les trois
// routes répondent.
func TestMountOpenAPIDocs_DevServesUIAndSources(t *testing.T) {
	h := docsRouter(t, &config.AppConfig{})

	code, body, hdr := getStatus(t, h, docsUIPath)
	if code != http.StatusOK {
		t.Fatalf("GET %s = %d, attendu 200", docsUIPath, code)
	}
	if !strings.Contains(body, "elements-api") {
		t.Errorf("la page /docs ne contient pas le composant Stoplight")
	}
	if !strings.Contains(body, docsYAMLPath) {
		t.Errorf("la page /docs ne pointe pas sur %s", docsYAMLPath)
	}
	if csp := hdr.Get("Content-Security-Policy"); !strings.Contains(csp, "default-src 'none'") {
		t.Errorf("CSP absente ou permissive sur /docs : %q", csp)
	}

	code, body, hdr = getStatus(t, h, docsJSONPath)
	if code != http.StatusOK {
		t.Fatalf("GET %s = %d, attendu 200", docsJSONPath, code)
	}
	if ct := hdr.Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, attendu application/json", ct)
	}
	if !strings.Contains(body, `"openapi"`) {
		t.Errorf("corps JSON inattendu : %.120s", body)
	}

	code, body, _ = getStatus(t, h, docsYAMLPath)
	if code != http.StatusOK {
		t.Fatalf("GET %s = %d, attendu 200", docsYAMLPath, code)
	}
	if !strings.Contains(body, "openapi:") {
		t.Errorf("corps YAML inattendu : %.120s", body)
	}
}

// TestMountOpenAPIDocs_GatedOnExposedDeployments — production ET démo : aucune des
// trois routes n'est montée.
func TestMountOpenAPIDocs_GatedOnExposedDeployments(t *testing.T) {
	cases := map[string]*config.AppConfig{
		"production": {Environment: "production"},
		"demo":       {DemoMode: true},
	}
	for name, cfg := range cases {
		t.Run(name, func(t *testing.T) {
			h := docsRouter(t, cfg)
			for _, p := range []string{docsUIPath, docsJSONPath, docsYAMLPath} {
				if code, _, _ := getStatus(t, h, p); code != http.StatusNotFound {
					t.Errorf("GET %s = %d, attendu 404 (route ne doit pas être montée)", p, code)
				}
			}
		})
	}
}

// TestMountOpenAPIDocs_NilDocIsNoop — document nil : aucun montage, aucun panic.
func TestMountOpenAPIDocs_NilDocIsNoop(t *testing.T) {
	r := chi.NewRouter()
	mountOpenAPIDocs(r, &config.AppConfig{}, nil)
	if code, _, _ := getStatus(t, r, docsUIPath); code != http.StatusNotFound {
		t.Errorf("GET %s = %d, attendu 404", docsUIPath, code)
	}
}

// TestDeclareSecuritySchemes_UsesSessionCookieSource — le scheme décrit le cookie
// RÉELLEMENT posé (source unique session.CookieName), pas un littéral recopié.
func TestDeclareSecuritySchemes_UsesSessionCookieSource(t *testing.T) {
	doc := docsTestDoc()
	declareSecuritySchemes(doc)

	scheme := doc.Components.SecuritySchemes[SessionCookieSchemeName]
	if scheme == nil {
		t.Fatalf("securityScheme %q absent", SessionCookieSchemeName)
	}
	if scheme.Type != "apiKey" || scheme.In != "cookie" {
		t.Errorf("scheme = {type:%q, in:%q}, attendu {apiKey, cookie}", scheme.Type, scheme.In)
	}
	if scheme.Name != session_platform.CookieName {
		t.Errorf("nom du cookie = %q, attendu %q (source unique session.CookieName)",
			scheme.Name, session_platform.CookieName)
	}
	if scheme.Description == "" {
		t.Error("description du scheme vide")
	}
}

// TestDeclareSecuritySchemes_NilSafe — document ou Components nil : no-op sans panic.
func TestDeclareSecuritySchemes_NilSafe(t *testing.T) {
	declareSecuritySchemes(nil)
	declareSecuritySchemes(&huma.OpenAPI{})
}
