// demo_paths_test.go — ratchet : le mode démo sert la FIXTURE, quoi qu'il arrive.
//
// Régression 2026-08-05 : la bascule vers la fixture était conditionnée à
// l'absence de la DB de production. Sur un poste ayant ses données réelles — donc
// tout poste de développement — la démo servait la production, `demo-player`
// n'y avait aucun match, et le harnais visuel skippait 6 pages sur 7 en annonçant
// « données absentes » (faux vert : zéro diff sur des pages vides). Le lancement
// de démo appliquait en prime les migrations de boot aux bases de production.
package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDemoWarehouseDBPath_IgnoresProductionDBPresence(t *testing.T) {
	fixtures := t.TempDir()
	prodDir := t.TempDir()

	prodShared := filepath.Join(prodDir, "data", "titles", "halo_infinite", "warehouse", "shared_matches_v2.duckdb")
	if err := os.MkdirAll(filepath.Dir(prodShared), 0o755); err != nil {
		t.Fatalf("mkdir prod: %v", err)
	}

	// Cas 1 : DB de production ABSENTE.
	absent := demoWarehouseDBPath(fixtures, prodShared)

	// Cas 2 : DB de production PRÉSENTE — c'est le cas qui régressait.
	if err := os.WriteFile(prodShared, []byte("donnees de production"), 0o600); err != nil {
		t.Fatalf("write prod db: %v", err)
	}
	present := demoWarehouseDBPath(fixtures, prodShared)

	if absent != present {
		t.Fatalf("la résolution démo dépend de l'existence de la DB de production :\n  absente = %s\n  présente = %s",
			absent, present)
	}
	want := filepath.Join(fixtures, "warehouse", "shared_matches_v2.duckdb")
	if present != want {
		t.Errorf("chemin démo attendu %s, obtenu %s", want, present)
	}
	if strings.HasPrefix(present, prodDir) {
		t.Errorf("le mode démo pointe l'arborescence de PRODUCTION : %s", present)
	}
}

// TestDemoWarehouseDBPath_AllWarehouseDBs : les quatre bases du warehouse
// doivent être traduites. `shared_social` et `shared_pve` étaient absents de la
// liste d'origine — la démo lisait et MIGRAIT donc les bases sociales et PvE de
// production même quand shared/metadata étaient correctement redirigés.
func TestDemoWarehouseDBPath_AllWarehouseDBs(t *testing.T) {
	fixtures := filepath.Join("X:", "fixture")
	prodBase := filepath.Join("Y:", "data", "titles", "halo_infinite", "warehouse")

	for _, name := range []string{
		"shared_matches_v2.duckdb",
		"metadata.duckdb",
		"shared_social.duckdb",
		"shared_pve.duckdb",
	} {
		got := demoWarehouseDBPath(fixtures, filepath.Join(prodBase, name))
		want := filepath.Join(fixtures, "warehouse", name)
		if got != want {
			t.Errorf("%s : attendu %s, obtenu %s", name, want, got)
		}
	}
}
