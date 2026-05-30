// Package migration — reconcile_metadata_seeds.go : réconciliation idempotente
// des seeds de traduction metadata (mode_name_tr + asset_translations playlists).
//
// Pourquoi (fix 2026-05-30) : les seeds applyModeNameTr / applyPlaylistFRSeeds
// sont portés par des migrations one-shot. Quand de nouvelles traductions sont
// ajoutées au seed APRÈS qu'une base a déjà enregistré la migration comme "done",
// cette base ne les reçoit JAMAIS (une migration done ne re-tourne pas) — d'où des
// sous-modes et playlists qui réapparaissent en anglais ("Team Slayer", "Quick
// Play") dans le picker de réassociation et les filtres.
//
// ReconcileMetadataSeeds rejoue ces seeds à CHAQUE boot, de façon strictement
// idempotente (CREATE TABLE IF NOT EXISTS + INSERT OR IGNORE côté modes ; UPDATE
// garde-fou WHERE name = EN + INSERT OR IGNORE côté playlists). Conséquence : toute
// traduction ajoutée au seed converge automatiquement au prochain démarrage, sans
// nouvelle migration repair — la régression "ça revient en anglais" ne peut plus
// se reproduire. Non destructif : n'écrase jamais une traduction FR déjà correcte.
package migration

import (
	"database/sql"
	"fmt"
)

// ReconcileMetadataSeeds ré-applique les seeds de traduction idempotents sur la
// metadata.duckdb. À appeler au boot, juste après RunForDB(db, TargetMetadata).
func ReconcileMetadataSeeds(db *sql.DB) error {
	if db == nil {
		return nil
	}
	if err := applyModeNameTr(db); err != nil {
		return fmt.Errorf("reconcile mode_name_tr: %w", err)
	}
	if err := applyPlaylistFRSeeds(db); err != nil {
		return fmt.Errorf("reconcile playlist FR: %w", err)
	}
	return nil
}
