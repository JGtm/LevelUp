package migration

// steps_shared_append_only_weapon_kills.go — éradication ART de weapon_kills
// (shared DB) — Phase 2 campagne #23046 (2026-06-21).
//
// **Pourquoi** : InsertWeaponKills (sync/writes.go) faisait `DELETE FROM
// weapon_kills WHERE match_id=? AND xuid=?` puis INSERT batch en TX. Le DELETE
// per-(match,xuid) retire N lignes de l'index idx_wk_match_xuid = vecteur DuckDB
// #23046 sur la DB SHARED (multi-writer). La table n'a PAS de PK technique :
// dé-indexer ne donnerait pas un DELETE PK-only (full-scan) — la forme correcte
// est append-only générationnel.
//
// **Stratégie append-only GÉNÉRATION (sans swap)** : la table n'ayant pas de PK
// à droper, on ajoute simplement generation_id (séquence weapon_kills_generation_seq,
// partagé par (match,xuid) d'un même write) + written_at via ALTER ADD COLUMN.
// La vue v_weapon_kills (déjà le point de lecture canonique, ADR 0016) ne garde,
// par (match_id, xuid), QUE les kills de la génération MAX (DENSE_RANK) — donc
// tous les readers de v_weapon_kills sont couverts sans changement.
//
// Idempotence : addColumnIfMissing + CREATE OR REPLACE VIEW.

import (
	"database/sql"
	"fmt"
)

func init() {
	Register(Migration{
		Name:        "shared_append_only_weapon_kills_v1",
		TargetDB:    TargetShared,
		Description: "weapon_kills append-only (generation_id + vue v_weapon_kills dernière génération) — élimine DELETE+INSERT sur idx_wk (#23046)",
		ApplySchema: applyAppendOnlyWeaponKills,
	})
}

func applyAppendOnlyWeaponKills(db *sql.DB) error {
	has, err := tableExists(db, "weapon_kills")
	if err != nil {
		return fmt.Errorf("append-only wk: check table: %w", err)
	}
	if !has {
		return nil
	}
	if err := addColumnIfMissing(db, "weapon_kills", "generation_id", "BIGINT DEFAULT 0"); err != nil {
		return fmt.Errorf("append-only wk: add generation_id: %w", err)
	}
	if err := addColumnIfMissing(db, "weapon_kills", "written_at", "TIMESTAMP DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP)"); err != nil {
		return fmt.Errorf("append-only wk: add written_at: %w", err)
	}
	return execScript(db, `
		CREATE SEQUENCE IF NOT EXISTS weapon_kills_generation_seq START 1;
		CREATE INDEX IF NOT EXISTS idx_wk_match_xuid_gen ON weapon_kills(match_id, xuid, generation_id);
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
