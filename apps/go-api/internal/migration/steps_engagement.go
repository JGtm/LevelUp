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
			// Append-only #23046 (2026-06-21) : PLUS d'index ART idx_pme_engagement_history
			// sur (mode_category, engagement_score_brut) — colonnes mutées par l'étage
			// engagement (INSERT pur taggé). Lecture via player_match_enrichment_latest.
			// Le swap append-only le supprime sur les DB existantes.
			return nil
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
			`)
		},
	})

	// ===== Player DB : reparation PK manquante sur engagement_coefficients =====
	// Bug (2026-06-01) : la migration create_engagement_coefficients_table utilise
	// CREATE TABLE IF NOT EXISTS avec PRIMARY KEY (xuid, mode_category). Sur les DB
	// joueur ou la table avait ete creee AVANT l'ajout de la PK, IF NOT EXISTS saute
	// la recreation -> la PK n'est jamais appliquee. Le UPSERT saveCoefficient
	// (ON CONFLICT (xuid, mode_category)) echoue alors a CHAQUE post-sync :
	//   "Binder Error: conflict target ... not referenced by a UNIQUE/PRIMARY KEY".
	// Fix : si la PK manque, on reconstruit la table avec la PK en preservant les
	// donnees (dedup defensif par (xuid, mode_category), garde la ligne la plus
	// recente). Idempotent : no-op si la PK est deja presente.
	Register(Migration{
		Name:        "repair_engagement_coefficients_primary_key",
		TargetDB:    TargetPlayer,
		Description: "Reconstruit engagement_coefficients avec PRIMARY KEY (xuid, mode_category) quand elle manque (CREATE TABLE IF NOT EXISTS historique)",
		ApplySchema: func(db *sql.DB) error {
			exists, err := tableExists(db, "engagement_coefficients")
			if err != nil {
				return fmt.Errorf("repair eng coefs PK: check table: %w", err)
			}
			if !exists {
				return nil
			}
			hasPK, err := hasPrimaryKey(db, "engagement_coefficients")
			if err != nil {
				return fmt.Errorf("repair eng coefs PK: check PK: %w", err)
			}
			if hasPK {
				return nil
			}
			return execScript(db, `
				CREATE TABLE engagement_coefficients__pkfix (
					xuid             VARCHAR NOT NULL,
					mode_category    VARCHAR NOT NULL,
					coef_team_share  DOUBLE NOT NULL,
					coef_lobby_share DOUBLE NOT NULL,
					n_matches        INTEGER NOT NULL,
					last_updated     TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
					PRIMARY KEY (xuid, mode_category)
				);
				INSERT INTO engagement_coefficients__pkfix
					SELECT xuid, mode_category, coef_team_share, coef_lobby_share, n_matches, last_updated
					FROM (
						SELECT *, ROW_NUMBER() OVER (
							PARTITION BY xuid, mode_category ORDER BY last_updated DESC
						) AS rn
						FROM engagement_coefficients
					) WHERE rn = 1;
				DROP TABLE engagement_coefficients;
				ALTER TABLE engagement_coefficients__pkfix RENAME TO engagement_coefficients;
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
			// Append-only #23046 (2026-06-21) : PLUS d'index ART idx_pme_engagement_paces
			// sur (mode_category) — colonne mutée par l'étage engagement. LoadRatioSamples
			// lit player_match_enrichment_latest. Le swap append-only le supprime sur
			// les DB existantes.
			return nil
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
