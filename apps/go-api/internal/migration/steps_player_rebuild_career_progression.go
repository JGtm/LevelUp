package migration

// steps_player_rebuild_career_progression.go — rebuild career_progression
// via swap pour défaire la corruption d'index ART qui faussait WHERE xuid = ?
//
// Contexte : bug DuckDB observé 2026-05-21 (même classe que celui documenté
// dans `docs/INCIDENT_2026-05-20_match_participants_index.md`). Les requêtes
// `SELECT ... FROM career_progression WHERE xuid = ?` retournaient un sous-
// ensemble strict des rows (ex Madina : 65 visibles sur 74 réelles), et la
// dernière row réellement persistée par CareerLiveService n'apparaissait pas.
//
// Conséquence métier : ARG_MAX(banner_image_url) FILTER (...) sur Madina
// pickait un snapshot 2026-05-08 où banner était vide (alors que les 9 rows
// 2026-05-20 manquantes portent toutes la bannière). Identité Spartan
// affichait CSR/LUSR seuls dans l'UI, sans banner/emblem/career_rank.
//
// Stratégie : pour CHAQUE player DB (TargetPlayer s'applique par joueur lors
// du init pool), rebuild `career_progression` via :
//   1. CREATE TABLE __rebuilt avec même schéma
//   2. INSERT SELECT * (préserve toutes les rows)
//   3. DROP table corrompue
//   4. RENAME __rebuilt → career_progression
//
// L'index ART par défaut sur le primary scan est recréé from scratch lors
// du INSERT, ce qui défait la corruption. Idempotente : protégée par une
// row sentinelle dans `sync_meta` (clé `career_progression_rebuilt_v1`) pour
// éviter de re-rebuilder à chaque boot.

import (
	"database/sql"
	"fmt"
	"log/slog"
)

const careerProgressionRebuildMetaKey = "career_progression_rebuilt_v1"

func init() {
	Register(Migration{
		Name:        "rebuild_career_progression_defeat_art_corruption",
		TargetDB:    TargetPlayer,
		Description: "Rebuild career_progression via swap pour défaire la corruption ART (WHERE xuid = ? retournait sous-ensemble strict — diag 2026-05-21).",
		ApplySchema: func(db *sql.DB) error {
			// Garde idempotence : si déjà appliqué, no-op.
			hasMeta, err := tableExists(db, "sync_meta")
			if err != nil {
				return fmt.Errorf("rebuild_career: check sync_meta: %w", err)
			}
			if hasMeta {
				var marker sql.NullString
				if err := db.QueryRowContext(bootCtx(), `SELECT value FROM sync_meta WHERE key = ?`,
					careerProgressionRebuildMetaKey).Scan(&marker); err == nil && marker.Valid {
					// Déjà rebuild → no-op.
					return nil
				}
			}

			hasTable, err := tableExists(db, "career_progression")
			if err != nil {
				return fmt.Errorf("rebuild_career: check table: %w", err)
			}
			if !hasTable {
				// Pas de table → migration ultérieure la créera, on marque
				// quand même le sentinel pour éviter de re-check à chaque boot.
				return markRebuildDone(db)
			}

			// Diag : compter rows pour log informatif (et estimer la perte
			// potentielle pre-rebuild).
			var before int
			if err := db.QueryRowContext(bootCtx(), `SELECT COUNT(*) FROM career_progression`).Scan(&before); err != nil {
				return fmt.Errorf("rebuild_career: count before: %w", err)
			}

			// Rebuild via swap. Schéma copié à l'identique de la création
			// initiale (cf. steps_player.go:36-51). Toutes les colonnes
			// nullables / DEFAULT préservées.
			stmts := []string{
				`CREATE TABLE career_progression__rebuilt (
					xuid VARCHAR,
					rank INTEGER,
					rank_name VARCHAR,
					rank_tier VARCHAR,
					current_xp INTEGER DEFAULT 0,
					xp_for_next_rank INTEGER DEFAULT 0,
					xp_total INTEGER DEFAULT 0,
					is_max_rank BOOLEAN DEFAULT FALSE,
					adornment_path VARCHAR DEFAULT '',
					spartan_id VARCHAR DEFAULT '',
					banner_image_url VARCHAR DEFAULT '',
					emblem_image_url VARCHAR DEFAULT '',
					backdrop_image_url VARCHAR DEFAULT '',
					recorded_at TIMESTAMP
				)`,
				`INSERT INTO career_progression__rebuilt
				 SELECT xuid, rank, rank_name, rank_tier, current_xp,
				        xp_for_next_rank, xp_total, is_max_rank,
				        adornment_path, spartan_id, banner_image_url,
				        emblem_image_url, backdrop_image_url, recorded_at
				 FROM career_progression`,
				`DROP TABLE career_progression`,
				`ALTER TABLE career_progression__rebuilt RENAME TO career_progression`,
			}
			for _, sqlStmt := range stmts {
				if _, err := db.ExecContext(bootCtx(), sqlStmt); err != nil {
					return fmt.Errorf("rebuild_career: swap step (%s): %w",
						firstWords(sqlStmt, 3), err)
				}
			}

			var after int
			if err := db.QueryRowContext(bootCtx(), `SELECT COUNT(*) FROM career_progression`).Scan(&after); err != nil {
				return fmt.Errorf("rebuild_career: count after: %w", err)
			}
			slog.Info("migration rebuild_career_progression: table rebuilt (ART corruption defeated)",
				"rows_before_scan", before,
				"rows_after_rebuild", after,
			)
			return markRebuildDone(db)
		},
	})
}

// markRebuildDone écrit le sentinel d'idempotence dans sync_meta.
// Robuste aux schémas legacy (sync_meta sans updated_at) : ajoute la colonne
// si manquante avant l'INSERT/UPSERT (cf. tests pool_migration_test.go qui
// seed un schéma 2-colonnes).
func markRebuildDone(db *sql.DB) error {
	hasMeta, err := tableExists(db, "sync_meta")
	if err != nil || !hasMeta {
		// Pas de sync_meta → la migration initiale ne s'est pas encore
		// appliquée. On laisse passer sans erreur ; la prochaine application
		// re-check.
		return nil
	}
	if err := addColumnIfMissing(db, "sync_meta", "updated_at", "TIMESTAMP DEFAULT CURRENT_TIMESTAMP"); err != nil {
		return fmt.Errorf("rebuild_career: ensure updated_at: %w", err)
	}
	_, err = db.ExecContext(bootCtx(), `
		INSERT INTO sync_meta (key, value, updated_at)
		VALUES (?, 'true', NOW())
		ON CONFLICT (key) DO UPDATE SET
			value = 'true',
			updated_at = NOW()
	`, careerProgressionRebuildMetaKey)
	if err != nil {
		return fmt.Errorf("rebuild_career: mark done: %w", err)
	}
	return nil
}
