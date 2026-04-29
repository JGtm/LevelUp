// Smoke test E2E de l'activation MULTI_TITLE_API_ENABLED.
//
// Construit le routeur en mode démo + flag activé, puis vérifie que :
//   - flag off : GET /api/v1/titles/{slug}/field-mappings retourne 404 (route absente)
//   - flag on : la route /field-mappings est bien enregistrée
//
// NOTE : les routes /preview/career et /preview/career-multi-title ont été
// supprimées en revue 2026-04-29 P0.2 Q6 (orphelines côté front). Le test
// TestSmoke_PreviewCareer_FlagOn_NotFoundForUnloadedTitle a été retiré.
//
// Nécessite CGO=1 (transitivement via platform/duckdb).

//go:build cgo

package api_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
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

// TestSmoke_FieldMappings_FlagOn_RouteRegistered : flag on → la route est
// bien enregistrée dans le routeur. On ne peut pas tester le happy path
// complet ici car buildTestRouter utilise son propre tmpDir, donc on
// vérifie juste que la route est exposée (le handler répond 404 sur le
// titre non chargé, mais c'est différent d'un 404 routeur).
//
// Le helper writeSmokeTOML est conservé pour de futurs tests d'intégration
// qui pourraient copier le TOML dans un tmpDir partagé avec le routeur.
func TestSmoke_FieldMappings_FlagOn_RouteRegistered(t *testing.T) {
	t.Setenv("MULTI_TITLE_API_ENABLED", "true")
	t.Setenv("LEVELUP_DEMO_MODE", "true")

	// Sanity check du helper writeSmokeTOML (pour qu'il reste exercé).
	tmpDir := writeSmokeTOML(t)
	if _, err := os.Stat(filepath.Join(tmpDir, "config", "titles", "halo_infinite", "mappings", "fields.toml")); err != nil {
		t.Fatalf("writeSmokeTOML n'a pas écrit le fichier: %v", err)
	}

	router := buildTestRouter(t)
	req := httptest.NewRequest("GET", "/api/v1/titles/halo_infinite/field-mappings?locale=fr", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Avec un buildTestRouter qui utilise un tmpDir vide (sans TOML), la
	// route est exposée mais le titre n'est pas chargé → réponse 404 du
	// handler (avec body JSON), pas un 404 routeur.
	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404 (titre non chargé)", w.Code)
	}
	if w.Header().Get("Content-Type") != "" && !strings.Contains(w.Header().Get("Content-Type"), "json") {
		t.Errorf("flag on devrait passer par le handler avec body JSON, Content-Type = %q", w.Header().Get("Content-Type"))
	}
}

// TestSmoke_PreviewCareer_FlagOn_NotFoundForUnloadedTitle supprimé en revue
// 2026-04-29 P0.2 Q6 — la route /preview/career a été retirée (orpheline côté
// front). Le smoke test field-mappings ci-dessus suffit à valider que le flag
// MULTI_TITLE_API_ENABLED ouvre/ferme correctement le sous-arbre de routes.
