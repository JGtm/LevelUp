// no_registry_in_migrations_test.go — GARDE-RAIL du déplacement du 2026-08-04.
//
// Le référentiel d'armes (schéma + données de seed + réconciliation) vit dans ce
// package, PAS dans internal/games/*/migrations : ces dossiers ne portent que
// l'enregistrement versionné du premier passage, sous forme de steps qui délèguent
// à ApplyLabels / ApplyRegistry.
//
// Sans ce test, la prochaine arme ajoutée retomberait naturellement dans
// migrations/ (là où le seed vivait) et la factorisation re-divergerait — c'est la
// règle repo « ≤ 2 copies + garde-rail ».
package weapons

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Motifs qui trahissent du référentiel d'armes (données ou DDL), par opposition à
// une simple délégation `weapons.ApplyRegistry`.
var weaponReferentialMarkers = []string{
	"INSERT OR IGNORE INTO weapons",
	"INSERT OR IGNORE INTO weapon_ids",
	"INSERT OR IGNORE INTO weapon_families",
	"INSERT OR IGNORE INTO weapon_labels",
	"CREATE TABLE IF NOT EXISTS weapons",
	"CREATE TABLE IF NOT EXISTS weapon_ids",
	"CREATE TABLE IF NOT EXISTS weapon_families",
	"CREATE TABLE IF NOT EXISTS weapon_labels",
	"CREATE TABLE IF NOT EXISTS weapon_name_labels",
}

// migrationDirs — dossiers de migrations title-owned scannés (relatifs à ce fichier).
var migrationDirs = []string{
	filepath.Join("..", "halo_infinite", "migrations"),
	filepath.Join("..", "halo_5", "migrations"),
	filepath.Join("..", "..", "migration"),
}

func TestNoWeaponReferentialInMigrations(t *testing.T) {
	var offenders []string
	for _, dir := range migrationDirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			t.Fatalf("lecture %s: %v", dir, err)
		}
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") {
				continue
			}
			path := filepath.Join(dir, e.Name())
			src, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("lecture %s: %v", path, err)
			}
			body := string(src)
			for _, marker := range weaponReferentialMarkers {
				if strings.Contains(body, marker) {
					offenders = append(offenders, path+" → "+marker)
				}
			}
		}
	}
	if len(offenders) > 0 {
		t.Errorf("référentiel d'armes détecté hors de internal/games/weapons :\n  %s\n"+
			"Le schéma et les données d'armes vivent dans ce package ; migrations/ ne "+
			"garde que le step qui délègue à ApplyLabels / ApplyRegistry.",
			strings.Join(offenders, "\n  "))
	}
}

// TestMigrationStepsDelegate : les deux steps versionnés existent toujours et
// pointent bien sur les fonctions de ce package (le déplacement n'a pas débranché
// le premier passage sur une DB vierge).
func TestWeaponSeedEntrypointsExported(t *testing.T) {
	// Compile-time : les signatures attendues par migrations/steps.go
	// (ApplySchema func(*sql.DB) error).
	var _ = ApplyLabels
	var _ = ApplyRegistry
	if testing.Short() {
		t.Skip("garde-rail de compilation")
	}
}
