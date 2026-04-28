package migration

// steps_engagement.go — migrations additives pour la metrique EngagementScore.
//
// Reference plan : .ai/PLAN_ENGAGEMENT_IMPLEMENTATION.md §3.1.
//
// Strategie : 100% additive (ADR 0005 compatible). Aucune colonne ni table
// existante n'est modifiee semantiquement.
//
//   - 3 colonnes nullable ajoutees a player_match_enrichment :
//       * engagement_score (DOUBLE)            : score 0-100 percentile
//       * engagement_score_brut (DOUBLE)       : residu brut (joueur - attendu)
//       * engagement_score_confidence (VARCHAR) : "full" / "partial" / "insufficient_history"
//
//   - 1 colonne nullable ajoutee a player_match_enrichment :
//       * mode_category (VARCHAR) : PvP_ranked / PvP_unranked / FFA / 1v1
//         (utilisee pour filtrer l'historique du percentile par categorie)
//
//   - 1 colonne ajoutee a shared.match_registry :
//       * match_intensity (DOUBLE) : events/min/joueur du lobby (caracteristique match)
//
//   - 1 nouvelle table dans la player DB :
//       * engagement_coefficients (xuid, mode_category, coef_team_share,
//         coef_lobby_share, n_matches, last_updated)
//
// Toutes les modifications utilisent ADD COLUMN IF NOT EXISTS / CREATE TABLE
// IF NOT EXISTS pour permettre la re-execution idempotente.

import (
	"database/sql"
)

func init() {
	// ===== Player DB : colonnes engagement_score sur player_match_enrichment =====
	Register(Migration{
		Name:        "add_engagement_score_columns_to_player_match_enrichment",
		TargetDB:    TargetPlayer,
		Description: "Ajoute engagement_score, engagement_score_brut, engagement_score_confidence et mode_category a player_match_enrichment (idempotent)",
		ApplySchema: func(db *sql.DB) error {
			return execScript(db, `
				ALTER TABLE player_match_enrichment ADD COLUMN IF NOT EXISTS engagement_score DOUBLE;
				ALTER TABLE player_match_enrichment ADD COLUMN IF NOT EXISTS engagement_score_brut DOUBLE;
				ALTER TABLE player_match_enrichment ADD COLUMN IF NOT EXISTS engagement_score_confidence VARCHAR;
				ALTER TABLE player_match_enrichment ADD COLUMN IF NOT EXISTS mode_category VARCHAR;
				CREATE INDEX IF NOT EXISTS idx_pme_engagement_history
					ON player_match_enrichment(mode_category, engagement_score_brut);
			`)
		},
	})

	// ===== Player DB : table engagement_coefficients =====
	Register(Migration{
		Name:        "create_engagement_coefficients_table",
		TargetDB:    TargetPlayer,
		Description: "Cree la table engagement_coefficients pour stocker coef_team_share et coef_lobby_share par (xuid, mode_category)",
		ApplySchema: func(db *sql.DB) error {
			return execScript(db, `
				CREATE TABLE IF NOT EXISTS engagement_coefficients (
					xuid             VARCHAR NOT NULL,
					mode_category    VARCHAR NOT NULL,
					coef_team_share  DOUBLE NOT NULL,
					coef_lobby_share DOUBLE NOT NULL,
					n_matches        INTEGER NOT NULL,
					last_updated     TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
					PRIMARY KEY (xuid, mode_category)
				);
				CREATE INDEX IF NOT EXISTS idx_engagement_coefficients_xuid
					ON engagement_coefficients(xuid);
			`)
		},
	})

	// ===== Shared DB : colonne match_intensity sur match_registry =====
	Register(Migration{
		Name:        "add_match_intensity_to_match_registry",
		TargetDB:    TargetShared,
		Description: "Ajoute match_intensity (DOUBLE) a shared.match_registry (events/min/joueur du lobby, caracteristique permanente du match)",
		ApplySchema: func(db *sql.DB) error {
			return execScript(db, `
				ALTER TABLE match_registry ADD COLUMN IF NOT EXISTS match_intensity DOUBLE;
			`)
		},
	})
}
