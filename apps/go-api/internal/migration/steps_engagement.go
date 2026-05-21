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
	"fmt"
)

func init() {
	// ===== Player DB : colonnes engagement_score sur player_match_enrichment =====
	Register(Migration{
		Name:        "add_engagement_score_columns_to_player_match_enrichment",
		TargetDB:    TargetPlayer,
		Description: "Ajoute engagement_score, engagement_score_brut, engagement_score_confidence et mode_category a player_match_enrichment (idempotent)",
		ApplySchema: func(db *sql.DB) error {
			// Skip silencieusement si la table n'existe pas (cas des tests qui
			// montent une player DB minimaliste sans player_match_enrichment).
			// La table est creee par EnsurePlayerSchema lors du sync ; les tests
			// concernes (battlepass, etc.) n'ont pas besoin de ces colonnes.
			exists, err := tableExists(db, "player_match_enrichment")
			if err != nil {
				return fmt.Errorf("engagement migration: check table: %w", err)
			}
			if !exists {
				return nil
			}
			for _, col := range []struct{ name, typ string }{
				{"engagement_score", colDouble},
				{"engagement_score_brut", colDouble},
				{"engagement_score_confidence", colVarchar},
				{"mode_category", colVarchar},
			} {
				if err := addColumnIfMissing(db, "player_match_enrichment", col.name, col.typ); err != nil {
					return err
				}
			}
			return createIndexSafe(db, `
				CREATE INDEX IF NOT EXISTS idx_pme_engagement_history
					ON player_match_enrichment(mode_category, engagement_score_brut)
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

	// ===== Player DB : colonnes paces engagement (Phase recompute coefs) =====
	// Sert au calcul de coef_team_share / coef_lobby_share via la mediane
	// glissante des ratios pace_joueur/pace_team. Persister les paces evite
	// de devoir recalculer la courbe de chaque match historique a chaque
	// recompute (ce qui serait O(N×duree) sinon).
	Register(Migration{
		Name:        "add_engagement_pace_columns_to_player_match_enrichment",
		TargetDB:    TargetPlayer,
		Description: "Ajoute engagement_pace_player, engagement_pace_team, engagement_pace_lobby et engagement_player_activity a player_match_enrichment (Phase recompute coefs)",
		ApplySchema: func(db *sql.DB) error {
			exists, err := tableExists(db, "player_match_enrichment")
			if err != nil {
				return fmt.Errorf("engagement paces migration: check table: %w", err)
			}
			if !exists {
				return nil
			}
			for _, col := range []struct{ name, typ string }{
				{"engagement_pace_player", colDouble},
				{"engagement_pace_team", colDouble},
				{"engagement_pace_lobby", colDouble},
				{"engagement_player_activity", colInteger},
			} {
				if err := addColumnIfMissing(db, "player_match_enrichment", col.name, col.typ); err != nil {
					return err
				}
			}
			// Index partiel sur les rows ayant des paces non-null. Optimise les
			// scans de LoadRatioSamples qui filtrent toujours sur cette condition.
			return createIndexSafe(db, `
				CREATE INDEX IF NOT EXISTS idx_pme_engagement_paces
					ON player_match_enrichment(mode_category)
			`)
		},
	})

	// ===== Shared DB : colonne match_intensity sur match_registry =====
	Register(Migration{
		Name:        "add_match_intensity_to_match_registry",
		TargetDB:    TargetShared,
		Description: "Ajoute match_intensity (DOUBLE) a shared.match_registry (events/min/joueur du lobby, caracteristique permanente du match)",
		ApplySchema: func(db *sql.DB) error {
			// create_base_shared_schema inclut deja match_intensity depuis v5.5 ;
			// ce ALTER est conserve pour les bases existantes anterieures.
			// Guard necessaire car steps_engagement.go s'enregistre avant steps_shared.go
			// (ordre alphabetique des init()), donc match_registry peut ne pas exister
			// en DB vierge au moment ou cette migration est evaluee.
			exists, err := tableExists(db, "match_registry")
			if err != nil {
				return fmt.Errorf("add_match_intensity: check table: %w", err)
			}
			if !exists {
				return nil
			}
			return execScript(db, `
				ALTER TABLE match_registry ADD COLUMN IF NOT EXISTS match_intensity DOUBLE;
			`)
		},
	})
}
