package migration

// steps_shared_match_csrs.go — Option A du plan pipeline CSR (suite Phase 9).
//
// Table dédiée pour persister le CSR de TOUS les joueurs d'un match ranked
// (et pas juste le joueur qui sync sa DB). Permet les comparaisons cross-joueurs
// (Squad, "qui était le mieux classé sur le match", coéquipiers en placement,
// etc.) sans dépendre de la player DB de chaque joueur.
//
// Le payload Halo /hi/matches/{id}/skill retourne déjà un MatchSkillData par
// participant — on jetait jusqu'ici toutes les valeurs CSR sauf celle du joueur
// sync. Avec cette table, on capture tout au moment du sync.
//
// Schéma volontairement aligné avec match_skill_rank (player DB) côté champs
// CSR : facilite les agrégations cross-tables si besoin futur.

import "database/sql"

func init() {
	Register(Migration{
		Name:        "add_shared_match_csrs",
		TargetDB:    TargetShared,
		Description: "Table shared.match_csrs : CSR par-match par-joueur (capture all participants)",
		ApplySchema: func(db *sql.DB) error {
			return execScript(db, `
				CREATE TABLE IF NOT EXISTS match_csrs (
					match_id                     VARCHAR NOT NULL,
					xuid                         VARCHAR NOT NULL,
					rating_type                  VARCHAR NOT NULL DEFAULT 'CSR',
					rating_value                 FLOAT,
					tier                         VARCHAR,
					sub_tier                     SMALLINT DEFAULT 0,
					tier_label                   VARCHAR,
					rating_delta                 FLOAT,
					measurement_matches_remaining INTEGER DEFAULT 0,
					season_id                    VARCHAR,
					created_at                   TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
					updated_at                   TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
					PRIMARY KEY (match_id, xuid)
				);
				CREATE INDEX IF NOT EXISTS idx_match_csrs_xuid    ON match_csrs(xuid);
				CREATE INDEX IF NOT EXISTS idx_match_csrs_season  ON match_csrs(season_id);
				CREATE INDEX IF NOT EXISTS idx_match_csrs_match   ON match_csrs(match_id);
			`)
		},
	})
}
