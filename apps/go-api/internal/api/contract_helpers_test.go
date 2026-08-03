// Package api_test — helpers de test pour contract_test.go (Sprint 29).
//
// buildTestRouter crée un routeur chi minimal en mode démo.
// Nécessite CGO=1 (dépendance transitive platform/duckdb via handlers).

//go:build cgo

package api_test

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/danielgtaylor/huma/v2"

	"levelup/go-api/internal/api/openapigen"
)

// ---------------------------------------------------------------------------
// buildTestRouter — construit le routeur chi en mode démo pour les tests de contrat.
//
// Le harnais VIT dans internal/api/openapigen (BuildDemoRouter) : c'est le MÊME
// assemblage que celui qui génère api/openapi.yaml (V72-01 / H6), donc les tests
// de contrat et le contrat publié décrivent exactement la même surface.
//
// Caractéristiques :
//   - DemoMode = true → pas d'accès DuckDB
//   - répertoire temporaire isolé (sessions, app_settings.json, data/cache)
//   - dépôt bootstrap bouchonné → /health et /bootstrap n'ont pas besoin de DB
// ---------------------------------------------------------------------------

func buildTestRouter(t *testing.T) http.Handler {
	t.Helper()
	router, _ := buildTestRouterWithDoc(t)
	return router
}

// buildTestRouterWithDoc construit le routeur de test ET retourne le document
// OpenAPI PARTAGÉ (V72-01 / H1) pour les tests qui vérifient la fidélité du
// document fusionné (cf. shared_openapi_doc_test.go).
func buildTestRouterWithDoc(t *testing.T) (http.Handler, *huma.OpenAPI) {
	t.Helper()
	router, doc, err := openapigen.BuildDemoRouter(t.Context(), openapigen.Options{
		Root: t.TempDir(),
		// Store de groupes partagé par les tests de contrat (jamais lu en démo).
		GroupStorePath: filepath.Join(os.TempDir(), "levelup_contract_groups.json"),
		// Les routes multi-titres n'ont plus de flag depuis le 2026-08-02 : elles
		// sont montées inconditionnellement par mountAPIV1.
		Prestige: parseBoolEnvFlag("PRESTIGE_ENABLED", true),
	})
	if err != nil {
		t.Fatalf("buildTestRouter: %v", err)
	}
	return router, doc
}

// parseBoolEnvFlag lit une variable d'environnement comme booléen.
// Retourne defaultVal si la variable est absente.
// Valeurs falsy reconnues : "0", "false", "no", "off".
// Valeurs truthy reconnues : "1", "true", "yes" (et tout le reste si defaultVal=true).
func parseBoolEnvFlag(key string, defaultVal bool) bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv(key)))
	if v == "" {
		return defaultVal
	}
	if defaultVal {
		switch v {
		case "0", "false", "no", "off":
			return false
		}
		return true
	}
	return v == "1" || v == "true" || v == "yes"
}
