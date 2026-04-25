// Smoke test E2E de l'activation MULTI_TITLE_API_ENABLED.
//
// Construit le routeur en mode démo + flag activé, puis vérifie que :
//   - flag off : GET /api/v1/titles/{slug}/field-mappings retourne 404 (route absente)
//   - flag on : la route preview/career est bien enregistrée
//
// Nécessite CGO=1 (transitivement via platform/duckdb).

//go:build cgo

package api_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

const smokeFieldsTOML = `
[meta]
title_slug     = "halo_infinite"
schema_version = 1

[fields.kills]
labels        = { en = "Kills", fr = "Éliminations" }
storage_unit  = "count"
display_unit  = "count"
format        = "integer"
display_order = 10
group         = "combat"

[fields.win_rate]
labels        = { en = "Win Rate", fr = "Taux de victoire" }
storage_unit  = "ratio"
display_unit  = "percent"
format        = "percent_1"
display_order = 20
group         = "career"
`

// writeSmokeTOML écrit le TOML HI minimal dans le répertoire de test.
// Retourne le chemin du repoRoot configuré pour buildTestRouter.
func writeSmokeTOML(t *testing.T) string {
	t.Helper()
	tmpDir := t.TempDir()
	mappingsDir := filepath.Join(tmpDir, "config", "titles", "halo_infinite", "mappings")
	if err := os.MkdirAll(mappingsDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(mappingsDir, "fields.toml"), []byte(smokeFieldsTOML), 0o644); err != nil {
		t.Fatalf("write toml: %v", err)
	}
	return tmpDir
}

func TestSmoke_FieldMappings_FlagOff_RouteAbsent(t *testing.T) {
	// Flag explicitement à false : la route /titles/{slug}/field-mappings ne
	// doit même pas exister dans chi → 404 sans handler match.
	t.Setenv("MULTI_TITLE_API_ENABLED", "false")

	router := buildTestRouter(t)
	req := httptest.NewRequest("GET", "/api/v1/titles/halo_infinite/field-mappings?locale=fr", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("flag off → status = %d, want 404", w.Code)
	}
}

func TestSmoke_FieldMappings_FlagOn_HappyPath(t *testing.T) {
	t.Setenv("MULTI_TITLE_API_ENABLED", "true")
	t.Setenv("LEVELUP_DEMO_MODE", "true")

	// Note : buildTestRouter recrée son propre tmpDir → on doit forcer l'utilisation
	// du nôtre via une variante. Ici on fait simple : on smoke-teste avec
	// buildTestRouter standard (qui utilise un tmpDir vide), donc le registry
	// chargera 0 titre. L'endpoint répond alors 404 sur halo_infinite.
	// Pour un vrai smoke test "happy path", il faut un test d'intégration
	// dédié qui copie le TOML du repo. Voir tests Phase A
	// (handlers/field_mappings_test.go) qui couvrent le happy path avec stub.
	t.Skip("smoke happy path nécessite un repoRoot peuplé — couvert par TestFieldMappingsHandler_Success_FR dans handlers/")
}

func TestSmoke_PreviewCareer_FlagOn_NotFoundForUnloadedTitle(t *testing.T) {
	t.Setenv("MULTI_TITLE_API_ENABLED", "true")
	t.Setenv("LEVELUP_DEMO_MODE", "true")

	router := buildTestRouter(t)

	// Sans TOML chargé, le resolver Semantic n'a rien → 404 attendu (ou 401
	// selon middleware). On teste juste que la route est bien enregistrée.
	req := httptest.NewRequest("GET", "/api/v1/titles/halo_infinite/preview/career?xuid=0xABC", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code == http.StatusNotFound {
		// Si la route n'est pas enregistrée du tout, on aurait un 404 sans body JSON.
		// Sinon, on attend un 404 du handler (title_not_found / title_semantic_not_found).
		var body map[string]any
		if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
			// 404 sans JSON → la route n'est pas enregistrée. Échec.
			t.Errorf("flag on mais route /preview/career absente du routeur : body=%q", w.Body.String())
		}
	}
}
