package migration

// steps_shared_h5_weapon_kill_kind.go — CAPTURE de la mecanique de kill Halo 5
// (kill_kind) sur weapon_kills. Phase 1 : capture SEULE, going-forward.
//
// PHASAGE VOLONTAIRE (decision utilisateur 2026-07-17) : cette migration ajoute la
// colonne pour CESSER DE PERDRE la mecanique du kill. Le code h5 la calcule deja
// (internal/games/halo_5/events.go::h5KillKind -> canonical.KillKind) puis la jetait
// a l'ingestion. Le CONSOMMATEUR (decoupage du bucket "Spartan" dans le donut des
// types de kill) et le BACKFILL de l'historique sont la PHASE 2 planifiee — PAS un
// oubli. Colonne consciemment capturee d'abord, exploitee ensuite.
//
// PIEGE VUE (traite ici) : v_weapon_kills fait `SELECT * EXCLUDE (rk)`. DuckDB FIGE
// les colonnes de sortie d'une vue a sa creation → sans recreation, kill_kind
// n'apparaitrait PAS via `*` meme apres l'ALTER. On recree donc la vue (definition
// IDENTIQUE a shared_append_only_weapon_kills_v1 : derniere generation par
// (match_id, xuid)) dans la MEME migration, apres l'ALTER, pour que kill_kind
// remonte via `*`.
//
// Append-only (ADR 0026) : ALTER ADD COLUMN seul, aucune PK ajoutee, aucun UPDATE,
// aucune reecriture de ligne. Idempotent (addColumnIfMissing + CREATE OR REPLACE
// VIEW). Doit s'executer APRES shared_append_only_weapon_kills_v1 (creation de la
// vue generationnelle + generation_id) — cf. canonicalOrder (order.go).

import (
	"database/sql"
	"fmt"
)

func init() {
	Register(Migration{
		Name:        "shared_h5_weapon_kill_kind_v1",
		TargetDB:    TargetShared,
		Description: "weapon_kills.kill_kind (mecanique de kill Halo 5 : weapon/melee/groundpound/shoulderbash) + recreation v_weapon_kills — capture Phase 1 (backfill + exploitation donut = Phase 2)",
		ApplySchema: applyWeaponKillKind,
	})
}

func applyWeaponKillKind(db *sql.DB) error {
	has, err := tableExists(db, "weapon_kills")
	if err != nil {
		return fmt.Errorf("wk kill_kind: check table: %w", err)
	}
	if !has {
		return nil
	}
	if err := addColumnIfMissing(db, "weapon_kills", "kill_kind", "VARCHAR"); err != nil {
		return fmt.Errorf("wk kill_kind: add column: %w", err)
	}
	// Recreation de la vue : definition byte-identique a shared_append_only_weapon_kills_v1
	// (v_weapon_kills = derniere generation par (match_id, xuid) via DENSE_RANK).
	// kill_kind remonte automatiquement via `*` maintenant qu'elle est colonne.
	return execScript(db, `
		CREATE OR REPLACE VIEW v_weapon_kills AS
		SELECT * EXCLUDE (rk) FROM (
			SELECT *,
			       COALESCE(reconciled_as, weapon_id) AS effective_weapon_id,
			       DENSE_RANK() OVER (PARTITION BY match_id, xuid ORDER BY generation_id DESC) AS rk
			FROM weapon_kills
		)
		WHERE rk = 1;
	`)
}
