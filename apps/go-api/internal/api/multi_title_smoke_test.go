// Smoke test E2E de l'INVARIANCE des routes multi-titres.
//
// Historique : ce fichier testait l'ouverture/fermeture du sous-arbre de routes
// par le flag de rollout MULTI_TITLE_API_ENABLED (un test « flag off → 404
// routeur », un test « flag on → route enregistrée »). Le flag a été retiré le
// 2026-08-02 (v7.3 lot 2, item 3.3) : les libellés servis par
// /titles/{slug}/field-mappings sont devenus la SOURCE UNIQUE du front, dont les
// dictionnaires de repli TS ont été supprimés dans le même commit. Une route
// éteignable servirait désormais des clés brutes à l'écran.
//
// Ce que ces tests protègent maintenant : la route est montée SANS aucune
// variable d'environnement, et le reste même si l'ancien nom de flag est
// repositionné à "false". Le second test est le garde-rail DISCRIMINANT du
// retrait : il redevient rouge si quelqu'un réintroduit un gate.
//
// NOTE : les routes /preview/career et /preview/career-multi-title
// (proof-of-concept Phase C) ont été supprimées en revue 2026-04-29 P0.2 Q6
// (orphelines côté front).
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

// assertFieldMappingsRouteMounted vérifie que la route field-mappings est bien
// ENREGISTRÉE dans chi. buildTestRouter utilise un tmpDir sans TOML, donc le
// titre n'est pas chargé et le handler répond 404 — mais avec un corps JSON, ce
// qui le distingue du 404 de routeur (route absente, aucun handler atteint).
func assertFieldMappingsRouteMounted(t *testing.T, router http.Handler) {
	t.Helper()
	req := httptest.NewRequestWithContext(t.Context(), "GET", "/api/v1/titles/halo_infinite/field-mappings?locale=fr", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404 (titre non chargé dans le tmpDir de test)", w.Code)
	}
	ct := w.Header().Get("Content-Type")
	if !strings.Contains(ct, "json") {
		t.Errorf("route NON montée : le 404 vient du routeur et non du handler (Content-Type = %q, want JSON)", ct)
	}
}

// TestSmoke_FieldMappings_RouteMountedWithoutAnyFlag : aucun flag positionné →
// la route existe quand même. C'est la situation de tout poste de dev
// fraîchement cloné (avant le 2026-08-02, il servait des clés brutes faute de
// flag activé — c'est précisément ce qui maintenait les dictionnaires de repli
// TS en vie).
func TestSmoke_FieldMappings_RouteMountedWithoutAnyFlag(t *testing.T) {
	t.Setenv("LEVELUP_DEMO_MODE", "true")

	// Sanity check du helper writeSmokeTOML (pour qu'il reste exercé).
	tmpDir := writeSmokeTOML(t)
	if _, err := os.Stat(filepath.Join(tmpDir, "config", "titles", "halo_infinite", "mappings", "fields.toml")); err != nil {
		t.Fatalf("writeSmokeTOML n'a pas écrit le fichier: %v", err)
	}

	assertFieldMappingsRouteMounted(t, buildTestRouter(t))
}

// TestSmoke_FieldMappings_RouteMountedEvenWithLegacyFlagOff est le garde-rail
// DISCRIMINANT du retrait : l'ancien nom de flag posé à "false" ne doit plus
// avoir le moindre effet. Réintroduire un `if cfg.MultiTitleAPIEnabled` — ou
// tout autre gate lisant cette variable — rend ce test rouge.
func TestSmoke_FieldMappings_RouteMountedEvenWithLegacyFlagOff(t *testing.T) {
	t.Setenv("MULTI_TITLE_API_ENABLED", "false")
	t.Setenv("LEVELUP_DEMO_MODE", "true")

	assertFieldMappingsRouteMounted(t, buildTestRouter(t))
}
