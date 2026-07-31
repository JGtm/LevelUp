package migration

// steps_shared_weapon_kills_v3.go — table shadow weapon_kills_v3 (attribution v3 film).
//
// Net-new, ADDITIVE : la table v2 weapon_kills (et sa vue v_weapon_kills) ne sont JAMAIS
// touchées. weapon_kills_v3 reçoit la sortie du pipeline d'attribution v3 (signal de tir
// décodé du film : source_signal, high_weapon_id high-32, killing_shot_hit, burst_final,
// shot_counter) en parallèle de v2 — comparaison/recall sans régression sur l'existant.
//
// Schéma cloné de weapon_kills (add_weapon_kills + add_weapon_kills_reconciled_as :
// match_id/xuid/time_ms/weapon_id UBIGINT/reconciled_as UBIGINT/delta_ms/confidence/
// attribution_path/swap_detected/delayed_damage/player_index) + colonnes v3.
//
// Écriture : DELETE-then-INSERT par (match_id, xuid) depuis le backfill diagnostic v3
// (sérialisé, MaxOpenConns(1)), HORS chemin live → pas de pression concurrente ART. Pas de
// PK contraignante (comme weapon_kills) ; un index (match_id, xuid) accélère le DELETE.
// Voir .ai/PLAN_WEAPON_ATTRIBUTION_V3.md §1.

import "database/sql"

func init() {
	Register(Migration{
		Name:        "shared_weapon_kills_v3",
		TargetDB:    TargetShared,
		Description: "Table shadow weapon_kills_v3 (attribution v3 film, additive — v2 inchangée)",
		ApplySchema: func(db *sql.DB) error {
			return execScript(db, `
				CREATE TABLE IF NOT EXISTS weapon_kills_v3 (
					match_id         VARCHAR NOT NULL,
					xuid             VARCHAR NOT NULL,
					time_ms          INTEGER NOT NULL,
					weapon_id        UBIGINT,
					reconciled_as    UBIGINT,
					delta_ms         INTEGER,
					confidence       VARCHAR NOT NULL DEFAULT 'none',
					attribution_path VARCHAR DEFAULT 'none',
					swap_detected    BOOLEAN NOT NULL DEFAULT FALSE,
					delayed_damage   BOOLEAN NOT NULL DEFAULT FALSE,
					player_index     INTEGER,
					source_signal    VARCHAR,
					high_weapon_id   UINTEGER,
					killing_shot_hit BOOLEAN,
					burst_final      BOOLEAN,
					shot_counter     INTEGER,
					written_at       TIMESTAMP DEFAULT CURRENT_TIMESTAMP
				);
				CREATE INDEX IF NOT EXISTS idx_wk_v3_match_xuid ON weapon_kills_v3(match_id, xuid);
			`)
		},
	})
}
